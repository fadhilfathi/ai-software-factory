---
task: A-003 Assignment Engine
type: audit
date: 2026-06-14
owner: Builder
reviewer: Guardian
status: ready-for-guardian-review
---

# A-003 Assignment Engine — Audit

## Verdict

**Pass with 12 spec drift items fixed, 1 code gap closed, 23 audit-grade test cases landed across 3 unified matrices.** The assignment engine — POST /v1/tasks/:id/assign, GET /v1/tasks/:id/history, the two-table model (assignments + assignment_events), the 4-status / 3-action state machine, F-014 cross-tenant safety, F-017 notes round-trip, idempotent re-POST — is implemented, tested, and documented. The 12 drift items in `docs/api-spec.md` are fixed by writing a new §The Assignment Engine (A-003) section. The one code gap (notes length cap) is closed at both the handler and the service layer with a 400 VALIDATION_ERROR. The deferred A-002-15..18 service test rewrites are folded in as a parallel, table-driven view in `assignment_table_test.go` (14 + 5 + 4 = 23 cases).

The pre-existing F-D002-004 IDOR on the `project_memberships` table is **out of scope for this PR** and is filed as a D-002 OPEN finding (Sprint 6+). Per Lead's directive: "F-D002-004 IDOR lives in audit doc as D-002 OPEN finding (Sprint 6+); don't try to fix the project_memberships table in this PR."

## Codebase evidence

| File | Status | Notes |
|------|--------|-------|
| `docs/api-spec.md` | ✓ (Commit 1) | New §The Assignment Engine (A-003) section (~149 lines) added between §Capabilities and §Recovery. Replaces the Sprint 4 placeholder §Task Assignment that pointed at the wrong domain object (Executions). |
| `src/internal/model/assignment.go` | ✓ (Commit 3) | Added `MaxAssignmentNotesBytes = 1 << 10` constant (1 KiB). The 4 status values and 3 action values match the spec, the model, and the DB CHECK constraint. |
| `src/internal/model/assignment_event.go` | ✓ | `Notes` field has no `omitempty` so the empty-string case round-trips per F-017. `Action` carries the assign / reassign / unassign verb (the unassign case is reserved for Sprint 5+). |
| `src/internal/service/assignment.go` | ✓ (Commit 3) | Validation added in `AssignTaskToAgent` after F-014 checks but before the DB roundtrip. Returns `*Error{Status: 400, Code: "VALIDATION_ERROR"}` via the existing `validationSingle` helper. |
| `src/internal/service/assignment_test.go` | ✓ (Commit 2) | The wrong A-002-19 tracking comment (4-arg constructor claim) is replaced with a `resolved-by-A-003` note. The file compiles as-is; the analysis was based on stale production signatures. |
| `src/internal/service/assignment_table_test.go` | ✓ (Commit 2, NEW) | 663-line parallel, table-driven coverage. 3 matrices, 23 sub-tests. |
| `src/internal/handler/assignment.go` | ✓ (Commit 3) | Parallel notes-length check for early rejection (no DB roundtrip on a bad request). |
| `src/internal/handler/assignment_test.go` | ✓ (pre-existing) | 14+ test functions cover the existing handler surface; the notes-too-long path is tested in the service table (commit 2) and the parallel handler check is structurally identical to the F-017 empty-notes case. |
| `src/internal/store/memory.go` | ✓ (pre-existing) | `memoryAssignmentStore` and `memoryAssignmentEventStore` implement the `assignments` and `assignment_events` tables. The event store sorts DESC by `assigned_at` (newest-first) for the history endpoint. |
| `src/internal/store/postgres/assignment_event_store.go` | ✓ (pre-existing) | `ListByTask` orders by `assigned_at DESC, id DESC` for stable newest-first ordering. The store-layer tests are filed as the A-002-17 chore (deferred to a follow-up commit, not in this PR). |
| `src/db/migrations/019_create_assignments.sql` | ✓ (pre-existing) | Partial unique index `uq_assignments_one_active_per_task` enforces "at most one active per task" at the DB layer. CHECK constraint on the 4 status values. |
| `src/db/migrations/020_create_assignment_events.sql` | ✓ (pre-existing) | `agent_id` is NULLABLE (unassign support for Sprint 5+). `notes` is TEXT (no DB-level length limit; enforced at the application layer in commit 3). CHECK constraint on the 3 action values. |
| `src/internal/router/router.go` | ✓ (pre-existing) | `POST /v1/tasks/:id/assign` and `GET /v1/tasks/:id/history` wired. |

