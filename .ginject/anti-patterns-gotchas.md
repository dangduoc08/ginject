# Anti-Patterns, Gotchas, & Common Mistakes

**Purpose**: Document what NOT to do and why. This prevents AI agents from generating broken code.

## 1. Dependency Resolution Gotchas

### 1.1 GOTCHA: Type Mismatch (Provider vs Handler Parameter)

**Problem**:
```go
// Provider registered as value
type UserService struct { ... }
func (u UserService) NewProvider() Provider { return u }

// Handler expects pointer
func (c UserController) READ(us *UserService) { ... }
```

**Why It Fails**:
- Provider key: `"main.UserService"`
- Parameter type: `*main.UserService`
- Keys don't match → `panic("dependency injection: can't resolve")`

**Solution**: Make types match
```go
// Option 1: Both values
func (c UserController) READ(us UserService) { ... }

// Option 2: Both pointers (recommended)
type UserService struct { ... }
func (u *UserService) NewProvider() Provider { return u }
func (c UserController) READ(us *UserService) { ... }
```

**Rule**: Provider type must match handler parameter type EXACTLY

---

### 1.2 GOTCHA: Calling NewProvider() Manually

**WRONG**:
```go
// In handler
func (c UserController) READ() {
    us := UserService{}.NewProvider()  // Manual call!
    us.GetUser(123)  // Wrong type!
}
```

**Why It's Wrong**:
- Framework doesn't know you called it
- Defeats dependency injection
- No cleanup/lifecycle management
- Each call creates new instance (not singletons)

**RIGHT**:
```go
// Let framework inject
func (c UserController) READ(us UserService) {
    us.GetUser(123)  // Framework-injected
}
```

**Rule**: Never manually call NewProvider() — use injection instead

---

### 1.3 GOTCHA: Circular Dependencies

**Problem**:
```go
type ServiceA struct {
    B ServiceB  // A depends on B
}

type ServiceB struct {
    A ServiceA  // B depends on A (CIRCULAR!)
}

// At app.Create():
// A.NewProvider() → inject B
// B.NewProvider() → inject A
// Infinite recursion → stack overflow
```

**Why It Fails**:
- No static dependency analysis
- Detected at runtime (too late)
- Causes unrecoverable panic

**Solutions**:

**Option 1: Restructure**
```go
// Create third service
type SharedLogic struct {
    // Common functionality
}

type ServiceA struct {
    Shared SharedLogic
}

type ServiceB struct {
    Shared SharedLogic
}
```

**Option 2: Lazy Loading**
```go
type ServiceA struct {
    B *ServiceB  // Don't auto-inject
}

func (a ServiceA) GetB() ServiceB {
    if a.B == nil {
        a.B = &ServiceB{}  // Create on demand
    }
    return a.B
}
```

**Option 3: Single Responsibility**
```go
// Avoid creating service that needs both A and B
// Instead: A depends on Repo, B depends on Repo
```

**Rule**: Avoid circular dependencies — restructure instead

---

## 2. Context Lifecycle Gotchas

### 2.1 GOTCHA: Retaining Pooled Context After Request

**WRONG**:
```go
var lastCtx *ctx.HTTPContext

func (c UserController) READ(httpCtx *ctx.HTTPContext) {
    lastCtx = httpCtx  // WRONG! Retain pooled object
    return "ok"
}

// Later request uses same pooled context
// lastCtx still holds reference → stale data!
```

**Why It's Wrong**:
- Contexts are pooled for reuse (performance)
- Retaining context = stale reference in next request
- Race conditions: multiple requests share same object
- Memory leaks: context never returns to pool

**RIGHT**:
```go
func (c UserController) READ(httpCtx *ctx.HTTPContext) {
    // Use httpCtx only within request
    return httpCtx.GetID()  // Use locally
    // After function returns, httpCtx released to pool
}
```

**Rule**: Never hold onto context reference after handler returns

---

### 2.2 GOTCHA: Mutating Context After Guard

**WRONG**:
```go
func (c UserController) READ(httpCtx *ctx.HTTPContext) {
    // Handler runs AFTER guard
    // If guard saw certain context state, don't change it
    httpCtx.Request.Header.Set("X-User-ID", "123")  // WRONG!
    // Guard already made decision based on original headers
}
```

**Why It's Wrong**:
- Guard made authorization decision based on original request
- Mutating headers after = bypassing guard logic
- Can cause security issues

**RIGHT**:
```go
// Use middleware to set headers (before guard)
app.BindGlobalMiddlewares(func(r *http.Request, w http.ResponseWriter, next ctx.Next) {
    // Runs BEFORE guard
    r.Header.Set("X-User-ID", "123")
    next()
})

// Guard sees modified request
func (c UserController) READ(httpCtx *ctx.HTTPContext) {
    // Use context
}
```

**Rule**: Mutations must happen in middleware (before guard), not in handler

---

## 3. Handler Pipeline Gotchas

### 3.1 GOTCHA: Panic in Guard

