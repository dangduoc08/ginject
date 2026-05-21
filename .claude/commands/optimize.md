Optimize the code in $ARGUMENTS for performance and correctness. If no argument is given, optimize the file currently open in the IDE or the file most recently discussed.

## Steps

1. **Read** the target file(s) fully before touching anything.

2. **Check tests and benchmarks** for the target package:
   - File naming convention: tests go in `<filename>_test.go`, benchmarks go in `<filename>_bench_test.go` (e.g. for `str.go` → `str_test.go` and `str_bench_test.go`).
   - If no test file exists, create `<filename>_test.go` with meaningful edge cases before making any changes.
   - If no benchmark file exists, create `<filename>_bench_test.go` with realistic, large-enough inputs (e.g. 1000+ iterations, real-world-sized data) that can actually reveal performance differences.
   - If both files already exist, review them and **add missing test cases** for any untested functions or uncovered edge cases before proceeding.
   - When writing test assertions, use `t.Error(testutils.DiffMessage(actual, expected, "desc"))` (import `"github.com/dangduoc08/ginject/testutils"`) instead of raw `t.Errorf` format strings.

3. **Run existing tests** to establish a baseline — all must pass before proceeding.

4. **Run benchmarks** (`go test -bench=. -benchmem`) to record baseline numbers.

5. **Analyze** — look specifically for:
   - Repeated work inside loops (regex compile, object creation, string concat with `+=`)
   - O(n²) patterns that can be reduced (e.g. string `+=` → `strings.Builder`)
   - Unnecessary allocations on hot paths (`make` / `[]T{}` that could be `var`)
   - Standard library functions that replace manual implementations (`strings.HasPrefix`, `strings.TrimPrefix`, `strings.HasSuffix`, `strings.TrimSuffix`, etc.)
   - Dead code or redundant conditions (`A || (A && B)` → `A`)
   - Potential panics from missing bounds checks
   - **Concurrency & race conditions**: shared state accessed without synchronization, goroutines leaking, channels never closed, `sync.Mutex` locked but not unlocked on all paths, `sync/atomic` misuse
   - **Security**: SQL/command injection via string concat, hardcoded secrets, unvalidated external input used in file paths or exec calls, missing TLS verification, use of `math/rand` where `crypto/rand` is required

6. **Lint** — if a `.golangci.yml` exists, run `golangci-lint run ./...` and fix all reported issues. If no config exists, run with default linters. Fix issues in this order:
   - Errors and bugs (`errcheck`, `staticcheck`, `govet`)
   - Security (`gosec`)
   - Style and correctness (`gocritic`, `errorlint`, `exhaustive`)
   - Cosmetic (`godot`, `goconst`, `misspell`)

7. **Apply** changes one at a time. After each change:
   - Run tests — if any fail, revert that specific change and note why.
   - For concurrency bugs, run with `-race` flag: `go test -race ./...`
   - Prefer readable, maintainable code over micro-optimizations. If the performance gain is marginal, keep the simpler version.
   - Always use functions available in newer versions of the language/stdlib over manual equivalents.
   - Security and race condition fixes are always applied regardless of readability trade-off.

8. **Run benchmarks again** and compare to baseline.

9. **Report** results in **two versions — English first, then Vietnamese** — each containing:
   - A concise table: what was optimized, why (one-line reason), and benchmark delta (ns/op, allocs before → after).
   - A separate section for lint fixes if any were applied.

   **English version** format:
   ```
   ## Optimization Report

   | Change | Why | ns/op before → after | allocs before → after |
   |--------|-----|-----------------------|------------------------|
   | ...    | ... | ...                   | ...                    |

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

   ### Sửa lỗi lint (nếu có)
   | Vấn đề | Cách sửa |
   ```

## Rules

- Never change the public API (function signatures, exported names).
- Do not add features, error handling, or abstractions beyond the task.
- Do not add comments to the code.
- If a change would affect callers outside the file, flag it before applying.
- Readability wins over performance when the gain is small — do not make code harder to read for marginal improvements.
- **Prefer built-ins over manual equivalents**: if the standard library (or language built-in) already provides an equivalent function, use it instead of a manual implementation — unless the built-in form is demonstrably less readable with no meaningful correctness or performance gain.
