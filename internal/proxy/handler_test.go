package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zhulang/llm-gateway/internal/apikey"
	"github.com/zhulang/llm-gateway/internal/balancer"
	"github.com/zhulang/llm-gateway/internal/billing"
	"github.com/zhulang/llm-gateway/internal/config"
	"github.com/zhulang/llm-gateway/internal/router"
	"github.com/zhulang/llm-gateway/internal/store"
)

// nopStore is a minimal Store implementation for proxy tests.
// 嵌入 store.Store 后，未显式 stub 的方法在编译期自动满足接口；
// 若运行时被调用会 panic，但代理 handler 测试只命中下方显式实现的方法子集。
type nopStore struct {
	store.Store
}

func (nopStore) Close() error                                          { return nil }
func (nopStore) CreateUser(_, _, _ string) (*store.User, error)        { return nil, fmt.Errorf("nop") }
func (nopStore) Authenticate(_, _ string) (*store.User, error)         { return nil, fmt.Errorf("nop") }
func (nopStore) UserCount() (int64, error)                             { return 0, nil }
func (nopStore) GetUserByID(_ string) (*store.User, error)             { return nil, fmt.Errorf("nop") }
func (nopStore) GetUserByPhone(_ string) (*store.User, error)          { return nil, fmt.Errorf("nop") }
func (nopStore) MarkFirstRechargeGranted(_ string) (bool, error)       { return false, nil }
func (nopStore) ListUsers(_, _ int) ([]store.User, int, error)         { return nil, 0, nil }
func (nopStore) ListUsersWithBalance(_, _ int, _ string) ([]store.UserWithBalance, int, int, int, float64, error) {
	return nil, 0, 0, 0, 0, nil
}
func (nopStore) UpdateUserStatus(_, _ string) error                    { return nil }
func (nopStore) UpdatePassword(_, _ string) error                      { return nil }
func (nopStore) GetAdminDashboardStats() (*store.AdminDashboardStats, error) {
	return nil, fmt.Errorf("nop")
}
func (nopStore) GetUserPricing(_, _ string) (*store.UserPricing, error) { return nil, nil }
func (nopStore) RecordRequestFailure(_, _ string, _ int) error             { return nil }
func (nopStore) GetAdminConsumptionStats(_ int) (*store.AdminConsumptionStats, error) {
	return nil, fmt.Errorf("nop")
}
func (nopStore) ListModels() ([]store.Model, error)                    { return nil, nil }
func (nopStore) GetModel(_ int64) (*store.Model, error)                { return nil, fmt.Errorf("nop") }
func (nopStore) GetModelByName(_ string) (*store.Model, error)         { return nil, fmt.Errorf("nop") }
func (nopStore) CreateModel(_, _, _ string, _ []store.Upstream) (*store.Model, error) {
	return nil, fmt.Errorf("nop")
}
func (nopStore) UpdateModel(_ int64, _, _, _ string, _ []store.Upstream) (*store.Model, error) {
	return nil, fmt.Errorf("nop")
}
func (nopStore) DeleteModel(_ int64) error                             { return nil }
func (nopStore) CreateAPIKey(_, _, _, _ string, _ *int) (*store.APIKey, error) { return nil, fmt.Errorf("nop") }
func (nopStore) ListAPIKeysByUser(_ string) ([]store.APIKey, error)    { return nil, nil }
func (nopStore) GetAPIKeyByHash(_ string) (*store.APIKey, error)       { return nil, fmt.Errorf("nop") }
func (nopStore) GetActiveAPIKeyByIDAndUser(_, _ string) (*store.APIKey, error) { return nil, nil }
func (nopStore) GetTenantAPIKeyByHash(_ string) (*store.TenantAPIKey, error)         { return nil, fmt.Errorf("nop") }
func (nopStore) GetTenantSubUserKeyByHash(_ string) (*store.TenantSubUserKey, error) { return nil, fmt.Errorf("nop") }
func (nopStore) DeleteAPIKey(_, _ string) error                        { return nil }
func (nopStore) RevokeAllAPIKeys(_ string) (int, error)                { return 0, nil }
func (nopStore) TouchAPIKeyLastUsed(_ string) error                    { return nil }
func (nopStore) BatchTouchAPIKeysLastUsed(_ []string) error            { return nil }
func (nopStore) GetBalance(_ string) (*store.Balance, error)           { return nil, fmt.Errorf("nop") }
func (nopStore) FreezeBalance(_ string, _ float64) error               { return nil }
func (nopStore) SettleBilling(_ string, _, _ float64, _, _ string, _ store.TokenUsage, _ string) error { return nil }
func (nopStore) DirectCharge(_ string, _ float64, _, _ string, _ store.TokenUsage, _ string) error   { return nil }
func (nopStore) UnfreezeBalance(_ string, _ float64) error             { return nil }
func (nopStore) Recharge(_ string, _ float64, _ string) error          { return nil }
func (nopStore) DeductForSubscription(_ string, _ float64, _ string) error { return nil }
func (nopStore) ListTransactions(_ string, _, _ int, _ string, _, _ *time.Time) ([]store.Transaction, int, *store.TransactionSums, error) {
	return nil, 0, nil, nil
}
func (nopStore) GetBillingStats(_ string, _ int) (*store.BillingStats, error) {
	return nil, fmt.Errorf("nop")
}
func (nopStore) ListPricing() ([]store.ModelPricing, error)            { return nil, nil }
func (nopStore) ListActivePricing() ([]store.ModelPricing, error)      { return nil, nil }
func (nopStore) UpsertPricing(_ string, _, _, _, _, _ float64, _ string, _ bool, _ []store.PricingTier, _ []store.TimeBasedPricingRule) error {
	return nil
}
func (nopStore) GetPricing(_ string) (*store.ModelPricing, error)      { return nil, fmt.Errorf("no pricing") }
func (nopStore) CreateOrder(_ string, _ float64, _ *string) (*store.Order, error) { return nil, fmt.Errorf("nop") }
func (nopStore) GetOrderByNo(_ string) (*store.Order, error)           { return nil, fmt.Errorf("nop") }
func (nopStore) MarkOrderPaid(_ string, _ []byte) error                { return nil }
func (nopStore) FulfillAlipayPaidOrder(_ string, _ []byte) error      { return nil }
func (nopStore) ListOrders(_ string, _, _ int) ([]store.Order, int, *store.OrderStatusCounts, error) { return nil, 0, nil, nil }
func (nopStore) ListAllOrders(_, _ int, _ string) ([]store.AdminOrder, int, error) { return nil, 0, nil }
func (nopStore) ExpireOrders() (int, error)                            { return 0, nil }
func (nopStore) ListSessions(_ string) ([]store.ChatSession, error)    { return nil, nil }
func (nopStore) CreateSession(_, _, _ string) (*store.ChatSession, error) {
	return nil, fmt.Errorf("nop")
}
func (nopStore) GetSession(_, _ string) (*store.ChatSession, error)    { return nil, fmt.Errorf("nop") }
func (nopStore) UpdateSessionTitle(_, _, _ string) error               { return nil }
func (nopStore) DeleteSession(_, _ string) error                       { return nil }
func (nopStore) ListMessages(_ string) ([]store.ChatMessage, error)    { return nil, nil }
func (nopStore) AddMessage(_ string, _, _ string, _ int, _ float64) (*store.ChatMessage, error) {
	return nil, fmt.Errorf("nop")
}
func (nopStore) CreateInvoiceTitle(_, _, _, _, _, _, _, _ string) (*store.InvoiceTitle, error) {
	return nil, fmt.Errorf("nop")
}
func (nopStore) UpdateInvoiceTitle(_ int64, _, _, _, _, _, _, _, _ string) (*store.InvoiceTitle, error) {
	return nil, fmt.Errorf("nop")
}
func (nopStore) DeleteInvoiceTitle(_ int64, _ string) error { return nil }
func (nopStore) ListInvoiceTitlesByUser(_ string) ([]store.InvoiceTitle, error) {
	return nil, nil
}
func (nopStore) GetInvoiceTitle(_ int64, _ string) (*store.InvoiceTitle, error) {
	return nil, fmt.Errorf("nop")
}
func (nopStore) GetInvoiceTitleByID(_ int64) (*store.InvoiceTitle, error) {
	return nil, fmt.Errorf("nop")
}
func (nopStore) SetDefaultInvoiceTitle(_ int64, _ string) error { return nil }
func (nopStore) ListAvailableOrders(_ string) ([]store.Order, error) { return nil, nil }
func (nopStore) CreateInvoiceRequest(_ string, _ int64, _, _ string, _ []string) (*store.InvoiceRequest, error) {
	return nil, fmt.Errorf("nop")
}
func (nopStore) ListInvoiceRequests(_ string, _, _ int) ([]store.InvoiceRequest, int, error) {
	return nil, 0, nil
}
func (nopStore) GetInvoiceRequest(_ int64, _ string) (*store.InvoiceRequest, error) {
	return nil, fmt.Errorf("nop")
}
func (nopStore) GetInvoiceRequestOrders(_ int64) ([]store.InvoiceRequestOrder, error) {
	return nil, nil
}
func (nopStore) CancelInvoiceRequest(_ int64, _ string) error         { return nil }
func (nopStore) UpdateInvoiceRequestStatus(_ int64, _ string) error   { return nil }
func (nopStore) CompleteInvoiceRequest(_ int64, _, _ string) error    { return nil }
func (nopStore) RejectInvoiceRequest(_ int64, _ string) error         { return nil }
func (nopStore) AdminListInvoiceRequests(_ string, _, _ int) ([]store.InvoiceRequestDetail, int, error) {
	return nil, 0, nil
}

