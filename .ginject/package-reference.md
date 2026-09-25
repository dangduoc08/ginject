# Package Reference & Public APIs

**Optimization**: Structured reference, minimal prose, organized by package.

**Verification policy**: every signature below was read directly from source on 2026-09-25. If you (an AI agent) are about to generate code against an API in this file, and this file is more than a few weeks old relative to the repo, spot-check the signature with a grep before trusting it — this file has previously contained fabricated APIs that looked exactly as confident as everything else in it.

---

## Package: `core`

**Responsibility**: Application lifecycle, handler invocation, DI container

### App

```go
type App struct {
    http    *HTTP
    ws      *WS
    event   *event.Event
    broker  memorybroker.Broker
    Logger  common.Logger
    // ... other unexported fields
}
```

**Public Methods** (all chainable, return `*App`, unless noted):

| Method | Purpose |
|--------|---------|
| `New() *App` | Create app instance |
| `Create(m *Module)` | Initialize app with module tree — call exactly once |
| `Listen(port int) error` | Start HTTP server, blocks until shutdown |
| `Stop()` | Trigger graceful shutdown programmatically (idempotent via `shutdownOnce`) |
| `Get(p Provider) any` | Look up an already-injected provider by type |
| `ServeHTTP(w, r)` | HTTP handler (auto-called by Go stdlib) |
| `BindGlobalMiddlewares(fn ...common.MiddlewareFn) *App` | Register global middleware |
| `BindGlobalGuards(g ...common.Guarder) *App` | Register global guards |
| `BindGlobalInterceptors(i ...common.Interceptable) *App` | Register global interceptors |
| `BindGlobalExceptionFilters(f ...common.ExceptionFilterable) *App` | Register global exception filters |
| `EnableWS(cfg *WSConfig, middlewares ...common.MiddlewareFn) *App` | Enable WebSocket support — **requires a `*WSConfig` argument**, not zero-arg |
| `EnableDevtool() *App` | Enable devtool snapshot (its gRPC transport is unimplemented — builds a snapshot, serves nothing) |
| `EnableAccessLog() *App` | Enable access logging |
| `EnableVersioning(v versioning.Versioning) *App` | Enable API versioning |
| `UseLogger(logger common.Logger) *App` | Replace the default logger |
| `UseLogOptions(opts *log.LogOptions) *App` | Configure the default logger |

**There is no `SetMaxBodySize`/`SetMaxRequestBodySize` method anywhere.** The request body cap is a fixed, unexported field (`maxRequestBodyBytes`) set once in `New()` to `DefaultMaxRequestBodyBytes` — see "HTTP Server Defaults" below.

### HTTP Server Defaults

Constants (`core/app.go`), all currently hardcoded with no public setters:

| Constant | Value | Applied in |
|----------|-------|------------|
| `DefaultMaxRequestBodyBytes` | `10 << 20` (10MB) | `ServeHTTP` wraps `r.Body` in `http.MaxBytesReader(w, r.Body, app.maxRequestBodyBytes)` before `c.Init` |
| `DefaultMaxHeaderBytes` | `1 << 20` (1MB) | `http.Server.MaxHeaderBytes` in `Listen()` |
| `DefaultReadHeaderTimeout` | `2 * time.Second` | `http.Server.ReadHeaderTimeout` |
| `DefaultReadTimeout` | `5 * time.Second` | `http.Server.ReadTimeout` |
| `DefaultWriteTimeout` | `15 * time.Second` | `http.Server.WriteTimeout` |
| `DefaultIdleTimeout` | `60 * time.Second` | `http.Server.IdleTimeout` |

Exceeding the body cap surfaces as a 413 `exception.RequestEntityTooLargeException` from `ctx.Body()`, not a raw error. These four timeouts bound the connection, not handler execution time — there is no per-handler deadline (see `ctx` section).

### Module

