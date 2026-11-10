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

## D-002 Fate of the exported `utils` package

- **Status:** decided 2026-11-10 (P1).
- **Context:**
  - main has an exported `utils` package, used by 28 files.
  - master replaced it with `internal/{color,crypto,num,slice,str,test}` and renamed the helpers: `utils.ArrFind` → `slice.Find`, `utils.StrUUID` → `crypto.UUID`, `utils.ErrorMessage` → `test.DiffMessage`.
- **Decision:** follow master and keep the helpers internal only.
  - P1 adds `internal/`.
  - Callers move to `internal/` as their packages are ported (P2/P3).
  - `utils` is deleted once its last caller has moved, at the end of P3 at the latest.
  - Until then, tests in packages not yet ported keep `utils.ErrorMessage`. Ported packages use `internal/test.DiffMessage`.

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

## D-008 Unused code in master's `internal/`

- **Status:** decided 2026-11-10 (P1).
- **Context:** `git grep` over master's non-test code finds no caller for:
  - `crypto.Random` and its helper `withCharset`;
  - `runtime.NodeID` (the whole `internal/runtime` package);
  - `slice.ToUnique`;
  - `ds.Trie.Remove` and its helper `removeGlobChild`.
- **Decision:** leave them out. Port only code that master calls, together with the tests for that code. Add a function back in the step that first needs it.
- **Extended 2026-11-10:** the per-package review found 4 more unused functions, `str.AddBegin`, `str.AddEnd`, `str.IsLower` and `str.IsUpper`, and removed them. From now on, the review checks every function of a package for callers.

## D-009 Where the route trie (`internal/ds`) is ported

- **Status:** decided 2026-11-10 (P1).
- **Decision:** port it in P1, laid out as master has it.
  - It has no caller until P3.
  - It is read again in P3 next to `routing`, with request-path benchmarks.

## D-010 Behaviour of the helpers

- **Status:** decided 2026-11-10 (P1).
- **Context:** a probe on main's `utils` showed three bugs, which master's versions fix:
  - `StrAddBegin("a", "/x")` panics with slice out of range;
  - `StrAddEnd("ab", "abc")` returns `"ab"`;
  - `NumF64ToAnyNum(-1, reflect.Uint8)` returns an `int`.
  
  main's `StrRandom` also uses `math/rand`, and `StrRemoveSpace` compiles a regexp on every call.
- **Decision:** take master's behaviour. The fixes take effect only when a caller switches from `utils` to `internal/` (P2/P3). Each switch is called out in that step's reading guide.

## D-011 `color`: reset per style instead of a full reset

- **Status:** decided 2026-11-10 (P1).
- **Context:** every `Fmt*` closed with `\x1b[0m`, which resets everything. A probe showed the problem: `FmtBold("%s", "a "+FmtRed("b")+" c")` loses the bold on `" c"`.
- **Decision:** each style closes only its own attribute:
  - foreground colours close with `39`;
  - backgrounds close with `49`;
  - bold and dim close with `22`;
  - italic closes with `23`.
  
  All escapes are written as `\x1b`. Output that isn't nested looks the same on a terminal.

## D-012 `color`: `%` behaves like printf

- **Status:** decided 2026-11-10 (P1).
- **Context:** with no args, the format string was printed verbatim, so `FmtRed("100%%")` gave `100%%`, while the same format with args gave `100%`. `go vet` treats these functions as printf wrappers, so it assumes printf semantics.
- **Decision:** take the fast path (plain concatenation) only when there are no args and the format string contains no `%`. Everything else goes through `fmt.Sprintf`.

## D-013 `color`: one internal helper

- **Status:** decided 2026-11-10 (P1).
- **Decision:** the 18 `Fmt*` functions keep their names and signatures. Each one becomes a single call to `wrap(openCode, resetCode, format, a...)`. `wrap` forwards `a...` variadically, so `go vet` still checks the format strings (verified with a probe).
- **Kept simple (2026-11-10).** `color` is cold path: startup and error output. An optimised version with `fmt.Appendf` and `unsafe.String` (1 alloc instead of 2, about 15% faster with args) was tried and reverted. The owner chose readability on the cold path (see CLAUDE.md, "Performance depends on the path").

## D-014 `crypto.UUID` returns only a string

