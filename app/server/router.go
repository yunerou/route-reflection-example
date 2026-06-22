package server

import (
	"context"
	"io"
	"net/http"

	rmux "github.com/yunerou/niarb/pkg/reflection-mux"
	"github.com/yunerou/niarb/shared/actx"
)

func (c *SvCmd) router(_ ServerType) http.Handler {
	coreMux := rmux.NewCore(
		&rmux.Config{
			EncoderDecoder: &encdecBase{},
			CommonInfo: rmux.CommonInfo{
				ServiceName: "example-service",
			},
		},
	)

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

	return coreMux.ExtractHandler()
}

type encdecBase struct{}

func (c *encdecBase) Encode(ctx context.Context, w io.Writer, v any) error {
	aCtx := actx.From(ctx)
	return aCtx.GetEncoder().Encode(ctx, w, v)
}

func (c *encdecBase) Decode(ctx context.Context, r io.Reader, v any) error {
	aCtx := actx.From(ctx)
	return aCtx.GetDecoder().Decode(ctx, r, v)
}
