package anthropic

import (
	"bufio"
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
	"github.com/zhulang/llm-gateway/internal/billing"
	"github.com/zhulang/llm-gateway/internal/metrics"
	"github.com/zhulang/llm-gateway/internal/moderation"
	"github.com/zhulang/llm-gateway/internal/proxy"
	"github.com/zhulang/llm-gateway/internal/router"
)

// Handler serves the Anthropic Messages API (/v1/messages).
type Handler struct {
	core *proxy.Core
}

// NewHandler creates a new Anthropic API handler.
func NewHandler(core *proxy.Core) *Handler {
	return &Handler{core: core}
}

// SetRouter atomically updates the router used by this handler.
func (h *Handler) SetRouter(rt *router.Router) {
	h.core.SetRouter(rt)
}

// extractAnthropicHeaders extracts Anthropic-specific headers from the client
// request to be forwarded to the upstream (e.g. anthropic-beta for 1m context).
func extractAnthropicHeaders(r *http.Request) http.Header {
	h := make(http.Header)
	for _, key := range []string{"anthropic-beta", "anthropic-version"} {
		if v := r.Header.Get(key); v != "" {
			h.Set(key, v)
		}
	}
	return h
}

// ServeHTTP implements http.Handler for the Anthropic Messages API.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	metrics.Get().RequestsTotal.Add(1)
	defer func() { metrics.Get().RecordLatency(time.Since(startTime)) }()

	rt := h.core.Router.Load()

	// 1. Auth: support both x-api-key and Authorization: Bearer
	auth, err := h.core.AuthenticateAny(r)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, "authentication_error", "Invalid or missing API key")
		return
	}

	if !auth.IsTenant() && !auth.IsSubUser() && auth.User.Status != "active" {
		WriteError(w, http.StatusForbidden, "permission_error", "Account is disabled")
		return
	}

	if !auth.IsTenant() && !auth.IsSubUser() {
		r = r.WithContext(context.WithValue(r.Context(), admin.CtxUserIDKey, auth.User.ID))
	}

	// 2. Read body
	bodyBytes, err := h.core.ReadBody(w, r)
	if err != nil {
		WriteError(w, http.StatusRequestEntityTooLarge, "invalid_request_error", "Request body too large")
		return
	}

	// 3. Parse Anthropic request
	var req MessagesRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request_error", "Invalid JSON in request body")
		return
	}

	// 4. Validate required fields
	// When extended thinking is enabled, max_tokens may be absent (budgeting
	// is controlled by thinking.budget_tokens instead).
	var rawFields map[string]json.RawMessage
	json.Unmarshal(bodyBytes, &rawFields)
	hasThinking := rawFields["thinking"] != nil

	if req.MaxTokens <= 0 && !hasThinking {
		WriteError(w, http.StatusBadRequest, "invalid_request_error", "max_tokens is required and must be > 0")
		return
	}

	// 5. Route
	model := req.Model
	model = router.NormalizeModelName(model)
	slog.Info("anthropic messages routing",
		"requested_model", req.Model,
		"normalized_model", model,
		"tenant_id", auth.TenantID(),
	)
	upstreams, modelInfo, found := rt.GetUpstreamsForTenant(auth.TenantID(), model)
	if !found {
		slog.Warn("model not found", "requested_model", model, "normalized", model, "tenant_id", auth.TenantID())
		WriteError(w, http.StatusNotFound, "not_found_error", "Model \""+model+"\" not found")
		return
	}
	canonicalName := modelInfo.CanonicalName
	slog.Info("model found", "canonical_name", canonicalName, "upstreams_count", len(upstreams))

	// 6. Model access check
	if err := h.core.CheckModelAccess(auth, canonicalName); err != nil {
		WriteError(w, http.StatusForbidden, "invalid_request_error", err.Error())
		return
	}

	// 7. Billing check
	requestID := proxy.GetRequestID(r)
	if billingErr := h.core.CheckBilling(auth, canonicalName); billingErr != nil {
		if proxy.IsNoPricingError(billingErr) {
			WriteError(w, http.StatusForbidden, "invalid_request_error", "Model billing not configured")
		} else {
			WriteError(w, http.StatusPaymentRequired, "invalid_request_error", billingErr.Error())
		}
		return
	}

	// 7.5 Content moderation (prompt-side)
	{
		roles := make([]string, len(req.Messages))
		contents := make([]json.RawMessage, len(req.Messages))
		for i, m := range req.Messages {
			roles[i] = m.Role
			contents[i] = m.Content
		}
		if v := h.core.CheckModeration(auth, canonicalName, moderation.ExtractAnthropicText(req.System, roles, contents)); v.Flagged {
			WriteError(w, http.StatusForbidden, "invalid_request_error", "Request rejected by content policy")
			return
		}
	}

	// 7. Extract Anthropic-specific headers to forward upstream
	extraHeaders := extractAnthropicHeaders(r)

	// 8. 同协议优先：声明了 anthropic 协议的上游走透传。
	//    若该批上游全部失败（5xx 或网络错误），fall through 到 OpenAI Chat 兜底。
	anthUpstreams := filterAnthropicUpstreams(upstreams)
	if len(anthUpstreams) > 0 {
		forwardBody := bodyBytes
		if anthUpstreams[0].Config.ModelOverride != "" {
			forwardBody = replaceModelInBody(forwardBody, anthUpstreams[0].Config.ModelOverride)
		}
		result, ferr := h.core.Failover(r.Context(), anthUpstreams, forwardBody, "POST", "/v1/messages", extraHeaders)
		if ferr == nil {
			// Failover 已穷尽 5xx 重试，返回的 status code 是上游最后状态。
			// 4xx 视为客户端错误透传（不 fallback，重写到 OpenAI 也大概率仍 4xx）。
			// 5xx 透传给客户端前，先尝试 OpenAI Chat 兜底。
			if result.Response.StatusCode < 500 {
				defer result.Cancel()
				if result.Response.StatusCode != http.StatusOK {
					w.Header().Set("Content-Type", result.Response.Header.Get("Content-Type"))
					w.WriteHeader(result.Response.StatusCode)
					io.Copy(w, result.Response.Body)
					result.Response.Body.Close()
					return
				}

				var usage billing.UsageInfo
				if req.Stream {
					usage = relayAnthropicStreamDirect(w, result.Response, canonicalName)
				} else {
					usage = h.relayAnthropicDirect(w, result.Response, canonicalName)
				}
				slog.Info("anthropic cache stats",
					"model", canonicalName,
					"provider", result.Upstream.Config.Provider,
					"upstream_name", result.Upstream.Config.UpstreamName,
					"base_url", result.Upstream.Config.BaseURL,
					"stream", req.Stream,
					"prompt_tokens", usage.PromptTokens,
					"cache_read_tokens", usage.CacheReadTokens,
					"cache_creation_tokens", usage.CacheCreationTokens,
				)
				h.core.AsyncChargeAuth(auth, canonicalName, requestID, usage)
				return
			}
			// 5xx：放弃这个 result，落到 OpenAI Chat 兜底
			result.Cancel()
		} else {
			slog.Warn("anthropic upstream failover failed for /v1/messages, falling back to OpenAI Chat",
				"model", canonicalName,
				"upstream_count", len(anthUpstreams),
				"error", ferr,
			)
		}
	} else {
		slog.Info("no anthropic upstream for /v1/messages, trying OpenAI Chat fallback",
			"model", canonicalName,
			"total_upstreams", len(upstreams),
		)
	}

	// 9. 兜底：声明了 openai/openai-compatible 协议的上游，做协议转换。
	openaiUpstreams := filterOpenAIChatUpstreams(upstreams)
	if len(openaiUpstreams) == 0 {
		WriteError(w, http.StatusServiceUnavailable, "server_error",
			"No compatible upstream configured for this model")
		return
	}

	// 转换请求：Anthropic MessagesRequest → OpenAI Chat Completions map
	openaiReqMap, convErr := ConvertRequest(&req)
	if convErr != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request_error",
			"Failed to convert request: "+convErr.Error())
		return
	}
	if openaiUpstreams[0].Config.ModelOverride != "" {
		openaiReqMap["model"] = openaiUpstreams[0].Config.ModelOverride
	}
	openaiBody, err := json.Marshal(openaiReqMap)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "api_error", "Failed to encode converted request")
		return
	}

	result, ferr := h.core.Failover(r.Context(), openaiUpstreams, openaiBody, "POST", "/v1/chat/completions", nil)
	if ferr != nil {
		slog.Warn("openai chat fallback failover failed for /v1/messages",
			"model", canonicalName,
			"upstream_count", len(openaiUpstreams),
			"error", ferr,
		)
		WriteError(w, http.StatusServiceUnavailable, "overloaded_error", "All upstream providers are unavailable")
		return
	}
	defer result.Cancel()

	if result.Response.StatusCode != http.StatusOK {
		w.Header().Set("Content-Type", result.Response.Header.Get("Content-Type"))
		w.WriteHeader(result.Response.StatusCode)
		io.Copy(w, result.Response.Body)
		result.Response.Body.Close()
		return
	}

	// 转换响应：OpenAI Chat → Anthropic
	var usage billing.UsageInfo
	if req.Stream {
		inputTokens := estimateInputTokensFromRequest(&req)
		usage = RelayStream(w, result.Response, canonicalName, inputTokens)
	} else {
		usage = h.relayNonStream(w, result.Response, canonicalName)
	}
	slog.Info("anthropic via openai chat upstream",
		"model", canonicalName,
		"provider", result.Upstream.Config.Provider,
		"upstream_name", result.Upstream.Config.UpstreamName,
		"base_url", result.Upstream.Config.BaseURL,
		"stream", req.Stream,
		"prompt_tokens", usage.PromptTokens,
	)
	h.core.AsyncChargeAuth(auth, canonicalName, requestID, usage)
}