- **Status:** decided 2026-11-10 (P1).
- **Context:** since Go 1.24, `crypto/rand.Read` never returns an error; it crashes the program instead. The repo uses Go 1.27.3 (1.25.5 when this was decided). So `UUID() (string, error)` always returned a nil error, and every caller handled an error that can't happen.
- **Decision:**
  - The signature becomes `UUID() string`.
  - The call is `_, _ = cryptoRand.Read(...)`, which satisfies `errcheck`.
  - Callers ported later (ctx, ws, memorybroker) drop their dead error handling.

## D-015 `test.DiffMessage` prints values with `%#v`

- **Status:** decided 2026-11-10 (P1).
- **Context:** with `%v`, a failure message can make different values look equal. A probe showed:
  - `1` and `"1"` both print `1`;
  - `""` and `" "` look the same;
  - `[]string{}` and `[]string(nil)` both print `[]`, although JSON encodes them as `[]` and `null`.
- **Decision:** print `expected` and `actual` with `%#v` (Go syntax). Large structs print longer, with field names. Argument order and layout stay the same: `DiffMessage(actual, expected, desc)` prints Expected first. This is cold path, since it only runs when a test fails.

## D-016 `test.DiffMessage` layout and colours

- **Status:** decided 2026-11-10 (P1).
- **Context:** the owner found the failure logs ugly:
  - the first line was empty, so `desc` sat apart from `file:line`;
  - the `Expected:` and `Actual:` labels didn't line up.
  
  The owner wants to keep colours.
- **Decision:**
  - The layout is `desc` on the `file:line` line, then `expected:` and `actual:` with their values in aligned columns.
  - The colours are: desc bold, labels dim, expected green, actual red. ANSI codes take no width, so the columns stay aligned.
  - Colours are always on. The owner chose this on 2026-11-10: no environment switch, so there's no `NO_COLOR`/`TERM` check.
  - `test` defines its own 6 escape constants instead of importing `color`, so there is no import cycle. `color`'s own tests use `DiffMessage` too.
  - Tests check the layout after stripping escape codes, and the exact coloured output.
  - D-015 (`%#v`) still applies. In `color`'s tests the values are themselves coloured strings, and `%#v` shows their codes as text. That is intended, because the codes are what's under test.

## D-017 `str.Enclose` gets a fast path

- **Status:** decided 2026-11-10 (P1, first `/pair-review`).
- **Context:**
  - `Enclose` is hot: master's `Router.Match` calls it on every request, and so does `ws.isWSPath` when WS is enabled.
  - Most request paths are already clean, yet the function copied every byte through a `strings.Builder`.
  - The owner argued that skipping small gains one function at a time loses their sum (10 × 14 ns = 140 ns). The per-function threshold was dropped (CLAUDE.md).
