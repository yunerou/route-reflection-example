package muxrouter

import (
	"fmt"
	"net/http"
	pathpkg "path"
	"reflect"
	"strings"
)

// ValidateRoute checks the method and reconciles path parameters declared in the
// route pattern against `path`-tagged fields of ReqParamT, panicking on mismatch.
// It returns the reflect.Type info used by adapters to build docs/schemas.
func ValidateRoute[ReqParamT, ReqBodyT, RespBodyT any](method, fullPath string) RouteTypeInfo {
	if !IsValidHTTPMethod(method) {
		panic(fmt.Sprintf("invalid HTTP method %q for route %q", method, fullPath))
	}

	pathParams := ExtractPathParams(fullPath)

	reqType := reflect.TypeFor[ReqParamT]()
	if reqType.Kind() == reflect.Pointer {
		reqType = reqType.Elem()
	}

	structPathParams := map[string]struct{}{}
	if reqType.Kind() == reflect.Struct {
		for i := 0; i < reqType.NumField(); i++ {
			field := reqType.Field(i)
			if field.PkgPath != "" {
				continue
			}
			if name, ok := FieldTagName(field, string(SourcePath)); ok {
				if _, exists := pathParams[name]; !exists {
					panic(fmt.Sprintf("path parameter %q in request type %s does not exist in route path %q", name, reqType.String(), fullPath))
				}
				structPathParams[name] = struct{}{}
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

	return RouteTypeInfo{
		ReqParamType: reqType,
		ReqBodyType:  reflect.TypeFor[ReqBodyT](),
		RespBodyType: reflect.TypeFor[RespBodyT](),
	}
}

// JoinPath joins a prefix and a path and ensures a leading slash.
func JoinPath(prefix, path string) string {
	joined := pathpkg.Join(prefix, path)
	if !strings.HasPrefix(joined, "/") {
		joined = "/" + joined
	}
	return joined
}

func IsValidHTTPMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete, http.MethodConnect,
		http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

func IsValidPathPrefix(prefix string) bool {
	for _, r := range prefix {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '-', r == '_', r == '.', r == '/':
		default:
			return false
		}
	}
	return true
}

func ExtractPathParams(path string) map[string]struct{} {
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

func FieldTagName(field reflect.StructField, tag string) (tagValue string, isValid bool) {
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
