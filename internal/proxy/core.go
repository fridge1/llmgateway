package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zhulang/llm-gateway/internal/apikey"
	"github.com/zhulang/llm-gateway/internal/balancer"
	"github.com/zhulang/llm-gateway/internal/billing"
	"github.com/zhulang/llm-gateway/internal/circuit"
	"github.com/zhulang/llm-gateway/internal/config"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/metrics"
	"github.com/zhulang/llm-gateway/internal/moderation"
	"github.com/zhulang/llm-gateway/internal/router"
	"github.com/zhulang/llm-gateway/internal/store"
	"github.com/zhulang/llm-gateway/internal/subscription"
)

// privateNetworks contains CIDR ranges that should not be accessible via upstream proxy.
var privateNetworks = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"::1/128",
	"fc00::/7",
}

var parsedPrivateNets []*net.IPNet

func init() {
	for _, cidr := range privateNetworks {
		_, ipNet, _ := net.ParseCIDR(cidr)
		parsedPrivateNets = append(parsedPrivateNets, ipNet)
	}
}

// validateUpstreamURL checks that the upstream URL uses an allowed scheme and does not
// resolve to a private/internal IP address.
func validateUpstreamURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid upstream URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("disallowed upstream URL scheme: %s", u.Scheme)
	}
	host := u.Hostname()
	ips, err := net.LookupHost(host)
	if err != nil {
		return fmt.Errorf("cannot resolve upstream host %s: %w", host, err)
	}
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		for _, ipNet := range parsedPrivateNets {
			if ipNet.Contains(ip) {
				return fmt.Errorf("upstream host %s resolves to private IP %s", host, ipStr)
			}
		}
	}
	return nil
}

// AuthResult encapsulates the result of API key authentication.
// Either User (for user-owned keys), TenantKey (for tenant-owned keys),
// or SubUserKey (for tenant sub-user keys) is non-nil.
type AuthResult struct {
	User       *store.User
	TenantKey  *store.TenantAPIKey
	SubUserKey *store.TenantSubUserKey
	SubUser    *store.TenantSubUser
	APIKeyID   string // populated for user-owned keys
	PlanID     *int   // 非空时该用户 key 仅可访问该套餐内模型
	// MemberTenantID is the tenant a user-owned key's owner belongs to
	// (earliest-joined active membership), or "" if none. When set, the request
	// is billed to and routed through this tenant even though it authenticated
	// with a personal key. Ignored for tenant/sub-user keys.
	MemberTenantID string
}

// IsTenant returns true if the authenticated key is a tenant-owned key.
func (a *AuthResult) IsTenant() bool { return a.TenantKey != nil }

// IsSubUser returns true if the authenticated key is a tenant sub-user key.
func (a *AuthResult) IsSubUser() bool { return a.SubUserKey != nil && a.SubUser != nil }

// TenantID returns the owning tenant's ID for tenant and sub-user keys, the
// member tenant for a user-owned key whose owner belongs to a tenant, or "".
// Used to resolve tenant upstream overrides.
func (a *AuthResult) TenantID() string {
	switch {
	case a.IsTenant():
		return a.TenantKey.TenantID
	case a.IsSubUser():
		return a.SubUserKey.TenantID
	}
	return a.MemberTenantID
}

// Core holds shared dependencies and provides common operations for all API format handlers.
type Core struct {
	CfgHolder           *config.Holder
	Router              atomic.Pointer[router.Router]
	Balancer            *balancer.RoundRobin
	Client              *http.Client
	Store               store.Store
	KeyCache            *apikey.Cache
	BillingService      *billing.BillingService
	SubscriptionService *subscription.Service
	TouchBatcher        *apikey.TouchBatcher
	ActiveBatcher       *apikey.ActiveBatcher

	billingCh   chan billingJob
	billingWg   sync.WaitGroup
	overflowSem chan struct{}

	// moderation is the optional content-safety screen (nil = disabled).
	moderation *moderation.Service
}

type billingJob struct {
	userID, model, requestID string
	tenantID                 string
	tenantKeyID              string
	subUserID                string
	subUserKeyID             string
	apiKeyID                 string
	usage                    billing.UsageInfo
}

// NewSharedHTTPClient creates an *http.Client tuned for proxying LLM requests.
// Both Handler and Core should share the same client to maximise connection reuse.
func NewSharedHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:          200,
			MaxIdleConnsPerHost:   50,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 30 * time.Minute,
			// Increase buffer sizes for smoother SSE streaming
			WriteBufferSize:       64 * 1024, // 64KB write buffer
			ReadBufferSize:        64 * 1024, // 64KB read buffer
		},
	}
}

