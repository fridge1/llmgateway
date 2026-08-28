package admin

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/zhulang/llm-gateway/internal/circuit"
	"github.com/zhulang/llm-gateway/internal/config"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/router"
	"github.com/zhulang/llm-gateway/internal/store"
)

// Handler provides admin API endpoints.
type Handler struct {
	cfgHolder *config.Holder
	router    atomic.Pointer[router.Router]
	db        *sql.DB
	store     store.Store
	// apiKeyAuth authenticates a request via the proxy auth chain
	// (user key / tenant key / sub-user key); injected from main to
	// avoid an import cycle with the proxy package.
	apiKeyAuth func(*http.Request) bool
}

// NewHandler creates a new Handler with the given config holder and initial router.
func NewHandler(cfgHolder *config.Holder, rt *router.Router, db *sql.DB, s store.Store) *Handler {
	h := &Handler{cfgHolder: cfgHolder, db: db, store: s}
	h.router.Store(rt)
	return h
}

// SetRouter atomically replaces the active router.
func (h *Handler) SetRouter(rt *router.Router) {
	h.router.Store(rt)
}

// SetAPIKeyAuth injects the API-key authentication function used by
// endpoints that accept gateway API keys (e.g. /v1/models).
func (h *Handler) SetAPIKeyAuth(fn func(*http.Request) bool) {
	h.apiKeyAuth = fn
}

// HandleHealth checks database connectivity, upstream availability, and billing queue health.
// Returns 200 if healthy, 503 if degraded.
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	result := map[string]any{"status": "ok"}
	statusCode := http.StatusOK

	if h.db != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := h.db.PingContext(ctx); err != nil {
			result["status"] = "degraded"
			result["db"] = "error"
			statusCode = http.StatusServiceUnavailable
		} else {
			result["db"] = "ok"
		}
	}

	rt := h.router.Load()
	if rt != nil {
		total, healthy := rt.UpstreamHealth()
		result["upstreams"] = map[string]int{"total": total, "healthy": healthy}
		if healthy == 0 && total > 0 {
			result["status"] = "degraded"
			statusCode = http.StatusServiceUnavailable
		}
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(result)
}

// HandleGetConfig returns the current configuration with API keys masked.
// Requires the X-Admin-Token header to match cfg.Server.AdminToken.
func (h *Handler) HandleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg := h.cfgHolder.Get()
	if r.Header.Get("X-Admin-Token") != cfg.Server.AdminToken {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "auth_error", "forbidden")
		return
	}

	masked := maskConfig(cfg)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(masked)
}

// HandleGetStatus returns per-upstream circuit breaker state and failure counts.
// Requires the X-Admin-Token header to match cfg.Server.AdminToken.
func (h *Handler) HandleGetStatus(w http.ResponseWriter, r *http.Request) {
	cfg := h.cfgHolder.Get()
	if r.Header.Get("X-Admin-Token") != cfg.Server.AdminToken {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "auth_error", "forbidden")
		return
	}

	rt := h.router.Load()
	entries := rt.Entries()

	type upstreamStatus struct {
		Provider         string `json:"provider"`
		UpstreamProvider string `json:"upstream_provider"`
		UpstreamName     string `json:"upstream_name"`
		BaseURL          string `json:"base_url"`
		State            string `json:"state"`
		FailureCount     int    `json:"failure_count"`
	}
	type modelStatus struct {
		Upstreams []upstreamStatus `json:"upstreams"`
	}

	models := make(map[string]modelStatus, len(entries))
	for modelName, upstreams := range entries {
		statuses := make([]upstreamStatus, len(upstreams))
		for i, u := range upstreams {
			failCount, state := u.Breaker.Counts()
			statuses[i] = upstreamStatus{
				Provider:         u.Config.Provider,
				UpstreamProvider: u.Config.UpstreamProvider,
				UpstreamName:     u.Config.UpstreamName,
				BaseURL:          u.Config.BaseURL,
				State:            state.String(),
				FailureCount:     failCount,
			}
		}
		models[modelName] = modelStatus{Upstreams: statuses}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{"models": models})
}

