# Middlewares

*Built-in middlewares for the `Ginject` framework. Drop them in as global or per-controller layers with no external dependencies.*

- [Middlewares](#middlewares)
  - [How to Use a Middleware](#how-to-use-a-middleware)
    - [Global (all routes)](#global-all-routes)
    - [Per-controller (all handlers)](#per-controller-all-handlers)
    - [Per-controller (specific handlers)](#per-controller-specific-handlers)
  - [CORS](#cors)
    - [Key Features](#key-features)
    - [Quick Start](#quick-start)
    - [`CORS` Fields](#cors-fields)
      - [AllowOrigin](#alloworigin)
      - [AllowHeaders](#allowheaders)
      - [ExposeHeaders](#exposeheaders)
      - [AllowMethods](#allowmethods)
      - [MaxAge](#maxage)
      - [IsAllowCredentials](#isallowcredentials)
      - [IsPreflightContinue](#ispreflightcontinue)
      - [OptionsSuccessStatus](#optionssuccessstatus)
    - [Recipes](#recipes)
      - [Allow all origins (default)](#allow-all-origins-default)
      - [Allow a list of origins](#allow-a-list-of-origins)
      - [Allow origins matching a pattern](#allow-origins-matching-a-pattern)
      - [Allow credentials with wildcard](#allow-credentials-with-wildcard)
      - [Preflight continue](#preflight-continue)
  - [Helmet](#helmet)
    - [Key Features](#key-features-1)
    - [Quick Start](#quick-start-1)
    - [Headers Set by Helmet](#headers-set-by-helmet)
    - [`Helmet` Fields](#helmet-fields)
      - [ContentSecurityPolicy](#contentsecuritypolicy)
      - [CrossOriginEmbedderPolicy](#crossoriginembedderpolicy)
      - [CrossOriginOpenerPolicy](#crossoriginopenpolicy)
      - [CrossOriginResourcePolicy](#crossoriginresourcepolicy)
      - [DNSPrefetchControl](#dnsprefetchcontrol)
      - [FrameOptions](#frameoptions)
      - [HSTS fields](#hsts-fields)
      - [PermittedCrossDomainPolicies](#permittedcrossdomainpolicies)
      - [ReferrerPolicy](#referrerpolicy)
    - [Recipes](#recipes-1)
      - [Custom CSP](#custom-csp)
      - [Disable HSTS (non-HTTPS dev environments)](#disable-hsts-non-https-dev-environments)
      - [HSTS with preload](#hsts-with-preload)
      - [Allow framing by same origin only](#allow-framing-by-same-origin-only)
  - [CSRF](#csrf-1)
    - [Key Features](#key-features-2)
    - [Quick Start](#quick-start-2)
    - [`CSRF` Fields](#csrf-fields)
    - [How It Works](#how-it-works)
    - [Helper Functions](#helper-functions)
    - [Reading the Token in a Handler](#reading-the-token-in-a-handler)
    - [Recipes](#recipes-2)
  - [RequestLogger](#requestlogger)
    - [Key Features](#key-features-3)
    - [Quick Start](#quick-start-3)
    - [Log Fields](#log-fields)
    - [Custom Logger](#custom-logger)

---

## How to Use a Middleware

All built-in middlewares implement the `common.MiddlewareFn` interface and are registered the same way.

### Global (all routes)

```go
app.BindGlobalMiddlewares(
    middlewares.Helmet{},
    middlewares.CORS{},
    middlewares.RequestLogger{},
)
```

Middlewares run in the order they are listed.

### Per-controller (all handlers)

```go
func (c UserController) NewController() core.Controller {
    c.BindMiddleware(middlewares.CORS{
        AllowOrigin: []string{"https://example.com"},
    })
    return c
}
```

### Per-controller (specific handlers)

```go
func (c UserController) NewController() core.Controller {
    c.BindMiddleware(
        middlewares.CORS{AllowOrigin: "https://admin.example.com"},
        c.CREATE_VERSION_1,
        c.UPDATE_VERSION_1,
    )
    return c
}
```

---

## CORS

Implements the [Fetch Standard CORS protocol](https://fetch.spec.whatwg.org/#http-cors-protocol). Handles both simple requests and preflight (`OPTIONS`) requests.

### Key Features

- Wildcard `*`, list of origins, or `*regexp.Regexp` pattern matching
- Automatic `Vary: Origin` header when the response depends on the request origin
- Credentials + wildcard: echoes the request origin instead of `*` (spec compliant)
- Blocks the `null` origin attack when credentials are enabled
- Preflight short-circuit: returns the configured success status without calling `next`
- Zero allocations on the fast path (options compiled once at startup)

### Quick Start

```go
import "github.com/dangduoc08/ginject/middlewares"

app.BindGlobalMiddlewares(middlewares.CORS{
    AllowOrigin:        []string{"https://app.example.com", "https://admin.example.com"},
    AllowHeaders:       []string{"Content-Type", "Authorization"},
    AllowMethods:       []string{"GET", "POST", "PUT", "DELETE"},
    MaxAge:             86400,
    IsAllowCredentials: true,
})
```

### `CORS` Fields

#### AllowOrigin

**Type:** `any` — `string | []string | *regexp.Regexp`  
**Default:** `"*"` (allow all origins)

Controls which origins are permitted.

| Value type       | Behaviour |
|------------------|-----------|
| `"*"`            | Allow all origins. If `IsAllowCredentials` is true, echoes the request origin instead of `*` |
| `[]string`       | Allow exact origins from the list (trailing `/` stripped automatically) |
| `*regexp.Regexp` | Allow origins matching the regexp |

```go
// Allow all
AllowOrigin: "*"

// Allow specific origins
AllowOrigin: []string{"https://app.example.com", "https://admin.example.com"}

// Allow by pattern
import "regexp"
AllowOrigin: regexp.MustCompile(`https://.*\.example\.com$`)
```

#### AllowHeaders

**Type:** `any` — `string | []string`  
**Default:** echoes `Access-Control-Request-Headers` from the preflight request

```go
AllowHeaders: []string{"Content-Type", "Authorization", "X-Request-ID"}
```

#### ExposeHeaders

**Type:** `any` — `string | []string`  
**Default:** `""` (not set)

Headers the browser is allowed to read from the response.

```go
ExposeHeaders: []string{"X-Request-ID", "X-RateLimit-Remaining"}
```

#### AllowMethods

**Type:** `[]string`  
**Default:** `["GET", "HEAD", "PUT", "PATCH", "POST", "DELETE"]`

```go
AllowMethods: []string{"GET", "POST"}
```

#### MaxAge

**Type:** `int` (milliseconds)  
**Default:** `5000` ms → `5` seconds in `Access-Control-Max-Age`

How long the browser may cache a preflight response. The value is automatically converted from milliseconds to seconds.

```go
MaxAge: 86400_000 // 24 hours → sets header to "86400"
```

#### IsAllowCredentials

**Type:** `bool`  
**Default:** `false`

Sets `Access-Control-Allow-Credentials: true`. When combined with `AllowOrigin: "*"`, the middleware echoes the actual request origin to remain spec-compliant.

```go
IsAllowCredentials: true
```

#### IsPreflightContinue

**Type:** `bool`  
**Default:** `false`

When `true`, the middleware calls `next()` after setting preflight headers, passing control to the next layer. When `false` (default), preflight requests are short-circuited and the response is sent immediately with the configured success status.

```go
IsPreflightContinue: false // short-circuit preflight (default)
IsPreflightContinue: true  // pass preflight to the next handler
```

#### OptionsSuccessStatus

**Type:** `int`  
**Default:** `204`

HTTP status code returned for preflight responses (when `IsPreflightContinue` is `false`). Some legacy browsers require `200`.

```go
OptionsSuccessStatus: 200 // for compatibility with older browsers
```

### Recipes

#### Allow all origins (default)

```go
app.BindGlobalMiddlewares(middlewares.CORS{})
```

#### Allow a list of origins

```go
app.BindGlobalMiddlewares(middlewares.CORS{
    AllowOrigin: []string{
        "https://app.example.com",
        "https://admin.example.com",
    },
})
```

#### Allow origins matching a pattern

```go
import "regexp"

app.BindGlobalMiddlewares(middlewares.CORS{
    AllowOrigin: regexp.MustCompile(`^https://.*\.example\.com$`),
})
```

#### Allow credentials with wildcard

When `IsAllowCredentials: true` and `AllowOrigin: "*"`, the browser requires the response to echo the actual origin rather than `*`. Ginject handles this automatically:

```go
app.BindGlobalMiddlewares(middlewares.CORS{
    IsAllowCredentials: true,
    // AllowOrigin defaults to "*" — middleware echoes request origin automatically
})
```

#### Preflight continue

Forward preflight requests to your own OPTIONS handler:

```go
app.BindGlobalMiddlewares(middlewares.CORS{
    IsPreflightContinue: true,
})
```

---

## Helmet

Sets security-related HTTP response headers. Inspired by the [Express Helmet](https://helmetjs.github.io/) library.

### Key Features

- 13 security headers set in a single middleware
- Secure defaults out of the box — zero configuration required
- Every header individually customisable or disableable
- Options compiled once at startup: zero per-request allocations

### Quick Start

```go
import "github.com/dangduoc08/ginject/middlewares"

// Use all defaults — recommended for production
app.BindGlobalMiddlewares(middlewares.Helmet{})
```

### Headers Set by Helmet

| Header | Default value |
|--------|--------------|
| `Content-Security-Policy` | See [ContentSecurityPolicy](#contentsecuritypolicy) |
| `Cross-Origin-Embedder-Policy` | `require-corp` |
| `Cross-Origin-Opener-Policy` | `same-origin` |
| `Cross-Origin-Resource-Policy` | `same-origin` |
| `Origin-Agent-Cluster` | `?1` |
| `Referrer-Policy` | `no-referrer` |
| `Strict-Transport-Security` | `max-age=15552000; includeSubDomains` |
| `X-Content-Type-Options` | `nosniff` |
| `X-DNS-Prefetch-Control` | `off` |
| `X-Download-Options` | `noopen` |
| `X-Frame-Options` | `SAMEORIGIN` |
| `X-Permitted-Cross-Domain-Policies` | `none` |
| `X-XSS-Protection` | `0` |

### `Helmet` Fields

#### ContentSecurityPolicy

**Type:** `string`  
**Default:**
```
default-src 'self';
base-uri 'self';
font-src 'self' https: data:;
form-action 'self';
frame-ancestors 'self';
img-src 'self' data:;
object-src 'none';
script-src 'self';
script-src-attr 'none';
style-src 'self' https: 'unsafe-inline';
upgrade-insecure-requests
```

Pass a custom value to override the entire policy:

```go
ContentSecurityPolicy: "default-src 'self'; script-src 'self' https://cdn.example.com"
```

#### CrossOriginEmbedderPolicy

**Type:** `string`  
**Default:** `"require-corp"`

Sets `Cross-Origin-Embedder-Policy`. Common values: `"require-corp"`, `"unsafe-none"`.

#### CrossOriginOpenerPolicy

**Type:** `string`  
**Default:** `"same-origin"`

Sets `Cross-Origin-Opener-Policy`. Common values: `"same-origin"`, `"same-origin-allow-popups"`, `"unsafe-none"`.

#### CrossOriginResourcePolicy

**Type:** `string`  
**Default:** `"same-origin"`

Sets `Cross-Origin-Resource-Policy`. Common values: `"same-origin"`, `"same-site"`, `"cross-origin"`.

#### DNSPrefetchControl

**Type:** `string`  
**Default:** `"off"`

Sets `X-DNS-Prefetch-Control`. Use `"on"` to enable browser DNS prefetching for a small performance gain at the cost of privacy.

#### FrameOptions

**Type:** `string`  
**Default:** `"SAMEORIGIN"`

Sets `X-Frame-Options`. Common values: `"SAMEORIGIN"`, `"DENY"`.

#### HSTS fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `HSTSMaxAge` | `int` (seconds) | `15552000` (180 days) | `max-age` directive |
| `HSTSExcludeSubDomains` | `bool` | `false` | Omit `includeSubDomains` directive |
| `HSTSPreload` | `bool` | `false` | Append `preload` directive |
| `DisableHSTS` | `bool` | `false` | Skip `Strict-Transport-Security` entirely |

```go
// Custom max-age, include preload
Helmet{
    HSTSMaxAge:  31536000, // 1 year
    HSTSPreload: true,
}

// Disable for HTTP-only development servers
Helmet{
    DisableHSTS: true,
}
```

#### PermittedCrossDomainPolicies

**Type:** `string`  
**Default:** `"none"`

Sets `X-Permitted-Cross-Domain-Policies`. Common values: `"none"`, `"master-only"`, `"all"`.

#### ReferrerPolicy

**Type:** `string`  
**Default:** `"no-referrer"`

Sets `Referrer-Policy`. Common values: `"no-referrer"`, `"strict-origin"`, `"strict-origin-when-cross-origin"`.

### Recipes

#### Custom CSP

```go
app.BindGlobalMiddlewares(middlewares.Helmet{
    ContentSecurityPolicy: strings.Join([]string{
        "default-src 'self'",
        "script-src 'self' https://cdn.example.com",
        "style-src 'self' 'unsafe-inline'",
        "img-src 'self' data: https:",
        "connect-src 'self' https://api.example.com",
    }, "; "),
})
```

#### Disable HSTS (non-HTTPS dev environments)

```go
app.BindGlobalMiddlewares(middlewares.Helmet{
    DisableHSTS: true,
})
```

#### HSTS with preload

```go
app.BindGlobalMiddlewares(middlewares.Helmet{
    HSTSMaxAge:  31536000,
    HSTSPreload: true,
})
```

#### Allow framing by same origin only

```go
// SAMEORIGIN is already the default, but to be explicit:
app.BindGlobalMiddlewares(middlewares.Helmet{
    FrameOptions: "SAMEORIGIN",
})

// Deny framing entirely
app.BindGlobalMiddlewares(middlewares.Helmet{
    FrameOptions: "DENY",
})
```

---

## CSRF

Protects state-changing endpoints against Cross-Site Request Forgery using the **double-submit cookie** pattern. Tokens are generated with `crypto/rand` and compared in constant time to prevent timing attacks.

### Key Features

- Protects POST, PUT, PATCH, DELETE — passes GET, HEAD, OPTIONS through unchanged
- Cryptographically secure token generation (`crypto/rand` + hex encoding)
- Constant-time token comparison (`crypto/subtle`) — no timing leak
- Token accepted from `X-CSRF-Token`, `X-XSRF-TOKEN` header, or `_csrf` form field
- Generates and sets the cookie automatically on first request
- Exposes the current token in the request context for embedding in HTML forms

### Quick Start

```go
import "github.com/dangduoc08/ginject/middlewares"

app.BindGlobalMiddlewares(middlewares.CSRF{})
```

### `CSRF` Fields

| Field         | Type     | Default          | Description                                          |
|---------------|----------|------------------|------------------------------------------------------|
| `TokenLength` | `int`    | `32`             | Bytes of entropy; token string is `2×length` hex chars |
| `CookieName`  | `string` | `"_csrf"`        | Name of the cookie that stores the token             |
| `HeaderName`  | `string` | `"X-CSRF-Token"` | Primary request header carrying the submitted token  |
| `ContextKey`  | `string` | `"csrf_token"`   | Key used to store the token in the request context   |

### How It Works

1. **Every request**: the middleware reads the `_csrf` cookie. If absent or empty, it generates a new token with `crypto/rand`, sets the cookie (`HttpOnly: false` so JS can read it), and stores the token in the request context.
2. **Safe methods** (GET, HEAD, OPTIONS): calls `next()` immediately — no validation.
3. **State-changing methods** (POST, PUT, PATCH, DELETE): extracts the submitted token in priority order:
   - `X-CSRF-Token` header
   - `X-XSRF-TOKEN` header
   - `_csrf` form field
   
   Compares it against the cookie value using `subtle.ConstantTimeCompare`. Mismatch or missing token → `ForbiddenException` (HTTP 403).

> **Cookie flags**: `SameSite=Lax` and `Secure=true` are strongly recommended in production. Set them by supplying a custom cookie via your own middleware or HTTP framework hooks — CSRF does not enforce them to stay framework-agnostic.

### Helper Functions

```go
// Generate a cryptographically secure hex token.
// length=0 uses the default (32 bytes → 64 hex chars).
token, err := middlewares.GenerateCSRFToken(32)

// Constant-time string comparison — safe against timing attacks.
ok := middlewares.CompareTokensSecurely(submitted, stored)
```

### Reading the Token in a Handler

The current CSRF token is stored in the request context under `ContextKey`. Use it to embed the token in server-rendered HTML forms:

```go
func (ctrl MyController) READ_form(c *ctx.Context) string {
    token, _ := c.Request.Context().Value("csrf_token").(string)
    return `<form method="POST">
        <input type="hidden" name="_csrf" value="` + token + `">
    </form>`
}
```

For SPA/API clients, read the `_csrf` cookie with JavaScript and send it as the `X-CSRF-Token` request header.

### Recipes

**Custom cookie and header name:**

```go
app.BindGlobalMiddlewares(middlewares.CSRF{
    CookieName: "XSRF-TOKEN",   // Angular default
    HeaderName: "X-XSRF-TOKEN", // Angular default
})
```

**Longer token (64 bytes of entropy):**

```go
app.BindGlobalMiddlewares(middlewares.CSRF{
    TokenLength: 64,
})
```

**Per-controller (protect only mutation routes):**

```go
func (ctrl ApiController) NewController() core.Controller {
    ctrl.BindMiddleware(
        middlewares.CSRF{},
        ctrl.CREATE,
        ctrl.UPDATE,
        ctrl.DELETE,
    )
    return ctrl
}
```

---

## RequestLogger

Logs every completed request using the framework's structured logger. Attaches to the `REQUEST_FINISHED` event so the log line always includes the final status code, even when the status is set deep in an interceptor or exception filter.

### Key Features

- Logs after the full response pipeline completes (status code is final)
- Handles both HTTP and WebSocket request types
- Injects the framework logger automatically via DI
- Structured key-value fields compatible with any log handler

### Quick Start

```go
import "github.com/dangduoc08/ginject/middlewares"

app.BindGlobalMiddlewares(middlewares.RequestLogger{})
```

No configuration needed. The middleware picks up the application logger automatically.

### Log Fields

**HTTP requests:**

| Field | Value |
|-------|-------|
| Message | Request URL |
| `Method` | HTTP method (`GET`, `POST`, …) |
| `Status` | HTTP status code |
| `Time` | Response time in milliseconds |
| `Protocol` | HTTP protocol version (`HTTP/1.1`, `HTTP/2.0`) |
| `User-Agent` | Request `User-Agent` header |
| `requestId` | Unique request ID assigned by the framework |

**WebSocket events:**

| Field | Value |
|-------|-------|
| Message | WebSocket event name |
| `Time` | Processing time in milliseconds |
| `Subprotocol` | WebSocket subprotocol |
| `User-Agent` | `User-Agent` header from the upgrade request |

**Example output (PrettyFormat):**

```
INFO  2006-01-02 15:04:05  RequestLogger
  ├── Method       "GET"
  ├── Status       200
  ├── Time         "3 ms"
  ├── Protocol     "HTTP/1.1"
  ├── User-Agent   "Mozilla/5.0 ..."
  └── requestId    "a1b2c3d4"
```

### Custom Logger

`RequestLogger` embeds `common.Logger`, which is resolved from the DI graph. To use a custom log handler, configure it on the application before `Create`:

```go
import (
    "github.com/dangduoc08/ginject/log"
    "github.com/dangduoc08/ginject/middlewares"
)

logger := log.NewLog(&log.LogOptions{
    LogFormat:  log.JSONFormat,
    Level:      log.InfoLevel,
    TimeFormat: time.RFC3339,
})

app.
    UseLogger(logger).
    BindGlobalMiddlewares(middlewares.RequestLogger{})
```

The `RequestLogger` instance receives the configured logger automatically — no manual wiring required.