- **Decision:**
  - When the input starts with `sep`, has no ASCII space and no doubled `sep` or `**`, return it as is if it ends with `sep`, otherwise return `str + sep`. Everything else goes to `encloseSlow`, the original algorithm.
  - A test checks that `Enclose` equals `encloseSlow` over 400k random inputs.
  - Results (6-run mean, `encloseSlow` being master's exact `Enclose`):
    - `/api/v1/users/123/orders`: 83.1 → 59.2 ns (1 alloc);
    - `.../orders/`: 81.0 → 27.8 ns, 0 allocs;
    - `/users`: 41.5 → 34.9 ns;
    - `/`: 20.6 → 3.0 ns, 0 allocs;
    - dirty `//users//...`: 93.9 → 110.0 ns (+17%, extra scan);
    - route pattern without a leading `/`: unchanged.

## D-018 `num.F64ToAnyNum` reports values it can't represent

- **Status:** decided 2026-11-10 (P1, `/pair-review internal/num`).
- **Context:** the function is hot. master's `ctx.Body.Bind` → `BindStruct` calls it for every numeric JSON field, and the client controls the value. A probe proved silent corruption:
  - `1e20` → `int64` gave `-9223372036854775808`;
  - `300` → `int8` and `uint8` gave `44`;
  - `1.9` → `int` gave `1`;
  - `-1` → `uint8` was clamped to `0`;
  - `3.4e39` → `float32` gave `+Inf`.
- **Decision:**
  - The signature is now `F64ToAnyNum(f64, kind) (any, bool)`.
  - `ok` is false for NaN/Inf, for out-of-range values, for a fraction when the target is an integer type, for a negative value when the target is unsigned, for magnitudes above `MaxFloat32` when the target is `float32`/`complex64`, and for unsupported kinds.
  - Integer kinds go through the generic `toInteger[T](f64, low, high)`, which is inlinable. It uses exact power-of-two bounds and checks wholeness with `float64(T(f)) == f`, without `math.Trunc`.
  - Cost against master (6-run mean): about +1–5 ns on valid input, and an overflow is now rejected in 3.7 ns with 0 allocs. A first version with `math.Ldexp`/`math.Trunc` cost +7–9 ns and was replaced.
- **For P3:** binding must turn `ok == false` into a 400 response.

## D-019 `slice.StrParse*` report parse failures; `Find`/`FindIndex` removed

- **Status:** decided 2026-11-10 (P1, `/pair-review internal/slice`).
- **Context:** `StrParse*` is hot. master's `ctx.Query/Form/Header/Param.Bind` → `BindStrArr` calls it for slice fields, and the client controls the input. A probe showed bad input silently turning into zero values:
  - `?ids=1,abc,3` → `[1 0 3]`;
  - `300` → `int8` gave `0`;
  - `-1` → `uint` gave `0`;
  - `yes` → `bool` gave `false`;
  - `1e400` → `float64` gave `0`;
  - `" 7"` → `int` gave `0`.
- **Decision:**
  - The 15 typed `StrParse*` functions return `([]T, bool)`. `ok` is false if any element fails to parse, whether from bad syntax or overflow. They share the helper `parseAll[T]`.
  - `StrParseAny` can't fail, so it keeps returning `[]any` (as in D-014: no impossible results).
  - `Find` and `FindIndex` had no caller in master and were removed (D-008).
  - Cost, measured with the result escaping as in `BindStrArr`: equal or faster than master with the same allocs (Int×5 85.9 → 70.5 ns; Int64×5 125.0 → 104.0 ns).
- **For P3:** binding turns `ok == false` into a 400 response.

## D-020 Route trie: cached `$`/`*` children, backtracking, unused code removed

- **Status:** decided 2026-11-10 (P1, `/pair-review internal/ds`).
- **Context:**
  - `Trie.Find` is hot: master's `Router.Match` calls it 1–2 times per request.
  - Per segment it did 2–3 map lookups: `Children[seg]`, `Children["*"]` and `Children["$"]`.
  - It walked greedily without backtracking. With `/users/me/` and `/users/$/profile/` both registered, `/users/me/profile/` returned 404.
  - `Len`, `ToJSON` and `genTrieMap` had no caller in master. (An earlier guide claimed devtool used `ToJSON`. That was wrong.)
- **Decision:**
  - **Cached children.** `Insert`, which is cold, sets `node.param` and `node.wildcard`. `Find` reads those fields instead of looking up `"$"` and `"*"` in the map. A test reproduces master's exact `Find` as `findMaster` and checks that it gives identical results, over 500k random lookups.
  - **Backtracking.** `Find` runs the greedy walk (`findGreedy`) first. It runs `search`, a depth-first walk in priority order (exact, then param, then `*`, then glob), only when the greedy walk found **neither a route nor a wildcard fallback**. When `search` wins, it returns `wildcardRaw == ""` and its own params, or nil when there are none.
    - Guarantee: for request segments that contain no `$` or `*`, whenever master returns anything (a route or a wildcard), the result is identical, params included. Backtracking only turns a master 404 into a match.
    - **Limit 1:** segments containing `$` or `*` may pick a different route or wildcard than master, because `exactChild` no longer treats them as literals. They may also turn a master miss into a wildcard match. go-reviewer found 6,465 differences in 100k hostile-segment lookups, mostly the intended capture of `$`.
    - **Limit 2:** backtracking doesn't run when a terminal `*` lies above the dead end. With `/a/*/`, `/a/b/x/` and `/a/$/c/`, the path `/a/b/c/` still returns the wildcard `/a/*/` (master's result), not `/a/$/c/`.
    - Tests: equality with `findMaster` over 500k random lookups; brute-force agreement on master misses, which recovers 23,344 of the 500k lookups.
  - **Placeholder keys are not literals.** The exact lookup (`exactChild`) ignores the keys `$` and `*`, and any key containing `*`. So a request segment that is literally `$` or `*` takes the param or wildcard branch.
    - Each child is then reachable through exactly one branch, so `search` visits each node at most once and its cost is linear.
    - This fixes two problems that `go-reviewer` found:
      - an exponential blow-up on hostile paths: depth 20 took 64 ms before, and depth 64 now takes 19 µs;
      - a bug inherited from master: `/users/$/` matched with no params, which would have caused an index panic at P3. It now captures `$` as the param.
  - **First version, replaced:** it ran `search` whenever no route matched and returned the greedy `wildcardRaw` alongside `search`'s params. go-reviewer showed that a wildcard fallback could then get params belonging to another route (`/a/b/$/*/` with `/a/$/c/d/e/`), and that `search` was exponential. The guarantee text in that version ("can't blow up") was wrong.
  - **Removed:** `Len`, `ToJSON` and `genTrieMap` (D-008).
  - **Revised 2026-11-10:**
    - `supportParams` was removed. In master it is read inside `Find`, but its only caller, `routing/router.go:179`, always passes `true`, and main's own trie has no such mode. `Find(path, sep)` drops that parameter.
    - `matchWildcard` moved from `fn.go` into `trie.go` (function-placement rule), and its parameter `str` became `segment` so it doesn't shadow the imported `str` package. `fn.go`, `fn_test.go` and `fn_bench_test.go` are gone.
