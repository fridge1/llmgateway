package ppt

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zhulang/llm-gateway/internal/proxy"
)

// RunBriefAnalyst executes Agent 1: converts user input into a structured Brief Document.
func RunBriefAnalyst(ctx context.Context, core *proxy.Core, task *PptTask) (*BriefDocument, *AgentResult, error) {
	systemPrompt := BriefAnalystPrompt(task.Language)

	userMessage := fmt.Sprintf(`Topic: %s
Slide count: %d
Audience: %s
Tone: %s
Purpose: %s`,
		task.Topic, task.SlideCount, task.Audience, task.Tone, task.Purpose)

	if task.ContextText != "" {
		userMessage += fmt.Sprintf("\n\nReference material provided by the user:\n---\n%s\n---", task.ContextText)
	}

	var brief BriefDocument
	result, err := RunAgentWithRetry(ctx, core, task.Model, systemPrompt, userMessage, 2, func(raw json.RawMessage) error {
		if err := json.Unmarshal(raw, &brief); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
		if brief.PresentationType == "" {
			return fmt.Errorf("empty presentation_type")
		}
		if len(brief.KeyMessages) == 0 {
			return fmt.Errorf("no key_messages")
		}
		return nil
	})
	if err != nil {
		return nil, result, fmt.Errorf("brief analyst failed: %w", err)
	}

	return &brief, result, nil
}
