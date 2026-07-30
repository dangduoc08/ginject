# Handler Execution & Dependency Resolution

**Optimization**: Pseudo-code, decision trees, state machines. Focus on exact execution order and invariants.

## 1. HTTP Handler Invocation

### 1.1 Invocation Flow

**Function**: `invokeHTTPHandlerByProviders(f, injectedProviders, httpCtx, event, pipeElapsed, handlerCalled)`

**Pseudo-Code**:
```
func invokeHTTPHandlerByProviders(
    f any,                           // handler function (e.g., UserController.READ_BY_ID)
    injectedProviders map[string]Provider,
    c *ctx.HTTPContext,
    ev *event.Event,
    pipeElapsed *time.Duration,
    handlerCalled *bool
) []reflect.Value {
    
    // Step 1: Reflect on handler signature
    fType := reflect.TypeOf(f)
    args := []reflect.Value{}
    
    // Step 2: For each parameter in handler signature
    for i := 0; i < fType.NumIn(); i++ {
        paramType := fType.In(i)
        dynamicArgKey := genProviderKey(paramType)
        
        // Step 3a: Check if it's a built-in dependency
        if _, isBuiltIn := knownHTTPDependencyKeys[dynamicArgKey]; isBuiltIn {
            
            // Step 3b: If it's a Pipeable with listeners, emit trace
            if isPipeable(dynamicArgKey) && ev.HasListeners(trace.EventName) {
                start := time.Now()
                dep := getHTTPDependency(dynamicArgKey, c, pipeValue)
                args.append(reflect.ValueOf(dep))
                *pipeElapsed += time.Since(start)
                ev.Emit(trace.EventName, trace.Event{...})
            } else {
                // Step 3c: Direct injection without tracing
                dep := getHTTPDependency(dynamicArgKey, c, pipeValue)
                args.append(reflect.ValueOf(dep))
            }
        } else {
            // Step 3d: Unknown parameter type
            panic(fmt.Errorf(
                "dependency injection: can't resolve dependencies. Param [%d] of type %v not available",
                i, paramType
            ))
        }
    }
    
    // Step 4: Call handler with resolved args
    *handlerCalled = true
    return reflect.ValueOf(f).Call(args)
}
```

### 1.2 Dependency Resolution: getHTTPDependency()

**Decision Tree**:
```
dynamicArgKey = genProviderKey(paramType)

if dynamicArgKey == httpContextKey:          // *ctx.HTTPContext
    return c
    
if dynamicArgKey == requestKey:              // *http.Request
    return c.Request
    
if dynamicArgKey == responseKey:             // http.ResponseWriter
    return c.ResponseWriter
    
if dynamicArgKey == bodyKey:                 // ctx.Body
    return c.body (parsed from c.Request.Body)
    
if dynamicArgKey == queryKey:                // ctx.Query
    return c.query (parsed from c.Request.URL.Query())
    
if dynamicArgKey == headerKey:               // ctx.Header
    return c.header (from c.Request.Header)
    
if dynamicArgKey == paramKey:                // ctx.Param
    return c.param (from route params)
    
if dynamicArgKey == formKey:                 // ctx.Form
    return c.form (from c.Request.Form)
    
if dynamicArgKey == fileKey:                 // ctx.File
    return c.file (from multipart upload)
    
if dynamicArgKey == nextKey:                 // ctx.Next (function)
    return c.Next (middleware chain continuation)
    
if dynamicArgKey == redirectKey:             // ctx.Redirect (function)
    return func(url string) { c.Redirect(url) }
    
if dynamicArgKey == publisherKey:            // common.Publisher
    return globalInterfaceByKey.Load(publisherKey)
    
if dynamicArgKey == "Pipeable*":             // Custom Pipeable
    return pipeValue.Interface().(Pipeable).Transform(...)
    
else:
    panic("unknown dependency")
```

### 1.3 Pipeable Dependencies (Custom Transforms)

**Pipeable Interface**:
```go
type Pipeable interface {
    Transform(raw any, metadata ArgumentMetadata) any
}
```

**Purpose**: Custom transformation of request data

**Examples**:
- `BodyPipeable`: Parse JSON/XML body into struct
- `QueryPipeable`: Validate/transform query params
- `ParamPipeable`: Convert `:id` string to int
- `HeaderPipeable`: Extract/validate specific headers

