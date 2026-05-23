# guards

Package `guards` provides production-ready guard implementations for the ginject framework.

A **guard** controls whether a request is allowed to proceed. It implements the `Guarder` interface:

```go
type Guarder interface {
    CanActivate(*ctx.Context) bool
}
```

If `CanActivate` returns `false`, the framework panics with a `ForbiddenException` (403). Guards may also panic directly with a different exception — `ThrottlerGuard` does this with `TooManyRequestsException` (429).

---

## ThrottlerGuard

`ThrottlerGuard` is a self-contained rate limiter guard. It supports three strategies, writes rate-limit headers on every response, and panics with HTTP 429 when the limit is exceeded.

### Types

#### `Strategy`

```go
type Strategy int

const (
    FixedWindow   Strategy = iota // count resets at fixed window boundaries
    SlidingWindow Strategy = iota // weighted approximation across two windows
    TokenBucket   Strategy = iota // continuous refill, smooth burst control
)
```

#### `ThrottlerOptions`

Configuration struct passed to `NewThrottler`.

| Field      | Type                       | Default                  | Description                                               |
|------------|----------------------------|--------------------------|-----------------------------------------------------------|
| `Limit`    | `int64`                    | `100`                    | Maximum requests allowed per TTL window                  |
| `TTL`      | `time.Duration`            | `time.Minute`            | Window duration (FixedWindow/SlidingWindow) or refill period (TokenBucket) |
| `Strategy` | `Strategy`                 | `FixedWindow`            | Rate limiting algorithm                                   |
| `KeyFunc`  | `func(*ctx.Context) string`| `defaultThrottlerKeyFunc`| Extracts the rate-limit key from the request              |
| `Store`    | `cache.Cache`              | `cache.NewMemoryCache()` | Backend for storing counters/state                        |

#### `ThrottlerGuard`

```go
type ThrottlerGuard struct {
    Limit    int64
    TTL      time.Duration
    Strategy Strategy
    KeyFunc  func(*ctx.Context) string
    Store    cache.Cache
}
```

All fields are exported. The guard can be constructed directly or via `NewThrottler`. Because `Store` is an interface (backed by a pointer), copying the struct by value still shares the same underlying backend.

### Constructor

```go
func NewThrottler(opts ThrottlerOptions) ThrottlerGuard
```

Applies defaults for any zero-value field and returns a ready-to-use `ThrottlerGuard`.

### Methods

#### `CanActivate`

```go
func (g ThrottlerGuard) CanActivate(c *ctx.Context) bool
```

Runs the selected strategy, writes rate-limit headers, then either returns `true` or panics:

- **Allowed**: sets `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` and returns `true`.
- **Blocked**: additionally sets `Retry-After` and panics with `exception.TooManyRequestsException("Too Many Requests")` (HTTP 429).

#### `NewGuard`

```go
func (g ThrottlerGuard) NewGuard() ThrottlerGuard
```

Required by the framework's guard registration mechanism. Returns the guard itself.

### Response Headers

| Header                  | Value                                    |
|-------------------------|------------------------------------------|
| `X-RateLimit-Limit`     | Configured `Limit`                       |
| `X-RateLimit-Remaining` | Requests remaining in the current window |
| `X-RateLimit-Reset`     | Unix timestamp when the window resets    |
| `Retry-After`           | Seconds until next allowed request (only when blocked) |

---

## Strategies

### FixedWindow

Counts requests within a fixed time window. The window is identified by `unix / windowSec`. The counter resets atomically at each boundary.

- **Cache key**: `rl:fw:{key}:{windowID}`
- **State**: 8-byte big-endian int64 counter
- **TTL on cache entry**: remaining seconds until window end (minimum 1s)
- **Behaviour**: allows exactly `Limit` requests per window; the `(Limit+1)`th request is blocked until the next boundary.

**Trade-off**: simple and cheap, but allows a burst of `2×Limit` requests straddling a boundary (end of window N + start of window N+1).

### SlidingWindow

Weighted approximation that smooths the boundary burst of FixedWindow. Uses two cache entries — current window and previous window — and blends them by how far through the current window the request arrived.