- **Cost** (6-run mean; the machine was noisy):
  - `/health/` 57.6 → 36.0 ns;
  - `/api/v1/users/123/orders/` 341 → 159 ns;
  - `/api/v1/products/9/reviews/3/` 446 → 168 ns;
  - `/api/v1/users/me/orders/` was a wrong 404 in 202 ns, and is now a correct match in 271 ns;
  - a real 404 (`/api/v2/nothing/`) went from 110 to 143 ns because of the backtracking walk;
  - after the placeholder check, valid routes cost a few ns more: `/health/` about 30 ns, `/api/v1/users/123/orders/` about 158 ns.

## D-021 Naming in `internal/`

- **Status:** decided 2026-11-10 (P1, readability lens).
- **Applied:**
  - the `slice.Iter` parameter `dimmensions` became `dimensions` (typo);
  - `color.wrap(open, close, …)` became `(openCode, resetCode, …)`, because `close` shadowed the builtin;
  - `num.F64ToAnyNum(f64, t)` became `(f64, kind)`;
  - `color.wrap` moved after its callers (the function-placement rule);
  - in package `str`, string parameters named `str` became `s`, so they don't shadow the package name (standard-library style).
- **Kept, with the owner's reasons:**
  - `internal/crypto` stays, because hashing and similar helpers will join `UUID`;
  - `internal/ds` stays, because more data structures will come;
  - `Trie.Raw` stays, because it holds the original route string.
  
  Don't re-propose these.
- `misspell` is enabled in `.golangci.yml`.

## D-022 `exception` and `aggregation` move from P2 to P3

- **Status:** decided 2026-11-10 (P2 survey).
- **Context:** both packages already exist on main with an API that differs from master's and is tied to `ctx`:
  - `aggregation`: `AggregationOperator = func(*ctx.Context, any) any` with `Map/Consume/Error/First/Of`, against master's `func(any) any` with `Transform/Tap/Filter`;
  - `exception`: `NewException(response any, code string)` with `GetHTTPStatus/GetResponse`, against master's `NewException(message string, code int)` with `GetCode/GetMessage/GetStatusText`, plus WS exceptions.
  
  Their callers on main are in `common`, `core` and `ctx` (the red zone), so porting them alone would break the build.
- **Decision:** P2 ports only `pattern`, `trace`, `event` and `log`. `exception` and `aggregation` are ported in P3 together with their callers, and their API divergences are asked then.

## D-023 `log.NewLog` returns `common.Logger`

- **Status:** decided 2026-11-10 (P2).
- **Context:** main's `NewLog` returns `*logInstance`, an unexported type. master's returns the `common.Logger` interface, which `WrapLogger` (masking) needs. main's callers (`core/app.go:128`, `sample`) still compile with either.
- **Decision:** take master's: `func NewLog(opts *LogOptions) common.Logger`. `common.Logger` is identical on main and master, so `log` can import main's `common` without touching the red zone.

## D-024 Pretty log format follows master

- **Status:** decided 2026-11-10 (P2).
- **Context:** the actual outputs of main's and master's `PrettyHandler` were compared on the same records.
  - **master:** structs and maps are rendered as a tree (maps sorted by key), values are column-aligned, and strings are unquoted.
  - **main:** one-line `%v`, quoted strings, no alignment.
- **Decision:** take master's format. master's handler shows an empty value for `error` attrs (`errors.New("boom")` prints nothing), and that bug is fixed separately during `/pair-review log`.