```go
type Module struct {
    id       string
    Name     string
    prefixes []string

    staticModules  []*Module
    dynamicModules []any
    providers      []Provider
    controllers    []Controller

    IsGlobal   bool
    OnInit     func()
    OnReady    func()
    OnShutdown func()

    HTTPExceptionFilters []common.HTTPLayer
    HTTPMiddlewares      []common.HTTPLayer
    HTTPGuards           []common.HTTPLayer
    HTTPInterceptors     []common.HTTPLayer
    HTTPMainHandlers     []common.HTTPLayer
    WSGuards             []common.WSLayer
    WSInterceptors       []common.WSLayer
    WSExceptionFilters   []common.WSLayer
    WSMainHandlers       []common.WSLayer
}
```

**Methods**: `Prefix(prefix string) *Module`, `ID() string`.

**Lifecycle hooks**: `OnInit` runs during `app.Create()`'s module walk. `OnReady` and `OnShutdown` are both called once per module, in the **same** order (a single pre-order walk of `collectModules()`) — `OnShutdown` is NOT reversed/LIFO relative to `OnReady`, despite that being the intuitive assumption. `OnShutdown` fires on SIGINT/SIGTERM or `app.Stop()`, guarded so it runs exactly once even if both paths fire.

### ModuleBuilder

The exported constructor is a **function**, not a type — the concrete type is unexported:

```go
func ModuleBuilder() *moduleBuilder   // core.moduleBuilder is unexported; use the constructor
```

| Method | Signature | Notes |
|--------|-----------|-------|
| `Imports` | `Imports(modules ...any) *moduleBuilder` | Takes `...any`, not `...*Module` — also accepts dynamic-module factory values |
| `Providers` | `Providers(providers ...Provider) *moduleBuilder` | |
| `Controllers` | `Controllers(controllers ...Controller) *moduleBuilder` | |
| `Build` | `Build() *Module` | |

**There is no `Export(...string)` method and no `IsGlobal(bool)` builder method — both are fabrications from an earlier bad doc pass.** `IsGlobal` is a public **field** set directly on the built `*Module` (`module.IsGlobal = true`), not chained on the builder.

### Provider / Controller

```go
type Provider interface {
    NewProvider() Provider
}

type Controller interface {
    NewController() Controller
}
```

Any struct implementing the respective method qualifies — no tags, no registration call needed beyond passing the value to `ModuleBuilder().Providers(...)` / `.Controllers(...)`.

---

## Package: `ctx`

**Responsibility**: Request context objects and dependency types

### HTTPContext

```go
type HTTPContext struct {
    *http.Request
    http.ResponseWriter

    Next      Next
    Code      int       // defaults to http.StatusOK
    Timestamp time.Time
    // body, form, file, query, header, param, ParamKeys, ParamValues are also present
}
```

**No `Deadline` field, no `SetDeadline()`, no `IsDeadlineExceeded()` method exist.** There is no per-request timeout API anywhere in this framework — only the server-level timeouts in `core.App`'s defaults table above.

| Method | Signature | Notes |
|--------|-----------|-------|
| `Init` | `Init(w http.ResponseWriter, r *http.Request)` | No `InitWithMaxBodySize` variant exists |
| `Reset` | `Reset()` | Clears all fields for pool reuse |
| `Status` | `Status(code int) *HTTPContext` | Sets `c.Code`; chainable — call BEFORE `JSON`/`Text`, since they write using whatever `Code` is current |
| `Text` | `Text(data string, args ...any)` | |
| `JSON` | `JSON(data ...any)` | Variadic, not `(data any)` |
| `JSONP` | `JSONP(data ...any)` | Falls back to `JSON` if no `?callback=` query param |
| `GetID` | `GetID() string` | |
| `Body` | `Body() Body` | Lazily parses JSON body; on `*http.MaxBytesError` panics `exception.RequestEntityTooLargeException` (413) instead of a raw error |

### WSContext

```go
type WSContext struct {
    *websocket.Conn
    Next      Next
    Timestamp time.Time
}
```