// Managed API key methods

func (nopStore) DeleteUser(_ string) error { return nil }

// Announcement methods
func (nopStore) ListAnnouncements(_, _ int) ([]store.Announcement, int, error) { return nil, 0, nil }
func (nopStore) CreateAnnouncement(_, _, _, _, _, _ string) (*store.Announcement, error) { return nil, fmt.Errorf("nop") }
func (nopStore) GetAnnouncementByID(_ int64) (*store.Announcement, error) { return nil, fmt.Errorf("nop") }
func (nopStore) UpdateAnnouncement(_ int64, _, _, _, _, _ string) (*store.Announcement, error) { return nil, fmt.Errorf("nop") }
func (nopStore) DeleteAnnouncement(_ int64) error { return nil }
func (nopStore) ListPublishedAnnouncements() ([]store.Announcement, error) { return nil, nil }

// Pricing change log methods
func (nopStore) InsertPricingChangeLog(_, _, _ string, _, _ map[string]any) error { return nil }
func (nopStore) ListPricingChangeLogs(_, _ int) ([]store.PricingChangeLog, int, error) { return nil, 0, nil }

func (nopStore) MarkAUPAccepted(_ string) error { return nil }

// Subscription methods
func (nopStore) ListSubscriptionPlans() ([]store.SubscriptionPlan, error) { return nil, nil }
func (nopStore) GetSubscriptionPlan(_ int) (*store.SubscriptionPlan, error) { return nil, nil }
func (nopStore) GetSubscriptionPlanModels(_ int) ([]string, error) { return nil, nil }
func (nopStore) GetActiveSubscription(_ string) (*store.UserSubscription, error) { return nil, nil }
func (nopStore) CreateSubscription(_ string, _ int, _ time.Time, _ string) (*store.UserSubscription, error) { return nil, nil }
func (nopStore) UpgradeSubscriptionTx(_ store.UpgradeSubscriptionParams) (*store.UserSubscription, error) { return nil, nil }
func (nopStore) CancelSubscription(_ string) error { return nil }
func (nopStore) ExpireSubscription(_ string) error { return nil }
func (nopStore) ExpireExpiredSubscriptions() (int, error) { return 0, nil }
func (nopStore) ExpireUserSubscriptionsByBrand(_, _ string) error { return nil }
func (nopStore) GetSubscriptionTotalUsage(_ string, _ time.Time) (float64, error) { return 0, nil }
func (nopStore) IncrementSubscriptionUsage(_, _, _ string, _ time.Time, _ store.TokenUsage, _ float64) error { return nil }
func (nopStore) GetSubscriptionUsageSummary(_ string, _ time.Time, _ float64) (*store.SubscriptionUsageSummary, error) { return nil, nil }
func (nopStore) CreateSubscriptionOrder(_ string, _ int, _ float64, _ string) (*store.SubscriptionOrder, error) { return nil, nil }
func (nopStore) CompleteSubscriptionOrder(_, _, _ string) error { return nil }
func (nopStore) ListUserSubscriptions(_, _ int) ([]store.UserSubscription, int, error) { return nil, 0, nil }
func (nopStore) ListUserSubscriptionHistory(_ string) ([]store.UserSubscription, error) { return nil, nil }
func (nopStore) GetSubscriptionOrderStats(_ int) (*store.SubscriptionOrderStats, error) { return nil, nil }
func (nopStore) ListAllSubscriptionOrders(_, _ int, _, _ string) ([]store.AdminSubscriptionOrder, int, error) { return nil, 0, nil }
func (nopStore) ListSubscriptionUsersUsage(_, _ int, _, _, _ string) ([]store.AdminSubscriptionUserUsage, int, int, float64, error) { return nil, 0, 0, 0, nil }
func (nopStore) RecordSubscriptionTransaction(_, _, _, _ string, _ float64, _ store.TokenUsage) error { return nil }

