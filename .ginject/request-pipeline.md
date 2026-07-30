# Request Pipeline: Middleware, Guards, Interceptors, Filters

**Optimization**: Execution order diagrams, truth tables for stage responsibilities, decision trees.

## 1. Pipeline Execution Order (HTTP)

### 1.1 Sequential Pipeline

**CRITICAL: Exact Execution Sequence**:
```
1. [Global Middlewares]
   ├─ Can call next() to continue
   ├─ Can terminate early
   └─ Executed in registration order
   
2. [Module Middlewares]
   ├─ Scoped to this route's module
   ├─ Can call next() to continue
   ├─ Executed in registration order
   └─ Skip if no module middlewares
   
3. [Guards]
   ├─ CanActivate() returns bool
   ├─ True = continue
   ├─ False = 403 Forbidden (early termination)
   ├─ Panic = NOT recoverable (propagates)
   └─ Executed in registration order
   
4. [Interceptors - PRE]
   ├─ Intercept() called BEFORE handler
   ├─ Can transform request context
   ├─ Panic = NOT recoverable (propagates)
   └─ Executed in registration order
   
5. [Handler Execution]
   ├─ Actual business logic
   ├─ Panics are RECOVERED
   ├─ Return value becomes response
   └─ Single handler only
   
6. [Interceptors - POST]
   ├─ Intercept() called AFTER handler
   ├─ Receives aggregation.Aggregation
   ├─ Can transform response
   ├─ Panic = NOT recoverable
   └─ Executed in REVERSE order (LIFO)
   
7. [Exception Filters]
   ├─ Called if handler panicked
   ├─ ONLY recovery point for panics
   ├─ Catch() writes error response
   ├─ Executed in REVERSE order (LIFO)
   └─ Filter can abort (write response)
```

### 1.2 Termination Points

| Layer | Can Terminate? | How | Code |
|-------|----------------|-----|------|
| Middleware | YES | Skip next() | N/A |
| Guard | YES | Return false | 403 |
| Interceptor | NO | Panic (propagates) | N/A |
| Handler | YES | Return error | 5xx |
| Exception Filter | YES | Write response | 4xx/5xx |

---

## 2. Middleware Layer

### 2.1 Middleware Interface

```go
type MiddlewareFn func(*http.Request, http.ResponseWriter, ctx.Next)

type Next = func()
```

**Responsibilities**:
- Inspect request
- Transform request (set headers, etc.)
- Call `next()` to continue chain
- Can terminate without calling next()
- Can transform response after next() returns

### 2.2 Middleware Registration

**Global** (applies to all routes):
```go
app.BindGlobalMiddlewares(func(r *http.Request, w http.ResponseWriter, next ctx.Next) {
    // Runs for every request
    next()
})
```

