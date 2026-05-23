# Cache Module

*Cache module is a part of the `Ginject` framework. Phase 1 ships a high-performance, thread-safe in-memory backend with TTL support. The abstraction is backend-portable: a future Redis or distributed backend requires no changes to application code.*

- [Cache Module](#cache-module)
  - [Key Features](#key-features)
  - [Architecture Overview](#architecture-overview)
    - [Cache Interface](#cache-interface)
    - [Memory Backend](#memory-backend)
    - [CacheService](#cacheservice)
  - [Usage](#usage)
    - [Register the module](#register-the-module)
    - [Inject CacheService](#inject-cacheservice)
    - [Store and retrieve values](#store-and-retrieve-values)
    - [TTL (time-to-live)](#ttl-time-to-live)
    - [Delete an entry](#delete-an-entry)
  - [`CacheModuleOptions` Parameters](#cachemoduleoptions-parameters)
    - [IsGlobal](#isglobal)
    - [OnInit](#oninit)
  - [`CacheService` Methods](#cacheservice-methods)
    - [Get](#get)
    - [Set](#set)
    - [Delete](#delete)
    - [Keys](#keys)
    - [TTL](#ttl)
  - [API Semantics](#api-semantics)
  - [Error Handling](#error-handling)
  - [Performance Notes](#performance-notes)
  - [Background Cleanup](#background-cleanup)
  - [Future Backends](#future-backends)

---

## Key Features

- Zero third-party dependencies
- Thread-safe — passes `go test -race ./...`
- TTL expiration with nanosecond precision
- Sharded map design — 64 independent shards, minimal lock contention
- Lazy expiry on Get + amortized cleanup on Set + background sweep
- Value isolation — callers cannot mutate stored or returned bytes
- Fail-closed on empty keys

---

## Architecture Overview

### Cache Interface

```go
type Cache interface {
    Get(ctx context.Context, key string) ([]byte, bool)
    Set(ctx context.Context, key string, val []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Keys(ctx context.Context) []string
    TTL(ctx context.Context, key string) (time.Duration, bool)
}
```

The interface is backend-portable across in-memory, Redis, Memcached, or any distributed system. No backend-specific behaviour leaks through it.

### Memory Backend

The built-in `memoryCache` implementation uses:

| Technique | Purpose |
|-----------|---------|
| 64 shards (power-of-2) | Spreads lock contention across shards |
| Per-shard `sync.RWMutex` | Concurrent reads on distinct shards |
| Inline FNV-1a hash | Zero-allocation key → shard mapping |
| Unix-nanosecond timestamps | Sub-millisecond TTL precision, no `time.Time` boxing |
| Copy-on-Set, copy-on-Get | Caller cannot corrupt stored data |
| Amortized shard cleanup | Every 100th write per shard scans and evicts expired entries |
| Background sweep goroutine | Sweeps all shards every 30 s for write-light / read-heavy patterns |

### CacheService

`CacheService` is the injectable `core.Provider`. It wraps any `Cache` backend and exposes the same three methods. The framework's DI system copies the `Backend` interface value at wiring time, so all injected instances share the same underlying cache.

---

## Usage

### Register the module

Import the cache module in your root (or feature) module:

```go
package main

import (
    "github.com/dangduoc08/ginject/core"
    "github.com/dangduoc08/ginject/modules/cache"
)

func main() {
    app := core.New()
    app.Create(
        core.ModuleBuilder().
            Imports(
                cache.Register(&cache.CacheModuleOptions{
                    IsGlobal: true,
                }),
            ).
            Controllers(AppController{}).
            Build(),
    )
    app.Listen(8080)
}
```

Setting `IsGlobal: true` makes `CacheService` available everywhere without importing the module again.

### Inject CacheService

Declare `cache.CacheService` as a field in any provider or controller:

```go
type AppController struct {
    common.REST
    CacheService cache.CacheService
}
```

The framework resolves and injects `CacheService` automatically.

### Store and retrieve values

All values are `[]byte`. Use `encoding/json`, `encoding/gob`, or any serialiser you prefer.

```go
import (
    "context"
    "encoding/json"
    "time"
)

ctx := context.Background()

type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

// Store
user := User{ID: 1, Name: "Alice"}
data, _ := json.Marshal(user)
_ = controller.CacheService.Set(ctx, "user:1", data, 5*time.Minute)

// Retrieve
raw, ok := controller.CacheService.Get(ctx, "user:1")
if ok {
    var u User
    _ = json.Unmarshal(raw, &u)
    fmt.Println(u.Name) // Alice
}
```

### TTL (time-to-live)

```go
// Expires in 10 seconds
_ = svc.Set(ctx, "session:abc", token, 10*time.Second)

// Never expires (ttl = 0 or negative)
_ = svc.Set(ctx, "config:featureFlag", value, 0)
_ = svc.Set(ctx, "config:featureFlag", value, -1)  // also never expires
```

### Delete an entry

```go
err := svc.Delete(ctx, "session:abc")
if err != nil {
    // only errors on empty key
}
```

---

## `CacheModuleOptions` Parameters

### IsGlobal

**Type:** `bool`  
**Default:** `false`  
**Required:** `false`

When `true`, `CacheService` is available in every module without explicit import.

```go
cache.Register(&cache.CacheModuleOptions{
    IsGlobal: true,
})
```

### OnInit

**Type:** `func()`  
**Default:** `nil`  
**Required:** `false`

Called before the cache module is wired into the DI graph. Use it to pre-populate the cache or validate configuration.

```go
cache.Register(&cache.CacheModuleOptions{
    OnInit: func() {
        fmt.Println("cache module initialising")
    },
})
```

---

## `CacheService` Methods

### Get

Returns a copy of the stored value and `true` if the key exists and has not expired. Returns `nil, false` if the key is missing, expired, or empty.

**Signature:**

```go
func (cs *CacheService) Get(ctx context.Context, key string) ([]byte, bool)
```

**Parameters:**
- `ctx` — `context.Context` (passed to backend; used by future async backends)
- `key` — cache key; must be non-empty

**Returns:**
- `[]byte` — a new copy of the stored bytes; safe to mutate
- `bool` — `true` if a valid (non-expired) entry was found

**Example:**

```go
val, ok := svc.Get(ctx, "user:1")
if !ok {
    // cache miss — fetch from DB
}
```

### Set

Stores the value under key. The stored copy is independent of `val` — mutations to `val` after calling `Set` do not affect the cache.

**Signature:**

```go
func (cs *CacheService) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error
```

**Parameters:**
- `ctx` — `context.Context`
- `key` — must be non-empty; returns `cache.ErrEmptyKey` otherwise
- `val` — value bytes; `nil` and empty slice are both stored as an empty entry
- `ttl` — expiry duration; `<= 0` means the entry never expires

**Returns:**
- `error` — non-nil only when `key` is empty

**Example:**

```go
err := svc.Set(ctx, "rate:user:42", []byte("5"), 1*time.Minute)
if err != nil {
    log.Fatal(err)
}
```

### Delete

Removes the key from the cache. Deleting a missing key is a no-op (not an error).

**Signature:**

```go
func (cs *CacheService) Delete(ctx context.Context, key string) error
```

**Parameters:**
- `ctx` — `context.Context`
- `key` — must be non-empty; returns `cache.ErrEmptyKey` otherwise

**Returns:**
- `error` — non-nil only when `key` is empty

**Example:**

```go
_ = svc.Delete(ctx, "session:abc")
```

### Keys

Returns all non-expired keys currently in the cache. The order is not guaranteed. The returned slice is a new allocation — safe to mutate.

**Signature:**

```go
func (cs *CacheService) Keys(ctx context.Context) []string
```

**Returns:**
- `[]string` — snapshot of live keys at the moment of the call; may be empty but never nil

**Example:**

```go
keys := svc.Keys(ctx)
fmt.Println("cached keys:", keys)
```

### TTL

Returns the remaining time-to-live for a key.

**Signature:**

```go
func (cs *CacheService) TTL(ctx context.Context, key string) (time.Duration, bool)
```

**Returns:**
- `(0, false)` — key does not exist, is expired, or is empty
- `(0, true)` — key exists with no expiry (set with `ttl <= 0`)
- `(remaining, true)` — key exists and expires in `remaining` duration

**Example:**

```go
d, ok := svc.TTL(ctx, "session:abc")
if !ok {
    // key missing or expired
} else if d == 0 {
    fmt.Println("permanent entry")
} else {
    fmt.Println("expires in", d)
}
```

---

## API Semantics

| Behaviour | Detail |
|-----------|--------|
| `ttl = 0` | Entry never expires |
| `ttl < 0` | Entry never expires |
| `ttl > 0` | Entry expires at `now + ttl` (nanosecond precision) |
| Expired entry on Get | Returns `nil, false` — same as missing |
| Empty key on Set/Delete | Returns `cache.ErrEmptyKey` |
| Empty key on Get | Returns `nil, false` (no error) |
| Empty key on TTL | Returns `0, false` (no error) |
| `nil` val on Set | Stored as empty entry; Get returns `[]byte{}, true` |
| Returned `[]byte` from Get | Independent copy — safe to mutate |
| Stored `[]byte` from Set | Independent copy — caller mutations do not affect cache |
| Overwrite semantics | Set always overwrites; no conditional write |
| Keys | Returns snapshot of live keys; expired keys excluded |
| TTL on permanent key | Returns `0, true` |
| TTL on expired/missing key | Returns `0, false` |
| Concurrent access | All operations are goroutine-safe |

---

## Error Handling

The only error `Cache` operations return is `cache.ErrEmptyKey`:

```go
var ErrEmptyKey = errors.New("cache: key must not be empty")
```

Check for it explicitly:

```go
if err := svc.Set(ctx, key, val, ttl); errors.Is(err, cache.ErrEmptyKey) {
    // validate key before calling Set
}
```

---

## Performance Notes

Benchmark results on Intel Core i7-9750H (amd64, 12 threads):

| Operation | Throughput | ns/op | allocs/op |
|-----------|-----------|-------|-----------|
| Get (hit, single-thread) | 34 M/s | 36 | 1 |
| Get (miss, single-thread) | 68 M/s | 19 | 0 |
| Set (fixed key) | 22 M/s | 55 | 1 |
| Get (parallel, 12 goroutines) | 270 M/s | 45 | 1 |
| Set (parallel, 12 goroutines) | 26 M/s | 471 | 3 |
| Mixed 75% Get / 25% Set (parallel) | 135 M/s | 89 | 2 |
| Get (expired) | 12 M/s | 102 | 0 |

The single allocation on `Get` is the output copy (`make([]byte, n)`). Callers that can tolerate the slice being overwritten on the next Get call could hold a reference to the internal storage, but that is intentionally not exposed to preserve backend portability.

---

## Background Cleanup

The in-memory backend starts one background goroutine per cache instance. It sweeps all 64 shards every **30 seconds** and removes expired entries. The sweep holds each shard's write lock for the duration of that shard's scan only — it never holds multiple shard locks simultaneously.

This goroutine is complementary to:

- **Lazy expiry**: expired entries return a miss immediately on `Get` without deleting from the map.
- **Amortized cleanup**: every 100th `Set` to a shard triggers an inline scan of that shard.

Together, these three strategies ensure expired entries are collected efficiently regardless of the read/write ratio.

The goroutine exits when the process terminates. For graceful shutdown, the underlying `*memoryCache` exposes a `Stop()` method, but `CacheService` does not surface it — it is not required for correctness.

---

## Future Backends

To add a Redis backend, implement the `Cache` interface and pass it via a custom `Register` variant or factory:

```go
type redisCache struct { client *redis.Client }

func (r *redisCache) Get(ctx context.Context, key string) ([]byte, bool)                    { /* ... */ }
func (r *redisCache) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error { /* ... */ }
func (r *redisCache) Delete(ctx context.Context, key string) error                           { /* ... */ }
func (r *redisCache) Keys(ctx context.Context) []string                                      { /* KEYS * or SCAN */ }
func (r *redisCache) TTL(ctx context.Context, key string) (time.Duration, bool)              { /* TTL command */ }

// Then wire it:
svc := cache.CacheService{Backend: &redisCache{client: rdb}}
```

No application code changes are required — `CacheService` remains the same injectable type.
