package billing

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/zhulang/llm-gateway/internal/config"
	"github.com/zhulang/llm-gateway/internal/pricing"
	"github.com/zhulang/llm-gateway/internal/store"
)

// ErrNoPricing is returned when a model has no pricing and the strategy is "reject".
var ErrNoPricing = errors.New("model billing not configured")

// UsageInfo holds token usage extracted from upstream responses.
type UsageInfo struct {
	PromptTokens     int
	CompletionTokens int
	CachedTokens     int // Deprecated: use CacheReadTokens instead
	CacheCreationTokens int
	CacheReadTokens     int
	// Ephemeral cache creation breakdown (from Anthropic usage.cache_creation).
	CacheCreation5mTokens int // ephemeral_5m_input_tokens
	CacheCreation1hTokens int // ephemeral_1h_input_tokens
	// CacheTokensIncludedInPrompt is true when PromptTokens already includes
	// cached tokens (OpenAI behaviour). When false, PromptTokens, CacheCreationTokens,
	// and CacheReadTokens are independent counts (Anthropic behaviour).
	CacheTokensIncludedInPrompt bool
	// ImageSize is the requested image resolution (e.g. "1K", "2K", "4K").
	// Used only when billing_type is "image".
	ImageSize string
	// ImageCount is the number of images generated. Used only when billing_type is "image".
	ImageCount int
}

// BillingService provides billing operations for the proxy layer.
type BillingService struct {
	store        store.Store
	pricingCache *pricing.Cache
	cfgHolder    *config.Holder

	// userTenant caches userID → primary pricing tenant ("" = none) with a short
	// TTL, so personal-key billing can apply tenant discounts without a DB hit
	// on every request.
	userTenantMu sync.RWMutex
	userTenant   map[string]userTenantEntry

	// notifyFunc, when set, replaces plain CreateNotification for user-facing
	// alerts so they can fan out to extra channels (e.g. SMS). Signature matches
	// notifier.Service.Notify.
	notifyFunc func(userID, eventType, title, content string, refType, refID *string)
}

type userTenantEntry struct {
	tenantID  string
	expiresAt time.Time
}

const userTenantTTL = 30 * time.Second

// NewBillingService creates a BillingService.
func NewBillingService(s store.Store, pc *pricing.Cache, cfgHolder *config.Holder) *BillingService {
	return &BillingService{
		store:        s,
		pricingCache: pc,
		cfgHolder:    cfgHolder,
		userTenant:   make(map[string]userTenantEntry),
	}
}

// SetNotifyFunc wires a multi-channel notifier for user alerts (optional).
func (bs *BillingService) SetNotifyFunc(f func(userID, eventType, title, content string, refType, refID *string)) {
	bs.notifyFunc = f
}

// defaultMaxTokens is used when max_tokens is not specified in the request.
const defaultMaxTokens = 4096

// maybeAlertLowBalance checks the user's remaining balance after a settlement
// and, if it has dropped below the configured threshold, sends a one-per-day
// low-balance notification. Best-effort: errors are logged, never propagated.
func (bs *BillingService) maybeAlertLowBalance(userID string) {
	if bs.cfgHolder == nil {
		return
	}
	threshold := bs.cfgHolder.Get().Retention.BalanceAlertThresholdCNY
	if threshold <= 0 {
		return
	}
	bal, err := bs.store.GetBalance(userID)
	if err != nil {
		return
	}
	available := bal.Balance - bal.Frozen
	if available >= threshold {
		return
	}
	claimed, err := bs.store.TryClaimNotificationDedup(userID, "balance_low")
	if err != nil {
		slog.Error("balance alert dedup failed", "user", userID, "error", err)
		return
	}
	if !claimed {
		return // already notified today
	}
	title := "余额不足提醒"
	content := fmt.Sprintf("您的账户可用余额仅剩 ¥%.2f，为避免服务中断，请及时充值。", available)
	if bs.notifyFunc != nil {
		bs.notifyFunc(userID, "balance_low", title, content, nil, nil)
		return
	}
	if _, err := bs.store.CreateNotification(userID, "balance_low", title, content, nil, nil); err != nil {
		slog.Error("create balance alert notification failed", "user", userID, "error", err)
	}
}

