# striptags

Loại bỏ các struct tag không cần thiết ở runtime (ví dụ `doc`, `example` dùng để build Swagger) khỏi binary Go **mà không chỉnh sửa source code của bạn**.

Tool đọc AST toàn bộ project, lọc các tag key trong danh sách cần xoá, ghi các file đã sửa ra thư mục tạm, rồi build bằng cơ chế `go build -overlay`. Source gốc giữ nguyên byte-for-byte; các bản sửa chỉ tồn tại trong thư mục temp và chỉ phục vụ đúng lần build đó.

## Vì sao dùng `-overlay` thay vì copy cả cây project

- Chỉ ghi ra những file thật sự có thay đổi.
- Build vẫn chạy trên cây project gốc, nên `//go:embed`, `replace => ../somelib`, `go.sum`, file generate và build tag theo `GOOS` đều không bị ảnh hưởng.
- Module path không đổi → import nội bộ tự khớp.

So với việc in lại toàn bộ AST, tool sửa ở mức **byte** đúng vị trí của tag literal, nên format, comment và khoảng trắng còn lại được giữ nguyên.

## Yêu cầu

- Go 1.16 trở lên (cần cờ `-overlay`).
- Chạy lệnh từ thư mục gốc của module (nơi có `go.mod`).

## Cài đặt

Đặt `striptags.go` ở đâu cũng được, ví dụ `tools/striptags/main.go`. Chạy thẳng bằng `go run`, hoặc build sẵn thành binary:

```bash
go build -o bin/striptags ./tools/striptags/main.go
```

## Cú pháp

```
striptags [-drop <keys>] [-root <dir>] [-o <output>] [target]
```

| Flag     | Mặc định      | Mô tả |
|----------|---------------|-------|
| `-drop`  | `doc,example` | Danh sách tag key cần xoá, cách nhau bằng dấu phẩy. Ví dụ `doc,example,openapi`. |
| `-root`  | `.`           | Thư mục module cần quét. |
| `-o`     | *(trống)*     | Đường dẫn binary đầu ra. Nếu set, tool sẽ tự chạy `go build`. Nếu bỏ trống, tool chỉ in lệnh gợi ý. |
| `target` | *(trống)*     | Package cần build, ví dụ `./cmd/app`. |

Tool tự bỏ qua `vendor/`, `testdata/`, thư mục ẩn và file `_test.go`.

## Hai chế độ chạy

### Chế độ 1 — tự build (đơn giản nhất)

```bash
go run tools/striptags/main.go -drop doc,example -root . -o ./app ./cmd/app
```

Tool rewrite tag → tạo overlay → gọi `go build -overlay … -o ./app ./cmd/app`. Biến môi trường được kế thừa nên cross-compile chạy bình thường:

```bash
GOOS=linux GOARCH=arm64 go run tools/striptags/main.go -o ./app-arm64 ./cmd/app
```

### Chế độ 2 — chỉ tạo overlay rồi tự build (linh hoạt)

Dùng chế độ này khi cần thêm cờ build như `-trimpath`, `-ldflags`, `-tags` (chế độ tự build chưa truyền được các cờ này). Bỏ `-o`, lấy đường dẫn overlay rồi tự chạy `go build`:

```bash
OVERLAY=$(go run tools/striptags/main.go -drop doc,example -root . 2>&1 >/dev/null | sed -n 's/.*overlay: //p')

go build -trimpath -ldflags="-s -w" -overlay "$OVERLAY" -o ./app ./cmd/app
```

`2>&1 >/dev/null` dùng để bắt dòng `overlay: …` mà tool in ra stderr; stdout là lệnh gợi ý nên bỏ đi.

Một overlay dùng được cho mọi target, nên có nhiều binary thì chỉ cần tạo overlay một lần:

```bash
for pkg in ./cmd/api ./cmd/worker; do
  go build -trimpath -ldflags="-s -w" -overlay "$OVERLAY" -o "bin/$(basename $pkg)" "$pkg"
done
```

## Tích hợp Makefile

```makefile
build-prod:
	@OVERLAY=$$(go run ./tools/striptags/main.go -drop doc,example -root . 2>&1 >/dev/null | sed -n 's/.*overlay: //p'); \
	go build -trimpath -ldflags="-s -w" -overlay "$$OVERLAY" -o bin/app ./cmd/app
```

## Thứ tự đúng trong workflow

Sau khi strip là mất luôn `doc`/`example`, nên **generate Swagger từ source gốc (còn tag) trước**, rồi mới strip để build binary production:

```bash
swag init -g cmd/app/main.go                      # đọc tag đầy đủ -> swagger.json
go run ./tools/striptags/main.go -o ./app ./cmd/app    # build bản đã strip
```

## Kiểm chứng kết quả

```bash
strings ./app | grep -cE 'doc:|example:'   # kỳ vọng: 0
ls -lh ./app                                # so size với bản chưa strip
git status                                  # sạch — source gốc không bị đụng
```

## Ví dụ

Trước:

```go
type User struct {
    ID    int    `json:"id" doc:"Unique identifier of the user" example:"42"`
    Name  string `json:"name" validate:"required" doc:"Full display name" example:"Jane Doe"`
}
```

File mà tool ghi ra temp (chỉ dùng để build):

```go
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name" validate:"required"`
}
```

`json` và `validate` được giữ; `doc`/`example` bị xoá. Tag được parse theo đúng grammar của `reflect.StructTag`, nên value chứa dấu cách như `doc:"Full display name"` vẫn xử lý đúng.

## Lưu ý

- Cờ `-overlay` chỉ thay đổi struct tag; phần debug info nên bổ sung `-trimpath` và `-ldflags="-s -w"` — đây là các cờ độc lập, không thay thế nhau.
- Nên đo lợi ích trước khi đưa vào pipeline: build cả hai bản rồi so `ls -lh` / `go tool nm`. Mức giảm size tuỳ khối lượng tag trong project.
- Nếu có file `.go` mà parser không đọc được (lỗi cú pháp, file generate hỏng), tool sẽ dừng và báo đúng tên file — sửa file đó rồi chạy lại.
