package huma

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	huma2 "github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	muxrouter "github.com/yunerou/niarb/pkg/mux-router"
)

type Router struct {
	config  muxrouter.Config
	mainMux *http.ServeMux
	api     huma2.API
	paths   map[string]*Group
}

type Group struct {
	router      *Router
	prefixRoute string

	lazyRegisters []func()
	serveOnce     sync.Once
}

func New(c muxrouter.Config) *Router {
	if c.ConvertError == nil {
		panic("muxrouter.Config.ConvertError is required")
	}
	return &Router{
		config:  c,
		mainMux: http.NewServeMux(),
		paths:   map[string]*Group{},
	}
}

func (r *Router) Create(prefix string) *Group {
	trimmed := strings.Trim(prefix, "/")
	if !muxrouter.IsValidPathPrefix(trimmed) {
		panic(fmt.Sprintf("invalid path prefix %q", prefix))
	}
	if g, ok := r.paths[trimmed]; ok {
		return g
	}
	g := &Group{router: r, prefixRoute: trimmed}
	r.paths[trimmed] = g
	return g
}

func (g *Group) runLazyRegister() {
	g.serveOnce.Do(func() {
		for _, f := range g.lazyRegisters {
			f()
		}
		g.lazyRegisters = nil
	})
}

func (r *Router) buildHumaConfig(enableDoc bool) huma2.Config {
	cfg := huma2.DefaultConfig(r.config.Doc.Title, r.config.Doc.Version)
	if r.config.Doc.OpenAPIPath != "" {
		cfg.OpenAPIPath = r.config.Doc.OpenAPIPath
	}
	if r.config.Doc.DocsPath != "" {
		cfg.DocsPath = r.config.Doc.DocsPath
	}
	if len(r.config.Formats) > 0 {
		formats := map[string]huma2.Format{}
		for _, rf := range r.config.Formats {
			marshal := rf.Formats.Marshal
			unmarshal := rf.Formats.Unmarshal
			hf := huma2.Format{
				Marshal:   func(w io.Writer, v any) error { return marshal(w, v) },
				Unmarshal: func(data []byte, v any) error { return unmarshal(bytes.NewReader(data), v) },
			}
			for _, h := range rf.Headers {
				formats[h] = hf
			}
		}
		cfg.Formats = formats
		if _, ok := formats[r.config.DefaultFormat]; ok && r.config.DefaultFormat != "" {
			cfg.DefaultFormat = r.config.DefaultFormat
		} else {
			cfg.DefaultFormat = muxrouter.JsonHeaders[0]
		}
	}
	if !enableDoc {
		cfg.DocsPath = ""
		cfg.OpenAPIPath = ""
		cfg.SchemasPath = ""
	}
	return cfg
}

func (r *Router) ExtractHandler(enableDoc bool) http.Handler {
	r.api = humago.New(r.mainMux, r.buildHumaConfig(enableDoc))
	for _, g := range r.paths {
		g.runLazyRegister()
	}
	return r.mainMux
}
