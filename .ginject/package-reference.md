# Package Reference & Public APIs

**Optimization**: Structured reference, minimal prose, organized by package.

---

## Package: `core`

**Responsibility**: Application lifecycle, handler invocation, DI container

### App

```go
type App struct {
    http *HTTP
    ws *WS
    event *event.Event
    broker memorybroker.Broker
    Logger common.Logger
    // ... other fields
}
```

**Public Methods**:

| Method | Purpose | When To Use |
|--------|---------|------------|
| `New() *App` | Create app instance | Before Create() |
| `Create(m *Module)` | Initialize app with module tree | Once at startup |
| `Listen(port int) error` | Log routes, then serve with graceful SIGINT/SIGTERM shutdown (30s drain) | Run app |
| `Stop()` | Trigger the shutdown path (module `OnShutdown`, once) | Programmatic shutdown |
| `Get(p Provider) any` | Resolve a constructed provider | After Create() |
| `UseLogger(common.Logger) *App` | Install logger (wrapped with `MaskFields` masking) | Before Create() |
| `ServeHTTP(w, r)` | HTTP request handler | Auto-called by Go |
| `BindGlobalMiddlewares(fn ...)` | Register global middleware | Before Create() |
| `BindGlobalGuards(fn ...)` | Register global guards | Before Create() |
| `BindGlobalInterceptors(fn ...)` | Register global interceptors | Before Create() |
| `BindGlobalExceptionFilters(fn ...)` | Register global exception filters | Before Create() |
| `EnableWS(cfg *WSConfig, mw ...MiddlewareFn) *App` | Enable WebSocket; middlewares run in the handshake and can reject the upgrade | Before Create() |
| `EnableVersioning(versioning.Versioning) *App` | Enable route versioning | Before Create() |
| `EnableDevtool() *App` | Build the devtool snapshot. **Transport is not implemented** — logs `DevtoolNotServed` | Before Create() |
| `EnableAccessLog()` | Enable access logging | Before Create() |
| `SetMaxRequestBodySize(n int64) *App` | Cap request bodies; `0` disables. Default `DefaultMaxRequestBodyBytes` = 10 MB | Before Create() |

**Constants**: `DefaultMaxRequestBodyBytes` (10 MB), `DefaultWSMaxConnections` (10000), `DefaultWSMaxPayloadBytes` (1 MB), `DefaultWSWriteTimeout` (10s). See [security-limits-and-defaults.md](security-limits-and-defaults.md).

> Earlier revisions of this file listed `SetMaxBodySize(bytes)`. No such method ever existed; the real name is `SetMaxRequestBodySize`.

### Module

```go
type Module struct {
    providers []Provider
    controllers []Controller
    // ... handler lists
}
```

**Factory Methods**:

| Method | Purpose |
|--------|---------|
| `ModuleBuilder() *ModuleBuilder` | Create module builder |
| `Build() *Module` | Build module |
| `NewModule() *Module` | Create module instance (internal) |

### ModuleBuilder

The builder type is **unexported** (`*moduleBuilder`); obtain one from `core.ModuleBuilder()`. Exactly four methods exist:

| Method | Purpose |
|--------|---------|
| `Imports(modules ...any) *moduleBuilder` | Import `*Module` values or `func(...) *core.Module` factories |
| `Providers(providers ...Provider) *moduleBuilder` | Register providers |
| `Controllers(controllers ...Controller) *moduleBuilder` | Register controllers |
| `Build() *Module` | Build the module |

> Earlier revisions listed `Export(...string)` and `IsGlobal(bool)` builder methods. **Neither exists.** `IsGlobal` is a field on `*Module`, set after `Build()`:
> ```go
> m := core.ModuleBuilder().Providers(Svc{}).Build()
> m.IsGlobal = true
> ```
> There is no export mechanism — a module's providers flow to its parent automatically via `prependInjectedModules`.

### Provider

```go
type Provider interface {
    NewProvider() Provider
}
```

**Implementation**: Any struct implementing this is a provider

---

## Package: `ctx`

**Responsibility**: Request context objects, dependency types

### HTTPContext

```go
type HTTPContext struct {
    *http.Request
    http.ResponseWriter
    
    Code int
    Timestamp time.Time
    Deadline time.Time
    // ... other fields
}
```

