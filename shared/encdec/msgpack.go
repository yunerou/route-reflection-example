package encdec

import (
	"bytes"
	"context"
	"io"

	"github.com/vmihailenco/msgpack/v5"
	"github.com/yunerou/niarb/shared/aerror"
)

func MsgpackEncoder() Encoder {
	return encoderFn(func(ctx context.Context, w io.Writer, v any) aerror.AError {
		e := msgpack.NewEncoder(w)
		e.SetCustomStructTag("json")

		err := e.Encode(v)
		if err != nil {
			return aerror.New(ctx, aerror.ErrEncoder, err)
		}
		return nil
	})
}

func MsgpackDecoder() Decoder {
	return decoderFn(func(ctx context.Context, r io.Reader, v any) aerror.AError {
		d := msgpack.NewDecoder(r)
		d.SetCustomStructTag("json")

		err := d.Decode(v)
		if err != nil {
			return aerror.New(ctx, aerror.ErrDecoder, err)
		}
		return nil
	})
}

func MsgpackMarshaler() Marshaler {
	return marshalerFn(func(ctx context.Context, v any) ([]byte, aerror.AError) {
		var buf bytes.Buffer
		err := MsgpackEncoder().Encode(ctx, &buf, v)
		if err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	})
}

func MsgpackUnmarshaler() Unmarshaler {
	return unmarshalerFn(func(ctx context.Context, data []byte, v any) aerror.AError {
		return MsgpackDecoder().Decode(ctx, bytes.NewReader(data), v)
	})
}
