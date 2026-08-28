package subscription

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/zhulang/llm-gateway/internal/billing"
	"github.com/zhulang/llm-gateway/internal/pricing"
	"github.com/zhulang/llm-gateway/internal/store"
)

// Service provides subscription business logic.
type Service struct {
	store        store.Store
	pricingCache *pricing.Cache
}

// NewService creates a subscription service.
func NewService(s store.Store, pc *pricing.Cache) *Service {
	return &Service{store: s, pricingCache: pc}
}

// AccessResult describes whether a model is covered by subscription and within quota.
type AccessResult struct {
	Covered     bool
	WithinQuota bool
}

// CheckAccess checks if the user has an active subscription covering the model and within quota.
// Subscriptions are evaluated in FIFO order (oldest first). The first subscription that covers
// the model and still has remaining quota is used.
func (s *Service) CheckAccess(userID, model string) (AccessResult, error) {
	subs, err := s.store.GetActiveSubscriptions(userID)
	if err != nil {
		return AccessResult{}, err
	}
	if len(subs) == 0 {
		return AccessResult{}, nil
	}

	covered := false
	for _, sub := range subs {
		models, err := s.store.GetSubscriptionPlanModels(sub.PlanID)
		if err != nil {
			continue
		}
		if !containsModel(models, model) {
			continue
		}
		covered = true
		period := currentPeriod(sub.StartedAt)
		totalUsed, err := s.store.GetSubscriptionTotalUsage(sub.ID, period)
		if err != nil {
			continue
		}
		effectiveQuota := sub.Plan.QuotaAmountCNY + sub.ExtraQuotaCNY
		if totalUsed < effectiveQuota {
			return AccessResult{Covered: true, WithinQuota: true}, nil
		}
	}
	if covered {
		return AccessResult{Covered: true, WithinQuota: false}, nil
	}
	return AccessResult{}, nil
}

// RecordUsage records subscription usage after a request completes.
// Uses FIFO logic to find the appropriate subscription.
func (s *Service) RecordUsage(userID, model, requestID string, usage billing.UsageInfo, apiKeyID string) error {
	sub, err := s.findSubscriptionForModelFIFO(userID, model)
	if err != nil || sub == nil {
		return fmt.Errorf("subscription: no active subscription covering model %s", model)
	}

	p, err := s.getPricing(model)
	if err != nil {
		slog.Warn("subscription: no pricing for model, skipping usage record", "model", model)
		return nil
	}

	// 计算实际费用（始终按单价计算）
	actualCost := billing.CalculateCost(usage, p)
	if actualCost <= 0 {
		return nil
	}

	// 计算配额消耗
	var quotaConsumed float64
	if isImagePlan(sub.Plan) {
		// 图片套餐：配额单位是张数
		if usage.ImageCount > 0 {
			quotaConsumed = float64(usage.ImageCount)
		} else {
			quotaConsumed = 1
		}
	} else {
		// 普通套餐：配额单位是金额
		quotaConsumed = actualCost
	}

	period := currentPeriod(sub.StartedAt)
	tokens := store.TokenUsage{
		PromptTokens:          usage.PromptTokens,
		CompletionTokens:      usage.CompletionTokens,
		CacheReadTokens:       usage.CacheReadTokens,
		CacheCreationTokens:   usage.CacheCreationTokens,
		CacheCreation5mTokens: usage.CacheCreation5mTokens,
		CacheCreation1hTokens: usage.CacheCreation1hTokens,
	}

	// 传入两个值：actualCost（交易金额）和 quotaConsumed（配额消耗）
	return s.store.RecordSubscriptionUsageAndTransaction(sub.ID, userID, model, requestID, period, tokens, actualCost, quotaConsumed, apiKeyID)
}

// GetUsageSummary returns the current period's usage summary for the user's first active subscription.
func (s *Service) GetUsageSummary(userID string) (*store.SubscriptionUsageSummary, error) {
	sub, err := s.store.GetActiveSubscription(userID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, nil
	}
	period := currentPeriod(sub.StartedAt)
	effectiveQuota := sub.Plan.QuotaAmountCNY + sub.ExtraQuotaCNY
	return s.store.GetSubscriptionUsageSummary(sub.ID, period, effectiveQuota)
}

// GetUsageSummaryForSubscription returns usage summary for a specific subscription.
func (s *Service) GetUsageSummaryForSubscription(sub *store.UserSubscription) (*store.SubscriptionUsageSummary, error) {
	period := currentPeriod(sub.StartedAt)
	effectiveQuota := sub.Plan.QuotaAmountCNY + sub.ExtraQuotaCNY
	return s.store.GetSubscriptionUsageSummary(sub.ID, period, effectiveQuota)
}