// HandleListModels returns the list of available chat models in OpenAI format.
// Accepts either a valid Bearer token from cfg.Server.AuthTokens or a user API key.
func (h *Handler) HandleListModels(w http.ResponseWriter, r *http.Request) {
	cfg := h.cfgHolder.Get()
	// Support both x-api-key (Anthropic convention) and Authorization: Bearer.
	token := r.Header.Get("x-api-key")
	if token == "" {
		token = httputil.ExtractBearerToken(r)
	}

	// Try admin auth tokens first, then fall back to gateway API keys
	// (user / tenant / sub-user keys via the injected proxy auth chain).
	authenticated := httputil.IsValidToken(token, cfg.Server.AuthTokens)
	if !authenticated && token != "" {
		if h.apiKeyAuth != nil {
			authenticated = h.apiKeyAuth(r)
		} else if h.store != nil {
			sum := sha256.Sum256([]byte(token))
			keyHash := hex.EncodeToString(sum[:])
			if ak, err := h.store.GetAPIKeyByHash(keyHash); err == nil && ak != nil {
				authenticated = true
			}
		}
	}

	// Also check JWT cookie for sub-user or regular user authentication
	if !authenticated {
		// Try to get user_id from context (set by JWT middleware if cookie exists)
		if userID, ok := r.Context().Value(CtxUserIDKey).(string); ok && userID != "" {
			authenticated = true
		}
	}

	if !authenticated {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "unauthorized")
		return
	}

	// Load models from store to get category info for filtering.
	var names []string
	if h.store != nil {
		if models, err := h.store.ListModels(); err == nil {
			for _, m := range models {
				name := m.Name
				if m.DisplayName != "" {
					name = m.DisplayName
				}
				names = append(names, name)
			}
		}
	}
	// Fallback to router if store unavailable.
	if len(names) == 0 {
		rt := h.router.Load()
		names = rt.ListModels()
	}

	type modelObject struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	}

	now := time.Now().Unix()
	data := make([]modelObject, len(names))
	for i, name := range names {
		data[i] = modelObject{
			ID:      name,
			Object:  "model",
			Created: now,
			OwnedBy: "llm-gateway",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"object": "list",
		"data":   data,
	})
}

