package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zhulang/llm-gateway/internal/admin"
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
)

// Handler is the core reverse proxy that authenticates requests, routes them
// to upstream LLM providers with failover, and relays responses (including SSE streams).
type Handler struct {
	cfgHolder      *config.Holder
	router         atomic.Pointer[router.Router]
	balancer       *balancer.RoundRobin
	client         *http.Client
	store          store.Store
	keyCache       *apikey.Cache
	billingService *billing.BillingService
	touchBatcher   *apikey.TouchBatcher
	core           *Core
}

// NewHandler creates a Handler with the given config holder, router, balancer, store, key cache, billing service, and shared HTTP client.
func NewHandler(cfgHolder *config.Holder, rt *router.Router, lb *balancer.RoundRobin, s store.Store, kc *apikey.Cache, bs *billing.BillingService, client *http.Client, tb *apikey.TouchBatcher, core *Core) *Handler {
	h := &Handler{
		cfgHolder:      cfgHolder,
		balancer:       lb,
		client:         client,
		store:          s,
		keyCache:       kc,
		billingService: bs,
		touchBatcher:   tb,
		core:           core,
	}
	h.router.Store(rt)
	return h
}

// SetRouter atomically replaces the current router (used during hot reload).
func (h *Handler) SetRouter(rt *router.Router) {
	h.router.Store(rt)
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	metrics.Get().RequestsTotal.Add(1)
	defer func() { metrics.Get().RecordLatency(time.Since(startTime)) }()

	cfg := h.cfgHolder.Get()
	rt := h.router.Load()

	// 1. Auth via API key: extract Bearer token -> SHA256 hash -> cache/DB lookup
	token := httputil.ExtractBearerToken(r)
	if token == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "Invalid or missing API key", "auth_error", "invalid_api_key")
		return
	}

	keyHash := apikey.HashAPIKey(token)

	var auth AuthResult

	// Try user key cache first, then DB, then tenant keys.
	cachedAuth := h.keyCache.Get(keyHash)
	if cachedAuth != nil {
		auth = AuthResult{User: cachedAuth.User, APIKeyID: cachedAuth.APIKeyID, MemberTenantID: cachedAuth.MemberTenantID}
	} else {
		ak, err := h.store.GetAPIKeyByHash(keyHash)
		if err == nil {
			u, err := h.store.GetUserByID(ak.UserID)
			if err != nil {
				httputil.WriteError(w, http.StatusUnauthorized, "Invalid or missing API key", "auth_error", "invalid_api_key")
				return
			}
			memberTenantID := resolveMemberTenantID(h.store, u.ID)
			h.keyCache.Set(keyHash, &apikey.CachedAuth{User: u, APIKeyID: ak.ID, MemberTenantID: memberTenantID})
			h.touchBatcher.Touch(ak.ID)
			auth = AuthResult{User: u, APIKeyID: ak.ID, MemberTenantID: memberTenantID}
		} else {
			tk, tkErr := h.store.GetTenantAPIKeyByHash(keyHash)
			if tkErr == nil {
				go func() { _ = h.store.TouchTenantAPIKeyLastUsed(tk.ID) }()
				auth = AuthResult{TenantKey: tk}
			} else {
				// Try tenant sub-user key from DB.
				sk, skErr := h.store.GetTenantSubUserKeyByHash(keyHash)
				if skErr == nil {
					subUser, err := h.store.GetTenantSubUser(sk.SubUserID)
					if err != nil {
						httputil.WriteError(w, http.StatusUnauthorized, "Invalid or missing API key", "auth_error", "invalid_api_key")
						return
					}
					if subUser.Status != "active" {
						httputil.WriteError(w, http.StatusForbidden, "Sub-user account is disabled", "auth_error", "account_disabled")
						return
					}
					go func() { _ = h.store.TouchTenantSubUserKeyLastUsed(sk.ID) }()
					auth = AuthResult{SubUserKey: sk, SubUser: subUser}
				} else {
					httputil.WriteError(w, http.StatusUnauthorized, "Invalid or missing API key", "auth_error", "invalid_api_key")
					return
				}
			}
		}
	}

	auth.recordIdentity(r)

	if !auth.IsTenant() && !auth.IsSubUser() && auth.User.Status != "active" {
		httputil.WriteError(w, http.StatusForbidden, "Account is disabled", "auth_error", "account_disabled")
		return
	}

	if !auth.IsTenant() && !auth.IsSubUser() {
		r = r.WithContext(context.WithValue(r.Context(), admin.CtxUserIDKey, auth.User.ID))
		if h.core != nil && h.core.ActiveBatcher != nil {
			h.core.ActiveBatcher.Touch(auth.User.ID)
		}
	}

	// 2. Read body with size limit
	maxBytes := cfg.Server.MaxRequestBodyBytes
	if maxBytes <= 0 {
		maxBytes = 1 << 20 // 1MB default
	}
	limitedReader := http.MaxBytesReader(w, r.Body, maxBytes)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		httputil.WriteError(w, http.StatusRequestEntityTooLarge, "Request body too large", "invalid_request_error", "request_too_large")
		return
	}

	// 3. Parse JSON to extract model and stream
	var reqBody map[string]any
	if err := json.Unmarshal(bodyBytes, &reqBody); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid JSON in request body", "invalid_request_error", "invalid_json")
		return
	}

	model, _ := reqBody["model"].(string)
	stream, _ := reqBody["stream"].(bool)
	// Preserve the exact model string the client requested so we can echo it
	// back unchanged in the response. Cursor registers custom models with a
	// "gw/" prefix and won't reconcile a response whose model field differs
	// from what it sent (an un-prefixed name like "gpt-5.4" also looks like an
	// official OpenAI model to Cursor — exactly what the prefix is meant to
	// avoid). Billing and routing still use the normalized canonical name.
	clientModel := model
	model = router.NormalizeModelName(model)

	if model == "" || len(model) > 256 {
		httputil.WriteError(w, http.StatusBadRequest, "invalid model name", "invalid_request_error", "invalid_model")
		return
	}

	// 3.5 Model access check
	if err := h.core.CheckModelAccess(&auth, model); err != nil {
		httputil.WriteError(w, http.StatusForbidden, err.Error(), "auth_error", "model_not_allowed")
		return
	}

	// 3.6 Billing: check balance before request
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

	// 3.7 Content moderation (prompt-side; no billing side effects yet)
	if v := h.core.CheckModeration(&auth, model, moderation.ExtractOpenAIText(reqBody)); v.Flagged {
		httputil.WriteError(w, http.StatusForbidden, "Request rejected by content policy", "invalid_request_error", "content_policy_violation")
		return
	}

	traceInbound(requestID, model, r, bodyBytes)

	// If streaming, inject stream_options to get usage info in the last chunk.
	if stream {
		metrics.Get().RequestsStream.Add(1)
		bodyBytes = injectStreamOptions(bodyBytes, reqBody)
	}

	// 4. Route
	upstreams, modelInfo, found := rt.GetUpstreamsForTenant(auth.TenantID(), model)
	canonicalName := modelInfo.CanonicalName
	if !found {
		httputil.WriteError(w, http.StatusNotFound,
			fmt.Sprintf("Model %q not found", model),
			"invalid_request_error", "model_not_found")
		return
	}

	// Filter to OpenAI-compatible upstreams — strict passthrough, no cross-protocol conversion.
	upstreams = filterOpenAIUpstreams(upstreams)
	if len(upstreams) == 0 {
		httputil.WriteError(w, http.StatusServiceUnavailable,
			"No compatible upstream configured for this model",
			"server_error", "no_compatible_upstream")
		return
	}

	// 5. Failover loop — priority mode: always start from primary upstream (index 0).
	// Upstreams are sorted by sort_order, so index 0 = primary channel.
	// Backup channels are only used when the primary's circuit breaker is open.
	n := len(upstreams)
	tried := make(map[int]bool, n)
	start := uint64(0)

	for len(tried) < n {
		// Find next untried upstream with healthy breaker
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

		// Build upstream request body (apply model_override)
		upstreamBody := bodyBytes
		if upstream.Config.ModelOverride != "" {
			upstreamBody = replaceModelInBody(bodyBytes, upstream.Config.ModelOverride)
		}

		// Build request with timeout
		timeout := cfg.Server.RequestTimeout
		if timeout <= 0 {
			timeout = 30 * time.Minute
		}
		ctx, cancel := context.WithTimeout(r.Context(), timeout)

		upstreamURL := strings.TrimRight(upstream.Config.BaseURL, "/") + r.URL.Path
		traceUpstreamBody(requestID, model, upstreamURL, false, upstreamBody)
		upReq, err := http.NewRequestWithContext(ctx, r.Method, upstreamURL, bytes.NewReader(upstreamBody))
		if err != nil {
			cancel()
			slog.Error("failed to create upstream request", "error", err)
			continue
		}

		// Copy relevant headers
		upReq.Header.Set("Content-Type", "application/json")
		upReq.Header.Set("Authorization", "Bearer "+upstream.Config.APIKey)

		// Send request
		resp, err := h.client.Do(upReq)
		if err != nil {
			cancel()
			if circuit.IsUpstreamFailure(err) {
				upstream.Breaker.RecordFailure()
			}
			slog.Warn("upstream request failed", "upstream", upstream.Config.BaseURL, "error", err)
			continue
		}

		// Check for error status codes — try next upstream instead of returning immediately.
		// Save the last error so we can return it if all upstreams fail.
		if resp.StatusCode >= 400 {
			lastErrBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			cancel()

			// Context-window errors are client errors (input too long), not upstream
			// faults. Return the upstream's response directly without penalising the
			// circuit breaker or attempting failover.
			if isContextWindowExceededError(lastErrBody) {
				slog.Warn("context window exceeded, returning upstream error to client",
					"upstream", upstream.Config.BaseURL,
					"status", resp.StatusCode,
					"body", truncateStr(string(lastErrBody), 1024),
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(resp.StatusCode)
				w.Write(lastErrBody)
				return
			}

			// Only 429 and 5xx are upstream faults worth penalising the circuit
			// breaker. 4xx client errors (e.g. 400 "model does not support image
			// input") mean the request itself is invalid — failover would just
			// fail the same way on the next upstream, and recording failure
			// would let one user's bad requests trip the breaker for everyone.
			if resp.StatusCode == 429 || resp.StatusCode >= 500 {
				upstream.Breaker.RecordFailure()
			}
			slog.Warn("upstream returned error status, trying next",
				"upstream", upstream.Config.BaseURL,
				"status", resp.StatusCode)

			slog.Warn("upstream error response",
				"upstream", upstream.Config.BaseURL,
				"status", resp.StatusCode,
				"body", truncateStr(string(lastErrBody), 1024),
			)

			// If no more upstreams to try, return this error to client
			allTried := true
			for i := 0; i < n; i++ {
				if !tried[i] {
					allTried = false
					break
				}
			}
			if allTried {
				slog.Error("all upstreams failed",
					"model", canonicalName,
					"last_status", resp.StatusCode,
					"last_body", truncateStr(string(lastErrBody), 1024),
				)
				if auth.User != nil {
					userID := auth.User.ID
					go func() { _ = h.store.RecordRequestFailure(userID, canonicalName, http.StatusServiceUnavailable) }()
				}
				ct := resp.Header.Get("Content-Type")
				if ct == "" {
					ct = "application/json"
				}
				w.Header().Set("Content-Type", ct)
				w.WriteHeader(resp.StatusCode)
				w.Write(lastErrBody)
				return
			}
			continue
		}

		// Success
		upstream.Breaker.RecordSuccess()
		slog.Info("request routed",
			"model", canonicalName,
			"upstream", upstream.Config.Provider,
			"base_url", upstream.Config.BaseURL,
			"status", resp.StatusCode,
		)

		var usage billing.UsageInfo
		if stream {
			usage = relayStream(w, resp, clientModel)
		} else {
			usage = relayNonStream(w, resp, clientModel)
		}

		// Async billing charge after successful relay.
		h.core.AsyncChargeAuth(&auth, canonicalName, requestID, usage)

		cancel()
		return
	}

	if auth.User != nil {
		userID := auth.User.ID
		go func() { _ = h.store.RecordRequestFailure(userID, canonicalName, http.StatusServiceUnavailable) }()
	}
	httputil.WriteError(w, http.StatusServiceUnavailable,
		"All upstream providers are unavailable",
		"server_error", "upstream_unavailable")
}

