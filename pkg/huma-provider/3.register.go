package humapvd

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

func extractHuma(m *coreHuma, enableDoc bool) http.Handler {
	allRoutesInfo := []RouteInfo{}
	if !enableDoc {
		// Disable Huma's built-in docs and OpenAPI endpoints when enableDoc is false.
		m.humaConfig.DocsPath = ""
		m.humaConfig.OpenAPIPath = ""
		m.humaConfig.SchemasPath = ""
	}
	m.humaAPI = humago.New(m.mainMux, m.humaConfig)

	for _, g := range m.paths {
		g.runLazyRegister()
		allRoutesInfo = append(allRoutesInfo, g.routeInfo...)
	}

	if enableDoc {
		m.mainMux.HandleFunc("GET /__reflection", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(reflectionResponse{
				CommonInfo: m.commonInfo,
				Routes:     allRoutesInfo,
			})
		})
	}
	return m.mainMux
}

func humaRegisterRoute[ReqParamT, ReqBodyT, RespBodyT any](
	core *coreHuma,
	method string,
	path string,
	meta RouteMeta,
	handler TypedHandler[ReqParamT, ReqBodyT, RespBodyT],
	middleware Middleware,
) {
	operationID := strings.ToLower(method) + strings.NewReplacer(
		"/", "-",
		"{", "",
		"}", "",
	).Replace(path)

	op := huma.Operation{
		OperationID: operationID,
		Method:      method,
		Path:        path,
		Summary:     meta.Summary,
		Description: meta.Description,
		Tags:        meta.Tags,
		Deprecated:  meta.Deprecated,
	}
	if middleware != nil {
		op.Middlewares = huma.Middlewares{convertToHumaMiddleware(middleware)}
	}

	huma.Register(core.humaAPI, op, func(ctx context.Context, input *RequestWrapper[ReqParamT, ReqBodyT]) (*ResponseWrapper[RespBodyT], error) {
		res := new(ResponseWrapper[RespBodyT])
		var err error
		res.Body, err = handler(ctx, input.Params, input.Body)
		if err != nil {
			return nil, core.convertErrorToHumaSchema(err)
		}
		return res, nil
	})
}