**Public Methods**:

| Method | Purpose | Returns |
|--------|---------|---------|
| `Init(w, r)` | Initialize context | - |
| `InitWithMaxBodySize(w, r, limit)` | Initialize with body size limit | - |
| `Reset()` | Clear all fields (for pooling) | - |
| `Status(code) *HTTPContext` | Set HTTP status code | self |
| `SetDeadline(duration) *HTTPContext` | Set request timeout | self |
| `IsDeadlineExceeded() bool` | Check if timeout passed | bool |
| `Text(data, args...) void` | Write text response | - |
| `JSON(data...) void` | Write JSON response | - |
| `JSONP(data...) void` | Write JSONP response | - |
| `Redirect(url) void` | Redirect response | - |
| `GetID() string` | Get request ID | string |

### WSContext

```go
type WSContext struct {
    *websocket.Conn
    Timestamp time.Time
}
```

**Public Methods**:

| Method | Purpose | Returns |
|--------|---------|---------|
| `Init(w, r)` | Initialize context | - |
| `Reset()` | Clear fields | - |
| `WSPayload() *WSPayload` | Get current message | *WSPayload |
| `GetID() string` | Get request ID | string |

### Built-in Dependency Types

**HTTP-specific**:
- `*ctx.HTTPContext` — full request context
- `*http.Request` — raw request
- `http.ResponseWriter` — raw response
- `ctx.Body` — parsed body
- `ctx.Query` — query parameters
- `ctx.Header` — headers
- `ctx.Param` — path parameters
- `ctx.Form` — form data
- `ctx.File` — file uploads
- `ctx.Next` — middleware continuation function
- `ctx.Redirect` — redirect function

**WebSocket-specific**:
- `*ctx.WSContext` — WS request context
- `*websocket.Conn` — raw connection
- `ctx.WSPayload` — incoming message
- `ctx.Next` — middleware continuation
- `common.Publisher` — message publisher

---

## Package: `common`

**Responsibility**: Pipeline interfaces, built-in filters

### Interfaces

**MiddlewareFn**:
```go
type MiddlewareFn func(*http.Request, http.ResponseWriter, ctx.Next)
```

**Guarder**:
```go
type Guarder interface {
    CanActivate(*ctx.HTTPContext) bool
}
```

**Interceptable**:
```go
type Interceptable interface {
    Intercept(*ctx.HTTPContext, *aggregation.Aggregation) any
}
```

**ExceptionFilterable**:
```go
type ExceptionFilterable interface {
    Catch(*exception.Exception, *ctx.HTTPContext)
}
```

**Logger**:
```go
type Logger interface {
    Log(level string, data any)
    LogStructured(level string, fields map[string]any)
}
```

**Publisher**:
```go
type Publisher interface {
    Publish(topic string, data any) error
}
```

### Embedded Types (for Controllers)

**REST** — Embed in controller to enable HTTP routing:
```go
type REST struct {
    Middleware common.Middleware
    Guard common.Guard
    Interceptor common.Interceptor
    ExceptionFilter common.ExceptionFilter
}

func (r REST) BindMiddleware(fn MiddlewareFn, handlers ...) REST
func (r REST) BindGuard(fn Guarder, handlers ...) REST
// ... etc
```

**WS** — Embed in controller to enable WebSocket:
```go
type WS struct {
    Middleware common.Middleware
    Guard common.Guard
    // ... other fields
}
```

---

## Package: `routing`

**Responsibility**: URL routing, path matching

### Router

```go
type Router struct {
    trie *ds.Trie
    routerItemByPattern map[string][]RouterItem
}
```

**Public Methods**:

| Method | Purpose | Returns |
|--------|---------|---------|
| `NewRouter() *Router` | Create router | *Router |
| `Match(path, method) (RouterItem, bool)` | Find route | (RouterItem, bool) |

### RouterItem

```go
type RouterItem struct {
    Method string
    Pattern string
    Handlers []ctx.HTTPHandler
    ParamKeys map[string][]int
}
```

### Naming Tokens

