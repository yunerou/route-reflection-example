package muxrouter

import (
	"net/http"
)

type Config struct {
	Core Adapter

	RegisterFormat       []RegisterFormat
	ConvertErrorToSchema func(error) StatusError
}

func NewCoreMuxRouter(config Config) *coreMuxRouter {
	if config.Core == nil {
		panic("Core adapter is required in Config")
	}

	mainMux := http.NewServeMux()
	config.Core.SetMux(mainMux)

	return &coreMuxRouter{
		adapter: config.Core,
		mainMux: mainMux,
	}
}

func RegisterRoute[ReqParamT, ReqBodyT, RespBodyT any](
	mux *GroupRouter,
	method string,
	path string,
	meta RouteMeta,
	handler TypedHandler[ReqParamT, ReqBodyT, RespBodyT],
	middleware Middleware,
) {
	mux.lazyRegisters = append(mux.lazyRegisters, func() {
		registerRoute(mux, method, path, meta, handler, middleware)
	})
}
