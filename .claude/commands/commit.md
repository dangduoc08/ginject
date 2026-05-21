Commit staged changes with an auto-generated message. If $ARGUMENTS is provided, use it as the commit message directly.

## Steps

1. Run `git diff --staged` to read all staged changes. If nothing is staged, tell the user and stop.

2. If $ARGUMENTS is provided, skip to step 4 using $ARGUMENTS as the message.

3. Analyze the diff and generate a commit message:
   - Follow the Conventional Commits format: `<type>(<scope>): <description>`
   - Types: `feat`, `fix`, `refactor`, `perf`, `test`, `chore`, `docs`
   - Scope: the package or module affected (e.g. `utils`, `routing`, `core`)
   - Description: imperative mood, lowercase, no period, max 72 chars
   - Add a body only if the change needs more context (non-obvious why)

4. Show the proposed commit message and ask for confirmation before committing.

5. On confirmation, run:
   ```
   git commit -m "<message>"
   ```

## Rules

- Never use `--no-verify`.
- Never amend an existing commit.
- Never stage additional files — only commit what is already staged.
- If the pre-commit hook fails, report the error and stop. Do not retry or bypass.
- Keep the message in English.
- Never add anything Claude-related to the commit: no `Co-Authored-By`, no `Generated with Claude`, no signatures, no trailers of any kind.
