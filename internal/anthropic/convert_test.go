package anthropic

import (
	"encoding/json"
	"testing"
)

func TestConvertRequest_SimpleText(t *testing.T) {
	req := &MessagesRequest{
		Model:     "claude-3-opus",
		MaxTokens: 1024,
		Messages: []Message{
			{Role: "user", Content: json.RawMessage(`"Hello"`)},
		},
	}

	oai, err := ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}

	msgs := oai["messages"].([]map[string]any)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0]["role"] != "user" || msgs[0]["content"] != "Hello" {
		t.Fatalf("unexpected message: %v", msgs[0])
	}
	if oai["max_tokens"] != 1024 {
		t.Fatalf("expected max_tokens 1024, got %v", oai["max_tokens"])
	}
}

func TestConvertRequest_SystemString(t *testing.T) {
	req := &MessagesRequest{
		Model:     "claude-3-opus",
		MaxTokens: 100,
		System:    json.RawMessage(`"You are helpful."`),
		Messages: []Message{
			{Role: "user", Content: json.RawMessage(`"Hi"`)},
		},
	}

	oai, err := ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}

	msgs := oai["messages"].([]map[string]any)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0]["role"] != "system" || msgs[0]["content"] != "You are helpful." {
		t.Fatalf("unexpected system message: %v", msgs[0])
	}
}

func TestConvertRequest_SystemContentBlocks(t *testing.T) {
	req := &MessagesRequest{
		Model:     "claude-3-opus",
		MaxTokens: 100,
		System:    json.RawMessage(`[{"type":"text","text":"Part 1"},{"type":"text","text":"Part 2"}]`),
		Messages: []Message{
			{Role: "user", Content: json.RawMessage(`"Hi"`)},
		},
	}

	oai, err := ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}

	msgs := oai["messages"].([]map[string]any)
	if msgs[0]["content"] != "Part 1\nPart 2" {
		t.Fatalf("unexpected system content: %v", msgs[0]["content"])
	}
}

