# Mux-router Adapters — Thiết kế

Ngày: 2026-06-23
Module: `github.com/yunerou/niarb`
Phạm vi: `pkg/mux-router` và `pkg/mux-router/adapters/{gomux,huma}`

## 1. Mục tiêu

Xây dựng một router có thể switch backend tại **build time** bằng build tag, để chủ
động tối ưu kích thước binary:

- `-tags prod` → backend **gomux** (chỉ `net/http`): binary nhỏ, không có doc generation.
- mặc định (`!prod`) → backend **huma** (`github.com/danielgtaylor/huma/v2`): tự động
  sinh OpenAPI schema, validate, và docs UI.

Code ứng dụng phải **giống hệt nhau** ở cả hai build; chỉ khác cờ `-tags`.

## 2. Quyết định kiến trúc cốt lõi

### 2.1 Giữ generic đầy đủ cho huma ("generic thuần")

`huma.Register[I, O]` yêu cầu generic type tại compile-time để sinh schema +
validate. Vì interface method trong Go **không thể generic**, nên seam
type-erased hiện tại (`Adapter.RegisterRoute(handler func(...any) any)`) không thể
truyền generic types ngược lại cho huma.

→ Bỏ seam interface type-erased. Việc đăng ký route là **generic free function do
mỗi adapter package cung cấp**. Switch backend xảy ra ở **mức package + build tag**,
không phải runtime.

### 2.2 Layout package (S1 — Facade + build-tag alias)

```
pkg/mux-router/                  (package muxrouter)  — BASE, không import backend nào
  ├─ adapters/gomux/             (package gomux)      — import base
  ├─ adapters/huma/              (package huma)       — import base + huma/v2
  └─ router/                     (package router)     — FACADE, build-tag, import 1 adapter
```

Đồ thị import: `router → adapters/{gomux|huma} → muxrouter(base)`. **Không có import
cycle** vì base không import adapter, và adapter không import facade.

Ứng dụng **chỉ import** `github.com/yunerou/niarb/pkg/mux-router/router`.

## 3. Base — `pkg/mux-router` (package `muxrouter`)

Chứa mọi thứ độc lập backend. Không import `gomux`/`huma` con và không import
`huma/v2`.

### 3.1 Types dùng chung (giữ từ code hiện tại)

- `ParamSource` + hằng `SourcePath`/`SourceQuery`/`SourceHeader`.
- `RouteMeta{ Summary, Description, Tags, Deprecated }`.
- `RouteTypeInfo{ ReqParamType, ReqBodyType, RespBodyType reflect.Type }`.
- `TypedHandler[ReqParamT, ReqBodyT, RespBodyT]` = `func(ctx, reqParam, reqBody) (resp, error)`.
- `Middleware = func(http.Handler) http.Handler` + `ChainMiddleware(...)`.
- `CommonInfo{ ServiceName, RequestHeaders, ResponseHeaders, ErrorResponseSchema }`
  (lấy shape từ `huma-provider`/`reflection-mux`).

### 3.2 Formats & lỗi (giữ từ `cm_config.go`, `cm_type.go`)

- `Format{ Marshal, Unmarshal }`, `RegisterFormat{ Headers []string, Formats Format }`.
- Sẵn `JsonSonicFormat` (bytedance/sonic) và `MsgPackFormat` (vmihailenco/msgpack).
- `StatusError interface { GetStatus() int; Error() string }` — shape này **tương thích
  `huma.StatusError`** nên huma adapter dùng lại trực tiếp.

### 3.3 `Config` hợp nhất (load-bearing)

Cả hai adapter nhận **cùng** `muxrouter.Config` để code app không đổi:

```go
type DocConfig struct {
    Title       string
    Version     string
    DocsPath    string // ví dụ "/docs"        (huma dùng; gomux bỏ qua)
    OpenAPIPath string // ví dụ "/openapi.json" (huma dùng; gomux bỏ qua)
}

type Config struct {
    Formats      []RegisterFormat
    ConvertError func(error) StatusError
    CommonInfo   CommonInfo
    Doc          DocConfig // chỉ huma dùng; gomux bỏ qua
}
```

`ConvertError` bắt buộc khác nil (panic nếu nil, như `huma-provider` hiện tại).

### 3.4 Helper generic / reflection dùng chung