## D-025 Log tags and masking are ported

- **Status:** decided 2026-11-10 (P2).
- **Context:** master adds two features that main lacks, which is why main prints passwords:
  - the struct tag `log:"-"` hides a field; `log:"name"` renames it;
  - `LogOptions.MaskFields` together with `WrapLogger` replaces matching keys or paths with `[REDACTED]`, using `pattern` syntax such as `password` or `user.*`.
- **Decision:** port both. `log_tag.go` uses `unsafe` to read unexported tagged fields. That is a heavy technique, and it needs the owner's explicit approval during `/pair-review log`.

## D-026 `trace`: unused transports removed

- **Status:** decided 2026-11-10 (P2, `/pair-review trace`).
- **Decision:** remove `TransportGQL` and `TransportRPC`, which have no user in master (D-008). Add them back with a GraphQL or RPC transport.

## D-027 `event`: framework-internal, lock-free reads

- **Status:** decided 2026-11-10 (P2, `/pair-review event`).
- **Context:**
  - master's `core` calls `HasListeners` on every pipeline stage of every request, even with tracing off. That is about 10–15 calls per request, and `Emit` runs on every stage when access log or devtool is on.
  - Both took `RWMutex` locks, and `Emit` took the write lock. At 12 CPUs in parallel, `HasListeners` cost 66.6 ns and `Emit` 159.4 ns.
  - 7 of the 11 methods had no caller.
- **Owner's intent (2026-11-10):** `event` exists mainly to measure the duration of each pipeline layer (middleware, guard, interceptor, pipe, handler) and to notify subscribers such as access log and devtool when a request finishes. The user-facing pub/sub is the message broker (R-2), which is exported. `event` stays simple and internal, so the two never overlap.
- **Decision:**
  - **Location:** moved to `internal/event`, so code outside the module can't import it. It had no caller on main yet. At P3, decide how master's exported `accesslog.AccessLogConfig.Event *event.Event` field is handled.
  - **Role:** framework-internal infrastructure. Only `NewEvent`, `On`, `Emit` and `HasListeners` remain. `Reset`, `SetMaxListeners`, `Once`, `Off`, `RemoveAllListeners`, `ListenerCount` and `EventNames` were removed (D-008). Unsubscribe comes back with devtool (R-1), which needs it for SSE clients that come and go.
  - **Copy-on-write:** listeners live in `atomic.Pointer[listenerMap]`. `HasListeners` and `Emit` read without locking and without copying. `On` (cold) copies the map under `writeMu`.
    - Emit iterates the snapshot taken when it started, and an `On` during an `Emit` is not seen by that `Emit` (tested).
    - The leak warning is fixed at 10 listeners per event (`maxListenersPerEvent`).
  - **Zero value usable:** `var e event.Event` works, because a nil snapshot means no listeners. master's zero value panicked in `On`.