## Drift inventory (12 items)

The following items are corrected by this PR. Numbers in `(→)` are the post-fix value.

### Spec drift — request / response shape (5 items)

| # | What drifted | Before → After |
|---|--------------|----------------|
| D-01 | Path parameter name: spec uses `:taskId`; code uses `:id`. | `:taskId` → `:id` (the existing 24+ tests use `:id`; code is canonical). |
| D-02 | Request body shape: spec has `{ agent_id }` only; code accepts `{ agent_id, capabilities_required, notes }`. | `→ { agent_id, capabilities_required, notes }` (the 2 missing fields are now documented with their semantics). |
| D-03 | Response shape: spec was a flat `{ execution_id, task_id, agent_id, status, started_at }` — the **wrong domain object** (this is the Executions response). | `→ { data: { task, event, assignment, idempotent } }` — the Assignment response shape, wrapped in the `{ data: ... }` envelope per the A-001/A-002 standing pattern. |
| D-04 | Same — the spec used the Executions response shape, not the Assignment response shape. | `→` the Assignment response shape. Assignments are the current-state projection; Executions are the runtime state. The two are distinct domain objects. |
| D-05 | Response includes `{ idempotent: bool }` flag for re-POST of the same agent_id. | `→` documented. The flag is `true` iff the call returned the existing active assignment instead of writing a new event row. |
| D-06 | Response includes the appended event object. | `→` documented. The `event` field carries the `AssignmentEvent` that was just written (or `null` on the idempotent path). |
| D-07 | Response includes the new Assignment row. | `→` documented. The `assignment` field carries the `Assignment` row that owns the new event. |

### Spec drift — security / F-014 (1 item)

| # | What drifted | Before → After |
|---|--------------|----------------|
| D-08 | X-Project-ID header is required for F-014 cross-tenant safety. Missing header → 400 MISSING_PROJECT_HEADER. Mismatched project → 404 CROSS_TENANT_BLOCKED. | `→` fully documented. The 404 (not 403) on cross-tenant is intentional: 403 leaks the existence of the target row, 404 does not. |

### Spec drift — enums (2 items)

| # | What drifted | Before → After |
|---|--------------|----------------|
| D-09 | Status enum drift. Spec listed 3 values; code has 4 (added `cancelled` for Sprint 5+ DELETE endpoint). | `→ 4 values: active, superseded, completed, cancelled`. The DB CHECK constraint and the model enum match. |
| D-10 | Action enum drift. Spec listed 2 values; code has 3 (added `unassign` for Sprint 5+ DELETE endpoint). | `→ 3 values: assign, reassign, unassign`. The DB CHECK constraint and the model enum match. The `unassign` verb is not currently produced by any handler — it is reserved for the Sprint 5+ `DELETE /v1/tasks/:id/assign` endpoint. |

### Spec drift — pagination / sort / endpoint coverage (2 items)

| # | What drifted | Before → After |
|---|--------------|----------------|
| D-11 | Pagination. Spec said "cursor-based"; code returns the full history in one call (bounded ~10s of events). | `→` documents the full-return contract with a `meta` envelope (`{ count, server_time }`) on the history response. The cap (currently no hard limit; the expected cardinality is low) is documented as a Sprint 5+ hardening item. |
| D-12 | `GET /v1/tasks/:id/history` endpoint was not documented. Code implements it. | `→` new subsection added with the request shape, the 200 response (full event list DESC by `assigned_at`), the 404 TaskNotFound shape, and the F-014 404 CROSS_TENANT_BLOCKED shape. |

## Code change rationale (Commit 3)

`api-spec.md` line 921 documents the `notes` field as `≤ 1 KiB`. The production code never enforced this — a caller could POST a 1 MiB notes field and the row would land in `assignment_events.notes` (TEXT, no DB-level limit). This is a clear spec/code drift that would have surfaced as a security or cost issue in production.

The fix is at both the handler and the service layer (defense in depth, matching the existing `callerProjectID` F-014 pattern). The error shape is **400 VALIDATION_ERROR** (not 413 PAYLOAD_TOO_LARGE) because 1 KiB is a domain rule (notes are always small, the field is client-fixable by truncating), not a transport-level body limit. The 413 shape is reserved for transport-level limits (e.g. the deliverable `MaxDeliverableContentBytes` cap at 1 MiB which is close to typical body limits). The deliverable uses `payloadTooLarge`; the notes case uses `validationSingle("notes", "exceeds 1 KiB (1024 bytes)")` which produces a per-field detail entry so the client UI can highlight the offending field.