// relayNonStream reads an OpenAI Chat Completions upstream response, converts
// it to Anthropic MessagesResponse format, and writes it to the client. Used
// by the /v1/messages entry when falling back to an OpenAI Chat-compatible
// upstream (no Anthropic-protocol upstream available or all of them failed).
func (h *Handler) relayNonStream(w http.ResponseWriter, resp *http.Response, canonicalModel string) billing.UsageInfo {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		WriteError(w, http.StatusBadGateway, "api_error", "Failed to read upstream response")
		return billing.UsageInfo{}
	}

	// Extract usage from the raw OpenAI response for billing
	usage := proxy.ExtractUsageFromJSON(body)

	// Convert to Anthropic format
	anthResp, err := ConvertResponse(body, canonicalModel)
	if err != nil {
		WriteError(w, http.StatusBadGateway, "api_error", "Failed to convert upstream response")
		return usage
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(anthResp)

	return usage
}

// relayAnthropicDirect relays an Anthropic upstream response directly to the client
// (no format conversion needed since both client and upstream speak Anthropic).
func (h *Handler) relayAnthropicDirect(w http.ResponseWriter, resp *http.Response, canonicalModel string) billing.UsageInfo {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		WriteError(w, http.StatusBadGateway, "api_error", "Failed to read upstream response")
		return billing.UsageInfo{}
	}

	// Extract usage from Anthropic response
	var anthResp MessagesResponse
	if err := json.Unmarshal(body, &anthResp); err == nil {
		// Rewrite model name to canonical
		anthResp.Model = canonicalModel
		rewritten, err := json.Marshal(&anthResp)
		if err == nil {
			body = rewritten
		}
	}

	usage := billing.UsageInfo{
		PromptTokens:                anthResp.Usage.InputTokens,
		CompletionTokens:            anthResp.Usage.OutputTokens,
		CacheCreationTokens:         anthResp.Usage.CacheCreationInputTokens,
		CacheReadTokens:             anthResp.Usage.CacheReadInputTokens,
		CacheTokensIncludedInPrompt: false,
	}
	if anthResp.Usage.CacheCreation != nil {
		usage.CacheCreation5mTokens = anthResp.Usage.CacheCreation.Ephemeral5mInputTokens
		usage.CacheCreation1hTokens = anthResp.Usage.CacheCreation.Ephemeral1hInputTokens
	}
	// Third-party compatibility (e.g. 4sapi): flat 5m/1h fields at usage level.
	if anthResp.Usage.ClaudeCacheCreation5mTokens > 0 && usage.CacheCreation5mTokens == 0 {
		usage.CacheCreation5mTokens = anthResp.Usage.ClaudeCacheCreation5mTokens
	}
	if anthResp.Usage.ClaudeCacheCreation1hTokens > 0 && usage.CacheCreation1hTokens == 0 {
		usage.CacheCreation1hTokens = anthResp.Usage.ClaudeCacheCreation1hTokens
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)

	return usage
}