| Method | Signature |
|--------|-----------|
| `Init` | `Init(conn *websocket.Conn)` — one parameter, NOT `(w, r)` |
| `Reset` | `Reset()` |
| `WSPayload` | `WSPayload() WSPayload` — returns the map by value, NOT `*WSPayload` |
| `SetWSPayload` | `SetWSPayload(p WSPayload)` |
| `Send` | `Send(data any)` |
| `SetSend` | `SetSend(fn func(data any))` |
| `Context` / `SetContext` | `Context() context.Context` / `SetContext(ctx context.Context)` |

`type WSPayload map[string]any` — a map type, not a struct.

### Built-in Dependency Types

**HTTP-specific**: `*ctx.HTTPContext`, `*http.Request`, `http.ResponseWriter`, `ctx.Body`, `ctx.Query`, `ctx.Header`, `ctx.Param`, `ctx.Form`, `ctx.File`, `ctx.Next`, `ctx.Redirect`, `common.Publisher`

**WebSocket-specific**: `*ctx.WSContext`, `*websocket.Conn`, `ctx.WSPayload`, `ctx.Next`, `common.Publisher`

---

## Package: `common`

**Responsibility**: Pipeline shape-checking, embeddable controller building blocks

### Cross-Cutting Concern Types

**Only `MiddlewareFn` is a real, compile-time Go interface. `Guarder`, `Interceptable`, and `ExceptionFilterable` are all `type X any` — checked at bind time via reflection**, not the compiler:

| Declared type | Real definition | Checked shape (func alias) | Bind-time check fails with |
|---|---|---|---|
| `MiddlewareFn` | `interface { Use(*http.Request, http.ResponseWriter, ctx.Next) }` | `Use = func(*http.Request, http.ResponseWriter, ctx.Next)` | compile error if no `Use` method |
| `Guarder` | `any` | `HTTPCanActivate = func(*ctx.HTTPContext) bool`, method name `"CanActivate"` | `GuardShapeError` |
| `Interceptable` | `any` | `HTTPIntercept = func(*ctx.HTTPContext, *aggregation.Aggregation) any`, method name `"Intercept"` | `InterceptorShapeError` |
| `ExceptionFilterable` | `any` | `HTTPCatch = func(*ctx.HTTPContext, *exception.Exception)`, method name `"Catch"` | `ExceptionFilterShapeError` |

**`Catch`'s parameter order is `(*ctx.HTTPContext, *exception.Exception)` — HTTPContext FIRST.** A common mistake (present in older docs and even in the project's own `CLAUDE.md`) is writing `Catch(*exception.Exception, *ctx.HTTPContext)`, which fails the reflection shape check.

A bare func literal does **not** satisfy any of these — you must bind a struct value with the matching method. Example:
```go
type AuthGuard struct{}
func (g AuthGuard) CanActivate(httpCtx *ctx.HTTPContext) bool { return isAuthorized(httpCtx) }
// c.BindGuard(AuthGuard{}, c.READ_BY_ID)
```

### `Logger` / `Publisher`

```go
type Logger interface {
    Debug(msg string, args ...any)
    Info(msg string, args ...any)
    Warn(msg string, args ...any)
    Error(msg string, args ...any)
    Fatal(msg string, args ...any)
}

type Publisher interface {
    Publish(topic string, payload ...any) error   // variadic payload, not a single `data any`
}
```

There is no `Log()`/`LogStructured()` method on `Logger` — it mirrors slog-style leveled logging.

### Embeddable Controller Types

**There is no `common.REST` type — this is a pervasive fabrication from earlier docs. The real embed is `common.HTTP`.** Four cross-cutting-concern types are embedded *separately* alongside it (or alongside `common.WS`), each with its own `Bind*` method:

```go
type UserController struct {
    common.HTTP              // required for HTTP routing
    common.Middleware        // optional: BindMiddleware(MiddlewareFn, handlers ...any) *Middleware
    common.Guard             // optional: BindGuard(Guarder, handlers ...any) *Guard
    common.Interceptor       // optional: BindInterceptor(Interceptable, handlers ...any) *Interceptor
    common.ExceptionFilter   // optional: BindExceptionFilter(ExceptionFilterable, handlers ...any) *ExceptionFilter
}
```

`common.HTTP` itself only carries routing bookkeeping (`PatternToFuncNameMap`, `RouterMap`, etc.) — it has no `Middleware`/`Guard`/`Interceptor`/`ExceptionFilter` fields; those are separate embeds. `common.WS` follows the same pattern for WebSocket controllers (`funcNameByEvent`, `EventMap`).

### `common.Construct` — Singleton-By-Type-Name Caveat

`Construct(obj any, constructor string) any` (common/fn.go) backs **every** bind of a Middleware/Guard/Interceptor/ExceptionFilter (global or per-controller, HTTP and WS). If the bound type defines an optional `New*()` constructor (`NewMiddleware`, `NewGuard`, `NewInterceptor`, `NewExceptionFilter`), `Construct` calls it **exactly once per Go type name**, process-wide, via a per-key `sync.Once`, and caches the result — every later bind of that same type reuses the cached instance regardless of the field values on the literal passed in.

**Consequence**: binding the same struct type twice with different configuration silently collapses to one instance — whichever was constructed first wins everywhere:
```go
// Controller A:
c.BindMiddleware(cors.CORS{AllowOrigin: []string{"https://a.example.com"}})
// Controller B (different config, same Go type):
c.BindMiddleware(cors.CORS{AllowOrigin: []string{"https://b.example.com"}})
// Whichever of A/B's cors.CORS is constructed FIRST wins for BOTH controllers.
```
This is the opposite of `Provider`'s per-injection semantics — do not assume a fresh instance per bind for any type that defines a `New*()` method (or is bound as a bare struct at all, since even without a `New*()` method the literal itself is still funneled through `Construct` and cached by type name).

---

## Package: `routing`

**Responsibility**: URL routing and path matching

### Router

```go
type Router struct {
    trie                *ds.Trie
    routerItemByPattern  map[string][]RouterItem
    GlobalMiddlewares    []ctx.HTTPHandler
    InjectableHandlers   map[string]any
}

func NewRouter() *Router
```

| Method | Signature |
|--------|-----------|
| `Match` | `Match(method, route, version string) (matched bool, pattern string, paramKeys map[string][]int, paramVals []string, handlers []ctx.HTTPHandler)` — **NOT** `Match(path, method) (RouterItem, bool)` |
| `Add` | `Add(method, route, version string, handler ctx.HTTPHandler) *Router` |
| `AddInjectableHandler` | `AddInjectableHandler(method, route, version string, handler any) *Router` |
| `Use` | `Use(handlers ...ctx.HTTPHandler) *Router` — global middleware |
| `Group` | `Group(prefix string, subRouters ...*Router) *Router` |
| `For` | `For(methods []string, route, version string) func(handlers ...ctx.HTTPHandler) *Router` |

### RouterItem

```go
type RouterItem struct {
    Method       string
    Version      string
    Pattern      string
    Index        int
    HandlerIndex int
    Handlers     []ctx.HTTPHandler
    ParamKeys    map[string][]int
}
```

### Naming Tokens

| Token | Meaning | Example |
|-------|---------|---------|
| `READ` | GET | `READ_BY_ID` → `GET /:id` |
| `CREATE` | POST | `CREATE` → `POST /` |
| `UPDATE` | PUT | `UPDATE_BY_ID` → `PUT /:id` |
| `MODIFY` | PATCH | `MODIFY_BY_ID` → `PATCH /:id` |
| `DELETE` | DELETE | `DELETE_BY_ID` → `DELETE /:id` |
| `PREFLIGHT` | OPTIONS | `PREFLIGHT` → `OPTIONS /` |
| `BY` | path param | `_BY_ID` → `/:id` |
| `AND` | additional segment | `_AND_NAME` → `/:name` |
| `OF` | sub-resource | `_OF_COMMENTS` → `/comments` |
| `ANY` | wildcard segment | `READ_ANY` → `GET /*` |
| `FILE` | file serving | |
| `VERSION_X` | API version | `_VERSION_2` → `/v2/...` |