// NewCore creates a Core with the given dependencies.
// It starts billing worker goroutines to process async billing jobs.
func NewCore(cfgHolder *config.Holder, rt *router.Router, lb *balancer.RoundRobin, s store.Store, kc *apikey.Cache, bs *billing.BillingService, client *http.Client, tb *apikey.TouchBatcher) *Core {
	billingCfg := cfgHolder.Get().Billing
	billingWorkers := billingCfg.Workers
	if billingWorkers <= 0 {
		billingWorkers = 10
	}
	billingQueueSize := billingCfg.QueueSize
	if billingQueueSize <= 0 {
		billingQueueSize = 10000
	}
	overflowConcurrency := billingCfg.OverflowConcurrency
	if overflowConcurrency <= 0 {
		overflowConcurrency = 100
	}

	c := &Core{
		CfgHolder:      cfgHolder,
		Balancer:       lb,
		Client:         client,
		Store:          s,
		KeyCache:       kc,
		BillingService: bs,
		TouchBatcher:   tb,
		billingCh:      make(chan billingJob, billingQueueSize),
		overflowSem:    make(chan struct{}, overflowConcurrency),
	}
	c.Router.Store(rt)

	for range billingWorkers {
		c.billingWg.Add(1)
		go c.billingWorker()
	}
	return c
}

// StopBillingWorkers drains the billing channel and waits for workers to finish.
func (c *Core) StopBillingWorkers() {
	close(c.billingCh)
	c.billingWg.Wait()
}

func (c *Core) billingWorker() {
	defer c.billingWg.Done()
	for job := range c.billingCh {
		c.executeBillingJob(job)
	}
}

func (c *Core) executeBillingJob(job billingJob) {
	metrics.Get().BillingJobsTotal.Add(1)
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		if job.subUserID != "" {
			err = c.BillingService.ChargeSubUser(job.subUserID, job.tenantID, job.model, job.requestID, job.usage, job.subUserKeyID)
		} else if job.tenantID != "" {
			err = c.BillingService.TenantCharge(job.tenantID, job.model, job.requestID, job.usage, job.tenantKeyID)
		} else {
			err = c.BillingService.Charge(job.userID, job.model, job.requestID, job.usage, job.apiKeyID)
		}
		if err == nil {
			break
		}
		time.Sleep(time.Duration(attempt+1) * time.Second)
	}
	if err != nil {
		slog.Error("billing charge failed after retries",
			"user_id", job.userID,
			"tenant_id", job.tenantID, "sub_user_id", job.subUserID,
			"model", job.model, "request_id", job.requestID, "error", err)
		return
	}
}

// SetRouter atomically replaces the current router (used during hot reload).
func (c *Core) SetRouter(rt *router.Router) {
	c.Router.Store(rt)
}

// AuthenticateBearer authenticates using Authorization: Bearer header.
func (c *Core) AuthenticateBearer(r *http.Request) (*AuthResult, error) {
	token := httputil.ExtractBearerToken(r)
	if token == "" {
		return nil, fmt.Errorf("missing token")
	}
	auth, err := c.authenticateToken(token)
	if err == nil {
		auth.recordIdentity(r)
	}
	return auth, err
}

// AuthenticateAny authenticates using x-api-key header first, then Authorization: Bearer.
func (c *Core) AuthenticateAny(r *http.Request) (*AuthResult, error) {
	token := r.Header.Get("x-api-key")
	if token == "" {
		token = httputil.ExtractBearerToken(r)
	}
	if token == "" {
		return nil, fmt.Errorf("missing token")
	}
	auth, err := c.authenticateToken(token)
	if err == nil {
		auth.recordIdentity(r)
	}
	return auth, err
}

// recordIdentity 把认证结果写入 context 中的身份容器，供 AccessLog 中间件记录到日志。
func (a *AuthResult) recordIdentity(r *http.Request) {
	switch {
	case a.IsTenant():
		httputil.SetIdentity(r, "tenant", a.TenantKey.Name, a.TenantKey.TenantID)
	case a.IsSubUser():
		httputil.SetIdentity(r, "sub_user", a.SubUser.Username, a.SubUser.ID)
	case a.User != nil:
		httputil.SetIdentity(r, "user", a.User.Phone, a.User.ID)
	}
}

