package anthropic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zhulang/llm-gateway/internal/apikey"
	"github.com/zhulang/llm-gateway/internal/balancer"
	"github.com/zhulang/llm-gateway/internal/billing"
	"github.com/zhulang/llm-gateway/internal/config"
	"github.com/zhulang/llm-gateway/internal/proxy"
	"github.com/zhulang/llm-gateway/internal/router"
	"github.com/zhulang/llm-gateway/internal/store"
)

// nopStore is a minimal Store implementation for handler tests.
type nopStore struct {
	// Embed the interface so any methods we don't explicitly stub still satisfy
	// store.Store at compile time. Calls to unstubbed methods will panic at
	// runtime, which is fine because the handler tests only exercise a small
	// subset.
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
func (nopStore) DeleteUser(_ string) error { return nil }
func (nopStore) UpdateUserStatus(_, _ string) error                    { return nil }
func (nopStore) UpdatePassword(_, _ string) error                      { return nil }
func (nopStore) GetAdminDashboardStats() (*store.AdminDashboardStats, error) {
	return nil, fmt.Errorf("nop")
}
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
func (nopStore) DeleteAPIKey(_, _ string) error                        { return nil }
func (nopStore) RevokeAllAPIKeys(_ string) (int, error)                { return 0, nil }
func (nopStore) TouchAPIKeyLastUsed(_ string) error                    { return nil }
func (nopStore) BatchTouchAPIKeysLastUsed(_ []string) error            { return nil }
func (nopStore) GetBalance(_ string) (*store.Balance, error) {
	return &store.Balance{Balance: 100.0}, nil
}
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
func (nopStore) GetPricing(_ string) (*store.ModelPricing, error) {
	return &store.ModelPricing{InputPrice: 1.0, OutputPrice: 2.0, CachedInputPrice: 0.1, IsActive: true}, nil
}
func (nopStore) GetUserPrimaryPricingTenant(_ string) (string, error) { return "", nil }
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
func (nopStore) GetSession(_, _ string) (*store.ChatSession, error) { return nil, fmt.Errorf("nop") }
func (nopStore) UpdateSessionTitle(_, _, _ string) error            { return nil }
func (nopStore) DeleteSession(_, _ string) error                    { return nil }
func (nopStore) ListMessages(_ string) ([]store.ChatMessage, error) { return nil, nil }
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
func (nopStore) GetUserPricing(_, _ string) (*store.UserPricing, error) { return nil, nil }
func (nopStore) RecordRequestFailure(_, _ string, _ int) error             { return nil }

func (nopStore) AcceptTenantInvitation(_, _ string) error { return nil }

func newTestConfig(upstreamURLs ...string) *config.Config {
	upstreams := make([]config.UpstreamConfig, len(upstreamURLs))
	for i, u := range upstreamURLs {
		upstreams[i] = config.UpstreamConfig{
			Provider: fmt.Sprintf("provider-%d", i),
			BaseURL:  u,
			APIKey:   fmt.Sprintf("upstream-key-%d", i),
			Weight:   1,
		}
	}
	return &config.Config{
		Server: config.ServerConfig{
			Port:                8080,
			AuthTokens:          []string{"valid-token"},
			RequestTimeout:      5 * time.Second,
			MaxRequestBodyBytes: 1 << 20,
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

// newTestAnthropicConfig creates a config with provider "anthropic" so the
// handler uses passthrough mode (forward raw Anthropic body directly).
func newTestAnthropicConfig(upstreamURL string) *config.Config {
	return &config.Config{
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
					{
						Provider:  "anthropic",
						Protocol:  "anthropic",
						BaseURL:   upstreamURL,
						APIKey:    "test-anthropic-key",
						Weight:    1,
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
}

func setupHandler(cfg *config.Config) *Handler {
	holder := config.NewHolder(cfg)
	rt := router.NewFromConfig(cfg)
	lb := balancer.NewRoundRobin()
	kc := apikey.NewCache(100, 5*time.Minute)
	s := nopStore{}

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
	core := proxy.NewCore(holder, rt, lb, s, kc, bs, proxy.NewSharedHTTPClient(), tb)
	return NewHandler(core)
}

func TestHandler_Auth_XAPIKey(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":      "msg_123",
			"type":    "message",
			"role":    "assistant",
			"model":   "test-model",
			"content": []map[string]any{{"type": "text", "text": "hello"}},
			"stop_reason": "end_turn",
			"usage":   map[string]any{"input_tokens": 5, "output_tokens": 3},
		})
	}))
	defer upstream.Close()

	cfg := newTestAnthropicConfig(upstream.URL)
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{
		"model":      "test-model",
		"max_tokens": 100,
		"messages":   []map[string]string{{"role": "user", "content": "hi"}},
	})
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewReader(body))
	req.Header.Set("x-api-key", "valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp MessagesResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Type != "message" {
		t.Fatalf("expected type 'message', got %q", resp.Type)
	}
	if resp.Role != "assistant" {
		t.Fatalf("expected role 'assistant', got %q", resp.Role)
	}
}

