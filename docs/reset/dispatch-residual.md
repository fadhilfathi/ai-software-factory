# Dispatch residual — captured 2026-06-14

Source: `dispatch.txt` (8 lines, 468 bytes) at the repository root.

## Original content

```
fatal: path 'docs/sprint5/lead-updates/2026-06-14-dev02-rebase-recovery.md' exists on disk, but not in 'origin/main'
```

## Interpretation

This is the captured stderr of a `git show origin/main:docs/sprint5/lead-updates/2026-06-14-dev02-rebase-recovery.md` invocation. The command was attempting to read a tree object from `origin/main` for a path that does not exist in that ref.

## Anomaly

As of 2026-06-14 cleanup:

- The referenced file `docs/sprint5/lead-updates/2026-06-14-dev02-rebase-recovery.md` is **not present on disk** (`docs/sprint5/lead-updates/` contains 26 other lead-update files but no `*dev02-rebase-recovery*`).
- The file is **not present in `origin/main`** either (verified via `git show origin/main:...`).

The "exists on disk" half of git's error message is therefore stale — the file may have existed transiently during a prior session and been removed. No action required beyond recording the anomaly for the Lead's reconciliation pass.

## Decision

- `dispatch.txt` deleted (it was a scratch error capture, not a real dispatch).
- This residual file is the only durable record of the capture.
