package gemini

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/balancer"
	"github.com/zhulang/llm-gateway/internal/bandwidth"
	"github.com/zhulang/llm-gateway/internal/billing"
	"github.com/zhulang/llm-gateway/internal/circuit"
	"github.com/zhulang/llm-gateway/internal/moderation"
	"github.com/zhulang/llm-gateway/internal/proxy"
	"github.com/zhulang/llm-gateway/internal/router"
)

// Handler serves the Google Gemini native API endpoints.
type Handler struct {
	core      *proxy.Core
	bwLimiter *bandwidth.Limiter
}

// NewHandler creates a new Gemini API handler.
func NewHandler(core *proxy.Core, bwl *bandwidth.Limiter) *Handler {
	return &Handler{core: core, bwLimiter: bwl}
}

// SetRouter atomically updates the router used by this handler.
func (h *Handler) SetRouter(rt *router.Router) {
	h.core.SetRouter(rt)
}

// ServeHTTP dispatches Gemini API requests based on URL path.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "Only POST is supported")
		return
	}

	model, action, apiVersion, ok := parseGeminiPath(r.URL.Path)
	if !ok {
		WriteError(w, http.StatusBadRequest, "Invalid Gemini API path")
		return
	}
	model = router.NormalizeModelName(model)

	switch action {
	case "generateContent", "streamGenerateContent":
		h.serveGenerate(w, r, model, action, apiVersion)
	case "cachedContents":
		h.serveCachedContents(w, r, model, apiVersion)
	default:
		WriteError(w, http.StatusBadRequest, "Unsupported action: "+action)
	}
}

// parseGeminiPath extracts model and action from Gemini API paths.
// Supported patterns:
//   /gemini/v1/models/{model}:generateContent
//   /gemini/v1beta/models/{model}:generateContent
//   /gemini/v1/models/{model}:streamGenerateContent
//   /gemini/v1beta/models/{model}:streamGenerateContent
//   /gemini/v1/models/{model}/cachedContents
//   /gemini/v1beta/models/{model}/cachedContents
func parseGeminiPath(path string) (model, action, apiVersion string, ok bool) {
	var prefix string
	if strings.HasPrefix(path, "/gemini/v1beta/models/") {
		prefix = "/gemini/v1beta/models/"
		apiVersion = "v1beta"
	} else if strings.HasPrefix(path, "/gemini/v1/models/") {
		prefix = "/gemini/v1/models/"
		apiVersion = "v1"
	} else if strings.HasPrefix(path, "/v1beta/models/") {
		prefix = "/v1beta/models/"
		apiVersion = "v1beta"
	} else if strings.HasPrefix(path, "/v1/models/") {
		prefix = "/v1/models/"
		apiVersion = "v1"
	} else {
		return "", "", "", false
	}

	rest := path[len(prefix):]
	if rest == "" {
		return "", "", "", false
	}

	// Check for colon-separated action (e.g., "gemini-3-pro-image-preview:generateContent")
	if idx := strings.LastIndex(rest, ":"); idx > 0 {
		return rest[:idx], rest[idx+1:], apiVersion, true
	}

	// Check for slash-separated action (e.g., "pa/gemini-3-flash-preview/cachedContents")
	if idx := strings.LastIndex(rest, "/"); idx > 0 {
		action = rest[idx+1:]
		if action == "cachedContents" {
			return rest[:idx], action, apiVersion, true
		}
	}

	return "", "", "", false
}

