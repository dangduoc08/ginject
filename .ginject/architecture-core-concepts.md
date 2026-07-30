# Ginject Core Concepts & Architecture

**Optimization**: This document is structured for AI parsing — explicit rules, decision trees, state machines. Minimal prose.

## 1. Fundamental Design Patterns

### 1.1 Module → Controller → Provider (DI Triangle)

**Structure**:
```
Module (composition root)
├── Controllers (handler containers, embedded REST/WS)
│   └── Requests routed to handler methods
├── Providers (domain logic, injected into controllers)
└── Imports (nested modules, recursive composition)
```

**Invariants**:
- Every Module is a singleton created at `app.Create()` via `NewModule()`
- Controllers are instantiated per-request (temporary, pooled)
- Providers are registered once, referenced repeatedly
- Module tree is immutable after `app.Create()`

**Dependency Resolution**:
1. Check if parameter type is built-in (HTTPContext, Body, Query, etc.)
2. Check if type matches registered Provider (by type name)
3. Call `Provider.NewProvider()` to instantiate
4. Inject via reflection into handler signature

**Cost**: ~5-10μs per dependency (reflection overhead)

---

### 1.2 Request Pipeline (HTTP)

**Sequential Execution Order**:
```
HTTP Request arrives
        ↓
[Global Middlewares] — can call next() or terminate
        ↓
[Module Middlewares] — scoped to this route's module
        ↓
[Guards] — CanActivate() returns bool, no recovery
        ↓
[Interceptors (pre)] — transform request
        ↓
[Handler Execution] — main business logic (recovers panics)
        ↓
[Interceptors (post)] — transform response + aggregation
        ↓
[Exception Filters] — ONLY place that can recover panics
        ↓
HTTP Response written
```

**Key Rules**:
- Only middleware can call `next()` to continue chain
- Guards must return bool; false = early termination (not an exception)
- Interceptors run pre and post handler, but post is aggregation-aware
- Exception filters are the panic recovery boundary — panics outside them propagate
- If any layer panics before guard/interceptor, it is recovered by globalHTTPExceptionFilter

---

### 1.3 WebSocket Event Dispatch

**Connection Lifecycle**:
```
HTTP upgrade request
        ↓
[Handshake middlewares run]
        ↓
websocket.Conn accepted + registered
        ↓
readLoop() spawned (blocks on conn.Receive())
writeLoop() spawned (drains send channel)
        ↓
Client sends JSON: { type: "publish", topic: "users.created", message: {...} }
        ↓
[Topic pattern matching: exact → suffix-wildcard → complex regex]
        ↓
[Handler lookup]
        ↓
[Same pipeline as HTTP: middleware → guard → interceptor]
        ↓
[Handler execution → broker.Publish(topic)]
        ↓
[Fanout: all subscribers get conn.TrySend() — non-blocking]
        ↓
[writeLoop() sends to wire when ready]
        ↓
Connection closes → readLoop exits → writeLoop exits
```

**Key Rules**:
- WebSocket is NOT streaming; it's RPC-over-WS with pub/sub
- Each message is a complete request/response cycle
- Fanout is fire-and-forget (callbacks are non-blocking)
- Send channel buffer is fixed (32 messages) — overflow silently drops
- No message ordering guarantees across connections

---

### 1.4 Dependency Injection via Reflection

**Resolution Algorithm**:
```
For each parameter in handler(Param1, Param2, ..., ParamN):
    type := typeof(Param)
    
    if type in BuiltInDependencies:
        resolve from HTTPContext/WSContext directly
        return
    
    providerKey := genProviderKey(type)
    if providerKey in injectedProviders:
        provider := injectedProviders[providerKey]
        instance := provider.NewProvider()
        return instance
    
    panic("dependency not found")
```

**Built-in HTTP Dependencies**:
- `*ctx.HTTPContext` — full request context
- `*http.Request` — raw request object
- `http.ResponseWriter` — raw response writer
- `ctx.Body` — parsed request body
- `ctx.Query` — URL query parameters
- `ctx.Header` — HTTP headers
- `ctx.Param` — path parameters (from `:id` patterns)
- `ctx.Form` — form-encoded data
- `ctx.File` — file uploads
- `ctx.Next` — function to call next middleware
- `ctx.Redirect` — function to redirect response
- `common.Publisher` — message publisher (global)
- Pipeable types (Body, Query, Header, Param, Form, File with custom transforms)

