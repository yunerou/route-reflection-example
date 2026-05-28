package reflectionmux

import (
	"context"
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

type ReflectionMux interface {
	SetPathPrefix(prefix string)
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
	PathParams         []ParamInfo // extract từ path pattern + struct tag
	QueryParams        []ParamInfo // extract từ `query` tag
	HeaderParams       []ParamInfo // extract từ `header` tag
	RequestBodySchema  string      // dựa vào struct type để sinh schema
	ResponseBodySchema string
	Meta               RouteMeta
}

type TypedHandler[ReqParamT, ReqBodyT, RespBodyT any, ErrorT error] = func(ctx context.Context, reqParam ReqParamT, reqBody ReqBodyT) (RespBodyT, ErrorT)

type RouteMeta struct {
	Summary     string
	Description string
	Tags        []string
	Deprecated  bool
}