// serveGenerate handles generateContent and streamGenerateContent.
func (h *Handler) serveGenerate(w http.ResponseWriter, r *http.Request, model, action, apiVersion string) {
	rt := h.core.Router.Load()

	// 1. Auth
	auth, err := h.core.AuthenticateBearer(r)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, "Invalid or missing API key")
		return
	}
	if !auth.IsTenant() && !auth.IsSubUser() && auth.User.Status != "active" {
		WriteError(w, http.StatusForbidden, "Account is disabled")
		return
	}
	if !auth.IsTenant() && !auth.IsSubUser() {
		r = r.WithContext(context.WithValue(r.Context(), admin.CtxUserIDKey, auth.User.ID))
	}

	// 2. Read body
	body, err := h.core.ReadBody(w, r)
	if err != nil {
		cfg := h.core.CfgHolder.Get()
		maxMB := cfg.Server.MaxRequestBodyBytes / (1024 * 1024)
		WriteError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("Request body too large (limit: %dMB)", maxMB))
		return
	}
	if len(body) > 10*1024*1024 {
		slog.Info("gemini: large request body", "model", model, "body_size_mb", len(body)/(1024*1024))
	}

	// 3. Route
	upstreams, modelInfo, found := rt.GetUpstreamsForTenant(auth.TenantID(), model)
	if !found {
		WriteError(w, http.StatusNotFound, "Model not found: "+model)
		return
	}
	canonicalName := modelInfo.CanonicalName

	// 4. Model access check
	if err := h.core.CheckModelAccess(auth, canonicalName); err != nil {
		WriteError(w, http.StatusForbidden, err.Error())
		return
	}

	// 5. Billing check
	requestID := proxy.GetRequestID(r)
	if billingErr := h.core.CheckBilling(auth, canonicalName); billingErr != nil {
		if proxy.IsNoPricingError(billingErr) {
			WriteError(w, http.StatusForbidden, "Model billing not configured")
		} else {
			WriteError(w, http.StatusPaymentRequired, billingErr.Error())
		}
		return
	}

	// 5.5 Content moderation (prompt-side)
	if v := h.core.CheckModeration(auth, canonicalName, moderation.ExtractGeminiText(body)); v.Flagged {
		WriteError(w, http.StatusForbidden, "Request rejected by content policy")
		return
	}

	isStream := action == "streamGenerateContent"

	// Determine if this is an image request by checking billing_type
	isImageReq := false
	if p, err := h.core.Store.GetPricing(canonicalName); err == nil && p.BillingType == "image" {
		isImageReq = true
	}

	// 5. Failover
	usage, ok := h.failoverAndRelay(w, r.Context(), upstreams, body, model, action, apiVersion, isStream, isImageReq)

	// 6. Async billing - only charge on success
	if ok {
		h.asyncChargeWithBillingType(auth, canonicalName, requestID, usage, body)
	}
}

// serveCachedContents handles the cachedContents endpoint.
func (h *Handler) serveCachedContents(w http.ResponseWriter, r *http.Request, model, apiVersion string) {
	rt := h.core.Router.Load()

	auth, err := h.core.AuthenticateBearer(r)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, "Invalid or missing API key")
		return
	}
	if !auth.IsTenant() && !auth.IsSubUser() && auth.User.Status != "active" {
		WriteError(w, http.StatusForbidden, "Account is disabled")
		return
	}
	if !auth.IsTenant() && !auth.IsSubUser() {
		r = r.WithContext(context.WithValue(r.Context(), admin.CtxUserIDKey, auth.User.ID))
	}

	body, err := h.core.ReadBody(w, r)
	if err != nil {
		cfg := h.core.CfgHolder.Get()
		maxMB := cfg.Server.MaxRequestBodyBytes / (1024 * 1024)
		WriteError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("Request body too large (limit: %dMB)", maxMB))
		return
	}

	upstreams, modelInfo, found := rt.GetUpstreamsForTenant(auth.TenantID(), model)
	if !found {
		WriteError(w, http.StatusNotFound, "Model not found: "+model)
		return
	}
	canonicalName := modelInfo.CanonicalName

	if err := h.core.CheckModelAccess(auth, canonicalName); err != nil {
		WriteError(w, http.StatusForbidden, err.Error())
		return
	}

	requestID := proxy.GetRequestID(r)
	if billingErr := h.core.CheckBilling(auth, canonicalName); billingErr != nil {
		if proxy.IsNoPricingError(billingErr) {
			WriteError(w, http.StatusForbidden, "Model billing not configured")
		} else {
			WriteError(w, http.StatusPaymentRequired, billingErr.Error())
		}
		return
	}

	// cachedContents is always non-streaming
	usage, ok := h.failoverAndRelay(w, r.Context(), upstreams, body, model, "cachedContents", apiVersion, false, false)

	if ok {
		h.core.AsyncChargeAuth(auth, canonicalName, requestID, usage)
	}
}

// buildUpstreamURL constructs the upstream URL for a Gemini request.
// If the base_url already contains a version path (e.g. "/v1beta"), it is used as-is;
// otherwise apiVersion is appended (e.g. "v1beta").
func buildUpstreamURL(baseURL, effectiveModel, action, apiVersion string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	// Only append version if base URL doesn't already end with one.
	if !strings.HasSuffix(baseURL, "/v1") && !strings.HasSuffix(baseURL, "/v1beta") {
		baseURL += "/" + apiVersion
	}
	if action == "cachedContents" {
		return fmt.Sprintf("%s/models/%s/%s", baseURL, effectiveModel, action)
	}
	return fmt.Sprintf("%s/models/%s:%s", baseURL, effectiveModel, action)
}

