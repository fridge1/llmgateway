package anthropic

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ConvertRequest converts an Anthropic MessagesRequest to an OpenAI Chat Completions request body.
func ConvertRequest(req *MessagesRequest) (map[string]any, error) {
	openAI := map[string]any{
		"model":      req.Model,
		"max_tokens": req.MaxTokens,
	}

	var messages []map[string]any

	// Handle system message
	if len(req.System) > 0 {
		sysMsg, err := convertSystemMessage(req.System)
		if err != nil {
			return nil, fmt.Errorf("invalid system field: %w", err)
		}
		if sysMsg != "" {
			messages = append(messages, map[string]any{
				"role":    "system",
				"content": sysMsg,
			})
		}
	}

	// Convert messages
	for _, msg := range req.Messages {
		converted, err := convertMessage(msg)
		if err != nil {
			return nil, fmt.Errorf("invalid message: %w", err)
		}
		messages = append(messages, converted...)
	}

	openAI["messages"] = messages

	if req.Stream {
		openAI["stream"] = true
		// Request usage info in streaming response for accurate billing
		openAI["stream_options"] = map[string]any{
			"include_usage": true,
		}
	}

	if req.Temperature != nil {
		openAI["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		openAI["top_p"] = *req.TopP
	}
	if len(req.StopSequences) > 0 {
		openAI["stop"] = req.StopSequences
	}

	// Convert tools
	if len(req.Tools) > 0 {
		oaiTools := make([]map[string]any, len(req.Tools))
		for i, t := range req.Tools {
			fn := map[string]any{"name": t.Name}
			if t.Description != "" {
				fn["description"] = t.Description
			}
			if len(t.InputSchema) > 0 {
				var schema any
				if err := json.Unmarshal(t.InputSchema, &schema); err != nil {
					return nil, fmt.Errorf("invalid tool input_schema: %w", err)
				}
				fn["parameters"] = schema
			}
			oaiTools[i] = map[string]any{
				"type":     "function",
				"function": fn,
			}
		}
		openAI["tools"] = oaiTools
	}

	// Convert tool_choice
	if len(req.ToolChoice) > 0 {
		tc, err := convertToolChoice(req.ToolChoice)
		if err != nil {
			return nil, fmt.Errorf("invalid tool_choice: %w", err)
		}
		if tc != nil {
			openAI["tool_choice"] = tc
		}
	}

	return openAI, nil
}

// convertSystemMessage handles system as either a plain string or []ContentBlock.
func convertSystemMessage(raw json.RawMessage) (string, error) {
	// Try string first
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}

	// Try array of content blocks
	var blocks []ContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return "", fmt.Errorf("system must be a string or array of content blocks")
	}

	var parts []string
	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, "\n"), nil
}

// convertMessage converts a single Anthropic message to one or more OpenAI messages.
func convertMessage(msg Message) ([]map[string]any, error) {
	// Try string content first
	var strContent string
	if err := json.Unmarshal(msg.Content, &strContent); err == nil {
		return []map[string]any{
			{"role": msg.Role, "content": strContent},
		}, nil
	}

	// Parse as array of content blocks
	var blocks []ContentBlock
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return nil, fmt.Errorf("content must be a string or array of content blocks")
	}

	if msg.Role == "assistant" {
		return convertAssistantBlocks(blocks)
	}

	// user role: split tool_result blocks into separate tool messages
	return convertUserBlocks(blocks)
}

// convertAssistantBlocks converts assistant content blocks to an OpenAI assistant message.
func convertAssistantBlocks(blocks []ContentBlock) ([]map[string]any, error) {
	var textContent string
	var toolCalls []map[string]any

	for _, b := range blocks {
		switch b.Type {
		case "text":
			textContent = b.Text
		case "tool_use":
			argsStr := "{}"
			if len(b.Input) > 0 {
				argsStr = string(b.Input)
			}
			toolCalls = append(toolCalls, map[string]any{
				"id":   b.ID,
				"type": "function",
				"function": map[string]any{
					"name":      b.Name,
					"arguments": argsStr,
				},
			})
		}
	}

	msg := map[string]any{"role": "assistant"}
	if textContent != "" {
		msg["content"] = textContent
	}
	if len(toolCalls) > 0 {
		msg["tool_calls"] = toolCalls
	}

	return []map[string]any{msg}, nil
}

