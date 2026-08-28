package responses

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"
)

// IsResponsesShapedBody reports whether a JSON body uses the OpenAI Responses
// API wire format (top-level "input" without "messages"). Cursor Agent sends
// this shape to /v1/chat/completions and expects Responses-shaped responses back.
func IsResponsesShapedBody(body []byte) bool {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return false
	}
	_, hasInput := m["input"]
	_, hasMessages := m["messages"]
	return hasInput && !hasMessages
}

// ConvertRequest converts a Responses API CreateRequest into an OpenAI Chat Completions request body.
func ConvertRequest(req *CreateRequest) (map[string]any, error) {
	out := map[string]any{
		"model": req.Model,
	}

	var messages []map[string]any

	// Instructions → system message
	if req.Instructions != nil && *req.Instructions != "" {
		messages = append(messages, map[string]any{
			"role":    "system",
			"content": *req.Instructions,
		})
	}

	// Parse Input — can be a string or []InputItem
	if len(req.Input) > 0 {
		inputMsgs, err := ConvertInput(req.Input)
		if err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		messages = append(messages, inputMsgs...)
	}

	out["messages"] = messages

	if req.Temperature != nil {
		out["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		out["top_p"] = *req.TopP
	}
	if req.MaxOutputTokens != nil {
		out["max_tokens"] = *req.MaxOutputTokens
	}
	if req.FrequencyPenalty != nil {
		out["frequency_penalty"] = *req.FrequencyPenalty
	}
	if req.PresencePenalty != nil {
		out["presence_penalty"] = *req.PresencePenalty
	}
	if req.User != "" {
		out["user"] = req.User
	}
	// NOTE: background is a Responses API gateway-level parameter, not forwarded to upstream
	if req.Stream {
		out["stream"] = true
	}

	// Tools — from req.Tools and from additional_tools items in input. Both
	// sources feed through the same converter so flat Responses-API function
	// tools ({"type":"function","name":"shell","parameters":{...}}) are
	// rewritten to the Chat Completions nested shape
	// ({"type":"function","function":{"name":"shell","parameters":{...}}})
	// regardless of which path delivered them.
	var tools []map[string]any
	for _, t := range req.Tools {
		tools = append(tools, convertToolToChatCompletions(t)...)
	}
	for _, raw := range extractAdditionalTools(req.Input) {
		var t Tool
		if err := json.Unmarshal(raw, &t); err != nil {
			continue
		}
		tools = append(tools, convertToolToChatCompletions(t)...)
	}
	if len(tools) > 0 {
		out["tools"] = tools
		// tool_choice is only valid alongside tools. Forwarding it when all
		// tools got dropped (e.g. only Responses-API built-ins like
		// web_search/local_shell were sent) makes the upstream reject the
		// request with "tool_choice is only allowed when tools are specified".
		if len(req.ToolChoice) > 0 {
			var tc any
			if err := json.Unmarshal(req.ToolChoice, &tc); err == nil {
				out["tool_choice"] = tc
			}
		}
	}

	return out, nil
}

// convertToolToChatCompletions converts a single tool definition from
// Responses-API shape into one or more Chat Completions tool definitions.
//
// Returns a slice (not a single map) because some Responses-API tool types
// expand to multiple Chat Completions tools:
//
//   - "function" (flat or nested) → 1 function tool
//   - "namespace" → N function tools (the nested tools[] flattened, each
//     prefixed with "<namespace>__" to avoid name collisions)
//   - other types (e.g. "web_search", "custom") → dropped. Chat Completions
//     only supports function tools; emitting these makes the upstream try
//     to parse them as functions and fail with "tools[N].function: missing
//     field name".
//
// Returns nil for entries that should be dropped (e.g. a function tool whose
// name is missing or empty — passing it through would make upstream reject
// the whole request with "tools[N].function: missing field name").
func convertToolToChatCompletions(t Tool) []map[string]any {
	switch t.Type {
	case "function":
		fn := buildFunctionMap(t)
		if name, ok := fn["name"].(string); !ok || name == "" {
			return nil
		}
		return []map[string]any{{
			"type":     "function",
			"function": fn,
		}}
	case "namespace":
		// Codex Desktop groups sub-tools under type=namespace. Chat
		// Completions has no namespace concept, so flatten: each nested
		// function tool becomes a top-level function, prefixed with
		// "<namespace>__<tool_name>" to avoid name collisions across
		// namespaces.
		nsName, _ := t.Extra["name"].(string)
		rawTools, ok := t.Extra["tools"]
		if !ok {
			return nil
		}
		rawToolsBytes, err := json.Marshal(rawTools)
		if err != nil {
			return nil
		}
		var nested []Tool
		if err := json.Unmarshal(rawToolsBytes, &nested); err != nil {
			return nil
		}
		var out []map[string]any
		for _, nt := range nested {
			ntFn := buildFunctionMap(nt)
			name, ok := ntFn["name"].(string)
			if !ok || name == "" {
				continue
			}
			if nsName != "" {
				ntFn["name"] = nsName + "__" + name
			}
			out = append(out, map[string]any{
				"type":     "function",
				"function": ntFn,
			})
		}
		return out
	case "":
		return nil
	default:
		// Non-function, non-namespace tool types (e.g. "web_search",
		// "local_shell", "custom") are Responses-API-specific built-ins
		// that Chat Completions upstreams reject — passing them through
		// makes the upstream try to deserialize them as function tools
		// and fail with "tools[N].function: missing field name". Drop
		// them rather than emit a malformed tool entry.
		return nil
	}
}

// buildFunctionMap extracts name/description/parameters/strict from a
// flat (Responses-API) or nested (Chat Completions) function tool into the
// "function" object expected by the Chat Completions tools array.
func buildFunctionMap(t Tool) map[string]any {
	fn := map[string]any{}
	if t.Function != nil {
		fn["name"] = t.Function.Name
		if t.Function.Description != "" {
			fn["description"] = t.Function.Description
		}
		if len(t.Function.Parameters) > 0 {
			var params any
			if err := json.Unmarshal(t.Function.Parameters, &params); err == nil {
				fn["parameters"] = params
			}
		}
	} else {
		if name, ok := t.Extra["name"].(string); ok {
			fn["name"] = name
		}
		if desc, ok := t.Extra["description"].(string); ok && desc != "" {
			fn["description"] = desc
		}
		if params, ok := t.Extra["parameters"]; ok {
			fn["parameters"] = params
		}
	}
	// "strict" is an OpenAI Responses-API function-tool flag that Chat
	// Completions also accepts at the function level. Forward it when
	// present so strict-mode tool definitions keep their semantics.
	if strict, ok := t.Extra["strict"]; ok {
		fn["strict"] = strict
	}
	return fn
}

// ConvertInput parses the Input field which can be a plain string or an array of InputItems.
// Returns a slice of Chat Completions message maps.
func ConvertInput(raw json.RawMessage) ([]map[string]any, error) {
	// Try string first
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return []map[string]any{
			{"role": "user", "content": s},
		}, nil
	}

	// Try array of raw items. Decoding per-item (rather than directly into
	// []InputItem) keeps one malformed/unexpected item from failing the whole
	// request, which is how Codex desktop image uploads previously triggered
	// "input must be a string or array of input items".
	var rawItems []json.RawMessage
	if err := json.Unmarshal(raw, &rawItems); err != nil {
		slog.Warn("responses: input is neither string nor array",
			"error", err, "input_head", truncateForLog(raw))
		return nil, fmt.Errorf("input must be a string or array of input items")
	}

	var msgs []map[string]any
	for _, ri := range rawItems {
		var item InputItem
		if err := json.Unmarshal(ri, &item); err != nil {
			slog.Warn("responses: skipping unparseable input item",
				"error", err, "item_head", truncateForLog(ri))
			continue
		}
		msg, err := convertInputItem(item)
		if err != nil {
			slog.Warn("responses: skipping unconvertible input item",
				"error", err, "type", item.Type, "item_head", truncateForLog(ri))
			continue
		}
		if msg == nil {
			continue
		}
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

// truncateForLog returns a short, log-safe prefix of raw JSON. Image inputs can
// carry very long base64 data URLs, so cap the length to avoid log bloat and
// leaking full payloads.
func truncateForLog(raw json.RawMessage) string {
	const max = 256
	if len(raw) <= max {
		return string(raw)
	}
	return string(raw[:max]) + "...(truncated)"
}

// convertInputItem converts a single InputItem to a Chat Completions message.
func convertInputItem(item InputItem) (map[string]any, error) {
	// Treat items without a type field but with a role as "message" (Codex / Responses API shorthand).
	itemType := item.Type
	if itemType == "" && item.Role != "" {
		itemType = "message"
	}

	switch itemType {
	case "message":
		msg := map[string]any{
			"role": item.Role,
		}
		content, err := convertContent(item.Content)
		if err != nil {
			return nil, err
		}
		msg["content"] = content
		return msg, nil

	// "custom_tool_call_output" / "custom_tool_call" are Codex CLI's custom-tool
	// variants: same shape as function_call_output / function_call (call_id +
	// output / arguments), and they flatten to the same Chat Completions roles.
	case "function_call_output", "custom_tool_call_output":
		return map[string]any{
			"role":         "tool",
			"tool_call_id": item.CallID,
			"content":      convertOutput(item.Output),
		}, nil

	case "function_call", "custom_tool_call":
		return map[string]any{
			"role":    "assistant",
			"content": nil,
			"tool_calls": []map[string]any{{
				"id":   item.CallID,
				"type": "function",
				"function": map[string]any{
					"name":      item.Name,
					"arguments": item.Arguments,
				},
			}},
		}, nil

	case "reasoning", "additional_tools":
		return nil, nil

	default:
		return nil, fmt.Errorf("unsupported input item type: %s", item.Type)
	}
}

// extractAdditionalTools scans the raw input array for additional_tools items and
// returns their tool definitions to be merged into the upstream tools array.
func extractAdditionalTools(rawInput json.RawMessage) []json.RawMessage {
	if len(rawInput) == 0 || rawInput[0] != '[' {
		return nil
	}
	var rawItems []json.RawMessage
	if err := json.Unmarshal(rawInput, &rawItems); err != nil {
		return nil
	}
	var result []json.RawMessage
	for _, ri := range rawItems {
		var item InputItem
		if err := json.Unmarshal(ri, &item); err != nil || item.Type != "additional_tools" {
			continue
		}
		if len(item.Tools) == 0 {
			continue
		}
		var tools []json.RawMessage
		if err := json.Unmarshal(item.Tools, &tools); err == nil {
			result = append(result, tools...)
		}
	}
	return result
}

// convertContent converts the Content field of an InputItem.
// It can be a plain string or an array of ContentParts.
func convertContent(raw json.RawMessage) (any, error) {
	if len(raw) == 0 {
		return "", nil
	}

	// Try string
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}

	// Try array of ContentPart
	var parts []ContentPart
	if err := json.Unmarshal(raw, &parts); err != nil {
		return nil, fmt.Errorf("content must be a string or array of content parts")
	}

	var result []map[string]any
	for _, p := range parts {
		switch p.Type {
		case "input_text", "output_text":
			result = append(result, map[string]any{
				"type": "text",
				"text": p.Text,
			})
		case "input_image":
			result = append(result, map[string]any{
				"type":      "image_url",
				"image_url": map[string]string{"url": string(p.ImageURL)},
			})
		case "input_audio":
			// Pass through as-is; not commonly used
			result = append(result, map[string]any{
				"type": "input_audio",
			})
		default:
			return nil, fmt.Errorf("unsupported content part type: %s", p.Type)
		}
	}
	return result, nil
}

// convertOutput converts a function_call_output's "output" field into a tool
// message content. The Responses API allows it to be a plain string or an array
// of content blocks (text and/or image results); Chat Completions tool messages
// take a string, so array forms are flattened to their concatenated text.
func convertOutput(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}

	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
	}
	if err := json.Unmarshal(raw, &parts); err == nil {
		var b strings.Builder
		for _, p := range parts {
			if p.Text != "" {
				if b.Len() > 0 {
					b.WriteByte('\n')
				}
				b.WriteString(p.Text)
			}
		}
		return b.String()
	}

	// Unknown shape: pass through the raw JSON so the upstream still sees something.
	return string(raw)
}
func ConvertResponse(openAIBody []byte, canonicalModel string) (*Response, error) {
	var chatResp struct {
		Choices []struct {
			Message struct {
				Role      string `json:"role"`
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(openAIBody, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse chat completions response: %w", err)
	}

	respObj := newResponse(generateResponseID(), canonicalModel, time.Now().Unix(), "completed", nil)
	resp := &respObj

	if len(chatResp.Choices) > 0 {
		choice := chatResp.Choices[0]

		// Text content as message output
		if choice.Message.Content != "" {
			resp.Output = append(resp.Output, messageItem(generateItemID(), "completed", []OutputContent{textPart(choice.Message.Content)}))
		}

		// Tool calls as function_call output items
		for _, tc := range choice.Message.ToolCalls {
			resp.Output = append(resp.Output, OutputItem{
				Type:      "function_call",
				ID:        generateFuncCallID(),
				CallID:    tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
				Status:    "completed",
			})
		}
	}

	usage := newResponseUsage(chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens)
	resp.Usage = &usage

	return resp, nil
}

var idCounter atomic.Int64

// generateResponseID creates a unique response ID with resp_ prefix.
func generateResponseID() string {
	return fmt.Sprintf("resp_%d_%d", time.Now().UnixNano(), idCounter.Add(1))
}

// generateItemID creates a unique item ID.
func generateItemID() string {
	return fmt.Sprintf("item_%d_%d", time.Now().UnixNano(), idCounter.Add(1))
}

// generateFuncCallID creates a unique function_call item ID. The Responses API
// uses an "fc_" prefix for function_call items; clients (e.g. Cursor Agent) key
// off item_id to associate argument deltas with the call.
func generateFuncCallID() string {
	return fmt.Sprintf("fc_%d_%d", time.Now().UnixNano(), idCounter.Add(1))
}

// generateReasoningID creates a unique reasoning item ID with the "rs_" prefix
// used by OpenAI's Responses API for reasoning output items.
func generateReasoningID() string {
	return fmt.Sprintf("rs_%d_%d", time.Now().UnixNano(), idCounter.Add(1))
}