- `ValidateRoute[ReqParamT, ReqBodyT, RespBodyT any](method, fullPath string) RouteTypeInfo`
  — bê nguyên khối validate trong `3.register_route.go`: kiểm tra method hợp lệ,
  đối chiếu path param trong pattern với tag `path` của struct, panic khi lệch, và
  trả về `RouteTypeInfo`. Cả hai adapter gọi hàm này trước khi đăng ký.
- Các helper thuần: `extractPathParams`, `fieldTagName`, `isValidHTTPMethod`,
  `isValidPathPrefix`, `joinPath(prefix, path)` (chuẩn hoá leading `/`).
- `TypeToSchema(reflect.Type) json.RawMessage` — chỉ dùng cho `/__reflection` của
  gomux (đặt ở base để gomux dùng mà không cần import huma).

### 3.5 Bỏ khỏi base

- `adapter.go` (interface `Adapter`).
- `1.corerouter.go` (`coreMuxRouter`), `2.group_router.go` (`GroupRouter`),
  `0.0.new.go` (`NewCoreMuxRouter` + `RegisterRoute` cũ).
- Phần generic của `3.register_route.go` đổi thành `ValidateRoute[...]`.

## 4. Adapter `gomux` — `pkg/mux-router/adapters/gomux` (package `gomux`)

Backend prod: chỉ `net/http`, không import huma.

### 4.1 API công khai (shape thống nhất với huma)

```go
type Router struct { /* mainMux *http.ServeMux, formats, convertError, commonInfo, paths */ }
type Group  struct { /* *Router, prefixRoute, lazyRegisters, serveOnce, routeInfo, routeHandlers */ }

func New(c muxrouter.Config) *Router
func (r *Router) Create(prefix string) *Group
func (r *Router) ExtractHandler(enableDoc bool) http.Handler
func RegisterRoute[ReqParamT, ReqBodyT, RespBodyT any](
    g *Group, method, path string, meta muxrouter.RouteMeta,
    handler muxrouter.TypedHandler[ReqParamT, ReqBodyT, RespBodyT],
    middleware muxrouter.Middleware,
)
```

### 4.2 Đăng ký route

- `RegisterRoute` append một closure lazy (giữ pattern `lazyRegisters` + `serveOnce`
  từ `reflection-mux`).
- Khi chạy lazy: gọi `muxrouter.ValidateRoute[P,B,R]`, rồi dựng `http.Handler`:
  1. Parse path/query/header vào `ReqParamT` bằng reflection
     (`parseRequest` + `setFieldValue` từ `reflection-mux/3.encdec.go`).
  2. Decode body vào `ReqBodyT` qua `Format` chọn theo `Content-Type`
     (mặc định format đầu tiên / JSON nếu không khớp).
  3. Gọi `handler(ctx, reqParam, reqBody)`.
  4. Lỗi → `ConvertError(err)` rồi ghi `GetStatus()` + body; thành công → encode
     `RespBodyT` theo format chọn (negotiate `Accept`, mặc định JSON), status 200.
- Middleware: bọc trực tiếp `http.Handler` (gomux là `net/http` thuần).

### 4.3 `ExtractHandler(enableDoc bool)`

- Chạy lazy register mọi group, gắn handler lên `mainMux`, kiểm tra trùng pattern
  (panic nếu trùng), trả `mainMux`.
- `enableDoc=true` → mount `GET /__reflection` trả JSON `{common, routes}` với schema
  thô từ `TypeToSchema`. `enableDoc=false` → không mount gì thêm.

## 5. Adapter `huma` — `pkg/mux-router/adapters/huma` (package `huma`)

Backend mặc định: import `huma/v2` + `adapters/humago`.

### 5.1 API công khai (cùng shape mục 4.1)

`Router`/`Group`/`New`/`Create`/`ExtractHandler(enableDoc)`/`RegisterRoute[P,B,R]`
— **chữ ký giống hệt** gomux để facade alias đồng nhất.

### 5.2 Wrapper types (từ `huma-provider/0.type.go`)

```go
type RequestWrapper[ReqParamT, ReqBodyT any] struct {
    Params ReqParamT // tag path/query/header được huma flatten thành params
    Body   ReqBodyT
}
type ResponseWrapper[RespBodyT any] struct { Body RespBodyT }
```

### 5.3 Đăng ký route (full generic)

- `RegisterRoute[P,B,R]` lazy → gọi `muxrouter.ValidateRoute[P,B,R]` (đồng nhất kiểm
  tra với gomux) rồi `huma.Register[RequestWrapper[P,B], ResponseWrapper[R]]` với
  `huma.Operation` dựng từ `RouteMeta` (OperationID sinh từ method+path như
  `huma-provider/3.register.go`).
