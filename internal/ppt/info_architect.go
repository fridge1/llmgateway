package ppt

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zhulang/llm-gateway/internal/proxy"
)

// RunInfoArchitect executes Agent 3: converts Story Arc into Slide Blueprints.
func RunInfoArchitect(ctx context.Context, core *proxy.Core, task *PptTask, arc *StoryArc) (*SlideBlueprintSet, *AgentResult, error) {
	systemPrompt := InfoArchitectPrompt(task.Language)

	arcJSON, err := json.Marshal(arc)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal story arc: %w", err)
	}

	userMessage := fmt.Sprintf("Story Arc:\n%s", string(arcJSON))

	var blueprints SlideBlueprintSet
	result, err := RunAgentWithRetry(ctx, core, task.Model, systemPrompt, userMessage, 2, func(raw json.RawMessage) error {
		if err := json.Unmarshal(raw, &blueprints); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
		if len(blueprints.Blueprints) == 0 {
			return fmt.Errorf("no blueprints")
		}
		return nil
	})
	if err != nil {
		return nil, result, fmt.Errorf("info architect failed: %w", err)
	}

	return &blueprints, result, nil
}
