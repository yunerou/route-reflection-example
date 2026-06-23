# Mux-router Switchable Adapters Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Cho phép switch backend router (gomux / huma) tại build time bằng build tag, code app giống hệt nhau, để tối ưu kích thước binary.

**Architecture:** `pkg/mux-router` là base dùng chung (types, Config hợp nhất, helper validate generic). Hai adapter package `adapters/gomux` (net/http thuần) và `adapters/huma` (huma/v2, có docs) mỗi cái tự cung cấp generic `RegisterRoute[P,B,R]`. Facade `router` có hai file build-tag alias + forward sang adapter được chọn. Đồ thị import `router → adapters/{gomux|huma} → muxrouter(base)`, không cycle.

**Tech Stack:** Go 1.25, `net/http`, `github.com/danielgtaylor/huma/v2` v2.38, `bytedance/sonic`, `vmihailenco/msgpack/v5`.

## Global Constraints

- Module path: `github.com/yunerou/niarb`.
- Base package `pkg/mux-router` (package `muxrouter`) **TUYỆT ĐỐI không** import `huma/v2` hay hai adapter con.
- `huma.StatusError` ≡ `interface { GetStatus() int; Error() string }`, trùng khít `muxrouter.StatusError` → giá trị thoả mãn structural, không cần wrap.
- Hai adapter phải giữ **đúng** chữ ký công khai sau để facade alias sạch:
  - `func New(muxrouter.Config) *Router`
  - `func (*Router) Create(string) *Group`
  - `func (*Router) ExtractHandler(bool) http.Handler`
  - `func RegisterRoute[P, B, R any](*Group, method, path string, muxrouter.RouteMeta, muxrouter.TypedHandler[P,B,R], muxrouter.Middleware)`
- gomux/huma package **không** mang build tag (chỉ facade mang). Binary chỉ link adapter mà facade import dưới tag đang dùng.
- TDD: test trước, chạy fail, implement, chạy pass, commit từng task.

---

### Task 1: Base — refactor `pkg/mux-router` (types, Config, validate, schema)

Xoá seam type-erased cũ và phần core/group (chuyển sang adapter). Giữ types/formats, thêm `CommonInfo`/`DocConfig`, gom helper validate generic thành `ValidateRoute`, đưa `TypeToSchema` vào base.

**Files:**
- Create: `pkg/mux-router/types.go`
- Create: `pkg/mux-router/config.go`
- Create: `pkg/mux-router/validate.go`
- Create: `pkg/mux-router/schema.go`
- Create: `pkg/mux-router/validate_test.go`
- Delete: `pkg/mux-router/adapter.go`, `pkg/mux-router/0.0.new.go`, `pkg/mux-router/1.corerouter.go`, `pkg/mux-router/2.group_router.go`, `pkg/mux-router/3.register_route.go`, `pkg/mux-router/cm_type.go`, `pkg/mux-router/cm_config.go`, `pkg/mux-router/cm_func.go`

**Interfaces:**
- Produces:
  - `muxrouter.ParamSource` + `SourcePath/SourceQuery/SourceHeader`
  - `muxrouter.RouteMeta{Summary,Description,Tags string/[]string,Deprecated bool}`
  - `muxrouter.RouteTypeInfo{ReqParamType,ReqBodyType,RespBodyType reflect.Type}`
  - `muxrouter.TypedHandler[P,B,R] = func(context.Context, P, B) (R, error)`
  - `muxrouter.Middleware = func(http.Handler) http.Handler`, `ChainMiddleware(...) Middleware`
  - `muxrouter.CommonInfo{ServiceName string; RequestHeaders,ResponseHeaders map[string]string; ErrorResponseSchema json.RawMessage}`
  - `muxrouter.DocConfig{Title,Version,DocsPath,OpenAPIPath string}`
  - `muxrouter.StatusError interface{GetStatus() int; Error() string}`
  - `muxrouter.Format`, `muxrouter.RegisterFormat`, `muxrouter.JsonSonicFormat`, `muxrouter.MsgPackFormat`, `muxrouter.JsonHeaders`, `muxrouter.MsgPackHeaders`
  - `muxrouter.Config{Formats []RegisterFormat; ConvertError func(error) StatusError; CommonInfo CommonInfo; Doc DocConfig}`
  - `muxrouter.ValidateRoute[P,B,R any](method, fullPath string) RouteTypeInfo`
  - `muxrouter.ExtractPathParams(path string) map[string]struct{}`
  - `muxrouter.FieldTagName(reflect.StructField, tag string) (string, bool)`
  - `muxrouter.IsValidHTTPMethod(string) bool`, `muxrouter.IsValidPathPrefix(string) bool`
  - `muxrouter.JoinPath(prefix, path string) string`
  - `muxrouter.TypeToSchema(reflect.Type) json.RawMessage`

- [ ] **Step 1: Delete obsolete base files**

```bash
git rm pkg/mux-router/adapter.go pkg/mux-router/0.0.new.go pkg/mux-router/1.corerouter.go pkg/mux-router/2.group_router.go pkg/mux-router/3.register_route.go pkg/mux-router/cm_type.go pkg/mux-router/cm_config.go pkg/mux-router/cm_func.go
```

- [ ] **Step 2: Create `pkg/mux-router/types.go`**