func TestHandler_Auth_Bearer(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":          "msg_123",
			"type":        "message",
			"role":        "assistant",
			"model":       "test-model",
			"content":     []map[string]any{{"type": "text", "text": "hello"}},
			"stop_reason": "end_turn",
			"usage":       map[string]any{"input_tokens": 5, "output_tokens": 3},
		})
	}))
	defer upstream.Close()

	cfg := newTestAnthropicConfig(upstream.URL)
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{
		"model":      "test-model",
		"max_tokens": 100,
		"messages":   []map[string]string{{"role": "user", "content": "hi"}},
	})
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_Auth_Missing(t *testing.T) {
	cfg := newTestConfig("http://localhost:1")
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{
		"model":      "test-model",
		"max_tokens": 100,
		"messages":   []map[string]string{{"role": "user", "content": "hi"}},
	})
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewReader(body))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}

	// Verify Anthropic error format
	var errResp anthropicError
	json.NewDecoder(rr.Body).Decode(&errResp)
	if errResp.Type != "error" {
		t.Fatalf("expected error type, got %q", errResp.Type)
	}
	if errResp.Error.Type != "authentication_error" {
		t.Fatalf("expected authentication_error, got %q", errResp.Error.Type)
	}
}

func TestHandler_ModelNotFound(t *testing.T) {
	cfg := newTestConfig("http://localhost:1")
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{
		"model":      "nonexistent",
		"max_tokens": 100,
		"messages":   []map[string]string{{"role": "user", "content": "hi"}},
	})
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewReader(body))
	req.Header.Set("x-api-key", "valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}

	var errResp anthropicError
	json.NewDecoder(rr.Body).Decode(&errResp)
	if errResp.Error.Type != "not_found_error" {
		t.Fatalf("expected not_found_error, got %q", errResp.Error.Type)
	}
}

func TestHandler_MissingMaxTokens(t *testing.T) {
	cfg := newTestConfig("http://localhost:1")
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{
		"model":    "test-model",
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
	})
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewReader(body))
	req.Header.Set("x-api-key", "valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_ThinkingWithoutMaxTokens(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request body is forwarded as Anthropic format (passthrough)
		var reqBody map[string]any
		json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody["thinking"] == nil {
			t.Error("expected thinking field in upstream request")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "msg_123",
			"type":  "message",
			"role":  "assistant",
			"model": "pa/claude-sonnet-4-6",
			"content": []map[string]any{
				{"type": "text", "text": "Hello!"},
			},
			"stop_reason": "end_turn",
			"usage":       map[string]any{"input_tokens": 10, "output_tokens": 5},
		})
	}))
	defer upstream.Close()

	cfg := newTestAnthropicConfig(upstream.URL)
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{
		"model":    "test-model",
		"messages": []map[string]any{{"role": "user", "content": "Hello"}},
		"thinking": map[string]any{"type": "enabled", "budget_tokens": 5000},
	})
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewReader(body))
	req.Header.Set("x-api-key", "valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_StreamModelNameRewrite(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		events := []string{
			`event: message_start`,
			`data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"pa/claude-sonnet-4-6","content":[],"usage":{"input_tokens":10,"cache_creation_input_tokens":0,"cache_read_input_tokens":5}}}`,
			``,
			`event: content_block_start`,
			`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
			``,
			`event: content_block_delta`,
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hi!"}}`,
			``,
			`event: content_block_stop`,
			`data: {"type":"content_block_stop","index":0}`,
			``,
			`event: message_delta`,
			`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":3}}`,
			``,
			`event: message_stop`,
			`data: {"type":"message_stop"}`,
			``,
		}
		for _, e := range events {
			fmt.Fprintf(w, "%s\n", e)
		}
		flusher.Flush()
	}))
	defer upstream.Close()

	cfg := newTestAnthropicConfig(upstream.URL)
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{
		"model":      "test-model",
		"max_tokens": 100,
		"stream":     true,
		"messages":   []map[string]any{{"role": "user", "content": "Hello"}},
	})
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewReader(body))
	req.Header.Set("x-api-key", "valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	respBody := rr.Body.String()

	// Verify model name was rewritten from pa/claude-sonnet-4-6 to test-model
	if strings.Contains(respBody, "pa/claude-sonnet-4-6") {
		t.Error("expected model name to be rewritten, but found 'pa/claude-sonnet-4-6' in response")
	}
	if !strings.Contains(respBody, `"model":"test-model"`) {
		t.Error("expected model name 'test-model' in response")
	}
	// Verify events are still present
	if !strings.Contains(respBody, "message_start") {
		t.Error("missing message_start event")
	}
	if !strings.Contains(respBody, "text_delta") {
		t.Error("missing text_delta event")
	}
	if !strings.Contains(respBody, "message_stop") {
		t.Error("missing message_stop event")
	}
}

