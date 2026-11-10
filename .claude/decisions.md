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
