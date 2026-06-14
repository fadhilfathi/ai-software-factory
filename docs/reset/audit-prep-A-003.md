# A-003 Assignment Engine — Pre-Scope Brief

**Owner of A-003:** Builder. Support: Leader (this brief).
**Spec:** `docs/sprint4/agent-orchestration-design.md` §Assignment + `docs/sprint5/agent-creation-management-design.md` (the F-014 + F-016 + TASK-421 line of work)
**Audit precedent:** `docs/reset/audit/A-001-audit.md`, `docs/reset/audit-prep-A-002.md`

## Deliverables (per the team contract)
- Task Assignment
- Ownership Tracking
- Assignment History

## Existing code floor (READ FIRST)

| File | Purpose | Status |
|------|---------|--------|
| `src/internal/handler/assignment.go` | `AssignTaskToAgent`, `ListTaskAssignments`, `ListAssignmentHistory` | Reads + 1 write. Uses the `projectIDFromContext` pattern (F-D002-004 IDOR surface). |
| `src/internal/service/assignment.go` | `AssignTaskToAgent` (F-014 fix), `ListTaskAssignments`, `ListAssignmentHistory` | Solid. Transactional write of (a) flip previous active to superseded, (b) create new active, (c) append event. All in one tx. |
| `src/internal/model/assignment.go` | `Assignment` row, `AssignmentEvent` row, `AssignmentStatus` enum, `AssignmentAction` enum, `AssignmentFilter` for list | Solid. `Notes omitempty` was removed in A-002-11 to fix a test assertion (good — empty string now serializes as `"notes":""`). |
| `src/internal/store/postgres/assignment.go` + `assignment_event.go` | PG-backed store | Read. Schema covered by migrations 019 + 020. |
| `migrations/019_*.sql`, `020_*.sql` | Schema | Read to confirm CHECK constraints. |
| `internal/integration/integration_test.go` | The 4-step T1 smoke + the 11 sub-cases T2 malformed-UUID | Will be extended to 15 steps in D-003 (which depends on A-003 service tests being solid). |

## Likely gaps to verify

### 1. Spec drift in `docs/api-spec.md` §Assignments
Same shape as A-001 (12 items) and A-002 (the pre-scope estimated ~12 items). Likely drift:
- `assignment_id` vs `id` (path vs body field naming)
- `caller_project_id` requirement in the request
- `notes` (text) required vs optional (model has it as required now; spec may have it optional)
- Status enum drift (`active`, `superseded`, `completed` — 3 in code, may be more in spec)
- Action enum drift (`assign`, `reassign` — 2 in code, may include `unassign` in spec)
- Pagination: cursor vs offset
- Sort order: assigned_at DESC vs created_at DESC

Cross-ref the handler responses (which use the `{data: ...}` envelope from A-002-11) with the spec.

### 2. Test coverage
- Service layer: `service/assignment_test.go` — coverage of `AssignTaskToAgent` happy/edge, idempotency, cross-tenant, reassign, unassign (if supported), F-014 crossTenantBlocked path.
- Model: `model/assignment_test.go` — `AssignmentStatus` enum validation, `AssignmentAction` enum validation, `AssignmentEvent` Notes serialization.
- Handler: `handler/assignment_test.go` — `AssignTaskToAgent` (happy, 400 missing header, 404 crossTenantBlocked, 404 task not found, 404 agent not found, 409 already active), `ListTaskAssignments` (happy, 400 missing header, empty result), `ListAssignmentHistory` (happy, with filter).
- Integration: extend `integration_test.go` per D-003 (15-step T1).

The A-002-15 fix is in flight for the 12 service test cases (`TestAssignTaskToAgent_*` / `TestListAssignmentHistory_*`). A-003 should verify the A-002-15 fix actually addresses all the gaps, then add what's missing.

### 3. The TASK-421 fix (cross-tenant 4th create)
A-002-11 already addressed this in the deliverable test redesign (3 deliverables, no cross-tenant 4th create). Verify the analogous logic in the assignment handler:
- Can a user from project A POST `/v1/assignments` with `X-Project-ID: B` and create an assignment in project B?
- The F-014 fix is supposed to block this (callerProjectID != task.ProjectID → crossTenantBlocked).
- The test should exercise this.

### 4. Idempotency
- Service returns `Idempotent: true` when the requested assignment already exists. Verify the test covers both first-time-assign and re-assign-to-same-agent paths.
- Verify the response shape for idempotent re-assign vs new assign (same status code? Same body?).

### 5. F-D002-004 IDOR
- Same X-Project-ID surface. Log in the audit doc as D-002 OPEN (Sprint 6+ work).
- The cross-tenant negative test in D-003 should exercise this.

## Audit doc shape (mirror A-001 / A-002)
`docs/reset/audit/A-003-audit.md`:
- Evidence: every existing file, the F-014 fix, the transactional write, the assignment_events append
- Drift inventory: 12 items, each with `code` `spec` `fix`
- Pre-push gate: tests + build + Guardian sign-off + secret-scan
- Hand-backs: anything that crosses into A-002 (done), B-001 (the 6-state lifecycle will affect assignment state), or C-002 (recovery will affect assignment retries)

## Suggested PR shape
- Commit 1: `docs(api-spec): fix assignment drift (N items)` (docs only — mirror A-001)
- Commit 2: `test(assignment): table-driven coverage for AssignTaskToAgent + ListTaskAssignments + ListAssignmentHistory` (extends A-002-15 fixes to the full service surface)
- Commit 3: `fix(assignment): [any code gap surfaced by the audit]` (e.g., unassign endpoint, notes validation, status enum drift)
- Commit 4: `docs(audit): A-003 assignment engine audit + pre-push gate`

If the spec drift is non-trivial, this PR can be 2 PRs. Builder's call.

## When this must land

After A-002 (Capability System) ships. The A-002-15 service test fixes (in flight) are upstream of A-003 — A-003 will extend them. The D-003 integration test (15-step T1) depends on A-003 shipping the assignment tests.

## What I (Leader) will do

- Review the audit doc.
- Surface the F-D002-004 IDOR in the D-002 report (it carries from F-013/F-014 to all 11 deliverables).
- Coordinate with Guardian on the D-003 cross-tenant negative test (Guardian owns D-003).

## What Guardian (D-002 / D-003) will do

- Review the F-014 cross-tenant check.
- Add the cross-tenant negative test to the D-003 deliverable.
- Verify the assignment history endpoint doesn't leak events across projects.
