package responses

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/zhulang/llm-gateway/internal/billing"
)

// respTrace enables verbose request/response logging on the Responses API path.
// Gated by GW_DEBUG_TRACE so it can be toggled in production without a rebuild.
// TODO(diag): temporary — remove once Cursor Agent compatibility is confirmed.
var respTrace = func() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("GW_DEBUG_TRACE")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}()

// fcEntry tracks a streamed function_call output item.
type fcEntry struct {
	outputIndex int
	itemID      string
	callID      string
	name        string
	args        string
}

// streamState centralizes Responses API SSE emission for both the
// Chat-Completions (RelayStream) and Anthropic (RelayAnthropicStream) relay
// paths so they stay spec-compliant and identical.
//
// Two invariants matter for strict clients (e.g. Cursor Agent):
//  1. Output items are emitted strictly sequentially: an item must be fully
//     "done" before the next item is "added" (no interleaving).
//  2. A message output item is created lazily — only when the model actually
//     produces assistant text. A tool-call-only turn emits no message item.
type streamState struct {
	responseID string
	model      string
	now        int64
	seq        int

	nextOutputIndex int
	openIndex       int // output_index of the currently open item, -1 if none

	messageStarted     bool
	messageItemID      string
	messageOutputIndex int
	messageText        string

	// Reasoning output item (emitted for reasoning-model clients like Cursor).
	emitReasoning      bool            // client requested reasoning → must emit a reasoning item
	encReasoning       bool            // client requested reasoning.encrypted_content
	reasoningRaw       json.RawMessage // echoed back in response object
	reasoningStarted   bool
	reasoningItemID    string
	reasoningIndex     int
	reasoningText      string
	reasoningPartOpen  bool
	reasoningCompleted bool

	fcByKey map[int]*fcEntry // upstream tool/block index -> entry
	fcOrder []*fcEntry
}

// relayOptions configures optional Responses API behaviours per request.
type relayOptions struct {
	emitReasoning    bool
	encryptedContent bool
	reasoningRaw     json.RawMessage // echoed back in response.reasoning field
}

func newStreamState(model string, opts ...relayOptions) *streamState {
	s := &streamState{
		responseID: generateResponseID(),
		model:      model,
		now:        time.Now().Unix(),
		openIndex:  -1,
		fcByKey:    make(map[int]*fcEntry),
	}
	if len(opts) > 0 {
		s.emitReasoning = opts[0].emitReasoning
		s.encReasoning = opts[0].encryptedContent
		s.reasoningRaw = opts[0].reasoningRaw
	}
	return s
}

// next returns the current sequence number and advances the counter.
func (s *streamState) next() int {
	n := s.seq
	s.seq++
	return n
}

func textPart(text string) OutputContent {
	return OutputContent{Type: "output_text", Text: text, Annotations: []any{}}
}

func (s *streamState) responseObject(status string, output []OutputItem, usage *ResponseUsage) Response {
	resp := newResponse(s.responseID, s.model, s.now, status, output)
	resp.Usage = usage
	if len(s.reasoningRaw) > 0 {
		resp.Reasoning = s.reasoningRaw
	}
	return resp
}

// emitStart sends response.created and response.in_progress. Call once, right
// after the SSE headers are written.
func (s *streamState) emitStart(w http.ResponseWriter, flusher http.Flusher) {
	skeleton := s.responseObject("in_progress", []OutputItem{}, nil)
	emitSSE(w, flusher, "response.created", streamResponseEvent{
		Type: "response.created", Response: skeleton, SequenceNumber: s.next(),
	})
	emitSSE(w, flusher, "response.in_progress", streamResponseEvent{
		Type: "response.in_progress", Response: skeleton, SequenceNumber: s.next(),
	})
}

// reasoningEncrypted returns an opaque encrypted_content token when the client
// requested reasoning.encrypted_content. The gateway is stateless and does not
// produce real OpenAI reasoning ciphertext; clients treat this as an opaque
// round-trip value (and the gateway ignores reasoning items on input), so a
// generated marker is sufficient to satisfy the schema.
func (s *streamState) reasoningEncrypted() *string {
	if !s.encReasoning {
		return nil
	}
	v := "gwenc_" + s.reasoningItemID
	return &v
}

