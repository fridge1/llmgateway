package responses

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// sseEvent is a parsed Responses API SSE event (event name + JSON data map).
type sseEvent struct {
	event string
	data  map[string]any
}

// parseSSE splits a recorded SSE body into typed events for assertions.
func parseSSE(t *testing.T, body string) []sseEvent {
	t.Helper()
	var events []sseEvent
	for _, block := range strings.Split(body, "\n\n") {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		var ev sseEvent
		for _, line := range strings.Split(block, "\n") {
			switch {
			case strings.HasPrefix(line, "event: "):
				ev.event = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				m := map[string]any{}
				raw := strings.TrimPrefix(line, "data: ")
				if err := json.Unmarshal([]byte(raw), &m); err != nil {
					t.Fatalf("failed to parse SSE data %q: %v", raw, err)
				}
				ev.data = m
			}
		}
		if ev.event != "" {
			events = append(events, ev)
		}
	}
	return events
}

// assertMonotonicSequence verifies every event carries a sequence_number and
// that the numbers strictly increase in emission order, as required by the
// OpenAI Responses API streaming spec.
func assertMonotonicSequence(t *testing.T, events []sseEvent) {
	t.Helper()
	prev := -1
	for i, ev := range events {
		raw, ok := ev.data["sequence_number"]
		if !ok {
			t.Fatalf("event %d (%s) missing sequence_number", i, ev.event)
		}
		num, ok := raw.(float64)
		if !ok {
			t.Fatalf("event %d (%s) sequence_number not numeric: %v", i, ev.event, raw)
		}
		if int(num) <= prev {
			t.Fatalf("event %d (%s) sequence_number %d not strictly increasing (prev %d)", i, ev.event, int(num), prev)
		}
		prev = int(num)
	}
}

// findEvent returns the first event with the given name, or fails.
func findEvent(t *testing.T, events []sseEvent, name string) sseEvent {
	t.Helper()
	for _, ev := range events {
		if ev.event == name {
			return ev
		}
	}
	t.Fatalf("expected event %q not found", name)
	return sseEvent{}
}