// injectStreamOptions adds stream_options with include_usage: true to the request body.
func injectStreamOptions(bodyBytes []byte, reqBody map[string]any) []byte {
	reqBody["stream_options"] = map[string]any{"include_usage": true}
	out, err := json.Marshal(reqBody)
	if err != nil {
		return bodyBytes
	}
	return out
}

// sseHeartbeatInterval is the interval at which keep-alive comment lines are
// flushed to the client while waiting for the upstream to produce the next SSE
// chunk. It must stay well below the idle timeout of any reverse proxy in front
// of us (Cloudflare ~100s, nginx default proxy_read_timeout 60s).
const sseHeartbeatInterval = 15 * time.Second

// startSSEHeartbeat launches a goroutine that flushes an SSE comment line
// (": keep-alive\n\n") to the client every sseHeartbeatInterval until stop is
// called. Clients per the SSE spec ignore lines starting with ":", so this
// does not pollute the conversation stream. The returned stop function must
// be called when the main relay loop ends, and the caller must wait on it
// (via the provided WaitGroup) to ensure no concurrent write to w outlives
// the relay goroutine.
//
// All writes to w (both heartbeat and main relay loop) must be guarded by mu
// because http.ResponseWriter is not safe for concurrent use.
//
// startSSEHeartbeat returns the shared mutex to lock around writes to w.
func startSSEHeartbeat(w http.ResponseWriter, flusher http.Flusher, wg *sync.WaitGroup, ctx context.Context) (mu *sync.Mutex, stop context.CancelFunc) {
	mu = &sync.Mutex{}
	hbCtx, hbCancel := context.WithCancel(ctx)
	wg.Add(1)
	go func() {
		defer wg.Done()
		t := time.NewTicker(sseHeartbeatInterval)
		defer t.Stop()
		for {
			select {
			case <-hbCtx.Done():
				return
			case <-t.C:
				mu.Lock()
				fmt.Fprint(w, ": keep-alive\n\n")
				flusher.Flush()
				mu.Unlock()
			}
		}
	}()
	return mu, hbCancel
}

