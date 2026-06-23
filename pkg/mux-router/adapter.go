package muxrouter

import (
	"context"
	"net/http"
)

type Adapter interface {
	// RegisterRoute is called with type-erased handler and RouteTypeInfo for documentation.
	// Use RouteTypeInfo to build schemas; call handler with concrete instances created via reflection.
	RegisterRoute(
		method string,
		path string,
		meta RouteMeta,
		middleware Middleware,
		typeInfo RouteTypeInfo,
		handler func(ctx context.Context, reqParam, reqBody any) (any, error),
	)
	RegisterFormat(f []RegisterFormat)
	RegisterErrorConverter(converter func(error) StatusError)
	SetMux(mux http.Handler)
}