func TestRelayStream_SimpleText(t *testing.T) {
	// Simulate an upstream SSE response with text deltas
	sseBody := strings.Join([]string{
		`data: {"id":"chatcmpl-1","choices":[{"delta":{"content":"Hel"},"index":0}]}`,
		"",
		`data: {"id":"chatcmpl-1","choices":[{"delta":{"content":"lo!"},"index":0}]}`,
		"",
		`data: {"id":"chatcmpl-1","choices":[{"delta":{},"finish_reason":"stop","index":0}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(sseBody)),
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
	}

	rr := httptest.NewRecorder()
	usage := RelayStream(rr, resp, "test-model")

	body := rr.Body.String()

	// Check SSE events are present
	if !strings.Contains(body, "event: response.created") {
		t.Fatal("expected response.created event")
	}
	if !strings.Contains(body, "event: response.in_progress") {
		t.Fatal("expected response.in_progress event")
	}
	if !strings.Contains(body, "event: response.output_text.delta") {
		t.Fatal("expected response.output_text.delta event")
	}
	if !strings.Contains(body, "event: response.content_part.done") {
		t.Fatal("expected response.content_part.done event")
	}
	if !strings.Contains(body, "event: response.output_item.done") {
		t.Fatal("expected response.output_item.done event")
	}
	if !strings.Contains(body, "event: response.completed") {
		t.Fatal("expected response.completed event")
	}

	// Check deltas contain the text
	if !strings.Contains(body, `"delta":"Hel"`) {
		t.Fatal("expected delta 'Hel' in output")
	}
	if !strings.Contains(body, `"delta":"lo!"`) {
		t.Fatal("expected delta 'lo!' in output")
	}

	// Check content type
	ct := rr.Header().Get("Content-Type")
	if ct != "text/event-stream" {
		t.Fatalf("expected Content-Type text/event-stream, got %q", ct)
	}

	// Check usage
	if usage.PromptTokens != 10 {
		t.Fatalf("expected prompt_tokens 10, got %d", usage.PromptTokens)
	}
	if usage.CompletionTokens != 5 {
		t.Fatalf("expected completion_tokens 5, got %d", usage.CompletionTokens)
	}
}

func TestRelayStream_FunctionCall(t *testing.T) {
	sseBody := strings.Join([]string{
		`data: {"id":"chatcmpl-1","choices":[{"delta":{"content":""},"index":0}]}`,
		"",
		fmt.Sprintf(`data: {"id":"chatcmpl-1","choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_abc","type":"function","function":{"name":"get_weather","arguments":""}}]},"index":0}]}`),
		"",
		`data: {"id":"chatcmpl-1","choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"loc"}}]},"index":0}]}`,
		"",
		`data: {"id":"chatcmpl-1","choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"ation\":\"NYC\"}"}}]},"index":0}]}`,
		"",
		`data: {"id":"chatcmpl-1","choices":[{"delta":{},"finish_reason":"tool_calls","index":0}],"usage":{"prompt_tokens":20,"completion_tokens":15}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(sseBody)),
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
	}

	rr := httptest.NewRecorder()
	usage := RelayStream(rr, resp, "test-model")

	body := rr.Body.String()

	if !strings.Contains(body, "event: response.function_call_arguments.delta") {
		t.Fatal("expected function_call_arguments.delta event")
	}
	if !strings.Contains(body, "event: response.output_item.done") {
		t.Fatal("expected output_item.done event")
	}
	if !strings.Contains(body, `"call_id":"call_abc"`) {
		t.Fatal("expected call_id in output")
	}

	if usage.PromptTokens != 20 {
		t.Fatalf("expected prompt_tokens 20, got %d", usage.PromptTokens)
	}
	if usage.CompletionTokens != 15 {
		t.Fatalf("expected completion_tokens 15, got %d", usage.CompletionTokens)
	}
}

func TestRelayStream_UsageExtraction(t *testing.T) {
	sseBody := strings.Join([]string{
		`data: {"id":"chatcmpl-1","choices":[{"delta":{"content":"ok"},"index":0}]}`,
		"",
		`data: {"id":"chatcmpl-1","choices":[{"delta":{},"finish_reason":"stop","index":0}],"usage":{"prompt_tokens":42,"completion_tokens":7}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(sseBody)),
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
	}

	rr := httptest.NewRecorder()
	usage := RelayStream(rr, resp, "test-model")

	if usage.PromptTokens != 42 {
		t.Fatalf("expected prompt_tokens 42, got %d", usage.PromptTokens)
	}
	if usage.CompletionTokens != 7 {
		t.Fatalf("expected completion_tokens 7, got %d", usage.CompletionTokens)
	}
}

