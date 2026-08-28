package anthropic

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/zhulang/llm-gateway/internal/billing"
)

// RelayStream translates an OpenAI SSE stream to Anthropic SSE format.
// It writes events to w and returns usage info for billing.
func RelayStream(w http.ResponseWriter, resp *http.Response, model string, inputTokens int) billing.UsageInfo {
	defer resp.Body.Close()

	flusher, ok := w.(http.Flusher)
	if !ok {
		WriteError(w, http.StatusInternalServerError, "api_error", "Streaming not supported")
		return billing.UsageInfo{}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	state := &streamState{
		w:           w,
		flusher:     flusher,
		model:       model,
		inputTokens: inputTokens,
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 10*1024*1024), 100*1024*1024) // 10MB initial, 100MB max
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := line[6:]
		if data == "[DONE]" {
			state.finish("")
			break
		}
		state.processChunk(data)
	}

	// If we never got a [DONE], close gracefully
	if !state.messageStopped {
		state.finish("")
	}

	return state.usage
}

type streamState struct {
	w           http.ResponseWriter
	flusher     http.Flusher
	model       string
	inputTokens int

	messageStarted bool
	messageStopped bool
	blockIndex     int
	textBlockOpen  bool
	toolBlockOpen  bool
	stopReason     string
	usage          billing.UsageInfo
	outputTokens   int
}

func (s *streamState) processChunk(data string) {
	var chunk map[string]any
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return
	}

	// Extract usage if present (from the final chunk with stream_options)
	if usageObj, ok := chunk["usage"].(map[string]any); ok {
		s.usage.CacheTokensIncludedInPrompt = true // OpenAI format: prompt_tokens includes cached
		if pt, ok := usageObj["prompt_tokens"].(float64); ok {
			s.usage.PromptTokens = int(pt)
			s.inputTokens = int(pt)
		}
		if ct, ok := usageObj["completion_tokens"].(float64); ok {
			s.usage.CompletionTokens = int(ct)
			s.outputTokens = int(ct)
		}
		if details, ok := usageObj["prompt_tokens_details"].(map[string]any); ok {
			if cached, ok := details["cached_tokens"].(float64); ok {
				s.usage.CacheReadTokens = int(cached)
			}
		}
	}

	// Emit message_start on first chunk
	if !s.messageStarted {
		s.messageStarted = true
		s.emitMessageStart()
	}

	choices, _ := chunk["choices"].([]any)
	if len(choices) == 0 {
		return
	}

	choice, _ := choices[0].(map[string]any)
	if choice == nil {
		return
	}

	// Check finish_reason
	if fr, ok := choice["finish_reason"].(string); ok && fr != "" {
		s.stopReason = mapFinishReason(fr)
	}

	delta, _ := choice["delta"].(map[string]any)
	if delta == nil {
		return
	}

	// Handle text content
	if content, ok := delta["content"].(string); ok && content != "" {
		if !s.textBlockOpen {
			s.openTextBlock()
		}
		s.emitTextDelta(content)
	}

	// Handle tool calls
	if toolCalls, ok := delta["tool_calls"].([]any); ok {
		for _, tc := range toolCalls {
			tcMap, _ := tc.(map[string]any)
			if tcMap == nil {
				continue
			}
			// If it has an "id" field, it's the start of a new tool call
			if id, ok := tcMap["id"].(string); ok && id != "" {
				// Close previous block if open
				s.closePreviousBlock()

				fnMap, _ := tcMap["function"].(map[string]any)
				name := ""
				if fnMap != nil {
					name, _ = fnMap["name"].(string)
				}
				s.openToolUseBlock(id, name)
			}

			// Emit argument deltas
			if fnMap, ok := tcMap["function"].(map[string]any); ok {
				if args, ok := fnMap["arguments"].(string); ok && args != "" {
					s.emitInputJSONDelta(args)
				}
			}
		}
	}
}

func (s *streamState) finish(reason string) {
	if s.messageStopped {
		return
	}
	s.messageStopped = true

	if !s.messageStarted {
		s.messageStarted = true
		s.emitMessageStart()
	}

	s.closePreviousBlock()

	if reason != "" {
		s.stopReason = reason
	}
	if s.stopReason == "" {
		s.stopReason = "end_turn"
	}

	// Fallback: if upstream didn't return prompt_tokens (e.g., Zhipu OpenAI-compatible API),
	// use the inputTokens parameter passed to RelayStream
	if s.usage.PromptTokens == 0 && s.inputTokens > 0 {
		s.usage.PromptTokens = s.inputTokens
	}

	s.emitMessageDelta()
	s.emitMessageStop()
}

func (s *streamState) closePreviousBlock() {
	if s.textBlockOpen {
		s.emitEvent("content_block_stop", ContentBlockStopEvent{
			Type:  "content_block_stop",
			Index: s.blockIndex,
		})
		s.textBlockOpen = false
		s.blockIndex++
	}
	if s.toolBlockOpen {
		s.emitEvent("content_block_stop", ContentBlockStopEvent{
			Type:  "content_block_stop",
			Index: s.blockIndex,
		})
		s.toolBlockOpen = false
		s.blockIndex++
	}
}

func (s *streamState) openTextBlock() {
	s.emitEvent("content_block_start", ContentBlockStartEvent{
		Type:  "content_block_start",
		Index: s.blockIndex,
		ContentBlock: struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{
			Type: "text",
			Text: "",
		},
	})
	s.textBlockOpen = true
}

func (s *streamState) openToolUseBlock(id, name string) {
	s.emitEvent("content_block_start", ContentBlockStartEvent{
		Type:  "content_block_start",
		Index: s.blockIndex,
		ContentBlock: ResponseBlock{
			Type: "tool_use",
			ID:   id,
			Name: name,
		},
	})
	s.toolBlockOpen = true
}

func (s *streamState) emitTextDelta(text string) {
	s.emitEvent("content_block_delta", ContentBlockDeltaEvent{
		Type:  "content_block_delta",
		Index: s.blockIndex,
		Delta: BlockDelta{
			Type: "text_delta",
			Text: text,
		},
	})
}

func (s *streamState) emitInputJSONDelta(partial string) {
	s.emitEvent("content_block_delta", ContentBlockDeltaEvent{
		Type:  "content_block_delta",
		Index: s.blockIndex,
		Delta: BlockDelta{
			Type:        "input_json_delta",
			PartialJSON: partial,
		},
	})
}

func (s *streamState) emitMessageStart() {
	s.emitEvent("message_start", MessageStartEvent{
		Type: "message_start",
		Message: MessagesResponse{
			ID:      fmt.Sprintf("msg_%d", 0), // placeholder
			Type:    "message",
			Role:    "assistant",
			Content: []ResponseBlock{},
			Model:   s.model,
			Usage: Usage{
				InputTokens:  s.inputTokens,
				OutputTokens: 0,
			},
		},
	})
}

func (s *streamState) emitMessageDelta() {
	s.emitEvent("message_delta", MessageDeltaEvent{
		Type: "message_delta",
		Delta: MessageDeltaBody{
			StopReason: s.stopReason,
		},
		Usage: MessageDeltaUsage{
			OutputTokens: s.outputTokens,
		},
	})
}

func (s *streamState) emitMessageStop() {
	s.emitEvent("message_stop", MessageStopEvent{
		Type: "message_stop",
	})
}

func (s *streamState) emitEvent(eventType string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	// Anthropic SSE format: pure data lines without event: prefix
	// (event type is already in the JSON payload's "type" field)
	fmt.Fprintf(s.w, "data: %s\n\n", string(data))
	s.flusher.Flush()
}
