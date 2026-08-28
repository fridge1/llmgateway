package ppt

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zhulang/llm-gateway/internal/proxy"
)

// RunVisualDesigner executes Agent 4: analyzes slides and generates image prompts for suitable slides.
func RunVisualDesigner(ctx context.Context, core *proxy.Core, model string, presentation map[string]interface{}) (*ImagePlan, *AgentResult, error) {
	systemPrompt := VisualDesignerPrompt()

	// Build a summary of slides for the LLM
	slides, _ := presentation["slides"].([]interface{})
	type slideSummary struct {
		Index           int      `json:"index"`
		Layout          string   `json:"layout"`
		Title           string   `json:"title"`
		Body            string   `json:"body,omitempty"`
		Bullets         []string `json:"bullets,omitempty"`
		LayoutRationale string   `json:"layout_rationale,omitempty"`
	}

	summaries := make([]slideSummary, 0, len(slides))
	for i, s := range slides {
		sm, ok := s.(map[string]interface{})
		if !ok {
			continue
		}
		ss := slideSummary{Index: i}
		if v, ok := sm["layout"].(string); ok {
			ss.Layout = v
		}
		if v, ok := sm["title"].(string); ok {
			ss.Title = v
		}
		if v, ok := sm["body"].(string); ok {
			ss.Body = v
		}
		if v, ok := sm["layoutRationale"].(string); ok {
			ss.LayoutRationale = v
		}
		if v, ok := sm["bullets"].([]interface{}); ok {
			for _, b := range v {
				if bs, ok := b.(string); ok {
					ss.Bullets = append(ss.Bullets, bs)
				}
			}
		}
		summaries = append(summaries, ss)
	}

	summaryJSON, _ := json.Marshal(summaries)
	userMessage := fmt.Sprintf("Here are the slides in the presentation:\n%s", string(summaryJSON))

	var plan ImagePlan
	result, err := RunAgentWithRetry(ctx, core, model, systemPrompt, userMessage, 2, func(raw json.RawMessage) error {
		if err := json.Unmarshal(raw, &plan); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
		// Validate slide indices
		for _, img := range plan.Images {
			if img.SlideIndex < 0 || img.SlideIndex >= len(slides) {
				return fmt.Errorf("invalid slide_index %d (total slides: %d)", img.SlideIndex, len(slides))
			}
			if img.Prompt == "" {
				return fmt.Errorf("empty prompt for slide_index %d", img.SlideIndex)
			}
		}
		return nil
	})
	if err != nil {
		return nil, result, fmt.Errorf("visual designer failed: %w", err)
	}

	return &plan, result, nil
}
