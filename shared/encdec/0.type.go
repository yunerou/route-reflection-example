package encdec

import (
	"context"
	"io"

	"github.com/yunerou/niarb/shared/aerror"
)

type Encoder interface {
	Encode(ctx context.Context, w io.Writer, v any) aerror.AError
}

type Decoder interface {
	Decode(ctx context.Context, r io.Reader, v any) aerror.AError
}

type Marshaler interface {
	Marshal(ctx context.Context, v any) ([]byte, aerror.AError)
}

type Unmarshaler interface {
	Unmarshal(ctx context.Context, data []byte, v any) aerror.AError
}

type encoderFn func(ctx context.Context, w io.Writer, v any) aerror.AError

func (e encoderFn) Encode(ctx context.Context, w io.Writer, v any) aerror.AError {
	return e(ctx, w, v)
}

type decoderFn func(ctx context.Context, r io.Reader, v any) aerror.AError

func (d decoderFn) Decode(ctx context.Context, r io.Reader, v any) aerror.AError {
	return d(ctx, r, v)
}

type marshalerFn func(ctx context.Context, v any) ([]byte, aerror.AError)

func (m marshalerFn) Marshal(ctx context.Context, v any) ([]byte, aerror.AError) {
	return m(ctx, v)
}

type unmarshalerFn func(ctx context.Context, data []byte, v any) aerror.AError

func (u unmarshalerFn) Unmarshal(ctx context.Context, data []byte, v any) aerror.AError {
	return u(ctx, data, v)
}