| Token | HTTP Method | Example |
|-------|-------------|---------|
| `READ` | GET | `READ_BY_ID` → GET /:id |
| `CREATE` | POST | `CREATE` → POST / |
| `UPDATE` | PUT | `UPDATE_BY_ID` → PUT /:id |
| `MODIFY` | PATCH | `MODIFY_BY_ID` → PATCH /:id |
| `DELETE` | DELETE | `DELETE_BY_ID` → DELETE /:id |
| `PREFLIGHT` | OPTIONS | `PREFLIGHT` → OPTIONS / |
| `BY` | Path param | `_BY_ID` → /:id |
| `AND` | Additional segment | `_AND_NAME` → /:name |
| `OF` | Sub-resource | `_OF_COMMENTS` → /comments |
| `VERSION_X` | API version | `_VERSION_2` → /v2/... |

---

## Package: `exception`

**Responsibility**: Exception types and handling

### Exception

```go
type Exception struct {
    message string
    error error
    code int
    stackTrace string
}
```

**Public Methods**:

| Method | Purpose | Returns |
|--------|---------|---------|
| `GetCode() int` | Get exception code | int |
| `GetMessage() string` | Get user message | string |
| `GetStatusText() string` | Get HTTP status text | string |
| `GetStackTrace() string` | Get captured stack trace | string |
| `SetStackTrace(trace) void` | Set stack trace | - |

**Built-in Constructors**:

```go
exception.BadRequestException(message string, opts...) Exception
exception.UnauthorizedException(message string, opts...) Exception
exception.ForbiddenException(message string, opts...) Exception
exception.NotFoundException(message string, opts...) Exception
exception.ConflictException(message string, opts...) Exception
exception.InternalServerErrorException(message string, opts...) Exception
exception.RequestTimeoutException(message string, opts...) Exception
exception.NotImplementedException(message string, opts...) Exception
exception.ServiceUnavailableException(message string, opts...) Exception
```

---

## Package: `event`

**Responsibility**: Event emission and listening

### Event

```go
type Event struct {
    // Internal sync.Map
}
```

**Public Methods**:

| Method | Purpose | Returns |
|--------|---------|---------|
| `NewEvent() *Event` | Create event emitter | *Event |
| `On(name string, fn func(data)) void` | Register listener | - |
| `Emit(name string, data any) void` | Emit event | - |
| `HasListeners(name string) bool` | Check if listeners | bool |

**Usage**:
```go
event := event.NewEvent()
event.On("user.created", func(data any) {
    // Handle event
})
event.Emit("user.created", userData)
```

---

## Package: `log`

**Responsibility**: Structured logging

### Log Levels

Standard levels:
- "DEBUG"
- "INFO"
- "WARN"
- "ERROR"
- "FATAL"

**Public Functions**:

```go
func NewLog(opts *LogOptions) Logger
func WrapLogger(logger Logger, maskFields []string) Logger
```

---

## Package: `memorybroker`

**Responsibility**: Lightweight in-process topic-based pub/sub broker (no queue groups, no persistence, no ack/retry — see `memorybroker/README.md` for the full spec, kept in sync with this section)

### Constructor & Options

```go
func NewMemoryBroker(opts ...Option) Broker
func WithPanicHandler(fn PanicHandler) Option   // fn(topic string, recovered any)
```

Subscriber panics are recovered per-handler by `callHandler` so one bad handler cannot break the publish loop. Default reports to stderr; `WithPanicHandler` routes it to your logger/metrics. A panic inside the handler is itself recovered.

### Broker Interface

```go
type Broker interface {
    Subscribe(topic string, handler MessageHandler) (Subscription, error)
    Unsubscribe(sub Subscription) error
    Publish(topic string, payload any) error
    PublishAsync(topic string, payload any) error
    Close() error
}

type Message struct {
    Topic     string    // Topic name
    Payload   any       // Message data
    Timestamp time.Time // Captured before dispatch (time.Now())
}

type MessageHandler func(*Message)

type Subscription interface {
    ID() string
    Topic() string
    Unsubscribe() error
}

var (
    ErrClosed              = errors.New("memorybroker: broker is closed")
    ErrNilHandler          = errors.New("memorybroker: handler must not be nil")
    ErrEmptyTopic          = errors.New("memorybroker: topic must not be empty")
    ErrForeignSubscription = errors.New("memorybroker: subscription does not belong to this broker")
)
```

