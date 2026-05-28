package llmembedding

import (
	"context"

	"github.com/yunerou/niarb/shared/aerror"
)

type embeddingProvider struct {
	adapter Adapter
}

func New(adapter Adapter) EmbeddingProvider {
	return &embeddingProvider{adapter: adapter}
}

func (p *embeddingProvider) Embed(ctx context.Context, text string) ([]float32, aerror.AError) {
	return p.adapter.Embed(ctx, text)
}

func (p *embeddingProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, aerror.AError) {
	return p.adapter.EmbedBatch(ctx, texts)
}

func (p *embeddingProvider) Dimension() int {
	return p.adapter.Dimension()
}
