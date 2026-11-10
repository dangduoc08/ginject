---
name: sync-docs
description: Find and fix drift between the docs (CLAUDE.md, README.md, package READMEs, .claude/port/ reading guides) and the source. Scope is $ARGUMENTS (a path or package) or, if empty, every doc touched by or describing code changed since the last commit that edited docs.
---

# Mission

Make the docs say only what the code actually does. The code is the source of truth, and docs are claims to verify.

## Steps

1. **Scope.** If $ARGUMENTS is given, use it. Otherwise run `git log -1 --format=%h -- '*.md'` to find the last docs commit, then use `git diff --name-only <that>..HEAD -- '*.go'` to list the code that changed since, and pick the docs that describe those packages.

2. **Extract claims.** From each doc in scope, list every checkable claim: identifiers, signatures, file paths, config fields, defaults, error messages, numbers (latency, sizes, limits) and code samples.

3. **Verify each claim against source.** Use `git grep`, `go doc <pkg>.<Name>`, and reading the definition. Verify a number by running the benchmark or test that measures it. Compile a code sample in the scratchpad against the local module (`replace` directive). For larger scopes, dispatch the `doc-verifier` subagent per doc in parallel.

4. **Fix.** Correct each wrong claim with a minimal edit. Delete claims that can't be verified instead of guessing. Never add a claim you didn't verify. Don't restructure docs that are correct.

5. **Check.** Run `make check`.

## Report

A table with columns: doc, claim, status (ok / fixed / removed), evidence (file:line or command). Then one line giving the totals.

## Rules

- Never change Go code to make a doc true. Report the mismatch instead.
- Keep the user's existing doc structure and language. A `README.vi.md` gets the same fixes as its English twin.
