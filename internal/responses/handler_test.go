package responses

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
// 嵌入 store.Store 后，未显式 stub 的方法在编译期自动满足接口；
// 若运行时被调用会 panic，但 handler 测试只命中下方显式实现的方法子集。
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
func (nopStore) GetTenantAPIKeyByHash(_ string) (*store.TenantAPIKey, error)         { return nil, fmt.Errorf("nop") }
func (nopStore) GetTenantSubUserKeyByHash(_ string) (*store.TenantSubUserKey, error) { return nil, fmt.Errorf("nop") }
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

func newTestConfig(upstreamURLs ...string) *config.Config {
	upstreams := make([]config.UpstreamConfig, len(upstreamURLs))
	for i, u := range upstreamURLs {
		upstreams[i] = config.UpstreamConfig{
			Provider: fmt.Sprintf("provider-%d", i),
			Protocol: "responses",
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

func TestResponses_Auth_MissingToken(t *testing.T) {
	cfg := newTestConfig("http://localhost:1")
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{"model": "test-model", "input": "hello"})
	req := httptest.NewRequest("POST", "/v1/responses", bytes.NewReader(body))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestResponses_Auth_InvalidToken(t *testing.T) {
	cfg := newTestConfig("http://localhost:1")
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{"model": "test-model", "input": "hello"})
	req := httptest.NewRequest("POST", "/v1/responses", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer wrong-token")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestResponses_ModelNotFound(t *testing.T) {
	cfg := newTestConfig("http://localhost:1")
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{"model": "nonexistent", "input": "hello"})
	req := httptest.NewRequest("POST", "/v1/responses", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestResponses_NonStream_Success(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]any
		json.NewDecoder(r.Body).Decode(&reqBody)

		if reqBody["model"] != "test-model" {
			t.Errorf("expected model 'test-model', got %v", reqBody["model"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":     "resp_123",
			"object": "response",
			"status": "completed",
			"model":  "test-model",
			"output": []map[string]any{
				{
					"type": "message",
					"role": "assistant",
					"content": []map[string]any{
						{"type": "output_text", "text": "Hello there!"},
					},
				},
			},
			"usage": map[string]any{
				"input_tokens":  10,
				"output_tokens": 5,
			},
		})
	}))
	defer upstream.Close()

	cfg := newTestConfig(upstream.URL)
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{
		"model": "test-model",
		"input": "Hello!",
	})
	req := httptest.NewRequest("POST", "/v1/responses", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["object"] != "response" {
		t.Fatalf("expected object 'response', got %v", resp["object"])
	}
	if resp["status"] != "completed" {
		t.Fatalf("expected status 'completed', got %v", resp["status"])
	}
	if resp["model"] != "test-model" {
		t.Fatalf("expected model 'test-model', got %v", resp["model"])
	}
}

func TestResponses_Stream_Success(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]any
		json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody["stream"] != true {
			t.Errorf("expected stream=true")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		flusher := w.(http.Flusher)

		chunks := []string{
			`event: response.created`,
			`data: {"id":"resp_1","object":"response","status":"in_progress","model":"test-model"}`,
			``,
			`event: response.output_text.delta`,
			`data: {"delta":"Hi"}`,
			``,
			`event: response.completed`,
			`data: {"id":"resp_1","object":"response","status":"completed","model":"test-model","usage":{"input_tokens":8,"output_tokens":2}}`,
			``,
		}
		for _, c := range chunks {
			fmt.Fprintf(w, "%s\n", c)
			flusher.Flush()
		}
	}))
	defer upstream.Close()

	cfg := newTestConfig(upstream.URL)
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{
		"model":  "test-model",
		"input":  "Hello!",
		"stream": true,
	})
	req := httptest.NewRequest("POST", "/v1/responses", bytes.NewReader(body))
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
	if !strings.Contains(respBody, "event: response.created") {
		t.Fatal("expected response.created event")
	}
	if !strings.Contains(respBody, "event: response.output_text.delta") {
		t.Fatal("expected output_text.delta event")
	}
	if !strings.Contains(respBody, "event: response.completed") {
		t.Fatal("expected response.completed event")
	}
}

func TestResponses_FunctionCall_Response(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":     "resp_fc",
			"object": "response",
			"status": "completed",
			"model":  "test-model",
			"output": []map[string]any{
				{
					"type":      "function_call",
					"call_id":   "call_xyz",
					"name":      "get_weather",
					"arguments": `{"location":"NYC"}`,
				},
			},
			"usage": map[string]any{"input_tokens": 25, "output_tokens": 10},
		})
	}))
	defer upstream.Close()

	cfg := newTestConfig(upstream.URL)
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{
		"model": "test-model",
		"input": "What's the weather in NYC?",
		"tools": []map[string]any{
			{
				"type": "function",
				"name": "get_weather",
			},
		},
	})
	req := httptest.NewRequest("POST", "/v1/responses", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	output, ok := resp["output"].([]any)
	if !ok || len(output) == 0 {
		t.Fatalf("expected output array, got %v", resp["output"])
	}
	first, _ := output[0].(map[string]any)
	if first["type"] != "function_call" {
		t.Fatalf("expected function_call, got %v", first["type"])
	}
	if first["call_id"] != "call_xyz" {
		t.Fatalf("expected call_id 'call_xyz', got %v", first["call_id"])
	}
	if first["name"] != "get_weather" {
		t.Fatalf("expected name 'get_weather', got %v", first["name"])
	}
}

