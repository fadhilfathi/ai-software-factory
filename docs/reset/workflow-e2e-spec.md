# D-003 — Workflow Validation Spec

| Field    | Value                                                           |
|----------|-----------------------------------------------------------------|
| Owner    | Guardian (slot `019ec4fe-604f-7551-9a76-38a621ddd256`)          |
| Status   | **AUTHORED — implements pre-scope at `audit-prep-D-003.md`**    |
| Date     | 2026-06-14                                                      |
| Reviewer | Guardian (D-003) + Builder (re-review on first run)             |
| Pre-req  | A-003 (DONE @ c4852b1) + integration scaffold (DONE @ 00bc9a5)  |

This spec is the wire-level contract for the D-003 deliverable: the T1 happy-path
E2E flow plus the cross-tenant negative tests. The audit (D-003) verifies that
the actual code matches this spec end-to-end via `go test -race -shuffle=on`.

---

## 1. T1 — Happy-path 15-step E2E flow

The flow Project → Task → Assignment → Execution → Deliverable → Done,
exercised through the real Gin router, real service layer, real in-memory
store, and the Aion `MockRuntime` (Mode A — no subprocess).

| #   | Step                                            | Endpoint                              | Status (start → end)              | Body shape / key fields                                                                  |
|-----|-------------------------------------------------|---------------------------------------|-----------------------------------|------------------------------------------------------------------------------------------|
| 1.1 | Create agent                                    | `POST /v1/agents`                     | — → 201                           | `{name, role, capabilities, metadata?}`; agent starts `initializing`                     |
| 1.2 | Read agent                                      | `GET /v1/agents/:id`                  | — → 200                           | full `agentResponse`                                                                     |
| 1.3 | Replace agent capabilities                      | `PUT /v1/agents/:id`                  | — → 200                           | `updateAgentRequest{capabilities: [...], version: N}` → version becomes N+1              |
| 1.4 | List agent capabilities                         | `GET /v1/agents/:id/capabilities`     | — → 200                           | array of `AgentCapabilityView` (per-capability display name, category, etc.)              |
| 1.5 | Create task in project                          | `POST /v1/projects/:id/tasks`         | — → 201                           | `{title, description?, priority?}`; default priority `medium`                            |
| 1.6 | Assign task to agent                            | `POST /v1/tasks/:id/assign`           | — → 200 (idempotent: **false**)   | `{agent_id, capabilities_required?, notes?}`; writes assignment_events row               |
| 1.7 | Re-POST same assign (idempotency)               | `POST /v1/tasks/:id/assign`           | — → 200 (idempotent: **true**)    | same body; service returns existing event with no new row written                        |
| 1.8 | List assignment history                         | `GET /v1/tasks/:id/history`           | — → 200, exactly 1 event          | `assignment_events` newest-first; one `assign` event from 1.6                            |
| 1.9 | Create execution                                | `POST /v1/executions`                 | — → 201, status: `assigned`       | `{task_id, agent_id, runtime?}`; worker is **not** auto-spawned (Sprint 6 shape)         |
| 1.10| PATCH execution to running                      | `PATCH /v1/executions/:id`            | `assigned` → 200, status: `running`| `{status: "running"}`                                                                    |
| 1.10a| Mock runtime auto-transition                    | (in-process goroutine)                | `running` → `review`              | MockRuntime completes after script's `Delay`; worker calls `driveWorker` → `review`      |
| 1.11| PATCH execution to completed                    | `PATCH /v1/executions/:id`            | `review` → 200, status: `completed`| `{status: "completed"}`                                                                  |
| 1.12| Re-read agent — `last_active_at` updated        | `GET /v1/agents/:id`                  | — → 200                           | `last_active_at` non-null and ≥ execution `started_at`                                   |
| 1.13| Create deliverable                              | `POST /v1/deliverables`               | — → 201, version: 1               | `{task_id, agent_id, title, content}`; 1 MiB content cap; backstop is `MaxBytesReader`  |
| 1.14| Update deliverable (new version)                | `PUT /v1/deliverables/:id`            | version 1 → 200, version: 2       | `{title, content}`; writes a new `deliverable_versions` row, bumps the head version      |
| 1.15| List deliverable versions                       | `GET /v1/deliverables/:id/versions`   | — → 200, **2 rows**               | newest first; both with the new `updated_at` ≥ `created_at`                              |

**State machine (B-001, 6 states):** `queued → assigned → running → review → (completed | failed)`. T1 exercises the happy branch only.

---

## 2. Cross-tenant negative tests (F-013/14/15/16 + F-D002-004 replay)

The path-implied fix (TASK-419..422) lives in the service layer:
`callerProjectID` is checked against `resource.ProjectID` on every read/write.
These tests verify that the wire-level behaviour is a 404 / empty list, NOT a
data leak.

The setup for each: create projects **A** and **B** (each with a user), an
agent in A, a task in A. The "attacker" is the user from B.