// Tenant pricing stubs
func (nopStore) ListTenantPricing(_ string) ([]store.TenantPricing, error) { return nil, nil }
func (nopStore) GetTenantPricing(_, _ string) (*store.TenantPricing, error) { return nil, nil }
func (nopStore) UpsertTenantPricing(_, _ string, _, _, _, _, _ float64, _ string, _ bool, _ []store.PricingTier, _ *float64, _ string) error { return nil }
func (nopStore) DeleteTenantPricing(_, _ string) error { return nil }
func (nopStore) HasTenantCustomPricing(_ string) (bool, error) { return false, nil }
func (nopStore) GetUserPrimaryPricingTenant(_ string) (string, error) { return "", nil }

// Tenant model upstream stubs
func (nopStore) ListTenantModelUpstreams(_ string) ([]store.TenantModelUpstream, error) { return nil, nil }
func (nopStore) ListAllTenantModelUpstreams() ([]store.TenantModelUpstream, error) { return nil, nil }
func (nopStore) ReplaceTenantModelUpstreams(_, _ string, _ []store.TenantModelUpstream) error { return nil }
func (nopStore) DeleteTenantModelUpstreams(_, _ string) error { return nil }

func newTestConfig(upstreamURLs ...string) *config.Config {
	upstreams := make([]config.UpstreamConfig, len(upstreamURLs))
	for i, u := range upstreamURLs {
		upstreams[i] = config.UpstreamConfig{
			Provider:  fmt.Sprintf("provider-%d", i),
			Protocol: "openai",
			BaseURL:   u,
			APIKey:    fmt.Sprintf("upstream-key-%d", i),
			Weight:    1,
		}
	}
	return &config.Config{
		Server: config.ServerConfig{
			Port:                8080,
			AuthTokens:          []string{"valid-token"},
			RequestTimeout:      5 * time.Second,
			MaxRequestBodyBytes: 1 << 20, // 1MB
		},
		Models: []config.ModelConfig{
			{
				Name:      "test-model",
				Upstreams: upstreams,
			},
		},
		CircuitBreaker: config.CircuitBreakerConfig{
			FailureThreshold:    5,
			RecoveryTimeout:     30 * time.Second,
			HalfOpenMaxRequests: 1,
		},
	}
}