```go
package muxrouter

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
)

type ParamSource string

const (
	SourcePath   ParamSource = "path"
	SourceQuery  ParamSource = "query"
	SourceHeader ParamSource = "header"
)

// TypedHandler is the generic handler shape every adapter registers.
type TypedHandler[ReqParamT, ReqBodyT, RespBodyT any] = func(ctx context.Context, reqParam ReqParamT, reqBody ReqBodyT) (RespBodyT, error)

type RouteMeta struct {
	Summary     string
	Description string
	Tags        []string
	Deprecated  bool
}

// RouteTypeInfo carries the concrete reflect.Type for each generic parameter.
type RouteTypeInfo struct {
	ReqParamType reflect.Type
	ReqBodyType  reflect.Type
	RespBodyType reflect.Type
}

type Middleware = func(http.Handler) http.Handler

func ChainMiddleware(middlewares ...Middleware) Middleware {
	return func(final http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

// CommonInfo documents service-wide metadata shared across routes.
type CommonInfo struct {
	ServiceName         string
	RequestHeaders      map[string]string
	ResponseHeaders     map[string]string
	ErrorResponseSchema json.RawMessage
}

// DocConfig configures documentation output. Only the huma adapter consumes it;
// the gomux adapter ignores it.
type DocConfig struct {
	Title       string
	Version     string
	DocsPath    string // e.g. "/docs"
	OpenAPIPath string // e.g. "/openapi.json"
}
```

- [ ] **Step 3: Create `pkg/mux-router/config.go`**

```go
package muxrouter

import (
	"io"

	"github.com/bytedance/sonic"
	"github.com/vmihailenco/msgpack/v5"
)

// Format defines how a body is (de)serialized.
type Format struct {
	Marshal   func(w io.Writer, v any) error
	Unmarshal func(r io.Reader, v any) error
}

// RegisterFormat binds a Format to the content-type headers that select it.
type RegisterFormat struct {
	Headers []string
	Formats Format
}

// StatusError is an error carrying an HTTP status. Matches huma.StatusError.
type StatusError interface {
	GetStatus() int
	Error() string
}

// Config is shared by every adapter so application code is build-tag agnostic.
type Config struct {
	Formats      []RegisterFormat
	ConvertError func(error) StatusError
	CommonInfo   CommonInfo
	Doc          DocConfig // huma only; gomux ignores
}

var (
	MsgPackHeaders = []string{"application/x-msgpack", "application/msgpack"}
	MsgPackFormat  = Format{
		Marshal: func(w io.Writer, v any) error {
			e := msgpack.NewEncoder(w)
			e.SetCustomStructTag("json")
			return e.Encode(v)
		},
		Unmarshal: func(r io.Reader, v any) error {
			d := msgpack.NewDecoder(r)
			d.SetCustomStructTag("json")
			return d.Decode(v)
		},
	}
	JsonHeaders     = []string{"application/json", "text/json"}
	JsonSonicFormat = Format{
		Marshal: func(w io.Writer, v any) error {
			return sonic.ConfigDefault.NewEncoder(w).Encode(v)
		},
		Unmarshal: func(r io.Reader, v any) error {
			return sonic.ConfigDefault.NewDecoder(r).Decode(v)
		},
	}
)
```

- [ ] **Step 4: Create `pkg/mux-router/validate.go`**

```go
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
```

- [ ] **Step 5: Create `pkg/mux-router/schema.go`**

```go
package muxrouter

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// TypeToSchema describes a reflect.Type as a JSON skeleton for lightweight docs.
//   - struct  -> {fieldName: <schema>} (json tag honored; empty struct -> null)
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
			name := field.Name
			if jsonTag := field.Tag.Get("json"); jsonTag != "" {
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
```

- [ ] **Step 6: Write failing test `pkg/mux-router/validate_test.go`**

```go
package muxrouter

import "testing"

type okParam struct {
	ID string `path:"id"`
}
type mismatchParam struct {
	Other string `path:"other"`
}

func TestValidateRoute_OK(t *testing.T) {
	info := ValidateRoute[okParam, struct{}, struct{}]("GET", "/users/{id}")
	if info.ReqParamType.Name() != "okParam" {
		t.Fatalf("expected okParam, got %s", info.ReqParamType.Name())
	}
}

func TestValidateRoute_MismatchPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on path param mismatch")
		}
	}()
	ValidateRoute[mismatchParam, struct{}, struct{}]("GET", "/users/{id}")
}

func TestTypeToSchema_Struct(t *testing.T) {
	type body struct {
		Name string `json:"name"`
	}
	got := string(TypeToSchema(typeFor[body]()))
	want := `{"name":"string"}`
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}
```

- [ ] **Step 7: Add the test helper used above to the test file**

Append to `validate_test.go`:

```go
import "reflect"

func typeFor[T any]() reflect.Type { return reflect.TypeFor[T]() }
```

(Combine the two `import` lines into one block when writing the file.)

- [ ] **Step 8: Run tests to verify pass**

Run: `go test ./pkg/mux-router/`
Expected: PASS (3 tests). If `go test` reports build errors in adapter dirs, ignore — only this package is targeted.

- [ ] **Step 9: Commit**

```bash
git add pkg/mux-router/
git commit -m "refactor(mux-router): base types, unified Config, ValidateRoute helper"
```

