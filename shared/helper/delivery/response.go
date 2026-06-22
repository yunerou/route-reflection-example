package delivery

import (
	"log/slog"
	"net/http"

	"github.com/yunerou/niarb/shared/actx"
	"github.com/yunerou/niarb/shared/aerror"
)

type ErrorBody struct {
	Code    string `json:"code" msgpack:"code"`
	Message string `json:"message" msgpack:"message"`
}

// WriteResponse serializes body using the encoder set on actx by the
// EncoderDecoder middleware. The Content-Type response header is also set
// by that middleware, so callers do not need to inspect the request here.
func WriteResponse(w http.ResponseWriter, r *http.Request, status int, body any) {
	w.WriteHeader(status)
	if body == nil {
		return
	}
	encoder := actx.From(r.Context()).GetEncoder()
	if aerr := encoder.Encode(r.Context(), w, body); aerr != nil {
		slog.Error("delivery: write response failed", slog.Any("err", aerr))
	}
}

// WriteAError maps an aerror.AError to an HTTP response.
func WriteAError(w http.ResponseWriter, r *http.Request, aerr aerror.AError) {
	code := aerr.ErrorCode()
	slog.InfoContext(r.Context(), "delivery: request failed",
		slog.String("code", code.Code()),
		slog.Int("http", code.HttpCode()),
		slog.String("path", r.URL.Path),
	)
	WriteResponse(w, r, code.HttpCode(), ErrorBody{
		Code:    code.Code(),
		Message: code.Error(),
	})
}
