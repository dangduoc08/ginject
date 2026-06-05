Commit staged changes with an auto-generated message. If $ARGUMENTS is provided, use it as the commit message directly.

## Steps

1. Run `git diff --staged` to read all staged changes. If nothing is staged, tell the user and stop.

2. If $ARGUMENTS is provided, skip to step 5 using $ARGUMENTS as the message for a single commit covering all staged changes.

3. Group the staged changes into logical commit units:
   - Inspect each changed file and classify it by concern (e.g. feature logic, tests, docs, config, tooling).
   - Merge files that are tightly coupled (e.g. a feature file and its test file, or a refactor spread across closely related files) into one commit.
   - Split files that address unrelated concerns into separate commits.
   - If all staged changes belong to one coherent concern, a single commit is fine — do not split arbitrarily.

4. For each commit group, generate a message:
   - Follow the Conventional Commits format: `<type>(<scope>): <description>`
   - Types: `feat`, `fix`, `refactor`, `perf`, `test`, `chore`, `docs`
   - Scope: the package or module affected (e.g. `internal`, `routing`, `core`)
   - Description: imperative mood, lowercase, no period, max 72 chars
   - Add a body only if the change needs more context (non-obvious why)

5. Show the proposed commit plan (groups + messages) and ask for confirmation before committing. If there are multiple commits, show them in order.

6. On confirmation, for each group in order:
   a. Unstage everything: `git restore --staged .`
   b. Stage only the files in this group: `git add <file1> <file2> ...`
   c. Run: `git commit -m "<message>"`
   d. If the commit fails, stop and report the error.

## Rules

- Never use `--no-verify`.
- Never amend an existing commit.
- Never stage files that were not in the original staged set.
- If the pre-commit hook fails, report the error and stop. Do not retry or bypass.
- Keep the message in English.
- Never add anything Claude-related to the commit: no `Co-Authored-By`, no `Generated with Claude`, no signatures, no trailers of any kind.
