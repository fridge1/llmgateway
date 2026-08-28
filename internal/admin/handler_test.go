package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zhulang/llm-gateway/internal/config"
	"github.com/zhulang/llm-gateway/internal/router"
)

func testConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Port:       8080,
			AuthTokens: []string{"t"},
			AdminToken: "your-admin-token",
		},
		Models: []config.ModelConfig{
			{
				Name: "test-model",
				Upstreams: []config.UpstreamConfig{
					{
						Provider: "openai",
						BaseURL:  "https://api.openai.com",
						APIKey:   "sk-secret-key",
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
}

func TestHealth(t *testing.T) {
	cfg := testConfig()
	holder := config.NewHolder(cfg)
	rt := router.NewFromConfig(cfg)
	h := NewHandler(holder, rt, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	h.HandleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %q", body["status"])
	}
}

func TestGetConfig_Authenticated(t *testing.T) {
	cfg := testConfig()
	holder := config.NewHolder(cfg)
	rt := router.NewFromConfig(cfg)
	h := NewHandler(holder, rt, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/config", nil)
	req.Header.Set("X-Admin-Token", "your-admin-token")
	w := httptest.NewRecorder()
	h.HandleGetConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if strings.Contains(body, "sk-secret-key") {
		t.Error("response should not contain the raw API key")
	}
}

func TestGetConfig_Unauthorized(t *testing.T) {
	cfg := testConfig()
	holder := config.NewHolder(cfg)
	rt := router.NewFromConfig(cfg)
	h := NewHandler(holder, rt, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/config", nil)
	// no X-Admin-Token header
	w := httptest.NewRecorder()
	h.HandleGetConfig(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestGetStatus(t *testing.T) {
	cfg := testConfig()
	holder := config.NewHolder(cfg)
	rt := router.NewFromConfig(cfg)
	h := NewHandler(holder, rt, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/status", nil)
	req.Header.Set("X-Admin-Token", "your-admin-token")
	w := httptest.NewRecorder()
	h.HandleGetStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := body["models"]; !ok {
		t.Error("expected response to have 'models' key")
	}
}

func TestListModels(t *testing.T) {
	cfg := testConfig()
	holder := config.NewHolder(cfg)
	rt := router.NewFromConfig(cfg)
	h := NewHandler(holder, rt, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer t")
	w := httptest.NewRecorder()
	h.HandleListModels(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data, ok := body["data"]
	if !ok {
		t.Fatal("expected response to have 'data' key")
	}
	models, ok := data.([]any)
	if !ok {
		t.Fatal("expected 'data' to be an array")
	}
	if len(models) != 1 {
		t.Errorf("expected 1 model, got %d", len(models))
	}
}

func TestListModels_XAPIKey(t *testing.T) {
	cfg := testConfig()
	holder := config.NewHolder(cfg)
	rt := router.NewFromConfig(cfg)
	h := NewHandler(holder, rt, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("x-api-key", "t")
	w := httptest.NewRecorder()
	h.HandleListModels(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data, ok := body["data"]
	if !ok {
		t.Fatal("expected response to have 'data' key")
	}
	models, ok := data.([]any)
	if !ok {
		t.Fatal("expected 'data' to be an array")
	}
	if len(models) != 1 {
		t.Errorf("expected 1 model, got %d", len(models))
	}
}
