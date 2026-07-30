# Modules & Providers System

**Optimization**: Initialization order diagrams, state machines, decision trees for composition.

## 1. Module System Basics

### 1.1 Module Purpose

**Module** = Composition root + configuration container

**Responsibilities**:
- Declare controllers (HTTP/WS handlers)
- Declare providers (business logic, services)
- Declare exports (what's available globally)
- Declare imports (dependency on other modules)
- Declare lifecycle hooks (setup/teardown)

### 1.2 Module Structure

```go
type Module struct {
    providers              []Provider
    controllers            []Controller
    exports                []string
    imports                []*Module
    
    HTTPMainHandlers       []HTTPMainHandler
    HTTPExceptionFilters   []HTTPExceptionFilter
    HTTPGuards             []HTTPGuard
    HTTPInterceptors       []HTTPInterceptor
    HTTPMiddlewares        []HTTPMiddleware
    
    WSHandlers             []WSHandler
    WSExceptionFilters     []WSExceptionFilter
    WSGuards               []WSGuard
    WSInterceptors         []WSInterceptor
    WSMiddlewares          []WSMiddleware
}
```

### 1.3 Module Tree

**Structure** (at app.Create()):
```
RootModule
├── Imports:
│   ├── AuthModule
│   │   └── Providers: JWTService, AuthService
│   ├── UserModule
│   │   ├── Controllers: UserController
│   │   └── Providers: UserService, UserRepository
│   └── ProductModule
│       ├── Controllers: ProductController
│       └── Providers: ProductService
├── Controllers: AppController
└── Providers: AppLogger, Config
```

**Traversal**: Depth-first walk during app.Create() to collect all resources

---

## 2. Creating Modules

### 2.1 Static Module (Singleton)

**Pattern**: Single instance, created once

```go
var AppModule = func() *core.Module {
    return core.ModuleBuilder().
        Controllers(
            AppController{},
        ).
        Providers(
            AppLogger{},
            DatabasePool{},
        ).
        Imports(
            AuthModule,
            UserModule,
        ).
        Build()
}()

// Usage:
app := core.New()
app.Create(AppModule)
```

**Characteristics**:
- Created at app startup
- Single instance for entire app lifetime
- Controllers are instantiated per-request

### 2.2 Dynamic Module (Factory)

**Pattern**: Factory function that creates module instances

```go
func NewConfigModule(envPath string) *core.Module {
    cfg := loadConfig(envPath)
    return core.ModuleBuilder().
        Providers(
            ConfigService{config: cfg},
        ).
        Build()
}

// Usage:
app := core.New()
app.Create(
    core.ModuleBuilder().
        Imports(
            NewConfigModule("/etc/app.env"),  // Create instance
            AuthModule,
        ).
        Build(),
)
```

**Characteristics**:
- Created on-demand
- Can accept parameters
- Used for configurable modules

---

## 3. Module Builder API

### 3.1 ModuleBuilder() Methods

```go
ModuleBuilder().
    Controllers(UserController{}, ProductController{}).
    Providers(UserService{}, ProductService{}).
    Imports(AuthModule, DatabaseModule).
    Export("UserService", "ProductService").  // Make globally available
    Build()
```

**Methods**:
- `Controllers(...Controller) ModuleBuilder` — Register HTTP/WS handlers
- `Providers(...Provider) ModuleBuilder` — Register business logic
- `Imports(...*Module) ModuleBuilder` — Import other modules
- `Export(...string) ModuleBuilder` — Export provider names globally
- `Build() *Module` — Create module instance

### 3.2 Module Chaining

```go
core.ModuleBuilder().
    Controllers(C1{}).
    Controllers(C2{}).  // Can chain multiple calls
    Providers(P1{}).
    Providers(P2{}).
    Imports(M1).
    Imports(M2).
    Build()
```

---

## 4. Provider System

### 4.1 Provider Interface

```go
type Provider interface {
    NewProvider() Provider
}
```

**Only Requirement**: Implement `NewProvider() Provider`

### 4.2 Provider Lifecycle

```
1. Declaration: Register provider in ModuleBuilder
2. Discovery: app.Create() walks module tree
3. Registration: Provider added to injectedProviders map
4. Key Generation: type.String() → key (e.g., "main.UserService")
5. Injection: When handler needs provider type
6. Instantiation: provider.NewProvider() called
7. Reuse: Caller holds reference (or discards immediately)
```

### 4.3 Provider Instantiation Timing

**CRITICAL**: `NewProvider()` called PER INJECTION, not at startup

```go
// WRONG - doesn't create singleton
type UserService struct { }
func (u UserService) NewProvider() Provider {
    return UserService{}  // NEW instance every time
}

// RIGHT - creates singleton
var userSingleton = UserService{repo: globalRepo}
type UserService struct { }
func (u UserService) NewProvider() Provider {
    return userSingleton  // REUSE instance
}
```

### 4.4 Provider with Initialization

```go
type DatabasePool struct {
    conn *sql.DB
}

func (d DatabasePool) NewProvider() Provider {
    // Lazy initialization (first call initializes)
    if d.conn == nil {
        d.conn = initDatabase()
    }
    return d
}

// OR: Initialize once at module creation
var dbPool = &DatabasePool{
    conn: initDatabase(),
}

func (d DatabasePool) NewProvider() Provider {
    return dbPool
}
```

---

## 5. Dependency Injection Between Providers

### 5.1 Provider-to-Provider Injection

**Inject providers into other providers**:

```go
type UserService struct {
    Repository UserRepository  // Inject provider
}

type UserRepository struct {
    DB DatabasePool  // Inject provider
}

// Providers are injected by type name matching
// UserRepository field will get DatabasePool provider
```

**How It Works**:
1. User service needs UserRepository
2. Framework looks up provider with type "UserRepository"
3. Finds UserRepository provider
4. Calls UserRepository.NewProvider()
5. Injects into UserService

### 5.2 Provider Injection in Controllers

```go
type UserController struct {
    common.REST
    UserService UserService  // Framework injects provider
}

func (c UserController) NewController() Controller {
    // Before NewController() returns, framework resolves dependencies
    // UserService field will be populated
    return c
}
```

### 5.3 Circular Dependencies (Detect at app.Create())

```go
type ServiceA struct {
    B ServiceB  // A → B
}

type ServiceB struct {
    A ServiceA  // B → A (circular!)
}

// At app.Create():
// ServiceA.NewProvider() → tries to inject ServiceB
// ServiceB.NewProvider() → tries to inject ServiceA
// Infinite recursion → stack overflow → panic
```

**Mitigation Strategies**:
1. **Restructure**: Move shared logic to third service
2. **Lazy Loading**: Defer initialization
3. **Manual Wiring**: Don't use injection for circular deps

---

## 6. Module Exports & Global Availability

### 6.1 Export Mechanism

```go
ModuleBuilder().
    Providers(
        UserService{},
        AuthService{},
        DatabasePool{},
    ).
    Export("UserService", "AuthService").  // Only these are global
    Build()
```

**Effect**:
- Exported providers available in other modules
- Non-exported providers only available within module
- Global scope: accessible to any module that imports this one

### 6.2 Export Semantics

**Rule**: Exported = globally available by type name

**Example**:
```go
// Module A exports UserService
ExportedModule := ModuleBuilder().
    Providers(UserService{}).
    Export("UserService").
    Build()

// Module B can use it
ModuleBuilder().
    Providers(
        MyService{},  // Can depend on UserService from Module A
    ).
    Imports(ExportedModule).
    Build()

// MyService.NewProvider():
// Can inject UserService from ExportedModule
```

---

## 7. Global Modules

### 7.1 IsGlobal Flag

```go
ModuleBuilder().
    IsGlobal(true).
    Providers(GlobalLogger{}).
    Build()
```

**Effect**:
- Providers available EVERYWHERE (all modules)
- No need to explicitly import module
- Global scope automatically

### 7.2 Global Module Semantics

**Without Global**:
```
Module A
  Imports(Module B)
    Module B providers available only in Module A

Module C
  Imports(Module B)
    Module B providers available only in Module C
```

**With IsGlobal**:
```
Global Module
  All modules automatically import
  Providers available everywhere
```

---

## 8. Module Initialization Order (app.Create())

### 8.1 Initialization Sequence (CRITICAL)

**MUST NOT BE REORDERED**:

```
app.Create(rootModule)
    ↓
1. initLogger()
   └─ Setup Logger (first dependency for everything)

2. initProviders(rootModule)
   ├─ Walk module tree (depth-first)
   ├─ Collect all providers
   ├─ Collect all controllers
   ├─ Collect all handlers
   └─ Build injectedProviders map
   
3. initWS(injectedProviders)
   ├─ IF EnableWS called:
   │  ├─ Create WS instance
   │  ├─ Set up WebSocket routing
   │  └─ Prepare for WS handlers
   └─ IF NOT called: skip (ws remains nil)
   
4. initMiddlewares(injectedProviders)
   ├─ Resolve global middleware instances
   ├─ Resolve module-scoped middleware
   └─ Build middleware chains
   
5. initExceptionFilters(injectedProviders)
   ├─ Resolve HTTP exception filters
   ├─ Resolve WS exception filters
   └─ Build filter chains
   
6. initGuards(injectedProviders)
   ├─ Resolve global guards
   ├─ Resolve route-specific guards
   └─ Register guards
   
7. initInterceptors(injectedProviders)
   ├─ Resolve interceptors
   └─ Register interceptors
   
8. initMainHandlers()
   ├─ Build routing trie
   ├─ Register HTTP handlers
   ├─ Build handler chains
   └─ Setup exception filter routing
   
9. initDevtool()
   ├─ IF Devtool enabled:
   │  └─ Create route/handler snapshot
   └─ IF NOT: skip
   
10. initAccessLog()
    ├─ IF AccessLog enabled:
    │  └─ Setup access logging
    └─ IF NOT: skip
```

### 8.2 Why Order Matters

**Dependency Cascade**:
- initProviders MUST run first (builds provider map)
- initWS MUST run before exception/guard/interceptor init (WS state is read)
- initDevtool MUST run last (needs fully populated state)

**If Reordered** (WILL BREAK):
- Exception filters might reference uninitialized WS state
- Handlers might be registered before providers available
- Devtool might see incomplete state

---

## 9. Module Best Practices

### 9.1 DO

- ✓ Use static modules for singletons (performance)
- ✓ Use dynamic modules for configuration (flexibility)
- ✓ Keep modules focused (single responsibility)
- ✓ Export only necessary providers
- ✓ Use module imports to declare dependencies
- ✓ Use IsGlobal sparingly (only for truly global services)

### 9.2 DON'T

- ✗ Use circular module dependencies
- ✗ Mutate module state after Create()
- ✗ Create provider instances manually (use framework injection)
- ✗ Store mutable state in static module variables
- ✗ Mix static and dynamic module patterns
- ✗ Call NewProvider() manually in handlers

---

## 10. Common Module Patterns

### 10.1 Service Layer Module

```go
var UserModule = func() *core.Module {
    return core.ModuleBuilder().
        Controllers(
            UserController{},
        ).
        Providers(
            UserService{},
            UserRepository{},
        ).
        Export("UserService").
        Build()
}()
```

### 10.2 Configuration Module

```go
func NewConfigModule(path string) *core.Module {
    config := loadConfigFromFile(path)
    
    return core.ModuleBuilder().
        IsGlobal(true).
        Providers(
            ConfigService{config: config},
        ).
        Build()
}

// Usage:
app.Create(
    core.ModuleBuilder().
        Imports(
            NewConfigModule("/etc/app.env"),
            UserModule,
        ).
        Build(),
)
```

### 10.3 Dependency Hierarchy

```go
// Layer 1: Infrastructure (database, cache)
var InfrastructureModule = func() *core.Module {
    return core.ModuleBuilder().
        IsGlobal(true).
        Providers(
            DatabasePool{},
            CacheService{},
        ).
        Export("DatabasePool", "CacheService").
        Build()
}()

// Layer 2: Business logic (services)
var BusinessModule = func() *core.Module {
    return core.ModuleBuilder().
        Providers(
            UserService{},      // Uses DatabasePool (injected)
            ProductService{},   // Uses CacheService (injected)
        ).
        Imports(InfrastructureModule).
        Export("UserService", "ProductService").
        Build()
}()

// Layer 3: Handlers (controllers)
var AppModule = func() *core.Module {
    return core.ModuleBuilder().
        Controllers(
            UserController{},    // Uses UserService (injected)
            ProductController{}, // Uses ProductService (injected)
        ).
        Imports(BusinessModule).
        Build()
}()

// Usage:
app.Create(AppModule)
```

