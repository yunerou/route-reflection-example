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

type MuxRouter interface {
	Create(PathPrefix string) *GroupRouter
	ExtractHandler() http.Handler
}

var (
	_ MuxRouter = (*coreMuxRouter)(nil)
)

type TypedHandler[ReqParamT, ReqBodyT, RespBodyT any] = func(ctx context.Context, reqParam ReqParamT, reqBody ReqBodyT) (RespBodyT, error)

type RouteMeta struct {
	Summary     string
	Description string
	Tags        []string
	Deprecated  bool
}

// RouteTypeInfo carries the concrete reflect.Type for each generic parameter.
// Adapters use this to build documentation schemas without needing generics.
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
