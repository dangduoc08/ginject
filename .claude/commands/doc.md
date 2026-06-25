Generate (or refresh) `README.md` for the package at $ARGUMENTS. If no argument is given, use the package of the file currently open in the IDE or the package most recently discussed.

## Steps

1. **Resolve the target package directory.**
   $ARGUMENTS may be a directory path, an import path, or a bare package name (e.g. `cache`, `modules/cache`, `github.com/dangduoc08/ginject/modules/cache`). Resolve it to exactly one directory under the repo. If it matches more than one directory, ask the user to disambiguate instead of guessing.

2. **Read the style reference.**
   Read `modules/config/README.md` in full before writing anything — every README produced by this command follows its structure and level of detail:
   - `# <Title>` heading, followed by an italicized one-line description.
   - A table of contents linking every section actually present (omit sections that don't apply to this package).
   - `## Key Features` — a short bullet list, only if the package has a handful of genuinely distinguishing features. Skip this section for trivial packages.
   - `## Usage` — one complete, realistic, compilable example showing the most common way to use the package (mirror the `main.go`-style example in `modules/config/README.md`).
   - A section per exported struct, documenting every exported field: its name, its Go type, its default value (if discernible), whether it's required, and a description (matching the `Type:` / `Default:` / `Required:` layout used for `ConfigModuleOptions`).
   - A section per exported function/method: description, parameters (position + type + description), return values (position + type + description), and a usage code block (matching the `Get` method section).
   - Two sections not present in `modules/config/README.md` but required when applicable: a `Rules` subsection under a function/method (step 5) and a package-level `## Benchmarks` section (step 6). Add them only when the package actually has the underlying test/benchmark coverage.

3. **Read the target package's exported surface.**
   Read every non-test `.go` file directly under the target package directory (do not recurse into subdirectories — a subdirectory is a different package). Enumerate:
   - Every exported type (struct, interface, type alias) and its exported fields, with each field's exact Go type.
   - Every exported function and method: receiver, name, parameter types, return types, and any doc comment.
   - Every exported constant and package-level variable.
   - Skip unexported identifiers entirely — they don't belong in the README.

4. **Cross-check real usage.**
   Search the package's own `_test.go` files and, if the package is used there, `sample/`, for how each type and method is actually constructed and called. Base every example on these real call patterns so the README never shows an API call that wouldn't compile against the current code.

5. **Derive behavior rules from tests.**
   For every exported function/method that has coverage in a `_test.go` file (excluding `_bench_test.go`), read its test cases and pull out the concrete behavioral contract: edge cases handled (empty/nil input, not-found), special return values (e.g. `-1` or `""` for "no match"), ordering/uniqueness guarantees, and conditions that trigger a panic or error. Only state a rule if an actual test asserts it — never infer or invent one. Keep these for use in step 9.

6. **Detect and capture benchmark results.**
   If the package directory contains a `_bench_test.go` file with `Benchmark*` functions, run `go test -run=^$ -bench=. -benchmem ./<package-import-path>` from the repo root and keep the raw output. If there's no benchmark file in the package, skip this — no `## Benchmarks` section is added.

7. **Infer intent from code, not from names.**
   If a field's or parameter's purpose isn't stated in a doc comment, read where it's consumed in the implementation before writing its description. Don't guess or invent behavior that isn't there.

8. **Audit an existing README before rewriting it.**
   If `README.md` already exists, check each section against the structure in step 2 (title + description, complete TOC, `Key Features` criteria, one compilable `Usage` example, `Type:`/`Default:`/`Required:` tags on every field, parameter/return/usage blocks per function) plus the `Rules` and `Benchmarks` sections from steps 5-6. Treat this as a refresh, not a rewrite: keep accurate existing prose untouched, fix only what's missing or now wrong relative to the guide or the current code, and add any section that's absent.

9. **Write `README.md`** in the target package directory, following the structure from step 2 plus the per-function `Rules` (step 5) and the package-level `## Benchmarks` section (step 6, only if benchmarks exist), populated with the findings from steps 3-7, and incorporating the audit from step 8 if the file already existed.

10. **Verify completeness.**
   Confirm every exported type, field, function, and method found in step 3 is documented in `README.md`. Confirm every field listing states its Go type explicitly. Confirm every test-derived rule from step 5 appears in the file, and that the `## Benchmarks` section (if applicable) is present and shows the captured numbers.

## Rules

- Never modify, create, or delete any `.go` file — output is strictly `README.md`.
- Every exported field's type must be shown explicitly, e.g. `` Type: `bool` ``, matching `modules/config/README.md`'s style.
- Every code example must use real exported identifiers from the package and must be valid, compilable Go.
- Do not document unexported identifiers.
- Do not add any Claude-related signature, attribution, or generated-by note to the file.
- If `README.md` already exists, this is a refresh, governed by step 8: keep accurate existing prose, correct anything now stale or non-compliant with the guide, and add missing sections — don't discard a well-written description just to rewrite it from scratch.
- If the package has no exported identifiers at all, stop and tell the user instead of producing an empty README.
- Per-function `Rules` bullets (step 5) must each trace back to a real assertion in a `_test.go` file — if a function has no test coverage, omit the `Rules` subsection for it rather than guessing.
- Benchmark numbers must come from an actual `go test -bench` run captured in step 6 — never fabricate or estimate them. Note above the results block that they're machine-dependent and were captured at doc-generation time.
