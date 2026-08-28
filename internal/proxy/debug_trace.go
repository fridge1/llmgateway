package proxy

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
)

// debug_trace.go — TEMPORARY diagnostic tracing for /v1/chat/completions.
//
// Enabled by setting env var GW_DEBUG_TRACE=1 (or true/yes/on). When off,
// every helper here is a near-zero-cost no-op.
//
// Three trace points feed `slog.Info` with prefix "trace.cursor":
//   1. inbound  — raw request body + selected headers from the client
//   2. upstream — the body actually sent to the Anthropic upstream (after
//                 OpenAI→Anthropic conversion if applicable)
//   3. sse      — every Anthropic SSE event seen and every OpenAI chunk emitted
//
// Each trace line carries `req_id` so the three streams can be stitched back
// together via `grep "req_id=..."`. Bodies are truncated to 8 KiB to bound
// log volume; raise `traceMaxBodyBytes` if you need more.
//
// Remove this whole file (and its call sites in handler.go) once the Cursor
// Agent issue is diagnosed.

var (
	traceEnabled       atomic.Bool
	traceMaxBodyBytes  = 256 * 1024
	traceModelKeywords = []string{"claude", "opus", "sonnet", "haiku", "gw/"}
)

func init() {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("GW_DEBUG_TRACE")))
	switch v {
	case "1", "true", "yes", "on":
		traceEnabled.Store(true)
		slog.Warn("GW_DEBUG_TRACE enabled — verbose request/response tracing is ON")
	}
}

func traceOn() bool { return traceEnabled.Load() }

// traceShouldLogModel decides whether a model name is interesting enough to
// dump. We only want Claude-family traffic so OpenAI/Gemini calls don't flood
// the log. Empty model passes (we may not have parsed it yet).
func traceShouldLogModel(model string) bool {
	if model == "" {
		return true
	}
	m := strings.ToLower(model)
	for _, kw := range traceModelKeywords {
		if strings.Contains(m, kw) {
			return true
		}
	}
	return false
}

func traceClipBody(body []byte) string {
	if len(body) <= traceMaxBodyBytes {
		return string(body)
	}
	return string(body[:traceMaxBodyBytes]) + "...[truncated " +
		intStr(len(body)-traceMaxBodyBytes) + " bytes]"
}

func intStr(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// traceInbound logs the original client request (headers + body) and a
// structured summary that surfaces the fields most useful for diagnosing
// tool-calling protocol issues (tools array, tool_choice, message roles).
func traceInbound(reqID, model string, r *http.Request, body []byte) {
	if !traceOn() || !traceShouldLogModel(model) {
		return
	}
	summary := summarizeChatRequest(body)
	slog.Info("trace.cursor inbound",
		"req_id", reqID,
		"model", model,
		"path", r.URL.Path,
		"ua", r.Header.Get("User-Agent"),
		"x_cursor_client", r.Header.Get("X-Cursor-Client"),
		"x_client", r.Header.Get("X-Client"),
		"x_stainless_pkg", r.Header.Get("X-Stainless-Package-Version"),
		"accept", r.Header.Get("Accept"),
		"content_type", r.Header.Get("Content-Type"),
		"summary", summary,
		"body", traceClipBody(body),
	)
}

// summarizeChatRequest extracts the tool-calling-relevant shape of an OpenAI
// Chat Completions request body. Returns a small map that's cheap to print.
func summarizeChatRequest(body []byte) map[string]any {
	out := map[string]any{}
	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		out["parse_error"] = err.Error()
		return out
	}
	out["top_keys"] = sortedKeys(req)
	if v, ok := req["stream"].(bool); ok {
		out["stream"] = v
	}
	if v, ok := req["tool_choice"]; ok {
		out["tool_choice"] = v
	}
	if v, ok := req["parallel_tool_calls"]; ok {
		out["parallel_tool_calls"] = v
	}
	if tools, ok := req["tools"].([]any); ok {
		out["tools_count"] = len(tools)
		var names []string
		for _, t := range tools {
			tm, _ := t.(map[string]any)
			if fn, ok := tm["function"].(map[string]any); ok {
				if name, _ := fn["name"].(string); name != "" {
					names = append(names, name)
				}
			}
		}
		out["tools_names"] = names
	} else {
		out["tools_count"] = 0
	}
	if msgs, ok := req["messages"].([]any); ok {
		out["messages_count"] = len(msgs)
		var roles []string
		var sysLen, lastUserLen int
		for i, m := range msgs {
			mm, _ := m.(map[string]any)
			role, _ := mm["role"].(string)
			roles = append(roles, role)
			contentStr := ""
			switch c := mm["content"].(type) {
			case string:
				contentStr = c
			case []any:
				for _, p := range c {
					pm, _ := p.(map[string]any)
					if t, _ := pm["text"].(string); t != "" {
						contentStr += t
					}
				}
			}
			if role == "system" {
				sysLen += len(contentStr)
			}
			if i == len(msgs)-1 && role == "user" {
				lastUserLen = len(contentStr)
			}
		}
		out["messages_roles"] = roles
		out["messages_system_chars"] = sysLen
		out["messages_last_user_chars"] = lastUserLen
	}
	return out
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// not actually sorted — order doesn't matter for diagnostic output
	return keys
}

// traceUpstreamBody logs the body actually sent to the upstream (post
// OpenAI→Anthropic conversion if any).
func traceUpstreamBody(reqID, model, upstreamURL string, isAnthropic bool, body []byte) {
	if !traceOn() || !traceShouldLogModel(model) {
		return
	}
	slog.Info("trace.cursor upstream-request",
		"req_id", reqID,
		"model", model,
		"upstream_url", upstreamURL,
		"is_anthropic_protocol", isAnthropic,
		"body", traceClipBody(body),
	)
}

// traceUpstreamRaw logs a single Anthropic SSE event as observed from the upstream.
func traceUpstreamRaw(reqID, model, eventType, data string) {
	if !traceOn() || !traceShouldLogModel(model) {
		return
	}
	clipped := data
	if len(clipped) > traceMaxBodyBytes {
		clipped = clipped[:traceMaxBodyBytes] + "...[truncated]"
	}
	slog.Info("trace.cursor upstream-sse",
		"req_id", reqID,
		"event", eventType,
		"data", clipped,
	)
}

// traceClientChunk logs a single OpenAI SSE chunk just before it is flushed
// to the client.
func traceClientChunk(reqID, model, chunk string) {
	if !traceOn() || !traceShouldLogModel(model) {
		return
	}
	clipped := strings.TrimRight(chunk, "\n")
	if len(clipped) > traceMaxBodyBytes {
		clipped = clipped[:traceMaxBodyBytes] + "...[truncated]"
	}
	slog.Info("trace.cursor client-chunk",
		"req_id", reqID,
		"chunk", clipped,
	)
}

// traceNonStreamResponse logs the full OpenAI body returned to the client in
// non-streaming mode.
func traceNonStreamResponse(reqID, model string, status int, body []byte) {
	if !traceOn() || !traceShouldLogModel(model) {
		return
	}
	slog.Info("trace.cursor client-response",
		"req_id", reqID,
		"status", status,
		"body", traceClipBody(body),
	)
}
