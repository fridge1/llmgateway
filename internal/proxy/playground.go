package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/balancer"
	"github.com/zhulang/llm-gateway/internal/billing"
	"github.com/zhulang/llm-gateway/internal/circuit"
	"github.com/zhulang/llm-gateway/internal/config"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/store"
)

// PlaygroundHandler proxies requests from the web playground.
// Uses JWT auth + key_id ownership verification instead of raw API keys.
type PlaygroundHandler struct {
	cfgHolder *config.Holder
	store     store.Store
	core      *Core
	client    *http.Client
}

func NewPlaygroundHandler(cfgHolder *config.Holder, s store.Store, core *Core, client *http.Client) *PlaygroundHandler {
	return &PlaygroundHandler{
		cfgHolder: cfgHolder,
		store:     s,
		core:      core,
		client:    client,
	}
}

func (h *PlaygroundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := h.cfgHolder.Get()
	rt := h.core.Router.Load()

	// 1. Get user ID from JWT context (set by JWT middleware)
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	// 2. Read body
	maxBytes := cfg.Server.MaxRequestBodyBytes
	if maxBytes <= 0 {
		maxBytes = 1 << 20
	}
	bodyBytes, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBytes))
	if err != nil {
		httputil.WriteError(w, http.StatusRequestEntityTooLarge, "Request body too large", "invalid_request_error", "request_too_large")
		return
	}

	// 3. Parse JSON, extract and remove key_id
	var reqBody map[string]any
	if err := json.Unmarshal(bodyBytes, &reqBody); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid JSON", "invalid_request_error", "invalid_json")
		return
	}

	keyID, _ := reqBody["key_id"].(string)
	if keyID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "key_id is required", "invalid_request_error", "missing_key_id")
		return
	}

	// 4. Verify key ownership
	key, err := h.store.GetActiveAPIKeyByIDAndUser(keyID, userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to verify key", "server_error", "db_error")
		return
	}
	if key == nil {
		httputil.WriteError(w, http.StatusForbidden, "key not found or not owned by you", "auth_error", "key_not_owned")
		return
	}

	// 5. Build auth result from user
	user, err := h.store.GetUserByID(userID)
	if err != nil || user.Status != "active" {
		httputil.WriteError(w, http.StatusForbidden, "Account is disabled", "auth_error", "account_disabled")
		return
	}
	auth := AuthResult{User: user}

	// 6. Remove key_id from body before proxying
	delete(reqBody, "key_id")
	bodyBytes, _ = json.Marshal(reqBody)

	model, _ := reqBody["model"].(string)
	stream, _ := reqBody["stream"].(bool)

	if err := h.core.CheckModelAccess(&auth, model); err != nil {
		httputil.WriteError(w, http.StatusForbidden, err.Error(), "auth_error", "model_not_allowed")
		return
	}

	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = fmt.Sprintf("req_%d", time.Now().UnixNano())
	}

	if billingErr := h.core.CheckBilling(&auth, model); billingErr != nil {
		if errors.Is(billingErr, billing.ErrNoPricing) {
			httputil.WriteError(w, http.StatusForbidden, "Model billing not configured", "billing_error", "no_pricing")
		} else {
			httputil.WriteError(w, http.StatusPaymentRequired, "Insufficient balance", "billing_error", "insufficient_balance")
		}
		return
	}

	if stream {
		bodyBytes = injectStreamOptions(bodyBytes, reqBody)
	}

	// 7. Route & failover (same logic as Handler.ServeHTTP)
	upstreams, modelInfo, found := rt.GetUpstreams(model)
	canonicalName := modelInfo.CanonicalName
	if !found {
		httputil.WriteError(w, http.StatusNotFound, fmt.Sprintf("Model %q not found", model), "invalid_request_error", "model_not_found")
		return
	}

	upstreams = filterOpenAIUpstreams(upstreams)
	if len(upstreams) == 0 {
		httputil.WriteError(w, http.StatusServiceUnavailable,
			"No compatible upstream configured for this model",
			"server_error", "no_compatible_upstream")
		return
	}

	n := len(upstreams)
	tried := make(map[int]bool, n)
	start := uint64(0)

	for len(tried) < n {
		var upstream *balancer.Upstream
		var idx int
		ufound := false
		for i := 0; i < n; i++ {
			idx = int((start + uint64(i)) % uint64(n))
			if tried[idx] {
				continue
			}
			if upstreams[idx].Breaker.AllowRequest() {
				upstream = &upstreams[idx]
				ufound = true
				break
			}
			tried[idx] = true
		}
		if !ufound {
			break
		}
		tried[idx] = true

		upstreamBody := bodyBytes
		if upstream.Config.ModelOverride != "" {
			upstreamBody = replaceModelInBody(bodyBytes, upstream.Config.ModelOverride)
		}

		timeout := cfg.Server.RequestTimeout
		if timeout <= 0 {
			timeout = 30 * time.Minute
		}
		ctx, cancel := context.WithTimeout(r.Context(), timeout)

		upstreamURL := strings.TrimRight(upstream.Config.BaseURL, "/") + "/v1/chat/completions"
		upReq, err := http.NewRequestWithContext(ctx, http.MethodPost, upstreamURL, bytes.NewReader(upstreamBody))
		if err != nil {
			cancel()
			slog.Error("failed to create upstream request", "error", err)
			continue
		}

		upReq.Header.Set("Content-Type", "application/json")
		upReq.Header.Set("Authorization", "Bearer "+upstream.Config.APIKey)

		resp, err := h.client.Do(upReq)
		if err != nil {
			cancel()
			if circuit.IsUpstreamFailure(err) {
				upstream.Breaker.RecordFailure()
			}
			slog.Warn("upstream request failed", "upstream", upstream.Config.BaseURL, "error", err)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			upstream.Breaker.RecordFailure()
			lastErrBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			cancel()

			allTried := true
			for i := 0; i < n; i++ {
				if !tried[i] {
					allTried = false
					break
				}
			}
			if allTried {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(resp.StatusCode)
				w.Write(lastErrBody)
				return
			}
			continue
		}

		upstream.Breaker.RecordSuccess()
		slog.Info("playground request routed", "model", canonicalName, "upstream", upstream.Config.Provider, "user_id", userID)

		var usage billing.UsageInfo
		if stream {
			usage = relayStream(w, resp, canonicalName)
		} else {
			usage = relayNonStream(w, resp, canonicalName)
		}

		h.core.AsyncChargeAuth(&auth, canonicalName, requestID, usage)
		cancel()
		return
	}

	httputil.WriteError(w, http.StatusServiceUnavailable, "All upstream providers are unavailable", "server_error", "upstream_unavailable")
}
