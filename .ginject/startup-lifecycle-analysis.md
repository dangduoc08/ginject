# Startup Lifecycle & Race Condition Analysis

**Purpose**: Document application startup sequence, module initialization guarantees, and identified race conditions.

**Last Updated**: v2.1 (matches current codebase)

---

## Critical Finding

⚠️ **POTENTIAL RACE CONDITION EXISTS**: WebSocket handlers can be called before dependent modules complete initialization if:
1. Handler depends on another module (e.g., Cache service)
2. Module uses async initialization (e.g., `go connectToRedis()`)
3. WebSocket client connects during that async phase
4. Framework does not guarantee all modules fully initialized when `Listen()` starts

---

## Startup Sequence

### Phase 1: App Creation (`core.New()`)

```
core.New()
  ├─ Create HTTP server (net/http)
  ├─ Create event emitter
  ├─ Create memorybroker (READY FOR PUB/SUB IMMEDIATELY)
  ├─ Create publisher (wraps broker)
  └─ Initialize context pools
```

**Key Point**: Broker is created and ready immediately. No subscriptions yet.

### Phase 2: App Initialization (`app.Create(m *Module)`)

Sequential phases (NO CONCURRENCY):

```
1. initLogger()
   └─ Create logger, register globally

2. initProviders(m)
   └─ CALLS: m.NewModule()
      ├─ CALLS: OnInit() for each module (SEQUENTIAL)
      ├─ Injects static modules
      ├─ Injects dynamic modules
      ├─ Registers all providers
      └─ Registers all controllers
      ↓ Lock held during entire NewModule()

3. initWS()
   └─ If WebSocket enabled: create WS handler

4. initMiddlewares()
5. initExceptionFilters()
6. initGuards()
7. initInterceptors()
8. initMainHandlers()
9. initDevtool()
10. initAccessLog()

11. callOnReady()
   └─ CALLS: OnReady() for each module (SEQUENTIAL)
```

**Key Point**: All initialization is SEQUENTIAL within `Create()`, NOT concurrent.

### Phase 3: Server Start (`app.Listen(port)`)

```
app.Listen(port)
  ├─ Log module initialization status
  ├─ Log HTTP routes
  ├─ Log WS events (if enabled)
  └─ Start HTTP server (IMMEDIATELY ACCEPTING CONNECTIONS)
```

**Key Point**: HTTP server accepts connections IMMEDIATELY after this call.

### Phase 4: WebSocket Client Connection

```
Client: GET /ws HTTP/1.1 Upgrade: websocket

Server: Run middleware chain (handshake phase)

Server: Accept connection → readLoop() starts

Client: Send message (e.g., TypeSubscribe)

Server: handleSubscribe()
  ├─ Run middleware
  ├─ DEPENDS ON: Handler parameter types
  ├─ RESOLVE: Dependency injection
  ├─ CALLS: Handler
```

**Key Point**: Dependencies resolved at THIS POINT, not before.

---

## Race Conditions

### Condition 1: Async Module Initialization ⚠️ REAL

**Scenario**:
```go
// Cache module
module.OnInit = func() {
    go client.Connect("redis://...")  // ← Async!
}

// WebSocket handler
app.BindWSHandler("event.update", func(cache CacheService) {
    cache.Set("key", "value")  // ← Depends on async!
})

// Timeline:
// T0: OnInit() starts, async connect spawned
// T1: OnInit() returns (Connect still pending!)
// T2: OnReady() called (Connect still pending!)
// T3: app.Listen() → HTTP server listening
// T4: WebSocket client connects
// T5: handleSubscribe() → depends on cache
// T6: cache.Set() → FAILS (Redis not connected)
```

**Likelihood**: HIGH if modules use goroutines

**Impact**: Handler fails, exception sent to client, no message loss (in-memory)

**Framework Guarantee**: NONE. Framework only waits for sync code to complete.

### Condition 2: Module Dependency Ordering ⚠️ THEORETICAL

**Scenario**:
```go
// Module A imports Module B, but B not listed first
moduleTree := core.ModuleBuilder().
    Imports(moduleA, moduleB).  // ← Order unspecified!
    Build()
```

**Likelihood**: LOW (good practice catches this)

**Impact**: Panic during initialization (DI resolution fails)

**Framework Guarantee**: None for import order.

### Condition 3: Broker Subscriptions ✓ NOT POSSIBLE

**Why this CANNOT happen**:
- Broker is created in `core.New()`
- Subscriptions only happen when WebSocket client sends TypeSubscribe message
- That happens AFTER HTTP server listening AFTER all modules initialized
- No automatic subscriptions during startup

---

## Module Lifecycle Guarantees

### What IS Guaranteed

