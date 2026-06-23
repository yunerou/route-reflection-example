package gomux

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	muxrouter "github.com/yunerou/niarb/pkg/mux-router"
)

type Router struct {
	codec        *codec
	convertError func(error) muxrouter.StatusError
	commonInfo   muxrouter.CommonInfo
	paths        map[string]*Group
}

type Group struct {
	router      *Router
	prefixRoute string

	lazyRegisters []func()
	serveOnce     sync.Once
	handlers      []routeHandler
	docs          []routeDoc
}

type routeHandler struct {
	pattern string
	h       http.Handler
}

// routeDoc is the lightweight doc record exposed at /__reflection.
type routeDoc struct {
	Method             string              `json:"method"`
	Path               string              `json:"path"`
	Meta               muxrouter.RouteMeta `json:"meta"`
	RequestBodySchema  json.RawMessage     `json:"requestBodySchema"`
	ResponseBodySchema json.RawMessage     `json:"responseBodySchema"`
}

func New(c muxrouter.Config) *Router {
	if c.ConvertError == nil {
		panic("muxrouter.Config.ConvertError is required")
	}
	return &Router{
		codec:        newCodec(c.Formats),
		convertError: c.ConvertError,
		commonInfo:   c.CommonInfo,
		paths:        map[string]*Group{},
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

func (r *Router) ExtractHandler(enableDoc bool) http.Handler {
	mux := http.NewServeMux()
	allDocs := []routeDoc{}
	const reflectionPattern = "GET /__reflection"
	seen := map[string]struct{}{}
	if enableDoc {
		seen[reflectionPattern] = struct{}{}
	}

	for _, g := range r.paths {
		g.runLazyRegister()
		for _, h := range g.handlers {
			if _, dup := seen[h.pattern]; dup {
				panic(fmt.Sprintf("duplicate route pattern %q", h.pattern))
			}
			seen[h.pattern] = struct{}{}
			mux.Handle(h.pattern, h.h)
		}
		allDocs = append(allDocs, g.docs...)
	}

	if enableDoc {
		mux.HandleFunc(reflectionPattern, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"common": r.commonInfo,
				"routes": allDocs,
			})
		})
	}
	return mux
}
