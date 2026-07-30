# Helmet Middleware

*`helmet` sets a battery of security-related HTTP response headers (CSP, HSTS, frame/cross-origin/referrer policies) as a Ginject middleware, modeled after the Node.js `helmet` package.*

- [Helmet Middleware](#helmet-middleware)
  - [Key Features](#key-features)
  - [Usage](#usage)
  - [Headers Set by Helmet](#headers-set-by-helmet)
  - [`Helmet` Struct](#helmet-struct)
    - [ContentSecurityPolicy](#contentsecuritypolicy)
    - [CrossOriginEmbedderPolicy](#crossoriginembedderpolicy)
    - [CrossOriginOpenerPolicy](#crossoriginopenerpolicy)
    - [CrossOriginResourcePolicy](#crossoriginresourcepolicy)
    - [DNSPrefetchControl](#dnsprefetchcontrol)
    - [FrameOptions](#frameoptions)
    - [HSTSMaxAge, HSTSExcludeSubDomains, HSTSPreload, DisableHSTS](#hstsmaxage-hstsexcludesubdomains-hstspreload-disablehsts)
    - [PermittedCrossDomainPolicies](#permittedcrossdomainpolicies)
    - [ReferrerPolicy](#referrerpolicy)
  - [`Helmet` Methods](#helmet-methods)
    - [NewMiddleware](#newmiddleware)
    - [Use](#use)
  - [Benchmarks](#benchmarks)

## Key Features
- Sensible secure-by-default values for every header — register `Helmet{}` with no fields set and still get a hardened response
- Strict default Content-Security-Policy covering `default-src`, `script-src`, `style-src`, `object-src`, and more
- `Strict-Transport-Security` (HSTS) on by default, with toggles for max-age, subdomains, and preload
- Every policy field can be overridden individually without affecting the others

## Usage

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

`Helmet` also works scoped to a single controller, since it satisfies `common.MiddlewareFn` directly:

```go
func (c PageController) NewController() core.Controller {
	c.BindMiddleware(helmet.Helmet{FrameOptions: "DENY"})
	return c
}
```

## Headers Set by Helmet

These headers are always set by `Use`/`NewMiddleware` and are not configurable:

| Header | Value |
|---|---|
| `X-Content-Type-Options` | `nosniff` |
| `X-Download-Options` | `noopen` |
| `X-XSS-Protection` | `0` |
| `Origin-Agent-Cluster` | `?1` |

The headers below this point are configurable through the `Helmet` struct's fields, documented one by one further down.

## `Helmet` Struct

### ContentSecurityPolicy
Type: `string`

Default: `"default-src 'self';base-uri 'self';font-src 'self' https: data:;form-action 'self';frame-ancestors 'self';img-src 'self' data:;object-src 'none';script-src 'self';script-src-attr 'none';style-src 'self' https: 'unsafe-inline';upgrade-insecure-requests"`

Required: `false`

Sets the `Content-Security-Policy` header.

```go
helmet.Helmet{ContentSecurityPolicy: "default-src 'none'"}
```

### CrossOriginEmbedderPolicy
Type: `string`

Default: `"require-corp"`

Required: `false`

Sets the `Cross-Origin-Embedder-Policy` header.

```go
helmet.Helmet{CrossOriginEmbedderPolicy: "unsafe-none"}
```

### CrossOriginOpenerPolicy
Type: `string`

Default: `"same-origin"`

Required: `false`

Sets the `Cross-Origin-Opener-Policy` header.

```go
helmet.Helmet{CrossOriginOpenerPolicy: "same-origin-allow-popups"}
```

### CrossOriginResourcePolicy
Type: `string`

Default: `"same-origin"`

Required: `false`

Sets the `Cross-Origin-Resource-Policy` header.

```go
helmet.Helmet{CrossOriginResourcePolicy: "cross-origin"}
```

### DNSPrefetchControl
Type: `string`

Default: `"off"`

Required: `false`

Sets the `X-DNS-Prefetch-Control` header.

```go
helmet.Helmet{DNSPrefetchControl: "on"}
```

### FrameOptions
Type: `string`

Default: `"SAMEORIGIN"`

Required: `false`

Sets the `X-Frame-Options` header.

```go
helmet.Helmet{FrameOptions: "DENY"}
```

### HSTSMaxAge, HSTSExcludeSubDomains, HSTSPreload, DisableHSTS
Type: `int`, `bool`, `bool`, `bool` respectively

Default: `HSTSMaxAge: 0` (→ `15552000`, 180 days), `HSTSExcludeSubDomains: false` (subdomains included), `HSTSPreload: false`, `DisableHSTS: false`

Required: `false`

Together these four fields build the `Strict-Transport-Security` header as `max-age=<HSTSMaxAge>[; includeSubDomains][; preload]`. A zero `HSTSMaxAge` falls back to `15552000`. Setting `DisableHSTS` to `true` omits the header entirely, overriding the other three fields.

```go
helmet.Helmet{
	HSTSMaxAge:            86400,
	HSTSExcludeSubDomains: true,
	HSTSPreload:           true,
} // -> Strict-Transport-Security: max-age=86400; preload

helmet.Helmet{DisableHSTS: true} // -> header omitted
```

### PermittedCrossDomainPolicies
Type: `string`

Default: `"none"`

Required: `false`

Sets the `X-Permitted-Cross-Domain-Policies` header.

```go
helmet.Helmet{PermittedCrossDomainPolicies: "master-only"}
```

### ReferrerPolicy
Type: `string`

Default: `"no-referrer"`

Required: `false`

Sets the `Referrer-Policy` header.

```go
helmet.Helmet{ReferrerPolicy: "strict-origin"}
```

## `Helmet` Methods

### NewMiddleware

Compiles the `Helmet` struct's fields into an internal options value once, and returns a `common.MiddlewareFn` that reuses those compiled options on every request.

#### Parameters
None.

#### Returns
- 1st value: `common.MiddlewareFn`

- Description: A middleware ready to bind via `BindGlobalMiddlewares` or `BindMiddleware`, with its headers compiled once.

#### Usage

```go
mw := helmet.Helmet{FrameOptions: "DENY"}.NewMiddleware()
app.BindGlobalMiddlewares(mw)
```

### Use

Sets every Helmet response header on the current request and always calls `next` — Helmet never short-circuits a request.

#### Rules
- `next` is always called (`TestHelmet_Use_CallsNext`).
- The four non-configurable headers are always set to their fixed values: `X-Content-Type-Options: nosniff`, `X-Download-Options: noopen`, `X-XSS-Protection: 0`, `Origin-Agent-Cluster: ?1` (`TestHelmet_Use_SetsXContentTypeOptions`, `TestHelmet_Use_SetsXDownloadOptions`, `TestHelmet_Use_SetsXXSSProtectionToZero`, `TestHelmet_Use_SetsOriginAgentCluster`).
- With no fields set, `Content-Security-Policy` is the package default string; setting `ContentSecurityPolicy` overrides it verbatim (`TestHelmet_Use_SetsDefaultCSP`, `TestHelmet_Use_SetsCustomCSP`).
- With no fields set, `Strict-Transport-Security` is `"max-age=15552000; includeSubDomains"`; setting `DisableHSTS: true` omits the header entirely (`TestHelmet_Use_SetsDefaultHSTS`, `TestHelmet_Use_SkipsHSTSWhenDisabled`).
- With no fields set, `X-Frame-Options` is `"SAMEORIGIN"`; `FrameOptions` overrides it (`TestHelmet_Use_SetsDefaultFrameOptions`, `TestHelmet_Use_SetsCustomFrameOptions`).
- With no fields set, `Referrer-Policy` is `"no-referrer"`; `ReferrerPolicy` overrides it (`TestHelmet_Use_SetsDefaultReferrerPolicy`, `TestHelmet_Use_SetsCustomReferrerPolicy`).
- With no fields set, `Cross-Origin-Embedder-Policy`, `Cross-Origin-Opener-Policy`, and `Cross-Origin-Resource-Policy` default to `"require-corp"`, `"same-origin"`, and `"same-origin"` respectively (`TestHelmet_Use_SetsCrossOriginHeaders`).
- With no fields set, `X-DNS-Prefetch-Control` is `"off"` (`TestHelmet_Use_SetsDefaultDNSPrefetchControl`).
- With no fields set, `X-Permitted-Cross-Domain-Policies` is `"none"` (`TestHelmet_Use_SetsDefaultPermittedCrossDomainPolicies`).

#### Parameters
- 1st parameter: `*http.Request` (`r`)

- Description: The current request; unused by `Helmet`, but part of the `common.MiddlewareFn` signature.

- 2nd parameter: `http.ResponseWriter` (`w`)

- Description: Its response headers are mutated in place.

- 3rd parameter: `ctx.Next` (`next`)

- Description: Called to pass control to the next handler in the chain.

#### Returns
None.

#### Usage

```go
helmet.Helmet{}.Use(r, w, next)
```

## Benchmarks

Benchmarks were run on an Intel Core i7-9750H CPU @ 2.60 GHz with `go test -bench=. -benchmem`.

Results are machine-dependent and were captured at documentation generation time (July 30, 2026).

| Benchmark | Operations | Time per op | Allocs | Bytes |
|---|---|---|---|---|
| `BenchmarkHelmet_Use_Defaults` (default security headers) | 589,419 | 2,091 ns/op | 15 | 248 B/op |
| `BenchmarkHelmet_Use_CustomCSP` (custom CSP) | 653,035 | 1,814 ns/op | 15 | 248 B/op |
| `BenchmarkHelmet_Use_DisableHSTS` (HSTS disabled) | 755,235 | 1,647 ns/op | 14 | 232 B/op |
| `BenchmarkLoadHelmetOptions` (compile Helmet config) | 4,306,753 | 320.1 ns/op | 5 | 245 B/op |

**Key observations:**

- Setting security headers costs ~1.6-2.1µs per request
- Disabling HSTS is slightly faster (~1.6µs) as it sets fewer headers
- LoadHelmetOptions is a one-time startup cost (~320ns), negligible compared to per-request overhead
- All allocations are small and predictable (~250 bytes, 14-15 allocations)