// ensureReasoning opens the reasoning output item. For reasoning-model clients
// it is emitted eagerly (even with an empty summary) so the response always
// carries a reasoning item, which strict clients require.
func (s *streamState) ensureReasoning(w http.ResponseWriter, flusher http.Flusher) {
	if !s.emitReasoning || s.reasoningStarted {
		return
	}
	s.closeOpenItem(w, flusher)
	s.reasoningStarted = true
	s.reasoningItemID = generateReasoningID()
	s.reasoningIndex = s.nextOutputIndex
	s.nextOutputIndex++

	emitSSE(w, flusher, "response.output_item.added", streamOutputItemEvent{
		Type: "response.output_item.added", OutputIndex: s.reasoningIndex,
		Item: reasoningItem(s.reasoningItemID, "in_progress", nil, nil), SequenceNumber: s.next(),
	})
	s.openIndex = s.reasoningIndex
}

// emitReasoningDelta streams a reasoning summary text delta, opening the summary
// part lazily on first delta.
func (s *streamState) emitReasoningDelta(w http.ResponseWriter, flusher http.Flusher, delta string) {
	if delta == "" {
		return
	}
	s.ensureReasoning(w, flusher)
	if !s.reasoningPartOpen {
		s.reasoningPartOpen = true
		emitSSE(w, flusher, "response.reasoning_summary_part.added", streamReasoningPartEvent{
			Type: "response.reasoning_summary_part.added", ItemID: s.reasoningItemID,
			OutputIndex: s.reasoningIndex, SummaryIndex: 0,
			Part: summaryTextPart{Type: "summary_text", Text: ""}, SequenceNumber: s.next(),
		})
	}
	s.reasoningText += delta
	emitSSE(w, flusher, "response.reasoning_summary_text.delta", streamReasoningDeltaEvent{
		Type: "response.reasoning_summary_text.delta", ItemID: s.reasoningItemID,
		OutputIndex: s.reasoningIndex, SummaryIndex: 0, Delta: delta, SequenceNumber: s.next(),
	})
}

// reasoningSummary returns the accumulated summary parts (empty slice if none).
func (s *streamState) reasoningSummary() []summaryTextPart {
	if s.reasoningText == "" {
		return []summaryTextPart{}
	}
	return []summaryTextPart{{Type: "summary_text", Text: s.reasoningText}}
}

// ensureMessage lazily opens the assistant message item on first text output.
func (s *streamState) ensureMessage(w http.ResponseWriter, flusher http.Flusher) {
	if s.messageStarted {
		return
	}
	s.closeOpenItem(w, flusher)
	s.messageStarted = true
	s.messageItemID = generateItemID()
	s.messageOutputIndex = s.nextOutputIndex
	s.nextOutputIndex++

	item := messageItem(s.messageItemID, "in_progress", nil)
	emitSSE(w, flusher, "response.output_item.added", streamOutputItemEvent{
		Type: "response.output_item.added", OutputIndex: s.messageOutputIndex, Item: item, SequenceNumber: s.next(),
	})
	emitSSE(w, flusher, "response.content_part.added", streamContentPartEvent{
		Type: "response.content_part.added", ItemID: s.messageItemID,
		OutputIndex: s.messageOutputIndex, ContentIndex: 0, Part: textPart(""), SequenceNumber: s.next(),
	})
	s.openIndex = s.messageOutputIndex
}

// emitTextDelta accumulates and emits a response.output_text.delta event.
func (s *streamState) emitTextDelta(w http.ResponseWriter, flusher http.Flusher, delta string) {
	if delta == "" {
		return
	}
	s.ensureMessage(w, flusher)
	s.messageText += delta
	emitSSE(w, flusher, "response.output_text.delta", streamTextDeltaEvent{
		Type: "response.output_text.delta", ItemID: s.messageItemID,
		OutputIndex: s.messageOutputIndex, ContentIndex: 0, Delta: delta, SequenceNumber: s.next(),
	})
}

// openFunctionCall closes the current item and opens a new function_call item.
func (s *streamState) openFunctionCall(w http.ResponseWriter, flusher http.Flusher, key int, callID, name string) {
	s.closeOpenItem(w, flusher)
	e := &fcEntry{outputIndex: s.nextOutputIndex, itemID: generateFuncCallID(), callID: callID, name: name}
	s.nextOutputIndex++
	s.fcByKey[key] = e
	s.fcOrder = append(s.fcOrder, e)

	item := OutputItem{
		Type: "function_call", ID: e.itemID, CallID: e.callID, Name: e.name, Status: "in_progress",
	}
	emitSSE(w, flusher, "response.output_item.added", streamOutputItemEvent{
		Type: "response.output_item.added", OutputIndex: e.outputIndex, Item: item, SequenceNumber: s.next(),
	})
	s.openIndex = e.outputIndex
}

