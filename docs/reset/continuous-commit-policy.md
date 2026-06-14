# Continuous commit policy — E-003

**Owner:** Ops
**Status:** STANDING (applies to every Builder commit, every sprint, no exceptions)
**Effective:** 2026-06-14, start of Sprint 6
**Replaces:** the implicit "commit at sprint end" pattern that broke Sprint 5

## Policy

After **every** task — Builder or otherwise — the following pipeline runs before code lands on `main`:

```
pull main  →  review  →  test  →  build  →  secret-scan  →  commit  →  push main
```

**No sprint-end batching. No "I'll push it later." No "it's just one more change."**

Every commit that lands on `main` is one Builder task's worth of work, scanned and reviewed at the moment of delivery, not at the end of the sprint.

## Why

Sprint 5 ended with a backlog of uncommitted work, dead worktrees, and a number of un-pushed changes. Root cause: the team was batching commits at the end of the sprint, which made recovery work costly and led to the "let me rebase / re-derive / re-push" loop that took days to resolve.

This policy eliminates the failure mode by making the commit a non-event: small, frequent, and pre-validated.

## How it works in practice

### Builder's job

1. Create a feature branch (e.g. `feat/sprint-N-task-NNN-short-name`).
2. Implement the task.
3. Run their own pre-commit checks: `npm run build`, `go test ./...`, etc.
4. Open a single commit on the feature branch. (Squash WIP locally; the commit on main will be a single change.)
5. Hand the branch name to Ops. The hand-off message goes in the team channel.

### Ops' job (this script)

For every Builder hand-off, Ops runs:

```bash
python scripts/ops-commit.py feat/sprint-N-task-NNN-short-name
```

The script does, in order:

1. `git fetch origin main && git checkout main && git pull --rebase origin main` — sync to latest.
2. `git rev-parse --verify <branch>` and `git merge-base --is-ancestor main <branch>` — confirm the branch is ahead of main, not diverged.
3. `gitleaks detect --source . --log-opts main..<branch>` — secret-scan the diff.
4. `python scripts/validate-infra.py` — static infra checks.
5. `git diff --stat main..<branch>` and `git log main..<branch>` — surface the change for Ops to review.
6. `git checkout <branch> && git rebase main` — linear history.
7. `git checkout main && git merge --ff-only <branch>` — fast-forward merge.
8. `git push origin main` — push.
9. Print a one-line summary for the Lead.

**The script never force-pushes, never rewrites published history, and never pushes if any step fails.** Non-zero exit at any step halts the pipeline.

### Flags

- `--dry-run` — run all checks but skip the rebase/merge/push. Use for rehearsal.
- `--skip-tests` — skip gitleaks and validate-infra. **Only** use when CI has already run them in a previous step (e.g. the branch was tested in a PR and the merge is a no-op re-apply). The default is to run them every time.

### Failure modes

| Failure | Recovery |
|---|---|
| Branch not ahead of main | Builder rebases onto latest main, re-hands off. |
| Gitleaks detects a secret | **Block.** Builder rotates the secret, rewrites the commit (e.g. `git commit --amend`), re-hands off. |
| Static check fails | Builder fixes the issue, re-hands off. |
| Rebase conflicts | Ops asks Builder to resolve; the script leaves the working tree mid-rebase. |
| Push fails (non-ff, auth) | Ops investigates manually; the local main is now ahead of origin and visible in `git status`. |

### What this script does NOT do

- It does **not** squash a multi-commit feature branch into a single commit. The Builder is expected to do that locally before handing off. (We keep a linear history; one feature = one commit on main.)
- It does **not** create a GitHub PR. We commit directly to main; PRs are reserved for Guardian sign-off on high-risk work, not for routine delivery.
- It does **not** run `go test ./...` or `npm test`. Those are the Builder's responsibility before hand-off. The CI workflow `ci.yml` is the contract.

## Standing rules for Ops

- The Ops commit-pipeline runs **at least once per Builder hand-off**, no exceptions.
- If a Builder hand-off sits in the queue for more than 30 minutes, Ops notifies the Lead.
- Ops never amends or rewrites a Builder's commit. If the commit needs changes, the Builder does it; Ops only rebases.
- If the build/test/secret-scan fails, Ops **blocks** the commit, regardless of how small the change is.

## Sprint 6+ review

This policy is reviewed at the end of every sprint. If the team is consistently merging 1-2 commits per day, the policy is working. If the team is back to batching, the Lead escalates.

## See also

- `scripts/ops-commit.py` — the policy, executable.
- `scripts/validate-infra.py` — static checks used at step 4.
- `tools/install-gitleaks.sh` — installs the scanner used at step 3.
- `.github/workflows/secret-scan.yml` — CI gate (parallel protection, not a substitute).
- `docs/reset/infra-runbook.md` — operational context.
- `docs/reset/secret-scan-report.md` — scanner configuration and baseline.

## History

- 2026-06-14: policy created by Ops as part of E-003, end of Sprint 5 cleanup.