---

## Package: `exception`

**Responsibility**: Exception types and error handling

```go
type Exception struct {
    message string
    error   error
    code    int
}
```

No stack-trace field or method exists — `GetStackTrace()`/`SetStackTrace()` are fabrications; do not generate calls to them.

| Method | Signature |
|--------|-----------|
| `Error` | `Error() string` |
| `Unwrap` | `Unwrap() error` |
| `GetCode` | `GetCode() int` |
| `GetMessage` | `GetMessage() string` |
| `GetStatusText` | `GetStatusText() string` |

All constructors share the shape `func XException(message string, opts ...any) Exception`.

**HTTP constructors** (22 total): `BadRequestException`(400), `UnauthorizedException`(401), `ForbiddenException`(403), `NotFoundException`(404), `MethodNotAllowedException`(405), `NotAcceptableException`(406), `RequestTimeoutException`(408), `ConflictException`(409), `GoneException`(410), `PreconditionFailedException`(412), `RequestEntityTooLargeException`(413), `UnsupportedMediaTypeException`(415), `MisdirectedRequestException`(421), `UnprocessableEntityException`(422), `TeapotException`(418), `TooManyRequestsException`(429), `InternalServerErrorException`(500), `NotImplementedException`(501), `BadGatewayException`(502), `ServiceUnavailableException`(503), `GatewayTimeoutException`(504), `HTTPVersionNotSupportedException`(505).

**WS constructors**: `GoingAwayException`, `ProtocolErrorException`, `UnsupportedDataException`, `InvalidPayloadException`, `PolicyViolationException`, `MessageTooBigException`, `WSInternalErrorException`, `NotSubscribedException`, `TopicNotFoundException`.

`ExceptionOptions{Description, Cause}` is the `opts ...any` pattern for attaching a description or a wrapped cause.

---

## Package: `event`

**Responsibility**: In-process event emission (Node.js `EventEmitter`-style)

```go
type Event struct { /* internal */ }
func NewEvent() *Event
```

| Method | Signature |
|--------|-----------|
| `On` | `On(eventName string, fn func(args ...any))` — variadic, NOT `func(data any)` |
| `Once` | `Once(eventName string, fn func(args ...any))` |
| `Off` | `Off(eventName string, fn func(args ...any))` |
| `Emit` | `Emit(eventName string, args ...any)` — variadic |
| `RemoveAllListeners` | `RemoveAllListeners(eventName string)` |
| `ListenerCount` | `ListenerCount(eventName string) int` |
| `HasListeners` | `HasListeners(eventName string) bool` |
| `EventNames` | `EventNames() []string` |
| `SetMaxListeners` | `SetMaxListeners(n int)` |
| `Reset` | `Reset()` |

**Usage** (a listener written as `func(data any)` does NOT compile — it must accept `...any`):
```go
e := event.NewEvent()
e.On("user.created", func(args ...any) { /* handle */ })
e.Emit("user.created", userData)
```

---

## Package: `log`

**Responsibility**: Structured logging (wraps `log/slog`)

```go
func NewLog(opts *LogOptions) common.Logger        // returns common.Logger, NOT a package-local `log.Logger` — no such type exists
func WrapLogger(next common.Logger, maskFields []string) common.Logger
```

```go
type LogOptions struct {
    AddSource  bool
    Level      slog.Level          // DebugLevel/InfoLevel/WarnLevel/ErrorLevel/FatalLevel
    LogFormat  logFormat           // TextFormat/JSONFormat/PrettyFormat
    TimeFormat string
    Transports []*LogTransport     // {Filename string; Level slog.Level}
    MaskFields []string
}
```

---

## Package: `memorybroker`