**Built-in WS Dependencies**:
- `*ctx.WSContext` — WebSocket connection context
- `*websocket.Conn` — raw connection
- `ctx.WSPayload` — parsed incoming message
- `ctx.Next` — function to call next middleware
- `common.Publisher` — message publisher
- Pipeable types (WSPayload with custom transforms)

**Critical Rules**:
- Parameter types MUST be exported structs or interfaces
- All fields in parameter types MUST be exported (for reflection to work)
- Circular dependencies → runtime panic (no static analysis)
- Unknown parameter types → immediate panic (fail-fast)

---

## 2. Lifecycle & Initialization Order

### 2.1 App.Create() Initialization Sequence

**CRITICAL**: Order is sacred and cannot be reordered:

```
1. initLogger()
   └─ Creates framework logger, wraps custom logger if provided
   
2. initProviders(module)
   └─ Walks module tree, collects all providers
   └─ Builds injectedProviders map (key=type.String(), value=Provider)
   └─ Builds HTTPMainHandlers list
   
3. initWS(injectedProviders)
   └─ IF EnableWS was called:
      ├─ Creates WS instance
      ├─ Registers WS context factories
      ├─ Sets up event routing
      └─ Sets up broker callbacks
   └─ Skipped if EnableWS not called
   
4. initMiddlewares(injectedProviders)
   └─ Resolves global middleware instances via DI
   └─ Resolves module-scoped middleware
   
5. initExceptionFilters(injectedProviders)
   └─ Resolves HTTP exception filters
   └─ Resolves WS exception filters (if WS enabled)
   
6. initGuards(injectedProviders)
   └─ Resolves global guards
   └─ Resolves route-specific guards
   
7. initInterceptors(injectedProviders)
   └─ Resolves global interceptors
   └─ Resolves route-specific interceptors
   
8. initMainHandlers()
   └─ Builds routing trie
   └─ Creates handler chains (middleware + handler)
   └─ Wires up exception filter lookup
   
9. initDevtool()
   └─ IF Devtool enabled: creates snapshot of routes/handlers
   
10. initAccessLog()
    └─ IF AccessLog enabled: sets up structured access logging
```

