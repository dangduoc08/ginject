## What and why

<!-- One or two sentences. Link the PORT.md step or issue. -->

## Checklist

- [ ] `make check` passes locally (paste the tail of the output if this PR was made by an agent)
- [ ] No exported identifier, signature or observable behaviour was removed or changed. If one was, it's called out below and approved.
- [ ] No new comments in Go code
- [ ] Every main/master API divergence in scope is decided in `.claude/decisions.md`
- [ ] Race evidence for shared state, and before/after benchmarks for request-path code
- [ ] Tests use the package's existing assertion helper
- [ ] CLAUDE.md "Architecture" and the step's reading guide match the code
- [ ] Reviewed by `go-reviewer`, and read line by line by the owner

## API / behaviour changes

<!-- "None", or list each one -->