---

### Task 2: gomux — encode/decode & param parsing helpers

Reflection-based request parsing and content-negotiated body (de)serialization, adapted from `pkg/reflection-mux/3.encdec.go` to use `muxrouter.Format`.

**Files:**
- Create: `pkg/mux-router/adapters/gomux/codec.go`
- Create: `pkg/mux-router/adapters/gomux/codec_test.go`

**Interfaces:**
- Consumes: `muxrouter.RegisterFormat`, `muxrouter.Format`, `muxrouter.FieldTagName`, `muxrouter.SourcePath/Query/Header`, `muxrouter.JsonSonicFormat`.
- Produces:
  - `type codec struct { ... }`
  - `newCodec(formats []muxrouter.RegisterFormat) *codec`
  - `(*codec) decodeBody(r *http.Request, v any) error`
  - `(*codec) encodeResponse(w http.ResponseWriter, r *http.Request, status int, body any)`
  - `parseParams[P any](r *http.Request) (P, error)`
  - `setFieldValue(v reflect.Value, raw string) error`

- [ ] **Step 1: Create `pkg/mux-router/adapters/gomux/codec.go`**

```go
package gomux

import (
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"reflect"
	"strconv"

	muxrouter "github.com/yunerou/niarb/pkg/mux-router"
)

// codec resolves a muxrouter.Format from request/response content types.
type codec struct {
	byType  map[string]muxrouter.Format
	def     muxrouter.Format // default when nothing matches
}

func newCodec(formats []muxrouter.RegisterFormat) *codec {
	c := &codec{byType: map[string]muxrouter.Format{}}
	for _, rf := range formats {
		for _, h := range rf.Headers {
			c.byType[h] = rf.Formats
		}
	}
	// Default: first registered JSON header, else built-in sonic JSON.
	if f, ok := c.byType[muxrouter.JsonHeaders[0]]; ok {
		c.def = f
	} else {
		c.def = muxrouter.JsonSonicFormat
	}
	return c
}

func (c *codec) pick(contentType string) muxrouter.Format {
	if contentType == "" {
		return c.def
	}
	if mt, _, err := mime.ParseMediaType(contentType); err == nil {
		if f, ok := c.byType[mt]; ok {
			return f
		}
	}
	return c.def
}

func (c *codec) decodeBody(r *http.Request, v any) error {
	if r.Body == nil || r.ContentLength == 0 {
		return nil
	}
	return c.pick(r.Header.Get("Content-Type")).Unmarshal(r.Body, v)
}

func (c *codec) encodeResponse(w http.ResponseWriter, r *http.Request, status int, body any) {
	f := c.pick(r.Header.Get("Accept"))
	w.WriteHeader(status)
	if body == nil {
		return
	}
	if err := f.Marshal(w, body); err != nil {
		slog.ErrorContext(r.Context(), "gomux: write response failed", slog.Any("err", err))
	}
}

// parseParams fills a fresh P from path/query/header values using struct tags.
func parseParams[P any](r *http.Request) (P, error) {
	var param P
	pv := reflect.ValueOf(&param).Elem()
	pt := pv.Type()
	if pt.Kind() == reflect.Pointer {
		pv.Set(reflect.New(pt.Elem()))
		pv = pv.Elem()
		pt = pt.Elem()
	}
	if pt.Kind() != reflect.Struct {
		return param, nil
	}
	query := r.URL.Query()
	for i := 0; i < pt.NumField(); i++ {
		field := pt.Field(i)
		if field.PkgPath != "" {
			continue
		}
		var (
			raw   string
			found bool
		)
		if name, ok := muxrouter.FieldTagName(field, string(muxrouter.SourcePath)); ok {
			raw, found = r.PathValue(name), true
		} else if name, ok := muxrouter.FieldTagName(field, string(muxrouter.SourceQuery)); ok {
			raw, found = query.Get(name), query.Has(name)
		} else if name, ok := muxrouter.FieldTagName(field, string(muxrouter.SourceHeader)); ok {
			raw = r.Header.Get(name)
			found = raw != ""
		}
		if !found {
			continue
		}
		if err := setFieldValue(pv.Field(i), raw); err != nil {
			return param, fmt.Errorf("parse parameter %q: %w", field.Name, err)
		}
	}
	return param, nil
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
```

- [ ] **Step 2: Write failing test `pkg/mux-router/adapters/gomux/codec_test.go`**

```go
package gomux

import (
	"net/http/httptest"
	"testing"
)

type qp struct {
	Limit int    `query:"limit"`
	Name  string `query:"name"`
}

func TestParseParams_Query(t *testing.T) {
	r := httptest.NewRequest("GET", "/?limit=5&name=bob", nil)
	got, err := parseParams[qp](r)
	if err != nil {
		t.Fatal(err)
	}
	if got.Limit != 5 || got.Name != "bob" {
		t.Fatalf("got %+v", got)
	}
}

func TestSetFieldValue_BadInt(t *testing.T) {
	r := httptest.NewRequest("GET", "/?limit=notanint", nil)
	if _, err := parseParams[qp](r); err == nil {
		t.Fatal("expected error for bad int")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./pkg/mux-router/adapters/gomux/`
Expected: FAIL to build (`parseParams` undefined) before Step 1 is in place; after Step 1, PASS. If you wrote Step 1 first, run now and expect PASS.

