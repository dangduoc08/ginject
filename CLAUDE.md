# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make test          # run all tests with coverage
make lint          # run golangci-lint
make deps          # install go + yarn dependencies
make tidy          # go mod tidy
make protoc        # regenerate protobuf from devtool/*.proto

# run a single test
go test ./routing/... -run TestRouter -v

# run the sample app
cd sample && go run main.go
```

## Architecture

Ginject is a NestJS-inspired dependency injection web framework for Go. The core pattern is **Module → Controller → Provider**.

### DI Container (`core/`)

- **Provider**: any struct implementing `NewProvider() Provider`. Providers are injected into other providers and controllers via struct field types (reflection-based, no tags needed).
- **Controller**: any struct implementing `NewController() Controller`. Must embed `common.REST` (HTTP) or `common.WS` (WebSocket) to register routes.
- **Module**: built with `core.ModuleBuilder().Imports(...).Controllers(...).Providers(...).Build()`. Modules compose the app tree. Set `IsGlobal: true` to make providers available everywhere.
- **App**: created via `core.New()`, wired by `app.Create(rootModule)`, started with `app.Listen(port)`.

### REST Route Convention

Controller method names are parsed into HTTP routes — no annotations needed. Naming tokens: `READ`→GET, `CREATE`→POST, `UPDATE`→PUT, `MODIFY`→PATCH, `DELETE`→DELETE, `PREFLIGHT`→OPTIONS. Separator tokens: `BY` (path param), `AND` (additional segment), `OF` (sub-resource), `ANY` (wildcard), `VERSION` (version tag), `FILE` (file serving).

Example: `READ_BY_ID_AND_NAME` → `GET /:id/:name`

Versioning adds a `_VERSION_X` suffix: `READ_VERSION_1` → version 1 of that route.

### Request Pipeline (per route)

```
GlobalMiddlewares → ModuleMiddlewares → Guards → Interceptors → MainHandler → ExceptionFilters
```

Each layer is bound with `BindMiddleware(fn, handlers...)`, `BindGuard(fn, handlers...)`, etc. on the `common.Middleware`, `common.Guard`, `common.Interceptor`, `common.ExceptionFilter` embedded fields of a controller.

Interfaces to implement:
- `MiddlewareFn`: `Use(*ctx.Context, ctx.Next)`
- `Guarder`: `CanActivate(*ctx.Context) bool`
- `Interceptable`: `Intercept(*ctx.Context, *aggregation.Aggregation) any`
- `ExceptionFilterable`: `Catch(*exception.Exception, *ctx.Context)`

### Handler Injection

Handler method parameters are injected by type — declare them in the handler signature and the framework resolves them from the request context. Types available: `*ctx.Context`, `*http.Request`, `http.ResponseWriter`, `ctx.Body`, `ctx.Query`, `ctx.Param`, `ctx.Header`, `ctx.Form`, `ctx.File`, `ctx.Next`, `ctx.Redirect`, `ctx.WSPayload`.

All types are re-exported from the root `ginject` package (`aliases.go`).

### Modules: Static vs Dynamic

- **Static modules**: `var MyModule = func() *core.Module { ... }` — singleton, created once.
- **Dynamic modules**: a struct with a `New(...)` factory — instantiated with arguments, used for configurable modules like `modules/config`.

### Built-in Modules (`modules/`)

- `modules/config`: `.env` file loader with typed struct binding. Register as global module, inject `ConfigService` to read values.
- `modules/cache`: in-memory LFU cache implementation.

### WebSocket

Controllers embed `common.WS` instead of `common.REST`. Method names map to event names. The framework handles the `websocket.Conn` lifecycle; handlers receive `ctx.WSPayload` for incoming events.

### Bootstrap-time Error Convention

Build/config failures (route conflicts, unresolved dependencies, invalid handlers) in `core/`, `common/`, `routing/` are reported via `panic()`, not returned as `error` — there is no error-return API (`Build()`/`Create()` return nothing). This only fires during `app.Create()`'s module/controller wiring, never on the request path, so an uncaught panic there is fail-fast by design.

Message format is `"<category>: <detail>"`:
- `"route conflict: ..."` (`common/rest.go`)
- `"event conflict: ..."` (`common/ws.go`)
- `"invalid handler: ..."` (`routing/router.go`)
- `"invalid module: ..."` (`core/module_builder.go`)
- `"dependency injection: ..."` (`core/fn.go`)

These are wrapped in `color.FmtRed(...)` since they only ever surface on a developer's terminal at startup. If an `err` you're panicking already came from a function that applies `color.FmtRed`, just `panic(err)` — don't re-wrap, or the ANSI codes nest and garble the output. The one exception is `invokeHandlerByProviders` in `core/fn.go`, which panics during per-request handling and is caught by `core/http.go`'s `recover()` — it stays a plain, uncolored `fmt.Errorf`.
