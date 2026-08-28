package anthropic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRelayStream_SimpleText(t *testing.T) {
	// Create a fake upstream that sends OpenAI SSE chunks
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		chunks := []string{
			`data: {"id":"chatcmpl-1","choices":[{"delta":{"role":"assistant","content":""},"index":0}]}`,
			`data: {"id":"chatcmpl-1","choices":[{"delta":{"content":"Hello"},"index":0}]}`,
			`data: {"id":"chatcmpl-1","choices":[{"delta":{"content":" world"},"index":0}]}`,
			`data: {"id":"chatcmpl-1","choices":[{"delta":{},"index":0,"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`,
			`data: [DONE]`,
		}
		for _, c := range chunks {
			fmt.Fprintf(w, "%s\n\n", c)
			flusher.Flush()
		}
	}))
	defer upstream.Close()

	resp, err := http.Get(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	usage := RelayStream(rr, resp, "claude-3-opus", 10)

	body := rr.Body.String()

	// Verify message_start event (Anthropic SSE puts type in JSON payload, no "event:" prefix)
	if !strings.Contains(body, `"type":"message_start"`) {
		t.Error("missing message_start event")
	}

	// Verify content_block_start for text
	if !strings.Contains(body, `"type":"content_block_start"`) {
		t.Error("missing content_block_start event")
	}

	// Verify text deltas
	if !strings.Contains(body, "text_delta") {
		t.Error("missing text_delta")
	}
	if !strings.Contains(body, "Hello") {
		t.Error("missing Hello text")
	}
	if !strings.Contains(body, " world") {
		t.Error("missing world text")
	}

	// Verify message_delta with stop_reason
	if !strings.Contains(body, `"type":"message_delta"`) {
		t.Error("missing message_delta event")
	}
	if !strings.Contains(body, "end_turn") {
		t.Error("missing end_turn stop_reason")
	}

	// Verify message_stop
	if !strings.Contains(body, `"type":"message_stop"`) {
		t.Error("missing message_stop event")
	}

	// Verify usage
	if usage.PromptTokens != 10 {
		t.Errorf("expected prompt_tokens 10, got %d", usage.PromptTokens)
	}
	if usage.CompletionTokens != 5 {
		t.Errorf("expected completion_tokens 5, got %d", usage.CompletionTokens)
	}
}

func TestRelayStream_ToolUse(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		chunks := []string{
			`data: {"id":"chatcmpl-1","choices":[{"delta":{"role":"assistant","tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"get_weather","arguments":""}}]},"index":0}]}`,
			`data: {"id":"chatcmpl-1","choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"loc"}}]},"index":0}]}`,
			`data: {"id":"chatcmpl-1","choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"ation\":"}}]},"index":0}]}`,
			`data: {"id":"chatcmpl-1","choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"NYC\"}"}}]},"index":0}]}`,
			`data: {"id":"chatcmpl-1","choices":[{"delta":{},"index":0,"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":15,"completion_tokens":20}}`,
			`data: [DONE]`,
		}
		for _, c := range chunks {
			fmt.Fprintf(w, "%s\n\n", c)
			flusher.Flush()
		}
	}))
	defer upstream.Close()

	resp, err := http.Get(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	usage := RelayStream(rr, resp, "claude-3-opus", 15)

	body := rr.Body.String()

	// Verify tool_use content block start
	if !strings.Contains(body, `"type":"tool_use"`) {
		t.Error("missing tool_use content block")
	}
	if !strings.Contains(body, `"name":"get_weather"`) {
		t.Error("missing tool name")
	}

	// Verify input_json_delta events
	if !strings.Contains(body, "input_json_delta") {
		t.Error("missing input_json_delta")
	}

	// Verify tool_use stop_reason
	if !strings.Contains(body, "tool_use") {
		t.Error("missing tool_use stop_reason in message_delta")
	}

	if usage.PromptTokens != 15 {
		t.Errorf("expected prompt_tokens 15, got %d", usage.PromptTokens)
	}
	if usage.CompletionTokens != 20 {
		t.Errorf("expected completion_tokens 20, got %d", usage.CompletionTokens)
	}
}

func TestRelayStream_MixedTextAndToolUse(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		chunks := []string{
			`data: {"id":"chatcmpl-1","choices":[{"delta":{"role":"assistant","content":"Let me check. "},"index":0}]}`,
			`data: {"id":"chatcmpl-1","choices":[{"delta":{"content":""},"index":0}]}`,
			`data: {"id":"chatcmpl-1","choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"search","arguments":""}}]},"index":0}]}`,
			`data: {"id":"chatcmpl-1","choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{}"}}]},"index":0}]}`,
			`data: {"id":"chatcmpl-1","choices":[{"delta":{},"index":0,"finish_reason":"tool_calls"}]}`,
			`data: [DONE]`,
		}
		for _, c := range chunks {
			fmt.Fprintf(w, "%s\n\n", c)
			flusher.Flush()
		}
	}))
	defer upstream.Close()

	resp, err := http.Get(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	_ = RelayStream(rr, resp, "claude-3-opus", 0)

	body := rr.Body.String()

	// Should have both text and tool_use blocks
	if !strings.Contains(body, "text_delta") {
		t.Error("missing text_delta")
	}
	if !strings.Contains(body, "input_json_delta") {
		t.Error("missing input_json_delta")
	}

	// Count content_block_start events (should be 2: one text, one tool_use)
	starts := strings.Count(body, `"type":"content_block_start"`)
	if starts != 2 {
		t.Errorf("expected 2 content_block_start events, got %d", starts)
	}
}

func TestRelayStream_EmptyContent(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		chunks := []string{
			`data: {"id":"chatcmpl-1","choices":[{"delta":{"role":"assistant"},"index":0}]}`,
			`data: {"id":"chatcmpl-1","choices":[{"delta":{},"index":0,"finish_reason":"stop"}]}`,
			`data: [DONE]`,
		}
		for _, c := range chunks {
			fmt.Fprintf(w, "%s\n\n", c)
			flusher.Flush()
		}
	}))
	defer upstream.Close()

	resp, err := http.Get(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	_ = RelayStream(rr, resp, "claude-3-opus", 0)

	body := rr.Body.String()

	// Should still have message_start and message_stop
	if !strings.Contains(body, `"type":"message_start"`) {
		t.Error("missing message_start")
	}
	if !strings.Contains(body, `"type":"message_stop"`) {
		t.Error("missing message_stop")
	}

	// Verify it produces valid JSON in all events
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			data := line[6:]
			var m map[string]any
			if err := json.Unmarshal([]byte(data), &m); err != nil {
				t.Errorf("invalid JSON in SSE data: %s", data)
			}
		}
	}
}