**Responsibility**: Lightweight in-process topic-based pub/sub broker (no queue groups, no persistence, no ack/retry — see `memorybroker/README.md` for the full spec, kept in sync with this section)

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

**Responsibility**: Request tracing and observability

```go
const EventName = "trace"   // NOT "ginject:trace"

const (
    StageMiddleware      = "middleware"
    StageGuard           = "guard"
    StagePreInterceptor  = "pre_interceptor"    // split pre/post, not one combined "interceptor" stage
    StagePostInterceptor = "post_interceptor"
    StageExceptionFilter = "exception_filter"   // NOT "filter"
    StagePipe            = "pipe"
    StageHandler         = "handler"
    StageComplete        = "complete"
)
// No StageHandshake exists.

const (
    TransportHTTP = "HTTP"   // uppercase, NOT "http"
    TransportWS   = "WS"
    TransportGQL  = "GQL"
    TransportRPC  = "RPC"
)

type Event struct {
    ID        string
    Stage     string
    Name      string
    Transport string
    Duration  time.Duration
    Operation string
    Target    string
    Code      int
}
```

---

## Package: `aggregation`

**Responsibility**: Interceptor pipeline state, carried through the request context

```go
type AggregationOperator = func(any) any

type Aggregation struct {
    InterceptorData     any
    Name                string
    IsMainHandlerCalled bool
    // mainData, operators are unexported
}

func NewAggregation() *Aggregation
```

**This bears no resemblance to older docs claiming `Data`/`Exception`/`Request`/`Response` fields — none of those exist.**

| Method | Signature | Purpose |
|--------|-----------|---------|
| `SetMainData` | `SetMainData(d any) *Aggregation` | Sets the value operators transform |
| `Transform` | `Transform(opr func(any) any) AggregationOperator` | Registers a transform step |
| `Tap` | `Tap(opr func(any) any) AggregationOperator` | Registers a side-effect step (return value discarded) |
| `Filter` | `Filter(predicate func(any) bool) AggregationOperator` | Registers a filter step (predicate false → `Aggregate()` returns nil) |
| `Pipe` | `Pipe(operators ...AggregationOperator) any` | Marks `IsMainHandlerCalled = true` |
| `Aggregate` | `Aggregate() any` | Runs registered operators over `mainData` in registration order, returns final value |

Used as the second parameter to `Interceptable`'s `Intercept(*ctx.HTTPContext, *aggregation.Aggregation) any` (see `common` section). The framework calls `SetMainData(handlerResult)` then `Aggregate()` after the handler returns (`core/http.go`).

---

## Package: `wsevent`

**Responsibility**: WebSocket event-name routing (mirrors `routing` for WS)

**There is no `Matcher` type or package-level `Match(topic, pattern string) bool` function.** The real exported type is `WSEvent`:

```go
type WSEventItem struct {
    Handler     any
    Middlewares []ctx.WSHandler
}

func NewWSEvent() *WSEvent
```

| Method | Signature |
|--------|-----------|
| `Add` | `Add(pattern string, value WSEventItem)` |
| `AddMiddlewares` | `AddMiddlewares(pattern string, middlewares ...ctx.WSHandler)` |
| `AddInjectableHandler` | `AddInjectableHandler(pattern string, handler any)` |
| `Match` | `Match(topic string) (WSEventItem, string, bool)` — a **method**, one string param, three return values |