func (c *Core) authenticateToken(token string) (*AuthResult, error) {
	keyHash := apikey.HashAPIKey(token)

	// Try user key cache first.
	cachedAuth := c.KeyCache.Get(keyHash)
	if cachedAuth != nil {
		metrics.Get().AuthCacheHits.Add(1)
		if cachedAuth.User.Status != "active" {
			return nil, fmt.Errorf("account disabled")
		}
		if c.ActiveBatcher != nil {
			c.ActiveBatcher.Touch(cachedAuth.User.ID)
		}
		return &AuthResult{User: cachedAuth.User, APIKeyID: cachedAuth.APIKeyID, PlanID: cachedAuth.PlanID, MemberTenantID: cachedAuth.MemberTenantID}, nil
	}
	metrics.Get().AuthCacheMisses.Add(1)

	// Try user key from DB.
	ak, err := c.Store.GetAPIKeyByHash(keyHash)
	if err == nil {
		u, err := c.Store.GetUserByID(ak.UserID)
		if err != nil {
			return nil, fmt.Errorf("invalid api key")
		}
		memberTenantID := resolveMemberTenantID(c.Store, u.ID)
		c.KeyCache.Set(keyHash, &apikey.CachedAuth{User: u, APIKeyID: ak.ID, PlanID: ak.PlanID, MemberTenantID: memberTenantID})
		c.TouchBatcher.Touch(ak.ID)
		if c.ActiveBatcher != nil {
			c.ActiveBatcher.Touch(u.ID)
		}

		if u.Status != "active" {
			return nil, fmt.Errorf("account disabled")
		}
		return &AuthResult{User: u, APIKeyID: ak.ID, PlanID: ak.PlanID, MemberTenantID: memberTenantID}, nil
	}

	// Try tenant key from DB.
	tk, tkErr := c.Store.GetTenantAPIKeyByHash(keyHash)
	if tkErr == nil {
		go func() { _ = c.Store.TouchTenantAPIKeyLastUsed(tk.ID) }()
		return &AuthResult{TenantKey: tk}, nil
	}

	// Try tenant sub-user key from DB.
	sk, skErr := c.Store.GetTenantSubUserKeyByHash(keyHash)
	if skErr == nil {
		subUser, err := c.Store.GetTenantSubUser(sk.SubUserID)
		if err != nil {
			return nil, fmt.Errorf("invalid sub-user key")
		}
		if subUser.Status != "active" {
			return nil, fmt.Errorf("sub-user disabled")
		}
		go func() { _ = c.Store.TouchTenantSubUserKeyLastUsed(sk.ID) }()
		return &AuthResult{SubUserKey: sk, SubUser: subUser}, nil
	}

	return nil, fmt.Errorf("invalid api key")
}

// CheckModelAccess verifies that the key is allowed to access the requested model.
// Returns nil if access is allowed.
func (c *Core) CheckModelAccess(auth *AuthResult, model string) error {
	if auth.User != nil && auth.PlanID != nil {
		models, err := c.Store.GetSubscriptionPlanModels(*auth.PlanID)
		if err != nil {
			return fmt.Errorf("无法验证模型权限")
		}
		for _, m := range models {
			if m == model {
				return nil
			}
		}
		return fmt.Errorf("模型 %s 不在该密钥所属套餐的可用范围内", model)
	}
	return nil
}

// CheckBilling checks whether the authenticated entity has sufficient balance/quota.
func (c *Core) CheckBilling(auth *AuthResult, model string) error {
	if auth.IsTenant() {
		if c.SubscriptionService != nil {
			ownerID := c.getTenantOwnerID(auth.TenantKey.TenantID)
			if ownerID != "" {
				result, err := c.SubscriptionService.CheckAccess(ownerID, model)
				if err != nil {
					slog.Warn("tenant subscription check failed, falling back to balance", "error", err)
				} else if result.Covered && result.WithinQuota {
					return nil
				}
			}
		}
		return c.BillingService.CheckTenantBalance(auth.TenantKey.TenantID, model)
	}

	if auth.IsSubUser() {
		if err := c.BillingService.CheckSubUserQuotaOnly(auth.SubUser.ID, auth.SubUserKey.TenantID, model); err != nil {
			return err
		}
		if c.SubscriptionService != nil {
			ownerID := c.getTenantOwnerID(auth.SubUserKey.TenantID)
			if ownerID != "" {
				result, err := c.SubscriptionService.CheckAccess(ownerID, model)
				if err != nil {
					slog.Warn("sub-user subscription check failed, falling back to balance", "error", err)
				} else if result.Covered && result.WithinQuota {
					return nil
				}
			}
		}
		return c.BillingService.CheckTenantBalance(auth.SubUserKey.TenantID, model)
	}

	// A personal key whose owner belongs to a tenant bills against that tenant,
	// mirroring the tenant-key path above (owner subscription first, then tenant balance).
	if auth.MemberTenantID != "" {
		if c.SubscriptionService != nil {
			ownerID := c.getTenantOwnerID(auth.MemberTenantID)
			if ownerID != "" {
				result, err := c.SubscriptionService.CheckAccess(ownerID, model)
				if err != nil {
					slog.Warn("member tenant subscription check failed, falling back to balance", "error", err)
				} else if result.Covered && result.WithinQuota {
					return nil
				}
			}
		}
		return c.BillingService.CheckTenantBalance(auth.MemberTenantID, model)
	}

	if c.SubscriptionService != nil {
		result, err := c.SubscriptionService.CheckAccess(auth.User.ID, model)
		if err != nil {
			slog.Warn("subscription check failed, falling back to balance", "error", err)
		} else if result.Covered && result.WithinQuota {
			return nil
		}
	}

	return c.BillingService.CheckBalance(auth.User.ID, model)
}

// IsNoPricingError returns true if the error is due to missing pricing configuration.
func IsNoPricingError(err error) bool {
	return errors.Is(err, billing.ErrNoPricing)
}

