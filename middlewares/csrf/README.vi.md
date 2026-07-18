# Middleware CSRF

*`csrf` hiện thực mẫu (pattern) double-submit-cookie dưới dạng một middleware của Ginject, phát ra một cookie chứa token và xác minh token đó so với một header hoặc form field trên các request làm thay đổi trạng thái (state-changing).*

- [Middleware CSRF](#middleware-csrf)
  - [Tính Năng Chính](#tính-năng-chính)
  - [Cách Dùng](#cách-dùng)
  - [Cách Hoạt Động](#cách-hoạt-động)
  - [Struct `CSRF`](#struct-csrf)
    - [TokenLength](#tokenlength)
    - [CookieName](#cookiename)
    - [HeaderName](#headername)
    - [ContextKey](#contextkey)
  - [Hàm](#hàm)
    - [GenerateCSRFToken](#generatecsrftoken)
    - [CompareTokensSecurely](#comparetokenssecurely)
  - [Phương Thức Của `CSRF`](#phương-thức-của-csrf)
    - [NewMiddleware](#newmiddleware)
    - [Use](#use)
  - [Benchmark](#benchmark)

## Tính Năng Chính
- Mẫu double-submit-cookie: không cần lưu trữ session ở phía server
- Request `GET`/`HEAD`/`OPTIONS` luôn đi qua nguyên vẹn; chỉ các method làm thay đổi trạng thái mới bị xác minh
- Nhận token được submit từ một header tùy chỉnh, header `X-XSRF-TOKEN`, hoặc form field `_csrf`, theo đúng thứ tự ưu tiên đó
- So sánh token theo thời gian không đổi (constant-time) để chống timing attack
- Token đang dùng được expose cho các handler kế tiếp qua request context

## Cách Dùng

```go
package main

import (
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/middlewares/csrf"
)

func main() {
	app := core.New()

	app.BindGlobalMiddlewares(csrf.CSRF{
		CookieName: "_csrf",
		HeaderName: "X-CSRF-Token",
	})

	app.Create(
		core.ModuleBuilder().Build(),
	)
}
```

`CSRF` cũng hoạt động khi gắn theo từng controller riêng, vì nó thỏa mãn trực tiếp interface `common.MiddlewareFn`:

```go
func (c FormController) NewController() core.Controller {
	c.BindMiddleware(csrf.CSRF{})
	return c
}
```

## Cách Hoạt Động

1. Với mỗi request, `Use` đọc token từ cookie đã cấu hình. Nếu cookie thiếu hoặc trống, nó sẽ sinh token mới bằng `GenerateCSRFToken` và đặt token đó vào một cookie không `HttpOnly`, để JavaScript phía client có thể đọc lại cho request kế tiếp.
2. Token đã xác định được lưu vào request context dưới key `ContextKey`, có thể truy cập từ handler qua `c.Request.Context().Value(...)`.
3. Với các method an toàn (`GET`, `HEAD`, `OPTIONS`), `next` được gọi ngay lập tức — không có việc xác minh nào xảy ra.
4. Với mọi method khác, token được submit sẽ được đọc từ `HeaderName`, sau đó `X-XSRF-TOKEN`, rồi form field `_csrf`, và được so sánh với token trong cookie bằng `CompareTokensSecurely`. Nếu không khớp, hàm sẽ panic với `exception.ForbiddenException`.

## Struct `CSRF`

### TokenLength
Type: `int`

Default: `32` (số byte entropy, được encode thành 64 ký tự hex)

Required: `false`

Một giá trị bằng 0 hoặc âm sẽ dùng giá trị mặc định.

```go
csrf.CSRF{TokenLength: 64}
```

### CookieName
Type: `string`

Default: `"_csrf"`

Required: `false`

```go
csrf.CSRF{CookieName: "my_csrf"}
```

### HeaderName
Type: `string`

Default: `"X-CSRF-Token"`

Required: `false`

Header được kiểm tra đầu tiên cho token được submit trên các request làm thay đổi trạng thái. `X-XSRF-TOKEN` luôn được kiểm tra như một phương án dự phòng, bất kể cấu hình này.

```go
csrf.CSRF{HeaderName: "X-My-CSRF"}
```

### ContextKey
Type: `string`

Default: `"csrf_token"`

Required: `false`

Key của request context, dưới đó token đang dùng được lưu để các handler kế tiếp đọc.

```go
csrf.CSRF{ContextKey: "my_key"}
```

## Hàm

### GenerateCSRFToken

Trả về một token ngẫu nhiên với độ an toàn mật mã, được encode dưới dạng hex.

#### Rules
- `length` bằng 32 cho ra chuỗi hex 64 ký tự, tức là output luôn dài `2 × length` ký tự hex (`TestGenerateCSRFToken_Length`).
- `length` bằng `0` (hoặc bất kỳ giá trị không dương nào) sẽ dùng giá trị mặc định 32 byte, cho ra token 64 ký tự (`TestGenerateCSRFToken_ZeroLengthUsesDefault`).
- Các lần gọi liên tiếp tạo ra các token khác nhau (`TestGenerateCSRFToken_Uniqueness`).
- Output chỉ chứa ký tự hex viết thường (`0`-`9`, `a`-`f`) (`TestGenerateCSRFToken_OnlyHexChars`).

#### Parameters
- Tham số thứ 1: `int` (`length`)

- Mô tả: Số byte ngẫu nhiên cần sinh; giá trị không dương sẽ dùng mặc định của package là 32.

#### Returns
- Giá trị thứ 1: `string`

- Mô tả: Token ngẫu nhiên đã encode hex, dài `2 × length` ký tự.

- Giá trị thứ 2: `error`

- Mô tả: Khác nil nếu nguồn sinh số ngẫu nhiên không tạo đủ byte.

#### Cách Dùng

```go
token, err := csrf.GenerateCSRFToken(32)
if err != nil {
	panic(err)
}
fmt.Println(token)
```

### CompareTokensSecurely

So sánh hai token theo thời gian không đổi (constant-time) để chống timing attack.

#### Rules
- Hai chuỗi bằng nhau, không trống, so sánh bằng nhau (`TestCompareTokensSecurely_Equal`).
- Hai chuỗi khác nhau so sánh không bằng nhau (`TestCompareTokensSecurely_Unequal`).
- Hai chuỗi trống so sánh bằng nhau (`TestCompareTokensSecurely_EmptyBothEqual`).
- Một chuỗi không trống so sánh với một chuỗi trống cho kết quả không bằng nhau (`TestCompareTokensSecurely_OneEmpty`).

#### Parameters
- Tham số thứ 1: `string` (`a`)

- Mô tả: Token đầu tiên cần so sánh.

- Tham số thứ 2: `string` (`b`)

- Mô tả: Token thứ hai cần so sánh.

#### Returns
- Giá trị thứ 1: `bool`

- Mô tả: `true` nếu `a` và `b` bằng nhau.

#### Cách Dùng

```go
if !csrf.CompareTokensSecurely(submitted, expected) {
	panic("token mismatch")
}
```

## Phương Thức Của `CSRF`

### NewMiddleware

Biên dịch các field của struct `CSRF` thành một giá trị options nội bộ một lần duy nhất, và trả về một `common.MiddlewareFn` tái sử dụng các option đã biên dịch đó cho mỗi request.

#### Rules
- Trả về một giá trị thuộc kiểu middleware đã biên dịch nội bộ của package, khác với struct `CSRF` chính nó (`TestCSRF_NewMiddleware_ReturnsCompiledCSRF`).

#### Parameters
Không có.

#### Returns
- Giá trị thứ 1: `common.MiddlewareFn`

- Mô tả: Một middleware sẵn sàng để gắn (bind) qua `BindGlobalMiddlewares` hoặc `BindMiddleware`, với các option CSRF đã được biên dịch một lần.

#### Cách Dùng

```go
mw := csrf.CSRF{TokenLength: 16}.NewMiddleware()
app.BindGlobalMiddlewares(mw)
```

### Use

Phát ra hoặc tái sử dụng cookie CSRF, expose token qua request context, và xác minh token được submit trên các request làm thay đổi trạng thái.

#### Rules
- Request `GET`, `HEAD`, và `OPTIONS` luôn gọi `next` mà không xác minh token nào (`TestCSRF_SafeMethod_GET`, `TestCSRF_SafeMethod_HEAD`, `TestCSRF_SafeMethod_OPTIONS`).
- Nếu cookie đã cấu hình bị thiếu, một token mới được sinh ra và đặt vào cookie có tên đó (`TestCSRF_SetsCookieWhenMissing`).
- Nếu cookie đã cấu hình đã có giá trị, không có cookie mới nào được đặt — token hiện có được tái sử dụng (`TestCSRF_ReusesExistingCookie`).
- Token đã xác định được lưu vào request context dưới key `ContextKey`, có thể đọc qua `c.Request.Context().Value(ContextKey)` (`TestCSRF_StoresTokenInContext`).
- Với request `POST`, `PUT`, `PATCH`, và `DELETE`, một token được submit qua `HeaderName` khớp với token trong cookie sẽ cho phép request đi qua (`TestCSRF_POST_ValidHeader`, `TestCSRF_PUT_ValidHeader`, `TestCSRF_PATCH_ValidHeader`, `TestCSRF_DELETE_ValidHeader`).
- Một token được submit qua header `X-XSRF-TOKEN` cũng được chấp nhận, ngay cả khi `HeaderName` chưa được đặt và dùng giá trị mặc định (`TestCSRF_POST_ValidAltHeader`).
- Một token được submit qua form field `_csrf` được chấp nhận khi không có token nào trong header (`TestCSRF_POST_ValidFormField`).
- Một token được submit bị thiếu, trống, không khớp, hoặc không hợp lệ theo cách khác sẽ panic với một exception CSRF thay vì gọi `next` (`TestCSRF_POST_MissingToken_Panics`, `TestCSRF_POST_EmptyToken_Panics`, `TestCSRF_POST_WrongToken_Panics`, `TestCSRF_POST_SpecialCharsToken_Panics`).
- An toàn khi gọi đồng thời từ nhiều goroutine, cho cả request an toàn và request làm thay đổi trạng thái (`TestCSRF_ConcurrentSafeRequests`, `TestCSRF_ConcurrentStateChanging`).

#### Parameters
- Tham số thứ 1: `*ctx.HTTPContext` (`c`)

- Mô tả: HTTPContext của request hiện tại; cookie, header response, và request context của nó bị thay đổi/đọc.

- Tham số thứ 2: `ctx.Next` (`next`)

- Mô tả: Được gọi để chuyển quyền xử lý cho handler kế tiếp trong chuỗi.

#### Returns
Không có.

#### Cách Dùng

```go
csrf.CSRF{}.Use(c, next)
```

## Benchmark

Được ghi lại bằng cách chạy `go test -run=^$ -bench=. -benchmem ./middlewares/csrf/...`. Các số liệu phụ thuộc vào máy chạy và được ghi lại tại thời điểm tạo tài liệu — hãy tự chạy lại lệnh này để có baseline mới.

```console
goos: darwin
goarch: amd64
pkg: github.com/dangduoc08/ginject/middlewares/csrf
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkGenerateCSRFToken-12                	 1887820	       779.3 ns/op	     128 B/op	       2 allocs/op
BenchmarkCompareTokensSecurely_Equal-12      	41467776	        26.68 ns/op	       0 B/op	       0 allocs/op
BenchmarkCompareTokensSecurely_Unequal-12    	43337484	        24.78 ns/op	       0 B/op	       0 allocs/op
BenchmarkCSRF_SafeMethod_NoCookie-12         	  318858	      4630 ns/op	    7035 B/op	      27 allocs/op
BenchmarkCSRF_SafeMethod_WithCookie-12       	  273988	      4002 ns/op	    6984 B/op	      29 allocs/op
BenchmarkCSRF_POST_ValidHeader-12            	  291968	      4036 ns/op	    7016 B/op	      32 allocs/op
BenchmarkLoadCSRFOptions-12                  	94848530	        10.89 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	github.com/dangduoc08/ginject/middlewares/csrf	10.179s
```