```
weighted = round(prevCount × (1 − elapsed/windowSec)) + currCount
```

- **Cache keys**: `rl:sw:c:{key}:{currWindowID}`, `rl:sw:p:{key}:{prevWindowID}` (read-only for prev)
- **State per key**: 8-byte big-endian int64 counter
- **TTL on current entry**: `2 × windowSec` seconds
- **Behaviour**: allows `Limit` weighted requests; burst at boundaries is smoothed proportionally.

**Trade-off**: more accurate than FixedWindow with the same storage cost; not perfectly precise (approximation, not a true sliding log).

### TokenBucket

Tokens accumulate at a constant rate (`Limit / TTL`) up to a maximum of `Limit`. Each request consumes one token; requests when `tokens < 1` are blocked.

- **Cache key**: `rl:tb:{key}`
- **State**: 16 bytes — `[8 bytes float64 tokens][8 bytes int64 last_refill_ns]`
- **TTL on cache entry**: `2 × TTL`
- **Behaviour**: allows sustained `Limit/TTL` req/s with burst up to `Limit`. The bucket refills continuously — a single idle period restores tokens proportionally.

**Trade-off**: best for smooth rate control and burst tolerance; slightly more expensive per call (float64 arithmetic, 16-byte state vs 8).

---

## Key Extraction

`defaultThrottlerKeyFunc` resolves the client IP in this priority order:

1. `X-Real-IP` header (trimmed)
2. First IP in `X-Forwarded-For` header (trimmed)
3. Host part of `RemoteAddr` via `net.SplitHostPort`; falls back to raw `RemoteAddr` if unparseable

Supply a custom `KeyFunc` to key by user ID, API token, or any other request attribute.

---

## Cache Backend

`Store` accepts any value implementing `cache.Cache`:

```go
type Cache interface {
    Get(ctx context.Context, key string) ([]byte, bool)
    Set(ctx context.Context, key string, val []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
}
```

The default backend is the in-process sharded memory cache from `modules/cache`. For multi-instance deployments, supply a Redis-backed or other distributed implementation.

---

## Usage

### Standalone

```go
g := guards.NewThrottler(guards.ThrottlerOptions{
    Limit:    60,
    TTL:      time.Minute,
    Strategy: guards.SlidingWindow,
})

// inside a handler or middleware
if !g.CanActivate(c) {
    // unreachable — CanActivate panics on block
}
```

### With the framework

Attach `ThrottlerGuard` to a controller via `BindGuard`:

```go
type ApiController struct {
    common.REST
    common.Guard
}

func (ctrl *ApiController) NewController() Controller {
    ctrl.BindGuard(
        guards.NewThrottler(guards.ThrottlerOptions{Limit: 100}),
        ctrl.READ,
        ctrl.CREATE,
    )
    return ctrl
}
```

To apply to all routes on a controller, omit the handler arguments:

```go
ctrl.BindGuard(guards.NewThrottler(guards.ThrottlerOptions{}))
```

### Custom key by user ID

```go
guards.NewThrottler(guards.ThrottlerOptions{
    Limit: 1000,
    TTL:   time.Hour,
    KeyFunc: func(c *ctx.Context) string {
        return c.Request.Header.Get("X-User-ID")
    },
})
```

### Shared Redis store (multi-instance)

```go
guards.NewThrottler(guards.ThrottlerOptions{
    Limit:    500,
    TTL:      time.Minute,
    Strategy: guards.TokenBucket,
    Store:    myRedisCache, // implements cache.Cache
})
```

---

## Benchmarks

Measured on Intel Core i7-9750H, Go 1.22, darwin/amd64:

| Benchmark                    | ns/op | B/op | allocs/op |
|------------------------------|-------|------|-----------|
| FixedWindow                  | 566   | 72   | 6         |
| SlidingWindow                | 793   | 120  | 9         |
| TokenBucket                  | 493   | 72   | 5         |
| DefaultKeyFunc (RemoteAddr)  | 131   | 16   | 1         |
| DefaultKeyFunc (X-Forwarded) | 141   | 16   | 1         |
| FixedWindow (parallel)       | 570   | 72   | 6         |
| TokenBucket (parallel)       | 591   | 72   | 5         |
