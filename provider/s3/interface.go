package s3provider

import (
	"context"
	"io"

	"github.com/yunerou/niarb/shared/aerror"
)

type PresignPostObjectOptions struct {
	Metadata map[string]string
	// max size of file in bytes
	MaxContentLength int64
	// min size of file in bytes
	MinContentLength int64
	// exact content type
	ContentTypePrefix *string
	ContentType       *string
}

type ObjectStorage interface {
	Upload(ctx context.Context, key string, body io.Reader, contentType string) aerror.AError
	Download(ctx context.Context, key string) (io.ReadCloser, aerror.AError)
	Delete(ctx context.Context, key string) aerror.AError
	List(ctx context.Context, prefix string) ([]string, aerror.AError)
	PresignPostObject(
		ctx context.Context,
		s3FilePath string,
		lifetimeSecs int64,
		opts *PresignPostObjectOptions,
	) (url string, values map[string]string, err aerror.AError)
	PresignGetObject(
		ctx context.Context,
		s3FilePath string,
		lifetimeSecs int64,
	) (url string, err aerror.AError)
	GetFileMetadata(
		ctx context.Context,
		s3FilePath string,
	) (metadata map[string]any, err aerror.AError)
}