// handleNoPricing applies the configured no-pricing strategy.
// Returns nil if the request should proceed, ErrNoPricing if it should be rejected.
func (bs *BillingService) handleNoPricing(model string) error {
	strategy := "allow"
	if bs.cfgHolder != nil {
		if s := bs.cfgHolder.Get().Server.NoPricingStrategy; s != "" {
			strategy = s
		}
	}
	switch strategy {
	case "reject":
		slog.Warn("no pricing for model, rejecting request", "model", model, "strategy", "reject")
		return ErrNoPricing
	case "warn":
		slog.Warn("no pricing for model, allowing request with warning", "model", model, "strategy", "warn")
		return nil
	default: // "allow"
		slog.Debug("no pricing for model, skipping billing", "model", model)
		return nil
	}
}

// PreCharge estimates cost based on model pricing and max_tokens, then freezes the amount.
// Returns the frozen amount (for later settlement) and an error if balance is insufficient.
func (bs *BillingService) PreCharge(userID, model string, maxTokens int) (float64, error) {
	pricing, err := bs.getPricingForUser(userID, model)
	if err != nil {
		return 0, bs.handleNoPricing(model)
	}
	if !pricing.IsActive {
		return 0, nil
	}

	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}

	// Estimate: assume max_tokens for both input and output as upper bound.
	// Price is per 1M tokens, so divide by 1_000_000.
	estimatedCost := (float64(maxTokens)*pricing.InputPrice + float64(maxTokens)*pricing.OutputPrice) / 1_000_000
	if estimatedCost <= 0 {
		return 0, nil
	}

	if err := bs.store.FreezeBalance(userID, estimatedCost); err != nil {
		return 0, fmt.Errorf("insufficient balance: %w", err)
	}

	return estimatedCost, nil
}

// Settle calculates the actual cost based on token usage and settles the billing.
func (bs *BillingService) Settle(userID string, frozenAmount float64, model, requestID string, usage UsageInfo, apiKeyID string) error {
	if frozenAmount <= 0 {
		return nil
	}

	pricing, err := bs.getPricingForUser(userID, model)
	if err != nil {
		// No pricing -- just unfreeze.
		return bs.store.UnfreezeBalance(userID, frozenAmount)
	}

	// Actual cost based on real token usage (including cache-tier pricing and time-based multiplier).
	actualCost := calculateCost(usage, effectivePricing(pricing, time.Now()))

	// Cap actual cost at frozen amount to avoid negative balance edge cases.
	if actualCost > frozenAmount {
		actualCost = frozenAmount
	}

	err = bs.store.SettleBilling(userID, frozenAmount, actualCost, model, requestID, store.TokenUsage{
		PromptTokens:          usage.PromptTokens,
		CompletionTokens:      usage.CompletionTokens,
		CacheReadTokens:       usage.CacheReadTokens,
		CacheCreationTokens:   usage.CacheCreationTokens,
		CacheCreation5mTokens: usage.CacheCreation5mTokens,
		CacheCreation1hTokens: usage.CacheCreation1hTokens,
	}, apiKeyID)
	if err == nil {
		bs.maybeAlertLowBalance(userID)
	}
	return err
}

// Unfreeze releases the frozen amount back to the user's balance.
func (bs *BillingService) Unfreeze(userID string, frozenAmount float64) error {
	if frozenAmount <= 0 {
		return nil
	}
	return bs.store.UnfreezeBalance(userID, frozenAmount)
}

