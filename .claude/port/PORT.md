# Port plan: master → main

This folder is **temporary**. It is deleted at P14. Long-lived items go to `.claude/roadmap/ROADMAP.md`, and decisions go to `.claude/decisions.md`.

Rebuild v3 on `main` by porting `master` (`13242d4`) one step at a time. The goal is **ownership**: when a step is done, the owner has read and accepted every line it brought in. Run each step with `/port P<n>`.

Line counts are non-test `.go` additions from `git diff --numstat main master -- <paths>`. They're a size guide, not a promise.

## Status board

| Step | Scope | Size | Status |
|---|---|---|---|
| P0 | Agentic foundation: CLAUDE.md, `.claude/`, Makefile `check`, lint baseline, CI | config only | done |
| P1 | `internal/` (color, crypto, ds, num, slice, str, test). D-002, D-008, D-009, D-010 | ~+800 | awaiting owner review |
| P2 | Leaf libs: `exception`, `aggregation`, `pattern`, `trace`, `event`, `log` | ~+1.1k | todo |
| P3 | Core, HTTP only: `ctx`, `common`, `routing`, `versioning`, `core`, `accesslog`. D-001, D-003, D-007 | ~+4.6k / −4.2k | todo |
| P4 | Baseline numbers: req/s, p50/p99, allocs/op and goroutines under load, against plain `net/http` | bench only | todo |
| P5 | `memorycache` + `modules/cache` (in-memory default, pluggable interface) | ~+0.3k | todo |
| P6 | `modules/config` | small | todo |
| P7 | Middlewares and guards: `cors`, `csrf`, `helmet`, `throttler` | ~+0.8k | todo |
| P8 | `modules/httpclient` | +1.2k | todo |
| P9 | `modules/storage`. Known data-loss bugs in `Tx`/`Compact`: decide whether to port, fix or redesign | +1.8k | todo |
| P10–P12 | Devtool, WebSocket and adapters are long-lived product work, not ports. They moved to `.claude/roadmap/ROADMAP.md` (R-1..R-3) | — | moved |
| P13 | Samples and docs: README, package READMEs, examples | docs | rolling |
| P14 | Wrap up the port: follow "When the port is finished" in `.claude/roadmap/ROADMAP.md`, then delete `.claude/port/` | docs | todo |

The order follows master's import graph: `internal` ← leaf libs ← `ctx`/`routing`/`common` ← `core`.

## Known coupling

Master's `core` imports `devtool`, `memorybroker` and `wsevent`. So P3 can't be a plain copy if WS and the gRPC devtool stay out. **D-007** (open, decide at P3) chooses how to handle this:
- (a) bring those packages in dormant;
- (b) cut the WS and devtool paths out of master's core while porting, which means writing new code in the red zone;
- (c) stub interfaces.

## Known master bugs to re-check when their step comes

These were found in the 2026-11-09 audit of master. Re-prove each one on the ported code before acting on it.

- `common.Construct` caches one instance per type name, so a second config of the same middleware or guard type is ignored (`common/fn.go`). P3/P7.
- Route trie (`internal/ds`):
  - Backtracking was fixed in P1 (D-020).
  - `search` picks the first route that matches by path, without checking the method; master's greedy walk does the same. At P3, check this against `Router.Match`'s method and version lookup.
  - The trie still has no lock, so `Insert` during `Find` is a data race. master only inserts during `Create()`, which was verified in P1. At P3, keep it that way, or guard any runtime route registration.
- Request ID (`ctx/context.go` in master, proven in P1):
  - `SetID` builds a full `crypto.UUID` and keeps only the first 8 hex chars, which is 32 random bits. That gives a 50% chance of a collision after about 77k requests, and devtool traces depend on these IDs.
  - It also costs about 430–470 ns per call.
  - Under parallel load, `crypto/rand` on macOS gets slower per op as CPUs are added: 431 ns with 1 CPU, 1178 ns with 12, which points to a global lock. Linux is untested.
  - Redesign the request ID for the hot path in P3.
  - **P3 target (agreed in P1):** stop calling `crypto.UUID` for the request ID and build it from `math/rand/v2`. Request IDs need uniqueness, not cryptographic strength.
    - Measured: 2× `Uint64` is 13.6 ns (2.5 ns parallel), against 346–391 ns for 16 bytes of `crypto/rand`.
    - The target is about 60 ns with no contention across CPUs and well over 32 bits of randomness.
    - `crypto.UUID` stays crypto-grade, because tokens use it.
