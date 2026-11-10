# P1 reading guide: `internal/`

P1 brings 7 packages from master (`13242d4`) onto main. That's about 700 lines of non-test code across 7 files. **No caller has been switched from `utils` yet**, so the framework's runtime behaviour has not changed.

## Decisions applied

| ID | Decision |
|---|---|
| D-002 | Keep only `internal/`. `utils` is deleted once its last caller has moved (end of P3) |
| D-008 | Leave out unused code: `crypto.Random`/`withCharset`, the whole `internal/runtime` package, `slice.ToUnique`, `ds.Trie.Remove`/`removeGlobChild`, and their tests and benchmarks |
| D-009 | Port `internal/ds` (the route trie) in P1. Read it again in P3 next to `routing` |
| D-010 | Take master's behaviour. Its bug fixes take effect when callers switch in P2/P3 |
| D-011 | `color`: each style resets only its own attribute (fg 39, bg 49, bold/dim 22, italic 23), so nesting is safe |
| D-012 | `color`: the fast path runs only with no args and no `%`. Otherwise `Sprintf`, so `%` behaves like printf |
| D-013 | `color`: the 18 `Fmt*` call one helper, `wrap`. Same API, and `go vet` still checks the format strings |
| D-014 | `crypto.UUID() string`: drop the dead error return |
| D-015 | `test.DiffMessage` prints values with `%#v`, so `1`/`"1"`, `""`/`" "` and `nil`/empty slices can be told apart |
| D-016 | `test.DiffMessage`: desc on the `file:line` line, aligned columns, always coloured. `test` doesn't import `color` |
| D-017 | `str.Enclose` fast path for already-clean paths, proven equal to the slow path |
| D-018 | `num.F64ToAnyNum` returns `(any, bool)` and rejects overflow, fractions into integers, negatives into unsigned, and NaN/Inf |
| D-019 | `slice.StrParse*` return `([]T, bool)`; `Find`/`FindIndex` removed |
| D-020 | Route trie: cached `$`/`*` children (−37–63%), backtracking on greedy miss, `Len`/`ToJSON` removed |
| D-021 | Naming: typo, builtin shadowing, `kind`, `s` in package `str`; `crypto`/`ds`/`Raw` kept by owner choice |

Compared with master, every package was reviewed and changed by the decisions above (D-008 … D-021): unused code removed, bugs fixed, hot paths optimised and names cleaned up. Use `git show master:<f> | diff - <f>` to see each change. P1 also adds concurrency, equivalence and hostile-input tests.

## Reading order: 3 parts

Read in dependency order. `test` depends on `color`, and `ds` depends on `str`.

### Part 1: formatting and conversion helpers (~300 lines)

1. **`internal/color/color.go`** (~95, reviewed and fixed in P1): `Fmt*` wraps text in ANSI codes through `wrap(openCode, resetCode, format, a...)`.
   - **Hot/cold map: cold.** All 38 call sites in master are cold: bootstrap panics and errors (`core/fn.go`, `common/*`), the startup banner (`logBoostrap`), event-registration warnings, and `NewPrettyHandler`, which runs once.
     - Logging itself **is** hot when access log is on (request → `accesslog` → `logger.Info` → `PrettyHandler.Handle`).
     - `Handle` doesn't call `color`. It writes level labels precomputed with `color` at construction, plus `log`'s own ANSI constants.
     - At P2, don't move `Fmt*` calls into `Handle`. Share constants instead.
   - **Fast path:** no args and no `%` in format means plain concatenation. Anything else goes through `fmt.Sprintf` (D-012).
   - **Reset per style** (D-011): fg colours close with `39`, backgrounds with `49`, bold/dim with `22`, italic with `23`. Nothing emits `\x1b[0m`, so `FmtBold("%s", "a "+FmtRed("b")+" c")` keeps `" c"` bold.
   - `go vet` still treats every `Fmt*` as a printf wrapper. A probe showed it flags `FmtRed("%d")` and non-constant formats.
   - Tests use `test.DiffMessage`. Since D-016 `test` no longer imports `color`, so there is no import cycle.