// TestRelayStream_FunctionCallResponsesCompliance verifies the OpenAI upstream
// path emits a spec-compliant Responses function-call event sequence that Cursor
// Agent relies on: item_id-tagged argument deltas, a function_call_arguments.done
// carrying the full arguments, and a monotonic sequence_number on every event.
func TestRelayStream_FunctionCallResponsesCompliance(t *testing.T) {
	sseBody := strings.Join([]string{
		`data: {"id":"chatcmpl-1","choices":[{"delta":{"content":""},"index":0}]}`,
		"",
		`data: {"id":"chatcmpl-1","choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_abc","type":"function","function":{"name":"get_weather","arguments":""}}]},"index":0}]}`,
		"",
		`data: {"id":"chatcmpl-1","choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"loc"}}]},"index":0}]}`,
		"",
		`data: {"id":"chatcmpl-1","choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"ation\":\"NYC\"}"}}]},"index":0}]}`,
		"",
		`data: {"id":"chatcmpl-1","choices":[{"delta":{},"finish_reason":"tool_calls","index":0}],"usage":{"prompt_tokens":20,"completion_tokens":15}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(sseBody)),
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
	}

	rr := httptest.NewRecorder()
	RelayStream(rr, resp, "test-model")

	events := parseSSE(t, rr.Body.String())
	assertMonotonicSequence(t, events)
	assertFunctionCallCompliance(t, events, "call_abc", "get_weather", `{"location":"NYC"}`)
}

// TestRelayAnthropicStream_FunctionCallResponsesCompliance verifies the Anthropic
// upstream path produces the same spec-compliant function-call event sequence.
func TestRelayAnthropicStream_FunctionCallResponsesCompliance(t *testing.T) {
	sseBody := strings.Join([]string{
		`event: message_start`,
		`data: {"message":{"usage":{"input_tokens":20}}}`,
		"",
		`event: content_block_start`,
		`data: {"index":0,"content_block":{"type":"text"}}`,
		"",
		`event: content_block_delta`,
		`data: {"index":0,"delta":{"type":"text_delta","text":"Let me check."}}`,
		"",
		`event: content_block_stop`,
		`data: {"index":0}`,
		"",
		`event: content_block_start`,
		`data: {"index":1,"content_block":{"type":"tool_use","id":"toolu_1","name":"get_weather"}}`,
		"",
		`event: content_block_delta`,
		`data: {"index":1,"delta":{"type":"input_json_delta","partial_json":"{\"loc"}}`,
		"",
		`event: content_block_delta`,
		`data: {"index":1,"delta":{"type":"input_json_delta","partial_json":"ation\":\"NYC\"}"}}`,
		"",
		`event: content_block_stop`,
		`data: {"index":1}`,
		"",
		`event: message_delta`,
		`data: {"usage":{"output_tokens":15}}`,
		"",
		`event: message_stop`,
		`data: {}`,
		"",
	}, "\n")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(sseBody)),
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
	}

	rr := httptest.NewRecorder()
	RelayAnthropicStream(rr, resp, "test-model")

	events := parseSSE(t, rr.Body.String())
	assertMonotonicSequence(t, events)
	assertFunctionCallCompliance(t, events, "toolu_1", "get_weather", `{"location":"NYC"}`)
}

// assertFunctionCallCompliance checks the function-call event sequence shared by
// both relay paths: the added item carries an fc_-prefixed id and matching
// call_id; argument deltas reference that item_id; and a function_call_arguments.done
// event carries the fully accumulated arguments under the same item_id.
func assertFunctionCallCompliance(t *testing.T, events []sseEvent, wantCallID, wantName, wantArgs string) {
	t.Helper()

	// Locate the function_call output_item.added event.
	var fcItemID string
	for _, ev := range events {
		if ev.event != "response.output_item.added" {
			continue
		}
		item, _ := ev.data["item"].(map[string]any)
		if item == nil || item["type"] != "function_call" {
			continue
		}
		id, _ := item["id"].(string)
		if !strings.HasPrefix(id, "fc_") {
			t.Fatalf("function_call item id should have fc_ prefix, got %q", id)
		}
		if cid, _ := item["call_id"].(string); cid != wantCallID {
			t.Fatalf("expected call_id %q, got %q", wantCallID, cid)
		}
		fcItemID = id
	}
	if fcItemID == "" {
		t.Fatal("no function_call output_item.added event found")
	}

	// Argument deltas must reference the function_call item_id.
	sawDelta := false
	for _, ev := range events {
		if ev.event != "response.function_call_arguments.delta" {
			continue
		}
		sawDelta = true
		if id, _ := ev.data["item_id"].(string); id != fcItemID {
			t.Fatalf("delta item_id %q does not match function_call item id %q", id, fcItemID)
		}
	}
	if !sawDelta {
		t.Fatal("expected at least one response.function_call_arguments.delta event")
	}

	// function_call_arguments.done must carry the full arguments and name.
	done := findEvent(t, events, "response.function_call_arguments.done")
	if id, _ := done.data["item_id"].(string); id != fcItemID {
		t.Fatalf("done item_id %q does not match function_call item id %q", id, fcItemID)
	}
	if name, _ := done.data["name"].(string); name != wantName {
		t.Fatalf("expected done name %q, got %q", wantName, name)
	}
	if args, _ := done.data["arguments"].(string); args != wantArgs {
		t.Fatalf("expected done arguments %q, got %q", wantArgs, args)
	}

	// response.completed must include the function_call with final arguments.
	completed := findEvent(t, events, "response.completed")
	respObj, _ := completed.data["response"].(map[string]any)
	output, _ := respObj["output"].([]any)
	found := false
	for _, o := range output {
		item, _ := o.(map[string]any)
		if item == nil || item["type"] != "function_call" {
			continue
		}
		if item["call_id"] == wantCallID {
			if args, _ := item["arguments"].(string); args != wantArgs {
				t.Fatalf("completed function_call arguments %q != %q", args, wantArgs)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("response.completed output missing the function_call item")
	}
}

// assertSequentialItems verifies output items never interleave: an item must be
// "done" before the next item is "added". Strict Responses clients (Cursor)
// break when a new item is added while another is still open.
func assertSequentialItems(t *testing.T, events []sseEvent) {
	t.Helper()
	openIdx := -1
	for _, ev := range events {
		switch ev.event {
		case "response.output_item.added":
			if openIdx != -1 {
				t.Fatalf("item interleaving: output_item.added (index %v) while index %d still open",
					ev.data["output_index"], openIdx)
			}
			oi, _ := ev.data["output_index"].(float64)
			openIdx = int(oi)
		case "response.output_item.done":
			openIdx = -1
		}
	}
}

// TestRelayStream_ToolOnlyNoPhantomMessage verifies a tool-call-only turn emits
// no assistant message item and keeps items strictly sequential.
func TestRelayStream_ToolOnlyNoPhantomMessage(t *testing.T) {
	sseBody := strings.Join([]string{
		`data: {"id":"c","choices":[{"delta":{"content":""},"index":0}]}`,
		"",
		`data: {"id":"c","choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_x","type":"function","function":{"name":"read_file","arguments":""}}]},"index":0}]}`,
		"",
		`data: {"id":"c","choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"p\":\"a\"}"}}]},"index":0}]}`,
		"",
		`data: {"id":"c","choices":[{"delta":{},"finish_reason":"tool_calls","index":0}],"usage":{"prompt_tokens":5,"completion_tokens":3}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")
	resp := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(sseBody)), Header: http.Header{}}
	rr := httptest.NewRecorder()
	RelayStream(rr, resp, "test-model")

	events := parseSSE(t, rr.Body.String())
	assertMonotonicSequence(t, events)
	assertSequentialItems(t, events)

	for _, ev := range events {
		if ev.event != "response.output_item.added" {
			continue
		}
		if item, _ := ev.data["item"].(map[string]any); item != nil && item["type"] == "message" {
			t.Fatal("tool-only turn must not emit a message output item")
		}
	}

	// The function_call must be at output_index 0 (no phantom message ahead of it).
	fcAdded := findEvent(t, events, "response.output_item.added")
	if oi, _ := fcAdded.data["output_index"].(float64); int(oi) != 0 {
		t.Fatalf("function_call should be at output_index 0, got %v", oi)
	}
}

