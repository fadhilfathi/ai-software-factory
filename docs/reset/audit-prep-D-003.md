# D-003 Workflow Validation — Pre-Scope Brief

**Owner of D-003:** Guardian. Support: Ops.
**Spec:** `docs/sprint4/test-plan.md` §1 (T1 — full lifecycle), `docs/sprint4/acceptance-report.md`
**Audit precedent:** `docs/reset/audit/A-001-audit.md`

## Deliverables (per the team contract)
End-to-end validation of the flow:
**Project → Task → Assignment → Execution → Deliverable → Done**

## Existing code floor (READ FIRST)

| File | Purpose | Status |
|------|---------|--------|
| `src/internal/integration/integration_test.go` | 407 lines. T1 (4-step smoke), T2 (11 sub-cases malformed-UUID). Sprint 4 TASK-411. | NOW COMPILES (A-002-12 fixed the import drift). T1 is a 4-step smoke; full 15-step lifecycle is deferred. |
| `src/internal/integration/router_test_helpers.go` | `newIntegrationRouter(t, s)` wires a real Gin router + Sprint 4 services + in-memory store. Bypasses auth via test middleware. | Read. |
| `docs/sprint4/test-plan.md` | 15-step T1 lifecycle | Read. |
| `docs/sprint4/acceptance-report.md` | Sprint 4 acceptance (T1 smoke green) | Read. |

## What D-003 needs to do

1. **Extend T1 from 4 steps to the full 15-step lifecycle** in `integration_test.go`. The 4-step smoke covers: create agent, create task, assign task, get assignment. The full lifecycle should add:
   - 5: list assignments
   - 6: list assignment history
   - 7: create execution
   - 8: list executions
   - 9: update execution status (running → review → completed)
   - 10: create deliverable
   - 11: list deliverables
   - 12: get deliverable
   - 13: create deliverable version
   - 14: list deliverable versions
   - 15: final state assertions (assignments active count, executions terminal count, deliverables visible)

2. **For each of the 15 steps, assert on the response and on the in-memory store state.** The Sprint 4 test stopped at the GET. The full lifecycle test should:
   - Use the response body (now wrapped in `{data: ...}` envelope per the A-002-11 deliverable change)
   - Use store queries to verify the cross-service state (e.g. assignment_events table got the right action; executions lifecycle is correct)
   - Cleanly fail with a clear assertion message if any step is off

3. **Add the 6-state lifecycle test path** (per the B-001 pre-scope at `docs/reset/audit-prep-B-001.md`). The integration test should exercise `queued → assigned → running → review → completed` (and a parallel path for `failed`). If B-001 hasn't shipped the 6-state change yet, the test should be conditional / skipped with a clear marker.

4. **Add the cross-tenant negative test** (per F-D002-004 / F-013 / F-014). The test should:
   - Create project A and project B with a user that is a member of A only
   - Try to read a task in B (should 404 crossTenantBlocked)
   - Try to assign the task in B (should 404 crossTenantBlocked)
   - Document the surface in the deliverable; do NOT try to fix the `project_memberships` table gap (Sprint 6+ work).

5. **Add a 3-deliverable happy path test** for `TestDeliverableHandler_List_WithFilters` (the redesign that landed in A-002-11). Make sure the redesigned test runs and the cross-tenant 4th create is properly blocked (per the TASK-421 fix).

6. **Wire the integration test into CI** (`.github/workflows/sprint-quality-gate.yml` has 14 steps; check if the integration test runs as part of the `go test -race` step, or if it needs a separate step). The `sprint-quality-gate.yml` step "integration tests" already exists (per Ops' CI report) — verify the trigger is correct.

7. **Deliverable**: `docs/reset/workflow-validation.md`. Format:
   - Executive summary (1 paragraph)
   - Step table: `Step | Method | Status | Notes` for the 15 steps + 1 negative test
   - For each FAIL: a fix-it ticket (or a reference to an existing ticket if it's already filed)
   - For each PASS: the test name + commit SHA that last touched it
   - Final verdict: PASS / PARTIAL / FAIL

8. **Push to main** under E-003 when done.

## When this must land

After A-002-09..18 fully clears the test gate. The integration test is now compilable, so D-003 can start in parallel with the rest of A-002-15, A-002-16, A-002-18. Coordinate with Builder so the integration test isn't fighting a half-fixed service surface.

## What I (Leader) will do

- Review the 15-step table for completeness.
- Surface the F-D002-004 cross-tenant negative test in the D-002 report.
- Cross-ref with the B-001 6-state lifecycle: if B-001 hasn't shipped yet, the T1 test should mark the new states as `skipped` with a TODO.

## What Builder (B-001) will do (downstream)

- Land the 6-state lifecycle so the integration test can exercise it.
- Provide a release note when B-001 ships so the T1 test can be unskipped.

## What Ops (E-001) will do (downstream)

- Confirm the integration test step in `sprint-quality-gate.yml` is wired correctly.
- Run a `docker compose up` smoke that exercises the integration test in a Linux container (the local Windows host can't run it without gcc + Go).
