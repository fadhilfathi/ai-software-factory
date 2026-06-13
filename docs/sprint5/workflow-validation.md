# Sprint 5 — Workflow Validation (test matrix)

**Owner:** Tester-01
**Source brief:** `docs/sprint5/brief.md` §6.9
**Linked harness:** `docs/sprint5/integration-test-plan.md` (TASK-510, this agent)
**Date:** 2026-06-13
**Status:** design draft (Wave 1-3 not yet started; matrix is the contract the harness will satisfy)

---

## 1. Purpose

Document the test matrix for the Sprint 5 real agent execution engine across the 5 workflow stages:

1. Agent creation
2. Task assignment
3. Task execution
4. Deliverable generation
5. Execution recovery

Each stage has a **manual scenario set** (UI-driven, run by a human) and an **automated scenario set** (API/integration-driven, run by Go tests in CI). The matrix is the canonical reference for:

- **What TASK-510 (Integration Testing harness) must automate** — every `A*.x` row maps to a Go test sub-test
- **What a human validates manually** — every `M*.x` row is a documented procedure, run before Sprint 5 closeout
- **What cross-stage E2E scenarios (E1..E5) must pass** — the "smoke" set that exercises multiple stages in one pass

## 2. Test layering (L0..L4)

| Layer | Scope | Owner | Sprint 5 task | Catches |
|---|---|---|---|---|
| **L0** | Unit tests (per service, per handler) | Dev-01/02/03/04 | per task (TASK-501..508) | Logic bugs in isolation |
| **L1** | Integration tests (cross-route, in-memory store) | Tester-01 | **TASK-510** | Cross-service wiring, state-machine transitions |
| **L2** | Workflow validation (end-to-end user-flow) | Tester-01 | **TASK-509** (this doc) | UX, dashboard, round-trip bugs |
| **L3** | Security review (auth, authz, XSS, injection) | Security-01 | TASK-511 | Auth/authz bugs |
| **L4** | Infrastructure validation (Docker, Aion subprocess, env) | DevOps-02 | TASK-512 | Env / spawn / process bugs |

**Layering rule:** a bug caught at L0 is cheaper to fix than at L4. The matrix is structured so that each layer has unique coverage — no overlap, no gaps. Where two layers could cover the same case (e.g., A1.1 + L0 unit test for `AgentService.Create`), the L0 test stays in the dev lane and the L1 test focuses on the cross-route wiring (handler → service → store), not the service's internal logic.

## 3. Stage 1 — Agent creation

| ID | Type | Pre | Action | Expected | Layer | Owner |
|----|------|-----|--------|----------|-------|-------|
| M1.1 | manual | empty store | Open `/agents`, click "+ New Agent", fill developer form (name, role=developer, capabilities=[coding, testing]), submit | Agent card appears in list; status badge = `available`; capability chips render | L2 | Dev-02 (UI build) + Tester-01 (matrix run) |
| M1.2 | manual | M1.1 done | Open `/agents`, create reviewer agent (name, role=reviewer, capabilities=[review]) | Reviewer card appears; capability chips = [review] | L2 | Dev-02 + Tester-01 |
| M1.3 | manual | M1.1, M1.2 done | Try to create agent with empty name | Form rejects submit; inline error "name required"; no row in store | L2 | Dev-02 + Tester-01 |
| A1.1 | automated | empty store | `POST /v1/agents` with valid body (project_id, name, role, capabilities) | 201; `data.id` is UUID; store has 1 row | L1 | Tester-01 (TASK-510) |
| A1.2 | automated | A1.1 done | `POST /v1/agents` with duplicate name in same project | 409; error.code = `DUPLICATE_AGENT_NAME` (Sprint 5 added `(project_id, name)` uniqueness — Lead Q1; new constant `model.AgentErrorDuplicate` + handler-side mapping; 1 unit + 1 integration test; filed under TASK-501 if not already in scope) | L1 | Tester-01 |
| A1.3 | automated | empty store | `POST /v1/agents` with `capabilities: ["c0d!ng"]` (malformed token) | 400; error.code = `VALIDATION_ERROR` | L1 | Tester-01 |
| A1.4 | automated | empty store | `POST /v1/agents` from a different project's caller (wrong `X-Project-ID`) | 404; error.code = `CROSS_TENANT_BLOCKED` (per TASK-419 fix) | L1 | Tester-01 |