// failoverAndRelay tries upstreams in priority order, relays the response, and returns usage.
// The bool return indicates whether the request was successfully relayed (true) or failed (false).
func (h *Handler) failoverAndRelay(w http.ResponseWriter, ctx context.Context, upstreams []balancer.Upstream, body []byte, model, action, apiVersion string, isStream bool, isImageReq bool) (billing.UsageInfo, bool) {
	// Filter to Gemini-protocol upstreams — strict passthrough, no cross-protocol conversion.
	geminiUpstreams := filterGeminiUpstreams(upstreams)
	if len(geminiUpstreams) == 0 {
		WriteError(w, http.StatusServiceUnavailable, "No compatible upstream configured for this model")
		return billing.UsageInfo{}, false
	}
	upstreams = geminiUpstreams

	cfg := h.core.CfgHolder.Get()
	timeout := cfg.Server.RequestTimeout
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}

	n := len(upstreams)
	tried := make(map[int]bool, n)

	for len(tried) < n {
		if ctx.Err() != nil {
			slog.Info("gemini: client disconnected, aborting failover", "model", model, "error", ctx.Err())
			return billing.UsageInfo{}, false
		}

		var idx int
		foundUpstream := false
		for i := 0; i < n; i++ {
			idx = i
			if tried[idx] {
				continue
			}
			if upstreams[idx].Breaker.AllowRequest() {
				foundUpstream = true
				break
			}
			tried[idx] = true
		}
		if !foundUpstream {
			break
		}
		tried[idx] = true
		upstream := &upstreams[idx]

		effectiveModel := model
		if upstream.Config.ModelOverride != "" {
			effectiveModel = upstream.Config.ModelOverride
		}

		baseURL := strings.TrimRight(upstream.Config.BaseURL, "/")
		upstreamURL := buildUpstreamURL(baseURL, effectiveModel, action, apiVersion)

		reqCtx, cancel := context.WithTimeout(ctx, timeout)
		upReq, err := http.NewRequestWithContext(reqCtx, http.MethodPost, upstreamURL, bytes.NewReader(body))
		if err != nil {
			cancel()
			slog.Error("gemini: failed to create upstream request", "error", err)
			continue
		}

		upReq.Header.Set("Content-Type", "application/json")
		upReq.Header.Set("Authorization", "Bearer "+upstream.Config.APIKey)

		resp, err := h.core.Client.Do(upReq)
		if err != nil {
			cancel()
			if circuit.IsUpstreamFailure(err) {
				upstream.Breaker.RecordFailure()
			}
			slog.Warn("gemini: upstream request failed", "upstream", upstream.Config.BaseURL, "error", err, "body_size_mb", len(body)/(1024*1024))
			continue
		}

		// All error status codes trigger failover
		if resp.StatusCode >= 400 {
			upstream.Breaker.RecordFailure()

			// Read error body for logging
			errBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			// Truncate error body for logging
			errBodyStr := string(errBody)
			if len(errBodyStr) > 512 {
				errBodyStr = errBodyStr[:512] + "..."
			}

			slog.Warn("gemini: upstream error status",
				"upstream", upstream.Config.BaseURL,
				"status", resp.StatusCode,
				"body", errBodyStr)

			allTried := len(tried) >= n
			if allTried {
				cancel()
				ct := resp.Header.Get("Content-Type")
				if ct == "" {
					ct = "application/json"
				}
				w.Header().Set("Content-Type", ct)
				w.WriteHeader(resp.StatusCode)
				w.Write(errBody)
				return billing.UsageInfo{}, false
			}

			cancel()
			continue
		}

		upstream.Breaker.RecordSuccess()
		slog.Info("gemini request routed",
			"model", model, "action", action,
			"upstream", upstream.Config.Provider,
			"base_url", upstream.Config.BaseURL,
			"status", resp.StatusCode)

		var usage billing.UsageInfo
		var relayOK bool
		if isStream {
			usage, relayOK = h.relayGeminiStream(w, resp, isImageReq)
		} else {
			usage, relayOK = h.relayGeminiNonStream(w, resp, isImageReq)
		}
		cancel()
		return usage, relayOK
	}

	WriteError(w, http.StatusServiceUnavailable, "All upstream providers are unavailable")
	return billing.UsageInfo{}, false
}