// CheckBalance verifies the user has a positive balance for a billable model.
func (bs *BillingService) CheckBalance(userID, model string) error {
	pricing, err := bs.getPricingForUser(userID, model)
	if err != nil {
		return bs.handleNoPricing(model)
	}
	if !pricing.IsActive {
		return nil
	}
	bal, err := bs.store.GetBalance(userID)
	if err != nil {
		return fmt.Errorf("unable to check balance")
	}
	available := bal.Balance - bal.Frozen
	if available <= 0 {
		return fmt.Errorf("insufficient balance")
	}
	return nil
}

// Charge calculates the actual cost based on usage and directly deducts from balance.
// Used for post-request billing when no pre-charge was done.
func (bs *BillingService) Charge(userID, model, requestID string, usage UsageInfo, apiKeyID string) error {
	pricing, err := bs.getPricingForUser(userID, model)
	if err != nil {
		return bs.handleNoPricing(model)
	}
	if !pricing.IsActive {
		return nil
	}

	actualCost := calculateCost(usage, effectivePricing(pricing, time.Now()))
	if actualCost <= 0 {
		return nil
	}

	tokens := store.TokenUsage{
		PromptTokens:          usage.PromptTokens,
		CompletionTokens:      usage.CompletionTokens,
		CacheReadTokens:       usage.CacheReadTokens,
		CacheCreationTokens:   usage.CacheCreationTokens,
		CacheCreation5mTokens: usage.CacheCreation5mTokens,
		CacheCreation1hTokens: usage.CacheCreation1hTokens,
	}
	if err := bs.store.DirectCharge(userID, actualCost, model, requestID, tokens, apiKeyID); err != nil {
		return err
	}
	bs.maybeAlertLowBalance(userID)
	return nil
}

// ChargeAndReturnCost calculates the actual cost, deducts from balance, and returns the cost.
// Used when the caller needs to know the charged amount (e.g. to save in a record).
func (bs *BillingService) ChargeAndReturnCost(userID, model, requestID string, usage UsageInfo, apiKeyID string) (float64, error) {
	pricing, err := bs.getPricingForUser(userID, model)
	if err != nil {
		return 0, bs.handleNoPricing(model)
	}
	if !pricing.IsActive {
		return 0, nil
	}

	actualCost := calculateCost(usage, effectivePricing(pricing, time.Now()))
	if actualCost <= 0 {
		return 0, nil
	}

	tokens := store.TokenUsage{
		PromptTokens:          usage.PromptTokens,
		CompletionTokens:      usage.CompletionTokens,
		CacheReadTokens:       usage.CacheReadTokens,
		CacheCreationTokens:   usage.CacheCreationTokens,
		CacheCreation5mTokens: usage.CacheCreation5mTokens,
		CacheCreation1hTokens: usage.CacheCreation1hTokens,
	}
	return actualCost, bs.store.DirectCharge(userID, actualCost, model, requestID, tokens, apiKeyID)
}

// CalculateCost computes the actual cost in CNY based on token usage and pricing (exported for subscription service).
func CalculateCost(usage UsageInfo, p *store.ModelPricing) float64 {
	return calculateCost(usage, p)
}

// effectivePricing applies any time-based multiplier to the pricing,
// returning a (possibly copied) pricing struct with scaled prices.
// t should be time.Now() for real billing calls.
func effectivePricing(p *store.ModelPricing, t time.Time) *store.ModelPricing {
	multiplier := resolveMultiplier(p.TimeBasedRules, t)
	return applyTimeMultiplier(p, multiplier)
}

func ceilTo4(cost float64) float64 {
	return math.Ceil(cost*10000) / 10000
}