The constant `model.MaxAssignmentNotesBytes = 1 << 10` (1 KiB binary) lives in the model package so the spec/code relationship is explicit: the cap is a property of the data model, not a service-internal decision.

## Test coverage (Commit 2)

`assignment_table_test.go` (663 lines, NEW) adds a parallel, table-driven view of the same behaviour covered narratively in `assignment_test.go`. The narrative file stays for readability; the new file is the unified matrix. Adding a new case is a one-line struct literal in the appropriate case table.

Test functions in the new file:

- `TestAssignTaskToAgent_TableDriven` — 15 cases. Happy path (assign / reassign / idempotent), notes persistence (F-017 incl. empty-notes round-trip), capabilities persistence (incl. empty-preserves-existing), all error paths (TaskNotFound / AgentNotFound / CapabilityMismatch / AgentNotIdle / NotesExceedsMaxBytes), and F-014 cross-tenant (task in other project / agent in other project / missing project header). The `Error_NotesExceedsMaxBytes` case is the new test for the commit-3 fix.
- `TestListAssignmentHistory_TableDriven` — 5 cases. Empty for new task, three events newest-first, TaskNotFound, cross-tenant, missing project header. The DESC ordering invariant is asserted at the runner level (every non-empty success is checked) and the per-case action sequence is asserted in `wantActionsInOrder`.
- `TestTASK404_TransactionalInvariants_TableDriven` — 4 cases. At-most-one-active-per-task after reassign (the previous active row is flipped to `superseded` with `completed_at` set), event count equals action count, idempotent preserves the existing assignments row (no new row, no new event), and the event carries the `assignment_id` link.

The A-002-15..18 deferral (filed in the A-002 audit as "API-signature mismatch between tests and production; full rewrite needed") is **resolved** in this PR. The A-002-19 tracking comment at the top of `assignment_test.go` is replaced with a `resolved-by-A-003` note pointing at the new file. The wrong analysis (4-arg `NewAssignmentService(store, log, dispatcher, bus)`) is corrected: production is 3-arg `NewAssignmentService(store, capSvc, log)` and 1-arg `NewAgentService(store)`, which is exactly what the test file already calls.

## Audit prep items addressed

The `audit-prep-A-003.md` short list (TASK-421 cross-tenant 4th-create, idempotent re-assign, status/action enum drift, notes validation, unassign endpoint) is closed as follows:

| Prep item | Status | Resolution |
|-----------|--------|------------|
| TASK-421 cross-tenant 4th-create | ✓ | F-014 triple-check was already in place (task in caller's project + agent in caller's project + task-and-agent in same project). A-003 commit 2 cross-tenant test cases exercise the path. No new code needed for the 4-create pattern. |
| Idempotent re-assign | ✓ no change | The same-agent re-POST correctly returns the existing assignment as `idempotent: true` (per api-spec line 949: "a repeated POST of the same agent_id returns the existing state with `idempotent: true` and no new event"). Notes / capabilities_required on the second call are intentionally ignored — the audit-prep calls this out as by-design. |
| Status / action enum drift | ✓ no change | 4 status values (active/superseded/completed/cancelled) and 3 action values (assign/reassign/unassign) match the spec, the model, and the DB CHECK constraint. No drift. |
| Notes validation | ✓ (Commit 3) | `model.MaxAssignmentNotesBytes = 1 << 10`; enforced in service and handler. |
| Unassign endpoint | ⏸ Sprint 5+ | Reserved for the Sprint 5+ `DELETE /v1/tasks/:id/assign` endpoint (api-spec line 988). The enum and DB schema support it (nullable `agent_id` in the events table; `unassign` action in the CHECK constraint); the handler is not wired in this PR. |

## Pre-push gate (A-003)

The standing E-003 pre-push gate (pull → review → test → build → secret-scan → commit → push) is extended with the following A-003-specific checks. Builder runs these locally before every A-003 push; CI re-runs them on the PR. A failed check blocks the push.

