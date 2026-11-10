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

- **Go version:** the repo is on Go 1.27.3 (`go.mod`; CI reads it via `go-version-file`). Use current idioms: `for range n`, `sync.WaitGroup.Go`, `b.Loop`, and the `slices`/`maps`/`strings` helpers. `go fix -diff ./...` lists the places modernizers would change.

- **Keep the existing code conventions.** They are the owner's style. Don't modernise or restyle code you aren't changing.
- **Where functions live in a package:**
  - Exported API goes in files named after their topic: `str.go`, `trie.go`, `router.go`.
  - An unexported helper used by two or more files goes in `fn.go`, with its tests in `fn_test.go`.
  - An unexported helper used by only one file stays in that file, placed after the function that calls it.
  - A package with a single code file has no `fn.go`. Everything stays in that file.
  - When a second file starts using a helper, move the helper and its tests to `fn.go`/`fn_test.go`.
- **Names are the documentation**, because there are no comments.
  - Use descriptive parameter names (`kind`, not `t`).
  - Don't shadow builtins (`close`, `len`, `new`) or the package's own name. In package `str`, a string parameter is `s`, following the standard library.
  - Fix typos in identifiers.
  - Package names may describe a future scope rather than the current content. For example `internal/crypto` (hashing will join UUID) and `internal/ds` (more data structures will come), plus `Trie.Raw` (the original route string), are owner decisions (D-021).
- **No comments in Go code you write.** This covers doc comments, inline comments and test comments. Let names carry the meaning, and put rationale in the report. Don't strip comments that already exist, including in code ported from master.
- **Tests:** use the helper that the package's existing tests already use (D-002).
  - Packages ported from master use `t.Error(test.DiffMessage(actual, expected, "desc"))` from `internal/test`.
  - Packages not yet ported keep `utils.ErrorMessage`.
  - Exception: `internal/test`'s own tests use `t.Errorf`, because `DiffMessage` can't assert on its own output.
- **Concurrency claims need evidence.** Run `-race`, write a concurrent test, or record goroutine counts before and after. "Should be safe" is not evidence.
- **Performance depends on the path.** Correctness and race-freedom always come first.
  - **Hot path:** code that runs on every request, such as route lookup and the trie, `ctx` creation, binding, the middleware/guard/interceptor/filter pipeline, response writing, and per-request logging.
    - Optimise for speed. Minimise allocations, and keep `fmt`, reflection, regexp and `defer` out. Do that work once at `Create()` time instead.
    - The owner must still be able to understand every optimisation, and the reading guide explains each one.
    - **Take every measurable gain, however small.** Small costs add up across the pipeline: ten functions at 14 ns each is 140 ns per request. A gain counts if it is stable across repeated runs or removes an alloc, and it needs no per-function threshold. This also applies to warm code that a client can trigger.
    - Every fast path needs a test proving it equals the slow path, for example over random inputs.
    - Defer only when the code will certainly be deleted or rewritten in a planned step. Record each deferral in PORT.md with a target number.
    - P4 measures the whole request end to end, so the cumulative cost stays visible.
  - **Cold path:** everything else, such as bootstrap, `Create()`, startup logging, error formatting and rarely called helpers.
    - Write for readability and ownership first. Only avoid obvious waste, like compiling a regexp on every call.
  - **Heavy techniques need the owner's approval for each use, before the code is written.** These are `unsafe`, `sync.Pool`, hand-managed buffers and restructuring code to make it inline. Bring a benchmark showing the gain and explain the constraint the technique brings.
  - **Type A vs type B.**
    - **Type A** is free (no readability cost, no extra branch) and applies everywhere, whatever the path: no regexp compiled per call, no needless alloc or copy, no `Sprintf` for plain concatenation.
    - **Type B** costs something: a fast path, `unsafe`, `sync.Pool`, manual buffers. It applies only on proven hot code, or on warm code a client can trigger.
    - If the class is undetermined, do type A now and log type B in `.claude/roadmap/ROADMAP.md`.
  - For input a client controls, bound the work with limits first.
  - Propose every optimisation before writing it, with expected numbers. Never optimise first and explain afterwards.
  - Benchmarks use `for b.Loop() { ... }` and call `b.ReportAllocs()`. Since Go 1.26, `b.Loop` keeps the results alive without blocking inlining, so no sink variable is needed. `b.RunParallel` is the exception and still needs a package-level sink.
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
- **Internal helpers (`internal/`, ported in P1):**
  - The packages are `color`, `crypto` (`UUID`), `num`, `slice`, `str`, `test` (`DiffMessage`) and `ds` (the route trie).
  - Code outside the module can't import them.
  - Nothing calls them yet. Callers move off the exported `utils` package as their own packages are ported, and `utils` is deleted at the end of P3 (D-002).
  - `ds.Trie` has no locking. `Insert` while `Find` runs is a data race (proven with `-race`), so the trie must be fully built before serving.
- **Devtool:** `app.EnableDevtool()`, served over gRPC (`devtool/`). It is planned to be replaced by REST/SSE/WebSocket, see PORT.md.

## Planning files

- `.claude/decisions.md` is **permanent**. Every owner decision goes here (D-001, …).
- `.claude/roadmap/ROADMAP.md` is **permanent**. It holds product work (devtool, WS, adapters), the performance audit backlog for undetermined functions, and recurring audits.
- `.claude/port/` is **temporary**. It holds the port plan and reading guides, and is deleted at P14 after its durable knowledge has moved into package READMEs or this file.

## Subagents and commands

- **Agents** (`.claude/agents/`):
  - `go-reviewer`: read-only diff review.
  - `test-writer`: edits `_test.go` only.
  - `doc-verifier`: read-only claim check.
- **Commands** (`.claude/commands/`):
  - `/port <step>`: port one PORT.md step from master.
  - `/pair-review <package>`: review a package with the owner across six lenses, all backed by probes: performance (a hot/warm/cold map the owner confirms), concurrency, security, correctness and errors, resources, and readability and conventions. Exported packages also get an API-surface lens.
  - `/commit`: runs `make check` first and warns when staged files also have unstaged edits. It summarises the staged diff (behaviour change, API change, and which D-xxx it implements) and recommends `go-reviewer` for unreviewed Go code. It then proposes Conventional Commit messages and commits only after the owner confirms.
  - `/sync-docs`
- A hook runs gofmt after every Go file edit.