func calculateCost(usage UsageInfo, p *store.ModelPricing) float64 {
	if p.BillingType == "image" {
		pricePerImage := p.InputPrice
		if usage.ImageSize == "4K" {
			pricePerImage = p.OutputPrice
		}
		count := usage.ImageCount
		if count <= 0 {
			count = 1
		}
		return ceilTo4(float64(count) * pricePerImage)
	}

	promptTokens := float64(usage.PromptTokens)
	cacheCreation := float64(usage.CacheCreationTokens)
	cacheRead := float64(usage.CacheReadTokens)
	completionTokens := float64(usage.CompletionTokens)

	// Both OpenAI and Anthropic include cached tokens in prompt_tokens/input_tokens.
	// Subtract cache_read_tokens to avoid double billing.
	if usage.CacheTokensIncludedInPrompt {
		promptTokens -= cacheRead
		if promptTokens < 0 {
			promptTokens = 0
		}
	}

	// Resolve effective prices: use tiered pricing if configured.
	inputPrice := p.InputPrice
	outputPrice := p.OutputPrice
	cachedInputPrice := p.CachedInputPrice

	if len(p.PricingTiers) > 0 {
		// Total input tokens for tier selection = prompt + cache_read + cache_creation
		totalInputTokens := usage.PromptTokens + usage.CacheReadTokens + usage.CacheCreationTokens
		for _, tier := range p.PricingTiers {
			if totalInputTokens >= tier.MinTokens && totalInputTokens < tier.MaxTokens {
				inputPrice = tier.InputPrice
				outputPrice = tier.OutputPrice
				cachedInputPrice = tier.CachedInputPrice
				break
			}
		}
	}

	// If we have a 5m/1h breakdown, apply differentiated cache creation pricing.
	// Otherwise fall back to total CacheCreationTokens × CacheCreationPrice (backward compat).
	var cacheCreationCost float64
	if usage.CacheCreation5mTokens > 0 || usage.CacheCreation1hTokens > 0 {
		cacheCreationCost = (float64(usage.CacheCreation5mTokens)*p.CacheCreationPrice +
			float64(usage.CacheCreation1hTokens)*p.CacheCreation1hPrice) / 1_000_000
	} else {
		cacheCreationCost = cacheCreation * p.CacheCreationPrice / 1_000_000
	}

	cost := (promptTokens*inputPrice+
		cacheRead*cachedInputPrice+
		completionTokens*outputPrice)/1_000_000 + cacheCreationCost

	if cost < 0 {
		return 0
	}
	return ceilTo4(cost)
}

// getPricing retrieves pricing via cache if available, otherwise from store directly.
func (bs *BillingService) getPricing(model string) (*store.ModelPricing, error) {
	if bs.pricingCache != nil {
		return bs.pricingCache.GetPricing(model)
	}
	return bs.store.GetPricing(model)
}

// resolveUserTenant returns the user's primary pricing tenant ("" = none),
// backed by a short-TTL cache to keep the billing hot path off the DB.
func (bs *BillingService) resolveUserTenant(userID string) string {
	if userID == "" {
		return ""
	}
	bs.userTenantMu.RLock()
	e, ok := bs.userTenant[userID]
	bs.userTenantMu.RUnlock()
	if ok && time.Now().Before(e.expiresAt) {
		return e.tenantID
	}

	tenantID, err := bs.store.GetUserPrimaryPricingTenant(userID)
	if err != nil {
		slog.Warn("failed to resolve user pricing tenant", "user_id", userID, "error", err)
		return ""
	}

	bs.userTenantMu.Lock()
	bs.userTenant[userID] = userTenantEntry{tenantID: tenantID, expiresAt: time.Now().Add(userTenantTTL)}
	bs.userTenantMu.Unlock()
	return tenantID
}