**No `ID`/`Metadata` on `Message`, no `Once`/`SubscribeQueue`/`Off`/`Topics`/`ListenerCount`/`Clear`/`Stats` on `Broker`** — these existed in an earlier, richer design and were deliberately removed for a leaner surface. Do not regenerate them from memory; verify against `memorybroker/broker.go` before citing the API.

### Pattern Matching

Segment-based matcher (`pattern` package), **not regex** — only a bare `*` per dot-separated segment is special:
- **Exact**: `user.login` — O(1) map lookup, matches only that literal topic
- **Suffix Wildcard**: `user.*` — O(1) map lookup by prefix; greedy, matches *any depth* below `user.` (`user.a`, `user.a.b`, ...), not just one level
- **Global**: `*` — O(1), matches every topic. `*` is the **only** wildcard token — `>` is NOT special-cased anywhere in the `pattern` package (verified empirically); a pattern containing `>` is parsed as a literal segment and matches nothing else. `memorybroker/README.md` previously claimed `>` was an alias for global — that was never true and has been corrected
- **Complex**: `user.*.profile`, `*.created` — O(n) scan over registered complex patterns; `*` can appear mid-path or more than once

### Delivery Model

- **Fan-Out only**: every subscription matching a topic fires on every `Publish`/`PublishAsync` — no `Once`, no queue-group load balancing
- **`Publish`**: synchronous, dispatches on the caller's goroutine; not tracked by `Close`'s `WaitGroup`
- **`PublishAsync`**: one goroutine per call (no worker pool, no bounded queue), tracked by an internal `sync.WaitGroup` so `Close` can drain in-flight calls
- **Snapshot dispatch**: handlers are copied into a private slice under `RLock`, the lock is released, *then* handlers run — a handler may safely call back into `Subscribe`/`Unsubscribe`/`Publish`/`PublishAsync` without deadlocking (verified; see Anti-Patterns for the one exception)
- **Panic recovery**: unconditional `recover()` around every handler call — one panicking handler never stops the rest or crashes the broker (not configurable, no `OnPanic` hook)

**Usage**:
```go
// Subscribe to pattern (via memorybroker package)
sub, _ := broker.Subscribe("users.*", func(msg *memorybroker.Message) {
    log.Printf("Topic: %s, Data: %v", msg.Topic, msg.Payload)
})

// Publish message
broker.Publish("users.created", userData)

// Cleanup
broker.Unsubscribe(sub)

// Alternative (preferred): Inject Publisher interface
app.BindWSHandler("users.created", func(pub common.Publisher) {
    pub.Publish("users.profile.updated", profileData)
})
```

### Architecture

- **Locking**: one `sync.RWMutex` guards four maps (`exactByTopic`, `prefixByPrefix`, `globalByID`, `complexByTopic`) — no sharding
- **`Close()`**: `CompareAndSwap`-guarded (idempotent, safe from any goroutine) → `wg.Wait()` for in-flight `PublishAsync` goroutines → clears all four maps. Does **not** wait for a concurrent synchronous `Publish` call (see Concurrency Model)
- **Cross-broker protection**: `Unsubscribe` checks `subscription.broker == b`, rejecting a `Subscription` from a different `MemoryBroker` with `ErrForeignSubscription`
- **`Unsubscribe(nil)`**: always `nil`, unconditionally, even on a closed broker — the one exception to "all calls return `ErrClosed` after Close"

### Key Features

