# AI Agent Quick Reference: Ginject Decision Trees

**Purpose**: Rapid decision-making for common AI code generation scenarios in Ginject.

---

## Decision Tree: "How do I implement this feature?"

```
Feature Request
  │
  ├─ "I need an HTTP endpoint"
  │   ├─ Embed common.REST in controller
  │   ├─ Create method: READ/CREATE/UPDATE/MODIFY/DELETE_BY_params
  │   ├─ Add parameters to method signature (injected types)
  │   ├─ Register controller in ModuleBuilder().Controllers(...)
  │   └─ Provider dependencies injected automatically
  │
  ├─ "I need WebSocket support"
  │   ├─ Embed common.WS in controller
  │   ├─ Create method: ON_TOPIC_NAME
  │   ├─ Inject *ctx.WSContext, *websocket.Conn, common.Publisher
  │   ├─ Register controller in ModuleBuilder().Controllers(...)
  │   └─ Use publisher.Publish() for fanout
  │
  ├─ "I need to validate input"
  │   ├─ Option 1: Create Pipeable struct (Transform method)
  │   ├─ Option 2: Validate in middleware and panic
  │   ├─ Option 3: Validate in handler and return error
  │   └─ NEVER validate in guard (guard is for auth, not validation)
  │
  ├─ "I need authentication"
  │   ├─ Create Guard struct with CanActivate()
  │   ├─ Guard checks JWT/session in httpCtx.Request.Header
  │   ├─ Return true if authorized, false if forbidden
  │   ├─ NEVER panic in guard
  │   └─ Bind guard to specific handlers
  │
  ├─ "I need to modify request/response"
  │   ├─ Use Middleware for pre-processing
  │   ├─ Use Interceptor for post-processing
  │   ├─ Middleware can call next() or stop
  │   └─ Interceptor can transform aggregation.Data
  │
  ├─ "I need to handle errors"
  │   ├─ Panic with exception.BadRequestException(...) in handler
  │   ├─ Exception will be caught and exception filter called
  │   ├─ Or return error from handler
  │   └─ Custom exception filter can catch specific exceptions
  │
  ├─ "I need access to database/service"
  │   ├─ Create Provider with NewProvider()
  │   ├─ Register in ModuleBuilder().Providers(...)
  │   ├─ Inject into handler/controller/other-provider
  │   ├─ Framework resolves dependency automatically
  │   └─ DON'T instantiate manually
  │
  ├─ "I need to log something"
  │   ├─ Inject common.Logger into handler
  │   ├─ Call logger.Log(level, data)
  │   ├─ Or use global app.Logger
  │   └─ Logger is available everywhere
  │
  └─ "I need to send events (WebSocket)"
      ├─ Inject common.Publisher into handler
      ├─ Call publisher.Publish(topic, data)
      ├─ All WS clients subscribed to topic receive message
      └─ Use topic patterns for selective delivery
```

---

## Decision Tree: "What's the right place for this code?"

```
Piece of Code
  │
  ├─ "Runs for every request"
  │   └─ Use Middleware (BindGlobalMiddlewares or BindMiddleware)
  │
  ├─ "Checks if request is allowed"
  │   └─ Use Guard (BindGuard on controller)
  │       Note: Guard MUST return bool, NOT panic
  │
  ├─ "Transforms request or response"
  │   ├─ Pre-handler transformation → Interceptor (pre-phase)
  │   ├─ Post-handler transformation → Interceptor (post-phase)
  │   └─ Use interceptor.Intercept(httpCtx, aggregation)
  │
  ├─ "Handles errors/panics"
  │   └─ Use Exception Filter (BindExceptionFilter on controller)
  │       Note: Only place that can catch panics
  │
  ├─ "Business logic for specific handler"
  │   └─ Write handler method in Controller
  │       Parameters injected from DI container
  │
  ├─ "Business logic reused across handlers"
  │   └─ Create Provider
  │       Register in ModuleBuilder().Providers(...)
  │       Inject into handlers/other-providers
  │
  ├─ "Initialization/setup code"
  │   ├─ If per-handler: put in middleware
  │   ├─ If per-module: create Provider with setup in NewProvider()
  │   └─ If global: use IsGlobal(true) module or BindGlobalMiddlewares
  │
  ├─ "Cleanup/teardown code"
  │   └─ Use defer in middleware or handler
  │       (Framework doesn't have explicit lifecycle hooks)
  │
  └─ "Logging, tracing, monitoring"
      ├─ Use event.Emit() for custom events
      ├─ Use Logger for structured logging
      └─ Can go in middleware, handler, or provider
```

