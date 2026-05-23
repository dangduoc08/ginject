Implement or upgrade the feature described in $ARGUMENTS. If a file path is given, inspect it first and upgrade the existing implementation. If no argument is given, use the file currently open in the IDE or the most recently discussed file.

## Steps

1. **Read** the target file(s) fully. If no file is given, infer the target from context.

2. **Classify** the task:
   - **New**: the feature or function does not exist yet → implement from scratch.
   - **Upgrade**: code exists but is incomplete, insecure, or missing coverage → improve in-place.

3. **Design before writing**:
   - State the public API (exported types, function signatures) in one short paragraph.
   - List the security-relevant invariants that must hold (input validation, auth checks, injection prevention, etc.).
   - If the design would change callers outside the file, flag it and ask before proceeding.

4. **Implement** the feature:
   - Follow the existing package conventions (naming, error handling style, DI patterns).
   - Keep the public API stable — do not break existing call sites.
   - Do not add features, abstractions, or error handling beyond what was asked.
   - Prefer standard library over third-party packages for new dependencies.

5. **Security pass** — after writing the implementation, check each of the following and fix any issue found:
   - **Injection**: SQL/command/path built from external input → use parameterized queries, `filepath.Clean`, `exec.Command` with separate args.
   - **Auth**: endpoints or functions that mutate state must have authorization checks.
   - **Input validation**: all values from request body, query, headers, env, or config must be validated at the boundary; reject early.
   - **Secrets**: no hardcoded tokens, passwords, or keys; read from env/config only.
   - **Cryptography**: use `crypto/rand` for tokens/nonces, never `math/rand`; use constant-time comparison for secrets (`subtle.ConstantTimeCompare`).
   - **TLS**: never disable `InsecureSkipVerify`; do not downgrade to plain HTTP for internal calls.
   - **Denial of service**: bound request body sizes (`http.MaxBytesReader`), timeouts on outbound calls, rate-limit loops over external input.
   - **Information leakage**: do not expose stack traces, internal paths, or raw DB errors to external callers.
   - **Race conditions**: shared mutable state accessed from multiple goroutines must be protected (`sync.Mutex`, `sync/atomic`, or channels).
   - **OWASP Top 10 applicability**: briefly check which OWASP categories apply to this feature and note any that are not covered.

6. **Check tests and benchmarks**:
   - File naming: tests → `<filename>_test.go`, benchmarks → `<filename>_bench_test.go`.
   - If **no test file** exists → create one.
   - If **tests exist** → review them and add missing cases (see coverage checklist below).
   - If **no benchmark file** exists → create one with realistic inputs (1 000+ iterations, real-world-sized data).
   - If **benchmarks exist** → add cases for any new code paths.
   - When writing test assertions use `t.Error(testutils.DiffMessage(actual, expected, "desc"))` (import `"github.com/dangduoc08/ginject/testutils"`), not raw `t.Errorf`.

   **Coverage checklist** — for every public function/method, verify at least:
   - Happy path (valid input, expected output).
   - Empty / zero / nil input.
   - Boundary values (off-by-one, max length, empty slice vs nil slice).
   - Invalid / malformed input (error path, should not panic).
   - Security-relevant inputs: oversized payloads, special characters, path traversal (`../`), SQL metacharacters, null bytes.
   - Concurrent access (run with `-race`) if the code touches shared state.
   - Any invariant stated in step 3 must have at least one failing-input test that confirms rejection.

7. **Run tests**:
   ```
   go test ./... -race -v
   ```
   All tests must pass before reporting done. If a test fails, fix the implementation (not the test) unless the test itself is wrong.

8. **Run benchmarks**:
   ```
   go test -bench=. -benchmem ./...
   ```
   Record the numbers.

9. **Lint** — if `.golangci.yml` exists run `golangci-lint run ./...`; otherwise run with default linters. Fix in order:
   - Errors and bugs (`errcheck`, `staticcheck`, `govet`)
   - Security (`gosec`)
   - Style (`gocritic`, `errorlint`, `exhaustive`)

10. **Report** results in **two versions — English first, then Vietnamese** — each containing:
    - What was implemented or upgraded (one-line summary per item).
    - Security invariants verified.
    - Test cases added (count + categories covered).
    - Benchmark numbers (ns/op, allocs/op).
    - Lint issues fixed, if any.

    **English version** format:
    ```
    ## Implementation Report

    ### What changed
    | Item | Action (new / upgraded) | File |
    |------|------------------------|------|

    ### Security invariants
    | Invariant | Status |
    |-----------|--------|

    ### Tests added
    | Test name | Category |
    |-----------|----------|

    ### Benchmarks
    | Benchmark | ns/op | allocs/op |
    |-----------|-------|-----------|

    ### Lint fixes (if any)
    | Issue | Fix |
    ```

    **Vietnamese version** format (immediately after English, under a `---` divider):
    ```
    ---
    ## Báo Cáo Implement

    ### Những thay đổi
    | Mục | Hành động (mới / nâng cấp) | File |
    |-----|---------------------------|------|

    ### Bất biến bảo mật
    | Bất biến | Trạng thái |
    |----------|------------|

    ### Test đã thêm
    | Tên test | Loại |
    |----------|------|

    ### Benchmark
    | Benchmark | ns/op | allocs/op |
    |-----------|-------|-----------|

    ### Sửa lỗi lint (nếu có)
    | Vấn đề | Cách sửa |
    ```

## Rules

- Never break the public API (exported names, function signatures) without flagging first.
- Do not add features, error handling, or abstractions beyond the stated task.
- Do not add comments to code unless the WHY is non-obvious (hidden constraint, subtle invariant, workaround).
- Security and race-condition fixes are always applied, even when they reduce readability.
- Each package owns its output — do not leave malformed data for downstream callers to normalise.
- If the implementation requires a new dependency, state it explicitly before adding it.
- **Normalisation before capture**: assign output struct fields only after all input normalisation is complete — never capture a pre-normalised value.
- **Hot-path initialisation**: any work that computes static config (string joins, map builds, defaults) must run once at init/constructor time, never on every request.
