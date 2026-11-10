# Port plan: master → main

Rebuild v3 on `main` by porting `master` (`13242d4`) one step at a time. The goal is **ownership**: when a step is done, the owner has read and accepted every line it brought in. Run each step with `/port P<n>`.

Line counts are non-test `.go` additions from `git diff --numstat main master -- <paths>`. They're a size guide, not a promise.

## Status board

| Step | Scope | Size | Status |
|---|---|---|---|
| P0 | Agentic foundation: CLAUDE.md, `.claude/`, Makefile `check`, lint baseline, CI | config only | awaiting owner review |
| P1 | `internal/` (color, crypto, ds, num, runtime, slice, str, test). Fate of exported `utils/` (D-002) | +907 | todo |
| P2 | Leaf libs: `exception`, `aggregation`, `pattern`, `trace`, `event`, `log` | ~+1.1k | todo |
| P3 | Core, HTTP only: `ctx`, `common`, `routing`, `versioning`, `core`, `accesslog`. D-001, D-003, D-007 | ~+4.6k / −4.2k | todo |
| P4 | Baseline numbers: req/s, p50/p99, allocs/op and goroutines under load, against plain `net/http` | bench only | todo |
| P5 | `memorycache` + `modules/cache` (in-memory default, pluggable interface) | ~+0.3k | todo |
| P6 | `modules/config` | small | todo |
| P7 | Middlewares and guards: `cors`, `csrf`, `helmet`, `throttler` | ~+0.8k | todo |
| P8 | `modules/httpclient` | +1.2k | todo |
| P9 | `modules/storage`. Known data-loss bugs in `Tx`/`Compact`: decide whether to port, fix or redesign | +1.8k | todo |
| P10 | Devtool rebuilt on REST/SSE/WS with API-key auth (D-004). Not a port. Design first | design | todo |
| P11 | WebSocket: `memorybroker`, `wsevent`, WS parts of `core`/`ctx` (D-006) | ~+1.1k | later |
| P12 | Adapter modules (Postgres, Redis, …) as separate Go modules (D-005) | design | later |
| P13 | Samples and docs: README, package READMEs, examples | docs | rolling |

The order follows master's import graph: `internal` ← leaf libs ← `ctx`/`routing`/`common` ← `core`.

## Known coupling

Master's `core` imports `devtool`, `memorybroker` and `wsevent`. So P3 can't be a plain copy if WS and the gRPC devtool stay out. **D-007** (open, decide at P3) chooses how to handle this:
- (a) bring those packages in dormant;
- (b) cut the WS and devtool paths out of master's core while porting, which means writing new code in the red zone;
- (c) stub interfaces.

## Known master bugs to re-check when their step comes

These were found in the 2026-11-09 audit of master. Re-prove each one on the ported code before acting on it.

- `common.Construct` caches one instance per type name, so a second config of the same middleware or guard type is ignored (`common/fn.go`). P3/P7.
- One App per process: package-level state in `core/module.go` and `common.InsertedRoutes`. P3.
- Versioning: an unversioned route returns 410 without a version and 404 with one. P3.
- `App.Stop()` runs `OnShutdown` hooks only. It doesn't stop the server. P3.
- The throttler trusts client-sent `X-Real-IP` and `X-Forwarded-For`. P7.
- `modules/config`: `IsIgnoreEnvFile` has no effect. P6.
- `memorycache` has no size cap and no eviction. P5.
- `storage`: `Compact` drops docs written in a `Tx`, `Tx` holds no lock during its callback, and Tx writes skip hooks and watchers. P9.
- `x/net` and `grpc` vulnerabilities, and grpc pulled into core by the devtool. P3/P10.

## Definition of done for a step

1. The code is ported and adapted, and `make check` passes. Paste the real output.
2. Every main/master API divergence in scope was decided by the owner and recorded in `.claude/decisions.md`.
3. Race and concurrency evidence for shared state in scope, and before/after benchmarks for request-path code.
4. `go-reviewer` has reviewed the diff.
5. A **reading guide** (`.claude/port/P<n>-reading-guide.md`) lists the files in call order, what each does, and what to look at: locks, goroutines, allocations, panics.
6. CLAUDE.md "Architecture" is updated for anything the step changed.
7. The status is set to "awaiting owner review". The owner reads it and commits. Agents never commit.
