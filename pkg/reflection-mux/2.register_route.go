package reflectionmux

import (
	"encoding/json"
	"fmt"
	"net/http"
	pathpkg "path"
	"reflect"
	"strconv"
	"strings"
)

func lazyRegisterRoute[ReqParamT, ReqBodyT, RespBodyT any, ErrorT error](
	mux PathReflectionMux,
	method string,
	path string,
	meta RouteMeta,
	handler TypedHandler[ReqParamT, ReqBodyT, RespBodyT, ErrorT],
	middleware Middleware,
) {
	m, ok := mux.(*pathReflectionMux)
	if !ok {
		panic(fmt.Sprintf("mux %q is not a reflectionMux", mux))
	}

	m.lazyRegisters = append(m.lazyRegisters, func() {
		registerRoute(m, method, path, meta, handler, middleware)
	})
}

func registerRoute[ReqParamT, ReqBodyT, RespBodyT any, ErrorT error](
	mux *pathReflectionMux,
	method string,
	path string,
	meta RouteMeta,
	handler TypedHandler[ReqParamT, ReqBodyT, RespBodyT, ErrorT],
	middleware Middleware,
) {
	if !isValidHTTPMethod(method) {
		panic(fmt.Sprintf("invalid HTTP method %q for route %q", method, path))
	}

	fullPath := pathpkg.Join(mux.prefixRoute, path)
	pathParams := extractPathParams(fullPath)

	reqType := reflect.TypeFor[ReqParamT]()
	if reqType.Kind() == reflect.Pointer {
		reqType = reqType.Elem()
	}

	routeInfo := RouteInfo{
		Method:             method,
		Path:               fullPath,
		Meta:               meta,
		RequestBodySchema:  typeToSchema(reflect.TypeFor[ReqBodyT]()),
		ResponseBodySchema: typeToSchema(reflect.TypeFor[RespBodyT]()),
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
				routeInfo.PathParams = append(routeInfo.PathParams, paramInfo(reqType, field, name, SourcePath))
			}
			if name, ok := fieldTagName(field, string(SourceQuery)); ok {
				routeInfo.QueryParams = append(routeInfo.QueryParams, paramInfo(reqType, field, name, SourceQuery))
			}
			if name, ok := fieldTagName(field, string(SourceHeader)); ok {
				routeInfo.HeaderParams = append(routeInfo.HeaderParams, paramInfo(reqType, field, name, SourceHeader))
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

	var httpHandler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqParam, reqBody, parseErr := defaultParseRequest[ReqParamT, ReqBodyT, ErrorT](mux.encoderDecoder, r)
		if any(parseErr) != nil {
			http.Error(w, error(parseErr).Error(), http.StatusBadRequest)
			return
		}

		resp, handlerErr := handler(r.Context(), reqParam, reqBody)
		if any(handlerErr) != nil {
			defaultWriteError(mux.encoderDecoder, mux.convertErrorSchema, w, r, error(handlerErr))
			return
		}
		defaultWriteResponse(mux.encoderDecoder, w, r, http.StatusOK, resp)
	})
	if middleware != nil {
		httpHandler = middleware(httpHandler)
	}

	mux.routeInfo = append(mux.routeInfo, routeInfo)
	mux.routeHandlers = append(mux.routeHandlers, RoutHandler{
		h:       httpHandler,
		pattern: method + " " + fullPath,
	})
}

func paramInfo(parentType reflect.Type, field reflect.StructField, name string, source ParamSource) ParamInfo {
	return ParamInfo{
		Name:            name,
		StructFieldName: field.Name,
		Source:          source,
		Type:            field.Type.Kind().String(),
		Doc:             fetchDocComment(parentType, field),
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

func setFieldValue(v reflect.Value, raw string) error {
	switch v.Kind() {
	case reflect.String:
		v.SetString(raw)
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		v.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(raw, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetFloat(f)
	default:
		return fmt.Errorf("unsupported field type %s", v.Type())
	}
	return nil
}

func fetchDocComment(structType reflect.Type, field reflect.StructField) string {
	if fieldComment, ok := CommentExtractor[structType][field.Name]; ok {
		return fieldComment
	}
	return ""
}

// typeToSchema mô tả schema của reflect.Type dưới dạng JSON (json.RawMessage),
// giúp nhìn thấy cấu trúc của type đó. Output luôn là JSON hợp lệ:
//   - struct  -> object {fieldName: <schema>} (giữ nguyên thứ tự field)
//   - slice   -> [<schema phần tử>]
//   - pointer -> schema của type được trỏ tới
//   - map     -> {"map[keyKind]": <schema value>}
//   - còn lại -> chuỗi tên kind, ví dụ "string", "int", "bool"
func typeToSchema(t reflect.Type) json.RawMessage {
	if t == nil {
		return json.RawMessage("null")
	}

	switch t.Kind() {
	case reflect.Struct:
		// Struct rỗng (vd: struct{}) coi như "không có body" -> trả về null.
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
				tagName := strings.Split(jsonTag, ",")[0]
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
			sb.Write(typeToSchema(field.Type))
		}
		sb.WriteByte('}')
		return json.RawMessage(sb.String())
	case reflect.Slice, reflect.Array:
		var sb strings.Builder
		sb.WriteByte('[')
		sb.Write(typeToSchema(t.Elem()))
		sb.WriteByte(']')
		return json.RawMessage(sb.String())
	case reflect.Pointer:
		return typeToSchema(t.Elem())
	case reflect.Map:
		var sb strings.Builder
		sb.WriteByte('{')
		key, _ := json.Marshal(fmt.Sprintf("map[%s]", t.Key().Kind()))
		sb.Write(key)
		sb.WriteByte(':')
		sb.Write(typeToSchema(t.Elem()))
		sb.WriteByte('}')
		return json.RawMessage(sb.String())
	default:
		val, _ := json.Marshal(t.Kind().String())
		return json.RawMessage(val)
	}
}
