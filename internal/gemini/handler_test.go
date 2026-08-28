package gemini

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zhulang/llm-gateway/internal/apikey"
	"github.com/zhulang/llm-gateway/internal/balancer"
	"github.com/zhulang/llm-gateway/internal/bandwidth"
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

func (nopStore) Close() error                                 { return nil }
func (nopStore) GetBalance(_ string) (*store.Balance, error) { return &store.Balance{Balance: 100.0}, nil }
func (nopStore) FreezeBalance(_ string, _ float64) error     { return nil }
func (nopStore) SettleBilling(_ string, _, _ float64, _, _ string, _ store.TokenUsage, _ string) error {
	return nil
}
func (nopStore) DirectCharge(_ string, _ float64, _, _ string, _ store.TokenUsage, _ string) error {
	return nil
}
func (nopStore) UnfreezeBalance(_ string, _ float64) error { return nil }
func (nopStore) Recharge(_ string, _ float64, _ string) error {
	return nil
}
func (nopStore) DeductForSubscription(_ string, _ float64, _ string) error { return nil }
func (nopStore) GetPricing(_ string) (*store.ModelPricing, error) {
	return &store.ModelPricing{InputPrice: 1.0, OutputPrice: 2.0, CachedInputPrice: 0.1, IsActive: true}, nil
}
func (nopStore) GetUserPricing(_, _ string) (*store.UserPricing, error) { return nil, nil }
func (nopStore) GetUserPrimaryPricingTenant(_ string) (string, error)   { return "", nil }
func (nopStore) RecordRequestFailure(_, _ string, _ int) error         { return nil }
func (nopStore) ListActivePricing() ([]store.ModelPricing, error)      { return nil, nil }

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
	bwl := bandwidth.NewLimiter(8, 5*time.Second)
	return NewHandler(core, bwl)
}

func TestGemini_NoGeminiUpstream_Returns503(t *testing.T) {
	// When a model only has OpenAI-protocol upstreams, the Gemini entry must
	// NOT cross-protocol convert. It must return 503 instead.
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
		"contents": []map[string]any{
			{"role": "user", "parts": []map[string]string{{"text": "hi"}}},
		},
	})
	req := httptest.NewRequest("POST", "/gemini/v1beta/models/test-model:generateContent", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("No compatible upstream")) {
		t.Fatalf("expected 'No compatible upstream' in body, got: %s", rr.Body.String())
	}
}