// HandleListImageShareModels returns the models accessible to an image_share
// session. Image-share sessions can use image-related models, filtered by
// category or name pattern. Restricted to role=image_share so other roles
// can't reach the unfiltered model list through this endpoint.
func (h *Handler) HandleListImageShareModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	role, _ := r.Context().Value(CtxRoleKey).(string)
	if role != "image_share" {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "auth_error", "wrong_role")
		return
	}
	if h.store == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]store.Model{})
		return
	}
	models, err := h.store.ListModels()
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, err.Error(), "server_error", "db_error")
		return
	}
	out := make([]store.Model, 0, len(models))
	for _, m := range models {
		// 图像分享模式允许的模型白名单
		if m.Name == "gpt-image-2" {
			out = append(out, m)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// HandleGetConfigNoAuth returns config without checking X-Admin-Token (use behind JWT middleware).
func (h *Handler) HandleGetConfigNoAuth(w http.ResponseWriter, r *http.Request) {
	cfg := h.cfgHolder.Get()
	masked := maskConfig(cfg)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(masked)
}

// HandleGetStatusNoAuth returns status without checking X-Admin-Token (use behind JWT middleware).
func (h *Handler) HandleGetStatusNoAuth(w http.ResponseWriter, r *http.Request) {
	rt := h.router.Load()
	entries := rt.Entries()

	type upstreamStatus struct {
		Provider     string `json:"provider"`
		BaseURL      string `json:"base_url"`
		State        string `json:"state"`
		FailureCount int    `json:"failure_count"`
	}
	type modelStatus struct {
		Upstreams []upstreamStatus `json:"upstreams"`
	}

	models := make(map[string]modelStatus, len(entries))
	for modelName, upstreams := range entries {
		statuses := make([]upstreamStatus, len(upstreams))
		for i, u := range upstreams {
			failCount, state := u.Breaker.Counts()
			statuses[i] = upstreamStatus{
				Provider:     u.Config.Provider,
				BaseURL:      u.Config.BaseURL,
				State:        state.String(),
				FailureCount: failCount,
			}
		}
		models[modelName] = modelStatus{Upstreams: statuses}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{"models": models})
}

// HandleTestUpstreamByName tests an upstream by model name + base_url (server-side lookup of API key).
func (h *Handler) HandleTestUpstreamByName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Model   string `json:"model"`
		BaseURL string `json:"base_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid JSON", "invalid_request_error", "bad_json")
		return
	}
	if req.Model == "" || req.BaseURL == "" {
		httputil.WriteError(w, http.StatusBadRequest, "model and base_url required", "invalid_request_error", "missing_fields")
		return
	}

	rt := h.router.Load()
	entries := rt.Entries()

	upstreams, ok := entries[req.Model]
	if !ok {
		writeTestResponse(w, false, "模型未找到", "")
		return
	}

	var apiKey string
	var breaker *circuit.Breaker
	for _, u := range upstreams {
		if u.Config.BaseURL == req.BaseURL {
			apiKey = u.Config.APIKey
			breaker = u.Breaker
			break
		}
	}
	if apiKey == "" {
		writeTestResponse(w, false, "未找到匹配的上游", "")
		return
	}

	url := strings.TrimRight(req.BaseURL, "/") + "/models"
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		writeTestResponse(w, false, "创建请求失败: "+err.Error(), "")
		return
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	start := time.Now()
	resp, err := http.DefaultClient.Do(httpReq)
	latency := time.Since(start)

	if err != nil {
		writeTestResponse(w, false, "连接失败: "+err.Error(), "")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		if breaker != nil {
			breaker.Reset()
		}
		writeTestResponse(w, true, "连接成功", latency.Round(time.Millisecond).String())
	} else if resp.StatusCode == http.StatusUnauthorized {
		writeTestResponse(w, false, "API Key 无效 (401 Unauthorized)", latency.Round(time.Millisecond).String())
	} else {
		writeTestResponse(w, false, "上游返回 HTTP "+http.StatusText(resp.StatusCode), latency.Round(time.Millisecond).String())
	}
}

// maskAPIKey masks an API key, showing the first 4 and last 4 characters
// separated by "****", or just "****" if the key is 8 characters or fewer.
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// maskConfig returns a map representation of cfg with API keys masked.
func maskConfig(cfg *config.Config) map[string]any {
	models := make([]map[string]any, len(cfg.Models))
	for i, m := range cfg.Models {
		upstreams := make([]map[string]any, len(m.Upstreams))
		for j, u := range m.Upstreams {
			upstreams[j] = map[string]any{
				"provider":          u.Provider,
				"protocol":          u.Protocol,
				"protocols":         u.Protocols,
				"base_url":          u.BaseURL,
				"api_key":           maskAPIKey(u.APIKey),
				"model_override":    u.ModelOverride,
				"weight":            u.Weight,
			}
		}
		models[i] = map[string]any{
			"name":      m.Name,
			"upstreams": upstreams,
		}
	}

	return map[string]any{
		"server": map[string]any{
			"port":                    cfg.Server.Port,
			"auth_tokens":             cfg.Server.AuthTokens,
			"admin_token":             maskAPIKey(cfg.Server.AdminToken),
			"request_timeout":         cfg.Server.RequestTimeout.String(),
			"max_request_body_bytes":  cfg.Server.MaxRequestBodyBytes,
			"shutdown_timeout":        cfg.Server.ShutdownTimeout.String(),
		},
		"models": models,
		"circuit_breaker": map[string]any{
			"failure_threshold":     cfg.CircuitBreaker.FailureThreshold,
			"recovery_timeout":      cfg.CircuitBreaker.RecoveryTimeout.String(),
			"half_open_max_requests": cfg.CircuitBreaker.HalfOpenMaxRequests,
		},
	}
}
