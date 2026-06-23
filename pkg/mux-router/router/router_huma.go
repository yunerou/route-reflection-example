//go:build !prod

package router

import (
	mr "github.com/yunerou/niarb/pkg/mux-router"
	"github.com/yunerou/niarb/pkg/mux-router/adapters/huma"
)

type Router = huma.Router
type Group = huma.Group

func New(c mr.Config) *Router { return huma.New(c) }

func RegisterRoute[ReqParamT, ReqBodyT, RespBodyT any](
	g *Group, method, path string, meta mr.RouteMeta,
	handler mr.TypedHandler[ReqParamT, ReqBodyT, RespBodyT], mw mr.Middleware,
) {
	huma.RegisterRoute[ReqParamT, ReqBodyT, RespBodyT](g, method, path, meta, handler, mw)
}
