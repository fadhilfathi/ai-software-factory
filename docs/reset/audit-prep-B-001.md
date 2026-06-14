# B-001 Execution Engine — Pre-Scope Brief

**Owner of B-001:** Builder
**Support:** Leader (this brief)
**Spec:** `docs/sprint5/agent-creation-management-design.md` §Execution Lifecycle (likely — verify the exact section)
**Audit precedent:** `docs/reset/audit/A-001-audit.md`, `docs/reset/audit-prep-A-002.md`

## Deliverables (per the team contract)
- Agent Execution Runtime
- Execution Lifecycle: **QUEUED → ASSIGNED → RUNNING → REVIEW → COMPLETED/FAILED** (6 states per the brief)

## Existing code floor (READ FIRST)

| File | Purpose | Status |
|------|---------|--------|
| `src/internal/model/execution.go` | `ExecutionStatus` enum (4 states), `Execution` row, `ExecutionFilter` for list | **GAP: enum is 4 states, spec is 6** |
| `src/internal/aion/runtime.go` | `Runtime` interface (Spawn/Wait/Cancel), `WorkerStatus` (5 states incl. cancelled) | Solid. Mock + Process runtimes in `mock.go` and `process.go`. |
| `src/internal/aion/process.go` | `ProcessRuntime` for the real `aion` CLI child process | Read. Has Windows + Unix process-management paths. |
| `src/internal/aion/mock.go` | `MockRuntime` for tests/dev | Read for in-flight test patterns. |
| `src/internal/agentfactory/agent_factory.go` | Spawn/Shutdown/Track | **Known bug (A-002-01)**: `Shutdown()` shadows struct type; syscall.Kill needs build tag. Hand-back in flight on `fix/A002-handbacks` branch. |
| `src/internal/service/execution.go` | State machine, persistence, cross-tenant checks | Read the state-transition rules. |
| `src/internal/handler/execution.go` | HTTP: list/get/update status, with `projectIDFromContext` (F-D002-004 IDOR surface — same as A-001 / A-002) | Reads. |
| `migrations/008_create_executions.sql`, `024_create_executions.sql`, `025_*.sql`, `026_add_agents_runtime.sql` | Schema | Verify the CHECK constraint covers all 6 states. |

## THE KEY GAP — lifecycle is 4 states, spec is 6

The brief asks for **QUEUED → ASSIGNED → RUNNING → REVIEW → COMPLETED/FAILED**. The current code has 4 states: `pending`, `running`, `completed`, `failed`. Two states are missing from the code:

- **`QUEUED`** — the task is in the system but no agent has been picked yet. Likely a NEW state that sits between task-creation and the first assignment. Maps to "dispatch queue depth" metric on the dashboard.
- **`REVIEW`** — the agent finished execution but the result is awaiting peer review (or a quality check) before being marked `COMPLETED`. This is a non-trivial new state because:
  - It needs to be a real status (with a CHECK constraint update)
  - The runtime emits `completed` from the worker perspective; the EXECUTION has a separate `review` state above that
  - The state machine in `service.ExecutionService` needs a new transition: `running → review → completed|failed`
  - The agent that did the work has finished; a DIFFERENT agent (or the human reviewer) transitions out of `review`
  - The dashboard needs a "in review" tab

### Suggested implementation

1. **Add two new `ExecutionStatus` constants** in `model/execution.go`:
   - `ExecutionStatusQueued = "queued"`
   - `ExecutionStatusReview = "review"`

2. **Update the migration CHECK constraint** (`008_create_executions.sql` and `024_create_executions.sql`):
   - Add `'queued'` and `'review'` to the allowed set
   - Verify the new state values propagate through the in-memory store tests

3. **Update the state machine** in `service/execution.go`:
   - `queued → assigned, failed` (assign = first agent picked; failed = e.g. capability mismatch persists past deadline)
   - `assigned → running, failed` (the agent starts; or the assignment is revoked)
   - `running → review, failed` (worker finished; or worker errored)
   - `review → completed, failed` (reviewer accepted; or reviewer rejected)
   - `completed → terminal`
   - `failed → terminal`

4. **Update the runtime → service handoff**:
   - The runtime emits `WorkerCompleted`; the service transitions the execution to `review` (not `completed` directly)
   - The runtime emits `WorkerFailed`; the service transitions the execution to `failed` (skipping `review` — failed is a separate path)

5. **Add the dashboard view** for "in review" — C-001 will own this. Pre-coordinate so the C-001 contract lines up with the new state.

6. **Tests** for the new state machine paths and the CHECK constraint.

## Smaller items to verify

- **A-002-01 hand-back in flight** — `agentfactory/agent_factory.go:Shutdown()` is broken on all OS; syscall.Kill needs build tag. This must land BEFORE the execution runtime can gracefully shut down agents in production. Builder is on it in the hand-backs branch.
- **F-D002-004 IDOR** — same X-Project-ID surface as A-001/A-002. Log as D-002 OPEN.
- **C-002 Recovery System** depends on the lifecycle. C-002 will need to handle the `queued` and `review` states too. Coordinate: B-001 ships first, C-002 follows.
- **In-memory store** is the default per `aion/runtime.go` (line 20: "Sprint 5 ships the in-memory store; a postgres-backed dispatch queue is a Sprint 6 follow-up"). Document this in the audit doc.

## Audit doc shape (mirror A-001 / A-002)
`docs/reset/audit/B-001-audit.md`:
- Evidence: every existing file, the current 4-state model, the state machine, the dispatch queue
- Drift inventory: the 6-state lifecycle gap (item 1 above), the API spec drift (similar to A-001's 12 items)
- Pre-push gate: tests + build + Guardian sign-off + secret-scan
- Hand-backs: anything that crosses into C-001 (dashboard) or C-002 (recovery)

## Suggested PR shape
- Commit 1: `docs(api-spec): fix execution lifecycle drift (6 states)` — model + spec align
- Commit 2: `feat(execution): extend lifecycle to 6 states (queued/assigned/running/review/completed/failed)` — model + CHECK constraint + state machine
- Commit 3: `feat(execution): runtime→service handoff transitions to review on worker complete` — service + runtime
- Commit 4: `test(execution): table-driven coverage for the 6-state machine` — every transition, every illegal transition
- Commit 5: `docs(audit): B-001 execution engine audit + pre-push gate`

## When this must land
After A-002 (Capability System) ships. The A-002-01..05 hand-backs in the fix/A002-handbacks branch need to land FIRST (B-001 depends on the Shutdown() fix).

## What I (Leader) will do
- Review the audit doc.
- Cross-check the 6-state lifecycle against the integration test in D-003 to make sure the workflow validation uses the right state names.
- Flag F-D002-004 in the D-002 report.

## What Guardian (D-002) will do
- Review the state machine for safe transitions (no way to skip review, no way to re-execute a terminal execution, etc.).
- Review the runtime cross-tenant boundary (`WorkerSpec` carries the project ID; runtime trusts it).
- Add `internal/aion/runtime.go` and `service/execution.go` to the D-002 review checklist.