// AsyncChargeAuth enqueues a billing charge for the authenticated entity.
func (c *Core) AsyncChargeAuth(auth *AuthResult, model, requestID string, usage billing.UsageInfo) {
	if c.SubscriptionService != nil {
		var ownerID, keyID string
		if auth.IsTenant() {
			ownerID = c.getTenantOwnerID(auth.TenantKey.TenantID)
			keyID = auth.TenantKey.ID
		} else if auth.IsSubUser() {
			ownerID = c.getTenantOwnerID(auth.SubUserKey.TenantID)
			keyID = auth.SubUserKey.ID
		} else if auth.MemberTenantID != "" {
			ownerID = c.getTenantOwnerID(auth.MemberTenantID)
			keyID = auth.APIKeyID
		} else {
			ownerID = auth.User.ID
			keyID = auth.APIKeyID
		}
		if ownerID != "" {
			result, err := c.SubscriptionService.CheckAccess(ownerID, model)
			if err == nil && result.Covered && result.WithinQuota {
				if recErr := c.SubscriptionService.RecordUsage(ownerID, model, requestID, usage, keyID); recErr != nil {
					slog.Error("subscription usage record failed", "user_id", ownerID, "model", model, "error", recErr)
				}
				if auth.IsSubUser() {
					c.incrementSubUserQuotaForSubscription(auth.SubUser.ID, auth.SubUserKey.TenantID, model, requestID, usage, auth.SubUserKey.ID)
				}
				return
			}
		}
	}

	job := billingJob{model: model, requestID: requestID, usage: usage}
	if auth.IsTenant() {
		job.tenantID = auth.TenantKey.TenantID
		job.tenantKeyID = auth.TenantKey.ID
	} else if auth.IsSubUser() {
		job.subUserID = auth.SubUser.ID
		job.tenantID = auth.SubUserKey.TenantID
		job.subUserKeyID = auth.SubUserKey.ID
	} else if auth.MemberTenantID != "" {
		job.tenantID = auth.MemberTenantID
		job.tenantKeyID = auth.APIKeyID
	} else {
		job.userID = auth.User.ID
		job.apiKeyID = auth.APIKeyID
	}

	// Defensive check: warn if apiKeyID is missing for user-key scenarios
	if job.userID != "" && job.apiKeyID == "" && job.tenantID == "" {
		slog.Warn("billingJob missing apiKeyID for user-key request",
			"user_id", job.userID, "model", model, "request_id", requestID)
	}

	select {
	case c.billingCh <- job:
	default:
		metrics.Get().BillingQueueOverflow.Add(1)
		slog.Warn("billing queue full, trying overflow",
			"model", model, "request_id", requestID)
		select {
		case c.overflowSem <- struct{}{}:
			go func() {
				defer func() { <-c.overflowSem }()
				c.executeBillingJob(job)
			}()
		default:
			metrics.Get().BillingJobsDropped.Add(1)
			slog.Error("billing overflow full, dropping job",
				"model", model, "request_id", requestID)
		}
	}
}

func (c *Core) getTenantOwnerID(tenantID string) string {
	tenant, err := c.Store.GetTenantByID(tenantID)
	if err != nil {
		slog.Warn("failed to get tenant for subscription check", "tenant_id", tenantID, "error", err)
		return ""
	}
	return tenant.OwnerID
}

// resolveMemberTenantID returns the tenant a user belongs to for billing/routing
// purposes: the earliest-joined active membership, or "" if the user is not a
// member of any active tenant. ListTenantsByUser already filters status='active'
// and orders by joined_at, so the first row is authoritative.
func resolveMemberTenantID(s store.Store, userID string) string {
	tenants, err := s.ListTenantsByUser(userID)
	if err != nil {
		slog.Warn("failed to resolve member tenant", "user_id", userID, "error", err)
		return ""
	}
	if len(tenants) == 0 {
		return ""
	}
	return tenants[0].ID
}

// MemberTenantIDForUser exposes member-tenant resolution for JWT/console
// callers that carry a bare user id instead of an API-key AuthResult.
func (c *Core) MemberTenantIDForUser(userID string) string {
	return resolveMemberTenantID(c.Store, userID)
}

func (c *Core) incrementSubUserQuotaForSubscription(subUserID, tenantID, model, requestID string, usage billing.UsageInfo, apiKeyID string) {
	p, err := c.BillingService.GetTenantPricingPublic(tenantID, model)
	if err != nil || p == nil {
		return
	}
	cost := billing.CalculateCost(usage, p)
	if cost <= 0 {
		return
	}
	if err := c.Store.IncrementSubUserQuotaUsed(subUserID, cost); err != nil {
		slog.Error("failed to increment sub-user quota for subscription usage",
			"sub_user_id", subUserID, "amount", cost, "error", err)
	}
	tokens := store.TokenUsage{
		PromptTokens:          usage.PromptTokens,
		CompletionTokens:      usage.CompletionTokens,
		CacheReadTokens:       usage.CacheReadTokens,
		CacheCreationTokens:   usage.CacheCreationTokens,
		CacheCreation5mTokens: usage.CacheCreation5mTokens,
		CacheCreation1hTokens: usage.CacheCreation1hTokens,
	}
	if err := c.Store.RecordSubUserTransaction(tenantID, subUserID, cost, model, requestID, tokens, apiKeyID); err != nil {
		slog.Error("failed to record sub-user transaction for subscription usage",
			"sub_user_id", subUserID, "tenant_id", tenantID, "amount", cost, "error", err)
	}
}

