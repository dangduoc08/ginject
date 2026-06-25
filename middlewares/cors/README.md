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
- Wildcard `*`, exact origin list, or `*regexp.Regexp` pattern matching for `AllowOrigin`
- Automatic `Vary: Origin` header whenever the response depends on the request's origin
- Spec-compliant credentials handling: echoes the request origin instead of `*` when `IsAllowCredentials` is set
- Blocks the `null` origin when credentials are enabled
- Preflight short-circuit: responds with the configured success status without calling `next`, unless `IsPreflightContinue` is set

## Usage

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

Controls which origins are permitted. A `string` is compared verbatim (`"*"` allows every origin); a `[]string` is matched exactly, with any trailing `/` trimmed from each configured origin before comparison; a `*regexp.Regexp` is matched with `MatchString`.

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
Type: `int` (milliseconds)

Default: `5000` (5 seconds)

Required: `false`

How long the browser may cache a preflight response; converted to seconds for the `Access-Control-Max-Age` header.

```go
cors.CORS{MaxAge: 86400000} // sets the header to "86400"
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
- An `AllowOrigin` `[]string` echoes the request origin and sets `Vary: Origin` when it matches (trailing `/` in the configured origin is ignored), and sets no header when it doesn't match; an empty list blocks every origin (`TestCORS_Use_SpecificOriginMap`, `TestCORS_Use_OriginTrailingSlashMatchesRequest`, `TestCORS_Use_SpecificOriginMapBlocked`, `TestCORS_Use_EmptySliceBlocksAllOrigins`).
- An `AllowOrigin` `*regexp.Regexp` echoes the request origin and sets `Vary: Origin` only when it matches (`TestCORS_Use_RegexpOrigin`, `TestCORS_Use_RegexpOriginNoMatch`).
- `Vary: Origin` is set whenever a specific origin is echoed, but never for the bare wildcard `"*"` (`TestCORS_Use_VaryForSpecificStringOrigin`, `TestCORS_Use_NoVaryForWildcard`).
- `IsAllowCredentials` sets `Access-Control-Allow-Credentials: true` (`TestCORS_Use_Credentials`).
- Wildcard `AllowOrigin` combined with `IsAllowCredentials` echoes the request origin instead of `*` and sets `Vary: Origin`, except when the request origin is `"null"`, which is never reflected (`TestCORS_Use_CredentialsWithWildcardEchosOrigin`, `TestCORS_Use_NullOriginWithCredentialsBlocked`).
- Wildcard `AllowOrigin` without `IsAllowCredentials` still sets `Access-Control-Allow-Origin: *` even for a `"null"` request origin (`TestCORS_Use_NullOriginWildcardNoCredentials`).
- `Access-Control-Allow-Methods`, `Access-Control-Max-Age`, and `Access-Control-Allow-Headers` are only set on `OPTIONS` (preflight) requests, never on other methods (`TestCORS_Use_PreflightOnlyHeaders`).
- A custom `AllowMethods` list is reflected in `Access-Control-Allow-Methods` on preflight (`TestCORS_Use_CustomAllowMethodsOnPreflight`).
- A string `AllowHeaders`/`ExposeHeaders` is passed through verbatim instead of being joined (`TestCORS_Use_AllowHeadersString`, `TestCORS_Use_ExposeHeadersString`).
- `next` is always called for non-`OPTIONS` requests (`TestCORS_Use_NextCalledForNonOptions`).
- For `OPTIONS` requests, `next` is called only when `IsPreflightContinue` is `true`; otherwise the response is written immediately with `OptionsSuccessStatus` (default `204`, or a configured value) and `next` is not called (`TestCORS_Use_OptionsPreflightContinue`, `TestCORS_Use_OptionsPreflightStatus`, `TestCORS_Use_CustomOptionsSuccessStatus`).

#### Parameters
- 1st parameter: `*ctx.Context` (`c`)

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

Captured by running `go test -run=^$ -bench=. -benchmem ./middlewares/cors/...`. Numbers are machine-dependent and were captured at doc-generation time — re-run the command yourself for a fresh baseline.

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