## 4. Stage 2 — Task assignment

| ID | Type | Pre | Action | Expected | Layer | Owner |
|----|------|-----|--------|----------|-------|-------|
| M2.1 | manual | M1.1, M1.2 done | From `/tasks/:id/assign`, pick the developer agent, confirm | Assignment row created; status badge = `assigned`; history list shows 1 row (action=assign) | L2 | Dev-02 + Tester-01 |
| M2.2 | manual | M2.1 done | Re-assign to the reviewer agent (different agent type) | Ownership transfers; history list shows 2 rows (assign, reassign); task card shows reviewer as owner | L2 | Dev-02 + Tester-01 |
| M2.3 | manual | M2.2 done | Unassign via the UI's "Unassign" button | History list shows 3 rows (assign, reassign, unassign); task card shows "no owner" | L2 | Dev-02 + Tester-01 |
| A2.1 | automated | A1.1, A1.2 done | `POST /v1/tasks/:id/assign` with valid agent + matching `capabilities_required` | 200; assignment.status = `active`; `idempotent: false`; 1 history event | L1 | Tester-01 |
| A2.2 | automated | A1.1 done | `POST /v1/tasks/:id/assign` with mismatched capabilities (agent has [coding], request requires [security]) | 409; error.code = `CAPABILITY_MISMATCH` (or whatever TASK-502 settles on) | L1 | Tester-01 |
| A2.3 | automated | A2.1 done | Re-`POST /v1/tasks/:id/assign` with same agent_id | 200; same assignment ID; `idempotent: true`; history still has 1 event (no new row) | L1 | Tester-01 |
| A2.4 | automated | A1.1 done (project A) | From project B, `POST /v1/tasks/:id/assign` (cross-tenant) | 404; error.code = `CROSS_TENANT_BLOCKED` (per TASK-420 fix) | L1 | Tester-01 |
| A2.5 | automated | A2.1 done | `POST /v1/tasks/:id/assign` with `notes: "manual run"` | history event row has `notes="manual run"` (F-017 fix verified) | L1 | Tester-01 |

## 5. Stage 3 — Task execution

| ID | Type | Pre | Action | Expected | Layer | Owner |
|----|------|-----|--------|----------|-------|-------|
| M3.1 | manual | M2.1 done (task assigned) | Open `/executions`, watch the row appear; click into the row | Live state transitions appear (or refresh shows progress); final state = `COMPLETED` | L2 | Dev-02 + Tester-01 |
| M3.2 | manual | M3.1 done | Open `/executions/:id`, scroll to the event log | Event log shows 5+ rows in order: QUEUED, ASSIGNED, RUNNING, REVIEW, COMPLETED, with timestamps | L2 | Dev-02 + Tester-01 |
| A3.1 | automated | A2.1 done | Subscribe to state events via `Watch(executionID)`; trigger execution | Within 5s, channel receives events in order; final state = `COMPLETED`; channel closes | L1 | Tester-01 (TASK-510, wait-without-sleep pattern) |
| A3.2 | automated | A3.1 done | Each event has a monotonic `At` timestamp; no out-of-order arrivals | assert monotonic; 0 failures | L1 | Tester-01 |
| A3.3 | automated | A3.1 done | After execution completes, attempt `PATCH /v1/executions/:id` with `{status: "running"}` (illegal transition) | 409; error.code = `ILLEGAL_TRANSITION` (per TASK-503 state machine) | L1 | Tester-01 |
| A3.4 | automated | A3.1 done | Cross-tenant: from project B, `GET /v1/executions/:id` | 404; error.code = `CROSS_TENANT_BLOCKED` (per TASK-422 fix) | L1 | Tester-01 |

## 6. Stage 4 — Deliverable generation