| #   | Scenario                                                                                | Endpoint / call                                   | Expected                                | Replays     |
|-----|-----------------------------------------------------------------------------------------|---------------------------------------------------|-----------------------------------------|-------------|
| CT1 | Attacker (B) reads agent in A by ID                                                     | `GET /v1/agents/:aAgentID` with `X-Project-ID: B` | 404 `CROSS_TENANT_BLOCKED`              | F-013       |
| CT2 | Attacker (B) updates agent in A (replace capabilities)                                  | `PUT /v1/agents/:aAgentID` with `X-Project-ID: B` | 404 `CROSS_TENANT_BLOCKED`              | F-014       |
| CT3 | Attacker (B) assigns a task in A to an agent in B                                       | `POST /v1/tasks/:aTaskID/assign` with `X-Project-ID: B` and `agent_id: <agentInB>` | 404 (task not in B's namespace)         | F-015       |
| CT4 | Attacker (B) creates an execution for a task in A                                      | `POST /v1/executions` with `X-Project-ID: B` and `task_id: <aTaskID>`           | 404 `CROSS_TENANT_BLOCKED`              | F-D002-004  |
| CT5 | Attacker (B) creates a deliverable for a task in A from an agent in A                  | `POST /v1/deliverables` with `X-Project-ID: B` and `task_id: <aTaskID>`        | 404 `CROSS_TENANT_BLOCKED`              | F-016       |
| CT6 | Attacker (B) lists executions in A                                                      | `GET /v1/executions?task_id=<aTaskID>` with `X-Project-ID: B`                  | 200, `data: []` (empty — B sees nothing of A) | F-013 (list variant) |

**Note on F-D002-004 (Sprint 6+):** the `POST /v1/agents` CREATE surface still
trusts the `X-Project-ID` header. An attacker authenticated in project X can
spoof `X-Project-ID: Y` and create an agent in Y. The fix is
`project_memberships` + `requireProjectMember` middleware. **Out of scope for
D-003** — this spec does not include a CT test for the create surface because
the fix is not yet landed. When the membership table is added, CT7 will be
filed.

---

## 3. What this spec does NOT cover

- **T2 (execution runtime, B-001)**: 11 sub-cases for the Aion MockRuntime
  state machine are already covered in `integration_test.go:TestIntegration_ExecutionRuntime_B001`
  and are **not duplicated** in D-003. D-003's T1 calls the runtime in steps
  1.9–1.11 as a side-effect; the T2 unit tests cover the runtime's contract.
- **T3 (B-002/B-003 cross-tenant webhook + retry)**: deferred to a future
  Sprint per the pre-scope. The webhook SSRF fix (F-D002-001) lands first.
- **Auth/login flow**: the integration tests use the middleware to inject
  a `user_id` directly. The `POST /v1/auth/login` and refresh-rotation paths
  are unit-tested in `handler/auth_test.go` (F-D002-015) and integration-tested
  by the D-002 review's static analysis only. A Sprint 7 follow-up would
  add `TestIntegration_LoginRefreshLogout` if requested.
- **Rate limiting / quota**: out of scope (per pre-scope).

---

## 4. Test file layout

```
src/internal/integration/
├── integration_test.go            # existing: 4-step smoke + 11 T2 sub-cases
├── workflow_t1_test.go            # NEW (D-003): T1 15-step happy path
└── workflow_cross_tenant_test.go  # NEW (D-003): CT1..CT6 cross-tenant blocks
```

Both new files use the same helper pattern as the existing
`integration_test.go` (real `httptest.Server`, real `router.New`, real
`store.NewMemoryStore`, real `aion.NewMockRuntime`). The shared
`doJSON` and `decodeData` helpers are duplicated into each file because
Go's `_test` packages cannot share unexported helpers across files
without a `helper_test.go` shim. A future refactor could extract
`workflow_helpers_test.go` — out of scope for D-003.

---

## 5. Audit verification checklist

The D-003 audit (`docs/reset/audit/D-003-audit-2026-06-14.md`) will confirm
each row above by reading the corresponding test file and verifying:

- [ ] Each T1 step is a `t.Run` subtest under
      `TestIntegration_Workflow_T1_HappyPath`.
- [ ] Each cross-tenant case is a `t.Run` subtest under
      `TestIntegration_Workflow_CrossTenant_Blocks`.
- [ ] Subtests run in sequence (T1) and have shared setup (projects A+B,
      user, agent, task).
- [ ] All asserts use `require` (not `assert`) for the must-pass checks.
- [ ] `go test -race -shuffle=on ./internal/integration/...` is green on
      the first CI run.
- [ ] No skips (`t.Skip`) except for documented environment gates.

---

## 6. Sign-off

This spec is the contract Guardian will verify against. Any drift between
this spec and the code will be filed as a `D-003-NN` finding in the audit
doc and routed to Builder for Sprint 6+ resolution.
