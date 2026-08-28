package router

import (
	"testing"
	"time"

	"github.com/zhulang/llm-gateway/internal/config"
)

func makeConfig(modelNames ...string) *config.Config {
	models := make([]config.ModelConfig, len(modelNames))
	for i, name := range modelNames {
		models[i] = config.ModelConfig{
			Name: name,
			Upstreams: []config.UpstreamConfig{
				{
					Provider: "openai",
					BaseURL:  "https://api.openai.com",
					APIKey:   "test-key",
					Weight:   1,
				},
			},
		}
	}
	return &config.Config{
		Server: config.ServerConfig{
			AuthTokens: []string{"test-token"},
		},
		Models: models,
		CircuitBreaker: config.CircuitBreakerConfig{
			FailureThreshold:    3,
			RecoveryTimeout:     10 * time.Second,
			HalfOpenMaxRequests: 1,
		},
	}
}

func TestRouter_GetUpstreams_Found(t *testing.T) {
	cfg := makeConfig("model-a")
	r := NewFromConfig(cfg)

	upstreams, info, ok := r.GetUpstreams("model-a")
	if !ok {
		t.Fatal("expected to find model-a, got !ok")
	}
	if info.CanonicalName != "model-a" {
		t.Errorf("expected canonical name %q, got %q", "model-a", info.CanonicalName)
	}
	if len(upstreams) != 1 {
		t.Errorf("expected 1 upstream, got %d", len(upstreams))
	}
	if upstreams[0].Breaker == nil {
		t.Error("expected Breaker to be non-nil")
	}
	if upstreams[0].Config.Provider != "openai" {
		t.Errorf("expected provider %q, got %q", "openai", upstreams[0].Config.Provider)
	}
}

func TestRouter_GetUpstreams_NotFound(t *testing.T) {
	cfg := makeConfig("model-a")
	r := NewFromConfig(cfg)

	_, _, ok := r.GetUpstreams("nonexistent")
	if ok {
		t.Fatal("expected !ok for nonexistent model, got ok")
	}
}

func TestRouter_GetUpstreams_GLMSuffixStrip(t *testing.T) {
	cfg := makeConfig("glm-5.1")
	r := NewFromConfig(cfg)

	// With [1m] suffix should still match
	upstreams, info, ok := r.GetUpstreams("glm-5.1[1m]")
	if !ok {
		t.Fatal("expected to find glm-5.1 via glm-5.1[1m], got !ok")
	}
	if info.CanonicalName != "glm-5.1" {
		t.Errorf("expected canonical name %q, got %q", "glm-5.1", info.CanonicalName)
	}
	if len(upstreams) != 1 {
		t.Errorf("expected 1 upstream, got %d", len(upstreams))
	}

	// Without suffix should still work
	_, _, ok = r.GetUpstreams("glm-5.1")
	if !ok {
		t.Fatal("expected to find glm-5.1 directly, got !ok")
	}

	// Non-GLM model with suffix should NOT be stripped
	cfg2 := makeConfig("claude-opus-4-6")
	r2 := NewFromConfig(cfg2)
	_, _, ok = r2.GetUpstreams("claude-opus-4-6[1m]")
	if ok {
		t.Fatal("expected !ok for non-GLM model with suffix, got ok")
	}
}

func TestRouter_ListModels(t *testing.T) {
	cfg := makeConfig("model-a", "model-b")
	r := NewFromConfig(cfg)

	models := r.ListModels()
	if len(models) != 2 {
		t.Errorf("expected 2 models, got %d", len(models))
	}
}