// relayNonStream reads the full response body, rewrites the model field, sends it,
// and returns usage info extracted from the response.
func relayNonStream(w http.ResponseWriter, resp *http.Response, respModel string) billing.UsageInfo {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("failed to read upstream response", "error", err)
		httputil.WriteError(w, http.StatusBadGateway, "Failed to read upstream response", "server_error", "upstream_read_error")
		return billing.UsageInfo{}
	}

	// Extract usage before rewriting.
	usage := extractUsageFromJSON(body)

	body = replaceModelInJSON(body, respModel)

	// Copy response headers
	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(body)

	return usage
}

// relayStream sets SSE headers and streams chunks, rewriting model in each data line.
// Returns accumulated usage info from the last chunk that contains usage data.
func relayStream(w http.ResponseWriter, resp *http.Response, respModel string) billing.UsageInfo {
	defer resp.Body.Close()

	flusher, ok := w.(http.Flusher)
	if !ok {
		slog.Error("response writer does not support flushing")
		httputil.WriteError(w, http.StatusInternalServerError, "Streaming not supported", "server_error", "no_flusher")
		return billing.UsageInfo{}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	var hbWg sync.WaitGroup
	mu, hbStop := startSSEHeartbeat(w, flusher, &hbWg, context.Background())
	defer func() {
		hbStop()
		hbWg.Wait()
	}()

	var usage billing.UsageInfo
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 10*1024*1024), 100*1024*1024) // 10MB initial, 100MB max
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			mu.Lock()
			fmt.Fprint(w, "\n")
			flusher.Flush()
			mu.Unlock()
			continue
		}

		// Try to extract usage from SSE data lines.
		if strings.HasPrefix(line, "data: ") {
			data := line[6:]
			if data != "[DONE]" {
				if u := extractUsageFromSSE(data); u.PromptTokens > 0 || u.CompletionTokens > 0 {
					u.CacheTokensIncludedInPrompt = true // OpenAI stream
					usage = u
					if traceOn() {
						slog.Info("trace.openai usage-raw", "model", respModel, "data", data)
					}
				}
			}
		}

		line = rewriteSSEModel(line, respModel)
		mu.Lock()
		fmt.Fprintf(w, "%s\n", line)
		flusher.Flush()
		mu.Unlock()
	}

	if err := scanner.Err(); err != nil {
		slog.Warn("upstream stream scan ended with error",
			"model", respModel,
			"error", err,
		)
	}

	return usage
}

