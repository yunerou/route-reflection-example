package llmchat

import (
	"context"

	"github.com/yunerou/niarb/shared/aerror"
)

// Adapter is the backend contract implemented by concrete providers
// (OpenAI, Ollama, vLLM, ...). The core Provider delegates to it.
type Adapter interface {
	Complete(ctx context.Context, req CompleteRequest) (*CompleteResponse, aerror.AError)
}