// relayAnthropicStreamDirect relays an Anthropic SSE stream directly to the client,
// extracting usage for billing.
func relayAnthropicStreamDirect(w http.ResponseWriter, resp *http.Response, canonicalModel string) billing.UsageInfo {
	defer resp.Body.Close()

	flusher, ok := w.(http.Flusher)
	if !ok {
		WriteError(w, http.StatusInternalServerError, "api_error", "Streaming not supported")
		return billing.UsageInfo{}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	var usage billing.UsageInfo
	usage.CacheTokensIncludedInPrompt = false
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 10*1024*1024), 100*1024*1024) // 10MB initial, 100MB max
	for scanner.Scan() {
		line := scanner.Text()

		// Extract usage from streaming events for billing and rewrite model name
		if strings.HasPrefix(line, "data: ") {
			data := line[6:]
			var evt map[string]any
			if err := json.Unmarshal([]byte(data), &evt); err == nil {
				if evtType, _ := evt["type"].(string); evtType == "message_delta" {
					if usageObj, ok := evt["usage"].(map[string]any); ok {
						if ot, ok := usageObj["output_tokens"].(float64); ok {
							usage.CompletionTokens = int(ot)
						}
						// message_delta usage is cumulative and authoritative: when a
						// turn invokes server-side tools (e.g. web_search), input_tokens
						// here exceeds the message_start value. Overwrite so billing/stats
						// reflect the final input count rather than the first segment.
						if it, ok := usageObj["input_tokens"].(float64); ok && int(it) > 0 {
							usage.PromptTokens = int(it)
						}
						// message_delta carries the final authoritative cache token counts;
						// always overwrite values set by message_start.
						if cc, ok := usageObj["cache_creation_input_tokens"].(float64); ok && int(cc) > 0 {
							usage.CacheCreationTokens = int(cc)
						}
						if cr, ok := usageObj["cache_read_input_tokens"].(float64); ok && int(cr) > 0 {
							usage.CacheReadTokens = int(cr)
						}
						if ccObj, ok := usageObj["cache_creation"].(map[string]any); ok {
							if v, ok := ccObj["ephemeral_5m_input_tokens"].(float64); ok && int(v) > 0 {
								usage.CacheCreation5mTokens = int(v)
							}
							if v, ok := ccObj["ephemeral_1h_input_tokens"].(float64); ok && int(v) > 0 {
								usage.CacheCreation1hTokens = int(v)
							}
						}
						if v, ok := usageObj["claude_cache_creation_5_m_tokens"].(float64); ok && int(v) > 0 {
							usage.CacheCreation5mTokens = int(v)
						}
						if v, ok := usageObj["claude_cache_creation_1_h_tokens"].(float64); ok && int(v) > 0 {
							usage.CacheCreation1hTokens = int(v)
						}
					}
				}
				if evtType, _ := evt["type"].(string); evtType == "message_start" {
					if msg, ok := evt["message"].(map[string]any); ok {
						// Rewrite model name to canonical
						msg["model"] = canonicalModel
						if rewritten, err := json.Marshal(evt); err == nil {
							line = "data: " + string(rewritten)
						}
						if usageObj, ok := msg["usage"].(map[string]any); ok {
							if it, ok := usageObj["input_tokens"].(float64); ok {
								usage.PromptTokens = int(it)
							}
							if cc, ok := usageObj["cache_creation_input_tokens"].(float64); ok {
								usage.CacheCreationTokens = int(cc)
							}
							if cr, ok := usageObj["cache_read_input_tokens"].(float64); ok {
								usage.CacheReadTokens = int(cr)
							}
							// Extract 5m/1h cache creation breakdown
							if ccObj, ok := usageObj["cache_creation"].(map[string]any); ok {
								if v, ok := ccObj["ephemeral_5m_input_tokens"].(float64); ok {
									usage.CacheCreation5mTokens = int(v)
								}
								if v, ok := ccObj["ephemeral_1h_input_tokens"].(float64); ok {
									usage.CacheCreation1hTokens = int(v)
								}
							}
							// Third-party compatibility (e.g. 4sapi): flat 5m/1h fields.
							if v, ok := usageObj["claude_cache_creation_5_m_tokens"].(float64); ok && int(v) > 0 && usage.CacheCreation5mTokens == 0 {
								usage.CacheCreation5mTokens = int(v)
							}
							if v, ok := usageObj["claude_cache_creation_1_h_tokens"].(float64); ok && int(v) > 0 && usage.CacheCreation1hTokens == 0 {
								usage.CacheCreation1hTokens = int(v)
							}
						}
					}
				}
			}
		}

		fmt.Fprintf(w, "%s\n", line)
		if line == "" {
			flusher.Flush()
		}
	}

	// Ensure final flush and send empty line to complete SSE stream
	fmt.Fprintf(w, "\n")
	flusher.Flush()

	return usage
}