- [ ] **Step 4: Run test to verify pass**

Run: `go test ./pkg/mux-router/adapters/gomux/`
Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
git add pkg/mux-router/adapters/gomux/
git commit -m "feat(gomux): reflection param parsing and format codec"
```

---

### Task 3: gomux — Router, Group, RegisterRoute, ExtractHandler

The gomux router core: lazy per-group registration, generic `RegisterRoute` building a `net/http` handler, `ExtractHandler` mounting routes on a `*http.ServeMux`.

**Files:**
- Create: `pkg/mux-router/adapters/gomux/router.go`
- Create: `pkg/mux-router/adapters/gomux/register.go`
- Create: `pkg/mux-router/adapters/gomux/router_test.go`

**Interfaces:**
- Consumes: `muxrouter.Config`, `muxrouter.ValidateRoute`, `muxrouter.JoinPath`, `muxrouter.IsValidPathPrefix`, `muxrouter.RouteMeta`, `muxrouter.TypedHandler`, `muxrouter.Middleware`, `*codec` (Task 2).
- Produces:
  - `type Router struct{...}`, `type Group struct{...}`
  - `New(muxrouter.Config) *Router`
  - `(*Router) Create(prefix string) *Group`
  - `(*Router) ExtractHandler(enableDoc bool) http.Handler`
  - `RegisterRoute[P,B,R any](g *Group, method, path string, meta muxrouter.RouteMeta, handler muxrouter.TypedHandler[P,B,R], mw muxrouter.Middleware)`
  - internal: `type routeHandler struct{ pattern string; h http.Handler }`, `type routeDoc struct{...}`, `(*Group) runLazyRegister()`

- [ ] **Step 1: Create `pkg/mux-router/adapters/gomux/router.go`**

```go
package gomux

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	muxrouter "github.com/yunerou/niarb/pkg/mux-router"
)

type Router struct {
	codec        *codec
	convertError func(error) muxrouter.StatusError
	commonInfo   muxrouter.CommonInfo
	paths        map[string]*Group
}

type Group struct {
	router      *Router
	prefixRoute string

	lazyRegisters []func()
	serveOnce     sync.Once
	handlers      []routeHandler
	docs          []routeDoc
}

type routeHandler struct {
	pattern string
	h       http.Handler
}

// routeDoc is the lightweight doc record exposed at /__reflection.
type routeDoc struct {
	Method             string          `json:"method"`
	Path               string          `json:"path"`
	Meta               muxrouter.RouteMeta `json:"meta"`
	RequestBodySchema  json.RawMessage `json:"requestBodySchema"`
	ResponseBodySchema json.RawMessage `json:"responseBodySchema"`
}

func New(c muxrouter.Config) *Router {
	if c.ConvertError == nil {
		panic("muxrouter.Config.ConvertError is required")
	}
	return &Router{
		codec:        newCodec(c.Formats),
		convertError: c.ConvertError,
		commonInfo:   c.CommonInfo,
		paths:        map[string]*Group{},
	}
}

func (r *Router) Create(prefix string) *Group {
	trimmed := strings.Trim(prefix, "/")
	if !muxrouter.IsValidPathPrefix(trimmed) {
		panic(fmt.Sprintf("invalid path prefix %q", prefix))
	}
	if g, ok := r.paths[trimmed]; ok {
		return g
	}
	g := &Group{router: r, prefixRoute: trimmed}
	r.paths[trimmed] = g
	return g
}

func (g *Group) runLazyRegister() {
	g.serveOnce.Do(func() {
		for _, f := range g.lazyRegisters {
			f()
		}
		g.lazyRegisters = nil
	})
}

func (r *Router) ExtractHandler(enableDoc bool) http.Handler {
	mux := http.NewServeMux()
	allDocs := []routeDoc{}
	const reflectionPattern = "GET /__reflection"
	seen := map[string]struct{}{}
	if enableDoc {
		seen[reflectionPattern] = struct{}{}
	}

	for _, g := range r.paths {
		g.runLazyRegister()
		for _, h := range g.handlers {
			if _, dup := seen[h.pattern]; dup {
				panic(fmt.Sprintf("duplicate route pattern %q", h.pattern))
			}
			seen[h.pattern] = struct{}{}
			mux.Handle(h.pattern, h.h)
		}
		allDocs = append(allDocs, g.docs...)
	}

	if enableDoc {
		mux.HandleFunc(reflectionPattern, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"common": r.commonInfo,
				"routes": allDocs,
			})
		})
	}
	return mux
}
```

- [ ] **Step 2: Create `pkg/mux-router/adapters/gomux/register.go`**

```go
package gomux

import (
	"net/http"

	muxrouter "github.com/yunerou/niarb/pkg/mux-router"
)