func TestNormalizeModelName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"gw/claude-sonnet-4", "claude-sonnet-4"},
		{"claude-sonnet-4", "claude-sonnet-4"},
		{"gw/gw/model", "gw/model"},
		{"", ""},
		{"gw/", ""},
	}
	for _, tt := range tests {
		got := NormalizeModelName(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeModelName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRouter_GetUpstreams_GWPrefix(t *testing.T) {
	cfg := makeConfig("claude-opus-4-7")
	r := NewFromConfig(cfg)

	// With gw/ prefix should resolve to the same model
	_, info, ok := r.GetUpstreams("gw/claude-opus-4-7")
	if !ok {
		t.Fatal("expected to find claude-opus-4-7 via gw/ prefix, got !ok")
	}
	if info.CanonicalName != "claude-opus-4-7" {
		t.Errorf("expected canonical name %q, got %q", "claude-opus-4-7", info.CanonicalName)
	}

	// Without prefix should still work
	_, _, ok = r.GetUpstreams("claude-opus-4-7")
	if !ok {
		t.Fatal("expected to find claude-opus-4-7 directly, got !ok")
	}
}

func TestRouter_GetUpstreams_GWPrefix_GLM(t *testing.T) {
	cfg := makeConfig("glm-5.1")
	r := NewFromConfig(cfg)

	// gw/ prefix + GLM suffix should both be stripped
	_, info, ok := r.GetUpstreams("gw/glm-5.1[1m]")
	if !ok {
		t.Fatal("expected to find glm-5.1 via gw/glm-5.1[1m], got !ok")
	}
	if info.CanonicalName != "glm-5.1" {
		t.Errorf("expected canonical name %q, got %q", "glm-5.1", info.CanonicalName)
	}
}

func addTenantOverride(cfg *config.Config, tenantID, model, baseURL string) {
	cfg.TenantModels = append(cfg.TenantModels, config.TenantModelConfig{
		TenantID:  tenantID,
		ModelName: model,
		Upstreams: []config.UpstreamConfig{
			{Provider: "openai", BaseURL: baseURL, APIKey: "tenant-key", Weight: 1},
		},
	})
}

func TestRouter_GetUpstreamsForTenant_Override(t *testing.T) {
	cfg := makeConfig("model-a")
	addTenantOverride(cfg, "tenant-1", "model-a", "https://tenant.example.com")
	r := NewFromConfig(cfg)

	upstreams, info, ok := r.GetUpstreamsForTenant("tenant-1", "model-a")
	if !ok {
		t.Fatal("expected tenant override hit, got !ok")
	}
	if info.CanonicalName != "model-a" {
		t.Errorf("expected canonical name %q, got %q", "model-a", info.CanonicalName)
	}
	// Tenant hit must return the exclusive pool: only the tenant upstream.
	if len(upstreams) != 1 || upstreams[0].Config.BaseURL != "https://tenant.example.com" {
		t.Fatalf("expected only tenant upstream, got %+v", upstreams)
	}
}

func TestRouter_GetUpstreamsForTenant_FallbackToGlobal(t *testing.T) {
	cfg := makeConfig("model-a", "model-b")
	addTenantOverride(cfg, "tenant-1", "model-a", "https://tenant.example.com")
	r := NewFromConfig(cfg)

	// Empty tenantID → global.
	ups, _, ok := r.GetUpstreamsForTenant("", "model-a")
	if !ok || ups[0].Config.BaseURL != "https://api.openai.com" {
		t.Fatalf("empty tenantID: expected global upstream, got %+v", ups)
	}

	// Unknown tenant → global.
	ups, _, ok = r.GetUpstreamsForTenant("tenant-x", "model-a")
	if !ok || ups[0].Config.BaseURL != "https://api.openai.com" {
		t.Fatalf("unknown tenant: expected global upstream, got %+v", ups)
	}

	// Known tenant but model without override → global.
	ups, _, ok = r.GetUpstreamsForTenant("tenant-1", "model-b")
	if !ok || ups[0].Config.BaseURL != "https://api.openai.com" {
		t.Fatalf("model without override: expected global upstream, got %+v", ups)
	}
}

func TestRouter_GetUpstreamsForTenant_AliasAndPrefix(t *testing.T) {
	cfg := makeConfig("pa/model-a")
	cfg.Models[0].DisplayName = "model-a"
	addTenantOverride(cfg, "tenant-1", "pa/model-a", "https://tenant.example.com")
	r := NewFromConfig(cfg)

	for _, requested := range []string{"pa/model-a", "model-a", "gw/model-a"} {
		ups, _, ok := r.GetUpstreamsForTenant("tenant-1", requested)
		if !ok {
			t.Fatalf("request %q: expected hit, got !ok", requested)
		}
		if ups[0].Config.BaseURL != "https://tenant.example.com" {
			t.Errorf("request %q: expected tenant upstream, got %q", requested, ups[0].Config.BaseURL)
		}
	}
}

func TestRouter_GetUpstreamsForTenant_OrphanOverride(t *testing.T) {
	// Override for a model that no longer exists in the global table.
	cfg := makeConfig("model-a")
	addTenantOverride(cfg, "tenant-1", "model-gone", "https://tenant.example.com")
	r := NewFromConfig(cfg)

	ups, info, ok := r.GetUpstreamsForTenant("tenant-1", "model-gone")
	if !ok {
		t.Fatal("expected orphan override to still route, got !ok")
	}
	if info.CanonicalName != "model-gone" || ups[0].Config.BaseURL != "https://tenant.example.com" {
		t.Fatalf("unexpected orphan result: info=%+v ups=%+v", info, ups)
	}

	// Non-tenant request for the orphan model must still miss.
	if _, _, ok := r.GetUpstreamsForTenant("", "model-gone"); ok {
		t.Fatal("expected !ok for orphan model without tenant")
	}
}

func TestRouter_TenantBreakers_ReusedAndIsolated(t *testing.T) {
	cfg := makeConfig("model-a")
	// Tenant override pointing at the SAME base URL as the global upstream.
	addTenantOverride(cfg, "tenant-1", "model-a", "https://api.openai.com")
	r1 := NewFromConfig(cfg)

	globalUps, _, _ := r1.GetUpstreamsForTenant("", "model-a")
	tenantUps, _, _ := r1.GetUpstreamsForTenant("tenant-1", "model-a")
	if globalUps[0].Breaker == tenantUps[0].Breaker {
		t.Fatal("tenant and global breakers for the same base URL must be distinct instances")
	}

	// Rebuild must reuse both breakers (preserve failure state).
	r2 := NewFromConfigWithBreakers(cfg, r1)
	globalUps2, _, _ := r2.GetUpstreamsForTenant("", "model-a")
	tenantUps2, _, _ := r2.GetUpstreamsForTenant("tenant-1", "model-a")
	if globalUps2[0].Breaker != globalUps[0].Breaker {
		t.Error("global breaker not reused across rebuild")
	}
	if tenantUps2[0].Breaker != tenantUps[0].Breaker {
		t.Error("tenant breaker not reused across rebuild")
	}
}