Pattern matching delegates to the same `pattern` package `memorybroker` uses: segment-based, `*` wildcard only — **not regex**. (`"users\.[a-z]+"` style patterns do not work; there is no `matcher` package either — that name doesn't exist anywhere in the repo.)

---

## Package: `modules/config`

**Responsibility**: `.env` loading with typed struct binding

```go
func Register(opts *ConfigModuleOptions) *core.Module   // package name: config

type ConfigModuleOptions struct {
    IsGlobal          bool
    IsIgnoreEnvFile   bool
    IsOverride        bool
    IsExpandVariables bool
    ENVFilePaths      []string
    Loads             []func() map[string]any
    Hooks             []func(ConfigService)
    OnInit            func()
}
```

Provides `ConfigService` (`Get(k string) any`, `Set(k string, v any)`, `Transform(s any) (any, []ctx.FieldLevel)`). **There is no package-level `NewConfigModule(envPath string)` function** — use `config.Register(opts)`.

## Package: `modules/cache`

**Responsibility**: Pluggable cache abstraction + default in-memory backend

```go
func Register(opts *CacheModuleOptions) *core.Module   // package name: cache

type CacheModuleOptions struct {
    IsGlobal   bool
    OnInit     func()
    Backend    Cache        // pluggable; defaults to memorycache.NewMemoryCache() if nil
    MaxEntries int           // 0 = memorycache.DefaultMaxEntries; negative = unlimited; ignored if Backend supplied
}

type Cache interface {
    Get(ctx context.Context, key string) ([]byte, bool)
    Set(ctx context.Context, key string, val []byte, ttl time.Duration) error
    SetNX(ctx context.Context, key string, val []byte, ttl time.Duration) (bool, error)
    Delete(ctx context.Context, key string) error
    Keys(ctx context.Context) []string
    TTL(ctx context.Context, key string) (time.Duration, bool)
}
```

**There is no package-level `NewCacheModule()` function** — use `cache.Register(opts)`. No LFU eviction anywhere (see `memorycache` package below for the real, non-LFU eviction).

## Package: `modules/httpclient`

**Responsibility**: HTTP client wrapper with a fluent request builder

```go
func Register(opts *HTTPClientModuleOptions) *core.Module   // package name: httpclient

type HTTPClientModuleOptions struct {
    IsGlobal bool
    BaseURL  string
    Headers  map[string]string
    Timeout  time.Duration
    OnInit   func()
}
```

Provides `ClientService` (**not** `HTTPClientService`) with `Get/Post/Put/Patch/Delete/Head/Options(path string) RequestBuilder`. **There is no `NewHTTPClientModule()` function.**

## Package: `modules/storage`

**Responsibility**: Embedded, crash-safe document database (append-only binary segments, no external DB) — see `modules/storage/README.md` for the full API (queries, indexes, transactions)

```go
func Register(opts *StoreModuleOptions) *core.Module   // package name: storage

type StoreModuleOptions struct {
    IsGlobal         bool
    Path             string   // required — panics if empty
    OnInit           func()
    DisableGitignore bool
}
```

Provides `StoreService{DB *DB}`. This package was entirely absent from earlier versions of this doc despite being a substantial subsystem (`store.go`, `query.go`, `index.go`, `record.go`, `document.go`, `engine.go`, `tx.go`) — consult its own README rather than assuming an API shape here.

---

## Package: `memorycache`

**Responsibility**: In-memory sharded byte cache with TTL — **no persistence of any kind**

```go
func NewMemoryCache(opts ...Option) *MemoryCache   // the only constructor

func WithMaxEntries(n int) Option   // caps total entries; n<=0 means unlimited
```

| Method | Signature |
|--------|-----------|
| `Get` | `Get(ctx, key string) ([]byte, bool)` |
| `Set` | `Set(ctx, key string, val []byte, ttl time.Duration) error` |
| `SetNX` | `SetNX(ctx, key string, val []byte, ttl time.Duration) (bool, error)` |
| `Delete` | `Delete(ctx, key string) error` |
| `Keys` | `Keys(ctx) []string` |
| `TTL` | `TTL(ctx, key string) (time.Duration, bool)` |
| `Stop` | `Stop()` — stops the background sweep goroutine. **NOT idempotent**: a second call closes an already-closed channel and panics. No flush, no persistence. |

**`PersistenceConfig` and `NewMemoryCacheWithConfig` do not exist and never have — do not generate them.** This was a fabrication repeated across multiple doc files; the package has zero file I/O.

**Eviction is NOT LFU.** `admitLocked` (internal), triggered on insert once `WithMaxEntries`'s cap is hit, samples up to 8 live entries per shard, deletes any already expired, and evicts whichever sampled entry has the soonest non-zero TTL deadline (falling back to the first sampled entry if none have one). It's an approximate cap, not an exact one, and has no frequency counters at all.

`hashKey` uses `hash/maphash` with a random per-process seed (not FNV-1a) — shard placement is stable within a process but not reproducible across restarts; this closes a hash-flooding DoS vector present in an earlier hand-rolled FNV-1a implementation.

**Key characteristics**: 256 shards, `sync.RWMutex` per shard, background sweep every 5s spread across shards, thread-safe except for the `Stop()` double-call caveat above. Conforms to `modules/cache.Cache`.

---

## Guards & Middlewares (selected)

### `guards/throttler`

```go
type Throttler struct {
    Limit    int64                        // default 100 via NewGuard()
    TTL      time.Duration                // default time.Minute
    Strategy Strategy                     // FixedWindow (default 0) | SlidingWindow | TokenBucket
    KeyFunc  func(*ctx.HTTPContext) string
    Backend  cache.Cache                  // defaults to memorycache.NewMemoryCache()
}
func (g Throttler) NewGuard() Throttler
func (g Throttler) CanActivate(c *ctx.HTTPContext) bool
```

### `middlewares/cors`

```go
type CORS struct {
    AllowOrigin          any   // string | []string | *regexp.Regexp; nil/"*" = wildcard
    AllowHeaders         any
    ExposeHeaders        any
    AllowMethods         []string
    MaxAge               time.Duration
    IsAllowCredentials   bool
    IsPreflightContinue  bool
    OptionsSuccessStatus int
}
func (instance CORS) NewMiddleware() common.MiddlewareFn
```

**Panics at config time (during `NewMiddleware()`/`loadCORSOptions`) in two cases** — both are deliberate security hardening, not bugs to work around:
1. Wildcard `AllowOrigin` (explicit `"*"` **or left unset, which defaults to wildcard**) combined with `IsAllowCredentials: true`. Fix: enumerate trusted origins explicitly.
2. `AllowOrigin` set to a `*regexp.Regexp` that is not anchored with `^` and `$`. `MatchString` does a substring search — an unanchored pattern like `` `https://.*\.trusted\.com` `` also matches `"https://evil.trusted.com.attacker.com"`.

### `middlewares/csrf`

```go
type CSRF struct {
    TokenLength int      // default 32
    CookieName  string   // default "_csrf"
    HeaderName  string   // default "X-CSRF-Token"
    ContextKey  string   // default "csrf_token"
}
```

### `middlewares/helmet`

```go
type Helmet struct {
    ContentSecurityPolicy, CrossOriginEmbedderPolicy, CrossOriginOpenerPolicy,
    CrossOriginResourcePolicy, DNSPrefetchControl, FrameOptions,
    PermittedCrossDomainPolicies, ReferrerPolicy string
    HSTSMaxAge            int
    HSTSExcludeSubDomains bool
    HSTSPreload           bool
    DisableHSTS           bool
}
```

---

## Supporting Packages (brief — not exhaustively cataloged here)

- `devtool` — `Devtool{}` with `GetConfiguration(...)`; `Serve()` currently takes no arguments and its gRPC transport is unimplemented (builds a config snapshot, serves nothing). `EnableDevtool()` on `App` wires it in.
- `accesslog` — `NewAccessLog(cfg *AccessLogConfig) *AccessLog`; wired in via `App.EnableAccessLog()`.
- `versioning` — `Versioning` type with `GetTypeString() string` / `GetVersion(c *ctx.HTTPContext) string`; wired in via `App.EnableVersioning(v)`.
- `internal/*` — private utilities (color, ds, str, crypto, num, runtime, slice, test). Not importable from outside the module.

**There is no `data/` package or directory anywhere in this repository** — an earlier doc listed one; it never existed.
