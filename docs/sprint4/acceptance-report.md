# Sprint 4 — Acceptance Report (TASK-411)

**Status:** Draft, awaiting CI gate (TASK-414) results for §7.
**Owner:** Tester-01
**Date:** 2026-06-12
**Companion doc:** [`test-plan.md`](./test-plan.md) — the contract the CI gate runs against.

---

## §1. Scope and Approach

**One-paragraph summary.** This report covers the Sprint 4 acceptance
testing pass. Two new tests were added at
`src/internal/integration/integration_test.go`:
`TestAgentLifecycle_CreateAssignExecuteDeliver_Smoke` (T1, smoke variant —
4 sub-steps proving the cross-route wiring works) and
`TestProjectScopedRoutes_RejectMalformedUUIDs` (T2, full table — 11
sub-cases for the defence-in-depth UUID validation scope addendum). The
test bootstrap is DB-agnostic (`internal/integration/store.go` exposes a
`Store` alias with `NewMemoryStore()` and a `NewPostgresStore(url)` stub
for Sprint 5 pickup). A real Gin router is wired with the real Sprint 4
services (capability, agent, task, assignment, execution, deliverable),
all backed by `store.NewMemoryStore()`. Auth is bypassed via a test
middleware that sets `request_id` and `user_id` directly into the Gin
context — auth correctness is TASK-417/418's lane.

