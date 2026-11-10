# Roadmap

This is long-lived work. Unlike `.claude/port/`, which is deleted once the port is done, this file stays. Decisions live in `.claude/decisions.md`.

## Product work

| ID | Item | Notes | Status |
|---|---|---|---|
| R-1 | Devtool rebuilt on REST/SSE/WS with API-key auth | D-004. It inspects routes, the module tree, the DI graph, request logs and traces. The startup product sits on top of it. Design first. Was P10. | todo |
| R-2 | WebSocket stack | D-006. Ports `memorybroker`, `wsevent` and the WS parts of `core`/`ctx`, after HTTP is solid. Was P11. | later |
| R-3 | Adapter modules (Postgres, Redis, …) | D-005. Each is a separate Go module, so the core never imports a driver. Was P12. | later |

## Performance audit backlog

These are functions whose hot/warm/cold class couldn't be settled from the code alone (`/pair-review` step 1). Every row says why the class is unknown, how to find out, and when to look again.

**Until a row is resolved:**
- **Type A** optimisations are applied anyway. They are free: no readability cost and no extra branch.
- **Type B** optimisations wait for evidence. They have a cost: a fast path, `unsafe`, `sync.Pool`, manual buffers.

| Function | Why the class is unknown | How to find out | Revisit at | Pending type B idea |
|---|---|---|---|---|
| `crypto.UUID` on Linux | Cost measured only on macOS: 401 ns, 978 ns at 12 CPUs, dominated by `crypto/rand`. Go 1.24+ uses vDSO `getrandom` on Linux, which may be far cheaper | Run `go test ./internal/crypto -bench . -cpu 1,4,12` on Linux (CI runner) | P4 | None for `UUID` itself (it must stay crypto-grade). Hot callers (request ID) move to `math/rand/v2` in P3 |

Ways to find out:
- `go test -covermode=count` over N requests through `ServeHTTP`: a line executed N times is hot.
- A `pprof` CPU profile of the sample app under load (`wrk`/`vegeta`).
- Benchmarks with each config toggled (versioning, WS, access log) and with the largest input a client can send.

## Recurring audits

| Audit | Trigger | How |
|---|---|---|
| End-to-end request cost | After each change to the request pipeline, and after P4 | `ServeHTTP` benchmarks plus `pprof`, compared with the P4 baseline |
| Vulnerabilities | Monthly and on every dependency change | `make vuln`; the CI `vuln` job |
| Go upgrade | Each Go minor release | Read the release notes, bump `go.mod`/golangci-lint/govulncheck, run `go fix -diff ./...`, re-run benchmarks |
| Goroutine leaks | When concurrency code changes | `goroutineleak` profile (Go 1.27) in tests |
| Release readiness | Before each release | godoc on the exported API, runnable `Example`s, `/sync-docs` over every README, a `make vuln` pass, and a `CGO_ENABLED=0` build for linux/amd64, linux/arm64, darwin and linux/386 |

## When the port is finished

1. Move the durable parts of each `.claude/port/P<n>-reading-guide.md` into the package READMEs or CLAUDE.md:
   - the hot/cold maps;
   - why each optimisation exists;
   - the edge cases.
2. Check that every open item in PORT.md is either done or has been moved here.
3. Remove the master-comparison test scaffolding, which only makes sense while master is the reference.
   - In `internal/ds/trie_test.go`, remove `findMaster`, `TestTrieFind_MatchesMasterWhenMasterFindsAnything` and `TestTrieFind_BacktrackKeepsParamsWithTheirRoute`'s master comparison. Rewrite that last test with fixed expected values.
   - `TestTrieFind_BacktrackingOnlyTurnsMasterMissesIntoMatches` also calls `findMaster`. Rewrite it so that whenever `Find` returns no wildcard, its result must equal `bruteForceMatch`.
   - Keep `bruteForceMatch`. It doesn't depend on master and still checks `Find`.
   - Look for other `*Master` reference copies added by later port steps and treat them the same way.
4. Delete `.claude/port/`, and remove `/port` from `.claude/commands/` and CLAUDE.md.
