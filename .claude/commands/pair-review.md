---
name: pair-review
description: Pair-review one ginject package with the owner across six lenses (performance via a hot/warm/cold map, concurrency, security, correctness and error handling, resource management, readability and conventions), plus API surface for exported packages. Every finding is backed by a probe, the owner decides each fix, then tests, benchmarks and docs are closed out. $ARGUMENTS is the package path, e.g. internal/str.
---

# Mission

Review the package at $ARGUMENTS together with the owner. The result is code the owner understands line by line: correct, race-free, safe against hostile input, with no leaks, and with speed spent only where requests actually pay for it. Talk to the owner in Vietnamese. Keep code, identifiers and docs in English.

Read `CLAUDE.md` ("Performance depends on the path", "Where functions live in a package") and `.claude/decisions.md` first.

## Steps

### 1. Map the callers (evidence, no guessing)

For **every** function in the package, exported or not:
- Find each caller with `git grep`. Search the current branch, and also `master` for packages that are still being ported.
- Classify the function by tracing up the call chain:
  - **hot**: runs on every request. Trace it from `ServeHTTP` or the WS read loop, and state **how many times per request** it runs (for example, once per path segment).
  - **warm**: runs only for some requests, such as JSONP, static files, errors or a specific middleware.
  - **cold**: runs at `Create()`, at startup, in tests, or on a failure path.
  - **unused**: no caller. Propose removal (D-008).
  - **undetermined**: the chain can't be traced statically or depends on outside factors, such as reflection, interfaces, function values in maps or closures, config, or the user's own app. Say why. Never guess.
- **Default rules:**
  - An API a user can call from a handler (`ctx.*`, module services) counts as **hot** or warm. The framework can't know how often user code calls it.
  - An error path a client can trigger (404, 405, validation failure) counts as **warm, client-triggered**. Bots can make it hot.
  - A config-gated feature (versioning, WS, access log, devtool) must cost almost nothing when off. Measure it both on and off.
- **From P3 onward**, once `ServeHTTP` runs on `main`, cross-check the static map against `go test -covermode=count` over N requests. A line executed N times is hot.
- Present the result as a table. Columns: function, class, caller chain (`file:line` → … → entry point), calls per request, current ns/op and allocs.

### 2. The owner confirms the map

Stop and ask the owner to confirm or correct the classification. Use AskUserQuestion in Vietnamese. Don't optimise anything until the map is agreed.

### 3. Hot path: measure, then propose

For each hot function:
- Benchmark it with realistic input taken from the real call site, for example a 5-segment API path. Use `for b.Loop()` and `b.ReportAllocs()`.
- Put it in context: its cost per request (ns × calls) and its share of a whole request, using the latest end-to-end `ServeHTTP` benchmark if one exists.
- Propose at most 1–2 optimisations. For each one, give:
  - what changes;
  - the expected gain, measured with a prototype benchmark, not estimated;
  - what it costs in readability;
  - any constraint it brings.
  
  Heavy techniques (`unsafe`, `sync.Pool`, manual buffers, restructuring code to make it inline) need the owner's explicit approval for each use.
- Ask the owner. Each question has one option per proposal, plus "keep as is", and shows before/after code in the preview.
- Apply what the owner approves. Re-run the benchmarks and show before and after side by side. Correctness and `-race` must still pass.

### 3b. Undetermined: type A now, type B later

- **Type A** (free: no readability cost, no extra branch, no heavy technique) is applied anywhere, whatever the class. Examples: no regexp compiled per call, no needless alloc or copy, no `Sprintf` for plain concatenation, no repeated work inside loops.
- **Type B** (it costs something: a fast path, `unsafe`, `sync.Pool`, manual buffers) waits for evidence. Add a row to `.claude/roadmap/ROADMAP.md` under "Performance audit backlog" with the reason, how to find out, when to revisit, and the type B idea.
- For input a client controls (path length, segment count, header and body size), **bound the work first** with limits, and optimise second.

### 3c. Concurrency (mandatory)