**How It Works**:
1. Handler declares parameter type that implements Pipeable
2. At invocation: `pipeValue.Interface().(Pipeable).Transform(rawData, metadata)`
3. Pipeable returns transformed/validated data
4. Handler receives transformed data

**Performance Note**: Pipeable transforms are traced separately if tracing enabled

---

## 2. WebSocket Handler Invocation

### 2.1 invokeWSHandlerByProviders()

**Nearly identical to HTTP**, but for WS dependencies:

**Pseudo-Code**:
```
func invokeWSHandlerByProviders(
    f any,
    injectedProviders map[string]Provider,
    c *ctx.WSContext,
    ev *event.Event,
    pipeElapsed *time.Duration,
    handlerCalled *bool
) []reflect.Value {
    
    fType := reflect.TypeOf(f)
    args := []reflect.Value{}
    
    for i := 0; i < fType.NumIn(); i++ {
        paramType := fType.In(i)
        dynamicArgKey := genProviderKey(paramType)
        
        if _, isBuiltIn := knownWSDependencyKeys[dynamicArgKey]; !isBuiltIn {
            panic("dependency not found")
        }
        
        dep := getWSDependency(dynamicArgKey, c, pipeValue)
        args.append(reflect.ValueOf(dep))
    }
    
    *handlerCalled = true
    return reflect.ValueOf(f).Call(args)
}
```

### 2.2 WebSocket-Specific Dependencies: getWSDependency()

**Decision Tree**:
```
if dynamicArgKey == wsContextKey:            // *ctx.WSContext
    return c
    
if dynamicArgKey == wsConnectionKey:         // *websocket.Conn
    return c.Conn
    
if dynamicArgKey == wsPayloadKey:            // ctx.WSPayload
    return c.WSPayload() (current message)
    
if dynamicArgKey == nextKey:                 // ctx.Next
    return c.Next
    
if dynamicArgKey == publisherKey:            // common.Publisher
    return globalInterfaceByKey.Load(publisherKey)
    
if dynamicArgKey == wsPayloadPipeableKey:    // Custom Pipeable
    return pipeValue.Interface().(WSPayloadPipeable).Transform(...)
    
else:
    panic("WS dependency not found")
```

**Available in WS but NOT HTTP**:
- `*ctx.WSContext`
- `*websocket.Conn`
- `ctx.WSPayload`

**NOT Available in WS**:
- Body, Query, Header, Param, Form, File (HTTP-specific)

---

## 3. Provider Instantiation

### 3.1 Provider.NewProvider() Call

**When**: Each time a provider is injected into a handler

**How**:
1. Lookup provider by type key in `injectedProviders`
2. Call `provider.NewProvider()` to get instance
3. Convert to reflect.Value
4. Append to handler args
5. Call handler with args

**Important**: Providers are NOT singletons. Each injection calls NewProvider().

### 3.2 Singleton Providers (Pattern)

**To create singleton providers**:

```go
// WRONG - creates new instance each time:
func (p UserService) NewProvider() Provider {
    return UserService{} // NEW instance every time
}

// RIGHT - return same instance:
var userServiceSingleton = UserService{repo: ...}

func (p UserService) NewProvider() Provider {
    return userServiceSingleton // Reuse instance
}
```

### 3.3 Provider with Initialization

```go
type EmailService struct {
    smtp *net.Conn
}

func (e EmailService) NewProvider() Provider {
    // Initialize connection once
    if e.smtp == nil {
        e.smtp = net.Dial("tcp", "localhost:25")
    }
    return e
}
```

---

## 4. Error Handling in Handler Invocation

### 4.1 Panic Recovery

**Point of Recovery**: `invokeHTTPHandlerByProviders()` has implicit panic recovery via defer

**Pseudo-Code**:
```
// In core/http.go or routing layer
defer func() {
    if rec := recover(); rec != nil {
        ex := common.NormalizeRecovered(rec)
        // Call exception filters with ex
    }
}()

invokeHTTPHandlerByProviders(...) // If this panics, recover catches it
```

### 4.2 Timeout Checking (Before Handler Invocation)

**Function**: Check `c.IsDeadlineExceeded()` before calling handler

**Pseudo-Code**:
```
if c.IsDeadlineExceeded() {
    panic(exception.RequestTimeoutException("request exceeded deadline"))
}

// Then call handler
invokeHTTPHandlerByProviders(...)
```

### 4.3 Common Handler Errors