// RelayChatStream relays an upstream Chat Completions SSE response directly to
// the client without Responses API conversion. Used by the responses handler
// for legacy clients (e.g. Cursor/1.0) that expect Chat Completions SSE format.
func RelayChatStream(w http.ResponseWriter, resp *http.Response, model string) billing.UsageInfo {
	return relayStream(w, resp, model)
}

// extractUsageFromJSON extracts usage.prompt_tokens and usage.completion_tokens from a JSON response body.
func extractUsageFromJSON(body []byte) billing.UsageInfo {
	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		return billing.UsageInfo{}
	}
	return extractUsageFromMap(resp)
}

// extractUsageFromSSE extracts usage info from a single SSE data payload (OpenAI format).
func extractUsageFromSSE(data string) billing.UsageInfo {
	var m map[string]any
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		return billing.UsageInfo{}
	}
	return extractUsageFromMap(m)
}

// extractUsageFromMap extracts usage info from a parsed JSON map (OpenAI format).
func extractUsageFromMap(m map[string]any) billing.UsageInfo {
	usageObj, ok := m["usage"]
	if !ok {
		return billing.UsageInfo{}
	}
	usageMap, ok := usageObj.(map[string]any)
	if !ok {
		return billing.UsageInfo{}
	}
	var info billing.UsageInfo
	info.CacheTokensIncludedInPrompt = true // OpenAI: prompt_tokens includes cached
	if pt, ok := usageMap["prompt_tokens"].(float64); ok {
		info.PromptTokens = int(pt)
	}
	if ct, ok := usageMap["completion_tokens"].(float64); ok {
		info.CompletionTokens = int(ct)
	}
	// Extract cache tokens from prompt_tokens_details
	if details, ok := usageMap["prompt_tokens_details"].(map[string]any); ok {
		if cached, ok := details["cached_tokens"].(float64); ok {
			info.CacheReadTokens = int(cached)
		}
		if info.CacheReadTokens == 0 {
			if cached, ok := details["cache_read_input_tokens"].(float64); ok {
				info.CacheReadTokens = int(cached)
			}
		}
		if cw, ok := details["cache_creation_input_tokens"].(float64); ok {
			info.CacheCreationTokens = int(cw)
		}
	}
	// Some providers put cache info at top-level usage or in completion_tokens_details
	if info.CacheReadTokens == 0 {
		if cr, ok := usageMap["cache_read_input_tokens"].(float64); ok {
			info.CacheReadTokens = int(cr)
		}
	}
	if info.CacheCreationTokens == 0 {
		if cw, ok := usageMap["cache_creation_input_tokens"].(float64); ok {
			info.CacheCreationTokens = int(cw)
		}
	}
	// yunwu.ai and similar providers expose cache creation as 5m/1h breakdown
	if v, ok := usageMap["claude_cache_creation_5_m_tokens"].(float64); ok && int(v) > 0 {
		info.CacheCreation5mTokens = int(v)
	}
	if v, ok := usageMap["claude_cache_creation_1_h_tokens"].(float64); ok && int(v) > 0 {
		info.CacheCreation1hTokens = int(v)
	}
	return info
}

