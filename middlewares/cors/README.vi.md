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
- Khớp `AllowOrigin` theo wildcard `*`, một origin đơn, danh sách origin chính xác, hoặc pattern `*regexp.Regexp` — tất cả dùng chung một logic so khớp cho cả request HTTP lẫn WebSocket
- Tự động thêm header `Vary: Origin` mỗi khi response phụ thuộc vào origin của request, kể cả khi origin đó bị từ chối (để cache không bao giờ trả nhầm response CORS cho origin khác)
- `Vary` được merge vào, chứ không ghi đè, giá trị mà middleware khác đã đặt trước đó, có loại trùng không phân biệt hoa thường
- Dấu `/` ở cuối được loại bỏ nhất quán cho mọi kiểu `AllowOrigin` (`string`, `[]string`) trước khi so sánh
- Xử lý credentials đúng chuẩn: echo lại origin của request thay vì `*` khi `IsAllowCredentials` được đặt
- Chặn origin `null` khi credentials được bật
- Short-circuit cho preflight: trả về status đã cấu hình mà không gọi `next`, trừ khi `IsPreflightContinue` được đặt
- Toàn bộ việc parse/join/chuẩn hóa chỉ diễn ra một lần, trong `NewMiddleware`; xử lý mỗi request không cấp phát thêm cho phần cấu hình

## Cách Dùng

```go
package main

import (
	"time"

	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/middlewares/cors"
)

func main() {
	app := core.New()

	app.BindGlobalMiddlewares(cors.CORS{
		AllowOrigin:        []string{"https://app.example.com"},
		AllowHeaders:       []string{"Content-Type", "Authorization"},
		AllowMethods:       []string{"GET", "POST", "PUT", "DELETE"},
		MaxAge:             24 * time.Hour,
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

Kiểm soát origin nào được phép. Chuỗi `"*"` cho phép mọi origin. Bất kỳ `string` nào khác, hoặc một `[]string`, được so khớp chính xác với header `Origin` của request (dấu `/` ở cuối origin đã cấu hình bị loại bỏ trước khi so sánh, nên `"https://app.example.com/"` và `"https://app.example.com"` là tương đương); origin không khớp sẽ không nhận được header CORS nào cả. Một `*regexp.Regexp` được so khớp bằng `MatchString`. Request HTTP và WebSocket dùng chung một hàm so khớp duy nhất.

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
Type: `time.Duration`

Default: `5 * time.Second`

Required: `false`

Thời gian browser được phép cache một response preflight; bị làm tròn xuống theo giây cho header `Access-Control-Max-Age`. Giá trị bằng 0 hoặc âm sẽ dùng giá trị mặc định.

```go
cors.CORS{MaxAge: 24 * time.Hour} // đặt header thành "86400"
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
- `AllowOrigin` dạng `string` đơn (khác `"*"`) được so khớp chính xác: chỉ echo lại origin của request khi trùng khớp, và không đặt header nào — kể cả không fallback về giá trị đã cấu hình — khi không khớp (`TestCORS_Use_SpecificStringOriginBlocked`).
- `AllowOrigin` dạng `[]string` echo lại origin của request khi khớp (dấu `/` ở cuối origin đã cấu hình bị bỏ qua), và không đặt header nào khi không khớp; một danh sách trống sẽ chặn mọi origin (`TestCORS_Use_SpecificOriginMap`, `TestCORS_Use_OriginTrailingSlashMatchesRequest`, `TestCORS_Use_SpecificOriginMapBlocked`, `TestCORS_Use_EmptySliceBlocksAllOrigins`).
- `AllowOrigin` dạng `*regexp.Regexp` echo lại origin của request chỉ khi khớp pattern (`TestCORS_Use_RegexpOrigin`, `TestCORS_Use_RegexpOriginNoMatch`).
- `Vary: Origin` được đặt bất cứ khi nào `AllowOrigin` khác wildcard `"*"` đơn thuần không kèm credentials — kể cả khi origin của request này bị từ chối, vì response vẫn phụ thuộc vào origin đối với các caller khác (`TestCORS_Use_VaryForSpecificStringOrigin`, `TestCORS_Use_VaryOriginSetEvenWhenOriginIsBlocked`, `TestCORS_Use_NoVaryForWildcard`).
- Các token của `Vary` được merge vào giá trị mà middleware khác đã đặt trước đó (không bao giờ ghi đè) và được loại trùng không phân biệt hoa thường (`TestCORS_Use_VaryMergesWithExistingHeader`, `TestCORS_Use_VaryNoDuplicateWhenAlreadyPresent`).
- `IsAllowCredentials` đặt `Access-Control-Allow-Credentials: true` (`TestCORS_Use_Credentials`).
- `AllowOrigin` wildcard kết hợp với `IsAllowCredentials` sẽ echo lại origin của request thay vì `*` và đặt `Vary: Origin`, ngoại trừ khi origin của request là `"null"`, trường hợp này không bao giờ được echo lại (`TestCORS_Use_CredentialsWithWildcardEchosOrigin`, `TestCORS_Use_NullOriginWithCredentialsBlocked`).
- `AllowOrigin` wildcard không có `IsAllowCredentials` vẫn đặt `Access-Control-Allow-Origin: *` ngay cả khi origin của request là `"null"` (`TestCORS_Use_NullOriginWildcardNoCredentials`).
- `Access-Control-Allow-Methods`, `Access-Control-Max-Age`, và `Access-Control-Allow-Headers` chỉ được đặt cho request `OPTIONS` (preflight), không bao giờ cho các method khác (`TestCORS_Use_PreflightOnlyHeaders`).
- Một danh sách `AllowMethods` tùy chỉnh được phản ánh trong `Access-Control-Allow-Methods` khi preflight (`TestCORS_Use_CustomAllowMethodsOnPreflight`).
- Một `AllowHeaders`/`ExposeHeaders` dạng string được truyền qua nguyên văn thay vì được join (`TestCORS_Use_AllowHeadersString`, `TestCORS_Use_ExposeHeadersString`).
- `next` luôn được gọi đối với request không phải `OPTIONS` (`TestCORS_Use_NextCalledForNonOptions`).
- Đối với request `OPTIONS`, `next` chỉ được gọi khi `IsPreflightContinue` là `true`; ngược lại response được viết ra ngay với `OptionsSuccessStatus` (mặc định `204`, hoặc giá trị đã cấu hình) và `next` không được gọi (`TestCORS_Use_OptionsPreflightContinue`, `TestCORS_Use_OptionsPreflightStatus`, `TestCORS_Use_CustomOptionsSuccessStatus`).
- Request WebSocket (`ctx.WSType`) dùng đúng các quy tắc so khớp origin giống HTTP, nhưng không bao giờ ghi header response — origin bị từ chối chỉ đơn giản là bỏ qua `next`.

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

