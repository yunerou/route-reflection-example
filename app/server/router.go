package server

import (
	"net/http"

	reflectionmux "github.com/yunerou/niarb/pkg/reflection-mux"
)

func (c *SvCmd) router(_ ServerType) http.Handler {
	exampleMux := reflectionmux.NewReflectionMux()
	// exampleMux.SetPathPrefix("test")

	reflectionmux.RegisterRoute(exampleMux, "GET", "/test/{param1}/{param2}",
		reflectionmux.RouteMeta{},
		c.exampleHandler.ExampleHandlerFunc,
		c.allMiddilewares,
	)
	// mux.Handle("/apis/raw-doc-mng/",
	// 	chainMiddleware(
	// 		http.StripPrefix("/apis/raw-doc-mng", c.rawDocMng.HTTPMux()),
	// 		c.allMiddilewares,
	// 	),
	// )
	// mux.Handle("/apis/knowstore/",
	// 	chainMiddleware(
	// 		http.StripPrefix("/apis/knowstore", c.knowstore.HTTPMux()),
	// 		c.allMiddilewares,
	// 	),
	// )

	return reflectionmux.ExtractReflectionMux(exampleMux)
}
