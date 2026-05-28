package llmchat

import (
	"context"

	"github.com/yunerou/niarb/shared/aerror"
)

type chatProvider struct {
	adapter Adapter
}

// New wraps an Adapter into the consumer-facing Provider.
func New(adapter Adapter) Provider {
	return &chatProvider{adapter: adapter}
}

func (p *chatProvider) Complete(ctx context.Context, req CompleteRequest) (*CompleteResponse, aerror.AError) {
	return p.adapter.Complete(ctx, req)
}