// getPricingForUser returns the effective pricing for a personal-key request.
// Priority: user pricing > tenant pricing > global pricing.
func (bs *BillingService) getPricingForUser(userID, model string) (*store.ModelPricing, error) {
	// Priority 1: User-specific pricing
	up, err := bs.getUserPricing(userID, model)
	if err != nil {
		// A transient DB error must NOT be treated as "no user pricing":
		// silently falling back to global pricing overcharges users who have a
		// discount configured. Propagate so the billing worker retries.
		slog.Warn("user pricing lookup failed, will retry via billing worker",
			"user_id", userID, "model", model, "error", err)
		return nil, err
	}
	if up != nil {
		return up, nil
	}

	// Priority 2: Tenant pricing (if user belongs to a tenant with custom pricing)
	if tenantID := bs.resolveUserTenant(userID); tenantID != "" {
		return bs.getTenantPricing(tenantID, model)
	}

	// Priority 3: Global pricing
	return bs.getPricing(model)
}

// getUserPricing returns user-specific pricing if configured, nil otherwise.
func (bs *BillingService) getUserPricing(userID, model string) (*store.ModelPricing, error) {
	if bs.pricingCache != nil {
		p, _, err := bs.pricingCache.GetUserPricing(userID, model)
		return p, err
	}
	up, err := bs.store.GetUserPricing(userID, model)
	if err == nil && up != nil {
		// If discount rate is set, apply it to global pricing
		if up.DiscountRate != nil {
			gp, gerr := bs.store.GetPricing(model)
			if gerr != nil {
				// Transient DB error — propagate so billing worker retries.
				return nil, gerr
			}
			if gp != nil {
				return pricing.ApplyDiscount(gp, *up.DiscountRate, up.IsActive), nil
			}
		}
		// Otherwise use absolute prices
		return up.ToModelPricing(), nil
	}
	return nil, err
}

// getTenantPricing returns tenant-specific pricing if configured, falling back to global pricing.
func (bs *BillingService) getTenantPricing(tenantID, model string) (*store.ModelPricing, error) {
	if bs.pricingCache != nil {
		p, _, err := bs.pricingCache.GetTenantPricing(tenantID, model)
		return p, err
	}
	tp, err := bs.store.GetTenantPricing(tenantID, model)
	if err == nil && tp != nil {
		if tp.DiscountRate != nil {
			gp, gerr := bs.store.GetPricing(model)
			if gerr != nil {
				// Transient DB error — propagate so billing worker retries.
				return nil, gerr
			}
			if gp != nil {
				return pricing.ApplyDiscount(gp, *tp.DiscountRate, tp.IsActive), nil
			}
		}
		return tp.ToModelPricing(), nil
	}
	return bs.store.GetPricing(model)
}

// ---------------------------------------------------------------------------
// Tenant billing methods
// ---------------------------------------------------------------------------

// TenantPreCharge estimates cost and freezes the amount from the tenant owner's balance.
func (bs *BillingService) TenantPreCharge(tenantID, model string, maxTokens int) (float64, error) {
	pricing, err := bs.getTenantPricing(tenantID, model)
	if err != nil {
		return 0, bs.handleNoPricing(model)
	}
	if !pricing.IsActive {
		return 0, nil
	}

	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}

	estimatedCost := (float64(maxTokens)*pricing.InputPrice + float64(maxTokens)*pricing.OutputPrice) / 1_000_000
	if estimatedCost <= 0 {
		return 0, nil
	}

	// Get tenant owner and freeze from their personal balance
	tenant, err := bs.store.GetTenantByID(tenantID)
	if err != nil {
		return 0, fmt.Errorf("unable to get tenant: %w", err)
	}

	if err := bs.store.FreezeBalance(tenant.OwnerID, estimatedCost); err != nil {
		return 0, fmt.Errorf("insufficient balance: %w", err)
	}

	return estimatedCost, nil
}

