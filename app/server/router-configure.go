package server

import (
	"log/slog"
	"net/http"
	"runtime"
	"strings"

	r "github.com/yunerou/niarb/app/server/router"
	mr "github.com/yunerou/niarb/pkg/mux-router"
	"github.com/yunerou/niarb/shared/aerror"
)

func convertErrorSchema(err error) mr.StatusError {
	switch e := err.(type) {
	case aerror.ADetailError:
		code := e.ErrorCode()
		return &ErrorResponse{
			status:  code.HttpCode(),
			Code:    code.Code(),
			Message: e.Error(),
			Details: e.Detail(),
		}
	case aerror.AError:
		code := e.ErrorCode()
		return &ErrorResponse{
			status:  code.HttpCode(),
			Code:    code.Code(),
			Message: code.Error(),
		}
	default:
		slog.Error("delivery: request failed with non-AError. Responding with default message", slog.Any("err", err))
		return &ErrorResponse{
			status:  http.StatusInternalServerError,
			Code:    "InternalServerError",
			Message: "An unexpected error occurred"}
	}
}

func (c *SvCmd) configRouter() *r.Router {
	supportedContentTypes := strings.Join(append(mr.JsonHeaders, mr.MsgPackHeaders...), ",")

	env := c.c.Env().Info
	core := r.New(
		mr.Config{
			Formats: []mr.RegisterFormat{
				{
					Headers: mr.JsonHeaders,
					Formats: mr.JsonSonicFormat,
				},
				{
					Headers: mr.MsgPackHeaders,
					Formats: mr.MsgPackFormat,
				},
			},
			DefaultFormat: mr.JsonHeaders[0],
			ConvertError:  convertErrorSchema,
			CommonInfo: mr.CommonInfo{
				ServiceName: env.AppName,
				RequestHeaders: map[string]string{
					"Content-Type": "Content-Type of the request body. Supported values: " + supportedContentTypes,
					"Accept":       "Content-Type to respond with. Supported values: " + supportedContentTypes,
				},
				ResponseHeaders: map[string]string{
					"Content-Type": "Content-Type of the request body. Supported values: " + supportedContentTypes,
				},
			},
			Doc: mr.DocConfig{
				Title:       env.AppName,
				Version:     env.Version,
				DocsPath:    "/__docs",
				OpenAPIPath: "/__openapi",
			},
		},
	)
	return core
}

func printStacktrace() {
	buf := make([]byte, 1<<16)
	n := runtime.Stack(buf, false)
	slog.Info("stacktrace", slog.String("stacktrace", string(buf[:n])))
}