**WRONG**:
```go
type AuthGuard struct {
    common.Guard
}

func (g AuthGuard) CanActivate(httpCtx *ctx.HTTPContext) bool {
    panic(exception.UnauthorizedException("not logged in"))  // WRONG!
}

// Panic is NOT recoverable in guard layer
// Unhandled panic → app crash
```

**Why It's Wrong**:
- Guard panics are NOT caught
- No exception filter to handle it
- Unrecoverable error
- Crashes entire request handling

**RIGHT**:
```go
// Option 1: Return false for forbidden
func (g AuthGuard) CanActivate(httpCtx *ctx.HTTPContext) bool {
    if !isAuthorized(httpCtx) {
        return false  // 403 Forbidden
    }
    return true
}

// Option 2: Throw in middleware (before guard)
app.BindGlobalMiddlewares(func(r *http.Request, w http.ResponseWriter, next ctx.Next) {
    if !isAuthorized(r) {
        panic(exception.UnauthorizedException(...))  // This IS caught
    }
    next()
})
```

**Rule**: Guards must return bool, not panic

---

### 3.2 GOTCHA: Infinite Loop in Middleware

**WRONG**:
```go
app.BindGlobalMiddlewares(func(r *http.Request, w http.ResponseWriter, next ctx.Next) {
    // Process request
    next()
    
    // WRONG: Calling next() again!
    next()  // Infinite recursion
})
```

**Why It's Wrong**:
- Calling next() multiple times = traversing chain multiple times
- Can cause infinite loops or stack overflow
- Double-processes handler

**RIGHT**:
```go
app.BindGlobalMiddlewares(func(r *http.Request, w http.ResponseWriter, next ctx.Next) {
    // Pre-processing
    startTime := time.Now()
    
    next()  // Single call
    
    // Post-processing
    duration := time.Since(startTime)
    log.Printf("Request took %v", duration)
})
```

**Rule**: Call next() exactly once per middleware

---

## 4. WebSocket Gotchas

### 4.1 GOTCHA: Message Drop on Slow Client

**PROBLEM**:
```go
broker.Publish("users.created", userData)  // Sends to all subscribers
// If subscriber's send channel buffer full (32 messages)
// Message is DROPPED silently
// No error, no exception, no notification
```

**Why It's Wrong**:
- Send channel: 32-message buffer per connection
- If client reads slowly, buffer fills
- New messages drop silently
- Client has no way to know

**Mitigation**:
```go
// Monitor connection lag
// Implement backpressure
// Implement message queue with persistence
// Implement client heartbeat + timeout

// In readLoop:
if channelFull {
    closeConnection(1008, "send buffer overflow")
}
```

**Rule**: Be aware that slow subscribers miss messages

---

### 4.2 GOTCHA: Accessing Raw Connection in Handler

**WRONG**:
```go
func (c ChatController) ON_MESSAGE(conn *websocket.Conn) {
    conn.Send(someMessage)  // Direct access to conn
    // But framework's fanout also uses conn
    // Race condition possible!
}
```

**Why It's Wrong**:
- Both handler and fanout write to connection
- No synchronization
- Can corrupt message stream

**RIGHT**:
```go
func (c ChatController) ON_MESSAGE(publisher common.Publisher) {
    publisher.Publish("topic", data)  // Safe, thread-safe
}
```

**Rule**: Use Publisher, never access raw conn directly

---

### 4.3 GOTCHA: No Message Ordering Guarantee

**PROBLEM**:
```go
// Broker publishes to 1000 subscribers
broker.Publish("event", data)
// Different clients might receive in different orders
// Not guaranteed FIFO across connections
```

**Why**:
- Broker iterates subscribers
- Each fanout callback is non-blocking TrySend()
- Order depends on scheduling

**Implication**:
- Don't assume Client A sees message before Client B
- If order matters, add sequence numbers to messages

**Rule**: Messages not ordered across connections

---

## 5. Module & Provider Gotchas

### 5.1 GOTCHA: Mutable Static Module Variables

**WRONG**:
```go
var MyModule = func() *core.Module {
    var counter = 0  // MUTABLE STATE!
    
    return core.ModuleBuilder().
        Providers(
            CounterService{counter: &counter},  // Shared mutable
        ).
        Build()
}()

// Every app.Create() reuses same counter!
```

**Why It's Wrong**:
- Module created once (static)
- Mutable state persists across app instances
- Multiple tests interfere with each other

**RIGHT**:
```go
// Option 1: Create new module each time
func NewMyModule() *core.Module {
    return core.ModuleBuilder().
        Providers(
            CounterService{counter: 0},
        ).
        Build()
}

// Option 2: Make counter immutable
var MyModule = func() *core.Module {
    return core.ModuleBuilder().
        Providers(
            CounterService{counter: 0},  // Immutable
        ).
        Build()
}()
```

**Rule**: Avoid mutable state in static modules

---

### 5.2 GOTCHA: NewProvider() Called Multiple Times

