# CORS Middleware

*`cors` implements the [Fetch Standard CORS protocol](https://fetch.spec.whatwg.org/#http-cors-protocol) as a Ginject middleware, handling both simple requests and `OPTIONS` preflight requests.*

- [CORS Middleware](#cors-middleware)
  - [Key Features](#key-features)
  - [Usage](#usage)
  - [`CORS` Struct](#cors-struct)
    - [AllowOrigin](#alloworigin)
    - [AllowHeaders](#allowheaders)
    - [ExposeHeaders](#exposeheaders)
    - [AllowMethods](#allowmethods)
    - [MaxAge](#maxage)
    - [IsAllowCredentials](#isallowcredentials)
    - [IsPreflightContinue](#ispreflightcontinue)
    - [OptionsSuccessStatus](#optionssuccessstatus)
  - [`CORS` Methods](#cors-methods)
    - [NewMiddleware](#newmiddleware)
    - [Use](#use)
  - [Benchmarks](#benchmarks)

## Key Features
- Wildcard `*`, single origin, exact origin list, or `*regexp.Regexp` pattern matching for `AllowOrigin` — all share the same matching logic for both HTTP and WebSocket requests
- Automatic `Vary: Origin` header whenever the response depends on the request's origin, even when this particular origin was rejected (so caches never serve a CORS response to the wrong origin)
- `Vary` is merged into, never overwrites, whatever other middleware already set, with case-insensitive de-duplication
- Trailing `/` is trimmed consistently from every `AllowOrigin` shape (`string`, `[]string`) before comparison
- Spec-compliant credentials handling: echoes the request origin instead of `*` when `IsAllowCredentials` is set
- Blocks the `null` origin when credentials are enabled
- Preflight short-circuit: responds with the configured success status without calling `next`, unless `IsPreflightContinue` is set
- All parsing/joining/normalization happens once, in `NewMiddleware`; per-request handling never allocates configuration state

## Usage

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

`CORS` also works scoped to a single controller, since it satisfies `common.MiddlewareFn` directly:

```go
func (c UserController) NewController() core.Controller {
	c.BindMiddleware(cors.CORS{AllowOrigin: "https://admin.example.com"})
	return c
}
```

## `CORS` Struct

### AllowOrigin
Type: `any` (`string` | `[]string` | `*regexp.Regexp`)

Default: `"*"` (set by `NewMiddleware`/`Use` when left `nil`)

Required: `false`

Controls which origins are permitted. The literal string `"*"` allows every origin. Any other `string`, or a `[]string`, is matched exactly against the request's `Origin` header (trailing `/` is trimmed from configured origins before comparison, so `"https://app.example.com/"` and `"https://app.example.com"` are equivalent); a non-matching origin gets no CORS headers at all. A `*regexp.Regexp` is matched with `MatchString`. HTTP and WebSocket requests are checked with the exact same matching function.

```go
cors.CORS{AllowOrigin: []string{"https://app.example.com", "https://admin.example.com"}}
```

### AllowHeaders
Type: `any` (`string` | `[]string`)

Default: unset — the preflight response echoes back the request's `Access-Control-Request-Headers` value

Required: `false`

```go
cors.CORS{AllowHeaders: []string{"Content-Type", "Authorization"}}
```

### ExposeHeaders
Type: `any` (`string` | `[]string`)

Default: unset — `Access-Control-Expose-Headers` is not sent

Required: `false`

```go
cors.CORS{ExposeHeaders: []string{"X-Request-ID"}}
```

### AllowMethods
Type: `[]string`

Default: `[]string{"GET", "HEAD", "PUT", "PATCH", "POST", "DELETE"}`

Required: `false`

Sent as `Access-Control-Allow-Methods` on preflight responses.

```go
cors.CORS{AllowMethods: []string{"GET", "POST"}}
```

### MaxAge
Type: `time.Duration`

Default: `5 * time.Second`

Required: `false`

How long the browser may cache a preflight response; truncated to whole seconds for the `Access-Control-Max-Age` header. A zero or negative value falls back to the default.

```go
cors.CORS{MaxAge: 24 * time.Hour} // sets the header to "86400"
```

### IsAllowCredentials
Type: `bool`

Default: `false`

Required: `false`

Sets `Access-Control-Allow-Credentials: true`. When combined with `AllowOrigin: "*"`, the actual request origin is echoed instead of `*`.

```go
cors.CORS{IsAllowCredentials: true}
```

### IsPreflightContinue
Type: `bool`

Default: `false`

Required: `false`

When `true`, `next` is called after preflight headers are set, passing control to the next handler. When `false`, the preflight request is short-circuited and a response is written immediately.

```go
cors.CORS{IsPreflightContinue: true}
```

### OptionsSuccessStatus
Type: `int`

Default: `204`

Required: `false`

HTTP status written for a short-circuited preflight response (only applies when `IsPreflightContinue` is `false`). Some legacy browsers require `200`.

```go
cors.CORS{OptionsSuccessStatus: 200}
```

## `CORS` Methods

### NewMiddleware

Compiles the `CORS` struct's fields into an internal options value once, and returns a `common.MiddlewareFn` that reuses those compiled options on every request. Prefer this over calling `Use` directly when registering the middleware once at startup.

#### Parameters
None.

#### Returns
- 1st value: `common.MiddlewareFn`

- Description: A middleware ready to bind via `BindGlobalMiddlewares` or `BindMiddleware`, with its CORS options compiled once.

#### Usage

```go
mw := cors.CORS{AllowOrigin: "https://example.com"}.NewMiddleware()
app.BindGlobalMiddlewares(mw)
```

### Use

Applies CORS headers to the current request and either calls `next` or short-circuits a preflight response. Recompiles the struct's options on every call (unlike the middleware returned by `NewMiddleware`, which compiles them once), but lets a `CORS` value be passed directly anywhere a `common.MiddlewareFn` is expected.

#### Rules
- With no `Origin` request header, no CORS headers are set and `next` is called immediately (`TestCORS_Use_NoOriginHeaderSkipsCORS`).
- A nil `AllowOrigin` defaults to `"*"`, setting `Access-Control-Allow-Origin: *` (`TestCORS_Use_SetsOriginStarByDefault`).
- A single-`string` `AllowOrigin` (other than `"*"`) is matched exactly: it echoes the request origin only when they match, and sets no header — including no fallback to the configured value — when they don't (`TestCORS_Use_SpecificStringOriginBlocked`).
- An `AllowOrigin` `[]string` echoes the request origin when it matches (trailing `/` in the configured origin is ignored), and sets no header when it doesn't match; an empty list blocks every origin (`TestCORS_Use_SpecificOriginMap`, `TestCORS_Use_OriginTrailingSlashMatchesRequest`, `TestCORS_Use_SpecificOriginMapBlocked`, `TestCORS_Use_EmptySliceBlocksAllOrigins`).
- An `AllowOrigin` `*regexp.Regexp` echoes the request origin only when it matches (`TestCORS_Use_RegexpOrigin`, `TestCORS_Use_RegexpOriginNoMatch`).
- `Vary: Origin` is set whenever `AllowOrigin` is anything other than the bare wildcard `"*"` without credentials — including when this particular request's origin was rejected, since the response still varies by origin for other callers (`TestCORS_Use_VaryForSpecificStringOrigin`, `TestCORS_Use_VaryOriginSetEvenWhenOriginIsBlocked`, `TestCORS_Use_NoVaryForWildcard`).
- `Vary` tokens are merged into any value already set by earlier middleware (never overwritten) and de-duplicated case-insensitively (`TestCORS_Use_VaryMergesWithExistingHeader`, `TestCORS_Use_VaryNoDuplicateWhenAlreadyPresent`).
- `IsAllowCredentials` sets `Access-Control-Allow-Credentials: true` (`TestCORS_Use_Credentials`).
- Wildcard `AllowOrigin` combined with `IsAllowCredentials` echoes the request origin instead of `*` and sets `Vary: Origin`, except when the request origin is `"null"`, which is never reflected (`TestCORS_Use_CredentialsWithWildcardEchosOrigin`, `TestCORS_Use_NullOriginWithCredentialsBlocked`).
- Wildcard `AllowOrigin` without `IsAllowCredentials` still sets `Access-Control-Allow-Origin: *` even for a `"null"` request origin (`TestCORS_Use_NullOriginWildcardNoCredentials`).
- `Access-Control-Allow-Methods`, `Access-Control-Max-Age`, and `Access-Control-Allow-Headers` are only set on `OPTIONS` (preflight) requests, never on other methods (`TestCORS_Use_PreflightOnlyHeaders`).
- A custom `AllowMethods` list is reflected in `Access-Control-Allow-Methods` on preflight (`TestCORS_Use_CustomAllowMethodsOnPreflight`).
- A string `AllowHeaders`/`ExposeHeaders` is passed through verbatim instead of being joined (`TestCORS_Use_AllowHeadersString`, `TestCORS_Use_ExposeHeadersString`).
- `next` is always called for non-`OPTIONS` requests (`TestCORS_Use_NextCalledForNonOptions`).
- For `OPTIONS` requests, `next` is called only when `IsPreflightContinue` is `true`; otherwise the response is written immediately with `OptionsSuccessStatus` (default `204`, or a configured value) and `next` is not called (`TestCORS_Use_OptionsPreflightContinue`, `TestCORS_Use_OptionsPreflightStatus`, `TestCORS_Use_CustomOptionsSuccessStatus`).
- WebSocket requests (`ctx.WSType`) use the exact same origin-matching rules as HTTP, but never write response headers — a rejected origin simply skips `next`.

#### Parameters
- 1st parameter: `*ctx.HTTPContext` (`c`)

- Description: The current request context; its response headers are mutated in place.

- 2nd parameter: `ctx.Next` (`next`)

- Description: Called to pass control to the next handler in the chain.

#### Returns
None.

#### Usage

```go
cors.CORS{AllowOrigin: "https://example.com"}.Use(c, next)
```

## Benchmarks

Captured by running `go test -run=^$ -bench=. -benchmem ./middlewares/cors/...`. Numbers are machine-dependent and were captured at doc-generation time — re-run the command yourself for a fresh baseline. `BenchmarkCORS_Use_*` give each iteration its own response recorder (like a real request would), so their allocations reflect the true per-request cost; `BenchmarkLoadCORSOptions` is the one-time, per-`NewMiddleware`-call compile step, not a per-request cost.

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
