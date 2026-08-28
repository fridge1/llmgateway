package ppt

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zhulang/llm-gateway/internal/proxy"
)

// RunContentStrategist executes Agent 2: converts Brief Document into a Story Arc.
func RunContentStrategist(ctx context.Context, core *proxy.Core, task *PptTask, brief *BriefDocument) (*StoryArc, *AgentResult, error) {
	systemPrompt := ContentStrategistPrompt(task.Language)

	briefJSON, err := json.Marshal(brief)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal brief: %w", err)
	}

	userMessage := fmt.Sprintf("Brief Document:\n%s", string(briefJSON))

	var arc StoryArc
	result, err := RunAgentWithRetry(ctx, core, task.Model, systemPrompt, userMessage, 2, func(raw json.RawMessage) error {
		if err := json.Unmarshal(raw, &arc); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
		if len(arc.Slides) == 0 {
			return fmt.Errorf("no slides")
		}
		if arc.NarrativePattern == "" {
			return fmt.Errorf("empty narrative_pattern")
		}
		return nil
	})
	if err != nil {
		return nil, result, fmt.Errorf("content strategist failed: %w", err)
	}

	return &arc, result, nil
}
