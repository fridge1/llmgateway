package moderation

import (
	"encoding/json"
	"testing"
)

func TestCheckMatchesKeywordCaseInsensitive(t *testing.T) {
	s := &Service{snap: &snapshot{enabled: true, enforceAll: true, keywords: []string{"违禁词", "badword"}}}

	if v := s.Check("这句话包含违禁词在中间"); !v.Flagged || v.MatchedRule != "违禁词" {
		t.Fatalf("expected flag on 违禁词, got %+v", v)
	}
	if v := s.Check("Contains BadWord here"); !v.Flagged || v.MatchedRule != "badword" {
		t.Fatalf("expected case-insensitive flag, got %+v", v)
	}
	if v := s.Check("perfectly clean text"); v.Flagged {
		t.Fatalf("expected clean, got %+v", v)
	}
	if v := s.Check(""); v.Flagged {
		t.Fatalf("empty text must not flag")
	}
}

func TestSnippetIsBoundedAndValid(t *testing.T) {
	s := &Service{snap: &snapshot{enabled: true, enforceAll: true, keywords: []string{"敏感"}}}
	long := "前缀前缀前缀前缀前缀前缀前缀前缀前缀前缀敏感后缀后缀后缀后缀后缀后缀后缀后缀后缀后缀"
	v := s.Check(long)
	if !v.Flagged {
		t.Fatal("expected flag")
	}
	if len(v.Snippet) == 0 || len(v.Snippet) > 180 {
		t.Fatalf("snippet size out of bounds: %d", len(v.Snippet))
	}
}

func TestApplicableRespectsScopes(t *testing.T) {
	snap := &snapshot{
		enabled:   true,
		keywords:  []string{"x"},
		modelSet:  map[string]bool{"gpt-x": true},
		tenantSet: map[string]bool{"tenant-1": true},
	}
	s := &Service{snap: snap}

	// enforceAll off: only opted-in model/tenant apply.
	if s.Applicable("other-model", "") {
		t.Fatal("non-opted model must not apply")
	}
	if !s.Applicable("gpt-x", "") {
		t.Fatal("opted-in model must apply")
	}
	if !s.Applicable("other-model", "tenant-1") {
		t.Fatal("opted-in tenant must apply")
	}

	// enforceAll on: everything applies.
	snap.enforceAll = true
	if !s.Applicable("other-model", "") {
		t.Fatal("enforceAll must apply to all")
	}

	// disabled: nothing applies.
	snap.enabled = false
	if s.Applicable("gpt-x", "tenant-1") {
		t.Fatal("disabled service must not apply")
	}
}

func TestExtractOpenAIText(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "system", "content": "system prompt"},
			map[string]any{"role": "user", "content": "hello world"},
			map[string]any{"role": "assistant", "content": "assistant reply skipped"},
			map[string]any{"role": "user", "content": []any{
				map[string]any{"type": "text", "text": "part text"},
				map[string]any{"type": "image_url", "image_url": map[string]any{"url": "http://x"}},
			}},
		},
	}
	got := ExtractOpenAIText(body)
	for _, want := range []string{"system prompt", "hello world", "part text"} {
		if !contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
	if contains(got, "assistant reply skipped") {
		t.Fatalf("assistant content must be skipped, got %q", got)
	}
}

func TestExtractAnthropicText(t *testing.T) {
	system := json.RawMessage(`"sys text"`)
	roles := []string{"user", "assistant", "user"}
	contents := []json.RawMessage{
		json.RawMessage(`"plain user"`),
		json.RawMessage(`"assistant skipped"`),
		json.RawMessage(`[{"type":"text","text":"block text"},{"type":"image","source":{}}]`),
	}
	got := ExtractAnthropicText(system, roles, contents)
	for _, want := range []string{"sys text", "plain user", "block text"} {
		if !contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
	if contains(got, "assistant skipped") {
		t.Fatalf("assistant content must be skipped, got %q", got)
	}
}

func TestExtractGeminiText(t *testing.T) {
	body := []byte(`{
		"systemInstruction": {"parts": [{"text": "gemini sys"}]},
		"contents": [
			{"role": "user", "parts": [{"text": "gemini user"}]},
			{"role": "model", "parts": [{"text": "model skipped"}]}
		]
	}`)
	got := ExtractGeminiText(body)
	if !contains(got, "gemini sys") || !contains(got, "gemini user") {
		t.Fatalf("missing expected text in %q", got)
	}
	if contains(got, "model skipped") {
		t.Fatalf("model content must be skipped, got %q", got)
	}
	if ExtractGeminiText([]byte("not json")) != "" {
		t.Fatal("unparseable body must return empty string")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