**Module-Scoped** (applies to this module's routes):
```go
type UserController struct {
    common.REST
    common.Middleware
}

func (c UserController) NewController() Controller {
    c.BindMiddleware(func(r *http.Request, w http.ResponseWriter, next ctx.Next) {
        // Runs only for UserController routes
        next()
    }, c.READ_BY_ID)  // Can specify specific handlers
    return c
}
```

### 2.3 Middleware Execution Order

**For single request**:
1. Global middlewares (order: first registered → last registered)
2. Module middlewares (order: first registered → last registered)
3. Handler
4. Post-handler processing
5. Response sent

**If middleware doesn't call next()**:
- Chain stops
- Handler is NOT called
- Control returns to middleware that called next()
- Response can still be written

---

## 3. Guard Layer

### 3.1 Guard Interface

```go
type Guarder interface {
    CanActivate(*ctx.HTTPContext) bool
}
```

**Responsibilities**:
- Inspect request (auth, permissions, etc.)
- Return bool: true = continue, false = forbid
- Cannot throw exceptions (panic is NOT recoverable)
- Cannot modify request (use middleware instead)

### 3.2 Guard Execution Rules

**Decision Tree**:
```
Guard.CanActivate(httpCtx) → bool

if true:
    Continue to interceptors
    
if false:
    Write 403 Forbidden response
    Skip handler + interceptors
    Skip exception filters
    End pipeline
    
if panic:
    NOT recoverable
    Propagate panic (unhandled)
```

### 3.3 Guard Registration

**Route-Specific**:
```go
type UserController struct {
    common.REST
    common.Guard
}

func (c UserController) NewController() Controller {
    c.BindGuard(func(httpCtx *ctx.HTTPContext) bool {
        // Check auth, permissions, etc.
        return isAuthorized(httpCtx)
    }, c.READ_BY_ID)  // Protect specific handler
    return c
}
```

**Common Guards**:
- `AuthGuard`: Check JWT token, user identity
- `RoleGuard`: Check user roles/permissions
- `RateLimitGuard`: Check rate limit
- `IpWhitelistGuard`: Check source IP

---

## 4. Interceptor Layer

### 4.1 Interceptor Interface

```go
type Interceptable interface {
    Intercept(*ctx.HTTPContext, *aggregation.Aggregation) any
}
```

**Responsibilities**:
- Pre-handler: Modify request context
- Post-handler: Access response + aggregation
- Return transformed data or pass-through

### 4.2 Intercept() Execution

**Pre-Handler Phase**:
```
Intercept(httpCtx, aggregation) called
    ├─ httpCtx.Code = 0 (handler not run yet)
    ├─ aggregation.Data = nil
    ├─ Can modify httpCtx for handler
    └─ Return value ignored (pre-phase)
```

**Post-Handler Phase**:
```
Intercept(httpCtx, aggregation) called
    ├─ httpCtx.Code = 200 (or whatever handler set)
    ├─ aggregation.Data = handler return value
    ├─ aggregation.Exception = null (if no panic)
    ├─ Can transform aggregation.Data
    └─ Return value replaces handler response
```

### 4.3 Aggregation Object

```go
type Aggregation struct {
    Data      any          // Handler return value
    Exception *exception.Exception  // If handler panicked
    Request   *http.Request
    Response  http.ResponseWriter
}
```

### 4.4 Interceptor Registration

```go
type UserController struct {
    common.REST
    common.Interceptor
}

func (c UserController) NewController() Controller {
    c.BindInterceptor(func(httpCtx *ctx.HTTPContext, agg *aggregation.Aggregation) any {
        // Pre-handler phase
        if agg.Data == nil {
            // Transform request
            return nil
        }
        
        // Post-handler phase
        if agg.Exception != nil {
            // Log exception
            return nil
        }
        
        // Transform response
        return map[string]any{
            "data": agg.Data,
            "timestamp": time.Now(),
        }
    }, c.READ_BY_ID)
    return c
}
```

### 4.5 Interceptor Execution Order

**Pre-Handler**:
- Executed in registration order (first registered → last registered)

**Post-Handler**:
- Executed in REVERSE order (last registered → first executed)
- Similar to middleware call stack unwinding

---

## 5. Exception Filter Layer

### 5.1 Exception Filter Interface

```go
type ExceptionFilterable interface {
    Catch(*exception.Exception, *ctx.HTTPContext)
}
```

**Responsibilities**:
- ONLY place that can recover panics
- Handle exception and write response
- Access full request context

### 5.2 Exception Filter Execution

**When Triggered**:
```
Handler or middleware or interceptor panics
    ↓
Panic recovered by invokeHTTPHandlerByProviders()
    ↓
common.NormalizeRecovered(panicValue) → *exception.Exception
    ↓
Exception filter chain called
    ↓
Filter.Catch(exception, httpCtx) → writes response
```

### 5.3 Filter Execution Order

**LIFO (Last-In-First-Out)**:
- Filters registered last are called first
- First filter that handles exception wins
- Multiple filters per route = stack behavior

**Example**:
```go
c.BindExceptionFilter(validationFilter, c.READ)    // Registered 1st → called 2nd
c.BindExceptionFilter(authFilter, c.READ)          // Registered 2nd → called 1st

// On exception: authFilter.Catch() → if not handled → validationFilter.Catch()
```

### 5.4 Filter Registration

**Global Filter** (all routes):
```go
app.BindGlobalExceptionFilters(globalHTTPExceptionFilter{})
```

**Route-Specific Filter**:
```go
type UserController struct {
    common.REST
    common.ExceptionFilter
}

func (c UserController) NewController() Controller {
    c.BindExceptionFilter(func(ex *exception.Exception, httpCtx *ctx.HTTPContext) {
        // Handle exception
        httpCtx.Status(ex.GetCode())
        httpCtx.JSON(map[string]any{
            "error": ex.GetMessage(),
            "trace": ex.GetStackTrace(),
        })
    }, c.READ_BY_ID)
    return c
}
```

### 5.5 Default Exception Filter

**Global Default** (always registered):
```go
type globalHTTPExceptionFilter struct {}

func (f globalHTTPExceptionFilter) Catch(ex *exception.Exception, c *ctx.HTTPContext) {
    // If no custom filter handles exception, this catches it
    statusCode, statusText := ex.GetHTTPStatus()
    c.JSON(statusCode, map[string]any{
        "code":    statusCode,
        "error":   statusText,
        "message": ex.GetMessage(),
    })
}
```

---

## 6. Pipeline Interactions

### 6.1 Cross-Layer Communication

**How layers talk to each other**:
- Middleware → Guard: Via httpCtx (modified request)
- Guard → Interceptor: Via httpCtx (if guard returns true)
- Interceptor → Handler: Via httpCtx (modified context)
- Handler → Interceptor: Via return value (in aggregation)
- Interceptor → Filter: Via exception in aggregation
- Filter → Response: Directly writes response

### 6.2 Context Mutations

**Safe to mutate in**:
- Middleware: httpCtx fields
- Guard: httpCtx (read-only recommended)
- Interceptor (pre): httpCtx fields
- Interceptor (post): aggregation.Data

**NOT safe to mutate in**:
- Exception filter: httpCtx already set

**Rule**: Never mutate httpCtx after guard stage

---

## 7. Error & Recovery Rules

### 7.1 What Happens on Panic?

| Layer | Panic Handling | Result |
|-------|----------------|--------|
| Middleware | Recovered | Exception filter called |
| Guard | NOT recovered | Propagates (unhandled) |
| Interceptor (pre) | NOT recovered | Propagates |
| Handler | Recovered | Exception filter called |
| Interceptor (post) | NOT recovered | Propagates |
| Exception Filter | NOT recovered | Unhandled panic |

### 7.2 Guard Returning False

```
Guard.CanActivate() returns false
    ↓
Handler NOT called
    ↓
Interceptor (post) NOT called
    ↓
Exception filter NOT called
    ↓
Status 403 Forbidden written
```

### 7.3 Timeout Handling

```
c.SetDeadline(5 * time.Second)
    ↓
Before handler: c.IsDeadlineExceeded() checked
    ↓
If true: panic(RequestTimeoutException())
    ↓
Exception filter handles: Status 408 Request Timeout
```

---

## 8. Performance Considerations

### 8.1 Pipeline Overhead

**Costs**:
- Each middleware: 10-50 μs
- Each guard: 5-20 μs
- Each interceptor: 20-100 μs
- Exception filter: 100-500 μs (if exception occurs)

**Example** (4 middlewares + 2 guards + 2 interceptors + handler):
- 4 × 20 μs = 80 μs (middleware)
- 2 × 10 μs = 20 μs (guard)
- 2 × 50 μs = 100 μs (interceptor)
- Total overhead: ~200 μs + handler time

### 8.2 Optimization Tips

1. **Minimize middleware count**: Move non-critical logic out
2. **Cache guard results**: Don't recalculate per request
3. **Defer heavy work**: Use goroutines for non-critical tasks
4. **Profile interceptors**: Check if post-processing is needed

---

## 9. Common Pipeline Patterns

### 9.1 Auth Flow

```
Middleware (CORS, logging)
    ↓
Guard (check JWT token)
    ↓ (if guard returns true)
Interceptor (enrich context with user)
    ↓
Handler (use user from context)
    ↓
Interceptor (transform response)
    ↓
Response sent
```

### 9.2 Error Handling Flow

```
Handler panics with BadRequestException
    ↓
Panic recovered
    ↓
Exception filter catches
    ↓
Filter writes 400 Bad Request + error details
    ↓
Response sent
```

### 9.3 Rate Limiting Flow

```
Middleware (no-op)
    ↓
Guard (check rate limit)
    ↓
If exceeded: return false → 403 Forbidden
    ↓
If allowed: continue
    ↓
Handler executes
```

