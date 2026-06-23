package muxrouter

import (
	"context"
	"net/http"
	"reflect"
)

type ParamSource string

const (
	SourcePath   ParamSource = "path"
	SourceQuery  ParamSource = "query"
	SourceHeader ParamSource = "header"
)

// TypedHandler is the generic handler shape every adapter registers.
type TypedHandler[ReqParamT, ReqBodyT, RespBodyT any] = func(ctx context.Context, reqParam ReqParamT, reqBody ReqBodyT) (RespBodyT, error)

type RouteMeta struct {
	Summary     string
	Description string
	Tags        []string
	Deprecated  bool
}

// RouteTypeInfo carries the concrete reflect.Type for each generic parameter.
type RouteTypeInfo struct {
	ReqParamType reflect.Type
	ReqBodyType  reflect.Type
	RespBodyType reflect.Type
}

type Middleware = func(http.Handler) http.Handler

func ChainMiddleware(middlewares ...Middleware) Middleware {
	return func(final http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

// CommonInfo documents service-wide metadata shared across routes.
type CommonInfo struct {
	ServiceName     string
	RequestHeaders  map[string]string
	ResponseHeaders map[string]string
}

// DocConfig configures documentation output. Only the huma adapter consumes it;
// the gomux adapter ignores it.
type DocConfig struct {
	Title       string
	Version     string
	DocsPath    string // e.g. "/docs"
	OpenAPIPath string // e.g. "/openapi.json"
}
