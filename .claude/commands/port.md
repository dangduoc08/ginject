---
name: port
description: Port one step of .claude/port/PORT.md from master onto main. $ARGUMENTS is the step id, e.g. P1. Stops at every API divergence for the owner's decision, and never commits.
---

# Mission

Bring step $ARGUMENTS of `.claude/port/PORT.md` from `master` onto `main`, so that the owner can read, understand and accept every line. Ownership matters more than speed.

## Steps

1. **Load context.** Read `CLAUDE.md`, `.claude/port/PORT.md` and `.claude/decisions.md`. If $ARGUMENTS is empty, or its status isn't `todo`, stop and say so. If an earlier step is not committed yet, stop and say so.

2. **Survey.** For the step's paths, run `git diff --stat main master -- <paths>` and `git diff --numstat`. Read master's version of every file in scope in full. List:
   - the files master adds, changes or deletes;
   - what the step depends on that isn't ported yet (check master's imports);
   - the known bugs from PORT.md that fall in scope.

3. **Divergences. This step is mandatory.** Compare every user-facing API and every layer behaviour in scope between main and master: exported names, signatures, struct fields, defaults, error messages and status codes. For each divergence that isn't already decided in `decisions.md`:
   - build a table: main vs master, a short code sample of each, and benchmark numbers if it's on the request path;
   - ask the owner with AskUserQuestion, one divergence per question, with master as the default option (not marked recommended unless the evidence says so);
   - record the answer in `decisions.md`, with the date.
   
   Don't write any code before every divergence in scope is decided.

4. **Port.** Bring files with `git show master:<path>` and apply the decisions. Port the tests with the code. Keep existing comments in ported code, and add none. Re-prove each in-scope known bug with a throwaway probe test. Report it, and don't fix it unless the owner decided to.

5. **Evidence.**
   - `make check`
   - `go test -race` on the touched packages
   - a concurrent test for any shared state in scope
   - for request-path code, `go test -bench . -benchmem -count 5` on main before the port and after, compared side by side

6. **Review.** Run the `go-reviewer` subagent on the working-tree diff. Fix critical findings. List the rest.

7. **Reading guide.** Write `.claude/port/$ARGUMENTS-reading-guide.md` with:
   - the files in call order, each with one line on what it does;
   - per file, what to scrutinise: locks, goroutines, allocations per request, panics, reflection;
   - the decisions applied, the bugs re-proven, and the benchmark table.
   
   If the step is larger than about 300 lines of non-test library code, split the guide into ordered chunks of about 300 lines.

8. **Hand over.**
   - Update CLAUDE.md "Architecture" for what changed.
   - Set the step to `awaiting owner review` in PORT.md.
   - Report in the owner's language: what changed, why, and the commands to verify it.
   - Stop. Never run `git commit`. The owner commits through `/commit`.

## Rules

- Scope is exactly the step. Anything else you notice goes in the report as a suggestion.
- Never edit master. Never push, rebase, reset or clean.
- If a port needs new code in the red zone (`core/`, `common/`, `ctx/`, `routing/`) beyond applying decisions, stop and explain it before writing it.