// ChargeTenantSync settles a tenant charge synchronously and returns the cost.
// Subscription quota (on the tenant owner) is consumed first; when it fully
// covers the model within quota, the cost returned is 0. Used by detached
// workers (e.g. async image tasks) that need the cost to persist on the task.
func (c *Core) ChargeTenantSync(tenantID, tenantKeyID, model, requestID string, usage billing.UsageInfo) (float64, error) {
	if c.SubscriptionService != nil {
		ownerID := c.getTenantOwnerID(tenantID)
		if ownerID != "" {
			if result, err := c.SubscriptionService.CheckAccess(ownerID, model); err == nil && result.Covered && result.WithinQuota {
				if recErr := c.SubscriptionService.RecordUsage(ownerID, model, requestID, usage, tenantKeyID); recErr != nil {
					slog.Error("subscription usage record failed", "user_id", ownerID, "model", model, "error", recErr)
				}
				return 0, nil
			}
		}
	}
	if err := c.BillingService.TenantCharge(tenantID, model, requestID, usage, tenantKeyID); err != nil {
		return 0, err
	}
	return c.costFor(model, usage, func() (*store.ModelPricing, error) {
		return c.BillingService.GetTenantPricingPublic(tenantID, model)
	}), nil
}

// ChargeSubUserSync settles a sub-user charge synchronously and returns the cost.
// Mirrors AsyncChargeAuth's sub-user branch: subscription quota (on the tenant
// owner) first, otherwise charge the sub-user.
func (c *Core) ChargeSubUserSync(subUserID, subUserKeyID, tenantID, model, requestID string, usage billing.UsageInfo) (float64, error) {
	if c.SubscriptionService != nil {
		ownerID := c.getTenantOwnerID(tenantID)
		if ownerID != "" {
			if result, err := c.SubscriptionService.CheckAccess(ownerID, model); err == nil && result.Covered && result.WithinQuota {
				if recErr := c.SubscriptionService.RecordUsage(ownerID, model, requestID, usage, subUserKeyID); recErr != nil {
					slog.Error("subscription usage record failed", "user_id", ownerID, "model", model, "error", recErr)
				}
				c.incrementSubUserQuotaForSubscription(subUserID, tenantID, model, requestID, usage, subUserKeyID)
				return 0, nil
			}
		}
	}
	if err := c.BillingService.ChargeSubUser(subUserID, tenantID, model, requestID, usage, subUserKeyID); err != nil {
		return 0, err
	}
	return c.costFor(model, usage, func() (*store.ModelPricing, error) {
		return c.BillingService.GetTenantPricingPublic(tenantID, model)
	}), nil
}

// costFor computes the cost from a pricing lookup; returns 0 on lookup failure
// (the charge already succeeded, the figure is informational on the task row).
func (c *Core) costFor(model string, usage billing.UsageInfo, lookup func() (*store.ModelPricing, error)) float64 {
	p, err := lookup()
	if err != nil || p == nil {
		slog.Warn("pricing lookup failed for cost display", "model", model, "error", err)
		return 0
	}
	return billing.CalculateCost(usage, p)
}