| Phase | Guarantee | Confidence |
|-------|-----------|------------|
| OnInit() called | SYNC completion before next phase | HIGH |
| OnReady() called | SYNC completion before Listen() | HIGH |
| All providers created | Before Listen() | HIGH |
| All controllers registered | Before Listen() | HIGH |
| Middleware chain installed | Before first request | HIGH |

### What IS NOT Guaranteed

| Item | Why |
|------|-----|
| Async operations complete | Framework doesn't track goroutines |
| External services ready | Connect() runs async outside framework |
| Module A ready before B | No import order guarantees |
| All modules same readiness level | Each has own OnInit/OnReady |

---

## Message Safety

### Broker Lifecycle

```
1. Broker created in core.New()
   ├─ In-memory storage: 4 maps (exact/prefix/global/complex), single sync.RWMutex, no sharding
   ├─ No goroutines started eagerly — PublishAsync spawns one goroutine per call, on demand
   └─ Ready for Publish/Subscribe IMMEDIATELY

2. Subscriptions only when WebSocket client connects
   ├─ Client sends TypeSubscribe
   ├─ handleSubscribe() called
   ├─ Middleware chain runs
   ├─ ws.connmgr.Subscribe() → memorybroker.Subscribe()
   └─ Handler installed

3. On message failure
   ├─ Exception caught
   ├─ Exception sent to client
   ├─ NO ACK/NACK (in-memory)
   ├─ NO RETRY
   └─ Message effectively lost (not persisted)
```

### Message Guarantees

| Scenario | Behavior | Loss Risk |
|----------|----------|-----------|
| Handler succeeds | Message processed, no ack needed | None |
| Handler panics | Exception caught, sent to client | Lost |
| Handler depends missing | Exception sent | Lost |
| Slow client buffer full | Message dropped via TrySend | Lost |
| App crashes mid-dispatch | Message lost (in-memory) | High |

---

## Recommendations to Avoid Race Conditions

### 1. Avoid Async in OnInit()

```go
// ❌ BAD
module.OnInit = func() {
    go connect()  // Async!
}

// ✓ GOOD
module.OnInit = func() {
    connect()  // Sync, blocks until ready
}
```

### 2. Use OnReady() for Background Tasks

```go
module.OnReady = func() {
    // After ALL modules initialized
    go startWorkerPool()
    go startHealthChecks()
}
```

### 3. Verify Dependencies Before Returning

```go
module.OnReady = func() {
    if !redis.IsConnected() {
        panic("Redis not connected!")
    }
}
```

### 4. Never Depend in Middleware

```go
// ❌ BAD - Dependency resolved in handler
app.BindWSHandler("event", func(cache CacheService) {
    cache.Set("key", "value")
})

// ✓ BETTER - Inject at app level, verify readiness
cacheService := ... // Inject, verify ready
app.BindWSHandler("event", func() {
    cacheService.Set("key", "value")
})
```

### 5. Wait for Client Connections After Ready

```go
// Good practice (though not enforced)
app.Create(rootModule)
// All modules initialized now

// Verify critical services
if err := verifyServices(); err != nil {
    log.Fatal(err)
}

// Now start listening
app.Listen(8080)
```

---

## Implementation Evidence

**File**: `core/app.go:212-224`
```go
func (app *App) Create(m *Module) {
    app.initLogger()                    // Sequential
    injectedProviders := app.initProviders(m)  // Calls NewModule() -> OnInit()
    // ... more sequential steps ...
    app.callOnReady()                   // Calls OnReady() for each module
}
```

**File**: `core/module.go:137-163`
```go
func (m *Module) NewModule() *Module {
    m.Lock()  // ← Critical section
    defer m.Unlock()
    
    if m.OnInit != nil {
        m.OnInit()  // ← Called here, MUST complete sync
    }
    // ... module setup ...
}
```

**File**: `core/app.go:637-693`
```go
func (app *App) Listen(port int) error {
    // ... logging ...
    server := &http.Server{...}
    return app.serveWithGracefulShutdown(server)  // Starts listening
}
```

---

## Summary

| Aspect | Status | Risk |
|--------|--------|------|
| Startup sequence is sequential | ✓ YES | LOW |
| All sync init complete before Listen | ✓ YES | LOW |
| Async operations tracked | ❌ NO | HIGH |
| Broker ready immediately | ✓ YES | N/A |
| Module init race conditions | ⚠️ POSSIBLE | MEDIUM |
| Message loss risk | ✓ UNDERSTOOD | MEDIUM |
| OnInit/OnReady ordering | ✓ GUARANTEED | LOW |

**Best Practice**: Use sync initialization in OnInit(), move async work to OnReady() with verification hooks.
