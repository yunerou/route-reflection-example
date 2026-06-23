package server

import (
	"context"
	"net/http"

	r "github.com/yunerou/niarb/app/server/router"
	mr "github.com/yunerou/niarb/pkg/mux-router"
)

func (c *SvCmd) router(_ ServerType) http.Handler {
	core := c.configRouter()

	basePath := core.Create("")
	r.RegisterRoute(basePath, "GET", "/health",
		mr.RouteMeta{},
		func(ctx context.Context, reqParam *struct{}, reqBody *struct{}) (string, error) {
			return "ok", nil
		},
		c.simpleMiddleware,
	)

	examplePath := core.Create("example")
	r.RegisterRoute(examplePath, "GET", "/{param1}/{param2}",
		mr.RouteMeta{},
		c.exampleHandler.ExampleHandlerFunc,
		c.allMiddilewares,
	)

	return core.ExtractHandler(true)
}