// isContextWindowExceededError reports whether the response body indicates the
// prompt exceeded the model's context limit. This is a client error and must
// not trigger upstream circuit-breaker failures or failover.
func isContextWindowExceededError(body []byte) bool {
	lower := strings.ToLower(string(body))
	for _, phrase := range []string{
		"exceeds the context window",
		"context_length_exceeded",
		"context window exceeded",
		"maximum context length",
		"prompt is too long",
	} {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

// retryConfig holds retry policy parameters
type retryConfig struct {
	maxRetries           int
	retryableStatusCodes map[int]bool
	baseDelay            time.Duration
	maxDelay             time.Duration
}

// isRetryableError determines if an error or status code is retryable
func isRetryableError(err error, statusCode int, respBody []byte, cfg retryConfig) bool {
	// Network errors are retryable
	if err != nil {
		return circuit.IsUpstreamFailure(err)
	}

	// Context window errors are NOT retryable (client error)
	if statusCode == 429 || statusCode >= 500 {
		if isContextWindowExceededError(respBody) {
			return false
		}
	}

	// Check against configured retryable status codes
	return cfg.retryableStatusCodes[statusCode]
}

// calculateBackoff computes exponential backoff with jitter
func calculateBackoff(attempt int, cfg retryConfig) time.Duration {
	// delay = min(baseDelay * 2^attempt, maxDelay)
	delay := cfg.baseDelay * (1 << uint(attempt))
	if delay > cfg.maxDelay {
		delay = cfg.maxDelay
	}

	// Add jitter: ±25%
	jitter := float64(delay) * 0.25
	jitterOffset := (rand.Float64()*2 - 1) * jitter

	finalDelay := time.Duration(float64(delay) + jitterOffset)
	if finalDelay < 0 {
		finalDelay = 0
	}

	return finalDelay
}

// FailoverResult holds the result of a successful upstream request.
type FailoverResult struct {
	Response *http.Response
	Upstream *balancer.Upstream
	Cancel   context.CancelFunc
}

// Failover tries upstreams in round-robin order with circuit breaker checking.
// The caller is responsible for calling result.Cancel() after processing the response.
// extraHeaders, if non-nil, are merged into the upstream request (e.g. anthropic-beta).
func (c *Core) Failover(ctx context.Context, upstreams []balancer.Upstream, body []byte, method, upstreamPath string, extraHeaders http.Header) (*FailoverResult, error) {
	cfg := c.CfgHolder.Get()

	n := len(upstreams)
	tried := make(map[int]bool, n)
	start := uint64(0) // priority mode: always try primary upstream first
	var lastErrResult *FailoverResult

	for len(tried) < n {
		var upstream *balancer.Upstream
		var idx int
		found := false
		for i := 0; i < n; i++ {
			idx = int((start + uint64(i)) % uint64(n))
			if tried[idx] {
				continue
			}
			if upstreams[idx].Breaker.AllowRequest() {
				upstream = &upstreams[idx]
				found = true
				break
			}
			tried[idx] = true
		}
		if !found {
			break
		}
		tried[idx] = true

		// Build retry configuration
		retryCfg := retryConfig{
			maxRetries:           cfg.CircuitBreaker.MaxRetries,
			retryableStatusCodes: make(map[int]bool),
			baseDelay:            cfg.CircuitBreaker.RetryBaseDelay,
			maxDelay:             cfg.CircuitBreaker.RetryMaxDelay,
		}
		for _, code := range cfg.CircuitBreaker.RetryableStatusCodes {
			retryCfg.retryableStatusCodes[code] = true
		}

		var lastErr error
		var lastErrBody []byte
		var lastResp *http.Response
		var lastCancel context.CancelFunc

		// Retry loop: try up to 1 + maxRetries times
		for attempt := 0; attempt <= retryCfg.maxRetries; attempt++ {
			// Apply backoff delay (skip first attempt)
			if attempt > 0 {
				backoff := calculateBackoff(attempt-1, retryCfg)
				slog.Debug("retrying upstream request",
					"upstream", upstream.Config.BaseURL,
					"attempt", attempt+1,
					"backoff", backoff,
				)

				select {
				case <-time.After(backoff):
				case <-ctx.Done():
					if lastCancel != nil {
						lastCancel()
					}
					break // Exit retry loop, try next upstream
				}

				metrics.Get().UpstreamRetries.Add(1)
			}

			// Apply model_override
			upstreamBody := body
			if upstream.Config.ModelOverride != "" {
				upstreamBody = replaceModelInBody(body, upstream.Config.ModelOverride)
			}

			timeout := cfg.Server.RequestTimeout
			if timeout <= 0 {
				timeout = 30 * time.Minute
			}
			reqCtx, cancel := context.WithTimeout(ctx, timeout)

			// Determine upstream URL
			var upstreamURL string
			baseURL := strings.TrimRight(upstream.Config.BaseURL, "/")
			if strings.HasPrefix(upstreamPath, "/v1/messages") && strings.Contains(baseURL, "/openai") {
				// Anthropic handler passthrough: swap /openai → /anthropic for ppinfra-style providers
				baseURL = strings.Replace(baseURL, "/openai", "/anthropic", 1)
				upstreamURL = baseURL + upstreamPath
			} else {
				upstreamURL = baseURL + upstreamPath
			}

			if err := validateUpstreamURL(upstreamURL); err != nil {
				slog.Warn("upstream URL targets private network", "url", upstreamURL, "error", err)
			}

			upReq, err := http.NewRequestWithContext(reqCtx, method, upstreamURL, bytes.NewReader(upstreamBody))
			if err != nil {
				cancel()
				lastErr = err
				if attempt == 0 {
					slog.Error("failed to create upstream request", "error", err)
				}
				break // Request creation error is not retryable
			}

			upReq.Header.Set("Content-Type", "application/json")
			if strings.HasPrefix(upstreamPath, "/v1/messages") && strings.Contains(upstream.Config.BaseURL, "anthropic.com") {
				// Direct Anthropic API (api.anthropic.com) uses x-api-key
				upReq.Header.Set("x-api-key", upstream.Config.APIKey)
				upReq.Header.Set("anthropic-version", "2023-06-01")
			} else {
				// All other upstreams (including Anthropic proxies like ppinfra) use Bearer
				upReq.Header.Set("Authorization", "Bearer "+upstream.Config.APIKey)
			}

			// Merge caller-supplied headers with allowlist to prevent header injection.
			for k, vals := range extraHeaders {
				lower := strings.ToLower(k)
				if lower == "host" || lower == "authorization" || lower == "cookie" || lower == "x-api-key" {
					continue
				}
				for _, v := range vals {
					upReq.Header.Set(k, v)
				}
			}

			// Ensure anthropic-version is set for all Anthropic Messages API requests
			// (needed by proxies like ppinfra even when client doesn't send it).
			if strings.HasPrefix(upstreamPath, "/v1/messages") {
				if upReq.Header.Get("anthropic-version") == "" {
					upReq.Header.Set("anthropic-version", "2023-06-01")
				}
			}

			if attempt == 0 {
				metrics.Get().UpstreamRequests.Add(1)
			}

			resp, err := c.Client.Do(upReq)
			if err != nil {
				cancel()
				lastErr = err
				if !isRetryableError(err, 0, nil, retryCfg) {
					break // Non-retryable error
				}
				if attempt < retryCfg.maxRetries {
					continue // Retry
				}
				break // Max retries reached
			}

			// Streaming response special handling: once connection is established, return immediately
			if resp.StatusCode == 200 {
				contentType := resp.Header.Get("Content-Type")
				if strings.Contains(contentType, "text/event-stream") {
					if lastErrResult != nil {
						lastErrResult.Response.Body.Close()
						lastErrResult.Cancel()
					}
					upstream.Breaker.RecordSuccess()
					if attempt > 0 {
						metrics.Get().UpstreamRetriesSucceeded.Add(1)
					}
					return &FailoverResult{
						Response: resp,
						Upstream: upstream,
						Cancel:   cancel,
					}, nil
				}
			}

			if resp.StatusCode == 429 || resp.StatusCode >= 500 {
				// Read body before deciding whether to record failure.
				errBody, _ := io.ReadAll(resp.Body)
				resp.Body.Close()

				// Context-window errors are client errors (input too long), not upstream
				// faults. Return immediately without penalising the circuit breaker or
				// attempting failover to other upstreams that would fail for the same reason.
				if isContextWindowExceededError(errBody) {
					slog.Warn("context window exceeded, returning to client without failover",
						"upstream", upstream.Config.BaseURL,
						"path", upstreamPath,
						"status", resp.StatusCode,
						"body", truncateStr(string(errBody), 1024),
					)
					resp.Body = io.NopCloser(bytes.NewReader(errBody))
					return &FailoverResult{
						Response: resp,
						Upstream: upstream,
						Cancel:   cancel,
					}, nil
				}

				lastErrBody = errBody
				lastResp = resp
				lastCancel = cancel

				if !isRetryableError(nil, resp.StatusCode, errBody, retryCfg) {
					break // Non-retryable status code
				}

				if attempt < retryCfg.maxRetries {
					continue // Retry
				}
				break // Max retries reached
			}

			// Success response
			if lastErrResult != nil {
				lastErrResult.Response.Body.Close()
				lastErrResult.Cancel()
			}
			upstream.Breaker.RecordSuccess()
			if attempt > 0 {
				metrics.Get().UpstreamRetriesSucceeded.Add(1)
			}
			return &FailoverResult{
				Response: resp,
				Upstream: upstream,
				Cancel:   cancel,
			}, nil
		}

		// All retries failed, record failure once to circuit breaker
		if lastErr != nil {
			if circuit.IsUpstreamFailure(lastErr) {
				upstream.Breaker.RecordFailure()
			}
			metrics.Get().UpstreamFailures.Add(1)
			metrics.Get().UpstreamFailovers.Add(1)
			slog.Warn("upstream request failed after retries",
				"upstream", upstream.Config.BaseURL,
				"attempts", retryCfg.maxRetries+1,
				"error", lastErr,
			)
			continue // Try next upstream
		}

		if lastResp != nil {
			upstream.Breaker.RecordFailure()
			metrics.Get().UpstreamFailures.Add(1)
			metrics.Get().UpstreamFailovers.Add(1)

			slog.Warn("upstream error response after retries",
				"upstream", upstream.Config.BaseURL,
				"attempts", retryCfg.maxRetries+1,
				"status", lastResp.StatusCode,
				"body", truncateStr(string(lastErrBody), 1024),
			)

			// Close previous saved error response if any
			if lastErrResult != nil {
				lastErrResult.Response.Body.Close()
				lastErrResult.Cancel()
			}
			// Replace body so caller can still read it
			lastResp.Body = io.NopCloser(bytes.NewReader(lastErrBody))
			lastErrResult = &FailoverResult{
				Response: lastResp,
				Upstream: upstream,
				Cancel:   lastCancel,
			}
			continue // Try next upstream
		}
	}

	// Return last error response if available, otherwise all unavailable
	if lastErrResult != nil {
		slog.Error("all upstreams exhausted, returning last error",
			"upstream", lastErrResult.Upstream.Config.BaseURL,
			"status", lastErrResult.Response.StatusCode,
		)
		return lastErrResult, nil
	}
	return nil, fmt.Errorf("all upstream providers are unavailable")
}

// AsyncCharge enqueues a billing charge for a user to be processed by the worker pool.
func (c *Core) AsyncCharge(userID, model, requestID string, usage billing.UsageInfo, apiKeyID string) {
	select {
	case c.billingCh <- billingJob{userID: userID, model: model, requestID: requestID, usage: usage, apiKeyID: apiKeyID}:
	default:
		slog.Warn("billing queue full, charging synchronously",
			"user_id", userID, "model", model, "request_id", requestID)
		go func() {
			if err := c.BillingService.Charge(userID, model, requestID, usage, apiKeyID); err != nil {
				slog.Error("billing charge failed", "user_id", userID, "model", model, "request_id", requestID, "error", err)
			}
		}()
	}
}

// ReadBody reads the request body with size limit from config.
func (c *Core) ReadBody(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	cfg := c.CfgHolder.Get()
	maxBytes := cfg.Server.MaxRequestBodyBytes
	if maxBytes <= 0 {
		maxBytes = 20 << 20 // 20MB default, needed for large context windows
	}
	limitedReader := http.MaxBytesReader(w, r.Body, maxBytes)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// InjectStreamOptions adds stream_options with include_usage: true.
func InjectStreamOptions(bodyBytes []byte, reqBody map[string]any) []byte {
	reqBody["stream_options"] = map[string]any{"include_usage": true}
	out, err := json.Marshal(reqBody)
	if err != nil {
		return bodyBytes
	}
	return out
}

// ExtractUsageFromJSON extracts usage from a JSON response body.
func ExtractUsageFromJSON(body []byte) billing.UsageInfo {
	return extractUsageFromJSON(body)
}

// ExtractUsageFromSSE extracts usage from a single SSE data payload.
func ExtractUsageFromSSE(data string) billing.UsageInfo {
	return extractUsageFromSSE(data)
}

// GetRequestID extracts or generates a request ID.
func GetRequestID(r *http.Request) string {
	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = fmt.Sprintf("req_%d", time.Now().UnixNano())
	}
	return requestID
}

// UpstreamProtocols 返回上游声明的协议集合，优先读 Protocols 数组，
// 数组为空时回退到旧 Protocol 单值字段（兜底兼容老数据）。
// 返回的协议字符串已小写化、去空白、去重。
func UpstreamProtocols(cfg config.UpstreamConfig) []string {
	seen := make(map[string]bool)
	var out []string
	add := func(p string) {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		out = append(out, p)
	}
	for _, p := range cfg.Protocols {
		add(p)
	}
	add(cfg.Protocol)
	return out
}

// HasProtocol 报告上游是否声明了某协议（基于 Protocols 数组，回退到 Protocol 单值）。
func HasProtocol(cfg config.UpstreamConfig, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	if want == "" {
		return false
	}
	for _, p := range UpstreamProtocols(cfg) {
		if p == want {
			return true
		}
	}
	return false
}

// IsOpenAIChatCompatible 报告上游是否声明了 openai 或 openai-compatible 协议，
// 用于 /v1/messages 与 /v1/responses 入口做协议转换兜底时挑选上游。
// 旧值 "openai-compatible-responses" 已被废弃但仍存在于历史数据中——
// 视为 OpenAI Chat 兼容以保持旧配置可继续被 /v1/chat/completions 路由。
func IsOpenAIChatCompatible(cfg config.UpstreamConfig) bool {
	for _, p := range UpstreamProtocols(cfg) {
		if p == "openai" || p == "openai-compatible" || p == "openai-compatible-responses" {
			return true
		}
	}
	return false
}

// IsAnthropicAPI returns true if the upstream declares the Anthropic Messages
// API protocol. Detection is based on the Protocols array (with fallback to the
// legacy Protocol field for older rows).
func IsAnthropicAPI(cfg config.UpstreamConfig) bool {
	return HasProtocol(cfg, "anthropic")
}

func isAnthropicAPI(cfg config.UpstreamConfig) bool { return IsAnthropicAPI(cfg) }

// IsGeminiAPI returns true if the upstream declares the Gemini API protocol.
func IsGeminiAPI(cfg config.UpstreamConfig) bool {
	return HasProtocol(cfg, "gemini")
}

// isGeminiAPI is deprecated, use IsGeminiAPI instead.
func isGeminiAPI(cfg config.UpstreamConfig) bool { return IsGeminiAPI(cfg) }

// IsResponsesAPI returns true if the upstream declares the OpenAI Responses API
// protocol. Includes the legacy "openai-compatible-responses" value.
func IsResponsesAPI(cfg config.UpstreamConfig) bool {
	for _, p := range UpstreamProtocols(cfg) {
		if p == "responses" || p == "openai-compatible-responses" {
			return true
		}
	}
	return false
}

// IsOpenAICompatible returns true if the upstream declares an OpenAI Chat
// Completions-compatible protocol. Used by the /v1/chat/completions entry to
// filter upstreams for strict passthrough.
func IsOpenAICompatible(cfg config.UpstreamConfig) bool {
	return IsOpenAIChatCompatible(cfg)
}
