Refactor the code in $ARGUMENTS for clarity and Go idiom compliance. If no argument is given, use the file currently open in the IDE or the most recently discussed file.

**Logic must not change.** Public API (exported names, function signatures) must not change. Only structure, naming, and style are in scope.

## Steps

1. **Read** the target file(s) fully before touching anything.

2. **Run existing tests** to record a passing baseline before any edits.
   ```
   go test ./... -race -count=1
   ```
   All tests must pass. If any fail before you start, stop and report — do not proceed.

3. **Check if the file should be split** before analyzing individual violations:
   - If the file is longer than ~400 lines, check whether it mixes distinct responsibilities (e.g. public API + REST handler + WS handler + shared helpers).
   - A natural split exists when: (a) each resulting file has a single clear focus, (b) no exported names need to move between files, and (c) all files stay in the same package.
   - If a split is warranted, perform it first (move functions only — no logic changes), run tests, then continue with the analysis below on each new file.
   - Typical split pattern for large handler files:
     - `foo.go` — struct, constructor, public API methods
     - `foo_rest.go` — HTTP request lifecycle handlers
     - `foo_ws.go` — WebSocket request lifecycle handlers
     - `foo_internal.go` — unexported helpers shared across the above

4. **Analyze** — identify every violation in these categories:

   **Naming**
   - Variable, parameter, and receiver names that are too long, too short, or don't follow Go convention (`i` → loop index is fine; `userInputDataFromRequestBodyJSON` → not fine).
   - Acronyms should be all-caps or all-lower, never mixed: `userID` not `userId`, `httpURL` not `httpUrl`, `wsHub` not `wsHb`.
   - Boolean names that don't read as a predicate: `isEnabled`, `hasItems`, `ok`.
   - Unexported names that shadow standard identifiers (`len`, `cap`, `error`, `string`, `new`).

   **Structure**
   - Functions longer than ~50 lines that have a clear split into named sub-steps — extract only when the extracted function has a single, obvious responsibility.
   - `if err != nil { return ..., err }` chains that can be collapsed without a helper.
   - Nested `if` blocks that can be flattened with early returns (guard clauses).
   - `else` after a `return` — remove the `else`.
   - Repeated literal values that should be a named constant (`const`).
   - Dead branches, tautological conditions (`if true`, `if x == x`).
   - Variable declarations separated from their first use by many lines — move them to the point of first use.

   **Go idioms**
   - `for i := 0; i < len(s); i++` → `for i, v := range s` where appropriate.
   - `new(T)` when `&T{}` is clearer.
   - Multiple return values used only for the error — check if a single `error` return suffices.
   - `fmt.Sprintf` used purely for concatenation — prefer `+` or `strings.Builder`.
   - `append` inside a loop when `make([]T, 0, n)` pre-allocation is known.
   - `interface{}` → `any` (Go 1.18+).
   - `switch` with a single `case` that could be an `if`.
   - Initialising a `map` with `make` then immediately populating — prefer a composite literal.
   - Struct fields that are always set together — consider grouping or a constructor.

5. **Apply** changes one file at a time, running tests after each file:
   ```
   go test ./... -race -count=1
   ```
   If any test fails after a change, revert that specific change and note it in the report.

6. **Run lint** after all changes:
   - If `.golangci.yml` exists: `golangci-lint run ./...`
   - Otherwise: `golangci-lint run --enable=govet,staticcheck,gocritic,misspell ./...`
   Fix only issues introduced or pre-existing in the files you touched.

7. **Report** results in **two versions — English first, then Vietnamese** — each containing a table of every change made.

   **English version** format:
   ```
   ## Refactor Report

   ### Changes
   | File | Line(s) | Category | Before | After | Reason |
   |------|---------|----------|--------|-------|--------|

   ### Lint fixes (if any)
   | Issue | Fix |

   ### Tests
   All N tests pass under -race.
   ```

   **Vietnamese version** format (immediately after English, under a `---` divider):
   ```
   ---
   ## Báo Cáo Refactor

   ### Thay đổi
   | File | Dòng | Loại | Trước | Sau | Lý do |
   |------|------|------|-------|-----|-------|

   ### Sửa lỗi lint (nếu có)
   | Vấn đề | Cách sửa |

   ### Tests
   Tất cả N tests pass với -race.
   ```

## Rules

- **Logic must not change.** If a structural change would alter observable behaviour in any code path, skip it and note it in the report.
- **Public API must not change.** Exported names, function signatures, interface implementations — do not touch.
- Do not add features, error handling, or new abstractions.
- Do not add or remove comments unless the comment describes *what* the code does (those should be removed); keep comments that explain *why* (hidden constraints, non-obvious invariants).
- Do not format with `gofmt` manually — the IDE/tool handles that. Focus on semantic clarity.
- If a rename would require changes in caller files outside the current target, flag it and ask before proceeding.
- Prefer the smallest change that achieves the clarity goal — do not refactor speculatively.
