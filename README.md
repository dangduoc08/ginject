# Ginject

A NestJS-inspired dependency injection web framework for Go. Build structured, modular HTTP and WebSocket servers using the **Module → Controller → Provider** pattern.

- [Ginject](#ginject)
  - [Features](#features)
  - [Installation](#installation)
  - [Quick Start](#quick-start)
  - [Core Concepts](#core-concepts)
    - [Application](#application)
    - [Module](#module)
    - [Provider](#provider)
    - [Controller](#controller)
    - [Routing Convention](#routing-convention)
    - [Handler Parameter Injection](#handler-parameter-injection)
    - [Request Pipeline](#request-pipeline)
    - [Middleware](#middleware)
    - [Guard](#guard)
    - [Interceptor](#interceptor)
    - [Exception Filter](#exception-filter)
    - [Pipe / DTO](#pipe--dto)
  - [WebSocket](#websocket)
  - [Versioning](#versioning)
  - [Built-in Modules](#built-in-modules)
    - [Config](#config)
    - [Cache](#cache)
    - [HTTP Client](#http-client)
  - [Built-in Middlewares](#built-in-middlewares)
    - [CORS](#cors)
    - [Helmet](#helmet)
    - [RequestLogger](#requestlogger)
  - [Exception Types](#exception-types)

---

## Features

- Reflection-based dependency injection — no tags, no code generation
- Convention-over-configuration routing via method names
- Full request pipeline: middleware → guard → interceptor → handler → exception filter
- WebSocket support with event-based routing
- API versioning (query, header, media type, custom)
- Type-safe handler parameter injection
- Built-in Config and Cache modules

---

## Installation

```bash
go get github.com/dangduoc08/ginject
```

---

## Quick Start

```go
package main

import (
    "github.com/dangduoc08/ginject/common"
    "github.com/dangduoc08/ginject/core"
)

type UserController struct {
    common.REST
}

func (c UserController) NewController() core.Controller { return c }

func (c UserController) READ() string {
    return "hello"
}

var UserModule = func() *core.Module {
    return core.ModuleBuilder().Controllers(UserController{}).Build()
}

func main() {
    app := core.New()
    app.Create(core.ModuleBuilder().Imports(UserModule).Build())
    app.Logger.Fatal("AppError", "error", app.Listen(3000))
}
```

`GET http://localhost:3000/` → `hello`

---

## Core Concepts

### Application

`core.New()` creates the application. Chain configuration before `Create`:

```go
app := core.New()

app.
    UseLogger(logger).
    BindGlobalMiddlewares(middlewares.CORS{}, middlewares.RequestLogger{}).
    BindGlobalGuards(RateLimiterGuard{}).
    BindGlobalInterceptors(ResponseInterceptor{}).
    BindGlobalExceptionFilters(CustomExceptionFilter{})

app.EnableVersioning(versioning.Versioning{
    Type: versioning.HEADER,
    Key:  "X-Api-Version",
})

app.Create(RootModule)
app.Logger.Fatal("AppError", "error", app.Listen(3000))
```

### Module

Modules group related controllers and providers. Static modules are singletons; dynamic modules accept configuration arguments.

**Static module:**

```go
var UserModule = func() *core.Module {
    module := core.ModuleBuilder().
        Imports(DatabaseModule).
        Controllers(UserController{}).
        Providers(UserService{}).
        Build()

    module.Prefix("users")
    return module
}
```

**Dynamic module** (configurable):

```go
type DatabaseModule struct {
    Host string
    Port int
}

func (d DatabaseModule) New() *core.Module {
    return core.ModuleBuilder().
        Providers(DatabaseService{Host: d.Host, Port: d.Port}).
        Build()
}

// Usage
core.ModuleBuilder().
    Imports(DatabaseModule{Host: "localhost", Port: 5432}).
    Build()
```

**Global module** — providers available everywhere without explicit import:

```go
module := core.ModuleBuilder().Providers(ConfigService{}).Build()
module.IsGlobal = true
```

### Provider

Any struct implementing `NewProvider() Provider`. Providers are injected by field type — declare the field and the framework resolves it automatically.

```go
type UserService struct{}

func (s UserService) NewProvider() core.Provider { return s }

func (s UserService) FindAll() []string {
    return []string{"alice", "bob"}
}

// Inject into another provider or controller
type UserController struct {
    common.REST
    UserService UserService // resolved automatically
}
```

### Controller

Implements `NewController() Controller`. Embed `common.REST` for HTTP or `common.WS` for WebSocket. Bind per-controller layers in `NewController`:

```go
type ManufacturerController struct {
    common.REST
    common.Guard
    common.Middleware
    common.Interceptor
    common.ExceptionFilter
    UserService UserService
}

func (c ManufacturerController) NewController() core.Controller {
    c.BindGuard(AuthGuard{}, c.CREATE_VERSION_1, c.DELETE_VERSION_1)
    c.BindMiddleware(LogMiddleware{}, c.UPDATE_VERSION_1)
    return c
}
```

### Routing Convention

Method names are parsed into HTTP routes — no annotations needed.

**HTTP method tokens:**

| Token       | HTTP Method |
|-------------|-------------|
| `READ`      | GET         |
| `CREATE`    | POST        |
| `UPDATE`    | PUT         |
| `MODIFY`    | PATCH       |
| `DELETE`    | DELETE      |
| `PREFLIGHT` | OPTIONS     |

**Path tokens:**

| Token     | Effect                  | Example                                     |
|-----------|-------------------------|---------------------------------------------|
| `BY`      | Path parameter          | `READ_BY_ID` → `GET /:id`                  |
| `AND`     | Additional segment      | `READ_AND_PROFILE` → `GET /profile`        |
| `OF`      | Sub-resource            | `READ_OF_USERS` → `GET /users`             |
| `ANY`     | Wildcard `*`            | `READ_ANY` → `GET /*`                      |
| `FILE`    | File extension wildcard | `READ_ANY_FILE_HTML` → `GET /*.html`       |
| `VERSION` | API version suffix      | `READ_VERSION_1` → version `1`             |

**Examples:**

```go
func (c UserController) READ() []User {}                     // GET /
func (c UserController) READ_BY_ID() User {}                 // GET /:id
func (c UserController) READ_BY_ID_AND_PROFILE() Profile {}  // GET /:id/profile
func (c UserController) CREATE_VERSION_1(body Body) Map {}   // POST / (version 1)
func (c UserController) DELETE_BY_ID_VERSION_1() {}          // DELETE /:id (version 1)
```

Return values are serialized automatically: structs/maps → JSON, strings/numbers → plain text.

### Handler Parameter Injection

Declare parameters by type — the framework resolves them from the request context:

```go
func (c UserController) READ_BY_ID(
    ctx    ginject.HTTPContext,  // *ctx.HTTPContext
    req    ginject.Request,  // *http.Request
    param  ginject.Param,    // path parameters
    query  ginject.Query,    // query string
    header ginject.Header,   // request headers
) User {
    id := param.Get("id")
    // ...
}

func (c UserController) CREATE(
    body ginject.Body,     // request body (JSON)
    res  ginject.Response, // http.ResponseWriter
) Map {}

func (c UserController) UPLOAD(
    form ginject.Form, // multipart form
    file ginject.File, // uploaded files
) Map {}
```

All types are re-exported from the root `ginject` package.

### Request Pipeline

```
GlobalMiddlewares
    → ModuleMiddlewares
        → Guards
            → Interceptors (pre)
                → MainHandler
            ← Interceptors (post / Pipe)
    ← ExceptionFilters (on panic)
```

Each layer runs per-request. Exception filters catch panics from any layer.

### Middleware

Implement `Use(*http.Request, http.ResponseWriter, ctx.Next)`:

```go
type LogMiddleware struct {
    common.Logger
}

func (m LogMiddleware) Use(r *http.Request, w http.ResponseWriter, next ctx.Next) {
    m.Info("request", "path", r.URL.Path)
    next()
}
```

**Global** (all routes):

```go
app.BindGlobalMiddlewares(LogMiddleware{})
```

**Per-controller** (all or specific handlers):

```go
c.BindMiddleware(LogMiddleware{})                      // all handlers
c.BindMiddleware(LogMiddleware{}, c.CREATE_VERSION_1)  // specific handler only
```

### Guard

Implement `CanActivate(*ctx.HTTPContext) bool` for a REST guard, or `CanActivate(*ctx.WSContext) bool` for a WS guard. Ginject inspects the bound guard's `CanActivate` signature to tell which one it is — a REST-shaped guard only ever runs for REST routes, a WS-shaped guard only for WS events. Returning `false` responds with 403 Forbidden:

```go
type AuthGuard struct {
    common.Guard
    config.ConfigService
    Secret string
}

func (g AuthGuard) NewGuard() AuthGuard {
    g.Secret = g.ConfigService.Get("AUTH_SECRET").(string)
    return g
}

func (g AuthGuard) CanActivate(c *ctx.HTTPContext) bool {
    return c.Header().Get("Authorization") == g.Secret
}
```

**Global:**

```go
app.BindGlobalGuards(AuthGuard{})
```

**Per-controller:**

```go
c.BindGuard(AuthGuard{}, c.CREATE_VERSION_1, c.DELETE_VERSION_1)
```

### Interceptor

Implement `Intercept(*ctx.HTTPContext, *aggregation.Aggregation) any` for a REST interceptor, or `Intercept(*ctx.WSContext, *aggregation.Aggregation) any` for a WS one. Ginject inspects the bound interceptor's `Intercept` signature to tell which one it is — a REST-shaped interceptor only ever wraps REST handlers, a WS-shaped one only WS handlers. Runs before and after the handler via `Pipe`:

```go
type ResponseInterceptor struct{}

func (i ResponseInterceptor) Intercept(c ginject.HTTPContext, agg ginject.Aggregation) any {
    return agg.Pipe(
        agg.Consume(func(c ginject.HTTPContext, data any) any {
            return ginject.Map{"data": data}
        }),
    )
}
```

**Global:**

```go
app.BindGlobalInterceptors(ResponseInterceptor{})
```

**Per-controller:**

```go
c.BindInterceptor(ResponseInterceptor{})
```

### Exception Filter

Implement `Catch(*ctx.HTTPContext, *exception.Exception)` for a REST exception filter, or `Catch(*ctx.WSContext, *exception.Exception)` for a WS one. Ginject inspects the bound filter's `Catch` signature to tell which one it is — a REST-shaped filter only ever catches REST panics, a WS-shaped filter only WS ones. Catches panics and unhandled exceptions:

```go
type AppExceptionFilter struct{}

func (f AppExceptionFilter) Catch(c *ctx.HTTPContext, ex *exception.Exception) {
    httpCode, _ := ex.GetHTTPStatus()
    c.Status(httpCode).JSON(ctx.Map{
        "code":    ex.GetCode(),
        "error":   ex.Error(),
        "message": ex.GetResponse(),
    })
}
```

**Global:**

```go
app.BindGlobalExceptionFilters(AppExceptionFilter{})
```

**Per-controller:**

```go
c.BindExceptionFilter(AppExceptionFilter{})
c.BindExceptionFilter(AppExceptionFilter{}, c.CREATE_VERSION_1)
```

Throw exceptions anywhere in handlers:

```go
func (c UserController) READ_BY_ID(param ginject.Param) User {
    user := findUser(param.Get("id"))
    if user == nil {
        panic(exception.NotFoundException("user not found"))
    }
    return *user
}
```

### Pipe / DTO

Pipes transform and validate handler parameters before injection. Implement the `Transform` method matching the parameter source:

```go
type CreateUserBody struct {
    Name  string `bind:"name"`
    Email string `bind:"email"`
}

func (dto CreateUserBody) Transform(body ginject.Body, meta common.ArgumentMetadata) any {
    result, _ := body.Bind(dto)
    return result
}

// Declare as handler parameter — automatically bound and transformed
func (c UserController) CREATE(body CreateUserBody) Map {
    return ginject.Map{"name": body.Name}
}
```

Supported pipe interfaces: `BodyPipeable`, `QueryPipeable`, `ParamPipeable`, `HeaderPipeable`, `FormPipeable`, `FilePipeable`, `ContextPipeable`, `WSPayloadPipeable`.

---

## WebSocket

Embed `common.WS` instead of `common.REST`. Method names map to subscribe event names:

```go
type ChatController struct {
    common.WS
}

func (c ChatController) NewController() core.Controller { return c }

func (c ChatController) MESSAGE(payload ginject.WSPayload) string {
    return payload.Get("text").(string)
}
```

The WS endpoint is available at `/ws`. Clients send JSON messages:

```json
{ "event": "message", "data": { "text": "hello" } }
```

---

## Versioning

Enable versioning before `Create`:

```go
app.EnableVersioning(versioning.Versioning{
    Type:           versioning.HEADER,
    Key:            "X-Api-Version",
    DefaultVersion: "1",
})
```

**Types:**

| Type         | Description                          | Example                                   |
|--------------|--------------------------------------|-------------------------------------------|
| `QUERY`      | Read version from query parameter    | `GET /users?v=1`                         |
| `HEADER`     | Read version from request header     | `X-Api-Version: 1`                       |
| `MEDIA_TYPE` | Read version from `Accept` header    | `Accept: application/json; version=1`    |
| `CUSTOM`     | Custom extractor function            | `Extractor: func(c *ctx.HTTPContext) string` |

Add `_VERSION_N` suffix to controller methods:

```go
func (c UserController) READ_VERSION_1() Map { /* v1 logic */ }
func (c UserController) READ_VERSION_2() Map { /* v2 logic */ }
```

---

## Built-in Modules

### Config

Reads `.env` files with type binding, variable expansion, and validation hooks. See the [Config module documentation](modules/config/README.md) for full details.

```go
import "github.com/dangduoc08/ginject/modules/config"

core.ModuleBuilder().
    Imports(
        config.Register(&config.ConfigModuleOptions{
            IsGlobal:          true,
            IsExpandVariables: true,
        }),
    ).
    Build()

// Inject ConfigService into any provider
type DBProvider struct {
    config.ConfigService
}

func (p DBProvider) NewProvider() core.Provider {
    host := p.Get("DB_HOST").(string)
    return p
}
```

**`ConfigModuleOptions`:**

| Field               | Type             | Description                                      |
|---------------------|------------------|--------------------------------------------------|
| `IsGlobal`          | `bool`           | Make providers available everywhere              |
| `IsIgnoreEnvFile`   | `bool`           | Use OS env vars only, skip `.env`                |
| `IsOverride`        | `bool`           | Override OS env vars with `.env` values          |
| `IsExpandVariables` | `bool`           | Enable `${VAR}` expansion in values              |
| `ENVFilePaths`      | `[]string`       | Custom `.env` file paths (default: `[".env"]`)   |
| `Loads`             | `[]ConfigLoadFn` | Custom loaders returning `map[string]any`        |
| `Hooks`             | `[]ConfigHookFn` | Hooks to transform/validate config at startup    |
| `OnInit`            | `func()`         | Lifecycle hook before module injection           |

### Cache

High-performance in-memory cache with TTL support and a backend-portable interface. See the [Cache module documentation](modules/cache/README.md) for full details.

```go
import (
    "context"
    "time"

    "github.com/dangduoc08/ginject/modules/cache"
)

core.ModuleBuilder().
    Imports(
        cache.Register(&cache.CacheModuleOptions{
            IsGlobal: true,
        }),
    ).
    Build()

// Inject CacheService into any provider or controller
type UserController struct {
    common.REST
    cache.CacheService
}

func (c *UserController) READ_BY_ID(param ginject.Param) any {
    ctx := context.Background()
    key := "user:" + param.Get("id")

    if data, ok := c.CacheService.Get(ctx, key); ok {
        return data
    }

    data := fetchFromDB(param.Get("id"))
    _ = c.CacheService.Set(ctx, key, data, 5*time.Minute)
    return data
}
```

**`CacheModuleOptions`:**

| Field      | Type     | Description                                 |
|------------|----------|---------------------------------------------|
| `IsGlobal` | `bool`   | Make `CacheService` available everywhere    |
| `OnInit`   | `func()` | Lifecycle hook before module injection      |

### HTTP Client

Axios-inspired outbound HTTP client with middleware chain, retry, streaming, SSE, SSRF protection, and timing. See the [HTTP Client module documentation](modules/httpclient/README.md) for full details.

```go
import "github.com/dangduoc08/ginject/modules/httpclient"

core.ModuleBuilder().
    Imports(
        httpclient.Register(&httpclient.HttpClientModuleOptions{
            IsGlobal: true,
            BaseURL:  "https://api.example.com",
        }),
    ).
    Build()

// Inject ClientService into any provider or controller
type UserService struct {
    httpclient.ClientService
}

func (s *UserService) FetchUser(id string) (*User, error) {
    resp, err := s.Get("/users/" + id).Send()
    if err != nil {
        return nil, err
    }
    var user User
    return &user, resp.JSON(&user)
}
```

**`HttpClientModuleOptions`:**

| Field      | Type                  | Description                                      |
|------------|-----------------------|--------------------------------------------------|
| `IsGlobal` | `bool`                | Make `ClientService` available everywhere        |
| `BaseURL`  | `string`              | Prepended to every relative request path         |
| `Headers`  | `map[string]string`   | Default headers sent on every request            |
| `Timeout`  | `time.Duration`       | Client-level timeout for all requests            |
| `OnInit`   | `func()`              | Lifecycle hook before module injection           |

---

## Built-in Middlewares

See the [Middlewares documentation](middlewares/README.md) for full details on all options and recipes.

### CORS

```go
import "github.com/dangduoc08/ginject/middlewares"

app.BindGlobalMiddlewares(middlewares.CORS{
    AllowOrigin:        []string{"https://example.com"},
    AllowHeaders:       []string{"Content-Type", "Authorization"},
    AllowMethods:       []string{"GET", "POST", "PUT", "DELETE"},
    MaxAge:             86400_000,
    IsAllowCredentials: true,
})
```

### Helmet

Sets 13 security headers (CSP, HSTS, X-Frame-Options, etc.) with secure defaults.

```go
app.BindGlobalMiddlewares(middlewares.Helmet{})
```

### RequestLogger

```go
app.BindGlobalMiddlewares(middlewares.RequestLogger{})
```

Logs method, path, status code, response time, and request ID for every request.

---

## Exception Types

| Function                           | HTTP Status |
|------------------------------------|-------------|
| `BadRequestException`              | 400         |
| `UnauthorizedException`            | 401         |
| `ForbiddenException`               | 403         |
| `NotFoundException`                | 404         |
| `MethodNotAllowedException`        | 405         |
| `RequestTimeoutException`          | 408         |
| `ConflictException`                | 409         |
| `GoneException`                    | 410         |
| `RequestEntityTooLargeException`   | 413         |
| `UnprocessableEntityException`     | 422         |
| `InternalServerErrorException`     | 500         |
| `NotImplementedException`          | 501         |
| `BadGatewayException`              | 502         |
| `ServiceUnavailableException`      | 503         |
| `GatewayTimeoutException`          | 504         |

```go
panic(exception.NotFoundException("resource not found"))

panic(exception.BadRequestException("invalid input", exception.ExceptionOptions{
    Description: "field 'email' is required",
}))
```

---

## License

[MIT](LICENSE.md)