// TenantSettle calculates the actual cost and settles the billing from tenant owner's balance.
func (bs *BillingService) TenantSettle(tenantID string, frozenAmount float64, model, requestID string, usage UsageInfo, apiKeyID string) error {
	if frozenAmount <= 0 {
		return nil
	}

	// Get tenant owner
	tenant, err := bs.store.GetTenantByID(tenantID)
	if err != nil {
		return fmt.Errorf("unable to get tenant: %w", err)
	}

	pricingInfo, err := bs.getTenantPricing(tenantID, model)
	if err != nil {
		return bs.store.UnfreezeBalance(tenant.OwnerID, frozenAmount)
	}

	actualCost := calculateCost(usage, effectivePricing(pricingInfo, time.Now()))
	if actualCost > frozenAmount {
		actualCost = frozenAmount
	}

	return bs.store.SettleBilling(tenant.OwnerID, frozenAmount, actualCost, model, requestID, store.TokenUsage{
		PromptTokens:          usage.PromptTokens,
		CompletionTokens:      usage.CompletionTokens,
		CacheReadTokens:       usage.CacheReadTokens,
		CacheCreationTokens:   usage.CacheCreationTokens,
		CacheCreation5mTokens: usage.CacheCreation5mTokens,
		CacheCreation1hTokens: usage.CacheCreation1hTokens,
	}, apiKeyID)
}

// TenantUnfreeze releases the frozen amount back to the tenant owner's balance.
func (bs *BillingService) TenantUnfreeze(tenantID string, frozenAmount float64) error {
	if frozenAmount <= 0 {
		return nil
	}

	// Get tenant owner
	tenant, err := bs.store.GetTenantByID(tenantID)
	if err != nil {
		return fmt.Errorf("unable to get tenant: %w", err)
	}

	return bs.store.UnfreezeBalance(tenant.OwnerID, frozenAmount)
}

// TenantCharge calculates the actual cost and directly deducts from tenant owner's balance.
func (bs *BillingService) TenantCharge(tenantID, model, requestID string, usage UsageInfo, apiKeyID string) error {
	pricing, err := bs.getTenantPricing(tenantID, model)
	if err != nil {
		return bs.handleNoPricing(model)
	}
	if !pricing.IsActive {
		return nil
	}

	actualCost := calculateCost(usage, effectivePricing(pricing, time.Now()))
	if actualCost <= 0 {
		return nil
	}

	// Get tenant owner
	tenant, err := bs.store.GetTenantByID(tenantID)
	if err != nil {
		return fmt.Errorf("unable to get tenant")
	}

	tokens := store.TokenUsage{
		PromptTokens:          usage.PromptTokens,
		CompletionTokens:      usage.CompletionTokens,
		CacheReadTokens:       usage.CacheReadTokens,
		CacheCreationTokens:   usage.CacheCreationTokens,
		CacheCreation5mTokens: usage.CacheCreation5mTokens,
		CacheCreation1hTokens: usage.CacheCreation1hTokens,
	}
	// Charge owner's personal balance
	return bs.store.DirectCharge(tenant.OwnerID, actualCost, model, requestID, tokens, apiKeyID)
}

// CheckTenantBalance verifies the tenant owner has a positive available balance.
func (bs *BillingService) CheckTenantBalance(tenantID, model string) error {
	pricing, err := bs.getTenantPricing(tenantID, model)
	if err != nil {
		return bs.handleNoPricing(model)
	}
	if !pricing.IsActive {
		return nil
	}

	// Get tenant owner
	tenant, err := bs.store.GetTenantByID(tenantID)
	if err != nil {
		return fmt.Errorf("unable to get tenant")
	}

	// Check owner's personal balance
	bal, err := bs.store.GetBalance(tenant.OwnerID)
	if err != nil {
		return fmt.Errorf("unable to check balance")
	}
	available := bal.Balance - bal.Frozen
	if available <= 0 {
		return fmt.Errorf("insufficient balance")
	}
	return nil
}

