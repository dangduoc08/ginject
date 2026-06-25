# Middleware CORS

*`cors` hiện thực [Fetch Standard CORS protocol](https://fetch.spec.whatwg.org/#http-cors-protocol) dưới dạng một middleware của Ginject, xử lý cả request thông thường (simple request) và request preflight `OPTIONS`.*

- [Middleware CORS](#middleware-cors)
  - [Tính Năng Chính](#tính-năng-chính)
  - [Cách Dùng](#cách-dùng)
  - [Struct `CORS`](#struct-cors)
    - [AllowOrigin](#alloworigin)
    - [AllowHeaders](#allowheaders)
    - [ExposeHeaders](#exposeheaders)
    - [AllowMethods](#allowmethods)
    - [MaxAge](#maxage)
    - [IsAllowCredentials](#isallowcredentials)
    - [IsPreflightContinue](#ispreflightcontinue)
    - [OptionsSuccessStatus](#optionssuccessstatus)
  - [Phương Thức Của `CORS`](#phương-thức-của-cors)
    - [NewMiddleware](#newmiddleware)
    - [Use](#use)
  - [Benchmark](#benchmark)

## Tính Năng Chính
- Khớp `AllowOrigin` theo wildcard `*`, danh sách origin chính xác, hoặc pattern `*regexp.Regexp`
- Tự động thêm header `Vary: Origin` mỗi khi response phụ thuộc vào origin của request
- Xử lý credentials đúng chuẩn: echo lại origin của request thay vì `*` khi `IsAllowCredentials` được đặt
- Chặn origin `null` khi credentials được bật
- Short-circuit cho preflight: trả về status đã cấu hình mà không gọi `next`, trừ khi `IsPreflightContinue` được đặt

## Cách Dùng

```go
package main

import (
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/middlewares/cors"
)

func main() {
	app := core.New()

	app.BindGlobalMiddlewares(cors.CORS{
		AllowOrigin:        []string{"https://app.example.com"},
		AllowHeaders:       []string{"Content-Type", "Authorization"},
		AllowMethods:       []string{"GET", "POST", "PUT", "DELETE"},
		MaxAge:             86400000,
		IsAllowCredentials: true,
	})

	app.Create(
		core.ModuleBuilder().Build(),
	)
}
```

`CORS` cũng hoạt động khi gắn theo từng controller riêng, vì nó thỏa mãn trực tiếp interface `common.MiddlewareFn`:

```go
func (c UserController) NewController() core.Controller {
	c.BindMiddleware(cors.CORS{AllowOrigin: "https://admin.example.com"})
	return c
}
```

## Struct `CORS`

### AllowOrigin
Type: `any` (`string` | `[]string` | `*regexp.Regexp`)

Default: `"*"` (được đặt bởi `NewMiddleware`/`Use` khi để `nil`)

Required: `false`

Kiểm soát origin nào được phép. Một `string` được so khớp nguyên văn (`"*"` cho phép mọi origin); một `[]string` được so khớp chính xác, với dấu `/` ở cuối mỗi origin đã cấu hình bị loại bỏ trước khi so sánh; một `*regexp.Regexp` được so khớp bằng `MatchString`.

```go
cors.CORS{AllowOrigin: []string{"https://app.example.com", "https://admin.example.com"}}
```

### AllowHeaders
Type: `any` (`string` | `[]string`)

Default: không đặt — response preflight sẽ echo lại giá trị `Access-Control-Request-Headers` của request

Required: `false`

```go
cors.CORS{AllowHeaders: []string{"Content-Type", "Authorization"}}
```

### ExposeHeaders
Type: `any` (`string` | `[]string`)

Default: không đặt — `Access-Control-Expose-Headers` không được gửi

Required: `false`

```go
cors.CORS{ExposeHeaders: []string{"X-Request-ID"}}
```

### AllowMethods
Type: `[]string`

Default: `[]string{"GET", "HEAD", "PUT", "PATCH", "POST", "DELETE"}`

Required: `false`

Được gửi dưới dạng `Access-Control-Allow-Methods` trong response preflight.

```go
cors.CORS{AllowMethods: []string{"GET", "POST"}}
```

### MaxAge
Type: `int` (milli giây)

Default: `5000` (5 giây)

Required: `false`

Thời gian browser được phép cache một response preflight; được chuyển sang giây cho header `Access-Control-Max-Age`.

```go
cors.CORS{MaxAge: 86400000} // đặt header thành "86400"
```

### IsAllowCredentials
Type: `bool`

Default: `false`

Required: `false`

Đặt `Access-Control-Allow-Credentials: true`. Khi kết hợp với `AllowOrigin: "*"`, origin thật của request sẽ được echo lại thay vì `*`.

```go
cors.CORS{IsAllowCredentials: true}
```

### IsPreflightContinue
Type: `bool`

Default: `false`

Required: `false`

Khi `true`, `next` được gọi sau khi các header preflight đã được đặt, chuyển quyền xử lý cho handler kế tiếp. Khi `false`, request preflight bị short-circuit và response được viết ra ngay lập tức.

```go
cors.CORS{IsPreflightContinue: true}
```

### OptionsSuccessStatus
Type: `int`

Default: `204`

Required: `false`

Mã trạng thái HTTP được viết ra cho một response preflight bị short-circuit (chỉ áp dụng khi `IsPreflightContinue` là `false`). Một số browser cũ yêu cầu `200`.

```go
cors.CORS{OptionsSuccessStatus: 200}
```

## Phương Thức Của `CORS`

### NewMiddleware

Biên dịch (compile) các field của struct `CORS` thành một giá trị options nội bộ một lần duy nhất, và trả về một `common.MiddlewareFn` tái sử dụng các option đã biên dịch đó cho mỗi request. Nên ưu tiên dùng hàm này hơn gọi `Use` trực tiếp khi đăng ký middleware một lần lúc khởi động.

#### Parameters
Không có.

#### Returns
- Giá trị thứ 1: `common.MiddlewareFn`

- Mô tả: Một middleware sẵn sàng để gắn (bind) qua `BindGlobalMiddlewares` hoặc `BindMiddleware`, với các option CORS đã được biên dịch một lần.

#### Cách Dùng

```go
mw := cors.CORS{AllowOrigin: "https://example.com"}.NewMiddleware()
app.BindGlobalMiddlewares(mw)
```

### Use

Áp dụng các header CORS cho request hiện tại và hoặc gọi `next`, hoặc short-circuit một response preflight. Biên dịch lại các option của struct mỗi lần được gọi (khác với middleware được trả về từ `NewMiddleware`, vốn chỉ biên dịch một lần), nhưng cho phép một giá trị `CORS` được truyền trực tiếp vào bất kỳ nơi nào cần một `common.MiddlewareFn`.

#### Rules
- Khi không có header `Origin` trong request, không có header CORS nào được đặt và `next` được gọi ngay lập tức (`TestCORS_Use_NoOriginHeaderSkipsCORS`).
- `AllowOrigin` là `nil` sẽ mặc định là `"*"`, đặt `Access-Control-Allow-Origin: *` (`TestCORS_Use_SetsOriginStarByDefault`).
- `AllowOrigin` dạng `[]string` echo lại origin của request và đặt `Vary: Origin` khi khớp (dấu `/` ở cuối origin đã cấu hình bị bỏ qua), và không đặt header nào khi không khớp; một danh sách trống sẽ chặn mọi origin (`TestCORS_Use_SpecificOriginMap`, `TestCORS_Use_OriginTrailingSlashMatchesRequest`, `TestCORS_Use_SpecificOriginMapBlocked`, `TestCORS_Use_EmptySliceBlocksAllOrigins`).
- `AllowOrigin` dạng `*regexp.Regexp` echo lại origin của request và đặt `Vary: Origin` chỉ khi khớp pattern (`TestCORS_Use_RegexpOrigin`, `TestCORS_Use_RegexpOriginNoMatch`).
- `Vary: Origin` được đặt mỗi khi một origin cụ thể được echo lại, nhưng không bao giờ được đặt cho wildcard `"*"` đơn thuần (`TestCORS_Use_VaryForSpecificStringOrigin`, `TestCORS_Use_NoVaryForWildcard`).
- `IsAllowCredentials` đặt `Access-Control-Allow-Credentials: true` (`TestCORS_Use_Credentials`).
- `AllowOrigin` wildcard kết hợp với `IsAllowCredentials` sẽ echo lại origin của request thay vì `*` và đặt `Vary: Origin`, ngoại trừ khi origin của request là `"null"`, trường hợp này không bao giờ được echo lại (`TestCORS_Use_CredentialsWithWildcardEchosOrigin`, `TestCORS_Use_NullOriginWithCredentialsBlocked`).
- `AllowOrigin` wildcard không có `IsAllowCredentials` vẫn đặt `Access-Control-Allow-Origin: *` ngay cả khi origin của request là `"null"` (`TestCORS_Use_NullOriginWildcardNoCredentials`).
- `Access-Control-Allow-Methods`, `Access-Control-Max-Age`, và `Access-Control-Allow-Headers` chỉ được đặt cho request `OPTIONS` (preflight), không bao giờ cho các method khác (`TestCORS_Use_PreflightOnlyHeaders`).
- Một danh sách `AllowMethods` tùy chỉnh được phản ánh trong `Access-Control-Allow-Methods` khi preflight (`TestCORS_Use_CustomAllowMethodsOnPreflight`).
- Một `AllowHeaders`/`ExposeHeaders` dạng string được truyền qua nguyên văn thay vì được join (`TestCORS_Use_AllowHeadersString`, `TestCORS_Use_ExposeHeadersString`).
- `next` luôn được gọi đối với request không phải `OPTIONS` (`TestCORS_Use_NextCalledForNonOptions`).
- Đối với request `OPTIONS`, `next` chỉ được gọi khi `IsPreflightContinue` là `true`; ngược lại response được viết ra ngay với `OptionsSuccessStatus` (mặc định `204`, hoặc giá trị đã cấu hình) và `next` không được gọi (`TestCORS_Use_OptionsPreflightContinue`, `TestCORS_Use_OptionsPreflightStatus`, `TestCORS_Use_CustomOptionsSuccessStatus`).

#### Parameters
- Tham số thứ 1: `*ctx.Context` (`c`)

- Mô tả: Context của request hiện tại; các header response của nó bị thay đổi trực tiếp (mutate in place).

- Tham số thứ 2: `ctx.Next` (`next`)

- Mô tả: Được gọi để chuyển quyền xử lý cho handler kế tiếp trong chuỗi.

#### Returns
Không có.

#### Cách Dùng

```go
cors.CORS{AllowOrigin: "https://example.com"}.Use(c, next)
```

## Benchmark

Được ghi lại bằng cách chạy `go test -run=^$ -bench=. -benchmem ./middlewares/cors/...`. Các số liệu phụ thuộc vào máy chạy và được ghi lại tại thời điểm tạo tài liệu — hãy tự chạy lại lệnh này để có baseline mới.

```console
goos: darwin
goarch: amd64
pkg: github.com/dangduoc08/ginject/middlewares/cors
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkCORS_Use_StarOrigin-12          	 4616331	       250.1 ns/op	      16 B/op	       1 allocs/op
BenchmarkCORS_Use_NilOriginDefault-12    	 4937156	       245.7 ns/op	      16 B/op	       1 allocs/op
BenchmarkCORS_Use_OriginMap-12           	 3482049	       345.5 ns/op	      32 B/op	       2 allocs/op
BenchmarkCORS_Use_Preflight-12           	  709976	      1646 ns/op	     256 B/op	       9 allocs/op
BenchmarkAllowedOrigin_Wildcard-12       	422648827	         3.038 ns/op	       0 B/op	       0 allocs/op
BenchmarkAllowedOrigin_Map-12            	100000000	        12.13 ns/op	       0 B/op	       0 allocs/op
BenchmarkAllowedOrigin_Regexp-12         	 2383460	       550.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkLoadCORSOptions-12              	 9613004	       148.2 ns/op	     144 B/op	       2 allocs/op
PASS
ok  	github.com/dangduoc08/ginject/middlewares/cors	12.736s
```
