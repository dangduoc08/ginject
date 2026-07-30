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
    broker broker.Broker
    Logger common.Logger
    // ... other fields
}
```

**Public Methods**:

| Method | Purpose | When To Use |
|--------|---------|------------|
| `New() *App` | Create app instance | Before Create() |
| `Create(m *Module)` | Initialize app with module tree | Once at startup |
| `Listen(port int)` | Start HTTP server | Run app |
| `ServeHTTP(w, r)` | HTTP request handler | Auto-called by Go |
| `BindGlobalMiddlewares(fn ...)` | Register global middleware | Before Create() |
| `BindGlobalGuards(fn ...)` | Register global guards | Before Create() |
| `BindGlobalInterceptors(fn ...)` | Register global interceptors | Before Create() |
| `BindGlobalExceptionFilters(fn ...)` | Register global exception filters | Before Create() |
| `EnableWS()` | Enable WebSocket support | Before Create() |
| `EnableDevtool()` | Enable development tools | Before Create() |
| `EnableAccessLog()` | Enable access logging | Before Create() |
| `SetMaxBodySize(bytes)` | Set max request body size | Before Create() |

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

```go
type ModuleBuilder struct {
    // Internal fields
}
```

**Methods** (all return *ModuleBuilder for chaining):

| Method | Purpose |
|--------|---------|
| `Controllers(...Controller) *ModuleBuilder` | Register controllers |
| `Providers(...Provider) *ModuleBuilder` | Register providers |
| `Imports(...*Module) *ModuleBuilder` | Import modules |
| `Export(...string) *ModuleBuilder` | Export provider names |
| `IsGlobal(bool) *ModuleBuilder` | Make module global |
| `Build() *Module` | Build module instance |

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

## Package: `broker`

**Responsibility**: Pub/sub message broker

### Broker

```go
type Broker interface {
    Subscribe(topic string, fn func(data any)) void
    Publish(topic string, data any) error
    Unsubscribe(topic string, fn func(data any)) void
}
```

**Usage**:
```go
broker.Subscribe("users.*", func(data any) {
    // Called when published to topic matching "users.*"
})

broker.Publish("users.created", userData)
```

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
func NewCacheModule() *core.Module
```

**Provides**: `CacheService` (in-memory LFU cache)

### HTTPClient Module

```go
func NewHTTPClientModule() *core.Module
```

**Provides**: `HTTPClientService` (wraps http.Client)

---

## Package: `memcache`

**Responsibility**: In-memory LFU cache

### MemoryCache

```go
type MemoryCache struct {
    // Internal implementation
}

func NewMemoryCache(capacity int) *MemoryCache
func (m *MemoryCache) Get(key string) (any, bool)
func (m *MemoryCache) Set(key string, value any)
func (m *MemoryCache) Delete(key string)
func (m *MemoryCache) Clear()
```

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