func TestResponses_RejectsDuplicateFunctionCallID(t *testing.T) {
	// A request whose input history contains two function_call items with the
	// same call_id must be rejected at the gateway rather than forwarded to
	// an upstream that will reject it the same way.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("upstream should not be called when duplicate call_id is detected")
	}))
	defer upstream.Close()

	cfg := newTestConfig(upstream.URL)
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{
		"model": "test-model",
		"input": []map[string]any{
			{"type": "message", "role": "user", "content": "check weather"},
			{"type": "function_call", "call_id": "call_dup", "name": "get_weather", "arguments": "{}"},
			{"type": "function_call_output", "call_id": "call_dup", "output": "sunny"},
			{"type": "function_call", "call_id": "call_dup", "name": "get_weather", "arguments": "{}"},
		},
	})
	req := httptest.NewRequest("POST", "/v1/responses", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "call_dup") {
		t.Fatalf("expected error body to mention call_id 'call_dup', got %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "duplicate_call_id") {
		t.Fatalf("expected error code 'duplicate_call_id', got %s", rr.Body.String())
	}
}

func TestResponses_AllowsUniqueFunctionCallIDs(t *testing.T) {
	// Sanity check: distinct call_ids must not trigger the duplicate guard.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":     "resp_ok",
			"object": "response",
			"status": "completed",
			"model":  "test-model",
			"output": []any{},
			"usage":  map[string]any{"input_tokens": 10, "output_tokens": 0},
		})
	}))
	defer upstream.Close()

	cfg := newTestConfig(upstream.URL)
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{
		"model": "test-model",
		"input": []map[string]any{
			{"type": "function_call", "call_id": "call_a", "name": "f", "arguments": "{}"},
			{"type": "function_call", "call_id": "call_b", "name": "f", "arguments": "{}"},
		},
	})
	req := httptest.NewRequest("POST", "/v1/responses", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestResponses_EchoesClientModelWithGWPrefix(t *testing.T) {
	// Under strict passthrough, the upstream's model field is forwarded as-is
	// to the client. The gateway routes by stripping gw/ prefix internally but
	// does not rewrite the model string in the body. Verify the request
	// succeeds and a response body is returned.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":     "resp_gw",
			"object": "response",
			"status": "completed",
			"model":  "provider-internal",
			"output": []map[string]any{
				{
					"type": "message",
					"role": "assistant",
					"content": []map[string]any{
						{"type": "output_text", "text": "ok"},
					},
				},
			},
			"usage": map[string]any{"input_tokens": 1, "output_tokens": 1},
		})
	}))
	defer upstream.Close()

	cfg := newTestConfig(upstream.URL)
	h := setupHandler(cfg)

	body, _ := json.Marshal(map[string]any{
		"model": "gw/test-model",
		"input": "hi",
	})
	req := httptest.NewRequest("POST", "/v1/responses", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["object"] != "response" {
		t.Fatalf("expected object 'response', got %v", resp["object"])
	}
}

func TestResponses_NoResponsesNoOpenAIChatUpstream_Returns503(t *testing.T) {
	// When a model only has Gemini-protocol upstreams, /v1/responses has no
	// responses upstream to passthrough and no openai/openai-compatible upstream
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
		"model": "test-model",
		"input": "hi",
	})
	req := httptest.NewRequest("POST", "/v1/responses", bytes.NewReader(body))
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

func TestResponses_FallbackToOpenAIChat_NonStream(t *testing.T) {
	// When a model only has OpenAI Chat-protocol upstreams, /v1/responses
	// falls back to converting the request to OpenAI Chat Completions and the
	// response back to Responses API format.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request was converted to Chat Completions format
		var reqBody map[string]any
		json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody["messages"] == nil {
			t.Error("expected messages field in converted request")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":      "chatcmpl-conv",
			"object":  "chat.completion",
			"model":   "test-model",
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": "Hello"}, "finish_reason": "stop"},
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
		"model": "test-model",
		"input": "hi",
	})
	req := httptest.NewRequest("POST", "/v1/responses", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	// Response should be in Responses API format
	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["object"] != "response" {
		t.Errorf("expected object 'response', got %v", resp["object"])
	}
}
