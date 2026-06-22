package encdec

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/yunerou/niarb/shared/aerror"
)

func JSONEncoder() Encoder {
	return encoderFn(func(ctx context.Context, w io.Writer, v any) aerror.AError {
		err := json.NewEncoder(w).Encode(v)
		if err != nil {
			return aerror.New(ctx, aerror.ErrEncoder, err)
		}
		return nil
	})
}

func JSONDecoder() Decoder {
	return decoderFn(func(ctx context.Context, r io.Reader, v any) aerror.AError {
		err := json.NewDecoder(r).Decode(v)
		if err != nil {
			return aerror.New(ctx, aerror.ErrDecoder, err)
		}
		return nil
	})
}

func JSONMarshaler() Marshaler {
	return marshalerFn(func(ctx context.Context, v any) ([]byte, aerror.AError) {
		var buf bytes.Buffer
		err := JSONEncoder().Encode(ctx, &buf, v)
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
		return JSONDecoder().Decode(ctx, bytes.NewReader(data), v)
	})
}
