package encdec

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/yunerou/niarb/shared/aerror"
)

func JSONEncoder(w io.Writer) Encoder {
	e := json.NewEncoder(w)
	return encoderFn(func(ctx context.Context, v any) aerror.AError {
		err := e.Encode(v)
		if err != nil {
			return aerror.New(ctx, aerror.ErrEncoder, err)
		}
		return nil
	})
}

func JSONDecoder(r io.Reader) Decoder {
	d := json.NewDecoder(r)
	return decoderFn(func(ctx context.Context, v any) aerror.AError {
		err := d.Decode(v)
		if err != nil {
			return aerror.New(ctx, aerror.ErrDecoder, err)
		}
		return nil
	})
}

func JSONMarshaler() Marshaler {
	return marshalerFn(func(ctx context.Context, v any) ([]byte, aerror.AError) {
		var buf bytes.Buffer
		err := JSONEncoder(&buf).Encode(ctx, v)
		if err != nil {
			return nil, err
		}

		data := buf.Bytes()
		if len(data) > 0 && data[len(data)-1] == '\n' {
			data = data[:len(data)-1]
		}
		return data, nil
	})
}

func JSONUnmarshaler() Unmarshaler {
	return unmarshalerFn(func(ctx context.Context, data []byte, v any) aerror.AError {
		return JSONDecoder(bytes.NewReader(data)).Decode(ctx, v)
	})
}
