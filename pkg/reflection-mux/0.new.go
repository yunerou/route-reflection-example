package reflectionmux

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

func NewCore() CoreReflectionMux {
	muxInstance := &coreReflectionMux{
		lazyRegisters: []func(){},
		routeInfo:     []RouteInfo{},
		serveOnce:     sync.Once{},
	}

	return muxInstance
}

func ExtractReflectionMux(muxes ...PathReflectionMux) http.Handler {
	mainMux := http.NewServeMux()
	allRoutesInfo := []RouteInfo{}

	// Seed with the reserved reflection pattern so user routes cannot collide with it.
	reflectionPattern := "GET /__reflection"
	seenPatterns := map[string]struct{}{reflectionPattern: {}}

	for _, mux := range muxes {
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
	mainMux.HandleFunc(reflectionPattern, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(allRoutesInfo)
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
