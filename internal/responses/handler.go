package responses

import (
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
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/metrics"
	"github.com/zhulang/llm-gateway/internal/proxy"
	"github.com/zhulang/llm-gateway/internal/router"
)

// Handler implements the OpenAI Responses API by translating requests/responses
// to/from the Chat Completions API format and proxying to upstream providers.
type Handler struct {
	core *proxy.Core
}

// NewHandler creates a new Responses API handler with the given proxy core.
func NewHandler(core *proxy.Core) *Handler {
	return &Handler{core: core}
}

// SetRouter atomically replaces the router (used during hot reload).
func (h *Handler) SetRouter(rt *router.Router) {
	h.core.SetRouter(rt)
}

// ServeHTTP implements http.Handler for the Responses API endpoint.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	metrics.Get().RequestsTotal.Add(1)
	defer func() { metrics.Get().RecordLatency(time.Since(startTime)) }()

	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "Only POST is supported", "invalid_request_error", "method_not_allowed")
		return
	}

	rt := h.core.Router.Load()

	// 1. Auth
	auth, err := h.core.AuthenticateBearer(r)
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "Invalid or missing API key", "auth_error", "invalid_api_key")
		return
	}

	if !auth.IsTenant() && !auth.IsSubUser() && auth.User.Status != "active" {
		httputil.WriteError(w, http.StatusForbidden, "Account is disabled", "auth_error", "account_disabled")
		return
	}

	// 2. Inject user ID into context
	if !auth.IsTenant() && !auth.IsSubUser() {
		r = r.WithContext(context.WithValue(r.Context(), admin.CtxUserIDKey, auth.User.ID))
	}

	// 3. Read body
	bodyBytes, err := h.core.ReadBody(w, r)
	if err != nil {
		httputil.WriteError(w, http.StatusRequestEntityTooLarge, "Request body too large", "invalid_request_error", "request_too_large")
		return
	}

	// 4. Parse into CreateRequest
	var req CreateRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid JSON in request body", "invalid_request_error", "invalid_json")
		return
	}

	if req.Model == "" {
		httputil.WriteError(w, http.StatusBadRequest, "model is required", "invalid_request_error", "missing_model")
		return
	}
	if dup := findDuplicateFunctionCallID(req.Input); dup != "" {
		slog.Warn("responses: duplicate call_id in input, rejecting",
			"call_id", dup, "user_agent", r.UserAgent())
		httputil.WriteError(w, http.StatusBadRequest,
			"Duplicate function_call call_id in input: "+dup+". Check conversation history for repeated tool calls.",
			"invalid_request_error", "duplicate_call_id")
		return
	}
	if respTrace {
		var top map[string]json.RawMessage
		_ = json.Unmarshal(bodyBytes, &top)
		keys := make([]string, 0, len(top))
		for k := range top {
			keys = append(keys, k)
		}
		// Dump tools array raw — needed to diagnose upstream "tools[N].function:
		// missing field name" errors where Codex sends a non-function tool type
		// that our converter passes through flat. Truncated to avoid log bloat.
		toolsRaw := top["tools"]
		if len(toolsRaw) > 60000 {
			toolsRaw = append(toolsRaw[:60000], []byte("...(truncated)")...)
		}
		slog.Info("trace.responses inbound",
			"model", req.Model,
			"stream", req.Stream,
			"user_agent", r.UserAgent(),
			"top_keys", strings.Join(keys, ","),
			"tools_count", len(req.Tools),
			"tools_raw", string(toolsRaw),
			"reasoning", string(top["reasoning"]),
			"include", string(top["include"]),
			"instructions_present", top["instructions"] != nil,
			"tool_choice", string(top["tool_choice"]),
			"text", string(top["text"]),
			"bodylen", len(bodyBytes))
	}
	clientModel := req.Model
	req.Model = router.NormalizeModelName(req.Model)

	// 5. Route
	upstreams, modelInfo, found := rt.GetUpstreamsForTenant(auth.TenantID(), req.Model)
	if !found {
		slog.Warn("responses model not found", "model", req.Model, "remote", httputil.ClientIP(r), "user_agent", r.UserAgent())
		httputil.WriteError(w, http.StatusNotFound,
			fmt.Sprintf("Model %q not found", req.Model),
			"invalid_request_error", "model_not_found")
		return
	}
	canonicalName := modelInfo.CanonicalName

	// 6. Model access check
	if err := h.core.CheckModelAccess(auth, canonicalName); err != nil {
		httputil.WriteError(w, http.StatusForbidden, err.Error(), "auth_error", "model_not_allowed")
		return
	}

	// 7. Billing check
	requestID := proxy.GetRequestID(r)
	if billingErr := h.core.CheckBilling(auth, canonicalName); billingErr != nil {
		if proxy.IsNoPricingError(billingErr) {
			httputil.WriteError(w, http.StatusForbidden, "Model billing not configured", "billing_error", "no_pricing")
		} else {
			httputil.WriteError(w, http.StatusPaymentRequired, billingErr.Error(), "billing_error", "insufficient_balance")
		}
		return
	}

	// 8. 同协议优先：声明了 responses 协议的上游走透传。
	//    若该批上游全部失败（5xx 或网络错误），fall through 到 OpenAI Chat 兜底。
	responsesUpstreams := filterResponsesUpstreams(upstreams)
	if len(responsesUpstreams) > 0 {
		result, err := h.core.Failover(r.Context(), responsesUpstreams, bodyBytes, "POST", "/v1/responses", nil)
		if err == nil {
			// 4xx 视为客户端错误透传（不 fallback）；5xx 走兜底
			if result.Response.StatusCode < 500 {
				defer result.Cancel()
				if result.Response.StatusCode >= 400 {
					body, _ := io.ReadAll(result.Response.Body)
					result.Response.Body.Close()
					ct := result.Response.Header.Get("Content-Type")
					if ct == "" {
						ct = "application/json"
					}
					w.Header().Set("Content-Type", ct)
					w.WriteHeader(result.Response.StatusCode)
					w.Write(body)
					return
				}

				slog.Info("responses api passthrough routed",
					"model", canonicalName,
					"upstream", result.Upstream.Config.Provider,
					"base_url", result.Upstream.Config.BaseURL,
					"status", result.Response.StatusCode,
				)

				var usage billing.UsageInfo
				if req.Stream {
					usage = RelayResponsesPassthroughStream(w, result.Response, clientModel)
				} else {
					usage = h.relayResponsesPassthroughNonStream(w, result.Response)
				}
				h.core.AsyncChargeAuth(auth, canonicalName, requestID, usage)
				return
			}
			// 5xx：放弃这个 result，落到 OpenAI Chat 兜底
			result.Cancel()
		} else {
			slog.Warn("responses upstream failover failed, falling back to OpenAI Chat",
				"model", canonicalName,
				"upstream_count", len(responsesUpstreams),
				"error", err,
			)
		}
	} else {
		slog.Info("no responses upstream for /v1/responses, trying OpenAI Chat fallback",
			"model", canonicalName,
			"total_upstreams", len(upstreams),
		)
	}

	// 9. 兜底：声明了 openai/openai-compatible 协议的上游，做协议转换。
	chatUpstreams := filterOpenAIChatUpstreams(upstreams)
	if len(chatUpstreams) == 0 {
		httputil.WriteError(w, http.StatusServiceUnavailable,
			"No compatible upstream configured for this model",
			"server_error", "no_compatible_upstream")
		return
	}

	// 转换请求：Responses CreateRequest → OpenAI Chat Completions map
	chatBody, err := ConvertRequest(&req)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "conversion_error")
		return
	}
	if req.Stream {
		chatBody["stream"] = true
		chatBody["stream_options"] = map[string]any{"include_usage": true}
	}
	if chatUpstreams[0].Config.ModelOverride != "" {
		chatBody["model"] = chatUpstreams[0].Config.ModelOverride
	}
	chatBodyBytes, err := json.Marshal(chatBody)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to encode request", "server_error", "encoding_error")
		return
	}

	result, err := h.core.Failover(r.Context(), chatUpstreams, chatBodyBytes, "POST", "/v1/chat/completions", nil)
	if err != nil {
		httputil.WriteError(w, http.StatusServiceUnavailable,
			"All upstream providers are unavailable",
			"server_error", "upstream_unavailable")
		return
	}
	defer result.Cancel()

	if result.Response.StatusCode >= 400 {
		body, _ := io.ReadAll(result.Response.Body)
		result.Response.Body.Close()
		ct := result.Response.Header.Get("Content-Type")
		if ct == "" {
			ct = "application/json"
		}
		w.Header().Set("Content-Type", ct)
		w.WriteHeader(result.Response.StatusCode)
		w.Write(body)
		return
	}

	slog.Info("responses api request routed to openai chat upstream",
		"model", canonicalName,
		"upstream", result.Upstream.Config.Provider,
		"base_url", result.Upstream.Config.BaseURL,
		"status", result.Response.StatusCode,
	)

	// 转换响应：Chat Completions → Responses
	skipReasoning := isLegacyCursor(r.UserAgent())
	opts := relayOptions{
		emitReasoning:    len(req.Reasoning) > 0 && !skipReasoning,
		encryptedContent: includesEncryptedReasoning(req.Include),
		reasoningRaw:     req.Reasoning,
	}
	if skipReasoning && len(req.Reasoning) > 0 {
		slog.Warn("legacy Cursor detected, disabling reasoning output",
			"user_agent", r.UserAgent(),
			"model", clientModel)
	}

	var usage billing.UsageInfo
	if req.Stream {
		usage = RelayStream(w, result.Response, clientModel, opts)
	} else {
		usage = h.relayNonStream(w, result.Response, clientModel)
	}
	h.core.AsyncChargeAuth(auth, canonicalName, requestID, usage)
}