| ID | Type | Pre | Action | Expected | Layer | Owner |
|----|------|-----|--------|----------|-------|-------|
| M4.1 | manual | A3.1 done (execution completed) | Open `/deliverables/:id` | Rendered markdown view; sanitized; links work | L2 | Dev-02 + Tester-01 |
| M4.2 | manual | M4.1 done; agent emitted a second version | Open `/deliverables/:id/versions` | List of versions; side-by-side diff shows the changes | L2 | Dev-02 + Tester-01 |
| A4.1 | automated | A3.1 done | Verify deliverable auto-captured (TASK-505) | 1 version row; `title`, `content` match the agent's emit | L1 | Tester-01 |
| A4.2 | automated | A4.1 done | Agent emits a second markdown update | 2 version rows; v1 preserved (append-only per TASK-406); diff exists | L1 | Tester-01 |
| A4.3 | automated | A1.1 done | `PUT /v1/deliverables/:id` with 10 MiB content body | 413 (F-023 fix verified per TASK-424) | L1 | Tester-01 |
| A4.4 | automated | A4.1 done | Deliverable content contains `<script>alert(1)</script>` | Stored as raw markdown; rendered view strips the script (no XSS — F-006 per TASK-409) | L1 | Tester-01 + Security-01 (TASK-511 cross-check) |

## 7. Stage 5 — Execution recovery

| ID | Type | Pre | Action | Expected | Layer | Owner |
|----|------|-----|--------|----------|-------|-------|
| M5.1 | manual | A1.1, A1.2 done; fake script makes first attempt fail | Open `/executions/:id` | Event log shows RUNNING → FAILED → retry → QUEUED → ... → COMPLETED (per TASK-508 retry policy) | L2 | Dev-02 + Tester-01 |
| M5.2 | manual | M5.1 done | Verify retry used a different worker (PID or worker_id differs) | Worker identity in event log changes between attempt 1 and attempt 2 | L2 | Dev-02 + Tester-01 |
| A5.1 | automated | A1.1 done; fake script = `[{Send: progress}, {Exit: 1}]` | Trigger execution | 2 execution rows in store; final state = `COMPLETED`; state events show retry path | L1 | Tester-01 (TASK-510) |
| A5.2 | automated | A5.1 setup; fake script fails both attempts | Trigger execution | 1 execution row; final state = `FAILED`; total budget = 2 (1 retry + 1 alt escalation); jittered backoff 100-500ms; env vars `RECOVERY_TOTAL_BUDGET=2`, `RECOVERY_BACKOFF_MIN_MS=100`, `RECOVERY_BACKOFF_MAX_MS=500`; heartbeat uses `last_progress_at` not just liveness — Lead Q3 + decisions file line 67; Dev-02 (TASK-508 owner) may revise | L1 | Tester-01 |
| A5.3 | automated | Developer agent has [coding] only; task requires [coding, security] | Assign; verify capability mismatch on first attempt; verify recovery reuses an existing agent with [security] | Existing agent with the missing capability is **reused** (NOT a new `agents` row); new assignment event in history. If no existing agent has [security] → state FAILED with structured error `ErrNoAlternativeAgent` (Lead Q2; final call deferred to Dev-02 / TASK-508) | L1 | Tester-01 (TASK-510) |
| A5.4 | automated | A5.1 done | All retry events delivered through the same `Watch` channel (no separate event bus) | Subscriber receives the FAILED event followed by retry events; channel closes on terminal | L1 | Tester-01 |

## 8. Cross-stage E2E scenarios (the "smoke" set)

These 5 scenarios are the canonical manual + automated smoke tests for Sprint 5 closeout. Each one crosses multiple stages in a single pass.

| ID | Scenarios covered | Action summary | Automation target |
|----|-------------------|----------------|-------------------|
| **E1** | M1.1+M1.2, M2.1, M3.1+M3.2, M4.1 | Happy path: create developer + reviewer, assign developer, watch execution, verify deliverable, then reviewer takes the next task (re-assign) | `TestE2E_HappyPath` (L1, automated) + manual run-through (L2) |
| **E2** | M5.1, M5.2 | Retry path: developer fails first attempt; system retries with a different worker; deliverable still lands | `TestE2E_RetryPath` (L1, automated) + manual run-through (L2) |
| **E3** | (none of the manual ones; this is purely a recovery case) | Capability mismatch recovery: developer lacks security; task needs it; recovery reuses an existing security-capable agent (or fails with `ErrNoAlternativeAgent` if none — per Lead Q2) | `TestE2E_RecoveryPath` (L1, automated) + manual run-through (L2) |
| **E4** | M3.1, M3.2 | Real-time state observability: subscribe to state events; verify 5+ events arrive in order; final state = COMPLETED | `TestE2E_StateObservability` (L1, automated) + manual verification (L2) |
| **E5** | (subprocess-only; no manual equivalent in Sprint 5) | Subprocess smoke: real `aion` CLI; gated by `AION_E2E=1` env var; runs in nightly CI | `TestE2E_SubprocessSmoke` (L1, automated, gated) |

