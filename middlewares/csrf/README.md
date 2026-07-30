# CSRF Middleware

*`csrf` implements the double-submit-cookie pattern as a Ginject middleware, issuing a token cookie and verifying it against a header or form field on state-changing requests.*

- [CSRF Middleware](#csrf-middleware)
  - [Key Features](#key-features)
  - [Usage](#usage)
  - [How It Works](#how-it-works)
  - [`CSRF` Struct](#csrf-struct)
    - [TokenLength](#tokenlength)
    - [CookieName](#cookiename)
    - [HeaderName](#headername)
    - [ContextKey](#contextkey)
  - [Functions](#functions)
    - [GenerateCSRFToken](#generatecsrftoken)
    - [CompareTokensSecurely](#comparetokenssecurely)
  - [`CSRF` Methods](#csrf-methods)
    - [NewMiddleware](#newmiddleware)
    - [Use](#use)
  - [Benchmarks](#benchmarks)

## Key Features
- Double-submit-cookie pattern: no server-side session storage required
- `GET`/`HEAD`/`OPTIONS` requests always pass through untouched; only state-changing methods are verified
- Accepts the submitted token from a custom header, the `X-XSRF-TOKEN` header, or a `_csrf` form field, in that priority order
- Constant-time token comparison to prevent timing attacks
- The active token is exposed to downstream handlers via the request context

## Usage

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

`CSRF` also works scoped to a single controller, since it satisfies `common.MiddlewareFn` directly:

```go
func (c FormController) NewController() core.Controller {
	c.BindMiddleware(csrf.CSRF{})
	return c
}
```

## How It Works

1. On every request, `Use` reads the token from the configured cookie. If the cookie is missing or empty, it generates a new token with `GenerateCSRFToken` and sets it on a non-`HttpOnly` cookie so client-side JavaScript can read it back for the next request.
2. The resolved token is stored in the request context under `ContextKey`, reachable from handlers via `c.Request.Context().Value(...)`.
3. For safe methods (`GET`, `HEAD`, `OPTIONS`), `next` is called immediately — no verification happens.
4. For any other method, the submitted token is read from `HeaderName`, then `X-XSRF-TOKEN`, then the `_csrf` form field, and compared against the cookie token with `CompareTokensSecurely`. A mismatch panics with `exception.ForbiddenException`.

## `CSRF` Struct

### TokenLength
Type: `int`

Default: `32` (bytes of entropy, encoded as 64 hex characters)

Required: `false`

A zero or negative value falls back to the default.

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

The header checked first for the submitted token on state-changing requests. `X-XSRF-TOKEN` is always checked as a fallback regardless of this setting.

```go
csrf.CSRF{HeaderName: "X-My-CSRF"}
```

### ContextKey
Type: `string`

Default: `"csrf_token"`

Required: `false`

The request-context key under which the active token is stored for downstream handlers to read.

```go
csrf.CSRF{ContextKey: "my_key"}
```

## Functions

### GenerateCSRFToken

Returns a hex-encoded, cryptographically random token.

#### Rules
- A `length` of 32 produces a 64-character hex string, i.e. the output is always `2 × length` hex characters (`TestGenerateCSRFToken_Length`).
- A `length` of `0` (or any non-positive value) falls back to the 32-byte default, producing a 64-character token (`TestGenerateCSRFToken_ZeroLengthUsesDefault`).
- Successive calls produce different tokens (`TestGenerateCSRFToken_Uniqueness`).
- The output contains only lowercase hex characters (`0`-`9`, `a`-`f`) (`TestGenerateCSRFToken_OnlyHexChars`).

#### Parameters
- 1st parameter: `int` (`length`)

- Description: Number of random bytes to generate; non-positive values fall back to the package default of 32.

#### Returns
- 1st value: `string`

- Description: The hex-encoded random token, `2 × length` characters long.

- 2nd value: `error`

- Description: Non-nil if the underlying random source fails to produce enough bytes.

#### Usage

```go
token, err := csrf.GenerateCSRFToken(32)
if err != nil {
	panic(err)
}
fmt.Println(token)
```

### CompareTokensSecurely

Compares two tokens in constant time to prevent timing attacks.

#### Rules
- Two equal, non-empty strings compare equal (`TestCompareTokensSecurely_Equal`).
- Two unequal strings compare unequal (`TestCompareTokensSecurely_Unequal`).
- Two empty strings compare equal (`TestCompareTokensSecurely_EmptyBothEqual`).
- A non-empty string compared against an empty string compares unequal (`TestCompareTokensSecurely_OneEmpty`).

#### Parameters
- 1st parameter: `string` (`a`)

- Description: The first token to compare.

- 2nd parameter: `string` (`b`)

- Description: The second token to compare.

#### Returns
- 1st value: `bool`

- Description: `true` if `a` and `b` are equal.

#### Usage

```go
if !csrf.CompareTokensSecurely(submitted, expected) {
	panic("token mismatch")
}
```

## `CSRF` Methods

### NewMiddleware

Compiles the `CSRF` struct's fields into an internal options value once, and returns a `common.MiddlewareFn` that reuses those compiled options on every request.

#### Rules
- Returns a value of the package's internal compiled middleware type, distinct from the `CSRF` struct itself (`TestCSRF_NewMiddleware_ReturnsCompiledCSRF`).

#### Parameters
None.

#### Returns
- 1st value: `common.MiddlewareFn`

- Description: A middleware ready to bind via `BindGlobalMiddlewares` or `BindMiddleware`, with its CSRF options compiled once.

#### Usage

```go
mw := csrf.CSRF{TokenLength: 16}.NewMiddleware()
app.BindGlobalMiddlewares(mw)
```

### Use

Issues or reuses the CSRF cookie, exposes the token via the request context, and verifies the submitted token on state-changing requests.

#### Rules
- `GET`, `HEAD`, and `OPTIONS` requests always call `next` without verifying a token (`TestCSRF_SafeMethod_GET`, `TestCSRF_SafeMethod_HEAD`, `TestCSRF_SafeMethod_OPTIONS`).
- If the configured cookie is missing, a new token is generated and set on a cookie of that name (`TestCSRF_SetsCookieWhenMissing`).
- If the configured cookie already has a value, no new cookie is set — the existing token is reused (`TestCSRF_ReusesExistingCookie`).
- The resolved token is stored in the request context under `ContextKey`, readable via `r.Context().Value(ContextKey)` (`TestCSRF_StoresTokenInContext`).
- For `POST`, `PUT`, `PATCH`, and `DELETE` requests, a token submitted via `HeaderName` that matches the cookie token allows the request through (`TestCSRF_POST_ValidHeader`, `TestCSRF_PUT_ValidHeader`, `TestCSRF_PATCH_ValidHeader`, `TestCSRF_DELETE_ValidHeader`).
- A token submitted via the `X-XSRF-TOKEN` header is also accepted, even when `HeaderName` is unset to its default (`TestCSRF_POST_ValidAltHeader`).
- A token submitted via the `_csrf` form field is accepted when no header token is present (`TestCSRF_POST_ValidFormField`).
- A missing, empty, mismatched, or otherwise invalid submitted token panics with a CSRF exception instead of calling `next` (`TestCSRF_POST_MissingToken_Panics`, `TestCSRF_POST_EmptyToken_Panics`, `TestCSRF_POST_WrongToken_Panics`, `TestCSRF_POST_SpecialCharsToken_Panics`).
- Safe to call concurrently from multiple goroutines, for both safe and state-changing requests (`TestCSRF_ConcurrentSafeRequests`, `TestCSRF_ConcurrentStateChanging`).

#### Parameters
- 1st parameter: `*http.Request` (`r`)

- Description: The current request; its cookies and method are read, and its context is replaced in place (via `*r = *r.WithContext(...)`, so the caller's `*ctx.HTTPContext` sees the update too) to carry the resolved token downstream.

- 2nd parameter: `http.ResponseWriter` (`w`)

- Description: The CSRF cookie is set on this when a new token is generated.

- 3rd parameter: `ctx.Next` (`next`)

- Description: Called to pass control to the next handler in the chain.

#### Returns
None.

#### Usage

```go
csrf.CSRF{}.Use(r, w, next)
```

## Benchmarks

Benchmarks were run on an Intel Core i7-9750H CPU @ 2.60 GHz with `go test -bench=. -benchmem`.

Results are machine-dependent and were captured at documentation generation time (July 30, 2026).

| Benchmark | Operations | Time per op | Allocs | Bytes |
|---|---|---|---|---|
| `BenchmarkGenerateCSRFToken` (token generation) | 2,083,831 | 603.7 ns/op | 2 | 128 B/op |
| `BenchmarkCompareTokensSecurely_Equal` (constant-time equal) | 52,997,137 | 23.67 ns/op | 0 | 0 B/op |
| `BenchmarkCompareTokensSecurely_Unequal` (constant-time unequal) | 47,164,563 | 27.35 ns/op | 0 | 0 B/op |
| `BenchmarkCSRF_SafeMethod_NoCookie` (GET, no existing cookie) | 324,175 | 5,239 ns/op | 20 | 6,187 B/op |
| `BenchmarkCSRF_SafeMethod_WithCookie` (GET, with cookie) | 325,197 | 4,002 ns/op | 22 | 6,135 B/op |
| `BenchmarkCSRF_POST_ValidHeader` (POST, valid token) | 330,140 | 3,573 ns/op | 25 | 6,167 B/op |
| `BenchmarkLoadCSRFOptions` (compile CSRF config) | 100,000,000 | 10.84 ns/op | 0 | 0 B/op |

**Key observations:**

- Token generation (~600ns) is the slowest operation
- Constant-time token comparison is extremely fast (~20-25ns)
- Safe methods (GET) cost ~4-5µs (mostly request context overhead)
- State-changing methods (POST) cost ~3.6µs (validation is fast)
- LoadCSRFOptions is negligible (~10ns), a true one-time startup cost