2. **`internal/test/test.go`** (~20, reviewed and fixed in P1): `DiffMessage(actual, expected, desc)` replaces `utils.ErrorMessage`. It is one `fmt.Sprintf` with a constant coloured format.
   - **Hot/cold map: cold.** Only `_test.go` files call it, and only when an assertion fails.
   - **D-016 layout:** `desc` stays on the `file:line` line, and the columns line up:
     ```
     trie_test.go:42: route list
         expected: []string{"/users/"}
         actual:   []string{"/users"}
     ```
   - **Colours:** desc is bold, labels are dim, expected is green and actual is red.
     - Colours are always on, with no environment switch (owner's choice).
     - `test` defines its own escape constants and doesn't import `color`, so there is no import cycle.
   - In `color`'s own tests, the values are coloured strings. `%#v` prints their codes as text (`"\x1b[31m..."`) inside the green/red value colour. That is intended, because those codes are what the test checks.
   - **D-015:** values are printed with `%#v`, so a failure never shows two different values as equal. Without it, `1` vs `"1"`, `""` vs `" "` and `[]string{}` vs `nil` look the same; the last pair encodes to `[]` and `null` in JSON.
   - Cold path: it runs only when an assertion fails. It costs about 700–900 ns and 9–12 allocs, which doesn't matter here.
   - Tests check the layout with the codes stripped (7 cases, including the ones `%v` made ambiguous) and the exact coloured output.
3. **`internal/num/num.go`** (~75, reviewed and fixed in P1): `F64ToAnyNum(f64, kind) (any, bool)` converts a JSON number (`float64`) to the struct field's kind.
   - **Hot/cold map: HOT.** Handler DTO → `ctx.Body.Bind` (http_body.go:110) → `BindStruct` (binding.go:14) → `F64ToAnyNum` (binding.go:87, 167), once per numeric field in the request body. The client controls the input.
   - **D-018:** it no longer corrupts values silently. `ok` is false for overflow (`1e20` → `int64` used to give `MinInt64`), for fractions into integer kinds (`1.9` → `int` used to give `1`), for negatives into unsigned kinds, for NaN/Inf, and for unsupported kinds.
   - `toInteger[T]` is a generic, inlinable helper. Bounds are exact powers of two, such as `[-2^63, 2^63)`, and wholeness is checked with `float64(T(f)) == f`.
   - Boxing into `any` costs 1 alloc for values above 255 and for floats. Only the caller can remove that (P3: `reflect.Value.SetInt`).

   | Input | master | now |
   |---|---|---|
   | 42 → int | 4.3 ns, 0 allocs | 5.5 ns, 0 allocs |
   | 12345 → int | 14.0 ns, 1 alloc | 15.8 ns, 1 alloc |
   | 1700000000123 → int64 | 12.7 ns, 1 alloc | 18.3 ns, 1 alloc |
   | 200 → uint8 | 2.8 ns | 4.7 ns |
   | 3.14 → float64 | 16.6 ns, 1 alloc | 14.3 ns, 1 alloc |
   | 1e20 → int64 (overflow) | 15.7 ns, 1 alloc, **wrong value** | 3.7 ns, 0 allocs, rejected |

4. **`internal/str/str.go`** (~110, reviewed and fixed in P1). `AddBegin`, `AddEnd`, `IsLower` and `IsUpper` were removed because nothing in master calls them (D-008, extended). Callers in master:
   - `Segment` is used by the trie on **every request**;
   - `Enclose` is used at `Create()` and also in `Router.Match` on **every request**;
   - `RemoveSpace` is used by `ctx.JSONP`;
   - `RemoveDup` is used by static file serving.

   **Hot/warm/cold map** (traced from `App.ServeHTTP` in master):

   | Function | Class | Calls per request | Cost |
   |---|---|---|---|
   | `Segment` | hot (`trie.Find` ← `Router.Match`) | about segments + 1, ×2 on the version-neutral retry | 44 ns for 6 calls, 0 allocs |
   | `Enclose` | hot in `Router.Match` and `ws.isWSPath`; cold in 13 `Create()`-time call sites | 1–2 | 59.2 ns, 1 alloc (27.8 ns, 0 allocs with a trailing `/`) |
   | `isASCIISpace` | hot, inside `Enclose` | per byte | — |
   | `RemoveSpace` | warm, client-triggered (JSONP) | 1 | 568 ns for 100 chars |
   | `RemoveDup` | warm, client-triggered (static files) | 1 | 162 ns, 1 alloc |

   **Before/after** (6-run mean; before = `encloseSlow`, identical to master's `Enclose`):

   | Input | Before | After |
   |---|---|---|
   | `/api/v1/users/123/orders` | 83.1 ns, 1 alloc | 59.2 ns, 1 alloc |
   | `/api/v1/users/123/orders/` | 81.0 ns, 1 alloc | 27.8 ns, 0 allocs |
   | `/users` | 41.5 ns, 1 alloc | 34.9 ns, 1 alloc |
   | `/` | 20.6 ns, 1 alloc | 3.0 ns, 0 allocs |
   | `//users//profile//settings//` (dirty) | 93.9 ns | 110.0 ns (+17%) |
   | `users/{id}` (route pattern, cold) | 54.8 ns | 55.9 ns |

   **D-017 fast path in `Enclose`:** when the input is already clean (starts with `sep`, no space, no `//` or `**`), it is returned as is, or with `sep` appended. Everything else goes to `encloseSlow`, the original algorithm. A test proves the two are equal over 400k random inputs.

   Tests cover every function at 100%. The `Segment` edge cases are pinned down: with no trailing sep it returns the rest **including** the leading sep, such as `"/b", -1`, and the trie ignores that result.

   Two caller bugs were found and recorded for P3:
   - `Match` strips spaces from request paths, so `/us%20er` matches `/user`;
   - the JSONP callback name is not validated, which allows XSS.

### Part 2: generics and UUID (~180 lines)

5. **`internal/slice/slice.go`** (~150, reviewed and fixed in P1):
   - **Hot/cold map:**

     | Function | Class | Caller chain (master) |
     |---|---|---|
     | `StrParse*` (16) | hot, user API | `ctx.Query/Form/Header/Param.Bind` → `BindStrArr` (slice fields) |
     | `Get` | hot, user API | `BindStrArr` (18 sites), `BindFile` |
     | `Iter` | hot, user API | `ctx.Body.Bind` → `bindArray` (ctx/fn.go:336) |
     | `Filter` | hot, user API | `ctx.Body.Get/Set` (http_body.go:48, 73) |
     | `Map` | warm (upload `BindFile`), cold (regexp init `routing/fn.go:11`, devtool) | |
     | `Find`, `FindIndex` | unused, removed | — |

   - **D-019:** the typed `StrParse*` functions return `([]T, bool)` through `parseAll[T]`, so bad input such as `?ids=1,abc,3` (which used to become `[1 0 3]`) is reported instead of zeroed. `StrParseAny` can't fail and still returns `[]any`.
   - `Iter` recurses into nested `[]any` and calls `cb` on the sub-array too, deepest first. The test pins down the visit order.
   - Benchmarks have to let the result escape, as the real caller does. Without that, the old code looked 0-alloc only because the compiler kept the unused result on the stack.

     | `StrParse*` with 5 values | master | now |
     |---|---|---|
     | Int | 85.9 ns, 2 allocs | 70.5 ns, 2 allocs |
     | Int64 | 125.0 ns, 2 allocs | 104.0 ns, 2 allocs |
     | Uint8 | 81.9 ns, 2 allocs | 71.5 ns, 2 allocs |
     | Float64 | 171.2 ns, 2 allocs | 171.6 ns, 2 allocs |

   - Tests cover every branch at 100%: each parser's success, overflow and garbage cases, `Get` bounds, empty and nil input.
6. **`internal/crypto/crypto.go`** (~25, reviewed and fixed in P1): `UUID() string` is a v4 UUID built from `crypto/rand`, and it matches RFC 9562 (the successor of 4122).
   - It sets version 4 in byte 6 and variant `10` in byte 8, and writes lowercase hex in `8-4-4-4-12` form into a stack array. It makes 1 alloc, for the final `string`.
   - **Hot/cold map: HOT.** `App.ServeHTTP` (core/app.go:157) → `c.Init` (:163) → `ctx.SetID` (http_context.go:42) → `UUID`, once per HTTP request. It is warm for WS connection IDs (core/ws.go:122) and broker subscription IDs. User-level code also uses it for security tokens (sample/shop), so it must stay crypto-grade.
     - Breakdown: `crypto/rand` 16 bytes takes 346–391 ns (about 90%), hex plus string 44 ns. `math/rand/v2` would take 13.6 ns.
     - Decision: keep `UUID` as is. At P3, the request ID moves to `math/rand/v2` (about 60 ns, from about 400 ns, about 11% of a request). Linux cost is logged in ROADMAP.
   - **D-014:** it returns no error, because since Go 1.24 `crypto/rand.Read` never returns one. `_, _ =` satisfies `errcheck`.
   - Tests cover format, lowercase hex, version, variant (`8/9/a/b` at index 19), and uniqueness across 32 goroutines × 1000 calls.
   - **Cost:** about 404 ns per call on Go 1.27.3 (430–470 ns on 1.25.5). Under `RunParallel`, ns/op rises with CPUs (431 → 1178 ns from 1 to 12 CPUs on macOS), which looks like a global lock in the OS random source.
   - **Caller issue for P3:** master's request ID keeps only 8 hex chars, so it collides with 50% probability after about 77k requests. Recorded in PORT.md.

### Part 3: route trie (~210 lines). The most important part

7. **`matchWildcard`** (now at the end of `trie.go`) matches a segment against a glob such as `*.html` or `a*b*c`, without regexp and with 0 allocs. It moved from `fn.go` because only `trie.go` uses it (function-placement rule).
8. **`internal/ds/trie.go`** (~215, reviewed and fixed in P1):
   - **Hot/cold map:**

     | Function | Class | Caller chain (master) |
     |---|---|---|
     | `Find` (`findGreedy`, `search`) | hot, 1–2 per request | `ServeHTTP` → `handleRequest` → `Router.Match` (router.go:177) |
     | `matchWildcard` | hot when glob routes exist | inside `Find` |
     | `Insert`, `NewTrie` | cold | `Create()` → `init*` → `route.Use/For/Add` → `Router.push` (router.go:86). There are no inserts after `Listen` |
     | `Len`, `ToJSON`, `genTrieMap` | unused, removed | — |

   - **Structure:** `Children` is a map of segments. `param` and `wildcard` cache the `"$"` and `"*"` children, and are set by `Insert`. `globChildren` lists the segments that contain `*`.
   - **`Find` = `findGreedy`, then `search` if needed (D-020).**
     - `findGreedy` is master's algorithm. For each segment it tries exact, then param, then `*`, then glob, without going back. It remembers the nearest terminal `*` child as the `wildcardRaw` fallback, and with the cached pointers it does 1 map lookup per segment.
     - Only when the greedy walk finds **neither a route nor a wildcard** does `search` run. It is a depth-first walk in the same priority order, popping params when it backs out. For request segments without `$` or `*`, whenever master returns anything, the result is identical (D-020). Segments containing `$` or `*` may differ from master, as intended.
     - `exactChild` ignores placeholder keys (`$`, `*`, globs). A literal `$` or `*` request segment therefore takes the param or wildcard branch, each node is reachable through one branch only, and `search` stays linear: a hostile path at depth 64 takes 19 µs. This also fixes master's `/users/$/` matching with no param.
     - Example: `/users/me/profile/` now matches `/users/$/profile/` with `me` as the param.
     - Not everywhere: if a terminal `*` sits above the dead end, master's wildcard result is kept. With `/a/*/`, `/a/b/x/` and `/a/$/c/`, the path `/a/b/c/` still gives `/a/*/`.
   - **Allocs:** routes with params cost 1 alloc (the `paramVals` slice, 64 B). Static routes have none.
   - **Concurrency:** there is no lock. Concurrent `Find`s are safe (64×500 test). `Insert` during `Find` races, but master only inserts during `Create()`.
   - **Tests:**
     - master's `Find`, kept verbatim as `findMaster`, must agree with the new `Find` wherever master returned a route or a wildcard (500k random lookups);
     - on master misses, a brute-force matcher must agree with every result (23,344 of 500k random lookups recovered);
     - regression tests: params stay with their route under wildcard fallback; a literal `$` is captured as the param; a hostile depth-24 placeholder path takes under 10 ms;
     - the cached-pointer invariants are checked;
     - coverage is 100%.
   - **`internal/ds/MATCHING.md`** is a committed table of real `Find` results: 10 scenarios, 33 request paths. `TestMatchingTableIsUpToDate` regenerates it and fails when the file drifts. Run with `-update-matching` to rewrite it.

## Benchmarks

### `color` (Intel i7-9750H; master numbers measured with a package-level sink, now with `b.Loop`; master: Go 1.25.5, range over 2 runs of 6 each; now: Go 1.27.3, `b.Loop`, mean of 5 runs)

| Benchmark | master ns/op | now ns/op | master allocs | now allocs | Note |
|---|---|---|---|---|---|
| FmtRed_NoArgs | 37.8–40.6 | 51.6 | 1 | 1 | `IndexByte` check, and `wrap` is not inlined |
| FmtRed_NoArgsPercent (`"cpu 100%%"`) | 36.9–45.1 | 118.6 | 1 | 2 | Now goes through `Sprintf` and prints the **correct** `100%` (master printed `100%%`) |
| FmtRed_Args | 168–171 | 158.4 | 2 | 2 | Same (noise) |
| FmtBGRed_Args | 168 | 169.3 | 2 | 2 | Same |
| FmtPurple_Args | 173–185 | 175.0 | 2 | 2 | Same |
| FmtBold_Nested | 201–216 | 231.6 | 4 | 5 | The inner constant call is no longer folded at compile time |

This is cold path, so the code is kept simple on purpose. A faster version with `fmt.Appendf` and `unsafe.String` (1 alloc, about 15% faster with args) was tried and rejected for readability (D-013).

### `ds`

Before = master's `trie.go`, after = now. 6-run mean, Go 1.27.3, Intel i7-9750H, noisy machine:

| Path | Before | After |
|---|---|---|
| `/health/` | 57.6 ns | 36.0 ns |
| `/api/v1/users/123/orders/` | 341.1 ns, 1 alloc | 159.2 ns, 1 alloc |
| `/api/v1/products/9/reviews/3/` | 446.0 ns, 1 alloc | 167.6 ns, 1 alloc |
| `/api/v1/users/me/orders/` | 201.9 ns, **wrong 404** | 271.4 ns, 16 B, correct match (backtracking) |
| `/api/v2/nothing/` (real 404) | 110.2 ns | 143.2 ns (backtracking walk) |
| bench `Static` / `WithParam` / `DeepParam` | 109.7 / 197.7 / 304.5 ns | 47.4 / 80.1 / 130.4 ns |

The older table below is from the first P1 run, before D-020:

| Benchmark | ns/op | B/op | allocs/op |
|---|---|---|---|
| TrieFind_Static | 73.7 | 0 | 0 |
| TrieFind_WithParam | 146.4 | 64 | 1 |
| TrieFind_DeepParam | 270.9 | 64 | 1 |
| TrieFind_NoMatch | 117.8 | 0 | 0 |
| MatchWildcard | 25.1 | 0 | 0 |

There's no before/after comparison, because no request-path code calls `internal/` yet. The comparison with main's `routing/trie.go` belongs in P3.

## Bugs proven in this step

| Bug | Where | Handled in |
|---|---|---|
| `StrAddBegin("a","/x")` panics | main `utils` | Fixed in `internal/str`, takes effect at P2/P3 |
| `StrAddEnd("ab","abc")` returns `"ab"` | main `utils` | Same as above |
| `NumF64ToAnyNum(-1, Uint8)` returns `int` | main `utils` | Fixed in `internal/num` |
| Trie doesn't backtrack | `internal/ds` (master) | **Fixed in P1** (D-020) |
| Trie `Insert` during `Find` races | `internal/ds` (master) | P3 |

## Lens pass for the whole step (added after the five-lens rule)

| Lens | Result | Evidence |
|---|---|---|
| Concurrency | ✅ No shared mutable state in any package. Every `var` is function-local, and the library code has no `go func`, `sync.*` or `unsafe`. `ds.Trie` is safe for concurrent `Find`. `Insert` during `Find` races, but inserts only happen at `Create()` (recorded for P3) | `grep -n '^var\|go func\|sync\.' internal/*/[a-z]*.go`; `TestTrieFind_ConcurrentReaders`, `TestUUID_ConcurrentUnique` under `-race` |
| Security (DoS) | ✅ A 1 MB hostile path (200k segments) takes 4 µs in `Find`, because the walk stops at trie depth. A 100k-char glob segment takes 7 µs. A 9k-deep JSON array takes 117 µs in `slice.Iter`, and `encoding/json` caps depth at 10k | throwaway probes in this P1 session |
| Security (silent corruption, injection) | Fixed: `num` overflow (D-018) and `slice` parse failures (D-019). For P3: `Router.Match` strips spaces from paths, and the JSONP callback allows XSS. `color` format strings must be constants (`go vet`) | probes in the D-018/D-019 entries |
| Readability and conventions | Fixed: typo `dimmensions`, `close` shadowing a builtin, `t` → `kind`, `wrap` placement, `str` → `s` in package `str` (D-021). Kept by owner choice: `crypto`, `ds`, `Raw`. `misspell` reports 0 issues and is now enabled in lint | `golangci-lint run --enable=misspell ./internal/...`; identifier scan |
| Correctness, errors, resources | ✅ No ignored errors left in `internal/`; the `strconv` ones were fixed. No growing maps or caches. Nothing to close | code review |
| API surface | n/a: everything is under `internal/` | — |
| Dependencies and binary | ✅ The standard library only. `CGO_ENABLED=0 go build ./...` works. `GOOS=linux GOARCH=386` builds `internal/` and compiles the `num` tests (`bits.UintSize`, no 64-bit `int` assumption). `linux/arm64` static build works | `go list -deps ./internal/...`; cross builds |
| Devtool readiness | `ds.Trie.ToJSON` was removed as unused. When devtool is rebuilt (R-1) and needs the route tree, add an export there, with a test | D-020 |
| Ecosystem compatibility | n/a for P1 | — |

## Go 1.27.3 upgrade (done during P1)

- **Toolchain:** go.mod is now `go 1.27.3` (was 1.25.5), golangci-lint is v2.15.0, and govulncheck is v1.9.0. CI reads the Go version from go.mod through `go-version-file`.
- **Forced dependency bump:** Go 1.27 moved HTTP/2 into the standard library, and x/net's `http2` server is now build-tagged `!(go1.27 && !http2legacy)`. grpc v1.81.1 stopped compiling (`undefined: http2.TrailerPrefix`). grpc is now v1.85.0 and x/net v0.62.0. govulncheck reports **no vulnerabilities**.
- **Modernizers:** `go fix -diff ./internal/...` found nothing in library code. It suggested `for range n` and `wg.Go` in 2 test files, and both were applied.
- **Benchmarks:** all of them now use `for b.Loop()`. Since Go 1.26, `b.Loop` keeps results alive without blocking inlining, so the sink variables are gone, except in `RunParallel` (`crypto`) and in `slice`'s `StrParse*` benchmarks. Those keep an escaping result (`escapedResult`) on purpose, as the real binding caller does.
- **For later steps:** Go 1.27 has `encoding/json/v2` (P3: response/binding speed), reflect iterators `Type.Fields`/`Methods`/`Ins`/`Outs` (P3: DI), the generally available `goroutineleak` profile (tests and devtool), and generic methods.

## How to verify

```bash
make check
go test -race -count=1 ./internal/... -v -run Concurrent
for f in $(git ls-tree -r --name-only master internal | grep -v runtime); do git show master:$f | diff - $f | grep '^>'; done
```

The last command prints every line that differs from master. Each difference maps to a decision listed at the top of this guide.
