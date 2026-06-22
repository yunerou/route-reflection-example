package humapvd

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type Config struct {
	HumaConfig               huma.Config
	ConvertErrorToHumaSchema func(error) huma.StatusError
	CommonInfo               CommonInfo
}

func (c *Config) Validate() {
	if c.ConvertErrorToHumaSchema == nil {
		panic("ConvertErrorToHumaSchema is required")
	}
}

func NewCore(c *Config) CoreHuma {
	mainMux := http.NewServeMux()
	return &coreHuma{
		humaConfig:               c.HumaConfig,
		commonInfo:               c.CommonInfo,
		convertErrorToHumaSchema: c.ConvertErrorToHumaSchema,

		mainMux: mainMux,
	}
}

func RegisterRoute[ReqParamT, ReqBodyT, RespBodyT any](
	mux *GroupHuma,
	method string,
	path string,
	meta RouteMeta,
	handler TypedHandler[ReqParamT, ReqBodyT, RespBodyT],
	middleware Middleware,
) {
	lazyRegisterRoute(mux, method, path, meta, handler, middleware)
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
