# Roadmap

This is long-lived work. Unlike `.claude/port/`, which is deleted once the port is done, this file stays. Decisions live in `.claude/decisions.md`.

## Product work

| ID | Item | Notes | Status |
|---|---|---|---|
| R-1 | Devtool rebuilt on REST/SSE/WS with API-key auth. `trace.Event` is its data model and today has only `Duration`. It needs a start timestamp, and parent/span IDs, to draw request timelines; adding them is additive | D-004. It inspects routes, the module tree, the DI graph, request logs and traces. The startup product sits on top of it. Design first. Was P10. | todo |
| R-2 | WebSocket stack | D-006. Implement D-030 (glob syntax: `*` one, `**` one or more at the end). Validate topics: reject empty segments (`a..b`, `b.`) and bound the topic length. Ports `memorybroker`, `wsevent` and the WS parts of `core`/`ctx`, after HTTP is solid. Rename `WSConfig` → `WSOptions` (D-032). Was P11. | later |
| R-3 | Adapter modules (Postgres, Redis, NATS, MQTT, …) | D-005. Pub/sub adapters translate D-030 patterns to the native superset and filter locally with `pattern.Match`; see the table in D-030. Each is a separate Go module, so the core never imports a driver. Was P12. | later |
| R-4 | Opt-in async log writer | D-031 C2. Wraps the writer with a buffered channel or ring buffer, never blocks the request, reports dropped records through a callback, and flushes in `Fatal` and at shutdown (like zerolog `diode` or zap `BufferedWriteSyncer`). Multiple outputs (the removed `LogOptions.Transports`) belong here too. | later |

## Feature backlog

Smaller improvements found during reviews, waiting for a step or a need. Move a row into PORT.md or a product item when it is scheduled.

| ID | Package | Item | Why | Trigger |
|---|---|---|---|---|
| B-1 | `log`, `common` | Request-scoped fields: a context-aware `common.Logger` (or `With`), so every line carries the request ID and trace ID | The main gap for production: lines of one request can't be correlated. Devtool (R-1) needs it too. Red zone (`common.Logger`) | Decide at P3 with `ctx` and `core` |
| B-2 | `log` | `LogOptions.Writer io.Writer` (default `os.Stdout`) | Output is fixed to stdout: hard to test, no file, no stderr. With a user writer, `lockedWriter.write` must unlock on panic (it has no `defer`, on purpose for the hot path) | When a user or a test needs it |
| B-3 | `log` | Accept `slog.Leveler` for `LogOptions.Level`, so `slog.LevelVar` can change the level at runtime | Turn Debug on temporarily, for example from devtool | With R-1 |
| B-4 | `log` | Sampling or rate limiting for repeated records | Client-triggered errors can flood a synchronous writer and slow every request (D-031 C2) | With R-4 |
| B-5 | `log`, `core` | `Fatal` runs the shutdown hooks before `os.Exit` | `os.Exit` skips graceful shutdown and `OnShutdown` | P3, when shutdown is ported |
| B-6 | `log` | Mask secrets inside messages and error text, not only by key | A token in a path (`/reset/{token}`) or an error string is logged as is. The access log logs `URL.Path` only, without the query | When there is a concrete case |
| B-7 | `log` | Log the `%w` chain (and optionally a stack) of errors as fields | Only `Error()` is printed today | With R-1 |
| B-8 | `log` | Export the `logFormat` type, or make the format constants a public type | Users can't declare a config field of that type | Before the first release |
| B-9 | `log` | ~~Fuzz tests for `writeEscaped` and masking~~ Done in P2 (D-031): three fuzz targets. Extend them when masking or the handler changes | Both take client-controlled input | Done |
| B-10 | `log` | Keep struct field order through masking (an ordered result instead of a map) | `WrapLogger` turns structs into maps, so Pretty prints fields sorted, not in declaration order | If users ask for it; it is a behaviour change |
| B-11 | `log` | Bound top-level `slog.Group` values and top-level `LogValuer`s by depth and node budget, in masking (`maskAttr`/`maskSlogValue`) and in the pretty handler (`appendFlattened`) | Nested ones are bounded. A top-level 5000-attr group is walked whole, and a self-referencing `LogValuer` passed directly as an argument (including one that returns `AnyValue(slog.Any("a", self))`) hangs or overflows the stack (stdlib Text/JSON overflow the same way) | If a LogValue builds groups from client data, or before the first release |
| B-12 | `log` | Call `LogValue()` once per record at the top level | Masking resolves a top-level `LogValuer`, and when nothing changes the handler resolves it again, so `LogValue()` runs twice. Forwarding the resolved value costs an args clone | If a costly or side-effecting `LogValue` shows up |
| B-13 | `log` | Keep duplicate keys when masking merges an empty-key group into a map | `{a: "outer", "": {a: "inner"}}` keeps only `a: inner`, while stdlib JSON prints both. Nothing leaks, every key goes through the rules | With B-10 |
| B-14 | `log` | Unwrap pointers in `slogValueOf`, so `*slog.Attr` and `*slog.Value` (top level, in maps, `[]*slog.Attr`) are masked like their values | They are `fmt.Stringer`s, so they print `password=…` in Text and Pretty. The README row "Stringer: only as a whole, by key" covers it, and slog never produces these pointers itself | Next change to masking |
| B-15 | tooling | Widen `make conventions`: D-032 parameters on continuation lines of a `func`, and D-002 helpers `f.Fatalf`, `tb.Errorf`, `t.Error(err)`, `t.Error(fmt.Errorf(…))` | The checks are heuristics; `go-reviewer` still sweeps each step (D-033) | When a miss shows up in a review |