func TestConvertRequest_Tools(t *testing.T) {
	req := &MessagesRequest{
		Model:     "claude-3-opus",
		MaxTokens: 100,
		Messages: []Message{
			{Role: "user", Content: json.RawMessage(`"What is the weather?"`)},
		},
		Tools: []Tool{
			{
				Name:        "get_weather",
				Description: "Get the weather",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}}}`),
			},
		},
	}

	oai, err := ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}

	tools := oai["tools"].([]map[string]any)
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0]["type"] != "function" {
		t.Fatalf("expected function type, got %v", tools[0]["type"])
	}
	fn := tools[0]["function"].(map[string]any)
	if fn["name"] != "get_weather" {
		t.Fatalf("expected get_weather, got %v", fn["name"])
	}
}

func TestConvertRequest_ToolUseInAssistant(t *testing.T) {
	req := &MessagesRequest{
		Model:     "claude-3-opus",
		MaxTokens: 100,
		Messages: []Message{
			{Role: "user", Content: json.RawMessage(`"What is the weather?"`)},
			{
				Role: "assistant",
				Content: json.RawMessage(`[
					{"type":"text","text":"Let me check."},
					{"type":"tool_use","id":"call_1","name":"get_weather","input":{"location":"NYC"}}
				]`),
			},
		},
	}

	oai, err := ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}

	msgs := oai["messages"].([]map[string]any)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	assistantMsg := msgs[1]
	if assistantMsg["content"] != "Let me check." {
		t.Fatalf("unexpected content: %v", assistantMsg["content"])
	}
	toolCalls := assistantMsg["tool_calls"].([]map[string]any)
	if len(toolCalls) != 1 {
		t.Fatalf("expected 1 tool_call, got %d", len(toolCalls))
	}
	if toolCalls[0]["id"] != "call_1" {
		t.Fatalf("unexpected tool_call id: %v", toolCalls[0]["id"])
	}
}

func TestConvertRequest_ToolResultInUser(t *testing.T) {
	req := &MessagesRequest{
		Model:     "claude-3-opus",
		MaxTokens: 100,
		Messages: []Message{
			{
				Role: "user",
				Content: json.RawMessage(`[
					{"type":"tool_result","tool_use_id":"call_1","content":"72°F and sunny"}
				]`),
			},
		},
	}

	oai, err := ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}

	msgs := oai["messages"].([]map[string]any)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0]["role"] != "tool" {
		t.Fatalf("expected tool role, got %v", msgs[0]["role"])
	}
	if msgs[0]["tool_call_id"] != "call_1" {
		t.Fatalf("unexpected tool_call_id: %v", msgs[0]["tool_call_id"])
	}
	if msgs[0]["content"] != "72°F and sunny" {
		t.Fatalf("unexpected content: %v", msgs[0]["content"])
	}
}

func TestConvertRequest_ToolChoiceAuto(t *testing.T) {
	req := &MessagesRequest{
		Model:      "claude-3-opus",
		MaxTokens:  100,
		Messages:   []Message{{Role: "user", Content: json.RawMessage(`"hi"`)}},
		ToolChoice: json.RawMessage(`"auto"`),
	}

	oai, err := ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	if oai["tool_choice"] != "auto" {
		t.Fatalf("expected auto, got %v", oai["tool_choice"])
	}
}

func TestConvertRequest_ToolChoiceAny(t *testing.T) {
	req := &MessagesRequest{
		Model:      "claude-3-opus",
		MaxTokens:  100,
		Messages:   []Message{{Role: "user", Content: json.RawMessage(`"hi"`)}},
		ToolChoice: json.RawMessage(`"any"`),
	}

	oai, err := ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	if oai["tool_choice"] != "required" {
		t.Fatalf("expected required, got %v", oai["tool_choice"])
	}
}

func TestConvertRequest_ToolChoiceSpecific(t *testing.T) {
	req := &MessagesRequest{
		Model:      "claude-3-opus",
		MaxTokens:  100,
		Messages:   []Message{{Role: "user", Content: json.RawMessage(`"hi"`)}},
		ToolChoice: json.RawMessage(`{"type":"tool","name":"get_weather"}`),
	}

	oai, err := ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	tc := oai["tool_choice"].(map[string]any)
	if tc["type"] != "function" {
		t.Fatalf("expected function type, got %v", tc["type"])
	}
	fn := tc["function"].(map[string]any)
	if fn["name"] != "get_weather" {
		t.Fatalf("expected get_weather, got %v", fn["name"])
	}
}

func TestConvertRequest_ImageContent(t *testing.T) {
	req := &MessagesRequest{
		Model:     "claude-3-opus",
		MaxTokens: 100,
		Messages: []Message{
			{
				Role: "user",
				Content: json.RawMessage(`[
					{"type":"text","text":"What is this?"},
					{"type":"image","source":{"type":"base64","media_type":"image/png","data":"abc123"}}
				]`),
			},
		},
	}

	oai, err := ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}

	msgs := oai["messages"].([]map[string]any)
	content := msgs[0]["content"].([]map[string]any)
	if len(content) != 2 {
		t.Fatalf("expected 2 content parts, got %d", len(content))
	}
	if content[1]["type"] != "image_url" {
		t.Fatalf("expected image_url type, got %v", content[1]["type"])
	}
	imgURL := content[1]["image_url"].(map[string]string)
	if imgURL["url"] != "data:image/png;base64,abc123" {
		t.Fatalf("unexpected image URL: %v", imgURL["url"])
	}
}

func TestConvertResponse_SimpleText(t *testing.T) {
	oaiResp := `{
		"id": "chatcmpl-123",
		"model": "gpt-4",
		"choices": [{"message": {"role": "assistant", "content": "Hello!"}, "finish_reason": "stop"}],
		"usage": {"prompt_tokens": 10, "completion_tokens": 5}
	}`

	resp, err := ConvertResponse([]byte(oaiResp), "claude-3-opus")
	if err != nil {
		t.Fatal(err)
	}

	if resp.Model != "claude-3-opus" {
		t.Fatalf("expected claude-3-opus, got %s", resp.Model)
	}
	if resp.Type != "message" {
		t.Fatalf("expected message type, got %s", resp.Type)
	}
	if resp.StopReason != "end_turn" {
		t.Fatalf("expected end_turn, got %s", resp.StopReason)
	}
	if len(resp.Content) != 1 || resp.Content[0].Type != "text" || resp.Content[0].Text != "Hello!" {
		t.Fatalf("unexpected content: %v", resp.Content)
	}
	if resp.Usage.InputTokens != 10 || resp.Usage.OutputTokens != 5 {
		t.Fatalf("unexpected usage: %v", resp.Usage)
	}
}

func TestConvertResponse_ToolCalls(t *testing.T) {
	oaiResp := `{
		"id": "chatcmpl-456",
		"model": "gpt-4",
		"choices": [{
			"message": {
				"role": "assistant",
				"content": null,
				"tool_calls": [{
					"id": "call_abc",
					"type": "function",
					"function": {"name": "get_weather", "arguments": "{\"location\":\"NYC\"}"}
				}]
			},
			"finish_reason": "tool_calls"
		}],
		"usage": {"prompt_tokens": 20, "completion_tokens": 15}
	}`

	resp, err := ConvertResponse([]byte(oaiResp), "claude-3-opus")
	if err != nil {
		t.Fatal(err)
	}

	if resp.StopReason != "tool_use" {
		t.Fatalf("expected tool_use, got %s", resp.StopReason)
	}
	if len(resp.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(resp.Content))
	}
	if resp.Content[0].Type != "tool_use" {
		t.Fatalf("expected tool_use type, got %s", resp.Content[0].Type)
	}
	if resp.Content[0].ID != "call_abc" {
		t.Fatalf("expected call_abc, got %s", resp.Content[0].ID)
	}
	if resp.Content[0].Name != "get_weather" {
		t.Fatalf("expected get_weather, got %s", resp.Content[0].Name)
	}
}

func TestConvertResponse_StopReasonMapping(t *testing.T) {
	tests := []struct {
		oaiReason string
		expected  string
	}{
		{"stop", "end_turn"},
		{"length", "max_tokens"},
		{"tool_calls", "tool_use"},
		{"", "end_turn"},
	}

	for _, tt := range tests {
		got := mapFinishReason(tt.oaiReason)
		if got != tt.expected {
			t.Errorf("mapFinishReason(%q) = %q, want %q", tt.oaiReason, got, tt.expected)
		}
	}
}

func TestConvertResponse_UsageWithCachedTokens(t *testing.T) {
	oaiResp := `{
		"id": "chatcmpl-789",
		"model": "gpt-4",
		"choices": [{"message": {"role": "assistant", "content": "ok"}, "finish_reason": "stop"}],
		"usage": {
			"prompt_tokens": 100,
			"completion_tokens": 50,
			"prompt_tokens_details": {"cached_tokens": 30}
		}
	}`

	resp, err := ConvertResponse([]byte(oaiResp), "test-model")
	if err != nil {
		t.Fatal(err)
	}

	if resp.Usage.InputTokens != 100 {
		t.Fatalf("expected input_tokens 100, got %d", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 50 {
		t.Fatalf("expected output_tokens 50, got %d", resp.Usage.OutputTokens)
	}
	if resp.Usage.CacheReadInputTokens != 30 {
		t.Fatalf("expected cache_read_input_tokens 30, got %d", resp.Usage.CacheReadInputTokens)
	}
}

func TestConvertResponse_EmptyChoices(t *testing.T) {
	oaiResp := `{"id":"chatcmpl-empty","model":"gpt-4","choices":[],"usage":{}}`

	resp, err := ConvertResponse([]byte(oaiResp), "test-model")
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.Content) != 0 {
		t.Fatalf("expected empty content, got %v", resp.Content)
	}
}

func TestConvertRequest_TemperatureAndTopP(t *testing.T) {
	temp := 0.7
	topP := 0.9
	req := &MessagesRequest{
		Model:       "claude-3-opus",
		MaxTokens:   100,
		Messages:    []Message{{Role: "user", Content: json.RawMessage(`"hi"`)}},
		Temperature: &temp,
		TopP:        &topP,
	}

	oai, err := ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}

	if oai["temperature"] != 0.7 {
		t.Fatalf("expected temperature 0.7, got %v", oai["temperature"])
	}
	if oai["top_p"] != 0.9 {
		t.Fatalf("expected top_p 0.9, got %v", oai["top_p"])
	}
}

func TestConvertRequest_StopSequences(t *testing.T) {
	req := &MessagesRequest{
		Model:         "claude-3-opus",
		MaxTokens:     100,
		Messages:      []Message{{Role: "user", Content: json.RawMessage(`"hi"`)}},
		StopSequences: []string{"END", "STOP"},
	}

	oai, err := ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}

	stop := oai["stop"].([]string)
	if len(stop) != 2 || stop[0] != "END" || stop[1] != "STOP" {
		t.Fatalf("unexpected stop sequences: %v", stop)
	}
}

func TestConvertRequest_MixedTextAndToolUseInAssistant(t *testing.T) {
	req := &MessagesRequest{
		Model:     "claude-3-opus",
		MaxTokens: 100,
		Messages: []Message{
			{
				Role: "assistant",
				Content: json.RawMessage(`[
					{"type":"text","text":"I will look that up."},
					{"type":"tool_use","id":"tc1","name":"search","input":{"q":"test"}},
					{"type":"tool_use","id":"tc2","name":"calc","input":{"expr":"1+1"}}
				]`),
			},
		},
	}

	oai, err := ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}

	msgs := oai["messages"].([]map[string]any)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0]["content"] != "I will look that up." {
		t.Fatalf("unexpected content: %v", msgs[0]["content"])
	}
	toolCalls := msgs[0]["tool_calls"].([]map[string]any)
	if len(toolCalls) != 2 {
		t.Fatalf("expected 2 tool_calls, got %d", len(toolCalls))
	}
}

