package server

import (
	"io"
	"log/slog"
	"net/http"
	"reflect"

	"github.com/danielgtaylor/huma/v2"
	"github.com/vmihailenco/msgpack/v5"
	rmux "github.com/yunerou/niarb/pkg/huma-provider"
	"github.com/yunerou/niarb/shared/aerror"
)

type ErrorResponse struct {
	status  int
	Message string `json:"message" doc:"Mô tả lỗi"`
	Code    string `json:"code" doc:"Code lỗi nội bộ" example:"INTERNAL_FAILURE"`
	Details any    `json:"details,omitempty" doc:"Chi tiết bổ sung, chỉ có trên một số Code nhất định"`
}

func (e *ErrorResponse) Error() string  { return e.Message }
func (e *ErrorResponse) GetStatus() int { return e.status }

func convertErrorSchema(err error) huma.StatusError {
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

func (c *SvCmd) configRouter() rmux.CoreHuma {
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
			ConvertErrorToHumaSchema: convertErrorSchema,
			HumaConfig:               humaConfig,
			CommonInfo: rmux.CommonInfo{
				ServiceName:         "example-service",
				ErrorResponseSchema: rmux.TypeToSchema(reflect.TypeFor[ErrorResponse]()),
			},
		},
	)
	return coreHuma
}