// relayGeminiNonStream reads the full upstream response, extracts usage, and writes it to the client.
func (h *Handler) relayGeminiNonStream(w http.ResponseWriter, resp *http.Response, isImageReq bool) (billing.UsageInfo, bool) {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		WriteError(w, http.StatusBadGateway, "Failed to read upstream response")
		return billing.UsageInfo{}, false
	}

	usage := extractGeminiUsage(body)

	if isImageReq && h.bwLimiter != nil {
		release, bwErr := h.bwLimiter.Acquire(context.Background())
		if bwErr != nil {
			WriteError(w, http.StatusServiceUnavailable, "bandwidth queue timeout")
			return billing.UsageInfo{}, false
		}
		defer release()
	}

	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(body)

	return usage, true
}

// relayGeminiStream relays an SSE stream from upstream, extracting usage from data chunks.
func (h *Handler) relayGeminiStream(w http.ResponseWriter, resp *http.Response, isImageReq bool) (billing.UsageInfo, bool) {
	defer resp.Body.Close()

	if isImageReq && h.bwLimiter != nil {
		release, bwErr := h.bwLimiter.Acquire(context.Background())
		if bwErr != nil {
			WriteError(w, http.StatusServiceUnavailable, "bandwidth queue timeout")
			return billing.UsageInfo{}, false
		}
		defer release()
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		WriteError(w, http.StatusInternalServerError, "Streaming not supported")
		return billing.UsageInfo{}, false
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	var usage billing.UsageInfo
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 10*1024*1024), 100*1024*1024) // 10MB initial, 100MB max

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "data: ") {
			data := line[6:]
			if data != "[DONE]" {
				if u := extractGeminiSSEUsage(data); u.PromptTokens > 0 || u.CompletionTokens > 0 {
					usage = u
				}
			}
		}

		fmt.Fprintf(w, "%s\n", line)
		if line == "" {
			flusher.Flush()
		}
	}
	flusher.Flush()

	if scanner.Err() != nil {
		slog.Warn("gemini: stream relay error", "error", scanner.Err())
		return usage, false
	}

	return usage, true
}

// asyncChargeWithBillingType charges based on the model's billing_type.
// For billing_type "image": charges per request (1 call = 1 unit), using imageSize from request body.
// For billing_type "token": charges based on actual token usage.
func (h *Handler) asyncChargeWithBillingType(auth *proxy.AuthResult, canonicalName, requestID string, usage billing.UsageInfo, body []byte) {
	// Get pricing to determine billing_type
	pricing, err := h.core.Store.GetPricing(canonicalName)
	if err != nil {
		// If pricing not found, use default token-based billing
		slog.Debug("pricing not found for model, using token billing", "model", canonicalName, "error", err)
		h.core.AsyncChargeAuth(auth, canonicalName, requestID, usage)
		return
	}

	if pricing.BillingType == "image" {
		if usage.PromptTokens == 0 && usage.CompletionTokens == 0 {
			slog.Warn("image billing skipped: no usage from upstream",
				"model", canonicalName, "request_id", requestID)
			return
		}
		// Image billing: charge per request, not per token
		imageUsage := billing.UsageInfo{
			PromptTokens: 1,
			ImageSize:    extractImageSize(body),
		}
		h.core.AsyncChargeAuth(auth, canonicalName, requestID, imageUsage)
		slog.Debug("image billing applied", "model", canonicalName, "request_id", requestID, "image_size", imageUsage.ImageSize)
	} else {
		// Token billing: use actual token usage from response
		h.core.AsyncChargeAuth(auth, canonicalName, requestID, usage)
	}
}

// extractImageSize parses the imageSize used for billing tier.
// Prefers the documented path generationConfig.responseFormat.image.imageSize,
// falling back to the legacy generationConfig.imageConfig.imageSize for older clients.
func extractImageSize(body []byte) string {
	var req struct {
		GenerationConfig struct {
			ResponseFormat struct {
				Image struct {
					ImageSize string `json:"imageSize"`
				} `json:"image"`
			} `json:"responseFormat"`
			ImageConfig struct {
				ImageSize string `json:"imageSize"`
			} `json:"imageConfig"`
		} `json:"generationConfig"`
	}
	if json.Unmarshal(body, &req) != nil {
		return ""
	}
	if s := req.GenerationConfig.ResponseFormat.Image.ImageSize; s != "" {
		return s
	}
	return req.GenerationConfig.ImageConfig.ImageSize
}

// filterGeminiUpstreams returns only upstreams configured with protocol="gemini".
func filterGeminiUpstreams(upstreams []balancer.Upstream) []balancer.Upstream {
	var out []balancer.Upstream
	for _, u := range upstreams {
		if proxy.IsGeminiAPI(u.Config) {
			out = append(out, u)
		}
	}
	return out
}

