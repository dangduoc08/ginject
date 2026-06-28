Optimize the code in $ARGUMENTS for performance and correctness. If no argument is given, optimize the file currently open in the IDE or the file most recently discussed.

## Steps

1. **Read** the target file(s) fully before touching anything.

2. **Check tests and benchmarks** for the target package:
   - File naming convention: tests go in `<filename>_test.go`, benchmarks go in `<filename>_bench_test.go` (e.g. for `str.go` → `str_test.go` and `str_bench_test.go`).
   - If no test file exists, create `<filename>_test.go` with meaningful edge cases before making any changes.
   - If no benchmark file exists, create `<filename>_bench_test.go` with realistic, large-enough inputs (e.g. 1000+ iterations, real-world-sized data) that can actually reveal performance differences.
   - If both files already exist, review them and **add missing test cases** for any untested functions or uncovered edge cases before proceeding.
   - When writing test assertions, use `t.Error(test.DiffMessage(actual, expected, "desc"))` (import `"github.com/dangduoc08/ginject/internal/test"`) instead of raw `t.Errorf` format strings.
   - For any type whose methods read or write shared state (struct fields, maps, slices) across calls, add a concurrency test that spawns goroutines calling those methods together (`sync.WaitGroup` + `go func()`, see `internal/ds/trie_test.go`'s `TestTrieConcurrentInsertAndFind_RequiresExternalLock` for the pattern). It must pass under plain `go test`; run it once with `go test -race` to see whether it's flagged. If the type owns its own lock, the race report is a real bug — fix it. If the type is a bare data structure that documents "caller must synchronize externally," keep the test as-is (it stays green without `-race`) — it is a regression guard, not a bug report.

3. **Run existing tests** to establish a baseline — all must pass before proceeding.

4. **Run benchmarks** (`go test -bench=. -benchmem`) to record baseline numbers.

5. **Analyze** — allocation reduction and raw performance (ns/op) are the top priority of this command; look for these first, then the remaining correctness/style/security items:
   - Repeated work inside loops (regex compile, object creation, repeated allocations)
   - O(n²) patterns that can be reduced (e.g. string `+=` → `strings.Builder`)
   - Unnecessary allocations on hot paths (`make` / `[]T{}` that could be `var`)
   - Standard library functions that replace manual implementations (`strings.HasPrefix`, `strings.TrimPrefix`, `strings.HasSuffix`, `strings.TrimSuffix`, etc.)
   - Dead code or redundant conditions (`A || (A && B)` → `A`)
   - Potential panics from missing bounds checks
   - **Normalisation before capture**: if a function normalises an input field (e.g. `nil → "*"`, `[]string → map`) and then assigns the result to an output struct, ensure the assignment happens **after** all normalisation steps, not before. Assigning before normalisation silently captures the pre-normalised value and the output struct carries stale data.
   - **Hot-path initialisation**: if a function builds options/config structs (string joins, map construction, defaults) and is called on every request, move that work into a one-time initialisation step (e.g. `NewMiddleware`, a constructor, or `sync.Once`) and cache the result.
   - **Security**: SQL/command injection via string concat, hardcoded secrets, unvalidated external input used in file paths or exec calls, missing TLS verification, use of `math/rand` where `crypto/rand` is required

6. **Concurrency & deadlock audit** — for every type or function that touches shared state, work through this checklist:

   **Race conditions**
   - [ ] Is every read and write of shared state protected by the same lock (or `sync/atomic`)? Check that RLock is not used where a write can race.
   - [ ] Does `Emit`/dispatch copy listeners into a **private slice** before releasing the lock? Holding a reference to the original backing array is not safe — a concurrent `Off` shifts elements in place while you iterate.
   - [ ] For "fire-once" semantics: are once-listeners **removed atomically before execution** (steal pattern: take slice reference + delete map entry in the same critical section)? Removing *after* execution has a window where a second concurrent caller snapshots the same listeners and fires them again.
   - [ ] Is I/O, `fmt.Print*`, or any blocking call done **outside** the lock? Holding a mutex across I/O serialises all other callers for the duration of the syscall.
   - [ ] Are values that need to be passed out of a critical section (counts, flags) captured into local variables before `Unlock`, then used after? Never read a shared field after releasing the lock without re-acquiring it.
   - [ ] Run `go test -race ./...` and fix every report before proceeding.

   **Deadlocks**
   - [ ] Does any callback, listener, or injected function call back into the same type (recursive locking)? Ensure the lock is **released before invoking external code**.
   - [ ] Is the lock upgrade path (RLock → WLock) safe? `sync.RWMutex` does not support upgrading — you must `RUnlock` first, creating a window; if the window matters, use `Lock` for the whole operation.
   - [ ] Is `Unlock` (or `RUnlock`) called on **every** return path, including early `return` and `panic`? Prefer `defer mu.Unlock()` immediately after `mu.Lock()` unless the lock must be released before a blocking call.
   - [ ] Are channel sends/receives inside a locked section? A channel operation can block indefinitely, holding the lock and starving other goroutines.
   - [ ] For `sync.WaitGroup`: is `Add` called before the goroutine is spawned (not inside it)? Is `Wait` called only after all `Add`s?

7. **Lint** — if a `.golangci.yml` exists, run `golangci-lint run ./...` and fix all reported issues. If no config exists, run with default linters. Fix issues in this order:
   - Errors and bugs (`errcheck`, `staticcheck`, `govet`)
   - Security (`gosec`)
   - Style and correctness (`gocritic`, `errorlint`, `exhaustive`)
   - Cosmetic (`godot`, `goconst`, `misspell`)

8. **Apply** changes one at a time. After each change:
   - Run tests — if any fail, revert that specific change and note why.
   - For concurrency bugs, run with `-race` flag: `go test -race ./...`
   - Allocation reduction and performance gains are the top priority — apply them whenever the benchmark shows a measurable improvement, even at a modest readability cost. Only keep the simpler version when the delta is within benchmark noise (no measurable gain).
   - Always use functions available in newer versions of the language/stdlib over manual equivalents.
   - Security, race condition, and allocation/performance fixes are always applied regardless of readability trade-off.

9. **Run benchmarks again** and compare to baseline.

10. **Report** results in **two versions — English first, then Vietnamese** — each containing:
   - A concise table: what was optimized / fixed, why (one-line reason), and benchmark delta (ns/op, allocs before → after).
   - A separate section for concurrency fixes if any were applied.
   - A separate section for lint fixes if any were applied.

   **English version** format:
   ```
   ## Optimization Report

   | Change | Why | ns/op before → after | allocs before → after |
   |--------|-----|-----------------------|------------------------|
   | ...    | ... | ...                   | ...                    |

   ### Concurrency fixes (if any)
   | Issue | Fix |

   ### Lint fixes (if any)
   | Issue | Fix |
   ```

   **Vietnamese version** format (immediately after English, under a `---` divider):
   ```
   ---
   ## Báo Cáo Tối Ưu

   | Thay đổi | Lý do | ns/op trước → sau | allocs trước → sau |
   |----------|-------|-------------------|---------------------|
   | ...      | ...   | ...               | ...                 |

   ### Sửa lỗi concurrency (nếu có)
   | Vấn đề | Cách sửa |

   ### Sửa lỗi lint (nếu có)
   | Vấn đề | Cách sửa |
   ```

## Rules

- Never change the public API (function signatures, exported names).
- Do not add features, error handling, or abstractions beyond the task.
- Do not add comments to the code.
- If a change would affect callers outside the file, flag it before applying.
- **Allocation and performance optimizations are the top priority of this command** — apply any change with a measurable allocs/op or ns/op improvement, even at a modest readability cost. Only keep the simpler version when the benchmark delta is within noise (no measurable gain).
- **Prefer built-ins over manual equivalents**: if the standard library (or language built-in) already provides an equivalent function, use it instead of a manual implementation — unless the built-in form is demonstrably less readable with no meaningful correctness or performance gain.
- **Each package owns its output**: a package must produce correct, clean output by itself — do not leave known artifacts (e.g. double slashes, trailing separators, off-by-one boundaries) for downstream callers to normalise. If the fix belongs in this file, apply it here.
