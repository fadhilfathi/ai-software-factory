# Dispatch — Ops — 2026-06-14

Slot: 019ec4fe-6055-7b80-a8be-b7c5fc51914b
Name: Ops
Model: MiniMax-M3 (aionrs)
Permission: YOLO

## Your slice
Owner of E-001 (infra validation), E-002 (github protection),
E-003 (continuous commit policy). The third is the standing rule
that governs every Builder → Guardian → Ops flow.

## First task: clean working tree + E-002 GitHub Protection

The working tree on main has Sprint 5 closeout noise that must
NOT be committed:

Worktrees (untracked dirs starting with `..`):
- `..devops01-wt/`
- `..devops01-wt-healthz/`
- `..devops01-wt-keyfunc/`
- `..devops01-wt-maxconns/`
- `..devops01-wt-pr8/`
- `..devops01-wt-routes/`
- `dev01-task-427-wt/`

Stale temp / dispatch files:
- `dispatch.txt`, `out.txt`, `out2.txt`, `out3.txt`, `out4.txt`,
  `out5.txt`, `push_output.txt`, `new_deliv_test.txt`,
  `new_tests.txt`, `ls.txt`
- `tmp.ps1`, `tmp2.ps1`, `tmp_copy.ps1`, `tmp_edit.ps1`,
  `tmp_inspect.ps1`, `tmp_commit_msg.txt`

Stale docs (decisions already executed; keep as archive or move
to a single bundle if too noisy):
- `docs/sprint5/lead-updates/` (40+ files dated 2026-06-14)

Actions:
1. Verify with `git status --short` that the untracked noise is
   exactly the items above. If something else appears, STOP and
   ping Leader before deleting.
2. The worktree dirs (`..devops01-*`, `dev01-task-427-wt/`) are
   likely git worktrees linked to branches. Check with
   `git worktree list` and `git worktree remove` each one cleanly
   (do not `rm -rf`). If a worktree has unmerged commits on its
   branch, decide with Leader whether to keep the branch (push
   it as a normal branch and remove the worktree) or drop it.
3. The `out*.txt`, `tmp*.ps1`, `tmp_commit_msg.txt`, `ls.txt`,
   `new_*_test.txt`, `push_output.txt` files are pure noise.
   Delete them.
4. `dispatch.txt` — read it once, summarize the residual action
   items to `docs/reset/dispatch-residual.md`, then delete the
   file. If it contains a secret, escalate immediately.
5. `docs/sprint5/lead-updates/` — leave in place for now. Do not
   commit any new files there. They are the audit trail.
6. Add an entry to `.gitignore` covering the noise patterns so
   they never reappear:
   ```
   # sprint closeout noise
   /out*.txt
   /push_output.txt
   /new_*.txt
   /ls.txt
   /tmp*.ps1
   /tmp_*.txt
   /dispatch.txt
   ```
7. `git status --short` must be clean (or only show new files
   you intentionally created for E-002 / E-001).
8. Commit: `chore(task-E002): clean sprint5 closeout noise + harden .gitignore`
9. Push to main: `git push origin main`. This is your one
   self-push for the reset.

## E-002 GitHub Protection — secret scan

Install or wire up a scanner. Options (pick one):
- `gitleaks` action in `.github/workflows/`
- `trufflehog` in CI
- A pre-commit hook using `gitleaks`

Recommended: gitleaks in CI, with `.gitleaks.toml` allowlist for
test fixtures. Wire it into
`.github/workflows/sprint-quality-gate.yml` so a failed scan
blocks merge to main.

Deliverable: `.github/workflows/secret-scan.yml` (or merged into
the quality gate) + `docs/reset/secret-scan-report.md` showing
the baseline scan is clean.

## E-001 Infrastructure Validation

1. Verify `docker-compose.yml` is valid: `docker compose config`.
2. If docker isn't available locally, parse YAML and run a dry
   config check. Note limitations in
   `docs/reset/infra-validation.md`.
3. For each service in compose: confirm healthcheck exists or
   note missing. The repo already has `scripts/healthcheck.sh` —
   use it.
4. Document run-book: `docs/reset/infra-runbook.md` — how to
   bring the stack up locally, how to seed the DB, how to run
   the migrations.

## E-003 Continuous Commit Policy (standing)

After every completed task:
1. Pull latest main
2. Review changes (`git diff origin/main..HEAD`)
3. Run tests
4. Verify build
5. Verify no secrets (gitleaks)
6. Commit
7. Push to main

Commit message format: `feat(task-A001): <imperative summary>`
(`fix`, `chore`, `docs`, `test` also OK).

## Anti-loop
- 2 failed CI runs on the same error → blocker file + Leader ping.
- Don't `rm -rf` worktree dirs. Use `git worktree remove` only.

Brief: `docs/reset/2026-06-14-brief.md`
