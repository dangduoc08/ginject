# RequestLogger Middleware

*`requestlogger` logs a structured summary of every finished HTTP or WebSocket request as a Ginject middleware.*

- [RequestLogger Middleware](#requestlogger-middleware)
  - [Key Features](#key-features)
  - [Usage](#usage)
  - [How It Works](#how-it-works)
  - [`RequestLogger` Struct](#requestlogger-struct)
    - [Logger](#logger)
  - [`RequestLogger` Methods](#requestlogger-methods)
    - [Use](#use)
  - [Benchmarks](#benchmarks)

## Key Features
- Logs once per request, after the response has actually finished, by subscribing to the `ctx.RequestFinished` broker event instead of wrapping `next` directly
- One log line per HTTP request with URL, method, status, response time, protocol, user agent, and request ID
- One log line per WebSocket event with the event name, response time, subprotocol, and user agent
- Delegates the actual write to whatever `common.Logger` implementation is configured — by default, Ginject's own structured logger

## Usage

```go
package main

import (
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/middlewares/requestlogger"
)

func main() {
	app := core.New()

	app.BindGlobalMiddlewares(requestlogger.RequestLogger{})

	app.Create(
		core.ModuleBuilder().Build(),
	)
}
```

`RequestLogger` also works scoped to a single controller, since it satisfies `common.MiddlewareFn` directly:

```go
func (c APIController) NewController() core.Controller {
	c.BindMiddleware(requestlogger.RequestLogger{})
	return c
}
```

## How It Works

`RequestLogger` embeds `common.Logger` rather than wrapping it in a private field, so the struct itself exposes `Debug`/`Info`/`Warn`/`Error`/`Fatal`. When bound via `BindGlobalMiddlewares` or `BindMiddleware`, Ginject's DI container injects the app's configured `common.Logger` (the `log` package's logger by default, or whatever was passed to `app.UseLogger`) into that embedded field automatically — you don't set it yourself in application code. When constructing a `RequestLogger` directly, as the package's own tests do, you must supply a `Logger` explicitly, since the embedded interface is otherwise `nil`.

`Use` doesn't log synchronously: it subscribes a handler to the `ctx.RequestFinished` broker event, then immediately calls `next`. The subscribed handler fires once the broker publishes `ctx.RequestFinished` for that context (after the response is actually sent), and logs an HTTP or WebSocket line depending on `ctx.HTTPContext.GetType()`.

## `RequestLogger` Struct

### Logger
Type: `common.Logger` (embedded)

Default: `nil`

Required: `false` when bound through `BindGlobalMiddlewares`/`BindMiddleware` (auto-injected by Ginject's DI container); otherwise must be supplied explicitly to avoid a nil-interface panic when a request finishes.

```go
requestlogger.RequestLogger{Logger: myLogger}
```

## `RequestLogger` Methods

### Use

Subscribes a one-shot logging handler to the `ctx.RequestFinished` event for the current context, then calls `next`.

#### Rules
- `next` is always called (`TestRequestLogger_Use_CallsNext`).
- Nothing is logged until `ctx.RequestFinished` is published for that context — calling `Use` alone does not log (`TestRequestLogger_Use_NoLogWithoutEventEmit`).
- For an HTTP context, once `ctx.RequestFinished` is published, exactly one `Info` call is made with the request's URL path as the message, and `Method`, `Status`, `Time`, `Protocol`, `User-Agent`, and the `ctx.RequestID` key/value pairs as args (`TestRequestLogger_Use_HTTPLogsURL`, `TestRequestLogger_Use_HTTPLogsMethod`, `TestRequestLogger_Use_HTTPLogsStatus`, `TestRequestLogger_Use_HTTPLogsProtocol`, `TestRequestLogger_Use_HTTPLogsRequestID`).
- The `Time` arg is the response time formatted as a Go `time.Duration` string ending in `"ms"` for sub-second latencies (`TestRequestLogger_Use_HTTPLogsTime`).
- A context whose type is neither the HTTP nor the WebSocket type (e.g. an empty `Type`) logs nothing when `ctx.RequestFinished` is published (`TestRequestLogger_Use_UnknownTypeNoLog`).

#### Parameters
- 1st parameter: `*ctx.HTTPContext` (`c`)

- Description: The current request context; its `Broker` is subscribed to for the finished-request event.

- 2nd parameter: `ctx.Next` (`next`)

- Description: Called to pass control to the next handler in the chain.

#### Returns
None.

#### Usage

```go
requestlogger.RequestLogger{Logger: myLogger}.Use(c, next)
```

## Benchmarks

Captured by running `go test -run=^$ -bench=. -benchmem ./middlewares/requestlogger/...`. Numbers are machine-dependent and were captured at doc-generation time — re-run the command yourself for a fresh baseline.

```console
goos: darwin
goarch: amd64
pkg: github.com/dangduoc08/ginject/middlewares/requestlogger
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkRequestLogger_Use_HTTP-12            	  214678	      5300 ns/op	    7147 B/op	      36 allocs/op
BenchmarkRequestLogger_Use_RegisterOnly-12    	 1000000	      1378 ns/op	     343 B/op	       4 allocs/op
PASS
ok  	github.com/dangduoc08/ginject/middlewares/requestlogger	4.692s
```