func RegisterRoute[ReqParamT, ReqBodyT, RespBodyT any](
	g *Group,
	method, path string,
	meta muxrouter.RouteMeta,
	handler muxrouter.TypedHandler[ReqParamT, ReqBodyT, RespBodyT],
	middleware muxrouter.Middleware,
) {
	g.lazyRegisters = append(g.lazyRegisters, func() {
		fullPath := muxrouter.JoinPath(g.prefixRoute, path)
		info := muxrouter.ValidateRoute[ReqParamT, ReqBodyT, RespBodyT](method, fullPath)

		c := g.router.codec
		convertError := g.router.convertError

		var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			param, err := parseParams[ReqParamT](req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			var body ReqBodyT
			if err := c.decodeBody(req, &body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			resp, hErr := handler(req.Context(), param, body)
			if hErr != nil {
				se := convertError(hErr)
				c.encodeResponse(w, req, se.GetStatus(), map[string]string{"error": se.Error()})
				return
			}
			c.encodeResponse(w, req, http.StatusOK, resp)
		})
		if middleware != nil {
			h = middleware(h)
		}

		g.handlers = append(g.handlers, routeHandler{
			pattern: method + " " + fullPath,
			h:       h,
		})
		g.docs = append(g.docs, routeDoc{
			Method:             method,
			Path:               fullPath,
			Meta:               meta,
			RequestBodySchema:  muxrouter.TypeToSchema(info.ReqBodyType),
			ResponseBodySchema: muxrouter.TypeToSchema(info.RespBodyType),
		})
	})
}
```

- [ ] **Step 3: Write failing test `pkg/mux-router/adapters/gomux/router_test.go`**

```go
package gomux

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	muxrouter "github.com/yunerou/niarb/pkg/mux-router"
)

type statusErr struct {
	code int
	msg  string
}

func (e statusErr) GetStatus() int { return e.code }
func (e statusErr) Error() string  { return e.msg }

func testConfig() muxrouter.Config {
	return muxrouter.Config{
		Formats:      []muxrouter.RegisterFormat{{Headers: muxrouter.JsonHeaders, Formats: muxrouter.JsonSonicFormat}},
		ConvertError: func(err error) muxrouter.StatusError { return statusErr{code: 500, msg: err.Error()} },
	}
}

type getParam struct {
	ID string `path:"id"`
}
type getResp struct {
	ID string `json:"id"`
}

func TestGomux_GetWithPathParam(t *testing.T) {
	r := New(testConfig())
	g := r.Create("/users")
	RegisterRoute(g, "GET", "/{id}", muxrouter.RouteMeta{},
		func(_ context.Context, p getParam, _ struct{}) (getResp, error) {
			return getResp{ID: p.ID}, nil
		}, nil)

	srv := httptest.NewServer(r.ExtractHandler(false))
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/users/42")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out getResp
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out.ID != "42" {
		t.Fatalf("got %+v", out)
	}
}

type postBody struct {
	Name string `json:"name"`
}