- **Inventory** all shared state: package-level vars, fields reached from more than one goroutine, maps, caches, pools and channels. For each item, say who reads it, who writes it, when, and what protects it.
- **Locks:** check lock ordering for deadlock, work done while holding a lock (especially I/O or user callbacks), and the scope of each lock. A global lock on a hot path is a finding.
- **Goroutines:** every goroutine needs a stop path, through context, a closed channel or shutdown. Look for channel sends that can block forever. Check with the `goroutineleak` profile (Go 1.27) where goroutines exist.
- **Evidence:** a concurrent test under `-race` for each piece of shared state, plus a probe for each suspected race.
- If the package has no shared state, say so with the grep that proves it, for example no package-level mutable `var`, `go func`, or `sync.*`.

### 3d. Security (mandatory)

- **Inventory** every input a client controls that reaches the package: path, query, headers, body, files, WS frames.
- For each input, check:
  - **bounds and DoS:** size limits, and algorithmic worst cases such as recursion depth, backtracking and quadratic loops;
  - **silent corruption:** values changed without an error, such as overflow, truncation, or bad syntax becoming a zero value;
  - **injection:** XSS, header and CRLF injection, path traversal, log injection;
  - **panics** on odd input;
  - **comparisons:** secrets compared in constant time (`crypto/subtle`);
  - **randomness:** `crypto/rand` wherever unpredictability matters.
- **Evidence:** a probe with hostile input (oversized, deeply nested, malformed) and its measured time and result.

### 3e. Correctness, errors and resources (mandatory)

- **Errors:**
  - no ignored errors (`_ = err`, or `v, _ :=` on parse);
  - errors are wrapped with `%w` so `errors.Is`/`errors.As` work;
  - request-path panics are recovered with the correct status;
  - bootstrap panics follow `"<category>: <detail>"`.
- **Resources:**
  - no map, cache or slice that only grows;
  - files, bodies and connections are closed;
  - `context` cancellation and timeouts are honoured;
  - shutdown releases everything.
- **API surface** (exported packages only; skip for `internal/`):
  - nothing is exported without need;
  - naming is consistent;
  - zero values are usable;
  - every exported name is a compatibility promise once released.

### 3f. Readability and conventions (mandatory)

The repo has no comments, so **names are the only documentation**. Check:
- **Names say what things are.** Avoid single letters outside tiny scopes and loops (`t reflect.Kind` → `kind`). Avoid names that shadow builtins (`close`, `len`, `new`, `cap`) or the package's own name (`str` inside package `str` → `s`). Names should be consistent across the package.
- **Typos** in identifiers, files and messages. The `misspell` linter catches common ones in strings and comments. Scan identifiers by eye too, because linters miss typos like `dimmensions`.
- **Function placement** follows CLAUDE.md ("Where functions live in a package"): a single-use helper sits after its caller, and shared helpers go in `fn.go`.
- **File and package names** say what they contain. Respect recorded owner decisions (D-021) and don't re-propose them.
- **Structure:** each function does one thing. Flag deep nesting and long functions only when they really hurt reading.
- **The owner's existing style** wins over general Go taste. Propose; don't impose.

### 4. Warm and cold path: readability

- Cold: review for clarity only. Propose a simplification when the code is harder than it needs to be. Type A fixes are fine here, but never apply type B to cold code.
- Warm: treat it as cold, unless it can be triggered per request by a client (for example an attacker-controlled path). In that case treat it as hot.

### 5. Close out

- **Tests:** cover every branch and pin down edge cases. Use `test.DiffMessage` and Go 1.27 idioms.
- **Benchmarks:** one per hot or warm function, with realistic input.
- **Caller bugs:** for problems that live in a caller rather than in this package, record each one in `.claude/port/PORT.md` under the port step that owns the caller. If no port step owns it, record it in `.claude/roadmap/ROADMAP.md`.
- **Decisions:** add each behaviour change to `.claude/decisions.md`.
- **Reading guide** (English and `.vi.md`): start the package's section with the hot/cold table. Then give one line per lens (performance, concurrency, security, correctness/resources, readability/conventions, API), with ✅ and the evidence command for a clean lens, or the finding and its fix. Then explain every optimisation that was applied.
- Run `make check` and report the real result.
- Ask the owner whether the package is done. Move on only after they confirm it is.

## Reporting

Report only the lenses with findings in detail. A clean lens gets one line, "✅ no issue", plus the command or probe that proves it, so reviews stay short.

## Rules

- Never classify a function as hot or cold without showing the call chain.
- Never optimise before the owner has confirmed the map.
- One package per run.