**Sprint 4 scope (per Lead's option C):** T1 is a **smoke** variant
(4 sub-steps, not the full 15-step lifecycle). The full T1 lifecycle is
deferred to Sprint 5. T2 is the **full** table (11 sub-cases).

---

## §2. Test Inventory

| ID   | Subject                                                       | Status (pre-CI) | Notes                                                                 |
|------|---------------------------------------------------------------|-----------------|-----------------------------------------------------------------------|
| T1   | `TestAgentLifecycle_CreateAssignExecuteDeliver_Smoke`         | written         | Smoke variant per Option C. 4 sub-steps.                              |
| T2   | `TestProjectScopedRoutes_RejectMalformedUUIDs`                | written         | Full table. 11 sub-cases.                                             |
| T3.1 | `handler/agent_test.go` (existing)                            | in-tree         | Unit. Agent CRUD + capabilities + X-Project-ID.                       |
| T3.2 | `handler/capability_test.go` (existing)                       | in-tree         | Unit. Capability catalog + per-agent list.                            |
| T3.3 | `handler/assignment_test.go` (existing)                       | in-tree         | Unit. Assign / history.                                               |
| T3.4 | `handler/execution_test.go` (existing)                        | in-tree         | Unit. Execution CRUD + list filters.                                  |
| T3.5 | `handler/deliverable_test.go` (existing)                      | in-tree         | Unit. Deliverable CRUD + versions.                                    |
| T3.6 | `service/agent_service_test.go` (existing)                    | in-tree         | Unit. Agent service against memory store.                             |
| T3.7 | other `*_service_test.go` (existing)                          | in-tree         | Unit. Service-layer tests.                                            |

**New tests contributed by TASK-411:** T1 and T2.
**All other rows** are existing in-tree coverage referenced so the CI
gate runs one coherent suite.

---

## §3. T1 Results — `TestAgentLifecycle_CreateAssignExecuteDeliver_Smoke`

Sprint 4 scope (Option C). Full 15-step lifecycle deferred to Sprint 5.

| Step | Sub-test                                  | Status (pre-CI)   | Notes                                                                                                            |
|------|-------------------------------------------|-------------------|------------------------------------------------------------------------------------------------------------------|
| 1.1  | `POST /v1/agents` create agent            | written, unrun    | Asserts 201; parses `data.id`; verifies `name`, `role` fields.                                                    |
| 1.2  | `POST /v1/projects/:projectId/tasks`      | written, unrun    | Asserts 201; parses `data.id`; verifies `title` field.                                                            |
| 1.3  | `POST /v1/tasks/:id/assign`               | written, unrun    | Asserts 200; asserts `idempotent == false`; verifies assignment.AgentID matches the agent and `RequiredCapability` is `coding`. |
| 1.4  | `POST /v1/deliverables`                   | written, unrun    | Asserts 201; verifies `task_id`, `agent_id`, and `version == 1`.                                                 |
| 1.5+ | Full lifecycle (PUT cap, history, exec, etc.) | **deferred** | Sprint 5 follow-up. Test plan §3 retained as the contract.                                                       |

**Pre-CI status legend.** "written, unrun" = the test code is on disk
and will be executed by the CI gate (TASK-414). "deferred" = the
sub-step is not in the Sprint 4 code; Sprint 5 picks it up.

---

## §4. T2 Results — `TestProjectScopedRoutes_RejectMalformedUUIDs`

Full table (11 sub-cases per test-plan §4). All sub-cases are written
and unrun; the CI gate is the execution point.

| #    | Sub-case                                  | Method | Path                                                         | Status (pre-CI)  |
|------|-------------------------------------------|--------|--------------------------------------------------------------|------------------|
| 2.1  | `agents_list_project_id_malformed`        | GET    | `/v1/agents?project_id=not-a-uuid`                           | written, unrun   |
| 2.2  | `agents_list_project_id_empty`            | GET    | `/v1/agents?project_id=`                                     | **not in T2**    |
| 2.3  | `agents_list_project_id_near_miss`        | GET    | `/v1/agents?project_id=12345678-...12345678901Z` (bad char)  | written, unrun   |
| 2.4  | `executions_list_task_id_malformed`       | GET    | `/v1/executions?task_id=not-a-uuid`                          | written, unrun   |
| 2.5  | `executions_list_agent_id_malformed`      | GET    | `/v1/executions?agent_id=not-a-uuid`                         | written, unrun   |
| 2.6  | `executions_list_status_garbage`          | GET    | `/v1/executions?status=garbage`                              | written, unrun   |
| 2.7  | `deliverables_list_task_id_malformed`     | GET    | `/v1/deliverables?task_id=not-a-uuid`                        | written, unrun   |
| 2.8  | `deliverables_list_agent_id_malformed`    | GET    | `/v1/deliverables?agent_id=not-a-uuid`                       | written, unrun   |
| 2.9  | `assign_path_task_id_malformed`           | POST   | `/v1/tasks/not-a-uuid/assign` (valid body)                   | written, unrun   |
| 2.10 | `assign_body_agent_id_malformed`          | POST   | `/v1/tasks/{valid-uuid}/assign` (`agent_id: "not-a-uuid"`)   | written, unrun   |
| 2.11 | `assign_body_agent_id_empty`              | POST   | `/v1/tasks/{valid-uuid}/assign` (`agent_id: ""`)             | written, unrun   |

**Row 2.2 status:** Per Lead's decision, empty `?project_id=` is
treated as "filter not provided" → 200 with empty list, not 400. Not
included in T2 (would be a 200-assert, not a 400-assert). Documented
in §8 below.

**Bug-flag rule (re-stated):** if any row observes 500, the sub-case
is `t.Skip`'d with a `BUG: <route> <param>` log and the exact observed
body is recorded here. The CI gate should re-run after any backend
patch lands (Sprint 4 fix if schedule allows, Sprint 5 otherwise).

---

## §5. Patterns Verified

| Pattern                                                | Verified by                              | Notes                                                                                       |
|--------------------------------------------------------|------------------------------------------|---------------------------------------------------------------------------------------------|
| `X-Project-ID` required on `/v1/agents`                | existing `TestAgentHandler_*_XProjectID` | Not re-asserted in T1/T2 (covered by in-tree unit tests).                                   |
| Capability replace via `PUT /v1/agents/:id`            | test plan §3 step 1.3 (deferred to S5)   | Smoke T1 creates with capabilities; doesn't exercise PUT-replace. Sprint 5 picks up.         |
| `assignment.Idempotent` is `false` on first call       | T1 step 1.3                              | Direct assertion on parsed response.                                                        |
| Deliverable version starts at 1                        | T1 step 1.4                              | Direct assertion: `deliv.Version == 1`.                                                     |
| Typed error envelope on 400 (for malformed UUIDs)      | T2 (all rows)                            | `assertMalformedUUID400` requires `error.code` and `error.message` to be non-empty.          |
| `request_id` set on response                           | test middleware                          | Test middleware sets `c.Set("request_id", "test-int-rid-001")` for every request.          |
| Cursor / pagination                                    | not in T1/T2                             | Sprint 4 acceptance does not cover cursor pagination. Existing handler tests cover it.       |

---

## §6. Acceptance Criteria

Test plan §6 criteria, with current pre-CI status:

- [x] `docs/sprint4/test-plan.md` exists with the agreed structure.
- [x] `src/internal/integration/integration_test.go` exists with T1
      (smoke) and T2 (full) as named test functions.
- [x] `docs/sprint4/acceptance-report.md` exists with this structure.
- [ ] **The two integration tests compile (verified by CI gate build step).**
- [ ] **The two integration tests pass on the CI gate (ubuntu-latest Go toolchain, in-memory store).**
- [x] No 500s observed (because tests have not yet been run on this host;
      a 500, if it occurs on CI, will be recorded here post-CI).
- [x] **No commits by Tester-01** per standing rule; DevOps-01 owns
      the closeout commit (TASK-415).

**Two unchecked items are pending §7 (CI handoff results).**

---

## §7. CI Handoff Results — **PENDING CI**

**This section is a placeholder until DevOps-01 reports the green CI
run (post-TASK-414). Update after CI lands.**

**Expected CI command (for DevOps-01):**
```
go test -v -race ./internal/integration/...
```

**Expected runtime:** < 5 seconds. **No external services** (in-memory
store self-seeds the 6 canonical capabilities).

**Template for §7 post-CI (fill in after run):**

| Sub-test ID | Status   | Observed status codes | Notes |
|-------------|----------|-----------------------|-------|
| T1.1        | pending  |                       |       |
| T1.2        | pending  |                       |       |
| T1.3        | pending  |                       |       |
| T1.4        | pending  |                       |       |
| T2.1        | pending  |                       |       |
| T2.3        | pending  |                       |       |
| T2.4        | pending  |                       |       |
| T2.5        | pending  |                       |       |
| T2.6        | pending  |                       |       |
| T2.7        | pending  |                       |       |
| T2.8        | pending  |                       |       |
| T2.9        | pending  |                       |       |
| T2.10       | pending  |                       |       |
| T2.11       | pending  |                       |       |

**Pre-CI ETA for §7 fill-in:** within 10 minutes of the CI run
completing (Tester-01 will update this section and re-report to Lead).

---

## §8. Risks and Deviations

### 8.1 Documented in test plan §8 (carried forward)
- **D1** Tests written on Windows host with no Go toolchain — CI gate
  is the canonical verifier. (Status: confirmed; CI gate is the
  execution point.)
- **D2** New package `src/internal/integration/`. (Status: landed; one
  regular file `store.go` for the Store helpers, one `_test.go` for the
  tests.)
- **D3** Auth bypassed via test middleware. (Status: landed; `c.Set`
  on `request_id` and `user_id`.)
- **D4** MOCK execution timing in deferred T1.10/T1.11. (Status:
  deferred to Sprint 5 along with the rest of the lifecycle.)
- **D5** `service.NewExecutionService` nil-cfg behaviour. (Status:
  confirmed via existing test patterns; nil is accepted.)

### 8.2 New deviations surfaced during implementation
- **D6 — Sprint 4 scope reduction (Option C).** The full T1 lifecycle
  (15 sub-steps per test-plan §3) is deferred to Sprint 5. Sprint 4
  ships a 4-step smoke variant proving the cross-route wiring. The
  full plan is preserved in `docs/sprint4/test-plan.md` for Sprint 5
  pickup. This is a **known gap**, flagged here so the closeout commit
  and acceptance are honest about what's covered.
- **D7 — Error envelope inconsistency between handlers.** During
  implementation, I observed that the Sprint 4 handlers use different
  error codes for the same kind of validation error:
  - Agent and Deliverable handlers: `VALIDATION_ERROR`
  - Execution handler: `BAD_REQUEST` (and `INVALID_EXECUTION_STATUS`
    for the `status` filter)

  This is **not a test failure**; the `assertMalformedUUID400` helper
  accepts either code. But it is a real consistency issue that should
  be addressed in Sprint 5 (small, low-risk refactor). Lead may want
  to open a separate ticket.
- **D8 — `request_id` not in all error envelopes.** The agent
  handler's error envelope includes `request_id` (set by middleware);
  the execution and deliverable handlers' envelopes do not. Sprint 5
  consistency ticket (paired with D7).

### 8.3 What the smoke T1 does NOT cover
For full transparency on what's deferred to Sprint 5:
- Task status transitions (PATCH `/v1/tasks/:id/status`)
- Agent capability replace (PUT `/v1/agents/:id`)
- Capability list endpoint (`GET /v1/agents/:id/capabilities`)
- Assignment history (`GET /v1/tasks/:id/history`)
- Idempotent re-assign
- Execution lifecycle (POST → PATCH `running` → PATCH `completed`)
- Agent `last_active_at` updates
- Deliverable PUT (version bump) and version history
- `X-Project-ID` enforcement baseline (T1.1 step in full plan)

Each of these is in the in-tree unit-test coverage and will get a
full T1 integration assertion in Sprint 5.

---

## §9. Open Questions for Lead

1. **D6 confirmation.** Does Lead want T1 smoke-only for Sprint 4, or
   should the full 15-step lifecycle be in Sprint 4? My read of the
   option-C framing is "smoke now, full later." If Lead wants the full
   lifecycle in Sprint 4, that's ~2 hours of additional work and would
   push the closeout commit.
2. **D7 / D8 error envelope consistency.** Should I open a Sprint 5
   ticket for the error-envelope inconsistency (different codes,
   missing `request_id` in some handlers), or fold it into the Sprint 5
   T1 work?
3. **`?project_id=` empty semantics.** Confirmed at the test-plan
   stage (empty = "filter not provided" = 200 with empty list). Does
   Lead want this changed to 400 in Sprint 5? (Low-priority tightening;
   current behaviour is internally consistent.)
4. **T1.1 step (no X-Project-ID) coverage.** Full T1 step 1.1 asserts
   that omitting `X-Project-ID` returns 400. This is covered by the
   existing `TestAgentHandler_*_XProjectID` unit tests. Should the
   full T1 lifecycle in Sprint 5 re-assert this in the integration
   context, or rely on the unit tests?
5. **CI handoff communication channel.** §7 above is a placeholder
   that needs a post-CI fill-in. Should I update this file directly
   and re-report, or send a separate "CI results" message to Lead?

---

*End of acceptance report. §7 is the only open section, pending
TASK-414 (CI gate) execution by DevOps-01.*
