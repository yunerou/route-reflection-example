package reflectionmux

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Config struct {
	EncoderDecoder     EncoderDecoder
	ConvertErrorSchema func(error) (httpStatus int, body any)

	CommonInfo CommonInfo
}

func NewCore(c *Config) CoreReflectionMux {
	return &coreReflectionMux{
		encoderDecoder:     c.EncoderDecoder,
		convertErrorSchema: c.ConvertErrorSchema,
		commonInfo:         c.CommonInfo,
	}
}

type reflectionResponse struct {
	CommonInfo CommonInfo  `json:"common"`
	Routes     []RouteInfo `json:"routes"`
}

func extractReflectionMux(m CoreReflectionMux) http.Handler {
	mainMux := http.NewServeMux()
	allRoutesInfo := []RouteInfo{}

	// Seed with the reserved reflection pattern so user routes cannot collide with it.
	reflectionPattern := "GET /__reflection"
	seenPatterns := map[string]struct{}{reflectionPattern: {}}

	for _, mux := range m.GetAllPaths() {
		// Register routes to mainMux
		for _, handler := range mux.getHandlers() {
			pattern := handler.Pattern()
			if _, exists := seenPatterns[pattern]; exists {
				panic(fmt.Sprintf("duplicate route pattern %q", pattern))
			}
			seenPatterns[pattern] = struct{}{}
			mainMux.Handle(pattern, handler.Handler())
		}
		allRoutesInfo = append(allRoutesInfo, mux.reflectionRouteInfo()...)
	}
	// Add reflection route
	mt := m.(*coreReflectionMux)
	mainMux.HandleFunc(reflectionPattern, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(reflectionResponse{
			CommonInfo: mt.commonInfo,
			Routes:     allRoutesInfo,
		})
	})

	return mainMux
}

func RegisterRoute[ReqParamT, ReqBodyT, RespBodyT any, ErrorT error](mux PathReflectionMux,
	method string,
	path string,
	meta RouteMeta,
	handler TypedHandler[ReqParamT, ReqBodyT, RespBodyT, ErrorT],
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
