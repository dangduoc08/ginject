# guards

## ThrottlerGuard

Rate limiter guard. Panics with HTTP 429 when the limit is exceeded. Sets rate-limit headers on every response.

### Options

| Field      | Type                        | Default                   | Description                          |
|------------|-----------------------------|---------------------------|--------------------------------------|
| `Limit`    | `int64`                     | `100`                     | Max requests per TTL window          |
| `TTL`      | `time.Duration`             | `time.Minute`             | Window duration / token refill period |
| `Strategy` | `Strategy`                  | `FixedWindow`             | `FixedWindow`, `SlidingWindow`, `TokenBucket` |
| `KeyFunc`  | `func(*ctx.HTTPContext) string` | client IP (see below)     | Extracts the rate-limit key          |
| `Store`    | `cache.Cache`               | in-process memory cache   | Backend for storing counters         |

### Response headers

| Header                  | Description                                    |
|-------------------------|------------------------------------------------|
| `X-RateLimit-Limit`     | Configured limit                               |
| `X-RateLimit-Remaining` | Requests remaining in the current window       |
| `X-RateLimit-Reset`     | Unix timestamp when the window resets          |
| `Retry-After`           | Seconds until next allowed request (429 only)  |

### Default key function

Priority order: `X-Real-IP` → first IP in `X-Forwarded-For` → `RemoteAddr`.

---

### Usage

**Bind to specific routes:**

```go
func (ctrl *ApiController) NewController() Controller {
    ctrl.BindGuard(
        guards.NewThrottler(guards.ThrottlerOptions{
            Limit:    60,
            TTL:      time.Minute,
            Strategy: guards.SlidingWindow,
        }),
        ctrl.READ,
        ctrl.CREATE,
    )
    return ctrl
}
```

**Bind to all routes on a controller:**

```go
ctrl.BindGuard(guards.NewThrottler(guards.ThrottlerOptions{Limit: 100}))
```

**Custom key (e.g. by user ID):**

```go
guards.NewThrottler(guards.ThrottlerOptions{
    Limit: 1000,
    TTL:   time.Hour,
    KeyFunc: func(c *ctx.HTTPContext) string {
        return c.Request.Header.Get("X-User-ID")
    },
})
```

**Custom store (e.g. Redis for multi-instance):**

```go
guards.NewThrottler(guards.ThrottlerOptions{
    Limit:    500,
    TTL:      time.Minute,
    Strategy: guards.TokenBucket,
    Store:    myRedisCache, // implements cache.Cache
})
```

---

## Strategies

### FixedWindow

Counts requests within a fixed time window. Counter resets at each boundary.

- **Cache key**: `rl:fw:{key}:{windowID}`
- **State**: 8-byte big-endian int64 counter
- **Trade-off**: simplest and cheapest, but allows a burst of `2×Limit` requests straddling a boundary.

### SlidingWindow

Blends the current and previous window counters weighted by elapsed time, smoothing the boundary burst.

```
weighted = round(prevCount × (1 − elapsed/windowSec)) + currCount
```

- **Cache keys**: `rl:sw:c:{key}:{windowID}`, `rl:sw:p:{key}:{prevWindowID}`
- **State**: 8-byte big-endian int64 per key
- **Trade-off**: more accurate than FixedWindow at the same storage cost; an approximation, not a true sliding log.

### TokenBucket

Tokens refill continuously at `Limit / TTL`. Each request consumes one token; blocked when `tokens < 1`.

- **Cache key**: `rl:tb:{key}`
- **State**: `[8 bytes float64 tokens][8 bytes int64 last_refill_ns]`
- **Trade-off**: smoothest rate control, no hard reset boundary; slightly more expensive (float64 arithmetic, 16-byte state).

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