func setupHandler(cfg *config.Config) *Handler {
	holder := config.NewHolder(cfg)
	rt := router.NewFromConfig(cfg)
	lb := balancer.NewRoundRobin()
	kc := apikey.NewCache(100, 5*time.Minute)
	s := nopStore{}

	// Pre-populate the cache with a test user for auth tokens.
	for _, token := range cfg.Server.AuthTokens {
		hash := apikey.HashAPIKey(token)
		kc.Set(hash, &apikey.CachedAuth{
			User: &store.User{
				ID:     "test-user-id",
				Phone:  "13800000000",
				Role:   "admin",
				Status: "active",
			},
		})
	}

	bs := billing.NewBillingService(s, nil, nil)
	tb := apikey.NewTouchBatcher(s, 5*time.Second)
	core := NewCore(holder, rt, lb, s, kc, bs, NewSharedHTTPClient(), tb)
	return NewHandler(holder, rt, lb, s, kc, bs, NewSharedHTTPClient(), tb, core)
}

func TestProxy_NonStream_Success(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the upstream receives the upstream API key, not the client token
		auth := r.Header.Get("Authorization")
		if auth != "Bearer upstream-key-0" {
			t.Errorf("expected upstream auth 'Bearer upstream-key-0', got %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "chatcmpl-123",
			"model": "test-model",
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "hello"}},
			},
		})
	}))
	defer upstream.Close()

	cfg := newTestConfig(upstream.URL)
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{
		"model":    "test-model",
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
	})
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["model"] != "test-model" {
		t.Fatalf("expected model 'test-model', got %v", resp["model"])
	}
}

