# HTTP Client Module

*Built-in HTTP client for the `Ginject` framework. Axios-inspired fluent API, Ginject-native dependency injection, and a full middleware/hook/retry pipeline — all on top of Go's stdlib `net/http`.*

- [HTTP Client Module](#http-client-module)
  - [Key Features](#key-features)
  - [Architecture Overview](#architecture-overview)
    - [Transport Layer](#transport-layer)
    - [Middleware Chain](#middleware-chain)
    - [Lifecycle Hooks](#lifecycle-hooks)
    - [ClientService](#clientservice)
  - [Usage](#usage)
    - [Register the module](#register-the-module)
    - [Inject ClientService](#inject-clientservice)
    - [Make requests](#make-requests)
    - [JSON body and response](#json-body-and-response)
    - [Form and multipart upload](#form-and-multipart-upload)
    - [Query parameters](#query-parameters)
    - [Custom headers](#custom-headers)
    - [Streaming response](#streaming-response)
    - [Server-Sent Events (SSE)](#server-sent-events-sse)
    - [Download a file](#download-a-file)
    - [Retry with backoff](#retry-with-backoff)
    - [Middleware](#middleware)
    - [Lifecycle hooks](#lifecycle-hooks-1)
    - [Debug mode](#debug-mode)
    - [Cookie jar](#cookie-jar)
  - [`HttpClientModuleOptions` Parameters](#httpclientmoduleoptions-parameters)
  - [`Client` Methods](#client-methods)
    - [HTTP methods](#http-methods)
    - [Configuration](#configuration)
    - [Security](#security)
    - [Hooks](#hooks)
  - [`RequestBuilder` Methods](#requestbuilder-methods)
  - [`Response` Fields and Methods](#response-fields-and-methods)
  - [Timing Information](#timing-information)
  - [Error Handling](#error-handling)
  - [Security Features](#security-features)
    - [SSRF Protection](#ssrf-protection)
    - [HTTPS Enforcement](#https-enforcement)
    - [TLS Configuration](#tls-configuration)
    - [Response Size Limit](#response-size-limit)
    - [Debug Header Masking](#debug-header-masking)
  - [Custom Transport Backend](#custom-transport-backend)

---

## Key Features

- Fluent `RequestBuilder` chaining — Axios-like ergonomics in Go
- First-class DI integration via `ClientService` — no boilerplate wiring
- Middleware chain: `Middleware func(Handler) Handler` — compose auth, logging, tracing
- Lifecycle hooks: `OnBeforeRequest`, `OnAfterResponse`, `OnError`
- Retry with configurable exponential backoff
- Streaming responses — `Stream()` and `SSE()` modes
- Server-Sent Event parser built in
- Download to file with optional progress callback
- SSRF protection via host allowlist/denylist hook
- HTTPS enforcement at the transport layer
- Per-request and client-level timeouts
- Timing info: DNS, TCP, TLS, TTFB, Total per request
- Debug mode with automatic sensitive-header masking
- Cookie jar (opt-in)
- Zero external dependencies

---

## Architecture Overview

### Transport Layer

Every request travels through a composable `http.RoundTripper` stack:

```
secureRoundTripper (SSRF / HTTPS check)
  └── *http.Transport (stdlib, cloned from DefaultTransport)
```

`SetTLSConfig`, `RequireHTTPS`, and `SetValidateHost` all configure this layer. They are applied atomically and take effect for the next request.

### Middleware Chain

Each `Send()` call builds a fresh chain from the client's registered middlewares and executes it:

```
middleware[0]
  └── middleware[1]
        └── ... middleware[n]
              └── finalHandler (http.Client.Do + body read)
```

Middlewares are applied in registration order (first registered = outermost). The `finalHandler` reads the response body (or leaves it open in stream mode) and builds the `*Response`.

### Lifecycle Hooks

Hooks run outside the middleware chain:

```
OnBeforeRequest  →  middleware chain  →  OnAfterResponse
                          ↓ (on any error)
                        OnError
```

`OnBeforeRequest` can mutate the `*http.Request` (inject auth tokens, add tracing headers, etc.). `OnAfterResponse` can inspect or transform the `*Response`. `OnError` is informational — it fires on transport errors, hook errors, and status validation failures.

### ClientService

`ClientService` is the injectable `core.Provider`. It wraps any `Client` implementation and exposes request methods directly, so embedded controllers and providers can call `c.Get(...)` without indirection.

---

## Usage

### Register the module

```go
import (
    "github.com/dangduoc08/ginject/core"
    "github.com/dangduoc08/ginject/modules/httpclient"
)

func main() {
    app := core.New()
    app.Create(
        core.ModuleBuilder().
            Imports(
                httpclient.Register(&httpclient.HttpClientModuleOptions{
                    IsGlobal: true,
                    BaseURL:  "https://api.example.com",
                }),
            ).
            Controllers(UserController{}).
            Build(),
    )
    app.Listen(8080)
}
```

### Inject ClientService

Embed `httpclient.ClientService` in any controller or provider:

```go
type UserController struct {
    common.REST
    httpclient.ClientService
}
```

The framework resolves and injects `ClientService` automatically.

### Make requests

All seven HTTP methods are available. The path is appended to the client's `BaseURL` when it doesn't start with `http://` or `https://`:

```go
// GET https://api.example.com/users
resp, err := c.Get("/users").Send()

// POST https://api.example.com/users
resp, err := c.Post("/users").JSON(payload).Send()

// DELETE with full URL (ignores BaseURL)
resp, err := c.Delete("https://other.example.com/items/42").Send()
```

### JSON body and response

```go
type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// Send JSON body
resp, err := c.Post("/users").
    JSON(CreateUserRequest{Name: "Alice", Email: "alice@example.com"}).
    Send()
if err != nil {
    panic(err)
}

// Unmarshal JSON response
var user User
if err := resp.JSON(&user); err != nil {
    panic(err)
}
fmt.Println(user.ID)
```

`Content-Type: application/json` is set automatically.

### Form and multipart upload

**URL-encoded form:**

```go
resp, err := c.Post("/login").
    Form(map[string]string{
        "username": "alice",
        "password": "secret",
    }).
    Send()
```

**Multipart file upload:**

```go
f, _ := os.Open("avatar.jpg")
defer f.Close()

resp, err := c.Post("/profile/avatar").
    File("avatar", "avatar.jpg", f).
    Field("user_id", "42").
    Send()
```

When `File()` is called, the request automatically becomes `multipart/form-data`. `Field()` adds text fields to the same multipart body.

### Query parameters

```go
resp, err := c.Get("/users").
    Query("page", 2).
    Query("limit", 20).
    Query("tags", []string{"admin", "active"}).
    Send()
// → GET /users?page=2&limit=20&tags=admin&tags=active
```

`[]string` values are added with `Add` (repeated key). Any other type is converted with `fmt.Sprint`.

### Custom headers

Per-request headers override client defaults:

```go
resp, err := c.Get("/protected").
    Header("Authorization", "Bearer "+token).
    Header("X-Request-ID", requestID).
    Send()
```

Set headers for all requests at the client level:

```go
c.SetHeader("X-App-Version", "2.1.0")
c.SetHeaders(map[string]string{
    "Accept":       "application/json",
    "X-Api-Key":    apiKey,
})
```

### Streaming response

Call `Stream()` to prevent the body from being buffered. The caller must close `Response.BodyStream` when done:

```go
resp, err := c.Get("/large-file").Stream().Send()
if err != nil {
    panic(err)
}
defer resp.BodyStream.Close()

_, _ = io.Copy(os.Stdout, resp.BodyStream)
```

### Server-Sent Events (SSE)

`SSE()` sets `Accept: text/event-stream` and keeps the body open. Use `NewSSEReader` to parse events:

```go
resp, err := c.Get("/events").SSE().Send()
if err != nil {
    panic(err)
}
defer resp.BodyStream.Close()

sr := httpclient.NewSSEReader(resp.BodyStream)
for {
    evt, ok := sr.Next()
    if !ok {
        break
    }
    fmt.Printf("event=%s data=%s\n", evt.Event, evt.Data)
}
```

`SSEEvent` fields:

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `string` | `id:` field |
| `Event` | `string` | `event:` field |
| `Data` | `string` | `data:` lines joined by `\n` |
| `Retry` | `int` | `retry:` reconnection hint (milliseconds) |

### Download a file

```go
// Simple download
err := c.Download("https://files.example.com/report.pdf", "/tmp/report.pdf")

// With progress callback
err := c.DownloadWithProgress(
    "https://files.example.com/data.zip",
    "/tmp/data.zip",
    func(p httpclient.Progress) {
        fmt.Printf("%.1f%%\n", p.Percent)
    },
)
```

`Progress` fields:

| Field | Type | Description |
|-------|------|-------------|
| `Total` | `int64` | Content-Length from response headers; 0 if unknown |
| `Current` | `int64` | Bytes received so far |
| `Percent` | `float64` | `Current / Total * 100`; 0 when Total is unknown |

### Retry with backoff

**Client-level** (applies to all requests):

```go
c.SetRetry(3)
c.SetRetryBackoff(100*time.Millisecond, 2*time.Second)
```

**Per-request** (overrides client defaults):

```go
resp, err := c.Post("/payments").
    JSON(payload).
    Retry(5).
    RetryBackoff(50*time.Millisecond, 1*time.Second).
    Send()
```

The retry condition retries when:
- A transport error occurred (network failure, timeout), or
- The response status code is ≥ 500

Backoff doubles on each attempt and is capped at `maxWait`. The total time is still bounded by the request timeout (if set).

### Middleware

```go
// Auth middleware
authMW := func(next httpclient.Handler) httpclient.Handler {
    return func(req *http.Request) (*httpclient.Response, error) {
        req.Header.Set("Authorization", "Bearer "+getToken())
        return next(req)
    }
}

// Logging middleware
logMW := func(next httpclient.Handler) httpclient.Handler {
    return func(req *http.Request) (*httpclient.Response, error) {
        start := time.Now()
        resp, err := next(req)
        log.Printf("%s %s %v", req.Method, req.URL.Path, time.Since(start))
        return resp, err
    }
}

c.Use(authMW, logMW)
```

Middlewares run in registration order. `authMW` wraps outermost, so it runs first on the way in and last on the way out.

### Lifecycle hooks

```go
// Inject a trace ID into every request
c.OnBeforeRequest(func(req *http.Request) error {
    req.Header.Set("X-Trace-ID", newTraceID())
    return nil
})

// Log every successful response
c.OnAfterResponse(func(resp *httpclient.Response) error {
    log.Printf("response: %d", resp.StatusCode)
    return nil
})

// Collect errors centrally
c.OnError(func(err error) {
    metrics.Increment("http.client.error")
})
```

Returning a non-nil error from `OnBeforeRequest` or `OnAfterResponse` aborts the request and is returned to the caller as a wrapped `*Error`.

### Debug mode

```go
c.EnableDebug()
```

Every request prints to stdout:

```
[httpclient] --> POST https://api.example.com/users
  >  Content-Type: application/json
  >  Authorization: ***
[httpclient] <-- 201 Created
  <  Content-Type: application/json
  timing: total=43ms ttfb=42ms dns=1ms tcp=12ms tls=18ms
```

Sensitive headers (`Authorization`, `Cookie`, `Set-Cookie`, `X-Api-Key`) are always replaced with `***`.

### Cookie jar

Disabled by default. Enable once per client:

```go
c.EnableCookies()
```

Cookies are persisted in-memory across requests using `net/http/cookiejar.New(nil)`.

---

## `HttpClientModuleOptions` Parameters

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `IsGlobal` | `bool` | `false` | Make `ClientService` available in every module |
| `BaseURL` | `string` | `""` | Prepended to every relative request path |
| `Headers` | `map[string]string` | `nil` | Default headers sent on every request |
| `Timeout` | `time.Duration` | `0` (no timeout) | Client-level timeout for all requests |
| `OnInit` | `func()` | `nil` | Called before the module is wired into the DI graph |

---

## `Client` Methods

### HTTP methods

Each method returns a `RequestBuilder` for the given path. If the path starts with `http://` or `https://`, it is used as-is and `BaseURL` is ignored.

| Method | HTTP verb |
|--------|-----------|
| `Get(path string) RequestBuilder` | GET |
| `Post(path string) RequestBuilder` | POST |
| `Put(path string) RequestBuilder` | PUT |
| `Patch(path string) RequestBuilder` | PATCH |
| `Delete(path string) RequestBuilder` | DELETE |
| `Head(path string) RequestBuilder` | HEAD |
| `Options(path string) RequestBuilder` | OPTIONS |

### Configuration

| Method | Description |
|--------|-------------|
| `Use(middlewares ...Middleware)` | Append global middlewares |
| `SetBaseURL(u string)` | Set or replace the base URL |
| `SetHeader(key, value string)` | Set a default header |
| `SetHeaders(headers map[string]string)` | Set multiple default headers |
| `SetTimeout(d time.Duration)` | Client-level timeout |
| `SetRetry(count int)` | Default retry count (0 = no retry) |
| `SetRetryBackoff(initial, max time.Duration)` | Backoff start and cap |
| `SetValidateStatus(fn func(int) bool)` | Status validator; default accepts 200–399 |
| `SetMaxResponseSize(n int64)` | Cap buffered response body; default 32 MB |
| `EnableDebug()` | Print request/response to stdout |
| `EnableCookies()` | Enable in-memory cookie jar |

### Security

| Method | Description |
|--------|-------------|
| `RequireHTTPS(v bool)` | Reject requests with `http://` scheme |
| `SetTLSConfig(cfg *tls.Config)` | Custom TLS settings (CA, client cert, etc.) |
| `SetValidateHost(fn func(string) bool)` | SSRF protection — return `false` to block a host |

### Hooks

| Method | Description |
|--------|-------------|
| `OnBeforeRequest(fn func(*http.Request) error)` | Called after headers are built, before the chain |
| `OnAfterResponse(fn func(*Response) error)` | Called after a successful response |
| `OnError(fn func(error))` | Called on any error (transport, hook, or status) |
| `Download(rawURL, filepath string) error` | Stream-download to a file |
| `DownloadWithProgress(rawURL, filepath string, fn func(Progress)) error` | Download with progress callback |

---

## `RequestBuilder` Methods

All methods return `RequestBuilder` for chaining. `Send()` terminates the chain.

| Method | Description |
|--------|-------------|
| `Context(ctx context.Context)` | Override the request context |
| `Header(key, value string)` | Set a per-request header (overrides client default) |
| `Headers(headers map[string]string)` | Set multiple per-request headers |
| `Query(key string, value any)` | Append a query parameter; `[]string` → repeated key |
| `JSON(v any)` | Set JSON body; sets `Content-Type: application/json` |
| `Form(v any)` | Set URL-encoded body; accepts `map[string]string`, `url.Values`, `map[string]any` |
| `Body(r io.Reader)` | Set a raw body reader |
| `File(field, filename string, r io.Reader)` | Add a file part; switches body to `multipart/form-data` |
| `Field(key, value string)` | Add a text part to a multipart body |
| `Timeout(d time.Duration)` | Per-request timeout (overrides client-level) |
| `Retry(count int)` | Per-request retry count |
| `RetryBackoff(initial, max time.Duration)` | Per-request backoff |
| `Stream()` | Keep response body open; caller must close `Response.BodyStream` |
| `SSE()` | Like `Stream()` with `Accept: text/event-stream` |
| `OnProgress(fn func(Progress))` | Progress callback for streaming or download |
| `Send() (*Response, error)` | Execute the request |

---

## `Response` Fields and Methods

| Field / Method | Type | Description |
|----------------|------|-------------|
| `StatusCode` | `int` | HTTP status code |
| `Headers` | `http.Header` | Response headers |
| `Body` | `[]byte` | Buffered body (nil in stream mode) |
| `BodyStream` | `io.ReadCloser` | Raw body stream (non-nil in stream/SSE mode) |
| `Raw` | `*http.Response` | Underlying stdlib response |
| `Timing` | `*TimingInfo` | Per-phase durations |
| `JSON(v any) error` | — | Unmarshal `Body` into v |
| `Text() string` | — | `Body` as UTF-8 string |
| `Bytes() []byte` | — | `Body` as raw bytes |

---

## Timing Information

Every non-streaming response has `Response.Timing` populated:

```go
resp, err := c.Get("/api/data").Send()
if err != nil {
    panic(err)
}

t := resp.Timing
fmt.Printf("total=%v ttfb=%v dns=%v tcp=%v tls=%v\n",
    t.Total, t.TTFB, t.DNS, t.TCP, t.TLS)
```

| Field | Description |
|-------|-------------|
| `DNS` | Time to resolve the hostname |
| `TCP` | Time to establish the TCP connection |
| `TLS` | Time for the TLS handshake (0 for HTTP) |
| `TTFB` | Time to first byte from the start of the request |
| `Total` | Wall-clock duration of the full `Send()` call |

Timing is collected via `net/http/httptrace` and uses `sync/atomic` so parallel-dial goroutines (Happy Eyeballs) cannot cause data races.

---

## Error Handling

`Send()` returns `*Error` on failure. `*Error` implements `error` and `Unwrap`:

```go
resp, err := c.Get("/resource").Send()
if err != nil {
    var httpErr *httpclient.Error
    if errors.As(err, &httpErr) {
        if httpErr.Response != nil {
            fmt.Println("status:", httpErr.Response.StatusCode)
        }
        fmt.Println("cause:", errors.Unwrap(httpErr))
    }
    return
}
```

`*Error` fields:

| Field | Type | Description |
|-------|------|-------------|
| `Request` | `*http.Request` | The request that failed |
| `Response` | `*Response` | The response (nil if the error is a transport error) |
| `Cause` | `error` | The underlying error |

Status validation runs after the middleware chain. By default, status codes outside 200–399 produce an error with `Response` set so the body is still accessible.

To accept all status codes:

```go
c.SetValidateStatus(func(_ int) bool { return true })
```

---

## Security Features

### SSRF Protection

Reject requests to internal or disallowed hosts before the TCP dial:

```go
import "net"

allowed := map[string]bool{
    "api.example.com":  true,
    "cdn.example.com":  true,
}

c.SetValidateHost(func(host string) bool {
    if _, ok := allowed[host]; ok {
        return true
    }
    // Reject private IP ranges
    if ip := net.ParseIP(host); ip != nil {
        return ip.IsGlobalUnicast() && !ip.IsPrivate()
    }
    return false
})
```

A rejected host returns `*Error` with cause `"httpclient: host not allowed: <host>"`. The TCP dial is never attempted.

### HTTPS Enforcement

```go
c.RequireHTTPS(true)
```

Requests with `http://` scheme are rejected at the transport layer before any network call.

### TLS Configuration

```go
import "crypto/tls"

c.SetTLSConfig(&tls.Config{
    MinVersion: tls.VersionTLS12,
    // custom CA pool, client certs, etc.
})
```

### Response Size Limit

The buffered response body is capped at the configured limit (default **32 MB**). Responses that exceed the limit return an error without allocating the full body:

```go
c.SetMaxResponseSize(5 << 20) // 5 MB
```

This limit does not apply in `Stream()` mode — the caller controls how much data is read.

### Debug Header Masking

When `EnableDebug()` is active, the following headers are automatically replaced with `***` in stdout output:

- `Authorization`
- `Cookie`
- `Set-Cookie`
- `X-Api-Key`

The actual request is not modified. This list is fixed and applies to both request and response headers.

---

## Custom Transport Backend

To inject a different `Client` implementation (e.g. one backed by a mock, a circuit breaker, or a service mesh sidecar), bypass `Register` and wire `ClientService` directly:

```go
type mockClient struct{}

func (m *mockClient) Get(path string) httpclient.RequestBuilder { /* ... */ }
// ... implement full Client interface

svc := httpclient.ClientService{Backend: &mockClient{}}

module := core.ModuleBuilder().Providers(svc).Build()
module.IsGlobal = true
```

No application code changes are required — `ClientService` remains the same injectable type.