// ServeGetHTTP handles GET /v1/responses/{id} — background response polling.
// We don't persist responses, so return 404 in proper JSON format rather than HTML.
func (h *Handler) ServeGetHTTP(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v1/responses/")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, `{"error":{"message":"Response %q not found","type":"invalid_request_error","code":"response_not_found"}}`, id)
}

// includesEncryptedReasoning reports whether the client requested
// reasoning.encrypted_content via the response.include parameter.
func includesEncryptedReasoning(include []string) bool {
	for _, v := range include {
		if v == "reasoning.encrypted_content" {
			return true
		}
	}
	return false
}

// filterResponsesUpstreams returns upstreams that natively support the
// Responses API (declared via the "responses" protocol).
func filterResponsesUpstreams(upstreams []balancer.Upstream) []balancer.Upstream {
	var out []balancer.Upstream
	for _, u := range upstreams {
		if proxy.IsResponsesAPI(u.Config) {
			out = append(out, u)
		}
	}
	return out
}

// filterOpenAIChatUpstreams returns upstreams that declare an OpenAI Chat
// Completions-compatible protocol (openai or openai-compatible). Used by the
// /v1/responses entry to pick the fallback upstream set when no Responses
// upstream is available (or all Responses upstreams failed).
func filterOpenAIChatUpstreams(upstreams []balancer.Upstream) []balancer.Upstream {
	var out []balancer.Upstream
	for _, u := range upstreams {
		if proxy.IsOpenAIChatCompatible(u.Config) {
			out = append(out, u)
		}
	}
	return out
}

