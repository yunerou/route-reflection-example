package llmembedding

import (
	"context"

	"github.com/yunerou/niarb/shared/aerror"
)

type Adapter interface {
	Embed(ctx context.Context, text string) ([]float32, aerror.AError)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, aerror.AError)
	Dimension() int
}
