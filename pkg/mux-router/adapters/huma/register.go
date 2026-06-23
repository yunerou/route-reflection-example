package huma

import (
	"context"
	"net/http"
	"reflect"
	"strings"

	huma2 "github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	muxrouter "github.com/yunerou/niarb/pkg/mux-router"
)

// RequestWrapper is the huma input type.
//
// huma only flattens path/query/header params from *embedded* fields, and Go
// forbids embedding a type parameter, so Params cannot be auto-bound by huma.
// Instead RequestWrapper implements huma.Resolver: Resolve reads each tagged
// field of Params from the huma.Context. Parameters are documented separately
// via the operation's Parameters list (see buildParams).
//
// Body is a pointer so huma treats it as optional (huma marks a non-pointer body
// field as required), keeping bodyless requests like GET valid.
type RequestWrapper[ReqParamT, ReqBodyT any] struct {
	Params ReqParamT `json:"-"`
	Body   *ReqBodyT
}

// Resolve binds Params from the request, satisfying huma.Resolver.
func (w *RequestWrapper[ReqParamT, ReqBodyT]) Resolve(ctx huma2.Context) []error {
	param, err := muxrouter.BindParams[ReqParamT](func(source muxrouter.ParamSource, name string) (string, bool) {
		switch source {
		case muxrouter.SourcePath:
			return ctx.Param(name), true
		case muxrouter.SourceQuery:
			v := ctx.Query(name)
			return v, v != ""
		case muxrouter.SourceHeader:
			v := ctx.Header(name)
			return v, v != ""
		default:
			return "", false
		}
	})
	if err != nil {
		return []error{err}
	}
	w.Params = param
	return nil
}

type ResponseWrapper[RespBodyT any] struct {
	Body RespBodyT
}

// buildParams describes ReqParamT's path/query/header fields as OpenAPI
// parameters so they appear in the generated docs.
func buildParams[ReqParamT any]() []*huma2.Param {
	t := reflect.TypeFor[ReqParamT]()
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	var params []*huma2.Param
	add := func(field reflect.StructField, source muxrouter.ParamSource, in string, required bool) {
		name, ok := muxrouter.FieldTagName(field, string(source))
		if !ok {
			return
		}
		params = append(params, &huma2.Param{
			Name:     name,
			In:       in,
			Required: required,
			Schema:   &huma2.Schema{Type: muxrouter.JSONSchemaType(field.Type.Kind())},
		})
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		add(field, muxrouter.SourcePath, "path", true)
		add(field, muxrouter.SourceQuery, "query", false)
		add(field, muxrouter.SourceHeader, "header", false)
	}
	return params
}

func RegisterRoute[ReqParamT, ReqBodyT, RespBodyT any](
	g *Group,
	method, path string,
	meta muxrouter.RouteMeta,
	handler muxrouter.TypedHandler[ReqParamT, ReqBodyT, RespBodyT],
	middleware muxrouter.Middleware,
) {
	g.lazyRegisters = append(g.lazyRegisters, func() {
		fullPath := muxrouter.JoinPath(g.prefixRoute, path)
		_ = muxrouter.ValidateRoute[ReqParamT, ReqBodyT, RespBodyT](method, fullPath)

		convertError := g.router.config.ConvertError
		op := huma2.Operation{
			OperationID: operationID(method, fullPath),
			Method:      method,
			Path:        fullPath,
			Summary:     meta.Summary,
			Description: meta.Description,
			Tags:        meta.Tags,
			Deprecated:  meta.Deprecated,
			Parameters:  buildParams[ReqParamT](),
		}
		if middleware != nil {
			op.Middlewares = huma2.Middlewares{convertMiddleware(middleware)}
		}

		huma2.Register(g.router.api, op,
			func(ctx context.Context, in *RequestWrapper[ReqParamT, ReqBodyT]) (*ResponseWrapper[RespBodyT], error) {
				var body ReqBodyT
				if in.Body != nil {
					body = *in.Body
				}
				out := new(ResponseWrapper[RespBodyT])
				var err error
				out.Body, err = handler(ctx, in.Params, body)
				if err != nil {
					// muxrouter.StatusError satisfies huma2.StatusError structurally.
					return nil, convertError(err)
				}
				return out, nil
			})
	})
}

func operationID(method, path string) string {
	return strings.ToLower(method) + strings.NewReplacer("/", "-", "{", "", "}", "").Replace(path)
}

func convertMiddleware(mw muxrouter.Middleware) func(huma2.Context, func(huma2.Context)) {
	return func(ctx huma2.Context, next func(huma2.Context)) {
		r, w := humago.Unwrap(ctx)
		mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			next(ctx)
		})).ServeHTTP(w, r)
	}
}