- ✓ Thread-safe concurrent Subscribe/Unsubscribe/Publish/PublishAsync/Close (single mutex, race-tested — see `broker_test.go`'s concurrent test suite)
- ✓ `publishInternal` sizes the handler slice exactly and skips allocation entirely when nothing matches a topic
- ✓ No sharding, no background sweep, no TTL/expiration — this package holds no state beyond active subscriptions (compare `memorycache`, which does shard and sweep)

---

## Package: `trace`

**Responsibility**: Tracing and observability

### Event Names & Stages

```go
const EventName = "ginject:trace"

// Stages:
const (
    StageHandshake = "handshake"        // WS handshake
    StageMiddleware = "middleware"      // Middleware execution
    StageGuard = "guard"                // Guard execution
    StageInterceptor = "interceptor"    // Interceptor execution
    StagePipe = "pipe"                  // Pipeable transformation
    StageHandler = "handler"            // Main handler
    StageExceptionFilter = "filter"     // Exception filter
    StageComplete = "complete"          // Request complete
)

// Transports:
const (
    TransportHTTP = "http"
    TransportWS = "ws"
)
```

### Trace Event

```go
type Event struct {
    ID string              // Request ID
    Stage string            // Execution stage
    Name string             // Handler/middleware/filter name
    Transport string        // "http" or "ws"
    Operation string        // HTTP method or WS operation
    Target string           // URL path or topic
    Code int                // HTTP status or WS close code
    Duration time.Duration  // Stage execution time
}
```

---

## Package: `aggregation`

**Responsibility**: Interceptor response aggregation

### Aggregation

```go
type Aggregation struct {
    Data any                           // Handler return value
    Exception *exception.Exception     // Panic exception (if any)
    Request *http.Request
    Response http.ResponseWriter
}
```

**Used in**: Interceptor post-handler phase

---

## Package: `modules`

**Responsibility**: Built-in modules

### Config Module

```go
func NewConfigModule(envPath string) *core.Module
```

**Provides**: `ConfigService` (loads .env files, typed struct binding)

### Cache Module

```go
func Register(opts *CacheModuleOptions) *core.Module

type CacheModuleOptions struct {
    IsGlobal   bool
    OnInit     CacheOnInitFn
    Backend    Cache   // supply your own; otherwise a memorycache is created
    MaxEntries int     // caps the default backend; 0 = memorycache.DefaultMaxEntries, negative = unlimited
}
```

**Provides**: `CacheService`, backed by `memorycache` (TTL + sampled eviction — **not** LFU).

**Lifecycle**: when `Register` creates the backend it wires `module.OnShutdown = backend.Stop`, so the sweeper goroutine is stopped. A caller-supplied `Backend` is left for the caller to stop.

> Earlier revisions listed `NewCacheModule()`. No such function exists.

### HTTPClient Module

```go
func Register(opts *HTTPClientModuleOptions) *core.Module
```

**Provides**: `ClientService` (wraps `http.Client`).

> Earlier revisions listed `NewHTTPClientModule()`. No such function exists.

---

## Package: `memorycache`

**Responsibility**: Sharded in-memory **TTL** cache with a bounded entry count.

Not LFU — there is no access-frequency tracking anywhere in the package. Eviction prefers expired entries, then the sampled entry closest to expiry (`evictSampleSize` = 8).

### MemoryCache

```go
type MemoryCache struct {
    // Internal sharded storage, background sweep
}

// Constructor (variadic options)
func NewMemoryCache(opts ...Option) *MemoryCache
func WithMaxEntries(n int) Option   // total cap across shards; 0 or less = unlimited

// Defaults
const DefaultMaxEntries = 100_000

// Operations (context-based API)
func (m *MemoryCache) Get(ctx context.Context, key string) ([]byte, bool)
func (m *MemoryCache) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error
func (m *MemoryCache) SetNX(ctx context.Context, key string, val []byte, ttl time.Duration) (bool, error)
func (m *MemoryCache) Delete(ctx context.Context, key string) error
func (m *MemoryCache) Keys(ctx context.Context) []string
func (m *MemoryCache) TTL(ctx context.Context, key string) (time.Duration, bool)
func (m *MemoryCache) Mutate(ctx context.Context, key string, fn func(old []byte, exists bool) (newVal []byte, ttl time.Duration)) ([]byte, error)
func (m *MemoryCache) Stop()  // stops the sweeper goroutine; idempotent (sync.Once)
```

**Sharding**: 256 shards, one `sync.RWMutex` each, keyed by `hashKey(key)&shardMask` — unrelated keys do not contend.

**`Stop()` does not flush anything** — it closes the done channel and waits for the sweeper. Earlier revisions described it as flushing persistence.

**`Mutate`** is the atomic read-modify-write path (`modules/cache.AtomicMutator`); it holds the shard lock across `fn`, so `fn` must not block.

### Architecture

- **Sharding**: 256 shards, one `sync.RWMutex` each
- **Eviction**: bounded by `DefaultMaxEntries` (100k) via `admitLocked` on every write; prefers expired entries, then the sampled entry closest to expiry (`evictSampleSize` = 8). Replacing an existing key never evicts
- **Opportunistic cleanup**: `evictLocked` runs every `cleanupEvery` (128) writes per shard, batch `cleanupBatch` (64)
- **Sweep**: background goroutine, one shard per tick, full cycle every 5s
- **Thread-safe**: per-shard locking; `Stop()` idempotent via `sync.Once`

### Usage

```go
cache := memorycache.NewMemoryCache()                          // bounded at DefaultMaxEntries
cache := memorycache.NewMemoryCache(memorycache.WithMaxEntries(0))  // unlimited
defer cache.Stop()

cache.Set(ctx, "key", []byte("value"), time.Hour)
val, ok := cache.Get(ctx, "key")
cache.Delete(ctx, "key")
```

> **There is no persistence.** Earlier revisions of this file documented a `PersistenceConfig` type, `NewMemoryCacheWithConfig`, atomic temp-file writes, dirty tracking and JSON snapshots. None of it exists, and `git log -S PersistenceConfig` shows it never did. The cache is memory-only and loses everything on restart.
>
> Those revisions also described eviction as **LFU**. There is no access-frequency tracking in the package.

---

## Package: `wsevent`

**Responsibility**: WebSocket event routing

### Matcher

```go
type Matcher struct {
    // Internal pattern matching
}

func Match(topic, pattern string) bool
```

**Patterns**:
- Exact: "users.created" matches "users.created"
- Wildcard: "users.*" matches "users.created", "users.updated"
- Regex: "users\.[a-z]+" matches "users.abc"

---

## Package: `matcher`

**Responsibility**: General pattern matching

**Usage**: Match strings against patterns (used by routing and WS event routing)

---

## Known Non-Public Packages

**Do NOT use** (internal implementation):
- `internal/` — Private utilities (color, ds, str, etc.)
- `devtool/` — Development tools
- `accesslog/` — Access logging
- `data/` — Internal data structures
- `versioning/` — Version management

---

## Packages Not Detailed Above

Each has a README in its own directory; this table exists so an agent knows the package is there and what its security-relevant surface is. Do not duplicate their docs here — link to them.

| Package | Responsibility | Security-relevant surface |
|---|---|---|
| `modules/storage` | Embedded append-only document DB (segments, primary/secondary/text indexes, transactions, compaction) | `StoreModuleOptions.Schemas` / `OpenWithSchemas` pre-declare schemas so one scan builds every index. `Model.Schema` **panics** if the engine cannot be opened. See `modules/storage/README.md` |
| `modules/config` | `.env` loader with typed struct binding | Errors name the key only, never the value |
| `modules/httpclient` | `http.Client` wrapper with retries | Retry buffers `rawBody` so a retried request is not sent empty; `BodyStream` is closed between attempts; failed downloads remove the partial file |
| `middlewares/cors` | CORS for HTTP **and** WS upgrades | Wildcard origin + `IsAllowCredentials` **panics** at config time. `compiledCORS.Use` gates WS upgrades via `matchOrigin`. See `middlewares/cors/README.md` |
| `middlewares/csrf` | Double-submit CSRF | `SameSite=Lax` + auto-`Secure` over TLS; `HttpOnly` is deliberately **false** |
| `middlewares/helmet` | Security headers | Strict CSP, COEP/COOP/CORP, `no-referrer`, HSTS by default |
| `guards/throttler` | Rate limiting | Keys off `RemoteAddr` only; `TrustProxyHeaders` opts into `X-Real-IP`/`X-Forwarded-For`. See `guards/throttler/README.md` |
| `pattern` | Topic pattern matching for broker/WS events | `*` is the **only** wildcard. Trailing `*` matches one or more remaining segments; mid-pattern `*` matches exactly one. `>` is a literal, never a wildcard |
| `versioning` | Route version resolution (header/URI strategies) | — |
| `accesslog` | Subscribes to trace events, logs per-stage timings | Logs method/route/duration only — no headers, no bodies |
| `devtool` | Builds a route/pipeline snapshot | **Transport not implemented.** `Serve(ctx) error` blocks until ctx is done. Still pulls `grpc` + `protobuf` into `go.mod` for code that does not run |

Full caps and defaults: [security-limits-and-defaults.md](security-limits-and-defaults.md).