---

## Decision Tree: "What dependency should I inject?"

```
I need access to...
  │
  ├─ "Current HTTP request"
  │   └─ *http.Request (HTTP only)
  │
  ├─ "Response writer"
  │   └─ http.ResponseWriter (HTTP only)
  │
  ├─ "Full request context with helpers"
  │   ├─ HTTP: *ctx.HTTPContext
  │   └─ WS: *ctx.WSContext
  │
  ├─ "Request body (parsed JSON/form)"
  │   └─ ctx.Body (HTTP only)
  │
  ├─ "Query parameters"
  │   └─ ctx.Query (HTTP only)
  │
  ├─ "Path parameters (:id, :name, etc.)"
  │   └─ ctx.Param (HTTP only)
  │
  ├─ "HTTP headers"
  │   └─ ctx.Header (HTTP only)
  │
  ├─ "File uploads"
  │   └─ ctx.File (HTTP only)
  │
  ├─ "WebSocket connection"
  │   └─ *websocket.Conn (WS only)
  │
  ├─ "Incoming WebSocket message"
  │   └─ ctx.WSPayload (WS only)
  │
  ├─ "Send response/message"
  │   ├─ HTTP: httpCtx.JSON() or Text() method
  │   └─ WS: common.Publisher for fanout
  │
  ├─ "Custom service/provider"
  │   └─ Register as Provider, inject by type
  │       Framework resolves automatically
  │
  ├─ "Logger"
  │   └─ common.Logger (globally available)
  │
  ├─ "Event publisher"
  │   └─ common.Publisher (for WS fanout)
  │
  ├─ "Call next middleware"
  │   └─ ctx.Next (function injected in middleware)
  │
  └─ "Redirect response"
      └─ ctx.Redirect (function injected in handler)
```

---

## Decision Tree: "How do I handle this error?"

```
Error Scenario
  │
  ├─ "Input validation error (bad data)"
  │   ├─ Option 1: Panic with exception.BadRequestException
  │   ├─ Option 2: Return error type from handler
  │   ├─ Option 3: Validate in middleware, call next() or not
  │   └─ Exception filter writes response
  │
  ├─ "Authentication error (no token/invalid token)"
  │   ├─ Option 1: Return false from Guard
  │   ├─ Option 2: Middleware checks before guard
  │   ├─ Option 3: Exception filter for custom auth errors
  │   └─ Result: 403 Forbidden or custom error response
  │
  ├─ "Authorization error (user lacks permission)"
  │   ├─ Same as authentication
  │   └─ Guard returns false = 403 Forbidden
  │
  ├─ "Database error (connection failed)"
  │   ├─ Panic with exception.ServiceUnavailableException
  │   ├─ Or return custom error from handler
  │   └─ Exception filter writes 503 Service Unavailable
  │
  ├─ "Resource not found"
  │   ├─ Panic with exception.NotFoundException
  │   └─ Exception filter writes 404 Not Found
  │
  ├─ "Request timeout (handler takes too long)"
  │   ├─ httpCtx.SetDeadline(5 * time.Second)
  │   ├─ Framework checks before invoking handler
  │   ├─ Panics with exception.RequestTimeoutException
  │   └─ Exception filter writes 408 Request Timeout
  │
  ├─ "Internal server error (unexpected panic)"
  │   ├─ Panic with exception.InternalServerErrorException
  │   ├─ Or any panic caught by exception filter
  │   └─ Exception filter writes 500 Internal Server Error
  │
  ├─ "Not implemented"
  │   ├─ Panic with exception.NotImplementedException
  │   └─ Exception filter writes 501 Not Implemented
  │
  └─ "Panic in guard/interceptor (unrecoverable)"
      ├─ Framework does NOT catch this
      ├─ Unhandled panic → app crash
      └─ NEVER panic in guard, NEVER panic in interceptor pre
```

---

## Decision Tree: "How do I structure my application?"