// TestRelayStream_TextThenToolSequential verifies a text-then-tool turn closes
// the message item before opening the function_call item.
func TestRelayStream_TextThenToolSequential(t *testing.T) {
	sseBody := strings.Join([]string{
		`data: {"id":"c","choices":[{"delta":{"content":"Reading."},"index":0}]}`,
		"",
		`data: {"id":"c","choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_y","type":"function","function":{"name":"read_file","arguments":"{}"}}]},"index":0}]}`,
		"",
		`data: {"id":"c","choices":[{"delta":{},"finish_reason":"tool_calls","index":0}],"usage":{"prompt_tokens":5,"completion_tokens":3}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")
	resp := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(sseBody)), Header: http.Header{}}
	rr := httptest.NewRecorder()
	RelayStream(rr, resp, "test-model")

	events := parseSSE(t, rr.Body.String())
	assertMonotonicSequence(t, events)
	assertSequentialItems(t, events)
	assertFunctionCallCompliance(t, events, "call_y", "read_file", "{}")
}

// TestRelayStream_TextDoneEvents verifies the text path emits output_text.done
// with the accumulated text and an item_id matching the message item.
func TestRelayStream_TextDoneEvents(t *testing.T) {
	sseBody := strings.Join([]string{
		`data: {"id":"chatcmpl-1","choices":[{"delta":{"content":"Hel"},"index":0}]}`,
		"",
		`data: {"id":"chatcmpl-1","choices":[{"delta":{"content":"lo!"},"index":0}]}`,
		"",
		`data: {"id":"chatcmpl-1","choices":[{"delta":{},"finish_reason":"stop","index":0}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(sseBody)),
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
	}

	rr := httptest.NewRecorder()
	RelayStream(rr, resp, "test-model")

	events := parseSSE(t, rr.Body.String())
	assertMonotonicSequence(t, events)

	added := findEvent(t, events, "response.output_item.added")
	item, _ := added.data["item"].(map[string]any)
	msgID, _ := item["id"].(string)

	textDelta := findEvent(t, events, "response.output_text.delta")
	if id, _ := textDelta.data["item_id"].(string); id != msgID {
		t.Fatalf("output_text.delta item_id %q != message id %q", id, msgID)
	}

	textDone := findEvent(t, events, "response.output_text.done")
	if txt, _ := textDone.data["text"].(string); txt != "Hello!" {
		t.Fatalf("expected output_text.done text 'Hello!', got %q", txt)
	}
	if id, _ := textDone.data["item_id"].(string); id != msgID {
		t.Fatalf("output_text.done item_id %q != message id %q", id, msgID)
	}
}

// firstAddedItem returns the item map of the first output_item.added event.
func firstAddedItem(t *testing.T, events []sseEvent) map[string]any {
	t.Helper()
	for _, ev := range events {
		if ev.event == "response.output_item.added" {
			item, _ := ev.data["item"].(map[string]any)
			return item
		}
	}
	t.Fatal("no output_item.added event")
	return nil
}

// TestRelayStream_ReasoningModel verifies that, for reasoning-model clients, a
// reasoning output item is emitted first (output_index 0), the message follows
// at output_index 1, and the reasoning item carries encrypted_content when
// requested. This mirrors Cursor's gpt-5.x Responses requests.
func TestRelayStream_ReasoningModel(t *testing.T) {
	sseBody := strings.Join([]string{
		`data: {"id":"c","choices":[{"delta":{"reasoning_content":"Think"},"index":0}]}`,
		"",
		`data: {"id":"c","choices":[{"delta":{"reasoning_content":"ing..."},"index":0}]}`,
		"",
		`data: {"id":"c","choices":[{"delta":{"content":"Answer."},"index":0}]}`,
		"",
		`data: {"id":"c","choices":[{"delta":{},"finish_reason":"stop","index":0}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")
	resp := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(sseBody)), Header: http.Header{}}
	rr := httptest.NewRecorder()
	RelayStream(rr, resp, "gw/gpt-5.5", relayOptions{emitReasoning: true, encryptedContent: true})

	events := parseSSE(t, rr.Body.String())
	assertMonotonicSequence(t, events)
	assertSequentialItems(t, events)

	// First output item must be the reasoning item at output_index 0.
	first := firstAddedItem(t, events)
	if first["type"] != "reasoning" {
		t.Fatalf("first output item should be reasoning, got %v", first["type"])
	}

	// Reasoning summary should have streamed.
	rd := findEvent(t, events, "response.reasoning_summary_text.delta")
	if d, _ := rd.data["delta"].(string); d == "" {
		t.Fatal("expected non-empty reasoning_summary_text.delta")
	}

	// Reasoning item done must carry encrypted_content and full summary text.
	var reasoningDone map[string]any
	for _, ev := range events {
		if ev.event == "response.output_item.done" {
			if item, _ := ev.data["item"].(map[string]any); item != nil && item["type"] == "reasoning" {
				reasoningDone = item
			}
		}
	}
	if reasoningDone == nil {
		t.Fatal("expected reasoning output_item.done")
	}
	if enc, _ := reasoningDone["encrypted_content"].(string); enc == "" {
		t.Fatal("reasoning item must include encrypted_content when requested")
	}

	// Message must come after reasoning, at output_index 1.
	for _, ev := range events {
		if ev.event == "response.output_item.added" {
			if item, _ := ev.data["item"].(map[string]any); item != nil && item["type"] == "message" {
				if oi, _ := ev.data["output_index"].(float64); int(oi) != 1 {
					t.Fatalf("message should be at output_index 1, got %v", oi)
				}
			}
		}
	}
}

// TestRelayStream_ReasoningModelNoUpstreamReasoning verifies that a reasoning
// item is still emitted (with an empty summary) even when the upstream produces
// no reasoning tokens — strict clients require its presence.
func TestRelayStream_ReasoningModelNoUpstreamReasoning(t *testing.T) {
	sseBody := strings.Join([]string{
		`data: {"id":"c","choices":[{"delta":{"content":"Hi."},"index":0}]}`,
		"",
		`data: {"id":"c","choices":[{"delta":{},"finish_reason":"stop","index":0}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")
	resp := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(sseBody)), Header: http.Header{}}
	rr := httptest.NewRecorder()
	RelayStream(rr, resp, "gw/gpt-5.5", relayOptions{emitReasoning: true, encryptedContent: true})

	events := parseSSE(t, rr.Body.String())
	assertMonotonicSequence(t, events)
	assertSequentialItems(t, events)

	first := firstAddedItem(t, events)
	if first["type"] != "reasoning" {
		t.Fatalf("first output item should be reasoning, got %v", first["type"])
	}
}
