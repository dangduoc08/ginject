# Throttler Guard

*Rate limiting guard for the Ginject framework with three strategies: fixed-window, sliding-window, and token-bucket algorithms. Integrates with any cache backend and provides standard rate-limit headers.*

---

## Table of Contents

- [Throttler Guard](#throttler-guard)
  - [Key Features](#key-features)
  - [Usage](#usage)
  - [Strategies](#strategies)
  - [`Throttler` Struct](#throttler-struct)
    - [Limit](#limit)
    - [TTL](#ttl)
    - [Strategy](#strategy)
    - [KeyFunc](#keyfunc)
    - [Backend](#backend)
  - [`Throttler` Methods](#throttler-methods)
    - [NewGuard](#newguard)
    - [CanActivate](#canactivate)
  - [Default Key Function](#default-key-function)
  - [Response Headers](#response-headers)
  - [Benchmarks](#benchmarks)

---

## Key Features

- **Three rate-limiting strategies** — fixed-window (efficient), sliding-window (accurate), token-bucket (fair)
- **Backend-agnostic** — works with any cache implementation (in-memory, Redis, etc.)
- **Automatic rate-limit headers** — sets `X-RateLimit-*` and `Retry-After` per RFC 6585
- **Smart key extraction** — default key function detects `X-Real-IP`, `X-Forwarded-For`, or `RemoteAddr`
- **Per-IP isolation** — each client tracked independently with zero crosstalk
- **Panic on limit exceeded** — returns `TooManyRequestsException` with 429 status

---

## Usage

```go
package main

import (
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/guards/throttler"
	"time"
)

type APIController struct {
	common.REST
	common.Guard
}

func (c APIController) NewController() core.Controller {
	// 100 requests per minute per IP
	guard := throttler.Throttler{
		Limit:    100,
		TTL:      time.Minute,
		Strategy: throttler.TokenBucket, // or FixedWindow, SlidingWindow
	}.NewGuard()

	c.BindGuard(guard, c.READ, c.CREATE)
	return c
}

func (c APIController) READ() {
	// This request counts against the rate limit
}

func main() {
	app := core.New()
	app.Create(
		core.ModuleBuilder().
			Controllers(APIController{}).
			Build(),
	)
	app.Listen(8080)
}
```

---

## Strategies

The `Strategy` constant controls the rate-limiting algorithm:

### FixedWindow (default)

Divides time into fixed buckets (e.g., 1-minute windows). Requests are counted within the current window; when the window resets, the counter resets to zero.

- **Pros**: Simple, O(1) per request, low overhead
- **Cons**: Allows burst at window boundaries (e.g., limit of 10 allows up to 20 requests if 10 come at the end of window 1 and 10 at the start of window 2)

```go
throttler.Throttler{Strategy: throttler.FixedWindow}
```

### SlidingWindow

Tracks a moving time window that slides continuously. On each request, older requests outside the window are discounted proportionally.

- **Pros**: Prevents boundary bursts, more accurate than fixed-window
- **Cons**: Slightly higher overhead (tracks two windows)

```go
throttler.Throttler{Strategy: throttler.SlidingWindow}
```

### TokenBucket

Requests consume tokens from a bucket that refills continuously at a fixed rate. Allows brief bursts up to the bucket capacity.

- **Pros**: Fair, handles varying load well, allows brief bursts
- **Cons**: Slightly higher overhead (float-point arithmetic)

```go
throttler.Throttler{Strategy: throttler.TokenBucket}
```

---

## `Throttler` Struct

### Limit

Type: `int64`

Default: `100`

Required: `false`

Maximum number of requests allowed within each TTL window. A zero or negative value defaults to 100.

```go
throttler.Throttler{Limit: 50} // 50 requests per TTL
```

---

### TTL

Type: `time.Duration`

Default: `time.Minute`

Required: `false`

Duration of each rate-limit window. After each TTL, the request count resets. A zero or negative value defaults to 1 minute.

```go
throttler.Throttler{TTL: 30 * time.Second} // 30-second windows
```

---

### Strategy

Type: `Strategy`

Default: `FixedWindow`

Required: `false`

The rate-limiting algorithm to use: `FixedWindow`, `SlidingWindow`, or `TokenBucket`. If omitted, defaults to `FixedWindow`.

```go
throttler.Throttler{Strategy: throttler.TokenBucket}
```

---

### KeyFunc

Type: `func(*ctx.HTTPContext) string`

Default: `defaultThrottlerKeyFunc` (extracts IP from `X-Real-IP`, `X-Forwarded-For`, or `RemoteAddr`)

Required: `false`

Function that extracts a unique identifier from the request context. Called once per request to determine which bucket to charge. Allows per-user, per-API-key, or custom rate limiting.

```go
throttler.Throttler{
	KeyFunc: func(c *ctx.HTTPContext) string {
		// Rate limit by API key instead of IP
		return c.Request.Header.Get("Authorization")
	},
}
```

---

### Backend

Type: `cache.Cache`

Default: `memorycache.NewMemoryCache()`

Required: `false`

The cache backend used to store request counts. Any implementation of the `cache.Cache` interface works. If nil, a new in-memory cache is created.

```go
throttler.Throttler{
	Backend: redisCache, // or any cache.Cache implementation
}
```

---

## `Throttler` Methods

### NewGuard

Applies defaults to the `Throttler` and returns itself, ready to be used as a guard.

**Signature**

```go
func (g Throttler) NewGuard() Throttler
```

**Parameters**

None (receiver is the `Throttler` instance).

**Returns**

- 1st value: `Throttler` — the configured throttler with defaults applied

**Rules**

- A zero or negative `Limit` defaults to 100
- A zero or negative `TTL` defaults to 1 minute
- A nil `KeyFunc` defaults to the built-in IP extraction function
- A nil `Backend` defaults to a new in-memory cache

**Usage**

```go
guard := throttler.Throttler{Limit: 50, TTL: 30 * time.Second}.NewGuard()
c.BindGuard(guard, c.READ)
```

---

### CanActivate

Checks if the request is within the rate limit. Sets rate-limit headers on the response. Panics with `TooManyRequestsException` if the limit is exceeded.

**Signature**

```go
func (g Throttler) CanActivate(c *ctx.HTTPContext) bool
```

**Parameters**

- 1st parameter: `*ctx.HTTPContext` — the HTTP request context

**Returns**

- 1st value: `bool` — always `true` on success; panics if limit exceeded

**Rules**

- Always sets `X-RateLimit-Limit`, `X-RateLimit-Remaining`, and `X-RateLimit-Reset` headers
- If limit exceeded, also sets `Retry-After` header with seconds until reset
- Panics with `TooManyRequestsException` (429 status) if the limit is exceeded
- Different keys (IPs) are tracked independently

**Usage**

```go
func (c UserController) NewController() core.Controller {
	guard := throttler.Throttler{Limit: 100}.NewGuard()
	c.BindGuard(guard, c.CREATE, c.UPDATE)
	return c
}
```

---

## Default Key Function

The built-in `KeyFunc` extracts the client IP address in this priority order:

1. **`X-Real-IP` header** — trusted proxy indicator (highest priority)
2. **`X-Forwarded-For` header** — first IP from the list (if multiple)
3. **`RemoteAddr`** — raw connection address (fallback)

This works correctly behind most reverse proxies (Nginx, Cloudflare, HAProxy, etc.) without configuration.

**Custom Key Function Example**

```go
throttler.Throttler{
	KeyFunc: func(c *ctx.HTTPContext) string {
		// Rate limit by user ID instead of IP
		userID := c.Request.Header.Get("X-User-ID")
		if userID != "" {
			return "user:" + userID
		}
		// Fallback to IP
		return c.Request.Header.Get("X-Forwarded-For")
	},
}
```

---

## Response Headers

The throttler always sets the following headers on every response:

| Header | Value | Purpose |
|---|---|---|
| `X-RateLimit-Limit` | `<limit>` | Maximum requests allowed in this window |
| `X-RateLimit-Remaining` | `<count>` | Requests remaining until limit (capped at 0 if exceeded) |
| `X-RateLimit-Reset` | `<unix-timestamp>` | Unix timestamp when the limit resets |
| `Retry-After` | `<seconds>` | **(only when limit exceeded)** Seconds to wait before retrying |

**Example Headers (successful request)**

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1719700260
```

**Example Headers (limit exceeded)**

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1719700260
Retry-After: 30
```

---

## Benchmarks

Benchmarks were run on an Intel Core i7-9750H CPU @ 2.60 GHz with `go test -bench=. -benchmem`.

Results are machine-dependent and were captured at documentation generation time (July 30, 2026).

| Benchmark | Operations | Time per op | Allocs | Bytes |
|---|---|---|---|---|
| `BenchmarkFixedWindow` (check limit, fixed window) | 2,102,038 | 567.3 ns/op | 5 | 56 B/op |
| `BenchmarkSlidingWindow` (check limit, sliding window) | 1,793,104 | 711.0 ns/op | 7 | 88 B/op |
| `BenchmarkTokenBucket` (check limit, token bucket) | 2,532,129 | 512.3 ns/op | 4 | 56 B/op |
| `BenchmarkDefaultKeyFunc_RemoteAddr` (extract IP) | 8,599,416 | 146.1 ns/op | 1 | 16 B/op |
| `BenchmarkDefaultKeyFunc_XForwardedFor` (extract from header) | 7,851,532 | 159.5 ns/op | 1 | 16 B/op |
| `BenchmarkGuard_Allow` (full guard check + headers) | 375,856 | 3,445 ns/op | 28 | 5,996 B/op |
| `BenchmarkFixedWindow_Parallel` (concurrent fixed window) | 2,133,231 | 586.3 ns/op | 5 | 56 B/op |
| `BenchmarkTokenBucket_Parallel` (concurrent token bucket) | 2,151,934 | 556.6 ns/op | 4 | 56 B/op |

**Key observations:**

- **Algorithm overhead**: token-bucket (~512ns) < fixed-window (~567ns) < sliding-window (~711ns)
- **IP extraction**: very fast (~150ns), dominated by header parsing
- **Full guard overhead**: ~3.4µs for a complete check including header setting
- **Concurrency**: all strategies scale well under parallel load with minimal lock contention
- **Memory**: small allocations (4-7 per request), predictable and stable