func TestHandler_NonStream_Success(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it received Anthropic format (passthrough)
		var reqBody map[string]any
		json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody["max_tokens"] == nil {
			t.Error("expected max_tokens in upstream request")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":          "msg_abc123",
			"type":        "message",
			"role":        "assistant",
			"model":       "test-model",
			"content":     []map[string]any{{"type": "text", "text": "Hello from the LLM!"}},
			"stop_reason": "end_turn",
			"usage":       map[string]any{"input_tokens": 10, "output_tokens": 5},
		})
	}))
	defer upstream.Close()

	cfg := newTestAnthropicConfig(upstream.URL)
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{
		"model":      "test-model",
		"max_tokens": 1024,
		"messages":   []map[string]any{{"role": "user", "content": "Hello"}},
	})
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewReader(body))
	req.Header.Set("x-api-key", "valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp MessagesResponse
	json.NewDecoder(rr.Body).Decode(&resp)

	if resp.Type != "message" {
		t.Fatalf("expected type 'message', got %q", resp.Type)
	}
	if !strings.HasPrefix(resp.ID, "msg_") {
		t.Fatalf("expected ID to start with 'msg_', got %q", resp.ID)
	}
	if resp.Model != "test-model" {
		t.Fatalf("expected model 'test-model', got %q", resp.Model)
	}
	if resp.StopReason != "end_turn" {
		t.Fatalf("expected stop_reason 'end_turn', got %q", resp.StopReason)
	}
	if len(resp.Content) != 1 || resp.Content[0].Type != "text" || resp.Content[0].Text != "Hello from the LLM!" {
		t.Fatalf("unexpected content: %v", resp.Content)
	}
	if resp.Usage.InputTokens != 10 || resp.Usage.OutputTokens != 5 {
		t.Fatalf("unexpected usage: %+v", resp.Usage)
	}
}

func TestHandler_Stream_Success(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		// Return Anthropic SSE events (passthrough — no conversion)
		chunks := []string{
			"event: message_start\ndata: " + `{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"test-model","usage":{"input_tokens":5,"output_tokens":0}}}`,
			"event: content_block_start\ndata: " + `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
			"event: content_block_delta\ndata: " + `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hi"}}`,
			"event: content_block_delta\ndata: " + `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"!"}}`,
			"event: content_block_stop\ndata: " + `{"type":"content_block_stop","index":0}`,
			"event: message_delta\ndata: " + `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":2}}`,
			"event: message_stop\ndata: " + `{"type":"message_stop"}`,
		}
		for _, c := range chunks {
			fmt.Fprintf(w, "%s\n\n", c)
			flusher.Flush()
		}
	}))
	defer upstream.Close()

	cfg := newTestAnthropicConfig(upstream.URL)
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{
		"model":      "test-model",
		"max_tokens": 100,
		"stream":     true,
		"messages":   []map[string]any{{"role": "user", "content": "Hello"}},
	})
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewReader(body))
	req.Header.Set("x-api-key", "valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("expected text/event-stream, got %q", ct)
	}

	respBody := rr.Body.String()

	// Verify Anthropic SSE format
	if !strings.Contains(respBody, "event: message_start") {
		t.Error("missing message_start event")
	}
	if !strings.Contains(respBody, "event: content_block_start") {
		t.Error("missing content_block_start event")
	}
	if !strings.Contains(respBody, "text_delta") {
		t.Error("missing text_delta")
	}
	if !strings.Contains(respBody, "event: message_stop") {
		t.Error("missing message_stop event")
	}
}