// appendFuncArgs accumulates and emits a function_call_arguments.delta event.
func (s *streamState) appendFuncArgs(w http.ResponseWriter, flusher http.Flusher, key int, delta string) {
	e := s.fcByKey[key]
	if e == nil || delta == "" {
		return
	}
	e.args += delta
	emitSSE(w, flusher, "response.function_call_arguments.delta", streamFuncArgsDeltaEvent{
		Type: "response.function_call_arguments.delta", ItemID: e.itemID,
		OutputIndex: e.outputIndex, Delta: delta, SequenceNumber: s.next(),
	})
}

func (s *streamState) fcByOutputIndex(idx int) *fcEntry {
	for _, e := range s.fcOrder {
		if e.outputIndex == idx {
			return e
		}
	}
	return nil
}

// closeOpenItem emits the done events for whatever item is currently open.
func (s *streamState) closeOpenItem(w http.ResponseWriter, flusher http.Flusher) {
	if s.openIndex < 0 {
		return
	}
	idx := s.openIndex
	s.openIndex = -1

	if s.reasoningStarted && !s.reasoningCompleted && idx == s.reasoningIndex {
		s.reasoningCompleted = true
		if s.reasoningPartOpen {
			emitSSE(w, flusher, "response.reasoning_summary_text.done", streamReasoningDoneEvent{
				Type: "response.reasoning_summary_text.done", ItemID: s.reasoningItemID,
				OutputIndex: idx, SummaryIndex: 0, Text: s.reasoningText, SequenceNumber: s.next(),
			})
			emitSSE(w, flusher, "response.reasoning_summary_part.done", streamReasoningPartEvent{
				Type: "response.reasoning_summary_part.done", ItemID: s.reasoningItemID,
				OutputIndex: idx, SummaryIndex: 0,
				Part: summaryTextPart{Type: "summary_text", Text: s.reasoningText}, SequenceNumber: s.next(),
			})
		}
		emitSSE(w, flusher, "response.output_item.done", streamOutputItemEvent{
			Type: "response.output_item.done", OutputIndex: idx, SequenceNumber: s.next(),
			Item: reasoningItem(s.reasoningItemID, "completed", s.reasoningSummary(), s.reasoningEncrypted()),
		})
		return
	}

	if s.messageStarted && idx == s.messageOutputIndex {
		emitSSE(w, flusher, "response.output_text.done", streamTextDoneEvent{
			Type: "response.output_text.done", ItemID: s.messageItemID,
			OutputIndex: idx, ContentIndex: 0, Text: s.messageText, SequenceNumber: s.next(),
		})
		emitSSE(w, flusher, "response.content_part.done", streamContentPartEvent{
			Type: "response.content_part.done", ItemID: s.messageItemID,
			OutputIndex: idx, ContentIndex: 0, Part: textPart(s.messageText), SequenceNumber: s.next(),
		})
		emitSSE(w, flusher, "response.output_item.done", streamOutputItemEvent{
			Type: "response.output_item.done", OutputIndex: idx, SequenceNumber: s.next(),
			Item: messageItem(s.messageItemID, "completed", []OutputContent{textPart(s.messageText)}),
		})
		return
	}

	if e := s.fcByOutputIndex(idx); e != nil {
		emitSSE(w, flusher, "response.function_call_arguments.done", streamFuncArgsDoneEvent{
			Type: "response.function_call_arguments.done", ItemID: e.itemID,
			OutputIndex: e.outputIndex, Name: e.name, Arguments: e.args, SequenceNumber: s.next(),
		})
		emitSSE(w, flusher, "response.output_item.done", streamOutputItemEvent{
			Type: "response.output_item.done", OutputIndex: e.outputIndex, SequenceNumber: s.next(),
			Item: OutputItem{
				Type: "function_call", ID: e.itemID, CallID: e.callID,
				Name: e.name, Arguments: e.args, Status: "completed",
			},
		})
	}
}