Được ghi lại bằng cách chạy `go test -run=^$ -bench=. -benchmem ./middlewares/cors/...`. Các số liệu phụ thuộc vào máy chạy và được ghi lại tại thời điểm tạo tài liệu — hãy tự chạy lại lệnh này để có baseline mới. `BenchmarkCORS_Use_*` cấp cho mỗi lần lặp một response recorder riêng (giống một request thật), nên số liệu cấp phát phản ánh đúng chi phí thực tế mỗi request; `BenchmarkLoadCORSOptions` là bước biên dịch một-lần-cho-mỗi-lần-gọi-`NewMiddleware`, không phải chi phí trên mỗi request.

```console
goos: darwin
goarch: amd64
pkg: github.com/dangduoc08/ginject/middlewares/cors
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkCORS_Use_StarOrigin-12          	 2834415	       363.1 ns/op	     528 B/op	       5 allocs/op
BenchmarkCORS_Use_NilOriginDefault-12    	 2751880	       451.0 ns/op	     528 B/op	       5 allocs/op
BenchmarkCORS_Use_OriginMap-12           	 2331158	       534.0 ns/op	     544 B/op	       6 allocs/op
BenchmarkCORS_Use_Preflight-12           	  447320	      3278 ns/op	    1216 B/op	      15 allocs/op
BenchmarkMatchOrigin_Wildcard-12         	320508262	         4.273 ns/op	       0 B/op	       0 allocs/op
BenchmarkMatchOrigin_Map-12              	58212392	        26.58 ns/op	       0 B/op	       0 allocs/op
BenchmarkMatchOrigin_Regexp-12           	 1000000	      1123 ns/op	       0 B/op	       0 allocs/op
BenchmarkLoadCORSOptions-12              	 2898069	       443.4 ns/op	     400 B/op	       4 allocs/op
PASS
ok  	github.com/dangduoc08/ginject/middlewares/cors	14.435s
```