- Handler huma gọi `handler(ctx, input.Params, input.Body)`; lỗi → `ConvertError(err)`
  ép về `huma.StatusError`.
- Middleware: convert qua `humago.Unwrap` (`convertToHumaMiddleware` từ
  `huma-provider/1.core.go`).

### 5.4 Map `muxrouter.Config` → `huma.Config`

- `Doc.Title/Version` → `huma.Config.Info`.
- `Doc.DocsPath/OpenAPIPath` → `huma.Config.DocsPath/OpenAPIPath`.
- `Config.Formats` → `huma.Config.Formats` (map theo content-type headers).
- `Config.ConvertError` dùng khi build response lỗi (kết quả thoả `huma.StatusError`).

### 5.5 `ExtractHandler(enableDoc bool)`

- Tạo `humago.New(mainMux, humaConfig)`, chạy lazy register, trả `mainMux`.
- `enableDoc=false` → xoá `DocsPath`/`OpenAPIPath`/`SchemasPath` trước khi tạo API
  (như `huma-provider/3.register.go`). `true` → giữ `/docs` + `/openapi.json`.

## 6. Facade `router` — `pkg/mux-router/router` (package `router`)

Hai file build-tag, cùng khai báo public surface:

```go
//go:build !prod
// router_huma.go
package router
import (
    mr "github.com/yunerou/niarb/pkg/mux-router"
    "github.com/yunerou/niarb/pkg/mux-router/adapters/huma"
)
type Router = huma.Router
type Group  = huma.Group
func New(c mr.Config) *Router { return huma.New(c) }
func RegisterRoute[P, B, R any](
    g *Group, method, path string, meta mr.RouteMeta,
    h mr.TypedHandler[P, B, R], mw mr.Middleware,
) { huma.RegisterRoute[P, B, R](g, method, path, meta, h, mw) }
```

```go
//go:build prod
// router_gomux.go
package router
import (
    mr "github.com/yunerou/niarb/pkg/mux-router"
    "github.com/yunerou/niarb/pkg/mux-router/adapters/gomux"
)
type Router = gomux.Router
type Group  = gomux.Group
func New(c mr.Config) *Router { return gomux.New(c) }
func RegisterRoute[P, B, R any](
    g *Group, method, path string, meta mr.RouteMeta,
    h mr.TypedHandler[P, B, R], mw mr.Middleware,
) { gomux.RegisterRoute[P, B, R](g, method, path, meta, h, mw) }
```

App dùng: `import "…/pkg/mux-router/router"`, gọi `router.New`, `router.RegisterRoute`,
`(*Router).Create`, `(*Router).ExtractHandler(enableDoc)`. Build gomux:
`go build -tags prod`.

## 7. Hợp đồng đồng nhất giữa hai adapter

Facade chỉ alias sạch nếu hai adapter giữ đúng các chữ ký sau (test build cả hai để
đảm bảo không lệch):

| Phần tử | Chữ ký |
|---|---|
| `New` | `func(muxrouter.Config) *Router` |
| `Create` | `func(*Router) Create(string) *Group` |
| `ExtractHandler` | `func(*Router) ExtractHandler(bool) http.Handler` |
| `RegisterRoute` | `func[P,B,R any](*Group, string, string, muxrouter.RouteMeta, muxrouter.TypedHandler[P,B,R], muxrouter.Middleware)` |

## 8. Kiểm thử / xác minh

- `go build ./...` và `go build -tags prod ./...` đều phải pass (đảm bảo facade alias
  khớp ở cả hai backend).
- `go vet ./pkg/mux-router/...`.
- Một app/test nhỏ đăng ký 1 route GET có path param + 1 route POST có body, chạy với
  cả hai build, so sánh response giống nhau; bản huma kiểm tra `/openapi.json` trả 200.
- Khẳng định binary gomux **không** link `huma/v2` (`go list -deps -tags prod` không
  chứa huma).

## 9. Ngoài phạm vi (future)

- Code-gen doc comment cho field (`CommentExtractor`/`generate.go` trong
  `reflection-mux`) — gomux `/__reflection` tạm để `Doc` rỗng; tích hợp sau.
- Validate theo schema cho gomux (huma đã có sẵn validate; gomux chỉ parse kiểu).
- Xoá/di chuyển `pkg/huma-provider`, `pkg/reflection-mux` — giữ làm tham chiếu, dọn sau.
