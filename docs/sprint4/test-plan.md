# Sprint 4 — Test Plan (TASK-411)

**Status:** Draft for Lead review (TASK-411 in_progress)
**Owner:** Tester-01
**Date:** 2026-06-12
**Scope:** Sprint 4 deliverables — agents, capabilities, assignment, executions, deliverables.
**Execution point:** Tests are **written** on this host (no Go toolchain available on Windows
per the team norm). The Pre-Commit Quality Gate (TASK-414) is the canonical execution
point. This plan is the contract the CI gate will run against.

---

## 1. Scope and Approach

### 1.1 In scope
- **Integration tests** for cross-route flows (the unit-test surface is already covered
  by existing per-handler / per-service `*_test.go` files).
- **End-to-end lifecycle** test: `TestAgentLifecycle_CreateAssignExecuteDeliver` —
  Create agent → assign task → run execution → emit deliverable.
- **Defence-in-depth UUID validation** test (per Lead's scope addendum):
  `TestProjectScopedRoutes_RejectMalformedUUIDs` — list endpoints and assign body
  must return HTTP 400 (not 500) for non-UUID filter parameters, with the typed
  error envelope.
- **Acceptance report** (`acceptance-report.md`) summarising results, gaps, and
  the CI handoff.

### 1.2 Out of scope
- **Patch any 500s** that the malformed-UUID test surfaces. The fix is backend
  handler code; per Lead's instruction, I flag the failure in the acceptance
  report and ping Lead, who will open a backend patch task. My lane is the test.
- **Performance / load tests** — not part of Sprint 4 acceptance.
- **Browser-level UI tests** — covered by Developer-02's per-page tests for the
  agent / assignment / deliverable / activity UIs.
- **Security review** — TASK-412 (Security-01), already signed off 2026-06-12.
- **Migration execution tests** — DevOps-01's docker-compose stack (TASK-413)
  is the canonical path; integration tests use the in-memory store seed.

### 1.3 Approach (deviation: new test package)
- **New package:** `src/internal/integration/`. Reasoning: cross-route flows
  don't naturally live inside any single handler package, and `internal/router/`
  is a wire-up location, not a test surface. A dedicated `integration` package
  matches the intent ("test the system end-to-end") and keeps the existing
  per-handler / per-service unit tests untouched.
- **Real services, in-memory store:** the test file uses
  `store.NewMemoryStore()` (canonical in-process fixture; seeds the 6
  capabilities from `016_agent_registry.sql` per `store/memory.go:90-95`) and
  constructs the real Sprint 4 services against it:
  - `service.NewAgentService(memStore)`
  - `service.NewTaskService(memStore, log)`
  - `service.NewCapabilityService(memStore, log)`
  - `service.NewAssignmentService(memStore, capSvc, log)`
  - `service.NewExecutionService(memStore, log, nil)` — nil cfg = defaults
  - `service.NewDeliverableService(memStore, log)`
- **Minimal router built inline:** the test file registers only the routes
  needed for the lifecycle (`/v1/agents/*`, `/v1/projects/:projectId/tasks`,
  `/v1/tasks/*`, `/v1/tasks/:id/assign`, `/v1/tasks/:id/history`,
  `/v1/executions`, `/v1/deliverables`, `/v1/capabilities`).
- **Auth bypassed via test middleware:** a `r.Use(...)` sets `request_id` and
  `user_id` directly into the Gin context. The real `middleware.Auth` (JWT +
  API-key validation) is not invoked. This is acceptable for the lifecycle
  test because auth correctness is the responsibility of TASK-417 / TASK-418
  unit tests; the integration test exercises behaviour behind auth.
- **No local execution:** tests are written to compile and run under
  `go test ./internal/integration/...` on a Linux CI runner (ubuntu-latest per
  the team norm). This host cannot run them.

### 1.4 Existing test conventions (catalogued in §5)
The codebase already has a clear test pattern. Surveyed in `src/internal/`:
- **Per-handler `*_test.go`** in `handler/` — uses `httptest`, hand-rolled
  mock services, `assert` / `require` from `testify`, `X-Project-ID` header
  in helpers.
- **Per-service `*_service_test.go`** in `service/` — uses
  `store.NewMemoryStore()` + real service, exercises the service surface.
- **Error envelope:** `{"error": {"code", "message", "details"}, "request_id"}`.
- **Standard status codes:** 400 (`VALIDATION_ERROR` / `INVALID_JSON`),
  401 / 403 (auth), 404 (`NOT_FOUND`), 409 (`VERSION_CONFLICT` /
  `CAPABILITY_MISMATCH`), 500 (internal).

---

## 2. Test Inventory

| ID | File | Type | Description | Sprint 4 task |
|---|---|---|---|---|
| T1 | `integration_test.go` | Integration | `TestAgentLifecycle_CreateAssignExecuteDeliver` — full happy-path E2E | TASK-402, 403, 404, 405, 406 |
| T2 | `integration_test.go` | Integration | `TestProjectScopedRoutes_RejectMalformedUUIDs` — defence-in-depth UUID validation | TASK-409 (scope addendum) |
| T3 (existing) | `handler/agent_test.go` | Unit | Agent CRUD + capabilities + X-Project-ID | TASK-402, 403 |
| T4 (existing) | `handler/capability_test.go` | Unit | Capability catalog + per-agent list | TASK-403 |
| T5 (existing) | `handler/assignment_test.go` | Unit | Assign / history endpoint | TASK-404 |
| T6 (existing) | `handler/execution_test.go` | Unit | Execution CRUD + list filters | TASK-405 |
| T7 (existing) | `handler/deliverable_test.go` | Unit | Deliverable CRUD + versions | TASK-406 |
| T8 (existing) | `service/agent_service_test.go` | Unit | Agent service against memory store | TASK-402 |
| T9 (existing) | other `*_service_test.go` | Unit | Service-layer tests | TASK-403-406 |

**New tests contributed by TASK-411:** T1 and T2. All others are existing
in-tree coverage referenced here so the CI gate runs a single coherent suite.

---

## 3. Test Cases (T1 — Lifecycle)

### `TestAgentLifecycle_CreateAssignExecuteDeliver`

End-to-end happy path that exercises the five Sprint 4 entities in sequence.
Every step asserts the typed error envelope is absent (`request_id` set, no
`error` block) and the response shape matches `api-spec.md`.

| Step | Call | Expected status | Assertion |
|---|---|---|---|
| 1.1 | `POST /v1/agents` (no `X-Project-ID`) | 400 | `VALIDATION_ERROR` "X-Project-ID header is required" (defence-in-depth baseline) |
| 1.2 | `POST /v1/agents` (valid body) | 201 | `data.id` is a UUID; `data.status == "initializing"`; `data.capabilities` echoes input |
| 1.3 | `PUT /v1/agents/:id` (replace capabilities with `["coding","testing"]`) | 200 | `data.version` bumped; `data.capabilities` == new set |
| 1.4 | `GET /v1/agents/:id/capabilities` | 200 | array contains `coding` and `testing` with `granted_at` |
| 1.5 | `POST /v1/projects/:projectId/tasks` | 201 | `data.id` is a UUID, `data.status == "backlog"` |
| 1.6 | `POST /v1/tasks/:id/assign` body `{"agent_id": "...", "capabilities_required": ["coding"]}` | 200 | `data.event.action == "assigned"`; `data.idempotent == false`; task now has active assignment |
| 1.7 | `POST /v1/tasks/:id/assign` re-POST (idempotency) | 200 | `data.idempotent == true`; no new event |
| 1.8 | `GET /v1/tasks/:id/history` | 200 | exactly 1 event (the original assign; idempotent re-POST does not append) |
| 1.9 | `POST /v1/executions` body `{"task_id": "...", "agent_id": "..."}` | 201 | `data.id` is a UUID, `data.status == "queued"` |
| 1.10 | `PATCH /v1/executions/:id` `{"status": "running"}` | 200 | status transitioned |
| 1.11 | `PATCH /v1/executions/:id` `{"status": "completed"}` | 200 | status transitioned; agent's `last_active_at` updated |
| 1.12 | `GET /v1/agents/:id` | 200 | `data.last_active_at` is non-nil and recent |
| 1.13 | `POST /v1/deliverables` body `{"task_id": "...", "agent_id": "...", "content": "..."}` | 201 | `data.id` is a UUID, `data.version == 1` |
| 1.14 | `PUT /v1/deliverables/:id` with new content | 200 | `data.version == 2` |
| 1.15 | `GET /v1/deliverables/:id/versions` | 200 | exactly 2 version rows, ordered |

**Test teardown:** the in-memory store is per-test (created in
`newIntegrationRouter`); no cleanup needed.

**Known non-assertions** (documented, not bugs):
- 1.10 / 1.11 timing: the MOCK execution goroutine in TASK-405 may auto-promote
  status; the test asserts the *post-condition* (final status == "completed"),
  not the precise transition timing.

---

## 4. Test Cases (T2 — Malformed UUIDs)

### `TestProjectScopedRoutes_RejectMalformedUUIDs`

Defence-in-depth: list endpoints and the assign body must return HTTP 400
with the typed error envelope when filter UUIDs are not valid UUIDs. A 500
is treated as a backend bug and flagged in the acceptance report.

**Table-driven sub-cases** (each row is one `t.Run`):

| # | Route | Filter / Body | Input | Expected | Notes |
|---|---|---|---|---|---|
| 2.1 | `GET /v1/agents` | `?project_id=not-a-uuid` | `not-a-uuid` | 400 + `VALIDATION_ERROR` | New coverage — no existing test for `project_id` filter |
| 2.2 | `GET /v1/agents` | `?project_id=` | (empty) | 200 + empty list (no filter applied) | Empty = "filter not provided", not 400. Documented behaviour. |
| 2.3 | `GET /v1/agents` | `?project_id=12345678-1234-1234-1234-12345678901Z` | (bad char) | 400 | Near-miss case |
| 2.4 | `GET /v1/executions` | `?task_id=not-a-uuid` | `not-a-uuid` | 400 | |
| 2.5 | `GET /v1/executions` | `?agent_id=not-a-uuid` | `not-a-uuid` | 400 | |
| 2.6 | `GET /v1/executions` | `?status=garbage` | `garbage` | 400 + `INVALID_EXECUTION_STATUS` | Already in `TestExecutionHandler_List_400_BadStatus` — re-asserted for completeness |
| 2.7 | `GET /v1/deliverables` | `?task_id=not-a-uuid` | `not-a-uuid` | 400 | |
| 2.8 | `GET /v1/deliverables` | `?agent_id=not-a-uuid` | `not-a-uuid` | 400 | |
| 2.9 | `POST /v1/tasks/:id/assign` | path `:id` | `not-a-uuid` | 400 | Already in `TestAssignmentHandler_Assign_InvalidTaskID` — re-asserted for completeness |
| 2.10 | `POST /v1/tasks/{valid-uuid}/assign` | body `agent_id` | `not-a-uuid` | 400 | Already in `TestAssignmentHandler_Assign_InvalidAgentID` — re-asserted for completeness |
| 2.11 | `POST /v1/tasks/{valid-uuid}/assign` | body `agent_id` | (empty) | 400 + "agent_id is required" | Empty-string check is in the existing test |

**Assertion helper** (in the test file):
```go
func assertValidationError400(t *testing.T, body []byte) {
    var env struct {
        Error struct {
            Code    string `json:"code"`
            Message string `json:"message"`
            Details any    `json:"details"`
        } `json:"error"`
        RequestID string `json:"request_id"`
    }
    require.NoError(t, json.Unmarshal(body, &env))
    assert.Equal(t, "VALIDATION_ERROR", env.Error.Code,
        "expected VALIDATION_ERROR, got %q (body=%s)", env.Error.Code, body)
    assert.NotEmpty(t, env.Error.Message)
    assert.NotEmpty(t, env.RequestID)
}
```

**Flagging rule (re-stated from Lead's addendum):** if any sub-case observes a
500, the test is marked `t.Skip` with a `BUG: <route> <param>` log, the
exact observed response is recorded in the acceptance report, and Lead is
pinged. The test is not failed; the gap is the backend fix, not the test.

---

## 5. Patterns Cataloged (Appendix)

Surveyed `src/internal/handler/*_test.go` and `src/internal/service/*_service_test.go`.

### 5.1 Handler test convention
```go
gin.SetMode(gin.TestMode)
r := gin.New()
r.Use(func(c *gin.Context) {
    c.Set("request_id", "test-rid-...")
    c.Next()
})
h := NewXxxHandler(mockSvc)
r.POST("/v1/...", h.Create)
// doRequest helper sets Content-Type, X-Project-ID, marshals body
```

### 5.2 Service test convention
```go
memStore := store.NewMemoryStore()
svc := service.NewXxxService(memStore, ...)
apiErr := svc.SomeMethod(...)
require.Nil(t, apiErr)
```

### 5.3 Error envelope
```json
{ "error": { "code": "VALIDATION_ERROR", "message": "...", "details": [...] },
  "request_id": "test-..." }
```

### 5.4 Project scoping convention
`X-Project-ID` is **required** on `/v1/agents/*` (and only on that route group).
Per the `project-scoping-convention` memory: omit = 400, no sentinel value
accepted. The lifecycle test honours this on agent routes and ignores it on
execution / deliverable / assign routes (which do not require it).

### 5.5 Capability assignment convention
Per `capability-assignment-via-put` memory: add/remove goes through
`PUT /v1/agents/:id` with the `capabilities` array (replace semantics).
There is no `POST /v1/agents/:id/capabilities` endpoint. The lifecycle test
reflects this in step 1.3.

---

## 6. Acceptance Criteria

TASK-411 is accepted when **all** of the following are true:

- [ ] `docs/sprint4/test-plan.md` exists with this structure (this file).
- [ ] `src/internal/integration/integration_test.go` exists and contains at
      minimum T1 and T2 as named test functions.
- [ ] `docs/sprint4/acceptance-report.md` exists with results, gaps, and
      CI handoff notes.
- [ ] The two integration tests compile (verified by CI gate build step).
- [ ] The two integration tests pass on the CI gate (ubuntu-latest Go
      toolchain, in-memory store — no Postgres required for these tests).
- [ ] If T2 observed any 500s, they are recorded in the acceptance report
      with route, parameter, observed status, and observed body, and Lead
      has been pinged with the specifics.
- [ ] No commits are made by Tester-01; TASK-415 (DevOps-01) is the
      closeout commit per the team norm.

---

## 7. CI Handoff (for DevOps-01, TASK-414 / 415)

- **Go module path:** `github.com/fadhilfathi/AI-Software-Factory`
- **Test command:** `go test -v -race ./internal/integration/...`
- **No external services required** for the integration suite — the
  in-memory store self-seeds the 6 canonical capabilities.
- **Expected runtime:** < 5 seconds (no I/O, no goroutines that wait on
  network).
- **Build dependency:** `github.com/gin-gonic/gin`,
  `github.com/google/uuid`, `github.com/stretchr/testify`,
  `go.uber.org/zap` (all already in `go.mod`).
- **Race detector:** enabled — the lifecycle test exercises a goroutine
  for the MOCK execution service; the race detector should be clean.

---

## 8. Risks and Deviations

| # | Item | Impact | Mitigation |
|---|---|---|---|
| D1 | Tests written on Windows host with no Go toolchain | Cannot verify they compile locally | CI gate is the canonical verifier; pre-commit gate (TASK-414) will fail the sprint if they don't compile |
| D2 | New package `internal/integration` | Adds one new dir to the repo | Tree is small, package is single-purpose, will be reviewed as part of the closeout commit |
| D3 | Auth bypassed via test middleware | Integration tests do not exercise auth correctness | Auth is covered by TASK-417 / TASK-418 unit tests; integration tests focus on behaviour behind auth |
| D4 | MOCK execution timing in 1.10 / 1.11 | Status transitions may be auto-promoted | Test asserts post-conditions, not transition timing |
| D5 | `service.NewExecutionService` nil-cfg behaviour | Sprint 4 may introduce a required cfg | Verified nil is accepted in Sprint 4 code; if Sprint 5 changes this, the test is updated with a `&service.DefaultExecutionServiceConfig{}` |

---

## 9. Open Questions for Lead

1. **T1 step 1.1:** assert that `POST /v1/agents` without `X-Project-ID`
   returns 400. This is the project-scoping convention, but it's a "happy
   path negative" — should it be in the lifecycle test, or a separate
   test? Default: in the lifecycle test, since it's step 1.
2. **T2 row 2.6 (`status=garbage`):** re-asserted for completeness. If
   the existing `TestExecutionHandler_List_400_BadStatus` already covers
   it, I can drop the row. Default: keep it (belt-and-braces for the CI
   gate's single coherent run).
3. **Idempotency assertion in 1.7:** the existing `TestAssign_Idempotent`
   may already cover this. Default: keep it in T1 for full E2E visibility.
4. **Memory store vs real Postgres:** the in-memory store seeds the 6
   canonical capabilities but may differ from the real Postgres in
   index-only or constraint-only ways. If Lead wants Postgres-backed
   integration, that's a TASK-413-style docker-compose dance and would
   push to Sprint 5.

---

*End of test plan. Awaiting Lead review.*