// convertUserBlocks converts user content blocks, splitting tool_result into separate tool messages.
func convertUserBlocks(blocks []ContentBlock) ([]map[string]any, error) {
	var msgs []map[string]any
	var contentParts []map[string]any

	for _, b := range blocks {
		switch b.Type {
		case "text":
			contentParts = append(contentParts, map[string]any{
				"type": "text",
				"text": b.Text,
			})
		case "image":
			if b.Source != nil {
				contentParts = append(contentParts, map[string]any{
					"type": "image_url",
					"image_url": map[string]string{
						"url": fmt.Sprintf("data:%s;base64,%s", b.Source.MediaType, b.Source.Data),
					},
				})
			}
		case "tool_result":
			// Flush any pending content parts as a user message first
			if len(contentParts) > 0 {
				msgs = append(msgs, map[string]any{
					"role":    "user",
					"content": contentParts,
				})
				contentParts = nil
			}
			toolContent := extractToolResultContent(b.Content)
			msgs = append(msgs, map[string]any{
				"role":         "tool",
				"tool_call_id": b.ToolUseID,
				"content":      toolContent,
			})
		}
	}

	// Flush remaining content parts
	if len(contentParts) > 0 {
		if len(contentParts) == 1 && contentParts[0]["type"] == "text" {
			msgs = append(msgs, map[string]any{
				"role":    "user",
				"content": contentParts[0]["text"],
			})
		} else {
			msgs = append(msgs, map[string]any{
				"role":    "user",
				"content": contentParts,
			})
		}
	}

	return msgs, nil
}

// extractToolResultContent gets a string from tool_result content (string or content blocks).
func extractToolResultContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var blocks []ContentBlock
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var parts []string
		for _, b := range blocks {
			if b.Type == "text" {
				parts = append(parts, b.Text)
			}
		}
		return strings.Join(parts, "\n")
	}
	return string(raw)
}

// convertToolChoice converts Anthropic tool_choice to OpenAI format.
func convertToolChoice(raw json.RawMessage) (any, error) {
	// Try string: "auto", "any", "none"
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		switch s {
		case "auto":
			return "auto", nil
		case "any":
			return "required", nil
		case "none":
			return "none", nil
		default:
			return s, nil
		}
	}

	// Try object: {"type":"tool","name":"X"} or {"type":"auto"} or {"type":"any"}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("tool_choice must be a string or object")
	}

	tcType, _ := obj["type"].(string)
	switch tcType {
	case "auto":
		return "auto", nil
	case "any":
		return "required", nil
	case "tool":
		name, _ := obj["name"].(string)
		return map[string]any{
			"type":     "function",
			"function": map[string]any{"name": name},
		}, nil
	default:
		return "auto", nil
	}
}

// ConvertResponse converts an OpenAI Chat Completions response to an Anthropic MessagesResponse.
func ConvertResponse(openAIBody []byte, canonicalModel string) (*MessagesResponse, error) {
	var oai map[string]any
	if err := json.Unmarshal(openAIBody, &oai); err != nil {
		return nil, fmt.Errorf("invalid OpenAI response: %w", err)
	}

	resp := &MessagesResponse{
		ID:    fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Type:  "message",
		Role:  "assistant",
		Model: canonicalModel,
	}

	// Extract choices
	choices, _ := oai["choices"].([]any)
	if len(choices) > 0 {
		choice, _ := choices[0].(map[string]any)
		if choice != nil {
			// finish_reason
			finishReason, _ := choice["finish_reason"].(string)
			resp.StopReason = mapFinishReason(finishReason)

			// message content and tool_calls
			message, _ := choice["message"].(map[string]any)
			if message != nil {
				resp.Content = extractResponseContent(message)
			}
		}
	}

	if resp.Content == nil {
		resp.Content = []ResponseBlock{}
	}

	// Extract usage
	if usageObj, ok := oai["usage"].(map[string]any); ok {
		if pt, ok := usageObj["prompt_tokens"].(float64); ok {
			resp.Usage.InputTokens = int(pt)
		}
		if ct, ok := usageObj["completion_tokens"].(float64); ok {
			resp.Usage.OutputTokens = int(ct)
		}
		if details, ok := usageObj["prompt_tokens_details"].(map[string]any); ok {
			if cached, ok := details["cached_tokens"].(float64); ok {
				resp.Usage.CacheReadInputTokens = int(cached)
			}
		}
	}

	return resp, nil
}

// extractResponseContent builds ResponseBlocks from an OpenAI message object.
func extractResponseContent(message map[string]any) []ResponseBlock {
	var blocks []ResponseBlock

	// Text content
	if content, ok := message["content"].(string); ok && content != "" {
		blocks = append(blocks, ResponseBlock{
			Type: "text",
			Text: content,
		})
	}

	// Tool calls
	if toolCalls, ok := message["tool_calls"].([]any); ok {
		for _, tc := range toolCalls {
			tcMap, _ := tc.(map[string]any)
			if tcMap == nil {
				continue
			}
			block := ResponseBlock{Type: "tool_use"}
			block.ID, _ = tcMap["id"].(string)
			if fn, ok := tcMap["function"].(map[string]any); ok {
				block.Name, _ = fn["name"].(string)
				if args, ok := fn["arguments"].(string); ok {
					block.Input = json.RawMessage(args)
				}
			}
			blocks = append(blocks, block)
		}
	}

	return blocks
}

// mapFinishReason maps OpenAI finish_reason to Anthropic stop_reason.
func mapFinishReason(reason string) string {
	switch reason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	case "tool_calls":
		return "tool_use"
	default:
		return "end_turn"
	}
}