| Error | Cause | Recovery |
|-------|-------|----------|
| `dependency injection: can't resolve` | Unknown parameter type | Panic → exception filter |
| `panic(exception.BadRequestException(...))` | Validation failure | Caught by exception filter |
| `return nil` from handler | No return value | OK, handler can return any type |
| Timeout exceeded | Handler takes too long | Panic with RequestTimeoutException |

---

## 5. Return Value Handling

### 5.1 Handler Return Types

**HTTP Handlers**:
- `func() string` → write as text/plain
- `func() []byte` → write as binary
- `func() map[string]any` → write as JSON
- `func() *MyStruct` → write as JSON
- `func() error` → if error, call exception filter
- `func()` (no return) → write empty 200 OK

**Return Value Processing**:
```
returnValues := reflect.ValueOf(f).Call(args)

if len(returnValues) > 0 {
    retVal := returnValues[0].Interface()
    
    if err, ok := retVal.(error); ok {
        // Handle as exception
        panic(err)
    }
    
    // Otherwise, write to response
    c.JSON(http.StatusOK, retVal)
}
```

### 5.2 WebSocket Handlers

**Return Types**:
- Similar to HTTP, but response sent via WebSocket message
- No HTTP status code (WS uses close codes instead)

---

## 6. Dependency Resolution Performance

### 6.1 Benchmarks

**Typical Costs** (from code comments):
- Per-injection reflection: 5-10 μs
- Context pooling allocation: ~500 ns
- Route matching: O(1) for exact, O(log n) for wildcard
- Full request (simple handler, no I/O): 1-2 ms

### 6.2 Optimization: getFnArgsByType() Caching

**Optimization**: Handler argument types are cached to avoid repeated reflection

**Pseudo-Code**:
```
var handlerArgTypeCache = make(map[string][]string)

func getFnArgsByType(fType reflect.Type, ...) {
    cacheKey := fType.String()
    
    if cachedArgTypes, ok := handlerArgTypeCache[cacheKey]; ok {
        // Use cached arg types
        for _, argType := range cachedArgTypes {
            // Process cached types
        }
    } else {
        // Reflect on handler for first time
        argTypes := []string{}
        for i := 0; i < fType.NumIn(); i++ {
            argType := fType.In(i).String()
            argTypes = append(argTypes, argType)
        }
        handlerArgTypeCache[cacheKey] = argTypes
    }
}
```

**Impact**: Reduces reflection cost on repeated calls to same handler

---

## 7. Panic Recovery Points

### 7.1 Where Panics Are Recovered

| Layer | Recovers? | Recovery Action |
|-------|-----------|-----------------|
| Handler invocation | YES | → exception filter |
| Middleware | YES | → exception filter |
| Guard | NO | → propagates |
| Interceptor | NO | → propagates |
| Exception filter | NO | → unrecoverable |

### 7.2 Panic Propagation Rules

**Rule 1**: Panic in handler → recovered by recovery wrapper

**Rule 2**: Panic in guard → NOT recovered → request terminates

**Rule 3**: Panic in exception filter → NOT recovered → unrecoverable error

**Rule 4**: Only exception filters can handle exceptions

---

## 8. Known Limitations & Gotchas

### 8.1 Parameter Type Matching

**GOTCHA**: Provider type must match parameter type exactly

```go
// Provider registered as
type UserService struct { ... }
func (u UserService) NewProvider() Provider { return u }

// Handler expects
func (c UserController) READ(us *UserService) { // WRONG!
    // Parameter is *UserService, provider is UserService
    // Type keys don't match: "*main.UserService" vs "main.UserService"
    // Result: panic("dependency injection: can't resolve")
}

// Correct:
func (c UserController) READ(us UserService) { // RIGHT!
    // Both are UserService, types match
}
```

### 8.2 Circular Dependency Panics

**GOTCHA**: Circular dependencies cause runtime panics at app.Create()

```go
// A → B → A cycle
type ServiceA struct {
    B ServiceB  // A depends on B
}

type ServiceB struct {
    A ServiceA  // B depends on A
}

// At app.Create(): ServiceA.NewProvider() → tries to inject B → 
// B.NewProvider() → tries to inject A → infinite recursion → panic
```

### 8.3 Nil Dependency Injection

**GOTCHA**: If provider returns nil, handler gets nil

```go
func (u UserService) NewProvider() Provider {
    if someCondition {
        return nil  // Returns nil
    }
    return u
}

// Handler gets nil
func (c *UserController) READ(us UserService) {
    us.DoSomething()  // NilPointerException if us is nil!
}
```

