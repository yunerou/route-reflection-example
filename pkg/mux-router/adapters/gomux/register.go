package gomux

import (
	"net/http"

	muxrouter "github.com/yunerou/niarb/pkg/mux-router"
)

func RegisterRoute[ReqParamT, ReqBodyT, RespBodyT any](
	g *Group,
	method, path string,
	meta muxrouter.RouteMeta,
	handler muxrouter.TypedHandler[ReqParamT, ReqBodyT, RespBodyT],
	middleware muxrouter.Middleware,
) {
	g.lazyRegisters = append(g.lazyRegisters, func() {
		fullPath := muxrouter.JoinPath(g.prefixRoute, path)
		info := muxrouter.ValidateRoute[ReqParamT, ReqBodyT, RespBodyT](method, fullPath)

		c := g.router.codec
		convertError := g.router.convertError

		var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			param, err := parseParams[ReqParamT](req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			var body ReqBodyT
			if err := c.decodeBody(req, &body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			resp, hErr := handler(req.Context(), param, body)
			if hErr != nil {
				se := convertError(hErr)
				c.encodeResponse(w, req, se.GetStatus(), map[string]string{"error": se.Error()})
				return
			}
			c.encodeResponse(w, req, http.StatusOK, resp)
		})
		if middleware != nil {
			h = middleware(h)
		}

		g.handlers = append(g.handlers, routeHandler{
			pattern: method + " " + fullPath,
			h:       h,
		})
		g.docs = append(g.docs, routeDoc{
			Method:             method,
			Path:               fullPath,
			Meta:               meta,
			RequestBodySchema:  muxrouter.TypeToSchema(info.ReqBodyType),
			ResponseBodySchema: muxrouter.TypeToSchema(info.RespBodyType),
		})
	})
}
