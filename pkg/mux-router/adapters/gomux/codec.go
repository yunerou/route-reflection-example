package gomux

import (
	"log/slog"
	"mime"
	"net/http"

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

// parseParams fills a fresh P from path/query/header values using struct tags,
// backed by the shared muxrouter.BindParams.
func parseParams[P any](r *http.Request) (P, error) {
	query := r.URL.Query()
	return muxrouter.BindParams[P](func(source muxrouter.ParamSource, name string) (string, bool) {
		switch source {
		case muxrouter.SourcePath:
			return r.PathValue(name), true
		case muxrouter.SourceQuery:
			return query.Get(name), query.Has(name)
		case muxrouter.SourceHeader:
			v := r.Header.Get(name)
			return v, v != ""
		default:
			return "", false
		}
	})
}
