package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"reflect"

	"github.com/danielgtaylor/huma/v2"
	"github.com/vmihailenco/msgpack/v5"
	rmux "github.com/yunerou/niarb/pkg/huma-provider"
	"github.com/yunerou/niarb/shared/actx"
	"github.com/yunerou/niarb/shared/aerror"
)

func (c *SvCmd) router(_ ServerType) http.Handler {
	var msgpackFormat = huma.Format{
		Marshal: func(w io.Writer, v any) error {
			e := msgpack.NewEncoder(w)
			e.SetCustomStructTag("json")
			return e.Encode(v)
		},
		Unmarshal: func(data []byte, v any) error {
			return msgpack.Unmarshal(data, v)
		},
	}

	humaConfig := huma.DefaultConfig("example-service", "1.0.0")
	humaConfig.Formats["application/msgpack"] = msgpackFormat

	coreHuma := rmux.NewCore(
		&rmux.Config{
			// EncoderDecoder:     &encdecBase{},
			// ConvertErrorSchema: convertErrorSchema,
			HumaConfig: humaConfig,
			CommonInfo: rmux.CommonInfo{
				ServiceName:         "example-service",
				ErrorResponseSchema: rmux.TypeToSchema(reflect.TypeFor[ErrorResponse]()),
			},
		},
	)

	basePath := coreHuma.Create("")
	rmux.RegisterRoute(basePath, "GET", "/health",
		rmux.RouteMeta{},
		func(ctx context.Context, reqParam *struct{}, reqBody *struct{}) (string, error) {
			return "ok", nil
		},
		c.simpleMiddleware,
	)

	examplePath := coreHuma.Create("test")
	rmux.RegisterRoute(examplePath, "GET", "/test/{param1}/{param2}",
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

	return coreHuma.ExtractHandler(true)
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

// ErrorResponse is the standard error body returned for every failed route.
// Its schema is exposed via CommonInfo.ErrorResponseSchema, and convertErrorSchema
// returns it as the response body.
type ErrorResponse struct {
	Code    string `json:"code" msgpack:"code"`
	Message string `json:"message" msgpack:"message"`
}

// convertErrorSchema maps a handler error to an HTTP status code and a standard
// ErrorResponse body. Errors that are not aerror.AError fall back to 500.
func convertErrorSchema(err error) (int, any) {
	aerr, ok := err.(aerror.AError)
	if !ok {
		slog.Error("delivery: request failed with non-AError. Responding with default message", slog.Any("err", err))
		return http.StatusInternalServerError, ErrorResponse{
			Code:    "InternalServerError",
			Message: "An unexpected error occurred",
		}
	}

	code := aerr.ErrorCode()
	slog.Info("delivery: request failed",
		slog.String("code", code.Code()),
		slog.Int("http", code.HttpCode()),
	)
	return code.HttpCode(), ErrorResponse{
		Code:    code.Code(),
		Message: code.Error(),
	}
}
