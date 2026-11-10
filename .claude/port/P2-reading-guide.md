# P2 reading guide: `trace`, `event`, `pattern`, `log`

P2 ports four leaf packages from master (`13242d4`). `exception` and `aggregation` moved to P3 (D-022), because their main callers sit in the red zone.

| Package | On main before P2 | Callers on main |
|---|---|---|
| `trace` | new | none until P3 (`core`, `accesslog`) |
| `internal/event` | new (master's `event/`, moved by D-027) | none until P3 (`core`, `accesslog`) |
| `pattern` | new | `log` (masking); `memorybroker` and `wsevent` at R-2 |
| `log` | existed (org version) | `core/app.go:128`, `sample` |

## Decisions applied

| ID | Decision |
|---|---|
| D-022 | `exception` and `aggregation` move to P3 |
| D-023 | `log.NewLog` returns `common.Logger` |
| D-024 | Pretty log format follows master: tree, aligned, unquoted strings. master's empty-`error` bug is fixed in `/pair-review log` |
| D-025 | Log tags (`log:"-"`, `log:"name"`) and `MaskFields` are ported |
| D-026 | `trace`: `TransportGQL` and `TransportRPC` removed (unused) |
| D-027 | `event` → `internal/event`: framework-internal (per-layer timing for access log and devtool), 4 methods, copy-on-write reads (lock-free) |
| D-029 | `pattern`: in-place matching (0 allocs), `IsGlobal` removed |
| D-030 | Glob-style syntax: `*` is exactly one segment, `**` is one or more at the end; adapters translate and filter locally |

## Reading order

`trace` → `internal/event` → `pattern` → `log`. `log` depends on `pattern` and `internal/color`.

### 1. `trace/trace.go` (~30 lines): the event data model

**Hot/cold map:**

| Item | Class | Used by (master) |
|---|---|---|
| `Event` (112 bytes) | hot when tracing is on: one per pipeline stage per request | `core/app.go`, `core/fn.go` (21 sites), `accesslog`. Every site is guarded by `HasListeners` |
| `EventName`, `Stage*`, `TransportHTTP`/`WS` | used with `Event` | 29 sites |

**Lenses:**
- **Performance:** nothing in the package costs anything. At its call sites (P3), boxing `Event` into `...any` costs 1 alloc per stage when tracing is on.
- **Concurrency, security, resources:** ✅ no state, no input, no functions.
- **Readability:** ✅
- **Devtool readiness:** `Event` is devtool's data model. It lacks a start time, the route pattern, stage order and parent, the error, and the version, and it has no `routing`/`response` stages. These are listed in PORT.md (P3) and in R-1.

### 2. `internal/event/event.go` (~70 lines): the trace bus

**Hot/cold map:**

| Function | Class | Calls per request (master) |
|---|---|---|
| `HasListeners` | **hot even with tracing off** (15 sites in `core`) | about 10–15 |
| `Emit` + `callEventListenerSafe` | hot when access log or devtool is on (13 sites) | about 10–15 |
| `NewEvent` | cold (`core/app.go:132`) | — |
| `On` | cold (`accesslog` init) | — |
| 7 other methods | unused, removed (D-027) | — |

**Lenses:**
- **Performance (D-027):** listeners sit in `atomic.Pointer[listenerMap]`, so reads take no lock and make no copy. `On` copies the map under `writeMu`, and it only runs at startup.
- **Concurrency:** writers serialise on `writeMu`, while readers see an immutable snapshot.
  - `TestEvent_ConcurrentOnEmitHasListeners` runs 4 writers and 8 readers under `-race` and checks that no `On` is lost.
  - `TestEvent_OnDuringEmitDoesNotAffectThatEmit` pins the snapshot semantics.
- **Security:** ✅ no client input. Event names are framework constants.
- **Correctness and resources:**
  - a panicking listener is recovered, reported on stderr, and the next listener still runs (tested);
  - listeners only grow, but registration happens at startup only, and there's a warning above 10 per event;
  - devtool (R-1) will need unsubscribe.
- **API:** 4 methods, and the zero value `var e Event` is usable (tested). master's zero value panicked in `On`.
- **Readability:** ✅ `snapshot` sits after its last caller, and `callEventListenerSafe` after `Emit`.

**Benchmarks** (Go 1.27.3, Intel i7-9750H, mean of 4 runs; before = master's `event.go`):

| Benchmark | Before, 12 CPUs | After, 12 CPUs |
|---|---|---|
| `HasListeners`, tracing off, parallel | 66.6 ns | 0.9 ns (3.6 ns when another event has a listener) |
| `HasListeners`, tracing off, single | 26.4 ns | 3.6 ns |
| `Emit`, 1 listener, parallel | 159.4 ns, 1 alloc | 11.5 ns, 1 alloc |
| `Emit`, 1 listener, single | 64.7 ns, 1 alloc | 35.8 ns, 1 alloc |
| `On` (cold) | 444 ns, 5 allocs | 226 ns, 5 allocs |

With tracing off and about 10–15 stages per request, the check cost about 0.5–1 µs per request under parallel load. It now costs about 10–50 ns. The 1 alloc in `Emit` is the benchmark boxing its argument; the real cost is the caller's (P3).

### 3. `pattern/pattern.go` (~110 lines): topic and key patterns

**Hot/cold map:**

| Function | Class | Callers (master) |
|---|---|---|
| `Match` (+ `matchSegments`) | hot with `MaskFields` (per log attr, `log_mask.go:140`); at R-2 per publish (`memory_broker.go:170`) and per WS message (`ws_event.go:98`) | |
| `Raw`, `IsExact` | hot with masking (`log_mask.go:125, 134`) | |
| `NewPattern` | cold (`WrapLogger`, `wsevent` register); warm at R-2 (broker subscribe) | |
| `Kind`, `SimplePrefix` | warm/cold at R-2 (broker subscribe/unsubscribe, `wsevent` register) | |
| `IsGlobal` | unused, removed | — |

**Semantics (D-030, glob-style)**, pinned by `TestMatch_SemanticsTable`:
- `*` matches **exactly one** segment, in any position: `chat.*` matches `chat.room1` but not `chat.room1.typing`.
- `**` matches **one or more** segments, and only as the last segment: `chat.**` matches both, but not `chat`.
- `**` alone matches everything.
- `**` anywhere but last, and `*` inside a segment (`a*b`), are literals.
- This replaces master's single `*`, whose meaning depended on its position. `MaskFields` uses the same syntax.

**Lenses:**
- **Performance (D-029):** in-place walk with `strings.Cut`, 0 allocs. It stops at the first mismatch, so a 10 KB hostile topic costs 20 ns.
- **Concurrency:** ✅ `Pattern` is an immutable value.
- **Security:** topics come from WS clients at R-2. The DoS from long topics is fixed here, and R-2 should still bound topic length and reject empty segments.
- **Correctness:** with an empty trailing segment, `chat.*.**` matches `chat.room1.` while `chat.**` doesn't match `chat.`. It is pinned in tests, and R-2 rejects empty segments anyway.
- **Readability:** ✅ test helpers sit after their callers.
- **`pattern/MATCHING.md`** is a committed table of real `Match` results: 10 scenarios, 29 topics, one per syntax rule, including the `MaskFields` usage. It is kept in sync by `TestMatchingTableIsUpToDate`, which you regenerate with `-update-matching`, the same way as `internal/ds/MATCHING.md`.

| Benchmark | Before (master) | After |
|---|---|---|
| complex, 3 segments | 101.1 ns, 1 alloc | 53.3 ns, 0 allocs |
| complex, middle + trailing | 138.4 ns, 1 alloc | 45.8 ns, 0 allocs |
| complex, 10 KB hostile topic | 41,116 ns, 40 KB | 19.8 ns, 0 allocs |
| suffix wildcard, one segment | 35.7 ns | 6.2 ns |
| suffix wildcard, miss | 30.3 ns | 4.4 ns |
| exact | 5.2 ns | 4.1 ns |

### 4. `log/` (~560 lines): the logger, the pretty handler and masking

Read `log.go` → `log_handler.go` → `log_mask.go`, with `fn.go` and `log_color.go` beside them. Reviewed in `/pair-review log` (D-031).

**Hot/cold map:**

| Function | Class | Callers (master) |
|---|---|---|
| `prettyHandler.Handle` + `format`, `buildNode`, `collectChildren`, `textOf`, `write*`, `writeEscaped` | **hot**: one record per log call; Pretty is the default format | user handlers through `app.Logger`; `accesslog` worker (1 per request, on its own goroutine) |
| `maskingLogger.*` + `transform`, `maskAny`, `maskValue`, `matches`, `matchesSuffix` | **hot**: it wraps every logger | `NewLog` → `WrapLogger` |
| `logInstance.*`, `enabled` | hot (user) | as above |
| WS `Error` calls | warm (connection failures) | `core/read_loop.go:42`, `write_loop.go:13`, `ws.go:124–170` |
| `NewLog`, `loadLogOptions`, `newPrettyHandler`, `WrapLogger` | cold | `core/app.go:128` → `Create()` |

**Lenses:**
- **Performance:**
  - `Handle` formats into a pooled buffer outside the lock and locks only around `Write` (D-031 C1, approved `sync.Pool`). Buffers above 64 KB are dropped instead of pooled.
  - Masking runs only when the level is enabled (`levelEnabler`), copies args only when a value changes, and builds no paths when there are no rules.
  - `format` collects up to 8 attributes in a stack array (approved), so a typical record allocates nothing: 1 → 0 allocs, ~720 → ~650 ns in an A/B run.
  - Pending type B: path building in masking, and `fmt` for non-string map keys (ROADMAP backlog).
- **Concurrency:**
  - Shared state: `lockedWriter` (one mutex per writer, shared by `With` clones), `bufferPool`, and the `sync.Once` singleton in `NewLog`. Rules and level colours are read-only after construction.
  - `TestPrettyHandler_Handle_ConcurrentRecordsStayWhole`: 16 goroutines × 200 records under `-race`, none lost or interleaved.
  - Writes are synchronous like `slog`/`zap`/`zerolog`, so a slow writer blocks every logging goroutine (C2). An opt-in async writer is R-4.
  - A second `NewLog` call returns the first logger and ignores its options (C3, kept from master).
- **Security:**
  - Control bytes in the message, keys and strings are escaped (S1), so a client-controlled URL in the access log can't forge lines or send terminal escapes.
  - Masking holes closed: a branch that matches is masked whole (B2), slices are traversed, rules are case-insensitive, `slog.Attr` or non-string keys no longer shift the key/value pairs (B3), and `slog.LogValuer` and `slog.Value` values are resolved before masking, at any depth. Nested ones become plain Go values, so JSON prints them and Pretty never expands a type that hides itself.
  - Fuzz targets: `FuzzWrapLogger_MaskedSecretNeverReachesOutput` (map, slice, LogValuer, group value, nesting, upper-case keys, a leading non-string key and `slog.Attr`, checked in Text, JSON and Pretty), `FuzzPrettyHandler_ClientTextCannotBreakTheLayout` and `FuzzMatchesSuffix_FastPathEqualsSlowPath`. Each ran 30–40 s locally with no failure (0.7–2.4 million inputs); `make check` replays the seeds.
  - Cyclic or very wide values stop at depth 16 and 1000 nodes per record, in both the handler and masking (B4). Before, a self-referencing map hung the logger while it held the global lock.
- **Correctness:** errors print `Error()` and Stringers `String()` (B1). The handler implements `Enabled`, `WithAttrs`, `WithGroup` and `AddSource`, and resolves `slog.LogValuer` (B5). `slog.Group` attributes are flattened with a key prefix. A user attribute named `level` no longer panics the JSON and Text formats (`replaceLevelLabel`).
- **Resources:** ✅ the pool drops oversized buffers (`TestPrettyHandler_Handle_OversizedRecordThenSmallRecord`). No goroutines.
- **API:** `PrettyHandler`, `NewPrettyHandler` and `PrettyHandlerOptions` are unexported. `LogOptions.Transports` and `LogTransport` are removed (unused). `Color`, `Colored` and `WrapLogger` stay exported for `accesslog` and custom loggers.
- **Readability:** `unsafe` is removed, so tags on unexported fields are ignored. ANSI codes come from `internal/color`, and shared helpers sit in `fn.go`.

**Masking rule (D-031):** every rule is matched with `pattern.Match` against every suffix of the attribute path, case-insensitively, at every node. A matching node is replaced whole by `[REDACTED]`.

| Rule | Logged | Result |
|---|---|---|
| `password` | `user=struct{Name, Password}` | `Password: [REDACTED]` |
| `password` | `users=[]struct{…Password}` | each `Password: [REDACTED]` |
| `user.*` | `user={card:{cvv}, name}` | `card` and `name` masked whole |
| `card.*` | `user={card:{cvv}}` | `user.card.cvv` masked (suffix `card.cvv`) |
| `password` | `slog.String("password", …)` | masked |

**Benchmarks** (Go 1.27.3, Intel i7-9750H, `-count 5`; before = master's code at the start of the review):

| Benchmark | Before | After |
|---|---|---|
| `Handle`, 3 attrs + map, 12 CPUs parallel | 1925 ns, 12 allocs | 700–1000 ns, 10 allocs |
| `Handle`, access-log record with nested stages, single | 6762 ns, 49 allocs | ~7 µs, 35 allocs |
| `Handle`, 3 scalar attrs, single | 524–689 ns, 1 alloc | 677–847 ns, 0 allocs |
| `Handle`, no attrs, single | 216–302 ns | 248–316 ns |
| `Debug` filtered by level, struct arg | 2454 ns, 11 allocs | 57 ns, 2 allocs |
| `Info`, scalar args, no rules | 65 ns, 2 allocs | 51 ns, 1 alloc |
| `Info`, tagged struct, no rules | 1073 ns, 14 allocs | ~900 ns, 8 allocs |
| `Info`, slice of 10 structs, 1 rule | not traversed (leaked) | 11.1 µs, 145 allocs |

Single-goroutine `Handle` is about 25% slower than master (+~150 ns with 3 attrs, +~25 ns with none), measured by alternating master and main 3 times on the same machine. `pprof` splits it between group flattening and `LogValuer` resolution (~6%), the escape scans (~5%) and escaped key widths (~4.5%). Smaller tweaks were within the machine's noise, so none were applied. The remaining allocs in the scalar and filtered rows are the caller's variadic slice and boxing.
