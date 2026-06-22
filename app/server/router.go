package server

import (
	"context"
	"net/http"

	rmux "github.com/yunerou/niarb/pkg/huma-provider"
)

func (c *SvCmd) router(_ ServerType) http.Handler {
	coreHuma := c.configRouter()

	basePath := coreHuma.Create("")
	rmux.RegisterRoute(basePath, "GET", "/health",
		rmux.RouteMeta{},
		func(ctx context.Context, reqParam *struct{}, reqBody *struct{}) (string, error) {
			return "ok", nil
		},
		c.simpleMiddleware,
	)

	examplePath := coreHuma.Create("example")
	rmux.RegisterRoute(examplePath, "GET", "/{param1}/{param2}",
		rmux.RouteMeta{},
		c.exampleHandler.ExampleHandlerFunc,
		c.allMiddilewares,
	)

	return coreHuma.ExtractHandler(true)
}
