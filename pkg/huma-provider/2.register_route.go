package humapvd

import (
	"encoding/json"
	"fmt"
	"net/http"
	pathpkg "path"
	"reflect"
	"strings"
)

func lazyRegisterRoute[ReqParamT, ReqBodyT, RespBodyT any](
	m *GroupHuma,
	method string,
	path string,
	meta RouteMeta,
	handler TypedHandler[ReqParamT, ReqBodyT, RespBodyT],
	middleware Middleware,
) {

	m.lazyRegisters = append(m.lazyRegisters, func() {
		registerRoute(m, method, path, meta, handler, middleware)
	})
}

func registerRoute[ReqParamT, ReqBodyT, RespBodyT any](
	m *GroupHuma,
	method string,
	path string,
	meta RouteMeta,
	handler TypedHandler[ReqParamT, ReqBodyT, RespBodyT],
	middleware Middleware,
) {
	if !isValidHTTPMethod(method) {
		panic(fmt.Sprintf("invalid HTTP method %q for route %q", method, path))
	}

	joined := pathpkg.Join(m.prefixRoute, path)
	if !strings.HasPrefix(joined, "/") {
		joined = "/" + joined
	}
	fullPath := joined
	pathParams := extractPathParams(fullPath)

	reqType := reflect.TypeFor[ReqParamT]()
	if reqType.Kind() == reflect.Pointer {
		reqType = reqType.Elem()
	}

	routeInfo := RouteInfo{
		Method:             method,
		Path:               fullPath,
		Meta:               meta,
		RequestBodySchema:  TypeToSchema(reflect.TypeFor[ReqBodyT]()),
		ResponseBodySchema: TypeToSchema(reflect.TypeFor[RespBodyT]()),
	}

	structPathParams := map[string]struct{}{}
	if reqType.Kind() == reflect.Struct {
		for i := 0; i < reqType.NumField(); i++ {
			field := reqType.Field(i)
			if field.PkgPath != "" {
				continue
			}

			if name, ok := fieldTagName(field, string(SourcePath)); ok {
				if _, exists := pathParams[name]; !exists {
					panic(fmt.Sprintf("path parameter %q in request type %s does not exist in route path %q", name, reqType.String(), fullPath))
				}
				structPathParams[name] = struct{}{}
				routeInfo.PathParams = append(routeInfo.PathParams, paramInfo(field, name, SourcePath))
			}
			if name, ok := fieldTagName(field, string(SourceQuery)); ok {
				routeInfo.QueryParams = append(routeInfo.QueryParams, paramInfo(field, name, SourceQuery))
			}
			if name, ok := fieldTagName(field, string(SourceHeader)); ok {
				routeInfo.HeaderParams = append(routeInfo.HeaderParams, paramInfo(field, name, SourceHeader))
			}
		}
	}

	if len(structPathParams) != len(pathParams) {
		panic(fmt.Sprintf("route %s %s expects %d path parameter(s), request type %s defines %d", method, fullPath, len(pathParams), reqType.String(), len(structPathParams)))
	}
	for name := range pathParams {
		if _, exists := structPathParams[name]; !exists {
			panic(fmt.Sprintf("route path parameter %q is missing from request type %s", name, reqType.String()))
		}
	}

	humaRegisterRoute(m.coreHuma, method, fullPath, meta, handler, middleware)
	m.routeInfo = append(m.routeInfo, routeInfo)
}

func paramInfo(field reflect.StructField, name string, source ParamSource) ParamInfo {
	return ParamInfo{
		Name:            name,
		StructFieldName: field.Name,
		Source:          source,
		Type:            field.Type.Kind().String(),
	}
}

func isValidHTTPMethod(method string) bool {
	switch method {
	case http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace:
		return true
	default:
		return false
	}
}

func extractPathParams(path string) map[string]struct{} {
	params := map[string]struct{}{}
	for start := strings.IndexByte(path, '{'); start != -1; start = strings.IndexByte(path, '{') {
		path = path[start+1:]
		end := strings.IndexByte(path, '}')
		if end == -1 {
			panic(fmt.Sprintf("invalid route path pattern: missing closing brace in %q", path))
		}
		name := path[:end]
		if before, ok := strings.CutSuffix(name, "..."); ok {
			name = before
		}
		if name == "" || strings.ContainsAny(name, "/{}") {
			panic(fmt.Sprintf("invalid route path parameter %q", path[:end]))
		}
		if _, exists := params[name]; exists {
			panic(fmt.Sprintf("duplicate route path parameter %q", name))
		}
		params[name] = struct{}{}
		path = path[end+1:]
	}
	return params
}

func fieldTagName(field reflect.StructField, tag string) (tagValue string, isValid bool) {
	value, ok := field.Tag.Lookup(tag)
	if !ok || value == "-" {
		return "", false
	}
	tagValue, _, _ = strings.Cut(value, ",")
	if tagValue == "" {
		tagValue = field.Name
	}
	return tagValue, true
}

// TypeToSchema describes the schema of a reflect.Type as JSON.
//   - struct  -> object {fieldName: <schema>}
//   - slice   -> [<element schema>]
//   - pointer -> schema of pointed-to type
//   - map     -> {"map[keyKind]": <value schema>}
//   - other   -> kind name, e.g. "string", "int", "bool"
func TypeToSchema(t reflect.Type) json.RawMessage {
	if t == nil {
		return json.RawMessage("null")
	}

	switch t.Kind() {
	case reflect.Struct:
		if t.NumField() == 0 {
			return json.RawMessage("null")
		}
		var sb strings.Builder
		sb.WriteByte('{')
		first := true
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			jsonTag := field.Tag.Get("json")
			name := field.Name
			if jsonTag != "" {
				tagName, _, _ := strings.Cut(jsonTag, ",")
				if tagName == "-" {
					continue
				}
				if tagName != "" {
					name = tagName
				}
			}
			if !first {
				sb.WriteByte(',')
			}
			first = false
			key, _ := json.Marshal(name)
			sb.Write(key)
			sb.WriteByte(':')
			sb.Write(TypeToSchema(field.Type))
		}
		sb.WriteByte('}')
		return json.RawMessage(sb.String())
	case reflect.Slice, reflect.Array:
		var sb strings.Builder
		sb.WriteByte('[')
		sb.Write(TypeToSchema(t.Elem()))
		sb.WriteByte(']')
		return json.RawMessage(sb.String())
	case reflect.Pointer:
		return TypeToSchema(t.Elem())
	case reflect.Map:
		var sb strings.Builder
		sb.WriteByte('{')
		key, _ := json.Marshal(fmt.Sprintf("map[%s]", t.Key().Kind()))
		sb.Write(key)
		sb.WriteByte(':')
		sb.Write(TypeToSchema(t.Elem()))
		sb.WriteByte('}')
		return json.RawMessage(sb.String())
	default:
		val, _ := json.Marshal(t.Kind().String())
		return json.RawMessage(val)
	}
}