// GetAllActiveSubscriptions returns all active subscriptions for the user.
func (s *Service) GetAllActiveSubscriptions(userID string) ([]store.UserSubscription, error) {
	return s.store.GetActiveSubscriptions(userID)
}

// Subscribe creates a new independent subscription for the user.
// Each purchase creates a separate subscription with its own expiry and quota.
func (s *Service) Subscribe(userID string, planID int) (*store.UserSubscription, error) {
	// Tenants with custom pricing manage their own billing — block plan purchases.
	if tenantID, err := s.store.GetUserPrimaryPricingTenant(userID); err == nil && tenantID != "" {
		return nil, fmt.Errorf("您所在的租户已配置专属定价，无法购买套餐")
	}

	plan, err := s.store.GetSubscriptionPlan(planID)
	if err != nil {
		return nil, err
	}
	if plan.Status != "active" {
		return nil, fmt.Errorf("subscription plan is not available")
	}

	brand := planBrand(plan.Name)

	// Trial plans can only be purchased once.
	if isTrialPlan(plan.Name) {
		_ = s.store.ExpireUserSubscriptionsByBrand(userID, brand)
		existing, err := s.store.GetActiveSubscriptionByBrand(userID, brand)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, fmt.Errorf("trial plan can only be purchased once")
		}
	}

	unit := "月"
	if plan.DurationDays <= 7 {
		unit = "周"
	}
	desc := fmt.Sprintf("订阅套餐: %s (¥%.2f/%s)", plan.DisplayName, plan.MonthlyPriceCNY, unit)
	if err := s.store.DeductForSubscription(userID, plan.MonthlyPriceCNY, desc); err != nil {
		return nil, fmt.Errorf("balance deduction failed: %w", err)
	}

	expiresAt := time.Now().AddDate(0, 0, plan.DurationDays)
	sub, err := s.store.CreateSubscription(userID, planID, expiresAt, brand)
	if err != nil {
		_ = s.store.Recharge(userID, plan.MonthlyPriceCNY, fmt.Sprintf("退款: 订阅创建失败 (%s)", plan.DisplayName))
		return nil, err
	}

	order, _ := s.store.CreateSubscriptionOrder(userID, planID, plan.MonthlyPriceCNY, "new")
	if order != nil {
		_ = s.store.CompleteSubscriptionOrder(order.ID, "balance", "")
	}
	return sub, nil
}

// GetActiveSubscription returns the user's active subscription with plan details.
func (s *Service) GetActiveSubscription(userID string) (*store.UserSubscription, error) {
	return s.store.GetActiveSubscription(userID)
}

// findSubscriptionForModelFIFO finds the earliest active subscription covering the model
// that still has remaining quota (FIFO consumption order).
func (s *Service) findSubscriptionForModelFIFO(userID, model string) (*store.UserSubscription, error) {
	subs, err := s.store.GetActiveSubscriptions(userID)
	if err != nil {
		return nil, err
	}
	for _, sub := range subs {
		models, err := s.store.GetSubscriptionPlanModels(sub.PlanID)
		if err != nil {
			continue
		}
		if !containsModel(models, model) {
			continue
		}
		period := currentPeriod(sub.StartedAt)
		totalUsed, err := s.store.GetSubscriptionTotalUsage(sub.ID, period)
		if err != nil {
			continue
		}
		effectiveQuota := sub.Plan.QuotaAmountCNY + sub.ExtraQuotaCNY
		if totalUsed < effectiveQuota {
			return &sub, nil
		}
	}
	return nil, nil
}

func (s *Service) getPricing(model string) (*store.ModelPricing, error) {
	if s.pricingCache != nil {
		return s.pricingCache.GetPricing(model)
	}
	return s.store.GetPricing(model)
}

func containsModel(models []string, target string) bool {
	for _, m := range models {
		if m == target {
			return true
		}
	}
	return false
}

// currentPeriod returns the subscription start date truncated to day, used as the billing period key.
func currentPeriod(startedAt time.Time) time.Time {
	return time.Date(startedAt.Year(), startedAt.Month(), startedAt.Day(), 0, 0, 0, 0, time.UTC)
}

// planBrand extracts the brand prefix from a plan name (e.g. "openai-pro" → "openai", "pro" → "claude").
func planBrand(name string) string {
	if strings.HasPrefix(name, "openai-") {
		return "openai"
	}
	if strings.HasPrefix(name, "image-") {
		return "image"
	}
	return "claude"
}

func isTrialPlan(name string) bool {
	return name == "trial" || name == "openai-trial"
}

// isImagePlan 判断套餐是否是图片套餐（按张数计费）。
func isImagePlan(p *store.SubscriptionPlan) bool {
	return p != nil && strings.HasPrefix(p.Name, "image-")
}