// finish closes the open item and emits response.completed.
func (s *streamState) finish(w http.ResponseWriter, flusher http.Flusher, usage billing.UsageInfo) {
	s.closeOpenItem(w, flusher)

	output := make([]OutputItem, 0, 2+len(s.fcOrder))
	if s.reasoningStarted {
		output = append(output, reasoningItem(s.reasoningItemID, "completed", s.reasoningSummary(), s.reasoningEncrypted()))
	}
	if s.messageStarted {
		output = append(output, messageItem(s.messageItemID, "completed", []OutputContent{textPart(s.messageText)}))
	}
	for _, e := range s.fcOrder {
		output = append(output, OutputItem{
			Type: "function_call", ID: e.itemID, CallID: e.callID,
			Name: e.name, Arguments: e.args, Status: "completed",
		})
	}

	ru := newResponseUsage(usage.PromptTokens, usage.CompletionTokens)
	emitSSE(w, flusher, "response.completed", streamResponseEvent{
		Type: "response.completed", Response: s.responseObject("completed", output, &ru), SequenceNumber: s.next(),
	})
}

// writeStreamHeaders writes the SSE response headers and returns the flusher.
func writeStreamHeaders(w http.ResponseWriter) (http.Flusher, bool) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"message": "Streaming not supported",
				"type":    "server_error",
				"code":    "no_flusher",
			},
		})
		return nil, false
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	return flusher, true
}

// emitSSE writes a single SSE event: event: {type}\ndata: {json}\n\n
func emitSSE(w http.ResponseWriter, flusher http.Flusher, eventType string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	if respTrace {
		slog.Info("trace.responses sse", "event", eventType, "data", string(data))
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(data))
	flusher.Flush()
}

// RelayStream translates an OpenAI Chat Completions SSE stream into Responses API SSE events.
func RelayStream(w http.ResponseWriter, resp *http.Response, model string, opts ...relayOptions) billing.UsageInfo {
	defer resp.Body.Close()

	flusher, ok := writeStreamHeaders(w)
	if !ok {
		return billing.UsageInfo{}
	}

	st := newStreamState(model, opts...)
	st.emitStart(w, flusher)
	// Reasoning-model clients (e.g. Cursor) require a reasoning output item even
	// when the upstream emits no reasoning tokens; open it eagerly so it is
	// always the first output item.
	st.ensureReasoning(w, flusher)
	var usage billing.UsageInfo

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
			break
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
					Reasoning        string `json:"reasoning"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens        int `json:"prompt_tokens"`
				CompletionTokens    int `json:"completion_tokens"`
				PromptTokensDetails *struct {
					CachedTokens int `json:"cached_tokens"`
				} `json:"prompt_tokens_details"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if chunk.Usage != nil {
			usage = billing.UsageInfo{
				PromptTokens:                chunk.Usage.PromptTokens,
				CompletionTokens:            chunk.Usage.CompletionTokens,
				CacheTokensIncludedInPrompt: true, // OpenAI format
			}
			if chunk.Usage.PromptTokensDetails != nil {
				usage.CacheReadTokens = chunk.Usage.PromptTokensDetails.CachedTokens
			}
		}

		if len(chunk.Choices) == 0 {
			continue
		}
		choice := chunk.Choices[0]

		if r := choice.Delta.ReasoningContent; r != "" {
			st.emitReasoningDelta(w, flusher, r)
		} else if r := choice.Delta.Reasoning; r != "" {
			st.emitReasoningDelta(w, flusher, r)
		}

		if choice.Delta.Content != "" {
			st.emitTextDelta(w, flusher, choice.Delta.Content)
		}

		for _, tc := range choice.Delta.ToolCalls {
			if tc.ID != "" {
				st.openFunctionCall(w, flusher, tc.Index, tc.ID, tc.Function.Name)
			}
			if tc.Function.Arguments != "" {
				st.appendFuncArgs(w, flusher, tc.Index, tc.Function.Arguments)
			}
		}
	}

	st.finish(w, flusher, usage)
	return usage
}

