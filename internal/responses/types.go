package responses

import "encoding/json"

// ---------- Request types ----------

// CreateRequest represents an OpenAI Responses API create request.
type CreateRequest struct {
	Model              string            `json:"model"`
	Input              json.RawMessage   `json:"input"`
	Instructions       *string           `json:"instructions,omitempty"`
	Tools              []Tool            `json:"tools,omitempty"`
	ToolChoice         json.RawMessage   `json:"tool_choice,omitempty"`
	Temperature        *float64          `json:"temperature,omitempty"`
	TopP               *float64          `json:"top_p,omitempty"`
	MaxOutputTokens    *int              `json:"max_output_tokens,omitempty"`
	FrequencyPenalty   *float64          `json:"frequency_penalty,omitempty"`
	PresencePenalty    *float64          `json:"presence_penalty,omitempty"`
	User               string            `json:"user,omitempty"`
	Stream             bool              `json:"stream,omitempty"`
	Background         bool              `json:"background,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	PreviousResponseID string            `json:"previous_response_id,omitempty"`
	// Reasoning is present when the client treats the model as a reasoning model
	// (e.g. Cursor sends {"effort":"medium","summary":"auto"} for gpt-5.x). When
	// set, the response MUST contain a reasoning output item or strict clients
	// reject the whole turn.
	Reasoning json.RawMessage `json:"reasoning,omitempty"`
	// Include carries response.include flags such as
	// "reasoning.encrypted_content" (stateless ZDR reasoning round-trip).
	Include []string `json:"include,omitempty"`
}

// InputItem represents a single input item (message, function_call,
// function_call_output, or reasoning).
//
// Output is json.RawMessage because the Responses API allows a
// function_call_output's "output" to be either a plain string or an array of
// content blocks (e.g. image tool results). A fixed string type would make the
// whole []InputItem unmarshal fail when a client sends the array form.
type InputItem struct {
	Type      string          `json:"type"`
	Role      string          `json:"role,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	CallID    string          `json:"call_id,omitempty"`
	Output    json.RawMessage `json:"output,omitempty"`
	Name      string          `json:"name,omitempty"`
	Arguments string          `json:"arguments,omitempty"`
	ID        string          `json:"id,omitempty"`
	Tools     json.RawMessage `json:"tools,omitempty"`
}

// ContentPart represents a content part within an input item.
// ImageURL is imageURLField because the Responses API uses a plain string while
// Chat Completions clients may send an object {"url": "..."}; both must parse.
type ContentPart struct {
	Type     string        `json:"type"`
	Text     string        `json:"text,omitempty"`
	ImageURL imageURLField `json:"image_url,omitempty"`
}

// imageURLField accepts either a JSON string ("https://...") or an object
// ({"url": "https://..."}) and exposes the URL as a string.
type imageURLField string

func (u *imageURLField) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*u = imageURLField(s)
		return nil
	}
	var obj struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	*u = imageURLField(obj.URL)
	return nil
}

// Tool represents a tool definition in the request.
type Tool struct {
	Type     string          `json:"type"`
	Function *FunctionDef    `json:"function,omitempty"`
	Extra    map[string]any  `json:"-"`
}

func (t *Tool) UnmarshalJSON(data []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if tp, ok := raw["type"].(string); ok {
		t.Type = tp
	}
	delete(raw, "type")

	if fn, ok := raw["function"]; ok {
		fnBytes, _ := json.Marshal(fn)
		var fd FunctionDef
		if err := json.Unmarshal(fnBytes, &fd); err == nil {
			t.Function = &fd
		}
		delete(raw, "function")
	}

	if len(raw) > 0 {
		t.Extra = raw
	}
	return nil
}

// FunctionDef defines a function tool.
type FunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// ---------- Response types ----------

