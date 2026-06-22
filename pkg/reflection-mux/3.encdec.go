package reflectionmux

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reflect"
)

// Config from COreReflectionMux when creating PathReflectionMux
type EncoderDecoder interface {
	Encode(ctx context.Context, w io.Writer, v any) error
	Decode(ctx context.Context, r io.Reader, v any) error
}

func defaultWriteResponse(de EncoderDecoder, w http.ResponseWriter, r *http.Request, status int, body any) {
	w.WriteHeader(status)
	if body == nil {
		return
	}
	if err := de.Encode(r.Context(), w, body); err != nil {
		ctx := r.Context()
		slog.ErrorContext(ctx, "delivery: write response failed", slog.Any("err", err))
	}
}

func defaultWriteError(de EncoderDecoder, convertErrorSchema func(error) (int, any), w http.ResponseWriter, r *http.Request, err error) {
	status, errBody := convertErrorSchema(err)
	defaultWriteResponse(de, w, r, status, errBody)
	return
}

func defaultParseRequest[ReqParamT, ReqBodyT any, ErrorT error](de EncoderDecoder, r *http.Request) (reqParam ReqParamT, reqBody ReqBodyT, err ErrorT) {
	var zeroError ErrorT

	pv := reflect.ValueOf(&reqParam).Elem()
	pt := pv.Type()
	if pt.Kind() == reflect.Pointer {
		pv.Set(reflect.New(pt.Elem()))
		pv = pv.Elem()
		pt = pt.Elem()
	}
	if pt.Kind() == reflect.Struct {
		query := r.URL.Query()
		for i := 0; i < pt.NumField(); i++ {
			field := pt.Field(i)
			if field.PkgPath != "" {
				continue
			}

			var (
				raw   string
				found bool
			)
			if name, ok := fieldTagName(field, string(SourcePath)); ok {
				raw, found = r.PathValue(name), true
			} else if name, ok := fieldTagName(field, string(SourceQuery)); ok {
				raw, found = query.Get(name), query.Has(name)
			} else if name, ok := fieldTagName(field, string(SourceHeader)); ok {
				raw = r.Header.Get(name)
				found = raw != ""
			}
			if !found {
				continue
			}

			errParse := setFieldValue(pv.Field(i), raw)
			if errParse != nil {
				panic(fmt.Sprintf("failed to parse parameter %q: %v", field.Name, errParse))
			}
		}
	}

	if r.Body != nil && r.ContentLength != 0 {
		ctx := r.Context()
		de.Decode(ctx, r.Body, &reqBody)
	}

	return reqParam, reqBody, zeroError
}
