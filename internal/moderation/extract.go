package moderation

import (
	"encoding/json"
	"strings"
)

// ExtractOpenAIText pulls user-authored text out of an OpenAI chat request
// body already unmarshalled to map[string]any. Only user/system messages are
// scanned; assistant turns echo prior model output and are skipped.
func ExtractOpenAIText(reqBody map[string]any) string {
	msgs, _ := reqBody["messages"].([]any)
	var sb strings.Builder
	for _, m := range msgs {
		msg, ok := m.(map[string]any)
		if !ok {
			continue
		}
		role, _ := msg["role"].(string)
		if role == "assistant" || role == "tool" {
			continue
		}
		switch c := msg["content"].(type) {
		case string:
			sb.WriteString(c)
			sb.WriteByte('\n')
		case []any:
			for _, part := range c {
				pm, ok := part.(map[string]any)
				if !ok {
					continue
				}
				if t, _ := pm["text"].(string); t != "" {
					sb.WriteString(t)
					sb.WriteByte('\n')
				}
			}
		}
	}
	return sb.String()
}

// anthropicContentBlock is the minimal shape needed to pull text out of
// Anthropic message content arrays.
type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ExtractAnthropicText pulls user text from Anthropic system + messages fields.
// roles[i] pairs with contents[i]; each content is either a JSON string or an
// array of content blocks. Assistant turns are skipped.
func ExtractAnthropicText(system json.RawMessage, roles []string, contents []json.RawMessage) string {
	var sb strings.Builder
	appendRaw := func(raw json.RawMessage) {
		if len(raw) == 0 {
			return
		}
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			sb.WriteString(s)
			sb.WriteByte('\n')
			return
		}
		var blocks []anthropicContentBlock
		if err := json.Unmarshal(raw, &blocks); err == nil {
			for _, b := range blocks {
				if b.Text != "" {
					sb.WriteString(b.Text)
					sb.WriteByte('\n')
				}
			}
		}
	}
	appendRaw(system)
	for i, content := range contents {
		if i < len(roles) && roles[i] == "assistant" {
			continue
		}
		appendRaw(content)
	}
	return sb.String()
}

// geminiRequest is the minimal shape of a Gemini generateContent body.
type geminiRequest struct {
	Contents []struct {
		Role  string `json:"role"`
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"contents"`
	SystemInstruction *struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"systemInstruction"`
}

// ExtractGeminiText parses a raw Gemini request body and pulls user text.
// Returns "" (skip moderation) if the body doesn't parse.
func ExtractGeminiText(body []byte) string {
	var req geminiRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return ""
	}
	var sb strings.Builder
	if req.SystemInstruction != nil {
		for _, p := range req.SystemInstruction.Parts {
			if p.Text != "" {
				sb.WriteString(p.Text)
				sb.WriteByte('\n')
			}
		}
	}
	for _, c := range req.Contents {
		if c.Role == "model" {
			continue
		}
		for _, p := range c.Parts {
			if p.Text != "" {
				sb.WriteString(p.Text)
				sb.WriteByte('\n')
			}
		}
	}
	return sb.String()
}