// CheckSubUserQuota verifies the sub-user has sufficient quota remaining,
// and also checks the tenant owner's personal balance.
func (bs *BillingService) CheckSubUserQuota(subUserID, tenantID, model string) error {
	pricing, err := bs.getTenantPricing(tenantID, model)
	if err != nil {
		return bs.handleNoPricing(model)
	}
	if !pricing.IsActive {
		return nil
	}

	estimatedCost := (float64(defaultMaxTokens)*pricing.InputPrice + float64(defaultMaxTokens)*pricing.OutputPrice) / 1_000_000
	if estimatedCost <= 0 {
		return nil
	}

	if err := bs.store.CheckSubUserQuota(subUserID, estimatedCost); err != nil {
		return err
	}

	subUser, err := bs.store.GetTenantSubUser(subUserID)
	if err != nil {
		return fmt.Errorf("unable to check sub-user")
	}
	tenant, err := bs.store.GetTenantByID(subUser.TenantID)
	if err != nil {
		return fmt.Errorf("unable to check tenant")
	}
	bal, err := bs.store.GetBalance(tenant.OwnerID)
	if err != nil {
		return fmt.Errorf("unable to check owner balance")
	}
	available := bal.Balance - bal.Frozen
	if available <= 0 {
		return fmt.Errorf("insufficient balance")
	}
	return nil
}

// CheckSubUserQuotaOnly checks only the sub-user's quota limit without checking owner balance.
// Used when the subscription path handles billing instead of balance deduction.
func (bs *BillingService) CheckSubUserQuotaOnly(subUserID, tenantID, model string) error {
	pricing, err := bs.getTenantPricing(tenantID, model)
	if err != nil {
		return bs.handleNoPricing(model)
	}
	if !pricing.IsActive {
		return nil
	}
	estimatedCost := (float64(defaultMaxTokens)*pricing.InputPrice + float64(defaultMaxTokens)*pricing.OutputPrice) / 1_000_000
	if estimatedCost <= 0 {
		return nil
	}
	return bs.store.CheckSubUserQuota(subUserID, estimatedCost)
}

// GetTenantPricingPublic returns the effective pricing for a model under a tenant.
func (bs *BillingService) GetTenantPricingPublic(tenantID, model string) (*store.ModelPricing, error) {
	return bs.getTenantPricing(tenantID, model)
}

// ChargeSubUser charges a sub-user for actual usage.
// Deducts from the tenant owner's personal balance and updates sub-user quota.
func (bs *BillingService) ChargeSubUser(subUserID, tenantID, model, requestID string, usage UsageInfo, apiKeyID string) error {
	pricing, err := bs.getTenantPricing(tenantID, model)
	if err != nil {
		return bs.handleNoPricing(model)
	}
	if !pricing.IsActive {
		return nil
	}

	actualCost := calculateCost(usage, effectivePricing(pricing, time.Now()))
	if actualCost <= 0 {
		return nil
	}

	tokens := store.TokenUsage{
		PromptTokens:          usage.PromptTokens,
		CompletionTokens:      usage.CompletionTokens,
		CacheReadTokens:       usage.CacheReadTokens,
		CacheCreationTokens:   usage.CacheCreationTokens,
		CacheCreation5mTokens: usage.CacheCreation5mTokens,
		CacheCreation1hTokens: usage.CacheCreation1hTokens,
	}

	tenant, err := bs.store.GetTenantByID(tenantID)
	if err != nil {
		return fmt.Errorf("get tenant: %w", err)
	}

	if err := bs.store.DirectCharge(tenant.OwnerID, actualCost, model, requestID, tokens, apiKeyID); err != nil {
		return fmt.Errorf("charge owner balance: %w", err)
	}

	if err := bs.store.IncrementSubUserQuotaUsed(subUserID, actualCost); err != nil {
		slog.Error("failed to increment sub-user quota", "sub_user_id", subUserID, "amount", actualCost, "error", err)
	}

	if err := bs.store.RecordSubUserTransaction(tenantID, subUserID, actualCost, model, requestID, tokens, apiKeyID); err != nil {
		slog.Error("failed to record sub-user transaction", "sub_user_id", subUserID, "tenant_id", tenantID, "amount", actualCost, "error", err)
	}

	return nil
}

