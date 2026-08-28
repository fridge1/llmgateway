package responses

import (
	"encoding/json"
	"testing"
)

func TestConvertInput_SkipsUnparseableItemNotWholeRequest(t *testing.T) {
	// One bad item must not fail the whole batch; good items still convert.
	input := `[{"type":"message","role":"user","content":"hi"},{"type":"web_search_call","id":"ws_1"}]`
	msgs, err := ConvertInput(json.RawMessage(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message (unknown item skipped), got %d: %v", len(msgs), msgs)
	}
	if msgs[0]["role"] != "user" || msgs[0]["content"] != "hi" {
		t.Fatalf("unexpected surviving message: %v", msgs[0])
	}
}

func TestConvertInput_NonArrayNonStringErrors(t *testing.T) {
	// A bare object (neither string nor array) still produces the clear error.
	if _, err := ConvertInput(json.RawMessage(`{"foo":"bar"}`)); err == nil {
		t.Fatal("expected error for object input, got nil")
	}
}

func TestIsResponsesShapedBody(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{"responses input", `{"model":"gpt-5.5","input":"hi"}`, true},
		{"responses input array", `{"model":"gpt-5.5","input":[{"type":"message","role":"user","content":"hi"}]}`, true},
		{"chat completions", `{"model":"gpt-5.5","messages":[{"role":"user","content":"hi"}]}`, false},
		{"both input and messages", `{"model":"x","input":"hi","messages":[]}`, false},
		{"invalid json", `{`, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsResponsesShapedBody([]byte(tc.body)); got != tc.want {
				t.Fatalf("IsResponsesShapedBody(%q) = %v, want %v", tc.body, got, tc.want)
			}
		})
	}
}