func TestGomux_PostWithBody(t *testing.T) {
	r := New(testConfig())
	g := r.Create("/things")
	RegisterRoute(g, "POST", "/", muxrouter.RouteMeta{},
		func(_ context.Context, _ struct{}, b postBody) (postBody, error) {
			return b, nil
		}, nil)

	srv := httptest.NewServer(r.ExtractHandler(false))
	defer srv.Close()

	resp, err := srv.Client().Post(srv.URL+"/things", "application/json", strings.NewReader(`{"name":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(out), `"name":"x"`) {
		t.Fatalf("got %s", out)
	}
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./pkg/mux-router/adapters/gomux/`
Expected: PASS (4 tests total including Task 2).

- [ ] **Step 5: Commit**

```bash
git add pkg/mux-router/adapters/gomux/
git commit -m "feat(gomux): Router/Group/RegisterRoute over net/http"
```

---

### Task 4: gomux — `/__reflection` doc endpoint test

Verify the doc endpoint is gated by `enableDoc` and returns the registered routes.

**Files:**
- Modify: `pkg/mux-router/adapters/gomux/router_test.go` (append)

**Interfaces:**
- Consumes: everything from Task 3.

- [ ] **Step 1: Append failing test to `router_test.go`**

```go
func TestGomux_ReflectionGated(t *testing.T) {
	r := New(testConfig())
	g := r.Create("/users")
	RegisterRoute(g, "GET", "/{id}", muxrouter.RouteMeta{Summary: "get user"},
		func(_ context.Context, p getParam, _ struct{}) (getResp, error) {
			return getResp{ID: p.ID}, nil
		}, nil)

	// enableDoc=false -> 404
	off := httptest.NewServer(r.ExtractHandler(false))
	defer off.Close()
	resp, _ := off.Client().Get(off.URL + "/__reflection")
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404 when docs disabled, got %d", resp.StatusCode)
	}

	// enableDoc=true -> 200 with the route listed
	r2 := New(testConfig())
	g2 := r2.Create("/users")
	RegisterRoute(g2, "GET", "/{id}", muxrouter.RouteMeta{Summary: "get user"},
		func(_ context.Context, p getParam, _ struct{}) (getResp, error) {
			return getResp{ID: p.ID}, nil
		}, nil)
	on := httptest.NewServer(r2.ExtractHandler(true))
	defer on.Close()
	resp2, _ := on.Client().Get(on.URL + "/__reflection")
	if resp2.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp2.StatusCode)
	}
	body, _ := io.ReadAll(resp2.Body)
	if !strings.Contains(string(body), "/users/{id}") {
		t.Fatalf("reflection missing route: %s", body)
	}
}
```

- [ ] **Step 2: Run test to verify pass**

Run: `go test ./pkg/mux-router/adapters/gomux/ -run TestGomux_Reflection -v`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add pkg/mux-router/adapters/gomux/router_test.go
git commit -m "test(gomux): reflection endpoint gated by enableDoc"
```

---

### Task 5: huma adapter — Router, Group, RegisterRoute, ExtractHandler

Full-generic huma registration preserving auto schema/validate/docs, with the same public surface as gomux.

**Files:**
- Create: `pkg/mux-router/adapters/huma/router.go`
- Create: `pkg/mux-router/adapters/huma/register.go`
- Create: `pkg/mux-router/adapters/huma/huma_test.go`

**Interfaces:**
- Consumes: `muxrouter.Config`, `muxrouter.ValidateRoute`, `muxrouter.JoinPath`, `muxrouter.IsValidPathPrefix`, `muxrouter.RouteMeta`, `muxrouter.TypedHandler`, `muxrouter.Middleware`; `huma/v2`, `huma/v2/adapters/humago`.
- Produces (same shape as gomux):
  - `type Router struct{...}`, `type Group struct{...}`
  - `New(muxrouter.Config) *Router`
  - `(*Router) Create(prefix string) *Group`
  - `(*Router) ExtractHandler(enableDoc bool) http.Handler`
  - `RegisterRoute[P,B,R any](g *Group, method, path string, meta muxrouter.RouteMeta, handler muxrouter.TypedHandler[P,B,R], mw muxrouter.Middleware)`
  - internal: `RequestWrapper[P,B]`, `ResponseWrapper[R]`, `(*Group) runLazyRegister()`

- [ ] **Step 1: Create `pkg/mux-router/adapters/huma/router.go`**

```go
package huma

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	huma2 "github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	muxrouter "github.com/yunerou/niarb/pkg/mux-router"
)

type Router struct {
	config       muxrouter.Config
	mainMux      *http.ServeMux
	api          huma2.API
	paths        map[string]*Group
}

type Group struct {
	router      *Router
	prefixRoute string

	lazyRegisters []func()
	serveOnce     sync.Once
}

func New(c muxrouter.Config) *Router {
	if c.ConvertError == nil {
		panic("muxrouter.Config.ConvertError is required")
	}
	return &Router{
		config:  c,
		mainMux: http.NewServeMux(),
		paths:   map[string]*Group{},
	}
}

func (r *Router) Create(prefix string) *Group {
	trimmed := strings.Trim(prefix, "/")
	if !muxrouter.IsValidPathPrefix(trimmed) {
		panic(fmt.Sprintf("invalid path prefix %q", prefix))
	}
	if g, ok := r.paths[trimmed]; ok {
		return g
	}
	g := &Group{router: r, prefixRoute: trimmed}
	r.paths[trimmed] = g
	return g
}

func (g *Group) runLazyRegister() {
	g.serveOnce.Do(func() {
		for _, f := range g.lazyRegisters {
			f()
		}
		g.lazyRegisters = nil
	})
}

func (r *Router) buildHumaConfig(enableDoc bool) huma2.Config {
	cfg := huma2.DefaultConfig(r.config.Doc.Title, r.config.Doc.Version)
	if r.config.Doc.OpenAPIPath != "" {
		cfg.OpenAPIPath = r.config.Doc.OpenAPIPath
	}
	if r.config.Doc.DocsPath != "" {
		cfg.DocsPath = r.config.Doc.DocsPath
	}
	if len(r.config.Formats) > 0 {
		formats := map[string]huma2.Format{}
		for _, rf := range r.config.Formats {
			hf := huma2.Format{
				Marshal:   func(w io.Writer, v any) error { return rf.Formats.Marshal(w, v) },
				Unmarshal: rf.Formats.Unmarshal,
			}
			for _, h := range rf.Headers {
				formats[h] = hf
			}
		}
		cfg.Formats = formats
		cfg.DefaultFormat = muxrouter.JsonHeaders[0]
	}
	if !enableDoc {
		cfg.DocsPath = ""
		cfg.OpenAPIPath = ""
		cfg.SchemasPath = ""
	}
	return cfg
}

func (r *Router) ExtractHandler(enableDoc bool) http.Handler {
	r.api = humago.New(r.mainMux, r.buildHumaConfig(enableDoc))
	for _, g := range r.paths {
		g.runLazyRegister()
	}
	return r.mainMux
}
```

NOTE: add `"io"` to the import block (used in the Formats closure).

- [ ] **Step 2: Create `pkg/mux-router/adapters/huma/register.go`**

```go
package huma

import (
	"context"
	"net/http"
	"strings"

	huma2 "github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	muxrouter "github.com/yunerou/niarb/pkg/mux-router"
)

// RequestWrapper is the huma input type. huma flattens Params' path/query/header
// tagged fields into individual parameters; Body becomes the request body.
type RequestWrapper[ReqParamT, ReqBodyT any] struct {
	Params ReqParamT
	Body   ReqBodyT
}

type ResponseWrapper[RespBodyT any] struct {
	Body RespBodyT
}

func RegisterRoute[ReqParamT, ReqBodyT, RespBodyT any](
	g *Group,
	method, path string,
	meta muxrouter.RouteMeta,
	handler muxrouter.TypedHandler[ReqParamT, ReqBodyT, RespBodyT],
	middleware muxrouter.Middleware,
) {
	g.lazyRegisters = append(g.lazyRegisters, func() {
		fullPath := muxrouter.JoinPath(g.prefixRoute, path)
		_ = muxrouter.ValidateRoute[ReqParamT, ReqBodyT, RespBodyT](method, fullPath)

		convertError := g.router.config.ConvertError
		op := huma2.Operation{
			OperationID: operationID(method, fullPath),
			Method:      method,
			Path:        fullPath,
			Summary:     meta.Summary,
			Description: meta.Description,
			Tags:        meta.Tags,
			Deprecated:  meta.Deprecated,
		}
		if middleware != nil {
			op.Middlewares = huma2.Middlewares{convertMiddleware(middleware)}
		}

		huma2.Register(g.router.api, op,
			func(ctx context.Context, in *RequestWrapper[ReqParamT, ReqBodyT]) (*ResponseWrapper[RespBodyT], error) {
				out := new(ResponseWrapper[RespBodyT])
				var err error
				out.Body, err = handler(ctx, in.Params, in.Body)
				if err != nil {
					// muxrouter.StatusError satisfies huma2.StatusError structurally.
					return nil, convertError(err)
				}
				return out, nil
			})
	})
}

func operationID(method, path string) string {
	return strings.ToLower(method) + strings.NewReplacer("/", "-", "{", "", "}", "").Replace(path)
}

func convertMiddleware(mw muxrouter.Middleware) func(huma2.Context, func(huma2.Context)) {
	return func(ctx huma2.Context, next func(huma2.Context)) {
		r, w := humago.Unwrap(ctx)
		mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			next(ctx)
		})).ServeHTTP(w, r)
	}
}
```

- [ ] **Step 3: Write failing test `pkg/mux-router/adapters/huma/huma_test.go`**

```go
package huma

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	muxrouter "github.com/yunerou/niarb/pkg/mux-router"
)