## Performance audit backlog

These are functions whose hot/warm/cold class couldn't be settled from the code alone (`/pair-review` step 1). Every row says why the class is unknown, how to find out, and when to look again.

**Until a row is resolved:**
- **Type A** optimisations are applied anyway. They are free: no readability cost and no extra branch.
- **Type B** optimisations wait for evidence. They have a cost: a fast path, `unsafe`, `sync.Pool`, manual buffers.

| Function | Why the class is unknown | How to find out | Revisit at | Pending type B idea |
|---|---|---|---|---|
| `crypto.UUID` on Linux | Cost measured only on macOS: 401 ns, 978 ns at 12 CPUs, dominated by `crypto/rand`. Go 1.24+ uses vDSO `getrandom` on Linux, which may be far cheaper | Run `go test ./internal/crypto -bench . -cpu 1,4,12` on Linux (CI runner) | P4 | None for `UUID` itself (it must stay crypto-grade). Hot callers (request ID) move to `math/rand/v2` in P3 |
| `log` masking of slices and large structs (`maskValue`) | Hot only when user code logs slices or big structs with `MaskFields` set; how often depends on the app. 10 tagged structs in a slice with 1 rule: 11 µs, 145 allocs, about 1 alloc per field for the path string (D-031) | Profile the sample app under load with `MaskFields` on and a handler that logs a slice of DTOs | P4 | Build paths in one reused byte buffer (append the key, match, truncate back) instead of concatenating a string per node |
| `log` `mapKeyString` and the `writeScalar` fallback | `fmt.Sprint`/`fmt.Fprint` run only for non-string map keys (`map[int]…`) and exotic kinds (complex, chan, func), which logs rarely contain | Count non-string-key maps in real access and app logs under load | P4 | `strconv` fast path for int, uint and bool keys, with a test that it equals `fmt.Sprint` |

Ways to find out:
- `go test -covermode=count` over N requests through `ServeHTTP`: a line executed N times is hot.
- A `pprof` CPU profile of the sample app under load (`wrk`/`vegeta`).
- Benchmarks with each config toggled (versioning, WS, access log) and with the largest input a client can send.

## Recurring audits

| Audit | Trigger | How |
|---|---|---|
| End-to-end request cost | After each change to the request pipeline, and after P4 | `ServeHTTP` benchmarks plus `pprof`, compared with the P4 baseline |
| Vulnerabilities | Monthly and on every dependency change | `make vuln`; the CI `vuln` job |
| Go upgrade | Each Go minor release | Read the release notes, bump `go.mod`/golangci-lint/govulncheck, run `go fix -diff ./...`, re-run benchmarks |
| Goroutine leaks | When concurrency code changes | `goroutineleak` profile (Go 1.27) in tests |
| Release readiness | Before each release | godoc on the exported API, runnable `Example`s, `/sync-docs` over every README, a `make vuln` pass, and a `CGO_ENABLED=0` build for linux/amd64, linux/arm64, darwin and linux/386 |

## When the port is finished

1. Move the durable parts of each `.claude/port/P<n>-reading-guide.md` into the package READMEs or CLAUDE.md:
   - the hot/cold maps;
   - why each optimisation exists;
   - the edge cases.
2. Check that every open item in PORT.md is either done or has been moved here, and that the "Convention sweep" table is empty.
2b. Run a final convention sweep: `go-reviewer` checks every package against the current CLAUDE.md, and `PORTED_DIRS` in the Makefile becomes the whole module (`./...`).
3. Remove the master-comparison test scaffolding, which only makes sense while master is the reference.
   - In `internal/ds/trie_test.go`, remove `findMaster`, `TestTrieFind_MatchesMasterWhenMasterFindsAnything` and `TestTrieFind_BacktrackKeepsParamsWithTheirRoute`'s master comparison. Rewrite that last test with fixed expected values.
   - `TestTrieFind_BacktrackingOnlyTurnsMasterMissesIntoMatches` also calls `findMaster`. Rewrite it so that whenever `Find` returns no wildcard, its result must equal `bruteForceMatch`.
   - Keep `bruteForceMatch`. It doesn't depend on master and still checks `Find`.
   - Look for other `*Master` reference copies added by later port steps and treat them the same way.
4. Delete `.claude/port/`, and remove `/port` from `.claude/commands/` and CLAUDE.md.
