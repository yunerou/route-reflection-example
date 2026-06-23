//go:build prod

package router

import (
	mr "github.com/yunerou/niarb/pkg/mux-router"
	"github.com/yunerou/niarb/pkg/mux-router/adapters/gomux"
)

type Router = gomux.Router
type Group = gomux.Group

func New(c mr.Config) *Router { return gomux.New(c) }

func RegisterRoute[ReqParamT, ReqBodyT, RespBodyT any](
	g *Group, method, path string, meta mr.RouteMeta,
	handler mr.TypedHandler[ReqParamT, ReqBodyT, RespBodyT], mw mr.Middleware,
) {
	gomux.RegisterRoute[ReqParamT, ReqBodyT, RespBodyT](g, method, path, meta, handler, mw)
}
