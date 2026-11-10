# CLAUDE.md

This file guides Claude Code (claude.ai/code) and any subagent in this repository.

## Project context

- Ginject is a NestJS-inspired dependency injection web framework for Go, built on the pattern **Module → Controller → Provider**.
- It is **open source**. The owner's startup product is the hosted **devtool** that inspects ginject apps.
- The owner hand-wrote v1. v2 (branch `master`, `13242d4`) was built with heavy AI help. It works, and its API and layer design are the target, but the owner lost ownership of the code.
- v3 rebuilds on branch `main` (created from the hand-written `org` branch at `072c3eb`) by **porting master module by module**. Every ported line must be read and accepted by the owner. See `.claude/port/PORT.md`.
- Priorities, in order:
  1. Correctness under concurrency: no data races, no deadlocks, no goroutine leaks.
  2. Performance on the request path.
  3. Backward compatibility of the user-facing API once a piece is ported.
  4. Ship as a single static Go binary.
- Built-in modules (cache, database, …) default to in-memory and stay pluggable. External backends (Postgres, Redis, …) come as **separate Go modules** (adapters), so the core never pulls their drivers.

## Commands

```bash
make check         # definition of done: fmt-check + vet + lint + race tests + tidy-check
make test          # go test ./... -cover -race
make lint          # golangci-lint (.golangci.yml, only reports issues introduced after 072c3eb)
make vuln          # govulncheck
make deps          # go + yarn dependencies
make tidy          # go mod tidy

go test ./routing/... -run TestRouteAdd -v
cd sample && go run main.go
```

## Ownership rules

These are the strictest rules in this file. They apply to every agent and subagent.

- **Only the owner commits.** Agents never run `git commit`. Leave the change in the working tree and report. The owner reads it and runs `/commit`.
- **Readable diffs.** Keep each change small enough to read in about 10 minutes: roughly 300 lines of non-test library code. When a piece can't be split and still compile (for example the core port), deliver it with a reading guide that splits it into ordered chunks.
- **Every change report has three lines:** what changed, why, and the exact command that verifies it.
- **Red zone:** `core/`, `common/`, `ctx/` and `routing/`. Agents prove problems and propose changes there. The owner approves every line.
- **No unrequested scope.** Don't add features, packages or refactors that aren't in `.claude/port/PORT.md` or asked for.
- **API divergences are the owner's call.** When main and master differ in user-facing API or layer behaviour, stop. Present a comparison table with code samples, and benchmarks when the hot path is involved. Wait for the decision, then record it in `.claude/decisions.md`. Master is the default target, but never assume it.

## Working rules

- **Keep the existing code conventions.** They are the owner's style. Don't modernise or restyle code you aren't changing.
- **No comments in Go code you write.** This covers doc comments, inline comments and test comments. Let names carry the meaning, and put rationale in the report. Don't strip comments that already exist, including in code ported from master.
- **Tests:** follow the helper used in the package's existing tests. On main that is `t.Error(utils.ErrorMessage(actual, expected, "desc"))`. Master's `internal/test.DiffMessage` replaces it only once decision D-002 is made.
- **Concurrency claims need evidence.** Run `-race`, write a concurrent test, or record goroutine counts before and after. "Should be safe" is not evidence.
- **Performance claims need numbers.** Run `go test -bench . -benchmem -count 5` before and after, on the same machine, and report both.
- **Done means `make check` passes.** Report its real output. Never claim a pass you didn't run.
- **Verify before you write docs.** The code is the source of truth. Master's docs (`git show master:CLAUDE.md`, `master:.ginject/`) describe master, not main. Use them as hints, never as facts about main.
- **Commits:** through `/commit` only, in Conventional Commits English with no AI attribution trailers. Never use `--no-verify`.
- **Exported method names with underscores** (`READ_BY_id`, `SUBSCRIBE_...`) are the routing DSL, not style violations.

## Architecture (main)

This section describes the code that is on this branch now. Update it in the same change that ports a piece which alters it.

- **Provider:** a struct implementing `NewProvider() Provider` (`core/provider.go`).
- **Controller:** a struct implementing `NewController() Controller` (`core/controller.go`). It embeds `common.REST` for HTTP or `common.WS` for WebSocket.
- **Module:** `core.ModuleBuilder().Imports(...).Controllers(...).Providers(...).Build()`, plus `Module.Prefix(...)` (`core/module_builder.go`, `core/module.go`).
- **App:** `core.New()`, then `app.Create(rootModule)`, then `app.Listen(port)` (`core/app.go`).
- **Request context:** `*ctx.Context`, which embeds `*http.Request` and `http.ResponseWriter` (`ctx/context.go`). `ctx.Next` is `func()`.
- **Layers:** the interfaces are typed, so a wrong signature is a compile error.
  - `common.MiddlewareFn`: `Use(*ctx.Context, ctx.Next)`
  - `common.Guarder`: `CanActivate(*ctx.Context) bool`
  - `common.Interceptable`: `Intercept(*ctx.Context, *aggregation.Aggregation) any`
  - `common.ExceptionFilterable`: `Catch(*ctx.Context, *exception.Exception)`
  - Each is bound on the controller with `BindMiddleware`, `BindGuard`, … and globally with `app.BindGlobal*`.
- **Routing DSL:** `common.ParseFnNameToURL` (`common/fn.go`).
  - Operations (`common.RESTOperations`): `READ`, `CREATE`, `UPDATE`, `MODIFY`, `DELETE`, `PREFLIGHT`, `SERVE`.
  - Tokens: `BY`, `AND`, `OF`, `ANY`, `FILE`, `VERSION`.
  - Routes are stored in a trie (`routing/`).
- **Devtool:** `app.EnableDevtool()`, served over gRPC (`devtool/`). It is planned to be replaced by REST/SSE/WebSocket, see PORT.md.

## Subagents and commands

- **Agents** (`.claude/agents/`):
  - `go-reviewer`: read-only diff review.
  - `test-writer`: edits `_test.go` only.
  - `doc-verifier`: read-only claim check.
- **Commands** (`.claude/commands/`):
  - `/port <step>`: port one PORT.md step from master.
  - `/commit`
  - `/check-staged`
  - `/sync-docs`
- A hook runs gofmt after every Go file edit.
