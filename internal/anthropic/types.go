package anthropic

import "encoding/json"

// --- Request types ---

// MessagesRequest is the Anthropic Messages API request body.
type MessagesRequest struct {
	Model         string            `json:"model"`
	MaxTokens     int               `json:"max_tokens"`
	System        json.RawMessage   `json:"system,omitempty"`
	Messages      []Message         `json:"messages"`
	Stream        bool              `json:"stream,omitempty"`
	Temperature   *float64          `json:"temperature,omitempty"`
	TopP          *float64          `json:"top_p,omitempty"`
	TopK          *int              `json:"top_k,omitempty"`
	Tools         []Tool            `json:"tools,omitempty"`
	ToolChoice    json.RawMessage   `json:"tool_choice,omitempty"`
	StopSequences []string          `json:"stop_sequences,omitempty"`
	Metadata      map[string]any    `json:"metadata,omitempty"`
}

// Message is a single message in the Anthropic conversation.
type Message struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// ContentBlock is one element of a multi-part content array.
type ContentBlock struct {
	Type string `json:"type"`

	// type=text
	Text string `json:"text,omitempty"`

	// type=image
	Source *ImageSource `json:"source,omitempty"`

	// type=tool_use
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`

	// type=tool_result
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`

	// type=thinking — extended-thinking block replayed by the client.
	// Signature is the opaque token Anthropic models require to verify
	// that the thinking block was not tampered with. We capture it here
	// so we can observe (and eventually carry) Gemini's thoughtSignature
	// through this same block on the round trip.
	Thinking  string `json:"thinking,omitempty"`
	Signature string `json:"signature,omitempty"`

	// type=redacted_thinking — opaque thinking block. data is base64.
	Data string `json:"data,omitempty"`
}

// ImageSource describes a base64-encoded image.
type ImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// Tool describes a tool available to the model.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema,omitempty"`
}

// --- Response types ---

// MessagesResponse is the Anthropic Messages API response body.
type MessagesResponse struct {
	ID           string          `json:"id"`
	Type         string          `json:"type"`
	Role         string          `json:"role"`
	Content      []ResponseBlock `json:"content"`
	Model        string          `json:"model"`
	StopReason   string          `json:"stop_reason"`
	StopSequence *string         `json:"stop_sequence"`
	Usage        Usage           `json:"usage"`
}

// ResponseBlock is one element in the response content array.
type ResponseBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`

	// type=thinking
	Thinking  string `json:"thinking,omitempty"`
	Signature string `json:"signature,omitempty"`

	// type=redacted_thinking
	Data string `json:"data,omitempty"`
}

// Usage holds token usage for the response.
type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
	// CacheCreation contains the TTL-level breakdown of cache creation tokens.
	CacheCreation *CacheCreationBreakdown `json:"cache_creation,omitempty"`
	// Third-party compatibility (e.g. 4sapi): flat 5m/1h cache creation fields.
	ClaudeCacheCreation5mTokens int `json:"claude_cache_creation_5_m_tokens,omitempty"`
	ClaudeCacheCreation1hTokens int `json:"claude_cache_creation_1_h_tokens,omitempty"`
}

// CacheCreationBreakdown contains ephemeral cache creation token counts by TTL.
type CacheCreationBreakdown struct {
	Ephemeral5mInputTokens int `json:"ephemeral_5m_input_tokens"`
	Ephemeral1hInputTokens int `json:"ephemeral_1h_input_tokens"`
}

// --- Streaming event types ---

// MessageStartEvent is the first event in a stream.
type MessageStartEvent struct {
	Type    string           `json:"type"`
	Message MessagesResponse `json:"message"`
}

// ContentBlockStartEvent signals the start of a new content block.
//
// ContentBlock is `any` instead of ResponseBlock so the text-block sender
// can emit `"text":""` without omitempty swallowing the field. Anthropic
// clients require content_block.text to be present (even empty) for
// type=text blocks. Tool-use blocks use ResponseBlock directly, which
// omits irrelevant fields normally.
type ContentBlockStartEvent struct {
	Type         string `json:"type"`
	Index        int    `json:"index"`
	ContentBlock any    `json:"content_block"`
}

// ContentBlockDeltaEvent contains incremental content.
type ContentBlockDeltaEvent struct {
	Type  string     `json:"type"`
	Index int        `json:"index"`
	Delta BlockDelta `json:"delta"`
}

// BlockDelta is the delta payload inside a content_block_delta event.
type BlockDelta struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	Thinking    string `json:"thinking,omitempty"`
	Signature   string `json:"signature,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
}

// ContentBlockStopEvent signals the end of a content block.
type ContentBlockStopEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}

// MessageDeltaEvent carries the final stop_reason and usage.
type MessageDeltaEvent struct {
	Type  string            `json:"type"`
	Delta MessageDeltaBody  `json:"delta"`
	Usage MessageDeltaUsage `json:"usage"`
}

// MessageDeltaBody contains the stop_reason in a message_delta event.
type MessageDeltaBody struct {
	StopReason   string  `json:"stop_reason"`
	StopSequence *string `json:"stop_sequence"`
}

// MessageDeltaUsage contains output_tokens in a message_delta event.
type MessageDeltaUsage struct {
	OutputTokens int `json:"output_tokens"`
}

// MessageStopEvent signals the end of the message stream.
type MessageStopEvent struct {
	Type string `json:"type"`
}