// Response represents an OpenAI Responses API response object.
//
// All fields are emitted (no omitempty) because strict clients — notably the
// Cursor Agent — deserialize this into a typed schema and reject the whole
// stream if expected keys are missing. The shape mirrors OpenAI's real
// response object: nullable fields are emitted as JSON null, and `usage` is
// null until the response completes.
type Response struct {
	ID                 string         `json:"id"`
	Object             string         `json:"object"`
	CreatedAt          int64          `json:"created_at"`
	Status             string         `json:"status"`
	Background         bool           `json:"background"`
	Error              any            `json:"error"`
	IncompleteDetails  any            `json:"incomplete_details"`
	Instructions       any            `json:"instructions"`
	MaxOutputTokens    any            `json:"max_output_tokens"`
	MaxToolCalls       any            `json:"max_tool_calls"`
	Model              string         `json:"model"`
	Output             []OutputItem   `json:"output"`
	ParallelToolCalls  bool           `json:"parallel_tool_calls"`
	PreviousResponseID any            `json:"previous_response_id"`
	Reasoning          any            `json:"reasoning"`
	Store              bool           `json:"store"`
	Temperature        any            `json:"temperature"`
	Text               any            `json:"text"`
	ToolChoice         any            `json:"tool_choice"`
	Tools              []any          `json:"tools"`
	TopP               any            `json:"top_p"`
	Truncation         any            `json:"truncation"`
	Usage              *ResponseUsage `json:"usage"`
	User               any            `json:"user"`
	Metadata           map[string]any `json:"metadata"`
}

// OutputItem represents an output item (message, function_call, or reasoning).
//
// Content/Summary are pointers so a message item emits "content":[] (present but
// empty) and a reasoning item emits "summary":[] while a function_call item
// omits both keys entirely — matching OpenAI exactly.
type OutputItem struct {
	Type             string             `json:"type"`
	ID               string             `json:"id"`
	Role             string             `json:"role,omitempty"`
	Content          *[]OutputContent   `json:"content,omitempty"`
	Status           string             `json:"status,omitempty"`
	CallID           string             `json:"call_id,omitempty"`
	Name             string             `json:"name,omitempty"`
	Arguments        string             `json:"arguments,omitempty"`
	Summary          *[]summaryTextPart `json:"summary,omitempty"`
	EncryptedContent *string            `json:"encrypted_content,omitempty"`
}

// summaryTextPart is one reasoning summary part (type "summary_text").
type summaryTextPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// reasoningItem builds a reasoning output item with summary always present
// (an empty, non-nil array while streaming). enc, when non-nil, sets
// encrypted_content for stateless reasoning round-trip.
func reasoningItem(id, status string, summary []summaryTextPart, enc *string) OutputItem {
	if summary == nil {
		summary = []summaryTextPart{}
	}
	s := summary
	item := OutputItem{Type: "reasoning", ID: id, Summary: &s, Status: status}
	item.EncryptedContent = enc
	return item
}

// newResponse builds a spec-shaped response object with all of OpenAI's
// fields populated (nullable ones as null, sensible defaults otherwise).
// Usage is left nil; callers set it to a ResponseUsage on completion.
func newResponse(id, model string, createdAt int64, status string, output []OutputItem) Response {
	if output == nil {
		output = []OutputItem{}
	}
	return Response{
		ID:                 id,
		Object:             "response",
		CreatedAt:          createdAt,
		Status:             status,
		Background:         false,
		Error:              nil,
		IncompleteDetails:  nil,
		Instructions:       nil,
		MaxOutputTokens:    nil,
		MaxToolCalls:       nil,
		Model:              model,
		Output:             output,
		ParallelToolCalls:  true,
		PreviousResponseID: nil,
		Reasoning:          map[string]any{"effort": nil, "summary": nil},
		Store:              false,
		Temperature:        1.0,
		Text:               map[string]any{"format": map[string]any{"type": "text"}},
		ToolChoice:         "auto",
		Tools:              []any{},
		TopP:               1.0,
		Truncation:         "disabled",
		Usage:              nil,
		User:               nil,
		Metadata:           map[string]any{},
	}
}

// messageItem builds an assistant message output item with content always
// present (an empty, non-nil array while streaming).
func messageItem(id, status string, content []OutputContent) OutputItem {
	if content == nil {
		content = []OutputContent{}
	}
	c := content
	return OutputItem{Type: "message", ID: id, Role: "assistant", Content: &c, Status: status}
}

// OutputContent represents a content block in an output item. Text has no
// omitempty: the Responses API always includes a "text" field on output_text
// parts (empty string while streaming), and strict clients reject a part that
// is missing it.
type OutputContent struct {
	Type        string `json:"type"`
	Text        string `json:"text"`
	Annotations []any  `json:"annotations"`
}

// ResponseUsage holds token usage in Responses API format, including the
// *_tokens_details sub-objects that OpenAI always sends.
type ResponseUsage struct {
	InputTokens         int                 `json:"input_tokens"`
	InputTokensDetails  inputTokensDetails  `json:"input_tokens_details"`
	OutputTokens        int                 `json:"output_tokens"`
	OutputTokensDetails outputTokensDetails `json:"output_tokens_details"`
	TotalTokens         int                 `json:"total_tokens"`
}

type inputTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

type outputTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"`
}

// newResponseUsage builds a spec-shaped usage object from token counts.
func newResponseUsage(inputTokens, outputTokens int) ResponseUsage {
	return ResponseUsage{
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
	}
}

// ---------- Streaming event types ----------

// StreamEvent represents a streaming SSE event for the Responses API.
type StreamEvent struct {
	Type string `json:"type"`
	// The payload varies by event type; we use a generic map or embed specific structs.
}

// streamResponseEvent wraps a full Response for response.created / response.completed etc.
type streamResponseEvent struct {
	Type           string   `json:"type"`
	Response       Response `json:"response"`
	SequenceNumber int      `json:"sequence_number"`
}

// streamOutputItemEvent is emitted for response.output_item.added / .done.
// The item carries its own id, so no separate item_id field is needed here.
type streamOutputItemEvent struct {
	Type           string     `json:"type"`
	OutputIndex    int        `json:"output_index"`
	Item           OutputItem `json:"item"`
	SequenceNumber int        `json:"sequence_number"`
}

// streamContentPartEvent is emitted for response.content_part.added / .done.
type streamContentPartEvent struct {
	Type           string        `json:"type"`
	ItemID         string        `json:"item_id"`
	OutputIndex    int           `json:"output_index"`
	ContentIndex   int           `json:"content_index"`
	Part           OutputContent `json:"part"`
	SequenceNumber int           `json:"sequence_number"`
}

// streamTextDeltaEvent is emitted for response.output_text.delta.
type streamTextDeltaEvent struct {
	Type           string `json:"type"`
	ItemID         string `json:"item_id"`
	OutputIndex    int    `json:"output_index"`
	ContentIndex   int    `json:"content_index"`
	Delta          string `json:"delta"`
	SequenceNumber int    `json:"sequence_number"`
}

// streamTextDoneEvent is emitted for response.output_text.done.
type streamTextDoneEvent struct {
	Type           string `json:"type"`
	ItemID         string `json:"item_id"`
	OutputIndex    int    `json:"output_index"`
	ContentIndex   int    `json:"content_index"`
	Text           string `json:"text"`
	SequenceNumber int    `json:"sequence_number"`
}

// streamReasoningPartEvent is emitted for
// response.reasoning_summary_part.added / .done.
type streamReasoningPartEvent struct {
	Type           string          `json:"type"`
	ItemID         string          `json:"item_id"`
	OutputIndex    int             `json:"output_index"`
	SummaryIndex   int             `json:"summary_index"`
	Part           summaryTextPart `json:"part"`
	SequenceNumber int             `json:"sequence_number"`
}

// streamReasoningDeltaEvent is emitted for response.reasoning_summary_text.delta.
type streamReasoningDeltaEvent struct {
	Type           string `json:"type"`
	ItemID         string `json:"item_id"`
	OutputIndex    int    `json:"output_index"`
	SummaryIndex   int    `json:"summary_index"`
	Delta          string `json:"delta"`
	SequenceNumber int    `json:"sequence_number"`
}

// streamReasoningDoneEvent is emitted for response.reasoning_summary_text.done.
type streamReasoningDoneEvent struct {
	Type           string `json:"type"`
	ItemID         string `json:"item_id"`
	OutputIndex    int    `json:"output_index"`
	SummaryIndex   int    `json:"summary_index"`
	Text           string `json:"text"`
	SequenceNumber int    `json:"sequence_number"`
}

// streamFuncArgsDeltaEvent is emitted for response.function_call_arguments.delta.
type streamFuncArgsDeltaEvent struct {
	Type           string `json:"type"`
	ItemID         string `json:"item_id"`
	OutputIndex    int    `json:"output_index"`
	Delta          string `json:"delta"`
	SequenceNumber int    `json:"sequence_number"`
}

// streamFuncArgsDoneEvent is emitted for response.function_call_arguments.done.
type streamFuncArgsDoneEvent struct {
	Type           string `json:"type"`
	ItemID         string `json:"item_id"`
	OutputIndex    int    `json:"output_index"`
	Name           string `json:"name"`
	Arguments      string `json:"arguments"`
	SequenceNumber int    `json:"sequence_number"`
}