func TestHandler_ToolUseResponse(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return Anthropic tool_use format (passthrough — no conversion)
		json.NewEncoder(w).Encode(map[string]any{
			"id":   "msg_tool",
			"type": "message",
			"role": "assistant",
			"model": "test-model",
			"content": []map[string]any{
				{
					"type":  "tool_use",
					"id":    "call_abc",
					"name":  "get_weather",
					"input": map[string]any{"location": "NYC"},
				},
			},
			"stop_reason": "tool_use",
			"usage":       map[string]any{"input_tokens": 20, "output_tokens": 15},
		})
	}))
	defer upstream.Close()

	cfg := newTestAnthropicConfig(upstream.URL)
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{
		"model":      "test-model",
		"max_tokens": 100,
		"messages":   []map[string]any{{"role": "user", "content": "What is the weather?"}},
		"tools": []map[string]any{
			{
				"name":         "get_weather",
				"description":  "Get the current weather",
				"input_schema": map[string]any{"type": "object", "properties": map[string]any{"location": map[string]string{"type": "string"}}},
			},
		},
	})
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewReader(body))
	req.Header.Set("x-api-key", "valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp MessagesResponse
	json.NewDecoder(rr.Body).Decode(&resp)

	if resp.StopReason != "tool_use" {
		t.Fatalf("expected stop_reason 'tool_use', got %q", resp.StopReason)
	}
	if len(resp.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(resp.Content))
	}
	if resp.Content[0].Type != "tool_use" {
		t.Fatalf("expected tool_use type, got %q", resp.Content[0].Type)
	}
	if resp.Content[0].ID != "call_abc" {
		t.Fatalf("expected id 'call_abc', got %q", resp.Content[0].ID)
	}
	if resp.Content[0].Name != "get_weather" {
		t.Fatalf("expected name 'get_weather', got %q", resp.Content[0].Name)
	}
}

func TestHandler_NoAnthropicNoOpenAIChatUpstream_Returns503(t *testing.T) {
	// When a model only has Gemini-protocol upstreams, /v1/messages has no
	// anthropic upstream to passthrough and no openai/openai-compatible upstream
	// to convert to. It must return 503 no_compatible_upstream.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("upstream should not be called when no compatible protocol is configured")
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
				Name: "test-model",
				Upstreams: []config.UpstreamConfig{
					{
						Provider: "google",
						Protocol: "gemini",
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
		"model":      "test-model",
		"max_tokens": 100,
		"messages":   []map[string]string{{"role": "user", "content": "hi"}},
	})
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewReader(body))
	req.Header.Set("x-api-key", "valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "No compatible upstream") {
		t.Fatalf("expected 'No compatible upstream' in body, got: %s", rr.Body.String())
	}
}

func TestHandler_FallbackToOpenAIChat_NonStream(t *testing.T) {
	// When a model only has OpenAI Chat-protocol upstreams, /v1/messages
	// falls back to converting the request to OpenAI Chat Completions and
	// the response back to Anthropic MessagesResponse format.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request was converted to OpenAI Chat format
		var reqBody map[string]any
		json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody["messages"] == nil {
			t.Error("expected messages field in converted request")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":      "chatcmpl-converted",
			"object":  "chat.completion",
			"model":   "test-model",
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": "Hello from OpenAI"}, "finish_reason": "stop"},
			},
			"usage": map[string]any{"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15},
		})
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
				Name: "test-model",
				Upstreams: []config.UpstreamConfig{
					{
						Provider: "openai-provider",
						Protocol: "openai",
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
		"model":      "test-model",
		"max_tokens": 100,
		"messages":   []map[string]string{{"role": "user", "content": "hi"}},
	})
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewReader(body))
	req.Header.Set("x-api-key", "valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	// Response should be in Anthropic MessagesResponse format
	var resp struct {
		Type    string `json:"type"`
		Role    string `json:"role"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v; body=%s", err, rr.Body.String())
	}
	if resp.Type != "message" {
		t.Errorf("expected type=message, got %q", resp.Type)
	}
	if resp.Role != "assistant" {
		t.Errorf("expected role=assistant, got %q", resp.Role)
	}
	if len(resp.Content) == 0 || resp.Content[0].Type != "text" {
		t.Errorf("expected content[0].type=text, got %+v", resp.Content)
	}
	if resp.Usage.OutputTokens != 5 {
		t.Errorf("expected output_tokens=5, got %d", resp.Usage.OutputTokens)
	}
}