type statusErr struct {
	code int
	msg  string
}

func (e statusErr) GetStatus() int { return e.code }
func (e statusErr) Error() string  { return e.msg }

func testConfig() muxrouter.Config {
	return muxrouter.Config{
		ConvertError: func(err error) muxrouter.StatusError { return statusErr{500, err.Error()} },
		Doc:          muxrouter.DocConfig{Title: "Test", Version: "1.0.0", DocsPath: "/docs", OpenAPIPath: "/openapi.json"},
	}
}

type getParam struct {
	ID string `path:"id"`
}
type getResp struct {
	ID string `json:"id"`
}

func TestHuma_GetWithPathParam(t *testing.T) {
	r := New(testConfig())
	g := r.Create("/users")
	RegisterRoute(g, "GET", "/{id}", muxrouter.RouteMeta{},
		func(_ context.Context, p getParam, _ struct{}) (getResp, error) {
			return getResp{ID: p.ID}, nil
		}, nil)

	srv := httptest.NewServer(r.ExtractHandler(true))
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/users/42")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out.ID != "42" {
		t.Fatalf("got %+v", out)
	}
}

func TestHuma_OpenAPIServedWhenDocEnabled(t *testing.T) {
	r := New(testConfig())
	g := r.Create("/users")
	RegisterRoute(g, "GET", "/{id}", muxrouter.RouteMeta{Summary: "get"},
		func(_ context.Context, p getParam, _ struct{}) (getResp, error) {
			return getResp{ID: p.ID}, nil
		}, nil)
	srv := httptest.NewServer(r.ExtractHandler(true))
	defer srv.Close()

	resp, _ := srv.Client().Get(srv.URL + "/openapi.json")
	if resp.StatusCode != 200 {
		t.Fatalf("expected openapi 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "/users/{id}") {
		t.Fatalf("openapi missing route: %s", body)
	}
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./pkg/mux-router/adapters/huma/`
Expected: PASS (2 tests). If `huma2.Format.Marshal` signature differs, run `go doc github.com/danielgtaylor/huma/v2.Format` and adjust the closure in router.go Step 1 to match.

- [ ] **Step 5: Commit**

```bash
git add pkg/mux-router/adapters/huma/
git commit -m "feat(huma): full-generic adapter with OpenAPI docs"
```

---

### Task 6: Facade `router` with build-tag alias + verification

Single import for the app; build tag picks the backend. Verify both builds compile and gomux build excludes huma.

**Files:**
- Create: `pkg/mux-router/router/router_huma.go`
- Create: `pkg/mux-router/router/router_gomux.go`
- Create: `pkg/mux-router/router/doc.go`

**Interfaces:**
- Consumes: both adapters.
- Produces (identical under each tag): `router.Router`, `router.Group`, `router.New`, `router.RegisterRoute`, plus methods via alias.

- [ ] **Step 1: Create `pkg/mux-router/router/router_huma.go`**

```go
//go:build !prod

package router

import (
	"github.com/danielgtaylor/huma/v2" // ensure module present under this tag
	mr "github.com/yunerou/niarb/pkg/mux-router"
	"github.com/yunerou/niarb/pkg/mux-router/adapters/huma"
)

var _ = huma.NewError // keep huma/v2 import referenced indirectly

type Router = huma.Router
type Group = huma.Group

func New(c mr.Config) *Router { return huma.New(c) }

func RegisterRoute[ReqParamT, ReqBodyT, RespBodyT any](
	g *Group, method, path string, meta mr.RouteMeta,
	handler mr.TypedHandler[ReqParamT, ReqBodyT, RespBodyT], mw mr.Middleware,
) {
	huma.RegisterRoute[ReqParamT, ReqBodyT, RespBodyT](g, method, path, meta, handler, mw)
}
```

NOTE: if the unused `huma/v2` import causes a compile error, delete both the import line and the `var _ = huma.NewError` line — the adapter package already pulls in huma/v2 transitively. Prefer deleting them; they are only a reminder.

- [ ] **Step 2: Create `pkg/mux-router/router/router_gomux.go`**

```go
//go:build prod

package router

import (
	mr "github.com/yunerou/niarb/pkg/mux-router"
	"github.com/yunerou/niarb/pkg/mux-router/adapters/gomux"
)

type Router = gomux.Router
type Group = gomux.Group

func New(c mr.Config) *Router { return gomux.New(c) }

func RegisterRoute[ReqParamT, ReqBodyT, RespBodyT any](
	g *Group, method, path string, meta mr.RouteMeta,
	handler mr.TypedHandler[ReqParamT, ReqBodyT, RespBodyT], mw mr.Middleware,
) {
	gomux.RegisterRoute[ReqParamT, ReqBodyT, RespBodyT](g, method, path, meta, handler, mw)
}
```

- [ ] **Step 3: Create `pkg/mux-router/router/doc.go`**

```go
// Package router is the build-tag facade over the mux-router adapters.
//
// Default build selects the huma adapter (OpenAPI docs + validation).
// Build with -tags prod to select the gomux adapter (net/http only, smaller binary).
//
// Application code is identical across builds:
//
//	r := router.New(cfg)
//	g := r.Create("/users")
//	router.RegisterRoute(g, "GET", "/{id}", meta, handler, nil)
//	h := r.ExtractHandler(enableDoc)
package router
```

- [ ] **Step 4: Simplify the huma facade file**

Remove the placeholder import reminder from Step 1 so the file is clean:

```go
//go:build !prod

package router

import (
	mr "github.com/yunerou/niarb/pkg/mux-router"
	"github.com/yunerou/niarb/pkg/mux-router/adapters/huma"
)

type Router = huma.Router
type Group = huma.Group

func New(c mr.Config) *Router { return huma.New(c) }

func RegisterRoute[ReqParamT, ReqBodyT, RespBodyT any](
	g *Group, method, path string, meta mr.RouteMeta,
	handler mr.TypedHandler[ReqParamT, ReqBodyT, RespBodyT], mw mr.Middleware,
) {
	huma.RegisterRoute[ReqParamT, ReqBodyT, RespBodyT](g, method, path, meta, handler, mw)
}
```

- [ ] **Step 5: Verify both builds compile**

Run:
```bash
go build ./... && go build -tags prod ./pkg/mux-router/router/
go vet ./pkg/mux-router/...
```
Expected: no output (success). `go build ./...` covers the default (huma) facade; the second covers the gomux facade.

- [ ] **Step 6: Verify gomux build excludes huma**

Run:
```bash
go list -tags prod -deps github.com/yunerou/niarb/pkg/mux-router/router | grep -c 'danielgtaylor/huma'
```
Expected: `0` (huma is not linked into the prod facade).

Cross-check the default build DOES include huma:
```bash
go list -deps github.com/yunerou/niarb/pkg/mux-router/router | grep -c 'danielgtaylor/huma'
```
Expected: a non-zero number.

- [ ] **Step 7: Run the full test suite**

Run: `go test ./pkg/mux-router/...`
Expected: PASS for base, gomux, and huma packages (router package has no tests).

- [ ] **Step 8: Commit**

```bash
git add pkg/mux-router/router/
git commit -m "feat(router): build-tag facade selecting gomux or huma adapter"
```

---

## Self-Review

**Spec coverage:**
- §3 Base (types/Config/formats/ValidateRoute/TypeToSchema) → Task 1. ✓
- §4 gomux adapter (codec, Router/Group/RegisterRoute, /__reflection) → Tasks 2, 3, 4. ✓
- §5 huma adapter (wrappers, RegisterRoute generic, config map, ExtractHandler) → Task 5. ✓
- §6 facade build-tag → Task 6. ✓
- §7 uniform contract → enforced by Task 6 Step 5 (both builds compile via shared facade signatures). ✓
- §8 verification (both tags build, gomux excludes huma, openapi 200) → Task 6 Steps 5–7, Task 5 Step 3. ✓
- §9 out-of-scope (comment codegen, schema validation for gomux) → not implemented, intentional. ✓

**Placeholder scan:** Step 1/Step 4 of Task 6 intentionally show the messy-then-clean huma facade; final state is Step 4's clean version. No TBD/TODO in shipped code. ✓

**Type consistency:** `New(muxrouter.Config) *Router`, `Create(string) *Group`, `ExtractHandler(bool) http.Handler`, `RegisterRoute[P,B,R](*Group, string, string, muxrouter.RouteMeta, muxrouter.TypedHandler[P,B,R], muxrouter.Middleware)` are identical in gomux (Task 3) and huma (Task 5), matching the facade (Task 6) and Global Constraints. `muxrouter.StatusError`/`huma2.StatusError` structural match confirmed. ✓
