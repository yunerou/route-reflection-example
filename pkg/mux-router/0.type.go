package muxrouter

import (
	"context"
	"encoding/json"
	"net/http"
)

type CoreHuma interface {
	Create(PathPrefix string) *GroupHuma
	ExtractHandler(enableDoc bool) http.Handler
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
	Doc             string // from the field's comment, if present
}

type BodyInfo struct {
	TypeName string
	Fields   []ParamInfo
}

type RouteInfo struct {
	Method             string
	Path               string
	PathParams         []ParamInfo     // extracted from path pattern + struct tag
	QueryParams        []ParamInfo     // extracted from `query` tag
	HeaderParams       []ParamInfo     // extracted from `header` tag
	RequestBodySchema  json.RawMessage // JSON schema of the request body, null if no body
	ResponseBodySchema json.RawMessage
	Meta               RouteMeta
}

// RequestWrapper is the Huma input type for all registered routes.
// Huma flattens named sub-structs, so path/query/header tags on Params fields
// are recognised as individual parameters. Body is treated as the request body.
type RequestWrapper[ReqParamT, ReqBodyT any] struct {
	Params ReqParamT // struct fields must have path, query, or header tags
	Body   ReqBodyT  // unmarshaled from the request body, if present
}

type ResponseWrapper[RespBodyT any] struct {
	Body RespBodyT // marshaled into the response body
}

type TypedHandler[ReqParamT, ReqBodyT, RespBodyT any] = func(ctx context.Context, reqParam ReqParamT, reqBody ReqBodyT) (RespBodyT, error)

type RouteMeta struct {
	Summary     string
	Description string
	Tags        []string
	Deprecated  bool
}

type CommonInfo struct {
	ServiceName         string
	RequestHeaders      map[string]string // key is the header name, value is its description; applied as common docs for all routes
	ResponseHeaders     map[string]string // key is the header name, value is its description; applied as common docs for all routes
	ErrorResponseSchema json.RawMessage   // JSON schema of the error response, if one is returned
}

type reflectionResponse struct {
	CommonInfo CommonInfo  `json:"common"`
	Routes     []RouteInfo `json:"routes"`
}