- **Result, 12 CPUs:**
  - `HasListeners` parallel: 66.6 → 0.9 ns, or 3.6 ns when another event has a listener (re-measured in P2 with a sink, so the call isn't eliminated). It is inlinable now.
  - `Emit` with 1 listener, parallel: 159.4 → 11.5 ns.
  - `On`: 444 → 317 ns.
  - The remaining 16 B alloc per `Emit` comes from the caller boxing its argument into `...any` (P3).

## D-028 Message and error conventions

- **Status:** decided 2026-11-10 (before continuing P2).
- **Context:** master mixes at least four styles:
  - two prefixes in one package (`memorybroker:` and `broker:`);
  - prefixes that don't match the package (`EventEmitter:`), or no prefix at all (`%v is not defined`);
  - bootstrap panics that are sometimes coloured and sometimes not;
  - 4 direct `os.Stderr` writes that bypass the logger, so JSON format and masking are lost;
  - inconsistent client messages (`"Access denied"`, `"Too Many Requests"`);
  - internal error text sent to clients (`BadRequestException(err.Error())`).
- **Decision:** four kinds of message, each with its own rule. They are written in CLAUDE.md, under "Messages and errors".
  1. **Bootstrap panics** (dev terminal, at `Create()`): `"<category>: <detail>"`, red via `color.FmtRed`. The category comes from the fixed list and applies to every package.
  2. **Returned errors** (user code):
     - lowercase, no trailing punctuation, no colour;
     - the prefix is exactly the package name;
     - wrap with `%w`;
     - export a sentinel `ErrXxx` for each condition callers check.
  3. **Runtime diagnostics** (operators): always through `common.Logger`, never `fmt` to stderr. `msg` is a fixed PascalCase event name (`EventListenerPanic`), and details go in key-value attrs.
  4. **Client-facing messages** (exceptions, HTTP): sentence case with no trailing period, and never internal error text. A 4xx states what the client got wrong, in our own words. A 5xx says "Internal server error" and the details go to the log, keyed by request ID. The JSON shape is decided at P3 with `exception`.
- **Applied now:** `internal/event` uses `event:` prefixes without colour: `event: possible listener leak: …` and `event: listener panic recovered: …`. It still writes to stderr until `core` gives it a logger at P3. Other packages are fixed when they are ported.

## D-029 `pattern`: in-place matching, no allocs

- **Status:** decided 2026-11-10 (P2, `/pair-review pattern`).
- **Context:**
  - `Match` is hot when `MaskFields` is set (once per log attr), and at R-2 (once per publish and per WS message).
  - Complex patterns ran `strings.Split(topic, ".")` on every call, costing 1 alloc. A 10 KB client topic cost 41 µs and 40 KB.
  - The suffix-wildcard branch built `simplePrefix+"."` on every call.
- **Decision:**
  - `matchSegments` walks the topic with `strings.Cut` and stops at the first mismatch. A test checks it against a split-based reference over 300k random complex patterns.
  - The suffix branch checks the `.` byte and the prefix directly, without concatenating.
  - `KindComplex` is the `default` case, so the unreachable `return false` is gone.
  - `IsGlobal` was removed (unused, D-008). The package stays public; the owner chose not to move it to `internal/`.
- **Result:**
  - complex 3-segment match: 101 → 53 ns, 1 → 0 allocs;
  - hostile 10 KB topic: 41,117 → 20 ns, 40 KB → 0;
  - suffix wildcard: 25–36 → 4–9 ns.
- **Unchanged on purpose (inherited from master):**
  - an empty trailing segment counts for a complex pattern but not for a suffix wildcard: `*.*` matches `b.`, while `b.*` doesn't.
  - Recorded for R-2: the broker should reject topics with empty segments.

## D-030 Pattern syntax is glob-style: `*` is one segment, `**` is one or more

- **Status:** decided 2026-11-10 (P2). This replaces master's single `*`.
- **Context:**
  - master's trailing `*` matched one or more segments while a middle `*` matched exactly one, so "exactly one trailing segment" couldn't be written: `chat.*` also matched `chat.room1.typing`.
  - Every major broker uses two symbols: NATS `*`/`>`, MQTT `+`/`#`, RabbitMQ `*`/`#`. Glob and `.gitignore` use `*`/`**`, which web developers already know.
- **Decision:**
  - `*` matches **exactly one** segment, in any position.
  - `**` matches **one or more** segments, and only as the last segment. `chat.**` does not match `chat` (like `.gitignore` `foo/**`, and NATS `>`).
  - `**` alone matches everything. `*` alone matches single-segment topics only.
  - `**` anywhere but last, and `*` inside a segment (`a*b`), are literals.
  - The syntax is ginject's own contract and never the backend's.
- **Adapters (R-3):** each adapter translates to the closest native superset and then filters locally with `pattern.Match`, so semantics are identical on every backend.

  | Backend | One segment | One or more at the end | Translation |
  |---|---|---|---|
  | NATS | `*` | `>` | exact 1:1 |
  | RabbitMQ | `*` | `#` (zero or more) | filter out the zero-segment match |
  | MQTT | `+` | `#` (zero or more); separator `/` | map `.` to `/`, then filter |
  | Redis PSUBSCRIBE | `*` also matches `.` | — | subscribe a superset, then filter |

  The cost: Redis, MQTT and RabbitMQ may deliver extra messages that the adapter drops.
- **Required at R-2:** restrict segment characters to `A-Z a-z 0-9 _ -`, and reject empty segments, so every topic valid in memory is valid on every backend (NATS reserves `* >` and spaces, MQTT `+ # /`, Redis `* ? [ ] \`).
- **`MaskFields` follows the same syntax:** `user.*` covers one level and `user.**` covers all levels.
- **Found while migrating:** a wildcard rule only masked leaves, so `password.*` matching a map-valued `password.person` left its children visible. This is fixed in `/pair-review log`: a rule that matches a node masks that node's whole subtree.

## D-031 `log` review: locking, limits, masking and handler fixes

- **Status:** decided 2026-11-10 (P2, `/pair-review log`).
- **Concurrency:**
  - **C1:** `prettyHandler` formats into a buffer from a `sync.Pool`, outside the lock, and locks only around `writer.Write`. Buffers above 64 KB are not returned to the pool. Measured with a prototype at 12 CPUs in parallel: 1.75 → 0.77 µs per record. Approved use of `sync.Pool`.
  - **C2:** writes stay synchronous, like `slog`, `zap`, `zerolog` and `logrus`. A slow writer blocks every logging goroutine; this is documented. An opt-in async writer is on the ROADMAP.
  - **C3:** `NewLog` keeps master's `sync.Once` singleton. A second call returns the first logger and ignores its options; this is documented.
- **Bounds (B4):** `maxDepth = 16` and `maxNodes = 1000` per record, in both the pretty tree and masking, for maps, slices, structs and anything nested in them (including `LogValuer` results). A `slog.Group` or self-referencing `LogValuer` passed directly as a top-level argument is not bounded yet, like stdlib slog (B-11). Past the depth the value prints `[MAX_DEPTH]`; past the node count, `… (N more)`. Fixed limits, not derived from RAM: a 100k-key map one level deep allocated 37 MB, so width matters more than depth, and container memory limits can change at runtime.
- **Pretty output:**
  - **B1:** a value implementing `error` prints `Error()`, and a `fmt.Stringer` prints `String()`, instead of the dereferenced struct.
  - **S1:** control bytes (< 0x20 and 0x7f) in the message, keys and every string value are escaped (`\n`, `\x1b`, …). Colours added by the framework are unaffected. A clean string costs one byte scan.
  - **B5:** `PrettyHandler`/`NewPrettyHandler` become unexported. The handler no longer embeds `slog.TextHandler`; it implements `Enabled`, `WithAttrs` and `WithGroup` itself and honours `AddSource`.
- **Masking:**
  - One rule for every pattern: each rule is matched with `pattern.Match` against every suffix of the attribute path, at every node. A matching node is replaced whole by `[REDACTED]`. This replaces master's two behaviours (exact rules matched by suffix, wildcard rules from the root and only at leaves).
  - Matching is case-insensitive, so the rule `password` covers the struct field `Password`.
  - Slices are traversed; their elements share the slice's path (`users` → `users.password`).
  - `slog.Attr` arguments, including `slog.Group`, are masked by their key and count as one argument.
  - **P1:** `transform` copies the args only when a value changes, and with no rules it builds no paths. `WrapLogger` can't return `next` unchanged, because it also turns tagged structs into maps for the JSON/Text handlers and custom loggers.
  - **P2:** masking runs only when the level is enabled.
- **After `go-reviewer` (2026-11-10):**
  - `slog.LogValuer` and `slog.Value` values are resolved before masking, at any depth, and the result is masked. At the top level an unchanged value is passed on as it was. Nested inside a map, struct or slice, a resolved value is always turned into plain Go values (a group becomes `map[string]any`), because the JSON handler prints a nested `slog.Value` as `{}` and the pretty handler would otherwise expand the original type field by field (second `go-reviewer` pass). The depth count carries through each `LogValuer` hop.
  - Empty-key groups inside a resolved `LogValuer` count against the node budget and the depth (third `go-reviewer` pass: an inline self-referencing group overflowed the stack).
  - Fourth `go-reviewer` pass (convention sweep): `slog.Attr` and `[]slog.Attr` nested as values, including a `slog.Value` that holds one, are masked like groups (they printed `password=…` in Text and Pretty, because `slog.Attr` is a `fmt.Stringer`); map children are sorted with `slices.SortFunc` instead of `sort.Slice` (access-log record: 42 → 35 allocs); single-letter parameters renamed; the shared constants moved to `fn.go` (D-034). `TextFormat`/`JSONFormat`/`PrettyFormat` now have their own `iota` block, so their values are 1, 2, 3 instead of 11, 12, 13; only the names are API (owner accepted, 2026-11-11).
  - Three fuzz targets guard the contract (B-9): a masked secret never reaches the Text, JSON or Pretty output; client text can't add lines or raw control bytes to the pretty output; `matchesSuffix` equals its slow reference.
  - The README gains a "What is supported" table. Inputs outside it are not part of the contract, and `log` is frozen for P2.
  - `format` collects up to 8 attributes in a stack array (approved hand-managed buffer); more than 8 falls back to `make`. 1 → 0 allocs per typical record.
  - `slog.Group` attributes are flattened with a key prefix in the pretty format; an empty-key group is inlined and an empty group dropped.
  - Bug fixes: a user attribute named `level` panicked the JSON/Text `ReplaceAttr` (inherited from master); `*Colored` lost its colour; escaped keys broke column alignment.
- **Removed:** the `unsafe` read of unexported fields with a `log` tag (R1), so tags on unexported fields are ignored; `LogOptions.Transports` and `LogTransport`, which nothing used; the `Level == -1` override.
- **Structure:** ANSI codes come from `internal/color`. `Color`/`Colored` move to `log_color.go`. Helpers used by more than one file (`kv`, `dereference`, `taggedFields`, `stringerType`, `errorType`) go to `fn.go`, so `log_tag.go` is gone and its tests are now `fn_test.go`. The tree builders stay in `log_handler.go`, their only caller, so there is no `log_tree.go`.

## D-032 Constructor settings are `opts *XxxOptions`

- **Status:** decided 2026-11-11.
- **Context:** main and master mix three styles: `Options` (`LogOptions`, `ExceptionOptions`, every `*ModuleOptions`, the middleware `*Options`), `Config` (`AccessLogConfig`, `WSConfig`, `HTTPConfig`, `retryConfig`) and `Opts` (`CacheOpts`).
- **Decision:**
  - The settings struct of a constructor is `XxxOptions`; for a module it is `XxxModuleOptions`.
  - The parameter is named `opts`, passed as a pointer, and `nil` means defaults: `NewLog(opts *LogOptions)`.
  - `Config`/`cfg` is reserved for the config module (`.env` loading), so it is never confused with `ConfigModule`/`ConfigService`. `Opts` is not used.
- **Retro-apply:** P1 and P2 code already follows it (`make conventions` checks it). Others are applied when each package is ported: `AccessLogConfig` → `AccessLogOptions` and `common.HTTPConfig` → `HTTPOptions` (P3), `WSConfig` → `WSOptions` (R-2), `CacheOpts` → `CacheModuleOptions` (cache module), `retryConfig` → `retryOptions` (httpclient).

## D-033 Ported code is kept in line with rule changes

- **Status:** decided 2026-11-11.
- **Context:** the port runs over many steps while CLAUDE.md keeps changing. D-032 and the embedded-field literal rule both arrived after P1 and P2 were written, so early steps could drift from later rules.
- **Decision:** three layers.
  1. **Mechanical checks in `make check`.** The `predeclared` linter (no shadowed builtins) and `make conventions` over `PORTED_DIRS`: `go fix -embedlit=false` must propose nothing, D-032 names (`XxxConfig`/`Opts`/`cfg`) are rejected, and ported tests use `test.DiffMessage` (D-002). Each step adds its packages to `PORTED_DIRS`.
  2. **Retro-apply.** Every rule change gets a D-xxx with a Retro-apply line. Affected ported packages are fixed in the same change or tracked in PORT.md "Convention sweep".
  3. **Convention sweep at each step gate** (DoD 4b): `go-reviewer` checks every ported package, not only the diff. A final sweep runs when the port is finished.
- **Retro-apply (this decision and D-032):** `make conventions` passes on `internal`, `log`, `pattern` and `trace`; `predeclared` reports nothing. Nothing to track.

## D-034 `fn.go` holds every shared unexported identifier

- **Status:** decided 2026-11-11.
- **Context:** the placement rule named helper functions only. `log` kept `maxDepth`/`maxNodes`/`maxDepthPlaceholder` in `log_handler.go` and the `label*` constants in `log.go`, while other files used them.
- **Decision:** the `fn.go` rule covers unexported functions, constants, variables and types used by two or more files of a package.
- **Retro-apply:** `log` fixed in P2. Every other ported package has a single code file, so nothing else is affected.

## D-035 Every `b.RunParallel` benchmark has an atomic sink

- **Status:** decided 2026-11-11.
- **Decision:** CLAUDE.md's rule stands for every `RunParallel` benchmark, even when the call has side effects: each goroutine counts locally and adds to a package-level `atomic` value at the end. A plain package-level variable written from every goroutine is a data race under `-race`.
- **Retro-apply:** `internal/crypto` (`BenchmarkUUID_Parallel` raced; now `atomic.Pointer[string]`), `internal/event` (`Emit` parallel) and `log` (`Handle` parallel) fixed in P2.