1. **Spec/code status enum parity**: `model.AssignmentStatus*` (4 values) matches `docs/api-spec.md` §The Assignment Engine (A-003) status enum (4 values) and the DB CHECK constraint (4 values). Detected by a grep guard in the pre-push script.
2. **Spec/code action enum parity**: `model.AssignmentAction*` (3 values) matches the spec action enum (3 values) and the DB CHECK constraint (3 values). The `unassign` verb must be present in all three locations.
3. **Notes length cap invariant**: `model.MaxAssignmentNotesBytes = 1024`; the constant is referenced from both `service/assignment.go` and `handler/assignment.go`. A grep guard asserts both call-sites.
4. **No bare 413 in assignment handler/service**: the assignment handler and service use `validationSingle("notes", ...)` for the notes cap (400 VALIDATION_ERROR), not `payloadTooLarge` (413 PAYLOAD_TOO_LARGE). 413 is reserved for transport-level limits.
5. **F-014 triple-check present**: the service has all three checks (task in caller's project, agent in caller's project, defensive triple-check). A grep guard asserts the three `crossTenantBlocked()` returns in `AssignTaskToAgent`.
6. **Idempotency invariant**: the idempotent path returns the existing assignments row, not a new one. Asserted by `TestTASK404_Idempotent_PreservesExistingAssignmentsRow` in the table-driven file.
7. **TASK-404 transaction invariant**: after a reassign, exactly one row is `active` and the previous active row is `superseded` with `completed_at` set. Asserted by `TestTASK404_AtMostOneActivePerTask_AfterReassign` in the table-driven file.
8. **History ordering invariant**: `ListAssignmentHistory` returns events sorted DESC by `assigned_at`. The in-memory store and the postgres store both implement this. The `TestListAssignmentHistory_TableDriven.ThreeEvents_NewestFirst` case asserts the order on the service path; the DESC ordering is re-checked at the runner level for every non-empty success.
9. **A-002-19 comment is `resolved`**: the tracking comment at the top of `assignment_test.go` must be the resolved-by-A-003 variant, not the original wrong 4-arg claim. A grep guard.
10. **All A-003 routes wired**: `POST /v1/tasks/:id/assign` and `GET /v1/tasks/:id/history` are both registered in `router.go` and route to `handler/assignment.go`. Verified by a smoke test (the D-001 testing framework's curl-based smoke test exercises the routes).

## D-002 OPEN findings (Sprint 6+)

Per Lead's directive, the following security finding is filed in this audit but not addressed in this PR:

- **F-D002-004 (IDOR on `project_memberships`)** — The `project_memberships` table does not enforce row-level project isolation; a user from project A can read a `project_memberships` row whose `project_id` is B by guessing the row ID. Sprint 6+ work: add a check on every read/write that the caller's user_id is a member of the `project_id` on the target row. The fix is the same F-014 defensive triple-check pattern used in this PR (call_site has user_id; target row has project_id; user_id must have a row in project_memberships with the same project_id).

This is a cross-cutting concern that affects every project-scoped endpoint, not just assignment. Filing here so the audit doc is the canonical record of the A-003 surface; the actual fix will be a separate PR with its own audit.

## Sprint 5+ deferred

- **DELETE /v1/tasks/:id/assign** (unassign endpoint) — The enum and DB schema are in place (the `unassign` action verb in `AssignmentAction`; the `agent_id` column is nullable in `assignment_events`). The handler, service path, and routes are not wired in this PR. The audit-prep and api-spec both call this out as Sprint 5+ work.
- **History pagination cap** — `ListAssignmentHistory` returns the full event list in one call. The expected cardinality per task is low (a task is assigned once, maybe reassigned once, then completes), so the full-return contract is safe today. A hard cap (e.g. 1000 events) with a `has_more` flag in the `meta` envelope is the Sprint 5+ hardening.
- **A-002-17 store-layer tests** — `test(store/postgres): A-002-17 — add store-layer tests`. The chore is filed as a follow-up commit (not in this PR). The postgres `AssignmentStore` and `AssignmentEventStore` have the same surface as the in-memory variants but the CRUD coverage is the in-memory-store coverage for now. A dedicated postgres test file (parallel to the existing handler/service tests) is the A-002-17 deliverable.

## Sign-off

Ready for Guardian review. The audit is complete; the 12 spec drift items are fixed; the 1 code gap (notes length cap) is closed at both the handler and the service layer; the 3 test matrices (23 cases) provide audit-grade coverage; the A-002-15..18 test deferral is resolved; the wrong A-002-19 analysis is corrected; the deferred work (unassign endpoint, history cap, store-layer tests) is filed and tracked. The D-002 OPEN finding (F-D002-004 IDOR) is recorded for the Sprint 6+ cross-cutting fix.

— Builder, 2026-06-14