**Why Order Matters**:
- initProviders MUST run first (builds everything else's dependency source)
- initWS MUST run before exception filters/guards/interceptors (WS state is referenced)
- initDevtool MUST run last (needs fully-populated state)

### 2.2 Per-Request Lifecycle (HTTP)

```
1. ServeHTTP(w, r) called
   └─ app.ctxPool.Get() → *ctx.HTTPContext (pooled allocation)
   
2. c.Init(w, r) — initialize context
   └─ Sets Timestamp = now()
   └─ Sets ResponseWriter, Request
   └─ Generates request ID
   
3. Route matching: routing.Router.Match(r.URL.Path, r.Method)
   └─ Returns RouterItem or 404 Not Found
   
4. Handler resolution from RouterItem
   └─ Get handler function from registry
   └─ Get handler name, version, route
   
5. Handler chain execution:
   ├─ Global middlewares
   ├─ Module middlewares
   ├─ Guards
   ├─ Interceptors (pre)
   ├─ Handler invocation (with panic recovery)
   ├─ Interceptors (post)
   └─ Exception filters (if panic recovered)
   
6. Trace emission (if listeners registered)
   └─ Send trace.Event with timing data
   
7. Context release: app.ctxPool.Put(c)
   └─ c.Reset() — clears all fields
   └─ Returns to pool for reuse
   
8. HTTP response written to wire
```

**Performance Characteristics**:
- Context pooling: ~500ns allocation saved per request
- Route matching: O(1) exact, O(log n) prefix wildcard, O(n) complex regex
- Dependency resolution: ~1-5μs per dependency × count
- Handler execution: 1-2ms fast path (no I/O)

### 2.3 WebSocket Connection Lifecycle

```
1. HTTP upgrade request arrives
2. Handshake middleware chain runs (same pipeline as HTTP)
3. websocket.Conn accepted
4. connID (UUID) generated
5. WSConnMgr.Register(connID, conn)
   └─ Creates send channel (buffer 32)
   └─ Spawns writeLoop() goroutine
   └─ Spawns readLoop() goroutine
6. Client can now send messages
7. readLoop() reads and dispatches to handlers
8. Each message creates temporary *ctx.WSContext (pooled)
9. Handler can broker.Publish() — fanout to all subscribers
10. Each subscriber: conn.TrySend() → non-blocking send to channel
11. writeLoop() drains channel and writes to wire
12. Connection closes → readLoop exits
13. Unregister() closes done channel → writeLoop exits
14. All subscriptions cleaned up
```

---

## 3. Context & Request Scoping

### 3.1 HTTPContext Structure

**Pooled Object** (sync.Pool):
```go
type HTTPContext struct {
    *http.Request              // Raw request
    http.ResponseWriter        // Raw response writer
    
    contextID                  // Request ID (8-char UUID truncated)
    dataWriter DataWriter      // Response body writer (Text/JSON/JSONP)
    
    body Body                  // Parsed request body
    form Form                  // Form data
    file File                  // File uploads
    query Query                // Query parameters
    header Header              // Headers
    param Param                // Path parameters
    ParamKeys map[string][]int // Indices of dynamic path params
    ParamValues []string       // Values of dynamic path params
    
    Next Next                  // Middleware chain continuation
    Code int                   // HTTP status code
    Timestamp time.Time        // Request start time
    Deadline time.Time         // Request deadline (if timeout set)
}
```

**Lifecycle**:
1. Allocated from pool via `app.ctxPool.Get()`
2. Initialized via `c.Init(w, r)` or `c.InitWithMaxBodySize(w, r, limit)`
3. Used throughout request processing
4. Reset via `c.Reset()` which clears all fields
5. Returned to pool via `app.ctxPool.Put(c)` for reuse

**Operations**:
- `SetDeadline(duration)` — sets timeout for handler
- `IsDeadlineExceeded()` — checks if deadline passed
- `Status(code)` — set HTTP status code
- `Text(data)` / `JSON(data)` / `JSONP(data)` — write response
- `Redirect(url)` — HTTP redirect

### 3.2 WSContext Structure

**Similar to HTTPContext but for WebSocket**:
```go
type WSContext struct {
    *websocket.Conn            // Raw WebSocket connection
    contextID                  // Request ID
    
    wsPayload *WSPayload       // Parsed incoming message
    Next Next                  // Middleware chain continuation
    Timestamp time.Time        // Message arrival time
}
```

---

## 4. Dependency Injection Details

### 4.1 Provider Registration

**Provider Interface**:
```go
type Provider interface {
    NewProvider() Provider
}
```

**How It Works**:
1. Any struct implementing `NewProvider() Provider` is a provider
2. Register via `ModuleBuilder().Providers(MyProvider{})`
3. At startup: `app.Create()` walks module tree and collects all providers
4. Key = `type.String()` of provider (e.g., "main.UserService")
5. Stored in `injectedProviders map[string]Provider`

**Critical Rule**:
- `NewProvider()` is called fresh every time it's injected
- NOT cached as singleton (called per-injection)
- To create singletons: have static module-level variables

### 4.2 Provider Key Generation

```go
func genProviderKey(provider Provider) string {
    return reflect.TypeOf(provider).String()
}
```

**Example**:
- Provider: `UserService{}`
- Key: `"main.UserService"`
- If injected into handler: `func(c *ctx.HTTPContext, us *UserService)`
  - Parameter type: `*UserService` → `"*main.UserService"`
  - Match: NOT FOUND (key mismatch!)
  - Solution: Provider should be `*UserService` or parameter should be `UserService`

**Critical Rule**:
- Provider types MUST match exactly with handler parameter types
- Pointer vs non-pointer matters!

### 4.3 Circular Dependency Handling

**Current Behavior**: Panic at runtime (no static analysis)

**Detection**: During `app.Create()`, if Provider A needs Provider B which needs Provider A:
- Call A.NewProvider()
- A.NewProvider() tries to inject B
- B.NewProvider() tries to inject A
- Infinite recursion → stack overflow → panic

**Mitigation Strategies**:
1. Restructure dependencies (use composition)
2. Lazy-load circular dependencies (defer initialization)
3. Use a third-party package to resolve circular deps

---

## 5. Error Handling & Recovery

### 5.1 Exception Type System

**Exception Interface** (exception/exception.go):
```go
type Exception struct {
    message string       // User-facing message
    error error          // Underlying error
    code int             // HTTP status code or WS close code
    stackTrace string    // Full stack trace if captured
}

Methods:
- GetCode() int
- GetMessage() string
- GetStatusText() string (HTTP status text)
- GetStackTrace() string
- SetStackTrace(trace string)
```

**Built-in Exception Types**:
- `BadRequestException` (400)
- `UnauthorizedException` (401)
- `ForbiddenException` (403)
- `NotFoundException` (404)
- `ConflictException` (409)
- `InternalServerErrorException` (500)
- `NotImplementedException` (501)
- `ServiceUnavailableException` (503)
- WebSocket close codes (various)

### 5.2 Panic Recovery

**Recovery Points**:
1. **Handler invocation**: `invokeHTTPHandlerByProviders()` wraps handler call in recover()
2. **Middleware execution**: `buildUseMiddleware()` wraps middleware call in recover()
3. **Anywhere else**: Panic propagates to exception filter

**Recovery Flow**:
```
Handler panics with exception.BadRequestException("invalid")
        ↓
recover() catches panic value
        ↓
common.NormalizeRecovered(value) converts to *exception.Exception
  - Extracts message
  - Captures stack trace via runtime.Stack()
  - Wraps in Exception if needed
        ↓
Exception filter chain called with exception
        ↓
Filter's Catch(exception, context) handles response
        ↓
HTTP response written (or WS message sent)
```

**NormalizeRecovered() Behavior**:
- Input: `interface{}` (panic value)
- If already `*exception.Exception`: return as-is
- If `error`: wrap in Exception with error message
- If `string`: wrap in Exception with string message
- Also captures `runtime.Stack()` trace

### 5.3 Exception Filter Architecture

**Filter Interface**:
```go
type ExceptionFilterable interface {
    Catch(*exception.Exception, *ctx.HTTPContext)
}
```

**Execution Order**:
- Multiple filters can be registered per route
- Filters are called in LIFO (stack) order: last-registered → first-executed
- First filter to handle the exception wins (no chaining)

**Global vs Route-Specific**:
- Global filters: catch all exceptions across entire app
- Route-specific filters: catch exceptions only for that route
- Route-specific filters run BEFORE global filters (LIFO)

**Default Filters**:
- `globalHTTPExceptionFilter` — catches all HTTP exceptions, writes JSON response
- `globalWSExceptionException` — catches all WS exceptions, sends close message

---

## 6. Concurrency Model

### 6.1 Request Isolation

**HTTP Requests**:
- Each request gets its own pooled `*ctx.HTTPContext`
- No shared mutable state within single request
- Thread-safe because each request is independent

**WebSocket Connections**:
- Each connection gets its own `*ctx.WSContext` per message
- readLoop: single goroutine per connection (not concurrent)
- writeLoop: single goroutine per connection (not concurrent)
- Send channel: buffered (32), non-blocking `TrySend()`

### 6.2 Global Shared State

**Thread-Safe**:
- `injectedProviders map[string]Provider` — read-only after app.Create()
- `routing.Router` — read-only after app.Create()
- `sync.Pool` for context pooling — handles synchronization
- `event.Event` — internal sync.Map for listeners
- `broker.Broker` — concurrent pub/sub with sync.RWMutex

**Not Thread-Safe (by design)**:
- Module state (captured at app.Create(), immutable after)
- Handler closures capturing request state

### 6.3 Goroutine Model

**Per HTTP Request**: 1 goroutine (within ServeHTTP)

**Per WebSocket Connection**:
- readLoop: 1 goroutine (blocking on conn.Receive)
- writeLoop: 1 goroutine (blocking on send channel)
- Total: 2 goroutines per active connection

**Total Max Goroutines** = 1 (main) + N (concurrent HTTP) + 2M (M active WS connections)

---

## 7. Key Architectural Invariants

**NEVER VIOLATE THESE**:

1. **Context Lifecycle**:
   - HTTPContext must be released via `app.releaseCtx()` after use
   - Never hold onto context reference after request completes
   - Never mutate context fields after guard stage

2. **Dependency Resolution**:
   - Handler parameters MUST match registered provider types exactly
   - Circular dependencies cause runtime panics (fail-fast)
   - All injectable fields MUST be exported

3. **Pipeline Execution**:
   - Only middleware can call `next()` to continue
   - Guards must return bool; cannot throw exceptions
   - Panics outside exception filters are unrecoverable

4. **Panic Recovery**:
   - Only exception filters can recover panics
   - Panics in guards/interceptors are NOT recoverable (propagate)
   - Stack traces are captured during NormalizeRecovered()

5. **Module Initialization**:
   - Module tree is immutable after `app.Create()`
   - Providers are singletons (created once per module tree walk)
   - Controllers are instantiated per-request

6. **WebSocket**:
   - Each message is a complete request/response (not streaming)
   - Send channel buffer (32) can overflow — messages drop
   - Fanout is fire-and-forget (no error handling)

7. **Concurrency**:
   - Each HTTP request is independent (no shared mutable state)
   - Each WS connection is single-threaded (readLoop + writeLoop)
   - Global state is read-only after app.Create()