// relayResponsesPassthroughNonStream pipes the upstream Responses API JSON
// response directly to the client and extracts usage for billing.
func (h *Handler) relayResponsesPassthroughNonStream(w http.ResponseWriter, resp *http.Response) billing.UsageInfo {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		httputil.WriteError(w, http.StatusBadGateway, "Failed to read upstream response", "server_error", "upstream_read_error")
		return billing.UsageInfo{}
	}
	usage := extractResponsesAPIUsage(body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
	return usage
}

// relayNonStream reads an OpenAI Chat Completions upstream response, converts
// it to Responses API format, and writes it to the client. Used by the
// /v1/responses entry when falling back to an OpenAI Chat-compatible upstream
// (no Responses-protocol upstream available or all of them failed).
func (h *Handler) relayNonStream(w http.ResponseWriter, resp *http.Response, canonicalModel string) billing.UsageInfo {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		httputil.WriteError(w, http.StatusBadGateway, "Failed to read upstream response", "server_error", "upstream_read_error")
		return billing.UsageInfo{}
	}

	// Extract usage from the raw Chat Completions response
	usage := proxy.ExtractUsageFromJSON(body)

	// Convert to Responses API format
	responsesResp, err := ConvertResponse(body, canonicalModel)
	if err != nil {
		httputil.WriteError(w, http.StatusBadGateway, "Failed to convert upstream response", "server_error", "conversion_error")
		return usage
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(responsesResp)

	return usage
}

// findDuplicateFunctionCallID scans Responses API input for function_call /
// custom_tool_call items that share a call_id. Returns the first duplicate
// call_id encountered, or "" if all call_ids are unique (or input is not an
// array / has no function_call items). The Responses API requires each
// function_call to have a unique call_id within a request's input history;
// upstreams reject duplicates with 400, so we fail fast at the gateway
// instead of forwarding an invalid request.
func findDuplicateFunctionCallID(rawInput json.RawMessage) string {
	if len(rawInput) == 0 || rawInput[0] != '[' {
		return ""
	}
	var items []InputItem
	if err := json.Unmarshal(rawInput, &items); err != nil {
		return ""
	}
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item.Type != "function_call" && item.Type != "custom_tool_call" {
			continue
		}
		if item.CallID == "" {
			continue
		}
		if _, ok := seen[item.CallID]; ok {
			return item.CallID
		}
		seen[item.CallID] = struct{}{}
	}
	return ""
}
