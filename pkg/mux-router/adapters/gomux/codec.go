package gomux

import (
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"reflect"
	"strconv"

	muxrouter "github.com/yunerou/niarb/pkg/mux-router"
)

// codec resolves a muxrouter.Format from request/response content types.
type codec struct {
	byType map[string]muxrouter.Format
	def    muxrouter.Format // default when nothing matches
}

func newCodec(formats []muxrouter.RegisterFormat) *codec {
	c := &codec{byType: map[string]muxrouter.Format{}}
	for _, rf := range formats {
		for _, h := range rf.Headers {
			c.byType[h] = rf.Formats
		}
	}
	// Default: first registered JSON header, else built-in sonic JSON.
	if f, ok := c.byType[muxrouter.JsonHeaders[0]]; ok {
		c.def = f
	} else {
		c.def = muxrouter.JsonSonicFormat
	}
	return c
}

func (c *codec) pick(contentType string) muxrouter.Format {
	if contentType == "" {
		return c.def
	}
	if mt, _, err := mime.ParseMediaType(contentType); err == nil {
		if f, ok := c.byType[mt]; ok {
			return f
		}
	}
	return c.def
}

func (c *codec) decodeBody(r *http.Request, v any) error {
	if r.Body == nil || r.ContentLength == 0 {
		return nil
	}
	return c.pick(r.Header.Get("Content-Type")).Unmarshal(r.Body, v)
}

func (c *codec) encodeResponse(w http.ResponseWriter, r *http.Request, status int, body any) {
	f := c.pick(r.Header.Get("Accept"))
	w.WriteHeader(status)
	if body == nil {
		return
	}
	if err := f.Marshal(w, body); err != nil {
		slog.ErrorContext(r.Context(), "gomux: write response failed", slog.Any("err", err))
	}
}

// parseParams fills a fresh P from path/query/header values using struct tags.
func parseParams[P any](r *http.Request) (P, error) {
	var param P
	pv := reflect.ValueOf(&param).Elem()
	pt := pv.Type()
	if pt.Kind() == reflect.Pointer {
		pv.Set(reflect.New(pt.Elem()))
		pv = pv.Elem()
		pt = pt.Elem()
	}
	if pt.Kind() != reflect.Struct {
		return param, nil
	}
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
		if name, ok := muxrouter.FieldTagName(field, string(muxrouter.SourcePath)); ok {
			raw, found = r.PathValue(name), true
		} else if name, ok := muxrouter.FieldTagName(field, string(muxrouter.SourceQuery)); ok {
			raw, found = query.Get(name), query.Has(name)
		} else if name, ok := muxrouter.FieldTagName(field, string(muxrouter.SourceHeader)); ok {
			raw = r.Header.Get(name)
			found = raw != ""
		}
		if !found {
			continue
		}
		if err := setFieldValue(pv.Field(i), raw); err != nil {
			return param, fmt.Errorf("parse parameter %q: %w", field.Name, err)
		}
	}
	return param, nil
}

func setFieldValue(v reflect.Value, raw string) error {
	switch v.Kind() {
	case reflect.String:
		v.SetString(raw)
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		v.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(raw, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetFloat(f)
	default:
		return fmt.Errorf("unsupported field type %s", v.Type())
	}
	return nil
}
