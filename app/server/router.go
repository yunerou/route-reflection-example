package server

import (
	"context"
	"net/http"

	rmux "github.com/yunerou/niarb/pkg/reflection-mux"
)

func (c *SvCmd) router(_ ServerType) http.Handler {
	coreMux := rmux.NewCore()

	mainMux := coreMux.Create("")
	rmux.RegisterRoute(mainMux, "GET", "/health",
		rmux.RouteMeta{},
		func(ctx context.Context, reqParam struct{}, reqBody struct{}) (struct{}, error) {
			return struct{}{}, nil
		},
		c.simpleMiddleware,
	)

	exampleMux := coreMux.Create("test")
	rmux.RegisterRoute(exampleMux, "GET", "/test/{param1}/{param2}",
		rmux.RouteMeta{},
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

	return rmux.ExtractReflectionMux(exampleMux)
}
