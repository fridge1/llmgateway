package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/zhulang/llm-gateway/internal/config"
)

const validYAML = `
server:
  port: 9090
  auth_tokens:
    - "sk-test-token"
  admin_token: "your-admin-token"
  request_timeout: 60s
  max_request_body_bytes: 1048576
  shutdown_timeout: 30s

models:
  - name: "deepseek-chat"
    upstreams:
      - provider: "deepseek"
        base_url: "https://api.deepseek.com/v1"
        api_key: "sk-xxx"
        weight: 1
      - provider: "siliconflow"
        base_url: "https://api.siliconflow.cn/v1"
        api_key: "sk-yyy"
        weight: 1

  - name: "qwen-plus"
    upstreams:
      - provider: "dashscope"
        base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1"
        api_key: "sk-aaa"
        weight: 1
      - provider: "siliconflow"
        base_url: "https://api.siliconflow.cn/v1"
        api_key: "sk-bbb"
        model_override: "Qwen/Qwen2.5-Plus"
        weight: 1

circuit_breaker:
  failure_threshold: 5
  recovery_timeout: 30s
  half_open_max_requests: 2
`

func TestLoadConfig_ValidFile(t *testing.T) {
	f, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(validYAML); err != nil {
		t.Fatal(err)
	}
	f.Close()

	cfg, err := config.LoadFromFile(f.Name())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Server fields
	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want 9090", cfg.Server.Port)
	}
	if len(cfg.Server.AuthTokens) != 1 || cfg.Server.AuthTokens[0] != "sk-test-token" {
		t.Errorf("Server.AuthTokens = %v, want [sk-test-token]", cfg.Server.AuthTokens)
	}
	if cfg.Server.AdminToken != "your-admin-token" {
		t.Errorf("Server.AdminToken = %q, want your-admin-token", cfg.Server.AdminToken)
	}
	if cfg.Server.RequestTimeout != 60*time.Second {
		t.Errorf("Server.RequestTimeout = %v, want 60s", cfg.Server.RequestTimeout)
	}
	if cfg.Server.MaxRequestBodyBytes != 1048576 {
		t.Errorf("Server.MaxRequestBodyBytes = %d, want 1048576", cfg.Server.MaxRequestBodyBytes)
	}
	if cfg.Server.ShutdownTimeout != 30*time.Second {
		t.Errorf("Server.ShutdownTimeout = %v, want 30s", cfg.Server.ShutdownTimeout)
	}

	// Models
	if len(cfg.Models) != 2 {
		t.Fatalf("len(Models) = %d, want 2", len(cfg.Models))
	}
	if cfg.Models[0].Name != "deepseek-chat" {
		t.Errorf("Models[0].Name = %q, want deepseek-chat", cfg.Models[0].Name)
	}
	if len(cfg.Models[0].Upstreams) != 2 {
		t.Fatalf("len(Models[0].Upstreams) = %d, want 2", len(cfg.Models[0].Upstreams))
	}
	if cfg.Models[0].Upstreams[0].Provider != "deepseek" {
		t.Errorf("Upstreams[0].Provider = %q, want deepseek", cfg.Models[0].Upstreams[0].Provider)
	}
	if cfg.Models[0].Upstreams[0].Weight != 1 {
		t.Errorf("Upstreams[0].Weight = %d, want 1", cfg.Models[0].Upstreams[0].Weight)
	}

	// CircuitBreaker
	if cfg.CircuitBreaker.FailureThreshold != 5 {
		t.Errorf("CircuitBreaker.FailureThreshold = %d, want 5", cfg.CircuitBreaker.FailureThreshold)
	}
	if cfg.CircuitBreaker.RecoveryTimeout != 30*time.Second {
		t.Errorf("CircuitBreaker.RecoveryTimeout = %v, want 30s", cfg.CircuitBreaker.RecoveryTimeout)
	}
	if cfg.CircuitBreaker.HalfOpenMaxRequests != 2 {
		t.Errorf("CircuitBreaker.HalfOpenMaxRequests = %d, want 2", cfg.CircuitBreaker.HalfOpenMaxRequests)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := config.LoadFromFile("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	// Truly invalid YAML that fails even after env expansion
	_, err := config.LoadFromBytes([]byte("\t\t\t"))
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestValidate_MissingAuthTokens(t *testing.T) {
	// Empty auth_tokens is allowed (just warns), no error expected.
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:       9090,
			AuthTokens: nil, // missing
		},
		Models: []config.ModelConfig{
			{Name: "m1", Upstreams: []config.UpstreamConfig{{Provider: "p1", BaseURL: "http://x", APIKey: "k", Weight: 1}}},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected no error (only warning) for missing auth_tokens, got: %v", err)
	}
}

func TestValidate_MissingModels(t *testing.T) {
	// Empty models is allowed (loaded from DB), no error expected.
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:       9090,
			AuthTokens: []string{"token"},
		},
		Models: nil, // missing
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected no error (only warning) for missing models, got: %v", err)
	}
}

func TestValidate_UnequalWeightsWarning(t *testing.T) {
	// Unequal weights should NOT cause an error - just a warning
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:       9090,
			AuthTokens: []string{"token"},
		},
		Models: []config.ModelConfig{
			{
				Name: "m1",
				Upstreams: []config.UpstreamConfig{
					{Provider: "p1", BaseURL: "http://x", APIKey: "k", Weight: 3},
					{Provider: "p2", BaseURL: "http://y", APIKey: "k2", Weight: 1},
				},
			},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected no error for unequal weights, got: %v", err)
	}
}

func TestLoadConfig_ModelOverride(t *testing.T) {
	cfg, err := config.LoadFromBytes([]byte(validYAML))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// qwen-plus second upstream has model_override
	qwen := cfg.Models[1]
	if qwen.Name != "qwen-plus" {
		t.Fatalf("expected qwen-plus model, got %q", qwen.Name)
	}
	if len(qwen.Upstreams) < 2 {
		t.Fatalf("expected at least 2 upstreams, got %d", len(qwen.Upstreams))
	}
	override := qwen.Upstreams[1].ModelOverride
	if override != "Qwen/Qwen2.5-Plus" {
		t.Errorf("ModelOverride = %q, want Qwen/Qwen2.5-Plus", override)
	}

	// first upstream should have empty model_override
	if qwen.Upstreams[0].ModelOverride != "" {
		t.Errorf("expected empty ModelOverride for first upstream, got %q", qwen.Upstreams[0].ModelOverride)
	}
}

func TestHolder(t *testing.T) {
	cfg1, err := config.LoadFromBytes([]byte(validYAML))
	if err != nil {
		t.Fatal(err)
	}

	h := config.NewHolder(cfg1)
	if h.Get() != cfg1 {
		t.Error("Get() should return initial config")
	}

	cfg2 := &config.Config{}
	h.Swap(cfg2)
	if h.Get() != cfg2 {
		t.Error("Get() should return swapped config")
	}
}