func TestProxy_Auth_MissingToken(t *testing.T) {
	cfg := newTestConfig("http://localhost:1")
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{"model": "test-model"})
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestProxy_Auth_InvalidToken(t *testing.T) {
	cfg := newTestConfig("http://localhost:1")
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{"model": "test-model"})
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer wrong-token")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestProxy_ModelNotFound(t *testing.T) {
	cfg := newTestConfig("http://localhost:1")
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{"model": "nonexistent"})
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestProxy_Failover_FirstUpstreamFails(t *testing.T) {
	// Primary upstream returns 500. With failover retry, the request should
	// immediately try the backup upstream and succeed.
	upstream1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, `{"error":"internal"}`)
	}))
	defer upstream1.Close()

	upstream2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "chatcmpl-456",
			"model": "test-model",
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "ok"}},
			},
		})
	}))
	defer upstream2.Close()

	cfg := newTestConfig(upstream1.URL, upstream2.URL)
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{
		"model":    "test-model",
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
	})
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	// Should immediately failover to backup and succeed
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 after immediate failover, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestProxy_Failover_429Triggers(t *testing.T) {
	// Primary upstream returns 429. With failover retry, the request should
	// immediately try the backup upstream and succeed.
	upstream1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprintln(w, `{"error":"rate limited"}`)
	}))
	defer upstream1.Close()

	upstream2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "chatcmpl-789",
			"model": "test-model",
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "recovered"}},
			},
		})
	}))
	defer upstream2.Close()

	cfg := newTestConfig(upstream1.URL, upstream2.URL)
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{
		"model":    "test-model",
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
	})
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	// Should immediately failover to backup and succeed
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 after immediate failover, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestProxy_ModelOverride_RewritesRequestAndResponse(t *testing.T) {
	var receivedModel string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(bodyBytes, &reqBody)
		receivedModel = reqBody["model"].(string)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "chatcmpl-override",
			"model": "provider-internal-model",
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "overridden"}},
			},
		})
	}))
	defer upstream.Close()

	cfg := newTestConfig(upstream.URL)
	cfg.Models[0].Upstreams[0].ModelOverride = "provider-internal-model"
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{
		"model":    "test-model",
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
	})
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify upstream received the overridden model
	if receivedModel != "provider-internal-model" {
		t.Fatalf("expected upstream to receive 'provider-internal-model', got %q", receivedModel)
	}

	// Verify response model is rewritten back to the client-requested name.
	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["model"] != "test-model" {
		t.Fatalf("expected response model 'test-model', got %v", resp["model"])
	}
}

// Cursor registers custom models with a "gw/" prefix and rejects responses
// whose model field differs from what it sent. The gateway must echo the
// client-supplied model string verbatim (prefix included) while still routing
// and billing on the normalized canonical name.
func TestProxy_GWPrefix_EchoesClientModel(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "chatcmpl-gw",
			"model": "provider-internal-model",
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "ok"}},
			},
		})
	}))
	defer upstream.Close()

	cfg := newTestConfig(upstream.URL)
	cfg.Models[0].Upstreams[0].ModelOverride = "provider-internal-model"
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{
		"model":    "gw/test-model",
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
	})
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["model"] != "gw/test-model" {
		t.Fatalf("expected response model echoed verbatim as 'gw/test-model', got %v", resp["model"])
	}
}