- Router.Match target (P3): stop calling `Enclose` on the request path (`path.Clean` already collapses `//`). Let the trie accept paths without a trailing `/`, so that matching costs **0 allocs**. Today the request-path `Enclose` costs 59.2 ns and 1 alloc after P1's fast path (83.1 ns before).
- Request path loses its spaces (proven in P1): master's `Router.Match` runs `str.Enclose(path.Clean(route), '/')` on every request, and `Enclose` strips whitespace. `Enclose("/us er", '/')` gives `"/user/"`, so `GET /us%20er` matches the `/user` route. `Enclose` is right for route patterns; the caller is wrong for request paths. Also measure the per-request cost of `Enclose` plus `path.Clean` (`Enclose` alone is about 84 ns and 1 alloc). P3.
- JSONP callback XSS (`ctx.JSONP` in master): the callback name from `?callback=` only has its spaces removed (`str.RemoveSpace`) before being written into the response. `?callback=alert(1);//` goes through. Validate it as a JS identifier. P3.
- Logging hot path (P2): with access log on, every request goes `accesslog` → `logger.Info` → `PrettyHandler.Handle` (master `log/log_handler.go:95`).
  - `Handle` takes a **global mutex per record** (`h.mu.Lock()`) and walks attrs with **reflection**. Review both as hot path.
  - The `color` package stays cold, because `NewPrettyHandler` precomputes the coloured level labels once, and `Handle` writes them plus `log`'s own ANSI constants.
  - Don't dedupe by calling `color.Fmt*` from `Handle`. Instead, export ANSI constants from `color` and reuse them (type A).
- Numeric binding (P3, from P1's `num` review):
  - `BindStruct` must handle `F64ToAnyNum`'s `ok == false` and reject the request with 400 (D-018).
  - Set struct fields with `reflect.Value.SetInt/SetUint/SetFloat` instead of boxing into `any` and calling `reflect.ValueOf`. That removes 1 alloc per numeric field.
  - JSON is decoded into `any`, so integers above 2^53 lose precision before binding: `9007199254740993` becomes `9007199254740992`. Use `json.Decoder.UseNumber` or `encoding/json/v2`.
- Query/form/header/param binding (P3, from P1's `slice` review):
  - `BindStrArr` must turn `StrParse*`'s `ok == false` into 400 (D-019).
  - Scalar fields also ignore `strconv` errors (`val, _ = strconv.ParseBool(...)`), so `?page=abc` becomes `0`. Reject those too.
- `ctx.Body.Get/Set` (P3, hot user API): every call does `strings.Split` plus `slice.Filter`, costing about 108 ns and 2 allocs. Walk the key path with `strings.Cut` for 0 allocs.
- One App per process: package-level state in `core/module.go` and `common.InsertedRoutes`. P3.
- Versioning: an unversioned route returns 410 without a version and 404 with one. P3.
- `App.Stop()` runs `OnShutdown` hooks only. It doesn't stop the server. P3.
- The throttler trusts client-sent `X-Real-IP` and `X-Forwarded-For`. P7.
- `modules/config`: `IsIgnoreEnvFile` has no effect. P6.
- `memorycache` has no size cap and no eviction. P5.
- `storage`: `Compact` drops docs written in a `Tx`, `Tx` holds no lock during its callback, and Tx writes skip hooks and watchers. P9.
- grpc is pulled into core by the devtool. P3 (devtool redesign: R-1). (The `x/net` and `grpc` vulnerabilities are gone: govulncheck v1.9.0 reports none after the Go 1.27 bump to grpc v1.85.0 and x/net v0.62.0.)

## Definition of done for a step

1. The code is ported and adapted, and `make check` passes. Paste the real output.
2. Every main/master API divergence in scope was decided by the owner and recorded in `.claude/decisions.md`.
3. Race and concurrency evidence for shared state in scope, and before/after benchmarks for request-path code.
4. `go-reviewer` has reviewed the diff.
4b. **Step gate:** the step-level lenses have been checked and reported:
   - **dependencies and the single static binary:** new deps justified (size, license, vulns), `CGO_ENABLED=0 go build ./...` works, and nothing assumes a 64-bit `int`;
   - **devtool readiness:** each component can expose inspectable data (routes, module/DI graph, pipeline timings) at near-zero cost when devtool is off;
   - **ecosystem compatibility:** `net/http` interop, `r.Context()` propagation, and safe defaults (server timeouts, header and body limits).
5. A **reading guide** (`.claude/port/P<n>-reading-guide.md`) lists the files in call order, what each does, and what to look at: locks, goroutines, allocations, panics. A faithful Vietnamese translation sits next to it as `P<n>-reading-guide.vi.md`.
6. CLAUDE.md "Architecture" is updated for anything the step changed.
7. The status is set to "awaiting owner review". The owner reads it and commits. Agents never commit.