**Problem**:
```go
type DatabasePool struct {
    conn *sql.DB
}

func (d DatabasePool) NewProvider() Provider {
    // WARNING: Called EACH TIME provider is injected!
    // If not careful, creates multiple connections
    return DatabasePool{
        conn: sql.Open(...),  // NEW connection each time!
    }
}

// Every handler that injects DatabasePool gets new connection
// Connection leak!
```

**Why It's Wrong**:
- NewProvider() called per injection, not at startup
- If you create new resource each time, resource leak
- Not singletons by default

**RIGHT**:
```go
var globalDBPool *sql.DB = nil

type DatabasePool struct {
    conn *sql.DB
}

func (d DatabasePool) NewProvider() Provider {
    if globalDBPool == nil {
        globalDBPool, _ = sql.Open(...)  // Create once
    }
    return DatabasePool{conn: globalDBPool}  // Reuse
}
```

**Rule**: NewProvider() should return same instance (singleton) or create fresh without resource leak

---

## 6. Exception Handling Gotchas

### 6.1 GOTCHA: Panic Outside Exception Filter

**WRONG**:
```go
func (c UserController) READ() {
    // Exception filter can only catch panics in:
    // - Handler execution
    // - Middleware
    // NOT in guard or interceptor pre-phase
    
    // If guard panics, exception filter doesn't catch it
}
```

**Why It's Wrong**:
- Exception filters only wrap handler/middleware
- Panics in guard = unrecoverable
- Propagates as unhandled panic

**Rule**: Guards must not panic, must return bool

---

### 5.2 GOTCHA: Multiple Exception Filters Fighting

**PROBLEM**:
```go
c.BindExceptionFilter(validationFilter, c.READ)
c.BindExceptionFilter(authFilter, c.READ)

// LIFO: authFilter called first
// If authFilter doesn't handle, validationFilter called
// But both might try to write response!

// Example:
// authFilter writes 401 Unauthorized
// validationFilter tries to write 400 Bad Request
// Response headers already sent!
```

**Why It's Wrong**:
- Multiple filters can conflict
- Once first filter writes response, others can't

**RIGHT**:
```go
c.BindExceptionFilter(func(ex *exception.Exception, httpCtx *ctx.HTTPContext) {
    // Handle different exception types
    switch ex.GetCode() {
    case http.StatusUnauthorized:
        // Handle auth error
    case http.StatusBadRequest:
        // Handle validation error
    }
    // Write single response
}, c.READ)
```

**Rule**: One filter per route, or make filter handle all cases

---

## 7. Performance Gotchas

### 7.1 GOTCHA: Reflection Overhead in Hot Path

**Problem**:
```go
// Handler invocation uses reflection
// Not fast in hot path (high-traffic routes)

invokeHTTPHandlerByProviders(handler, ..., httpCtx, ...)
    // Reflects on handler signature each time
    // ~5-10μs per dependency
    // With 5 dependencies: 25-50μs per request
```

**Mitigation**:
- Minimize handler parameters
- Cache handler signatures (framework does this)
- Profile hot handlers

**Rule**: Be aware of reflection cost in high-traffic routes

---

### 7.2 GOTCHA: Synchronous Exceptions in Fast Path

**Problem**:
```go
// Exception creation allocates new objects
// In exception filter, creates response object

exception.NewException(...)
    ├─ Allocate Exception struct
    ├─ Allocate error message
    └─ Allocate stack trace string

// Every exception = allocation
// High-error-rate app = memory pressure
```

**Mitigation**:
- Minimize exceptions in hot path
- Reuse exception instances
- Use object pooling

**Rule**: Exceptions are expensive, avoid in hot path

---

## 8. Testing Gotchas

### 8.1 GOTCHA: Global State Pollution Between Tests

**WRONG**:
```go
func TestUserController(t *testing.T) {
    app := core.New()
    app.Create(MyModule)  // Global module with shared state!
    
    // Test 1: app.http.ctxPool used
    // Test 2: app.http.ctxPool reused from Test 1
    // Context state bleeding between tests!
}
```

**Why It's Wrong**:
- sync.Pool reuses objects
- Context not fully reset between tests
- Tests interfere with each other

**RIGHT**:
```go
func TestUserController(t *testing.T) {
    app := core.New()
    module := NewMyModule()  // Create NEW module each test
    app.Create(module)
    
    // No shared state pollution
}
```

**Rule**: Create new module/app instances per test

---

### 8.2 GOTCHA: Testing WebSocket Without Actual Connection

**WRONG**:
```go
// Trying to test WS handler without real connection
func TestWSHandler(t *testing.T) {
    c := ctx.NewWSContext()
    // c.Conn is nil!
    
    app.ws.handleRequest(c)  // Panics: nil pointer dereference
}
```

**Why It's Wrong**:
- WSContext needs real websocket.Conn
- Can't test without real connection or mock

**RIGHT**:
```go
// Use httptest.Server with websocket upgrade
// Or mock websocket.Conn interface
// Or test handler logic separately (use dependency injection)
```

**Rule**: WS tests need real connections or mocks