func TestProxy_GWPrefix_EchoesClientModel_Stream(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected flusher")
		}
		chunks := []string{
			`data: {"id":"chatcmpl-1","model":"provider-internal-model","choices":[{"delta":{"content":"hi"}}]}`,
			`data: [DONE]`,
		}
		for _, c := range chunks {
			fmt.Fprintf(w, "%s\n\n", c)
			flusher.Flush()
		}
	}))
	defer upstream.Close()

	cfg := newTestConfig(upstream.URL)
	cfg.Models[0].Upstreams[0].ModelOverride = "provider-internal-model"
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{
		"model":    "gw/test-model",
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
		"stream":   true,
	})
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"model":"gw/test-model"`) {
		t.Fatalf("expected streamed model echoed as 'gw/test-model', got: %s", rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "provider-internal-model") {
		t.Fatalf("upstream-internal model name leaked into response: %s", rr.Body.String())
	}
}

func TestProxy_Stream_SSERelay(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected flusher")
		}

		chunks := []string{
			`data: {"id":"chatcmpl-1","model":"test-model","choices":[{"delta":{"content":"hel"}}]}`,
			`data: {"id":"chatcmpl-1","model":"test-model","choices":[{"delta":{"content":"lo"}}]}`,
			`data: [DONE]`,
		}
		for _, chunk := range chunks {
			fmt.Fprintf(w, "%s\n\n", chunk)
			flusher.Flush()
		}
	}))
	defer upstream.Close()

	cfg := newTestConfig(upstream.URL)
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{
		"model":    "test-model",
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
		"stream":   true,
	})
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("expected Content-Type text/event-stream, got %q", ct)
	}

	respBody := rr.Body.String()
	if !strings.Contains(respBody, "data: ") {
		t.Fatal("expected SSE data lines in response")
	}
	if !strings.Contains(respBody, "[DONE]") {
		t.Fatal("expected [DONE] marker in response")
	}
}

func TestIsAnthropicAPI(t *testing.T) {
	tests := []struct {
		name     string
		cfg      config.UpstreamConfig
		expected bool
	}{
		{
			name:     "anthropic provider with anthropic.com URL",
			cfg:      config.UpstreamConfig{Provider: "anthropic", Protocol: "anthropic", BaseURL: "https://api.anthropic.com/v1"},
			expected: true,
		},
		{
			name:     "anthropic provider with third-party URL",
			cfg:      config.UpstreamConfig{Provider: "anthropic", Protocol: "anthropic", BaseURL: "https://4sapi.com"},
			expected: true,
		},
		{
			name:     "openai provider",
			cfg:      config.UpstreamConfig{Provider: "openai", Protocol: "openai", BaseURL: "https://api.openai.com"},
			expected: false,
		},
		{
			name:     "empty provider",
			cfg:      config.UpstreamConfig{Provider: "", BaseURL: "https://api.anthropic.com"},
			expected: false,
		},
		{
			name:     "multi-protocol upstream with anthropic",
			cfg:      config.UpstreamConfig{Provider: "agg", Protocols: []string{"openai", "anthropic"}, BaseURL: "https://agg.example"},
			expected: true,
		},
		{
			name:     "legacy Protocol-only fallback to anthropic",
			cfg:      config.UpstreamConfig{Provider: "anthropic", Protocol: "anthropic", BaseURL: "https://x"},
			expected: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAnthropicAPI(tt.cfg); got != tt.expected {
				t.Errorf("IsAnthropicAPI() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestProxy_NoOpenAIUpstream_Returns503(t *testing.T) {
	// When a model only has Anthropic-protocol upstreams, the OpenAI Chat
	// Completions entry must NOT cross-protocol convert. It must return 503
	// no_compatible_upstream instead.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("upstream should not be called under strict passthrough")
	}))
	defer upstream.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:                8080,
			AuthTokens:          []string{"valid-token"},
			RequestTimeout:      5 * time.Second,
			MaxRequestBodyBytes: 1 << 20,
		},
		Models: []config.ModelConfig{
			{
				Name: "claude-sonnet-4-6",
				Upstreams: []config.UpstreamConfig{
					{
						Provider: "anthropic",
						Protocol: "anthropic",
						BaseURL:  upstream.URL,
						APIKey:   "test-key",
						Weight:   1,
					},
				},
			},
		},
		CircuitBreaker: config.CircuitBreakerConfig{
			FailureThreshold:    5,
			RecoveryTimeout:     30 * time.Second,
			HalfOpenMaxRequests: 1,
		},
	}
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{
		"model":    "claude-sonnet-4-6",
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
	})
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "no_compatible_upstream") {
		t.Fatalf("expected no_compatible_upstream in body, got: %s", rr.Body.String())
	}
}

func TestProxy_MixedProtocolUpstreams_OnlyUsesOpenAI(t *testing.T) {
	// When a model has both OpenAI and Anthropic upstreams, /v1/chat/completions
	// must only route to OpenAI. If the OpenAI one is down, it returns 503
	// rather than cross-protocol failing over to Anthropic.
	var anthropicCalled bool
	anthropicUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		anthropicCalled = true
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"msg_01","type":"message","role":"assistant","content":[{"type":"text","text":"hi"}],"model":"claude","stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer anthropicUpstream.Close()

	openaiUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"error":"upstream gone"}`))
	}))
	defer openaiUpstream.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:                8080,
			AuthTokens:          []string{"valid-token"},
			RequestTimeout:      5 * time.Second,
			MaxRequestBodyBytes: 1 << 20,
		},
		Models: []config.ModelConfig{
			{
				Name: "mix-model",
				Upstreams: []config.UpstreamConfig{
					{
						Provider: "openai-provider",
						Protocol: "openai",
						BaseURL:  openaiUpstream.URL,
						APIKey:   "k",
						Weight:   1,
					},
					{
						Provider: "anthropic-provider",
						Protocol: "anthropic",
						BaseURL:  anthropicUpstream.URL,
						APIKey:   "k",
						Weight:   1,
					},
				},
			},
		},
		CircuitBreaker: config.CircuitBreakerConfig{
			FailureThreshold:    5,
			RecoveryTimeout:     30 * time.Second,
			HalfOpenMaxRequests: 1,
		},
	}
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{
		"model":    "mix-model",
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
	})
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if anthropicCalled {
		t.Error("Anthropic upstream must not be called from /v1/chat/completions under strict passthrough")
	}
	// Strict passthrough: OpenAI upstream returned 502, no cross-protocol failover
	// to Anthropic. The gateway forwards the upstream's last error status code.
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 (upstream last error, no cross-protocol failover), got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestProxy_PriorityFailover_AlwaysTriesPrimaryFirst(t *testing.T) {
	// With two upstreams, all requests should always go to the primary (index 0).
	var primaryHits, backupHits int

	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryHits++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "chatcmpl-primary",
			"model": "test-model",
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "from primary"}},
			},
		})
	}))
	defer primary.Close()

	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backupHits++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "chatcmpl-backup",
			"model": "test-model",
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "from backup"}},
			},
		})
	}))
	defer backup.Close()

	cfg := newTestConfig(primary.URL, backup.URL)
	h := setupHandler(cfg)

	// Send 5 requests — all should hit primary
	for i := 0; i < 5; i++ {
		body, _ := json.Marshal(map[string]any{
			"model":    "test-model",
			"messages": []map[string]string{{"role": "user", "content": "hi"}},
		})
		req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer valid-token")
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, rr.Code)
		}
	}

	if primaryHits != 5 {
		t.Errorf("expected primary to receive all 5 requests, got %d", primaryHits)
	}
	if backupHits != 0 {
		t.Errorf("expected backup to receive 0 requests, got %d", backupHits)
	}
}

