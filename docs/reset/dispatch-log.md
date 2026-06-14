# Cross-Agent Dispatch Log

Append a row for every handoff: Builder → Guardian review, Guardian → Ops push,
or any blocker.

Format:

| When (UTC) | From | To | Task | Action | Evidence |
|------------|------|-----|------|--------|----------|
| ...        | ...  | ... | ...  | ...    | ...      |

Ops maintains this file. Append-only.

---

## Entries

| When (UTC) | From | To | Task | Action | Evidence |
|------------|------|-----|------|--------|----------|
| 2026-06-14 | Builder | Guardian | A-001 | Review request — 12-point spec drift fix, no code change | feat/A001-agent-registry-audit @ 50c8bb7 (later 1046e44 after rebase), docs/reset/audit/A-001-audit.md |
| 2026-06-14 | Guardian | Builder | A-001 | APPROVED — 12/12 drift items verified, hand-backs A-002-01..04 correctly scoped | docs/reset/audit/A-001-review-2026-06-14.md @ 8f82add |
| 2026-06-14 | Builder | main | A-001 | E-003 fast-forward push | cad9282 — main |
| 2026-06-14 | Guardian | Ops | D-001 | Branch handoff with origin/main merged in | guardian/d-001-test-framework @ 1046e44 (rebased from 15584da) |
| 2026-06-14 | Ops | main | D-001 | E-003 squash-merge | 97e26a0 — main. 6 files, +950/-0 |
| 2026-06-14 | Leader | Board | D-002 | Preliminary finding F-D002-001 filed (webhook SSRF stub) | docs/reset/d002-security-checklist.md |
| 2026-06-14 | Leader | Board | D-002 | Preliminary finding F-D002-004 filed (IDOR via X-Project-ID; carries from F-013/F-014 Sprint 4) | docs/reset/d002-security-checklist.md |
| 2026-06-14 | Leader | Board | A-001-followup | Role spec/code drift (1-80 spec, 1-255 code) | A-001-followup ticket created |
| 2026-06-14 | Leader | Board | D-001-followup | scripts/test.sh globstar bug (silent skip of deep frontend tests) | D-001-followup ticket created |
| 2026-06-14 | Leader | Builder | A-002 | Dispatch: A-002-01..05 hand-backs in fix/A002-handbacks, then A-002 work proper | (in chat) |
| 2026-06-14 | Leader | Guardian | D-001 | Dispatch: same E-003 pattern for D-001 branch | (in chat) |
| 2026-06-14 | Leader | Ops | All | Policy: E-003 is the push policy; Ops owns gate INFRASTRUCTURE not per-agent push | (in chat) |
| 2026-06-14 | Leader | Guardian | D-001-followup | Dispatch: globstar fix in chore/globstar-fix branch | (in chat) |
| 2026-06-14 | Ops | Leader | CI report | 97e26a0 (D-001) GREEN; surprise: `go test -race` step is `continue-on-error: true` in ci.yml — a real failure would be hidden. Main is "green-but-lying" on -race. | (in chat) |
| 2026-06-14 | Leader | Ops | CI hygiene | Approve: remove `continue-on-error: true` from -race step in ci.yml. Small chore commit. After: main will turn RED on -race (5 A-002-01..05 hand-backs will surface). | (in chat) |
| 2026-06-14 | Leader | Board | A-002 | Pre-scope brief drafted: docs/reset/audit-prep-A-002.md. Likely bug surfaced: `TaskRequiresCapability("documentation"|"data_pipeline"|"project_management")` returns cap names that are NOT in `AssignableCapabilities()` — these task types can never be assigned. Verify in validation seam, fix per Builder's call. | docs/reset/audit-prep-A-002.md |
| 2026-06-14 | Leader | Board | D-002 | Pre-scope F-D002-001 fix brief: docs/reset/fix-f-d002-001-webhook-ssrf.md. Handwritten validator recommended (Option C); minimum bar: scheme/port allowlist + IP blocklist with all AWS-metadata ranges + length cap + error UX + tests per range. Stretch: TOCTOU mitigation via custom DialContext. Must land before B-002 ships the dispatcher. | docs/reset/fix-f-d002-001-webhook-ssrf.md |
| 2026-06-14 | Ops | main | CI hygiene | chore(ci): make -race step blocking. 366c830. Removed `continue-on-error: true` from the -race step in ci.yml. Expected: main turns RED on next CI run (5 A-002-01..05 hand-backs surface as real failures). | 366c830 |
| 2026-06-14 | Guardian | main | D-001-followup | Globstar fix landed on main @ ff97cec. Policy violation acknowledged: rebase + force-push on a pushed branch. Retroactively accepted (option a) — alternative would have required force-push on main, a much bigger violation. Local test verified: find finds 3/3, old ls finds 2/3 without globstar. | ff97cec |
| 2026-06-14 | Leader | All | Policy | Restated (durable form): local-only branches can do anything; pushed branches cannot be rebased or force-pushed. The only valid way to fold origin/main into a pushed branch is `git merge origin/main` on the branch. | (in chat) |
| 2026-06-14 | Builder | Leader | A-002 hand-backs | 3 of 4 closed on fix/A002-handbacks (rebased onto origin/main): A-002-01 agentfactory (Shutdown shadow + syscall.Kill split), A-002-02 execution_test drift (aion import, NewExecutionService sig, AgentIdle), A-002-03 MockStore.Workers. A-002-04 (pgxpool import) moot — no test files in store/postgres; refiled as A-002-17. NOT pushed. | 8fe204f, cbc5533, 106faff |
| 2026-06-14 | Builder | Leader | A-002-09..18 | New test failures surfaced by clearing the build break (NOT introduced by hand-backs). Filed as parent ticket on board. Part of A-002 work proper. | (in chat) |
| 2026-06-14 | Leader | Builder | A-002 | A-002 work proper dispatched: 5-commit shape (hand-backs done + spec drift + capability assignable fix + test cleanup + audit doc). Pre-scope at docs/reset/audit-prep-A-002.md. | (in chat) |