## 9. Acceptance

- **L0** (unit tests) — all green (Dev-01/02/03/04 lanes)
- **L1** (integration tests) — all `A*.x` rows + all `E*.x` rows pass in `go test -count=1 -timeout 10m ./internal/integration/...`
- **L2** (workflow validation) — all `M*.x` rows have a documented procedure; all 5 E*.x scenarios run successfully end-to-end on a fresh checkout
- **L3** (security review) — all findings from TASK-511 are either fixed (marked `FIXED-IN-PATCH`) or explicitly accepted by Lead; no Critical/High open
- **L4** (infrastructure validation) — TASK-512 confirms Aion subprocess can spawn in the `api` container; `sprint-quality-gate.yml` step 13 passes

## 10. Risks

1. **Stage 5 (recovery) depends on TASK-508 design decisions** — the retry policy and the "alternative path" for capability mismatch aren't fully specified yet. **Mitigation:** the matrix rows in §7 have a `?` next to the error.code for the capability mismatch case; will firm up once TASK-508 lands.
2. **M3.1 depends on whether `/executions` is "live" (SSE/3s polling per TASK-506)** — the manual scenario may need adjustment if Dev-02 picks SSE (event-driven) vs polling. **Mitigation:** the matrix describes the user's expectation (state transitions appear); the implementation detail is Dev-02's call.
3. **L1 layer needs the `Runtime` interface to land (TASK-501)** — if the interface signature differs from my design assumption, the harness needs a rewrite. **Mitigation:** flag in TASK-510 risks; design assumes a minimal `Runtime{Spawn, Send, Recv, Cancel}` shape.
4. **State manager's `Watch()` API is a design assumption** (TASK-503) — if it doesn't expose event channels, the wait-without-sleep pattern falls back to `assert.Eventually` polling. Slower but works.

## 11. Open questions for Lead

**Resolved (Lead answered 2026-06-13):**
1. ✅ A1.2 — duplicate name: 409 with code `DUPLICATE_AGENT_NAME`; new constant `model.AgentErrorDuplicate`; handler-side mapping; 1 unit + 1 integration test. Filed as follow-up under TASK-501 if not already in scope.
2. ✅ A5.3 — alternative agent: reuse existing agent with the missing capability; if none exists → state FAILED with `ErrNoAlternativeAgent`. Final call deferred to Dev-02 (TASK-508 owner).
3. ✅ Stage 5 retry limit: total budget = 2 (1 retry + 1 alt escalation); jittered backoff 100-500ms; env vars `RECOVERY_TOTAL_BUDGET=2` (default 2), `RECOVERY_BACKOFF_MIN_MS=100` (default 100), `RECOVERY_BACKOFF_MAX_MS=500` (default 500); heartbeat uses `last_progress_at` not just liveness (decisions file line 67). Final number deferred to Dev-02 (TASK-508 owner).

**Still open:**
4. **`/executions` live updates** — does the dashboard use SSE or polling? M3.1's manual procedure assumes "refresh shows progress"; if SSE, the manual procedure is "wait for the row to update without refresh."
5. ✅ **State machine shape** — RESOLVED (Decision 4, 2026-06-14, ~01:30 UTC dispatch): **brief is canonical**. 6 states: `QUEUED → ASSIGNED → RUNNING → REVIEW → COMPLETED/FAILED`. NOT architecture-overview.md's shape (no `pending`, no `cancelled`). Analyst-01 will update `architecture-overview.md` §3 to match. The execution state machine is **separate** from the WorkerStatus state machine (`pending | running | completed | failed | cancelled` from `model/worker.go`) — they coexist. Harness design uses the brief shape; no test-logic rewrite needed.

## 12. Cross-references

- TASK-510 harness design: `docs/sprint5/integration-test-plan.md`
- TASK-511 security findings: `docs/sprint5/security-review.md` (not yet written; Security-01's lane)
- TASK-512 infrastructure report: `docs/sprint5/infra-fixes.md` (exists from TASK-429; will be extended)
- TASK-513 CI/CD gate: `.github/workflows/sprint-quality-gate.yml` (Sprint 4 baseline; will be extended with the Aion-runtime check per brief §6.13)
- Sprint 4 baseline matrix: `docs/sprint4/test-plan.md` (24 cross-route sub-cases; all still passing)
