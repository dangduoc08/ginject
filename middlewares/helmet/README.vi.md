# Middleware Helmet

*`helmet` đặt một loạt header HTTP response liên quan tới bảo mật (CSP, HSTS, các chính sách frame/cross-origin/referrer) dưới dạng một middleware của Ginject, lấy cảm hứng từ package `helmet` của Node.js.*

- [Middleware Helmet](#middleware-helmet)
  - [Tính Năng Chính](#tính-năng-chính)
  - [Cách Dùng](#cách-dùng)
  - [Các Header Được Helmet Đặt](#các-header-được-helmet-đặt)
  - [Struct `Helmet`](#struct-helmet)
    - [ContentSecurityPolicy](#contentsecuritypolicy)
    - [CrossOriginEmbedderPolicy](#crossoriginembedderpolicy)
    - [CrossOriginOpenerPolicy](#crossoriginopenerpolicy)
    - [CrossOriginResourcePolicy](#crossoriginresourcepolicy)
    - [DNSPrefetchControl](#dnsprefetchcontrol)
    - [FrameOptions](#frameoptions)
    - [HSTSMaxAge, HSTSExcludeSubDomains, HSTSPreload, DisableHSTS](#hstsmaxage-hstsexcludesubdomains-hstspreload-disablehsts)
    - [PermittedCrossDomainPolicies](#permittedcrossdomainpolicies)
    - [ReferrerPolicy](#referrerpolicy)
  - [Phương Thức Của `Helmet`](#phương-thức-của-helmet)
    - [NewMiddleware](#newmiddleware)
    - [Use](#use)
  - [Benchmark](#benchmark)

## Tính Năng Chính
- Giá trị mặc định an toàn hợp lý cho mọi header — chỉ cần đăng ký `Helmet{}` mà không đặt field nào, response vẫn được "hardened"
- Content-Security-Policy mặc định nghiêm ngặt, bao gồm `default-src`, `script-src`, `style-src`, `object-src`, và nhiều hơn nữa
- `Strict-Transport-Security` (HSTS) được bật mặc định, kèm các tùy chọn cho max-age, subdomain, và preload
- Mỗi field policy có thể được override riêng lẻ mà không ảnh hưởng tới các field khác

## Cách Dùng

```go
package main

import (
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/middlewares/helmet"
)

func main() {
	app := core.New()

	app.BindGlobalMiddlewares(helmet.Helmet{})

	app.Create(
		core.ModuleBuilder().Build(),
	)
}
```

`Helmet` cũng hoạt động khi gắn theo từng controller riêng, vì nó thỏa mãn trực tiếp interface `common.MiddlewareFn`:

```go
func (c PageController) NewController() core.Controller {
	c.BindMiddleware(helmet.Helmet{FrameOptions: "DENY"})
	return c
}
```

## Các Header Được Helmet Đặt

Các header dưới đây luôn được `Use`/`NewMiddleware` đặt và không thể cấu hình:

| Header | Giá trị |
|---|---|
| `X-Content-Type-Options` | `nosniff` |
| `X-Download-Options` | `noopen` |
| `X-XSS-Protection` | `0` |
| `Origin-Agent-Cluster` | `?1` |

Các header bên dưới đây có thể cấu hình thông qua các field của struct `Helmet`, được ghi lại lần lượt ở phần tiếp theo.

## Struct `Helmet`

### ContentSecurityPolicy
Type: `string`

Default: `"default-src 'self';base-uri 'self';font-src 'self' https: data:;form-action 'self';frame-ancestors 'self';img-src 'self' data:;object-src 'none';script-src 'self';script-src-attr 'none';style-src 'self' https: 'unsafe-inline';upgrade-insecure-requests"`

Required: `false`

Đặt header `Content-Security-Policy`.

```go
helmet.Helmet{ContentSecurityPolicy: "default-src 'none'"}
```

### CrossOriginEmbedderPolicy
Type: `string`

Default: `"require-corp"`

Required: `false`

Đặt header `Cross-Origin-Embedder-Policy`.

```go
helmet.Helmet{CrossOriginEmbedderPolicy: "unsafe-none"}
```

### CrossOriginOpenerPolicy
Type: `string`

Default: `"same-origin"`

Required: `false`

Đặt header `Cross-Origin-Opener-Policy`.

```go
helmet.Helmet{CrossOriginOpenerPolicy: "same-origin-allow-popups"}
```

### CrossOriginResourcePolicy
Type: `string`

Default: `"same-origin"`

Required: `false`

Đặt header `Cross-Origin-Resource-Policy`.

```go
helmet.Helmet{CrossOriginResourcePolicy: "cross-origin"}
```

### DNSPrefetchControl
Type: `string`

Default: `"off"`

Required: `false`

Đặt header `X-DNS-Prefetch-Control`.

```go
helmet.Helmet{DNSPrefetchControl: "on"}
```

### FrameOptions
Type: `string`

Default: `"SAMEORIGIN"`

Required: `false`

Đặt header `X-Frame-Options`.

```go
helmet.Helmet{FrameOptions: "DENY"}
```

### HSTSMaxAge, HSTSExcludeSubDomains, HSTSPreload, DisableHSTS
Type: lần lượt là `int`, `bool`, `bool`, `bool`

Default: `HSTSMaxAge: 0` (→ `15552000`, 180 ngày), `HSTSExcludeSubDomains: false` (subdomain được bao gồm), `HSTSPreload: false`, `DisableHSTS: false`

Required: `false`

Bốn field này cùng nhau xây dựng header `Strict-Transport-Security` dưới dạng `max-age=<HSTSMaxAge>[; includeSubDomains][; preload]`. `HSTSMaxAge` bằng 0 sẽ dùng giá trị mặc định `15552000`. Đặt `DisableHSTS` thành `true` sẽ bỏ hẳn header này, override cả ba field còn lại.

```go
helmet.Helmet{
	HSTSMaxAge:            86400,
	HSTSExcludeSubDomains: true,
	HSTSPreload:           true,
} // -> Strict-Transport-Security: max-age=86400; preload

helmet.Helmet{DisableHSTS: true} // -> header bị bỏ qua
```

### PermittedCrossDomainPolicies
Type: `string`

Default: `"none"`

Required: `false`

Đặt header `X-Permitted-Cross-Domain-Policies`.

```go
helmet.Helmet{PermittedCrossDomainPolicies: "master-only"}
```

### ReferrerPolicy
Type: `string`

Default: `"no-referrer"`

Required: `false`

Đặt header `Referrer-Policy`.

```go
helmet.Helmet{ReferrerPolicy: "strict-origin"}
```

## Phương Thức Của `Helmet`

### NewMiddleware

Biên dịch các field của struct `Helmet` thành một giá trị options nội bộ một lần duy nhất, và trả về một `common.MiddlewareFn` tái sử dụng các option đã biên dịch đó cho mỗi request.

#### Parameters
Không có.

#### Returns
- Giá trị thứ 1: `common.MiddlewareFn`

- Mô tả: Một middleware sẵn sàng để gắn (bind) qua `BindGlobalMiddlewares` hoặc `BindMiddleware`, với các header đã được biên dịch một lần.

#### Cách Dùng

```go
mw := helmet.Helmet{FrameOptions: "DENY"}.NewMiddleware()
app.BindGlobalMiddlewares(mw)
```

### Use

Đặt mọi header response của Helmet cho request hiện tại và luôn gọi `next` — Helmet không bao giờ short-circuit một request.

#### Rules
- `next` luôn được gọi (`TestHelmet_Use_CallsNext`).
- Bốn header không thể cấu hình luôn được đặt giá trị cố định: `X-Content-Type-Options: nosniff`, `X-Download-Options: noopen`, `X-XSS-Protection: 0`, `Origin-Agent-Cluster: ?1` (`TestHelmet_Use_SetsXContentTypeOptions`, `TestHelmet_Use_SetsXDownloadOptions`, `TestHelmet_Use_SetsXXSSProtectionToZero`, `TestHelmet_Use_SetsOriginAgentCluster`).
- Khi không đặt field nào, `Content-Security-Policy` là chuỗi mặc định của package; đặt `ContentSecurityPolicy` sẽ override nó nguyên văn (`TestHelmet_Use_SetsDefaultCSP`, `TestHelmet_Use_SetsCustomCSP`).
- Khi không đặt field nào, `Strict-Transport-Security` là `"max-age=15552000; includeSubDomains"`; đặt `DisableHSTS: true` sẽ bỏ hẳn header này (`TestHelmet_Use_SetsDefaultHSTS`, `TestHelmet_Use_SkipsHSTSWhenDisabled`).
- Khi không đặt field nào, `X-Frame-Options` là `"SAMEORIGIN"`; `FrameOptions` sẽ override nó (`TestHelmet_Use_SetsDefaultFrameOptions`, `TestHelmet_Use_SetsCustomFrameOptions`).
- Khi không đặt field nào, `Referrer-Policy` là `"no-referrer"`; `ReferrerPolicy` sẽ override nó (`TestHelmet_Use_SetsDefaultReferrerPolicy`, `TestHelmet_Use_SetsCustomReferrerPolicy`).
- Khi không đặt field nào, `Cross-Origin-Embedder-Policy`, `Cross-Origin-Opener-Policy`, và `Cross-Origin-Resource-Policy` lần lượt mặc định là `"require-corp"`, `"same-origin"`, và `"same-origin"` (`TestHelmet_Use_SetsCrossOriginHeaders`).
- Khi không đặt field nào, `X-DNS-Prefetch-Control` là `"off"` (`TestHelmet_Use_SetsDefaultDNSPrefetchControl`).
- Khi không đặt field nào, `X-Permitted-Cross-Domain-Policies` là `"none"` (`TestHelmet_Use_SetsDefaultPermittedCrossDomainPolicies`).

#### Parameters
- Tham số thứ 1: `*ctx.HTTPContext` (`c`)

- Mô tả: HTTPContext của request hiện tại; các header response của nó bị thay đổi trực tiếp (mutate in place).

- Tham số thứ 2: `ctx.Next` (`next`)

- Mô tả: Được gọi để chuyển quyền xử lý cho handler kế tiếp trong chuỗi.

#### Returns
Không có.

#### Cách Dùng

```go
helmet.Helmet{}.Use(c, next)
```

## Benchmark

Được ghi lại bằng cách chạy `go test -run=^$ -bench=. -benchmem ./middlewares/helmet/...`. Các số liệu phụ thuộc vào máy chạy và được ghi lại tại thời điểm tạo tài liệu — hãy tự chạy lại lệnh này để có baseline mới.

```console
goos: darwin
goarch: amd64
pkg: github.com/dangduoc08/ginject/middlewares/helmet
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkHelmet_Use_Defaults-12       	  864396	      1294 ns/op	     248 B/op	      15 allocs/op
BenchmarkHelmet_Use_CustomCSP-12      	  914866	      1288 ns/op	     248 B/op	      15 allocs/op
BenchmarkHelmet_Use_DisableHSTS-12    	 1000000	      1186 ns/op	     232 B/op	      14 allocs/op
BenchmarkLoadHelmetOptions-12         	 6019968	       202.6 ns/op	     245 B/op	       5 allocs/op
PASS
ok  	github.com/dangduoc08/ginject/middlewares/helmet	6.810s
```