// ServeCountTokens handles the /v1/messages/count_tokens endpoint.
// It authenticates the request, routes by model, and tries to proxy to the upstream.
// If the upstream doesn't support count_tokens (404), falls back to local estimation.
func (h *Handler) ServeCountTokens(w http.ResponseWriter, r *http.Request) {
	rt := h.core.Router.Load()

	// 1. Auth
	auth, err := h.core.AuthenticateAny(r)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, "authentication_error", "Invalid or missing API key")
		return
	}
	if !auth.IsTenant() && !auth.IsSubUser() && auth.User.Status != "active" {
		WriteError(w, http.StatusForbidden, "permission_error", "Account is disabled")
		return
	}
	if !auth.IsTenant() && !auth.IsSubUser() {
		r = r.WithContext(context.WithValue(r.Context(), admin.CtxUserIDKey, auth.User.ID))
	}

	// 2. Read body
	bodyBytes, err := h.core.ReadBody(w, r)
	if err != nil {
		WriteError(w, http.StatusRequestEntityTooLarge, "invalid_request_error", "Request body too large")
		return
	}

	// 3. Extract model and messages from request
	var req map[string]any
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request_error", "Invalid JSON in request body")
		return
	}

	model, _ := req["model"].(string)
	if model == "" {
		WriteError(w, http.StatusBadRequest, "invalid_request_error", "Model is required")
		return
	}

	// 4. Route
	upstreams, _, found := rt.GetUpstreamsForTenant(auth.TenantID(), model)
	if !found {
		slog.Warn("count_tokens: model not found", "requested_model", model)
		WriteError(w, http.StatusNotFound, "not_found_error", "Model \""+model+"\" not found")
		return
	}

	// 5. Only anthropic upstreams support count_tokens
	if !proxy.IsAnthropicAPI(upstreams[0].Config) {
		WriteError(w, http.StatusBadRequest, "invalid_request_error", "count_tokens is only supported for Anthropic models")
		return
	}

	// 6. Apply model_override if needed
	forwardBody := bodyBytes
	if upstreams[0].Config.ModelOverride != "" {
		forwardBody = replaceModelInBody(forwardBody, upstreams[0].Config.ModelOverride)
	}

	// 7. Try to forward to upstream
	extraHeaders := extractAnthropicHeaders(r)
	result, err := h.core.Failover(r.Context(), upstreams, forwardBody, "POST", "/v1/messages/count_tokens", extraHeaders)

	// 8. If upstream doesn't support count_tokens (404), use local estimation
	if err != nil || (result != nil && result.Response.StatusCode == 404) {
		if result != nil {
			result.Cancel()
		}
		slog.Debug("upstream doesn't support count_tokens, using local estimation", "model", model)

		// Local token estimation
		system, _ := req["system"].(string)
		messages, _ := req["messages"].([]any)
		inputTokens := estimateTokens(system, messages)

		response := map[string]any{
			"input_tokens": inputTokens,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}
	defer result.Cancel()

	// 9. Relay upstream response
	defer result.Response.Body.Close()
	body, err := io.ReadAll(result.Response.Body)
	if err != nil {
		WriteError(w, http.StatusBadGateway, "api_error", "Failed to read upstream response")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(result.Response.StatusCode)
	w.Write(body)
}

// estimateTokens provides a rough token count estimation.
// Uses ~4 characters per token as a rough approximation (Claude's average).
func estimateTokens(system string, messages []any) int {
	totalChars := len(system)

	for _, m := range messages {
		msg, ok := m.(map[string]any)
		if !ok {
			continue
		}

		content := msg["content"]

		// Handle string content
		if str, ok := content.(string); ok {
			totalChars += len(str)
			continue
		}

		// Handle array content (multi-modal)
		if arr, ok := content.([]any); ok {
			for _, item := range arr {
				if block, ok := item.(map[string]any); ok {
					if text, ok := block["text"].(string); ok {
						totalChars += len(text)
					}
				}
			}
		}
	}

	// ~4 characters per token is a reasonable approximation for English text
	// Add 20% buffer for JSON structure overhead
	tokens := int(float64(totalChars) / 4.0 * 1.2)

	return tokens
}

// replaceModelInBody replaces the "model" field in a JSON body.
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

// filterAnthropicUpstreams returns only upstreams with Anthropic protocol.
func filterAnthropicUpstreams(upstreams []balancer.Upstream) []balancer.Upstream {
	var out []balancer.Upstream
	for _, u := range upstreams {
		if proxy.IsAnthropicAPI(u.Config) {
			out = append(out, u)
		}
	}
	return out
}

// filterOpenAIChatUpstreams returns upstreams that declare an OpenAI Chat
// Completions-compatible protocol (openai or openai-compatible). Used by the
// /v1/messages entry to pick the fallback upstream set when no Anthropic
// upstream is available (or all Anthropic upstreams failed).
func filterOpenAIChatUpstreams(upstreams []balancer.Upstream) []balancer.Upstream {
	var out []balancer.Upstream
	for _, u := range upstreams {
		if proxy.IsOpenAIChatCompatible(u.Config) {
			out = append(out, u)
		}
	}
	return out
}

// estimateInputTokensFromRequest computes a rough prompt-token estimate from a
// parsed Anthropic MessagesRequest. It is used as a placeholder for the
// message_start event's input_tokens on the OpenAI-compatible streaming path,
// where the upstream only returns real usage in the final chunk — too late
// for message_start. The estimate never affects billing (billing uses the
// authoritative usage from the upstream's final chunk); it only fills in
// message_start.usage.input_tokens so clients observing that event see a
// non-zero value consistent with the request size.
func estimateInputTokensFromRequest(req *MessagesRequest) int {
	totalChars := 0

	if len(req.System) > 0 {
		if sys, err := convertSystemMessage(req.System); err == nil {
			totalChars += len(sys)
		}
	}

	for _, msg := range req.Messages {
		// String content
		var strContent string
		if err := json.Unmarshal(msg.Content, &strContent); err == nil {
			totalChars += len(strContent)
			continue
		}

		// Array of content blocks
		var blocks []ContentBlock
		if err := json.Unmarshal(msg.Content, &blocks); err != nil {
			continue
		}
		for _, b := range blocks {
			switch b.Type {
			case "text":
				totalChars += len(b.Text)
			case "tool_use":
				// tool_use carries function name + arguments as JSON; count
				// both so tool-heavy turns aren't drastically under-estimated.
				totalChars += len(b.Name)
				totalChars += len(b.Input)
			case "tool_result":
				totalChars += len(b.Content)
			case "thinking":
				totalChars += len(b.Thinking)
			}
		}
	}

	return int(float64(totalChars) / 4.0 * 1.2)
}

