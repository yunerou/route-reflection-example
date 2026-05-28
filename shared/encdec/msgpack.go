package encdec

import (
	"bytes"
	"context"
	"io"

	"github.com/vmihailenco/msgpack/v5"
	"github.com/yunerou/niarb/shared/aerror"
)

func MsgpackEncoder(w io.Writer) Encoder {
	e := msgpack.NewEncoder(w)
	e.SetCustomStructTag("json")

	return encoderFn(func(ctx context.Context, v any) aerror.AError {
		err := e.Encode(v)
		if err != nil {
			return aerror.New(ctx, aerror.ErrEncoder, err)
		}
		return nil
	})
}

func MsgpackDecoder(r io.Reader) Decoder {
	d := msgpack.NewDecoder(r)
	d.SetCustomStructTag("json")

	return decoderFn(func(ctx context.Context, v any) aerror.AError {
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
		err := MsgpackEncoder(&buf).Encode(ctx, v)
		if err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	})
}

func MsgpackUnmarshaler() Unmarshaler {
	return unmarshalerFn(func(ctx context.Context, data []byte, v any) aerror.AError {
		return MsgpackDecoder(bytes.NewReader(data)).Decode(ctx, v)
	})
}
