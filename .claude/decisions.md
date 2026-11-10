# Decisions

Each entry records a choice the owner made where main and master differ, or where the design was open. Agents read this file before porting and never re-open a decided entry without the owner asking.

Format: `D-<n>`, then status (open / decided), context, options, decision, date.

## D-001 Middleware signature

- **Status:** open. Decide when porting step P2 (core layers).
- **Context:**
  - main: `Use(*ctx.Context, ctx.Next)`.
  - master: `Use(*http.Request, http.ResponseWriter, ctx.Next)`.
  - In both, `ctx.Next` is `func()`, so a middleware can't hand a new `*http.Request` (for example `r.WithContext(...)`) to the next layer. This is inferred from the signature and still needs a probe.
  - The owner worries the master shape is "anti-Go".
- **Options:**
  - A. A framework context, renamed `HTTPContext`.
  - B. Master's `(r, w, next)`.
  - C. A or B, plus also accepting stdlib `func(http.Handler) http.Handler`, so ecosystem middleware such as OpenTelemetry plugs in.
- **To bring:** a probe for the `next` limitation, a benchmark of each option, and a code sample of each.

## D-002 Test assertion helper

- **Status:** open. Decide when porting step P1 (internal).
- **Context:** main uses `utils.ErrorMessage`. Master deleted the exported `utils` package and moved helpers to `internal/` (`internal/test.DiffMessage`).
- **Options:** keep `utils` exported, move to `internal/`, or keep both for a while.

## D-003 Layer interfaces: typed vs `any`

- **Status:** open. Decide when porting step P2.
- **Context:**
  - main: `Guarder`, `Interceptable` and `ExceptionFilterable` are interfaces, so a wrong signature is a compile error.
  - master: they are `any`, and the method is found by reflection, so a wrong signature panics at `Create()`.
  - Master did this so that one type can serve both HTTP and WS.
- **Options:**
  - keep typed interfaces per transport;
  - keep `any`;
  - typed for HTTP now, and revisit when WS is ported.

## D-004 Devtool transport and auth

- **Status:** decided 2026-11-10.
- **Decision:**
  - Drop gRPC. Use REST, SSE or WebSocket, depending on the kind of inspection.
  - The devtool should inspect everything it can: routes, module tree, DI graph, request logs, traces.
  - Apps authenticate with an API key or token that the user gets after registering on the hosted devtool, and configures in source.

## D-005 Adapters for external backends

- **Status:** decided 2026-11-10.
- **Decision:** each adapter (Postgres, Redis, …) is a separate Go module, so core never imports a driver. Users can always write their own module with a connection string and their own methods.

## D-006 WebSocket timing

- **Status:** decided 2026-11-10.
- **Decision:** port WebSocket after the HTTP stack is solid. Until then, keep main's WS code untouched and don't extend it.

## D-007 Core port vs WS and devtool coupling

- **Status:** open. Decide when porting step P3.
- **Context:** master's `core` imports `devtool` (gRPC), `memorybroker` and `wsevent`. D-006 puts WS later, and D-004 replaces the gRPC devtool.
- **Options:**
  - a. Bring those packages in dormant.
  - b. Cut the WS and devtool paths out of core while porting. That means new red-zone code.
  - c. Put interfaces in core and leave the implementations for later.
