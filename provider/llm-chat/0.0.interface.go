package llmchat

import (
	"context"

	"github.com/yunerou/niarb/shared/aerror"
)

// Provider is the consumer-facing chat completion interface.
type Provider interface {
	Complete(ctx context.Context, req CompleteRequest) (*CompleteResponse, aerror.AError)
}

// ResponseFormat constrains the output format. Type "json_object" requests
// strict JSON from the model (OpenAI-compatible). Leave nil for free text.
type ResponseFormat struct {
	Type string // "text" | "json_object"
}

// CompleteRequest is the input to a single chat completion call.
// All optional pointers (ResponseFormat, Temperature, MaxTokens) are omitted
// from the wire request when nil.
type CompleteRequest struct {
	System         string
	User           string
	ResponseFormat *ResponseFormat
	Temperature    *float32
	MaxTokens      *int
}

// CompleteResponse is the assistant message content. For json_object requests
// this is the raw JSON string; the caller is responsible for decoding.
type CompleteResponse struct {
	Content string
}
