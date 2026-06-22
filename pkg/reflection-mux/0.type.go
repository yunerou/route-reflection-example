package reflectionmux

import (
	"context"
	"encoding/json"
	"net/http"
)

type RoutHandler struct {
	h       http.Handler
	pattern string
}

func (r RoutHandler) Pattern() string {
	return r.pattern
}
func (r RoutHandler) Handler() http.Handler {
	return r.h
}

type CoreReflectionMux interface {
	Create(PathPrefix string) PathReflectionMux
	GetAllPaths() map[string]PathReflectionMux // key là path prefix, value là PathReflectionMux tương ứng
	ExtractHandler() http.Handler
}

type PathReflectionMux interface {
	getHandlers() []RoutHandler
	reflectionRouteInfo() []RouteInfo
}

type ParamSource string

const (
	SourcePath   ParamSource = "path"
	SourceQuery  ParamSource = "query"
	SourceHeader ParamSource = "header"
)

type ParamInfo struct {
	Name            string
	StructFieldName string
	Source          ParamSource
	Type            string // "string", "int", "bool", ...
	Doc             string // từ comment của field nếu có
}
type BodyInfo struct {
	TypeName string
	Fields   []ParamInfo
}
type RouteInfo struct {
	Method             string
	Path               string
	PathParams         []ParamInfo     // extract từ path pattern + struct tag
	QueryParams        []ParamInfo     // extract từ `query` tag
	HeaderParams       []ParamInfo     // extract từ `header` tag
	RequestBodySchema  json.RawMessage // JSON schema của request body, null nếu không có body
	ResponseBodySchema json.RawMessage
	Meta               RouteMeta
}

type TypedHandler[ReqParamT, ReqBodyT, RespBodyT any, ErrorT error] = func(ctx context.Context, reqParam ReqParamT, reqBody ReqBodyT) (RespBodyT, ErrorT)

type RouteMeta struct {
	Summary     string
	Description string
	Tags        []string
	Deprecated  bool
}

type CommonInfo struct {
	ServiceName         string
	RequestHeaders      map[string]string // Key là tên header, value là mô tả header đó. Dùng để document chung cho tất cả route.
	ResponseHeaders     map[string]string // Key là tên header, value là mô tả header đó. Dùng để document chung cho tất cả route.
	ErrorResponseSchema json.RawMessage   // JSON schema của error response, nếu có trả về.
}