func TestProxy_PriorityFailover_FallsBackToBackup(t *testing.T) {
	// Primary returns 500. With in-request failover retry, the very first request
	// should immediately try backup and succeed — no need to wait for breaker to open.
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, `{"error":"primary down"}`)
	}))
	defer primary.Close()

	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "chatcmpl-backup",
			"model": "test-model",
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "from backup"}},
			},
		})
	}))
	defer backup.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:                8080,
			AuthTokens:          []string{"valid-token"},
			RequestTimeout:      5 * time.Second,
			MaxRequestBodyBytes: 1 << 20,
		},
		Models: []config.ModelConfig{
			{
				Name: "test-model",
				Upstreams: []config.UpstreamConfig{
					{Provider: "primary", Protocol: "openai", BaseURL: primary.URL, APIKey: "key-0", Weight: 1},
					{Provider: "backup", Protocol: "openai", BaseURL: backup.URL, APIKey: "key-1", Weight: 1},
				},
			},
		},
		CircuitBreaker: config.CircuitBreakerConfig{
			FailureThreshold:    5,
			RecoveryTimeout:     1 * time.Minute,
			HalfOpenMaxRequests: 1,
		},
	}
	h := setupHandler(cfg)

	// Very first request should immediately failover to backup
	body, _ := json.Marshal(map[string]any{
		"model":    "test-model",
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
	})
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 from backup (immediate failover), got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	choices, _ := resp["choices"].([]any)
	if len(choices) == 0 {
		t.Fatal("expected non-empty choices from backup")
	}
}

func TestProxy_Failover_AllUpstreamsFail_ReturnsLastError(t *testing.T) {
	// Both upstreams return 500. Client should get the last error.
	upstream1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, `{"error":"primary down"}`)
	}))
	defer upstream1.Close()

	upstream2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintln(w, `{"error":"backup also down"}`)
	}))
	defer upstream2.Close()

	cfg := newTestConfig(upstream1.URL, upstream2.URL)
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{
		"model":    "test-model",
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
	})
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	// Should get error since both failed
	if rr.Code == http.StatusOK {
		t.Fatal("expected error status when all upstreams fail, got 200")
	}
}
