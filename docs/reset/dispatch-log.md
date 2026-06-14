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
| 2026-06-14 | Builder | main | A-001-followup | Role length tightened to 80 chars (code in `internal/validation/validate.go` + migration `027_tighten_agent_role_length.sql` with pre-flight guard). 64c0d09 on main. | 64c0d09 |
| 2026-06-14 | Builder | main | A-002-09..18 | All 10 sub-items closed on `fix/A002-handbacks` (rebased 6 times through main while shipping 6 commits). Includes: A-002-09 dispatch unused uuid import, A-002-10 events RoundTrip, A-002-11 handler test fixes, A-002-12 integration build, A-002-13 middleware auth/api-key, A-002-14 router role matrix, A-002-15 service assignment+history, A-002-17 store-layer tests, A-002-18 config. Closed 65f7661, 00bc9a5, 3c4c4cb, ecd2e6f, bc4f4ae, 1972843, a6ff0c0, 13e59ac, bb48597. | (10 commits) |
| 2026-06-14 | Builder | main | A-003 | Assignment Engine shipped. 4 commits + 1 chore on `feat/A003-assignment-engine`: docs (d8a578b) 12-item drift + test (d0fe5b4) 23-case matrix + fix (0688bfe) notes cap 1 KiB + docs (47c1b55) audit + chore (c4852b1) store-layer tests (A-002-17). Last SHIP 4d53ed9 was A-001-followup. A-003 lands 4d53ed9. A-002 hand-backs were 3c4c4cb/ecd2e6f/00bc9a5 landed before A-003. | c4852b1, 4d53ed9 |
| 2026-06-14 | Ops | main | Go static-analysis | Step 13 of `scripts/validate-infra.py` — `gofmt -l`, `go vet`, future `staticcheck -checks=SA4009,SA4010` (shadow + unused-var). Catches A-002-01 class bugs at the gate. 0a0937c on main. | 0a0937c |
| 2026-06-14 | Builder | main | B-001 c1 | Execution Engine spec drift (12-item, mirrors A-001/A-002/A-003 pattern). f02425b. | f02425b |
| 2026-06-14 | Builder | main | B-001 c2 | 6-state lifecycle (queued/assigned/running/review/completed/failed). 9 files, +189/-58. Migration 028 drops 4-state CHECK, adds 6-state CHECK, rewrites 'pending' → 'assigned'. driveWorker/mock consolidated in c2 (state machine rejects running→completed, so runtime had to be updated for self-consistency). c1a65df on main. | c1a65df |
| 2026-06-14 | Leader | main | Pre-scope sweep | Committed 9 untracked pre-scope briefs (8 prior scratch + 1 new C-002/TASK-508). First Leader commit on main. Files: audit-prep-A-002, A-003, B-001, B-002, B-003, C-001, C-002 (NEW), D-003, fix-f-d002-001-webhook-ssrf. b1226f4 on main, pushed to origin. | b1226f4 |
| 2026-06-14 | Guardian | main | D-002 | Security Review shipped. 236-line review covering 4 preliminary + 18 Sprint 4 + 13 new findings. Sign-off APPROVED with conditions. Worktree `guardian-d002-wt` on `guardian/d-002-security-review` → merge origin/main (v2 rule) → push branch → ff-merge to main → push main. 22+ new commits landed during review; re-merged origin/main twice. aafad88 on main. | aafad88 |
| 2026-06-14 | Leader | Board | D-002 routing | F-D002-001 (HIGH webhook SSRF) → B-002 HARD GATE (pre-scope + brief already exist). F-D002-003 (INFO cookie secure) → Ops E-001 wave. F-D002-004 (HIGH X-Project-ID IDOR) → Sprint 6+ backlog. F-D002-005..018 (14 new LOW/INFO) → Sprint 7+ non-blocking. F-D002-014 (govulncheck in CI) → Ops gate pass. F-D002-007 (notes cap) closed at 0688bfe — verified. B-002 task description updated to call out F-D002-001 gate. | docs/reset/security-review.md §5.1, §5.2, §5.3 |
| 2026-06-14 | Leader | Guardian | D-003 | Dispatch: T1 happy-path E2E (Project→Task→Assignment→Execution→Deliverable→Done) + cross-tenant negative tests (F-013/14/15/16 replay). 5-commit shape: docs(api-spec) → feat(workflow) T1 scaffolding → feat(workflow) cross-tenant neg → test(workflow) table-driven → docs(audit). Pre-scope at docs/reset/audit-prep-D-003.md. | (in chat) |
| 2026-06-14 | Leader | Ops | F-D002-003 + F-D002-014 | Dispatch Ops-followup: (a) F-D002-003 secure cookie env var (v1 GA condition) — handler/auth.go:43 + test + docs/auth-design.md. (b) F-D002-014 govulncheck in CI (gate pass) — 4 lines YAML. Ops confirmed receipt, priority order (a)→(b), ETA ~45 min total. | (in chat) |