// replaceModelInBody replaces the "model" field in JSON body bytes.
func replaceModelInBody(body []byte, newModel string) []byte {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return body
	}
	m["model"] = newModel
	out, err := json.Marshal(m)
	if err != nil {
		return body
	}
	return out
}

// replaceModelInJSON replaces the "model" field in a JSON response body using
// byte-level replacement to avoid full JSON unmarshal/marshal overhead.
func replaceModelInJSON(body []byte, newModel string) []byte {
	const prefix = `"model":"`
	idx := bytes.Index(body, []byte(prefix))
	if idx < 0 {
		return body
	}
	start := idx + len(prefix)
	end := bytes.IndexByte(body[start:], '"')
	if end < 0 {
		return body
	}
	end += start
	result := make([]byte, 0, len(body))
	result = append(result, body[:start]...)
	result = append(result, []byte(newModel)...)
	result = append(result, body[end:]...)
	return result
}

// rewriteSSEModel rewrites the model field in SSE "data: " lines using byte-level
// replacement to avoid full JSON unmarshal/marshal overhead on every chunk.
func rewriteSSEModel(line string, newModel string) string {
	if !strings.HasPrefix(line, "data: ") {
		return line
	}
	data := line[6:]
	if data == "[DONE]" {
		return line
	}
	// Fast path: use byte-level replacement for the "model" field
	// Pattern: "model":"<value>"
	const prefix = `"model":"`
	idx := strings.Index(data, prefix)
	if idx < 0 {
		return line
	}
	start := idx + len(prefix)
	end := strings.IndexByte(data[start:], '"')
	if end < 0 {
		return line
	}
	end += start
	return "data: " + data[:start] + newModel + data[end:]
}

// truncateStr truncates a string to maxLen bytes, appending "..." if truncated.
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// filterOpenAIUpstreams returns only upstreams that accept OpenAI Chat Completions format.
func filterOpenAIUpstreams(upstreams []balancer.Upstream) []balancer.Upstream {
	var out []balancer.Upstream
	for _, u := range upstreams {
		if IsOpenAICompatible(u.Config) {
			out = append(out, u)
		}
	}
	return out
}

