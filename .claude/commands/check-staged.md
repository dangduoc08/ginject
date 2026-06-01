Inspect the staged changes for correctness, logic integrity, and impact relative to the last commit. Stash unstaged work first so the working tree is clean and the diff is unambiguous.

## Steps

1. **Abort if nothing is staged.**
   Run `git diff --staged --stat`. If the output is empty, tell the user and stop.

2. **Stash unstaged changes.**
   Run `git stash --keep-index --include-untracked -m "check-staged: temp stash"`.
   If it fails (nothing to stash), continue — the tree is already clean.
   Record whether a stash was created so you can restore it at the end.

3. **Collect the staged diff.**
   Run `git diff --staged` to get the full diff.
   Run `git diff --staged --stat` for a summary.

4. **Build and test.**
   Run `go build ./...` and `go test ./...`.
   Record any failures — they are findings.

5. **Analyse the diff for impact.** For each changed file, examine:
   - **Logic changes** — Does the observable behavior change? Are any conditions, data flows, or call sequences different from the last commit?
   - **API / public surface** — Are any exported types, function signatures, or package-level variables added, removed, or altered?
   - **Concurrency** — Any new shared state? Any lock added, removed, or reordered?
   - **Error handling** — Any error paths added, removed, or silenced?
   - **Performance** — Any hot-path allocations or work moved in or out of a loop?
   - **Security** — Any injection surface, credential exposure, or TLS weakening?

6. **Restore stashed changes.**
   If a stash was created in step 2, run `git stash pop`.
   If `pop` fails, report the conflict to the user and stop — do NOT lose their work.

7. **Write the report** — English first, then Vietnamese under a `---` divider. Format:

```
## Staged Change Impact Report

**Build:** ✅ pass / ❌ fail  
**Tests:** ✅ pass / ❌ fail (list failing tests if any)

### Files changed
| File | +lines | -lines | Category |
|------|--------|--------|----------|
| ...  | ...    | ...    | ...      |

### Findings
| # | File | Finding | Severity |
|---|------|---------|----------|
| 1 | ...  | ...     | info / warning / critical |

### Conclusion
**Behaviour change vs last commit:** YES / NO  
<one or two sentences — what changed functionally, or confirm nothing changed>
```

```
---
## Báo Cáo Tác Động Thay Đổi Staged

**Build:** ✅ pass / ❌ fail  
**Tests:** ✅ pass / ❌ fail (liệt kê test thất bại nếu có)

### Files thay đổi
| File | +dòng | -dòng | Phân loại |
|------|-------|-------|-----------|
| ...  | ...   | ...   | ...       |

### Phát hiện
| # | File | Nội dung | Mức độ |
|---|------|----------|--------|
| 1 | ...  | ...      | info / warning / critical |

### Kết luận
**Thay đổi hành vi so với commit cuối:** CÓ / KHÔNG  
<một hoặc hai câu — mô tả thay đổi về mặt chức năng, hoặc xác nhận không có gì thay đổi>
```

## Rules

- Never commit, push, or modify staged files.
- If `git stash pop` fails, stop immediately and warn the user — restoring their work takes priority over completing the report.
- Severity guide:
  - **critical** — behaviour change, broken API, race condition, build/test failure
  - **warning** — encapsulation leak, missing error handling, performance regression
  - **info** — cosmetic, naming, dead code, style
- Keep findings concise — one line per finding.
- The Conclusion must be explicit: state YES or NO, not "possibly" or "depends".
