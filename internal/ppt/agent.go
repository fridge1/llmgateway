package ppt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/zhulang/llm-gateway/internal/circuit"
	"github.com/zhulang/llm-gateway/internal/proxy"
)

// AgentResult holds the parsed output and token usage from one agent call.
type AgentResult struct {
	RawJSON          json.RawMessage
	TotalTokens      int
	PromptTokens     int
	CompletionTokens int
}

// RunAgent sends a system+user prompt to the LLM via upstream routing and returns the parsed JSON response.
func RunAgent(ctx context.Context, core *proxy.Core, model, systemPrompt, userMessage string) (*AgentResult, error) {
	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userMessage},
		},
		"temperature":     0.7,
		"max_tokens":      16384,
		"response_format": map[string]string{"type": "json_object"},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	rt := core.Router.Load()
	upstreams, _, found := rt.GetUpstreams(model)
	if !found {
		return nil, fmt.Errorf("model %q not found in router", model)
	}

	for _, upstream := range upstreams {
		if !upstream.Breaker.AllowRequest() {
			continue
		}

		baseURL := strings.TrimRight(upstream.Config.BaseURL, "/")
		url := baseURL + "/v1/chat/completions"

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
		if err != nil {
			continue
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+upstream.Config.APIKey)

		resp, err := core.Client.Do(httpReq)
		if err != nil {
			if circuit.IsUpstreamFailure(err) {
				upstream.Breaker.RecordFailure()
			}
			continue
		}

		respBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			upstream.Breaker.RecordFailure()
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("upstream error %d: %s", resp.StatusCode, string(respBytes))
		}

		upstream.Breaker.RecordSuccess()

		var chatResp struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
			Usage struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(respBytes, &chatResp); err != nil {
			return nil, fmt.Errorf("parse response: %w", err)
		}

		if len(chatResp.Choices) == 0 {
			return nil, fmt.Errorf("no choices in response")
		}

		content := chatResp.Choices[0].Message.Content
		content = strings.TrimSpace(content)

		// Remove markdown code block markers if present
		if strings.HasPrefix(content, "```") {
			// Find the end of the opening marker (could be ```json or just ```)
			firstNewline := strings.Index(content, "\n")
			if firstNewline > 0 {
				content = content[firstNewline+1:]
			}
			// Remove closing ```
			content = strings.TrimSuffix(content, "```")
			content = strings.TrimSpace(content)
		}

		return &AgentResult{
			RawJSON:          json.RawMessage(content),
			TotalTokens:      chatResp.Usage.TotalTokens,
			PromptTokens:     chatResp.Usage.PromptTokens,
			CompletionTokens: chatResp.Usage.CompletionTokens,
		}, nil
	}

	return nil, fmt.Errorf("all upstreams failed for model %q", model)
}

// RunAgentWithRetry wraps RunAgent with retry logic.
// On transient failures or JSON parse failures it retries up to maxRetries times
// with the original prompt — feeding the failed output back in just compounds
// truncation, so each retry is a clean attempt.
func RunAgentWithRetry(ctx context.Context, core *proxy.Core, model, systemPrompt, userMessage string, maxRetries int, unmarshalFn func(json.RawMessage) error) (*AgentResult, error) {
	var lastErr error
	var accumulated AgentResult

	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, err := RunAgent(ctx, core, model, systemPrompt, userMessage)
		if err != nil {
			lastErr = err
			continue
		}

		accumulated.PromptTokens += result.PromptTokens
		accumulated.CompletionTokens += result.CompletionTokens
		accumulated.TotalTokens += result.TotalTokens
		accumulated.RawJSON = result.RawJSON

		if unmarshalFn == nil {
			return &accumulated, nil
		}

		if err := unmarshalFn(result.RawJSON); err != nil {
			lastErr = err
			continue
		}

		return &accumulated, nil
	}

	return &accumulated, fmt.Errorf("failed after %d retries: %w", maxRetries+1, lastErr)
}