// RelayAnthropicStream converts an Anthropic SSE stream to Responses API SSE events.
func RelayAnthropicStream(w http.ResponseWriter, resp *http.Response, model string, opts ...relayOptions) billing.UsageInfo {
	defer resp.Body.Close()

	flusher, ok := writeStreamHeaders(w)
	if !ok {
		return billing.UsageInfo{}
	}

	st := newStreamState(model, opts...)
	st.emitStart(w, flusher)
	st.ensureReasoning(w, flusher)
	var usage billing.UsageInfo

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 10*1024*1024), 100*1024*1024) // 10MB initial, 100MB max
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "event: ") {
			continue
		}
		eventType := line[7:]

		if !scanner.Scan() {
			break
		}
		dataLine := scanner.Text()
		if !strings.HasPrefix(dataLine, "data: ") {
			continue
		}
		data := dataLine[6:]

		switch eventType {
		case "message_start":
			var evt struct {
				Message struct {
					Usage struct {
						InputTokens              int `json:"input_tokens"`
						CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
						CacheReadInputTokens     int `json:"cache_read_input_tokens"`
						CacheCreation            *struct {
							Ephemeral5mInputTokens int `json:"ephemeral_5m_input_tokens"`
							Ephemeral1hInputTokens int `json:"ephemeral_1h_input_tokens"`
						} `json:"cache_creation"`
						// Third-party compatibility (e.g. 4sapi): flat 5m/1h fields.
						ClaudeCacheCreation5mTokens int `json:"claude_cache_creation_5_m_tokens"`
						ClaudeCacheCreation1hTokens int `json:"claude_cache_creation_1_h_tokens"`
					} `json:"usage"`
				} `json:"message"`
			}
			if json.Unmarshal([]byte(data), &evt) == nil {
				usage.PromptTokens = evt.Message.Usage.InputTokens
				usage.CacheCreationTokens = evt.Message.Usage.CacheCreationInputTokens
				usage.CacheReadTokens = evt.Message.Usage.CacheReadInputTokens
				if evt.Message.Usage.CacheCreation != nil {
					usage.CacheCreation5mTokens = evt.Message.Usage.CacheCreation.Ephemeral5mInputTokens
					usage.CacheCreation1hTokens = evt.Message.Usage.CacheCreation.Ephemeral1hInputTokens
				}
				// Third-party compatibility (e.g. 4sapi): flat 5m/1h fields.
				if evt.Message.Usage.ClaudeCacheCreation5mTokens > 0 && usage.CacheCreation5mTokens == 0 {
					usage.CacheCreation5mTokens = evt.Message.Usage.ClaudeCacheCreation5mTokens
				}
				if evt.Message.Usage.ClaudeCacheCreation1hTokens > 0 && usage.CacheCreation1hTokens == 0 {
					usage.CacheCreation1hTokens = evt.Message.Usage.ClaudeCacheCreation1hTokens
				}
			}

		case "content_block_start":
			var evt struct {
				Index        int `json:"index"`
				ContentBlock struct {
					Type string `json:"type"`
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"content_block"`
			}
			if json.Unmarshal([]byte(data), &evt) != nil {
				continue
			}
			if evt.ContentBlock.Type == "tool_use" {
				st.openFunctionCall(w, flusher, evt.Index, evt.ContentBlock.ID, evt.ContentBlock.Name)
			}

		case "content_block_delta":
			var evt struct {
				Index int `json:"index"`
				Delta struct {
					Type        string `json:"type"`
					Text        string `json:"text"`
					Thinking    string `json:"thinking"`
					PartialJSON string `json:"partial_json"`
				} `json:"delta"`
			}
			if json.Unmarshal([]byte(data), &evt) != nil {
				continue
			}
			switch evt.Delta.Type {
			case "text_delta":
				st.emitTextDelta(w, flusher, evt.Delta.Text)
			case "thinking_delta":
				st.emitReasoningDelta(w, flusher, evt.Delta.Thinking)
			case "input_json_delta":
				st.appendFuncArgs(w, flusher, evt.Index, evt.Delta.PartialJSON)
			}

		case "message_delta":
			var evt struct {
				Usage struct {
					OutputTokens             int `json:"output_tokens"`
					CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
					CacheReadInputTokens     int `json:"cache_read_input_tokens"`
					CacheCreation            *struct {
						Ephemeral5mInputTokens int `json:"ephemeral_5m_input_tokens"`
						Ephemeral1hInputTokens int `json:"ephemeral_1h_input_tokens"`
					} `json:"cache_creation"`
					ClaudeCacheCreation5mTokens int `json:"claude_cache_creation_5_m_tokens"`
					ClaudeCacheCreation1hTokens int `json:"claude_cache_creation_1_h_tokens"`
				} `json:"usage"`
			}
			if json.Unmarshal([]byte(data), &evt) == nil {
				usage.CompletionTokens = evt.Usage.OutputTokens
				// 4sapi puts cache tokens in message_delta instead of message_start
				if evt.Usage.CacheCreationInputTokens > 0 && usage.CacheCreationTokens == 0 {
					usage.CacheCreationTokens = evt.Usage.CacheCreationInputTokens
				}
				if evt.Usage.CacheReadInputTokens > 0 && usage.CacheReadTokens == 0 {
					usage.CacheReadTokens = evt.Usage.CacheReadInputTokens
				}
				if evt.Usage.CacheCreation != nil {
					if evt.Usage.CacheCreation.Ephemeral5mInputTokens > 0 && usage.CacheCreation5mTokens == 0 {
						usage.CacheCreation5mTokens = evt.Usage.CacheCreation.Ephemeral5mInputTokens
					}
					if evt.Usage.CacheCreation.Ephemeral1hInputTokens > 0 && usage.CacheCreation1hTokens == 0 {
						usage.CacheCreation1hTokens = evt.Usage.CacheCreation.Ephemeral1hInputTokens
					}
				}
				if evt.Usage.ClaudeCacheCreation5mTokens > 0 && usage.CacheCreation5mTokens == 0 {
					usage.CacheCreation5mTokens = evt.Usage.ClaudeCacheCreation5mTokens
				}
				if evt.Usage.ClaudeCacheCreation1hTokens > 0 && usage.CacheCreation1hTokens == 0 {
					usage.CacheCreation1hTokens = evt.Usage.ClaudeCacheCreation1hTokens
				}
			}

		case "content_block_stop", "message_stop", "ping":
			// no-op; items are closed lazily on the next item or at finish
		}
	}

	st.finish(w, flusher, usage)
	return usage
}

// RelayResponsesPassthroughStream pipes upstream Responses API SSE events
// directly to the client, extracting usage from the final response.completed event.
func RelayResponsesPassthroughStream(w http.ResponseWriter, resp *http.Response, _ string) billing.UsageInfo {
	defer resp.Body.Close()
	flusher, ok := writeStreamHeaders(w)
	if !ok {
		return billing.UsageInfo{}
	}

	var usage billing.UsageInfo
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintf(w, "%s\n", line)
		flusher.Flush()
		if data, found := strings.CutPrefix(line, "data: "); found {
			if u := extractResponsesAPIUsageFromSSE(data); u.PromptTokens > 0 || u.CompletionTokens > 0 {
				usage = u
			}
		}
	}
	return usage
}

// extractResponsesAPIUsage extracts billing.UsageInfo from a Responses API
// JSON body (uses input_tokens/output_tokens, not prompt_tokens/completion_tokens).
func extractResponsesAPIUsage(body []byte) billing.UsageInfo {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return billing.UsageInfo{}
	}
	return parseResponsesUsageMap(m)
}

// extractResponsesAPIUsageFromSSE extracts usage from a response.completed SSE data payload:
// {"type":"response.completed","response":{"usage":{...}}}
func extractResponsesAPIUsageFromSSE(data string) billing.UsageInfo {
	var evt struct {
		Type     string `json:"type"`
		Response struct {
			Usage map[string]any `json:"usage"`
		} `json:"response"`
	}
	if err := json.Unmarshal([]byte(data), &evt); err != nil || evt.Type != "response.completed" {
		return billing.UsageInfo{}
	}
	return parseResponsesUsageMap(map[string]any{"usage": evt.Response.Usage})
}

func parseResponsesUsageMap(m map[string]any) billing.UsageInfo {
	usageObj, ok := m["usage"]
	if !ok {
		return billing.UsageInfo{}
	}
	usageMap, ok := usageObj.(map[string]any)
	if !ok {
		return billing.UsageInfo{}
	}
	var info billing.UsageInfo
	info.CacheTokensIncludedInPrompt = true
	if v, ok := usageMap["input_tokens"].(float64); ok {
		info.PromptTokens = int(v)
	}
	if v, ok := usageMap["output_tokens"].(float64); ok {
		info.CompletionTokens = int(v)
	}
	if details, ok := usageMap["input_tokens_details"].(map[string]any); ok {
		if cached, ok := details["cached_tokens"].(float64); ok {
			info.CacheReadTokens = int(cached)
		}
	}
	return info
}