```
Architecture Question
  │
  ├─ "How many modules should I have?"
  │   ├─ Minimum: 1 root module
  │   ├─ Recommended: 1 per feature (users, products, orders)
  │   ├─ Use imports to compose
  │   └─ Each module handles one responsibility
  │
  ├─ "How do I share code between modules?"
  │   ├─ Create shared Provider (register in imported module)
  │   ├─ Use IsGlobal(true) for truly global services
  │   ├─ Export(...) specific providers from module
  │   └─ Inject into handlers/controllers in other modules
  │
  ├─ "Where should I put validation logic?"
  │   ├─ Option 1: Middleware (runs before handler)
  │   ├─ Option 2: Interceptor pre-phase (runs before handler)
  │   ├─ Option 3: Pipeable (custom transform for specific param)
  │   └─ NOT in Guard (guards are for auth/permissions)
  │
  ├─ "Where should I put caching logic?"
  │   ├─ Create Provider with injected CacheService
  │   ├─ Cache wrapper around database queries
  │   ├─ Invalidate cache on writes
  │   └─ Use memorycache.MemoryCache or external cache
  │
  ├─ "How do I structure database access?"
  │   ├─ Create Repository Provider (data access)
  │   ├─ Create Service Provider (business logic)
  │   ├─ Service depends on Repository
  │   ├─ Controller depends on Service (not directly on Repository)
  │   └─ All via dependency injection
  │
  ├─ "How do I handle cross-cutting concerns?"
  │   ├─ Logging → Middleware or interceptor
  │   ├─ Error handling → Exception filter
  │   ├─ Authentication → Guard or middleware
  │   ├─ Tracing → event.On() listener
  │   └─ Rate limiting → Guard
  │
  └─ "How do I test this application?"
      ├─ Create new app instance per test
      ├─ Create new module per test (avoid static state)
      ├─ Mock providers via dependency injection
      ├─ Use httptest.Server for HTTP testing
      ├─ For WS: use real websocket connection or mock interface
      └─ Never share state between tests
```

---

## Decision Tree: "Is this pattern safe?"

```
Pattern Question
  │
  ├─ "Should I hold onto context after handler?"
  │   └─ NO - context is pooled, reused in next request
  │       Result: stale data, wrong request
  │
  ├─ "Should I mutate context after guard?"
  │   └─ NO - guard made decisions based on original state
  │       Result: bypass guard logic, security issue
  │
  ├─ "Should I call next() multiple times?"
  │   └─ NO - causes double-processing
  │       Result: handler called multiple times
  │
  ├─ "Should I panic in guard?"
  │   └─ NO - panic not caught in guard
  │       Result: unhandled panic, app crash
  │
  ├─ "Should I panic in interceptor?"
  │   └─ NO - panic not caught in interceptor pre-phase
  │       Result: unhandled panic, app crash
  │
  ├─ "Should I hold state in provider?"
  │   └─ DEPENDS - if mutable, protect with sync.Mutex
  │       Result: race conditions without mutex
  │
  ├─ "Should I use raw conn.Send() in WS handler?"
  │   └─ NO - concurrent with fanout writes
  │       Result: message corruption
  │
  ├─ "Should I store mutable state in module?"
  │   └─ NO - module is singleton, shared across requests
  │       Result: race conditions
  │
  ├─ "Should I call NewProvider() manually?"
  │   └─ NO - breaks dependency injection
  │       Result: defeats pooling, lifecycle management
  │
  └─ "Should I have circular dependencies?"
      └─ NO - causes infinite recursion at app.Create()
          Result: stack overflow, unrecoverable panic
```

---

## Quick Rules (Memorize These)

1. **Context Lifecycle**: Never hold context after handler returns
2. **Dependency Injection**: Let framework inject, never call NewProvider()
3. **Guard Behavior**: Must return bool, never panic
4. **Exception Handling**: Only exception filter catches panics
5. **WebSocket**: Use Publisher, never raw conn access
6. **Middleware**: Call next() exactly once
7. **Pipeline Order**: Middleware → Guard → Interceptor → Handler → Filter
8. **Module Init**: NEVER reorder app.Create() steps
9. **Provider Injection**: Type MUST match exactly (pointer vs value)
10. **Performance**: Reflection cost ~5-10μs per dependency

---

## File Organization Pattern

```
myapp/
├── main.go
├── modules/
│   ├── users/
│   │   ├── user_controller.go
│   │   ├── user_service.go
│   │   ├── user_repository.go
│   │   └── user_module.go
│   ├── products/
│   │   ├── product_controller.go
│   │   ├── product_service.go
│   │   └── product_module.go
│   └── shared/
│       ├── config_service.go
│       ├── database.go
│       └── shared_module.go
├── common/
│   ├── middleware.go (global middleware)
│   ├── guards.go
│   └── filters.go
└── tests/
    └── ...
```

