# C-002 Recovery System — Pre-Scope Brief

**Owner of C-002:** Builder
**Support:** Leader (this brief) + Guardian (D-002 security review)
**Spec / design doc:** *to be confirmed — search for `docs/sprint5/recovery-design.md`; if absent, design from scratch in the audit doc*
**Internal task ID:** TASK-508 (per comment refs in `model/worker.go`, `dispatch/queue.go`, `aion/process.go`, `model/execution.go`)
**Audit precedent:** `docs/reset/audit/A-001-audit.md` (12-point spec-drift fix model) + `docs/reset/audit/B-001-audit.md` (when B-001 closes; 6-state lifecycle model)

---

## Deliverables (per the team contract)
- **Retry limits** — max attempt count, backoff policy, dead-letter routing
- **Failure handling** — recover vs. fail vs. cancel decision tree on worker/agent death
- **Recovery workflow** — health probe + container cleanup + state-machine handoff

## Existing code floor (READ FIRST)

| File | Purpose | Status |
|------|---------|--------|
| `src/internal/service/orchestrator.go:70` | `HandleAgentFailure(agentID)` — **stub returning "not implemented"** | Primary seam to fill |
| `src/internal/service/orchestrator.go` | `checkAgentHealth` — currently a no-op `Debug` log | Need real health probe |
| `src/internal/model/worker.go` | `Worker.Attempt int` (1-based, "bumped by recovery layer on retry — TASK-508") | Field exists, no writer yet |
| `src/internal/model/worker.go:32` | `WorkerCancelled` status | Declared, never written |
| `src/internal/model/worker.go:38` | `WorkerStatus.IsTerminal()` | Used by monitoring (TASK-506) to filter polling — C-002 must respect this |
| `src/internal/service/execution.go:239` | `model.ExecutionStatusQueued: {}` — `assigned → queued` "operator / C-002 recovery return" edge | Edge is in the state machine, no caller |
| `src/internal/dispatch/queue.go:161` | `Retries` / `Dropped` counters (queue-level) | Comment: "TASK-508 will define the dead-letter semantics" — not yet defined |
| `src/internal/aion/process.go:301` | `_ = lastLine // reserved for TASK-506 / TASK-508 recovery context` | Line buffer reserved for recovery error context |
| `src/db/migrations/028_extend_executions_lifecycle.sql:6` | `assigned → running, failed, queued (operator/recovery return)` | Migration done at B-001 c2; C-002 fills the "recovery return" use case |
| `src/internal/agentfactory/agent_factory.go:7` | Comment: "future call sites (POST /v1/tasks/:id/execute, recovery)…" | Anticipates C-002 call site |
| `src/internal/middleware/middleware.go:172` | `gin.RecoveryWithWriter` (HTTP panic recovery) | NOT the same as task recovery — note the naming clash |

Total: ~70 LOC of stub orchestrator + 97 LOC worker model + state-machine edge ready to use. Solid floor.

---

## Gaps to verify on inspection (likely findings)

### 1. Spec drift — `docs/api-spec.md` §Executions / §Workers
- **Attempt field** — exists in code, almost certainly missing from spec. A-001 / A-002 hit this exact pattern.
- **Worker statuses** — code has 5 (pending, running, completed, failed, cancelled); spec may show 4.
- **Recovery endpoint** — is there a `POST /v1/agents/:id/recover` or `POST /v1/executions/:id/retry` in the spec? If not, design it.
- **Dead-letter queue** — the dispatch comment "TASK-508 will define the dead-letter semantics" implies the spec is silent. Builder to define.
- **6-state machine** — the new `assigned → queued` recovery edge needs a doc paragraph.

### 2. `HandleAgentFailure` stub — signature and contract
Currently:
```go
// HandleAgentFailure attempts to recover or re-assign tasks for a failed agent.
func (o *AgentOrchestrator) HandleAgentFailure(agentID string) error {
    return fmt.Errorf("not implemented")
}
```
- **What does it return on partial recovery?** (Some workers recovered, some failed — error or nil?)
- **What if the agent is healthy at call time?** (Late callback — no-op? Idempotent skip?)
- **What's the dedup key?** (AgentID risks double-fire if multiple workers fail simultaneously. WorkerID would be safer — consider `HandleWorkerFailure(workerID)` as the primitive, with `HandleAgentFailure` as the fan-out wrapper.)

### 3. `checkAgentHealth` — health probe protocol
Current: a `Debug` log stub. Real options:
- **Docker healthcheck** — if runtime is subprocess/Docker, use the container's healthcheck status
- **HTTP /health** — if the Aion worker exposes one (likely not in the mock runtime)
- **Last-seen heartbeat** — Worker.LastHeartbeatAt field (doesn't exist yet) + 30s timeout = dead
- **Process-alive check** — `syscall.Kill(pid, 0)` for the ProcessRuntime

Recommend: last-seen heartbeat as the canonical signal (works for both mock and process runtimes), with Docker `inspect` as a Sprint 6+ refinement.

### 4. Container cleanup on agent death
The agentfactory spawns Docker containers (`aion/process.go`). When an agent dies:
- **Kill + remove** the container (clean, but operator loses forensics)
- **Kill only** (container preserved, can `docker logs` after — better for ops)
- **Detach** (let it orphan — bad, leaks)

Recommend: kill-only with a `retained_containers` table for forensics. Sprint 6+ can add TTL.

### 5. Retry policy — config vs. hard-coded
- **Max attempts** — 3 is the industry default; 5 is more forgiving. Config-driven (`config.MaxRetryAttempts`) preferred.
- **Backoff** — exponential (1s, 2s, 4s, 8s) is standard; fixed (constant retry interval) is simpler. Config-driven.
- **Per-task override** — long-running tasks may want a higher budget. Consider `task.retry_budget` field in the model (Sprint 6+; Sprint 4+5 use global config).

### 6. Dead-letter routing
The dispatch comment explicitly leaves this open. Options:
- **In-memory ring buffer** — fast, lost on restart (consistent with Sprint 4+5 in-memory floor)
- **Persisted `dead_letter` table** — durable, postgres-only, Sprint 6+
- **Hybrid** — in-memory pointer + sprint-quality-gate snapshots to disk

Recommend: in-memory `DeadLetterStore` interface (mirroring `Store`) with a `MemoryDeadLetterStore` impl for Sprint 4+5. Operator endpoint `GET /v1/dead-letter` + `POST /v1/dead-letter/:id/replay`.

### 7. Idempotency
- `HandleAgentFailure(agentID)` — if called twice in a 5s window (e.g. health probe + crash detection both fire), do we double-recover?
- **Fix**: a `recovery_in_flight map[uuid.UUID]time.Time` with 5s TTL. Or use a per-agent mutex via the store layer.

### 8. State-machine edges
- `running → failed` (worker crashed mid-execution) — already in B-001
- `running → queued` (operator or recovery returns it to queue) — **NOT in B-001 c2; need to add**
- `review → queued` (reviewer rejected, send back) — likely needed for the C-002 deliverable
- `failed → assigned` (operator manual retry) — Sprint 6+
- `cancelled → *` (terminal, no recovery) — Sprint 6+

Recommend C-002 adds `running → queued` and `review → queued` only; the rest is Sprint 6+.

### 9. WorkerCancelled status — orphan reference
`WorkerCancelled` is declared but never written. C-002 should use it for: user-cancelled executions, recovery-cancelled orphans. Otherwise delete the constant.

### 10. Postgres persistence for `Worker` rows
Currently in-memory only (per `model/worker.go:3` comment). C-002 doesn't need to add postgres persistence (out of scope), but the `Worker` row recovery metadata (e.g. `recovery_reason`) should be JSON-typed so it survives the Sprint 6+ migration.

---

## Cross-agent handoffs (likely)

- **From A-003 (Assignment Engine):** recovery re-assignment re-uses the `AssignmentService.AssignTaskToAgent` path. F-014 triple-check (capability + project + tenant) must apply to recovery-reassignment too — DO NOT bypass.
- **From B-001 (Execution Engine):** C-002 reads `Execution.Status` to decide if recovery is needed; must respect the 6-state model. C-002 will need to add `running → queued` and `review → queued` edges.
- **From B-002 (Agent Communication):** if B-002 introduces a webhook for "agent dead" notifications, C-002 should be a webhook subscriber (not a poller). Spec this when B-002 lands.
- **From B-003 (Deliverable Storage):** dead-lettered executions may have partial deliverables. C-002 should NOT roll back the `deliverable_versions` append-only invariant — partial deliverables stay.

---

## Audit doc shape (mirror A-001 / B-001)
`docs/reset/audit/C-002-audit.md`:
- Evidence: every existing file, line range, public API surface
- Drift inventory: 8-12 items, each with `code` `spec` `fix`
- Pre-push gate: tests + build + Guardian sign-off + secret-scan
- Hand-backs: anything that crosses into D-002 (retry DoS, idempotency)

---

## Suggested PR shape (5 commits — your call to subdivide)

1. `docs(api-spec): document recovery model + Attempt field + dead-letter semantics` (docs only — mirror A-001 / B-001)
2. `feat(recovery): implement HandleAgentFailure + retry budget + dead-letter routing` (closes the orchestrator stub)
3. `feat(recovery): health probe + container cleanup on agent death` (`checkAgentHealth` no-op → real probe; container kill-only)
4. `test(recovery): table-driven coverage of retry budget, terminal transitions, dead-letter routing, idempotency`
5. `docs(audit): C-002 recovery system audit + pre-push gate`

If the spec drift is non-trivial, split commit 1 into `docs(api-spec): …` + `feat(recovery): spec→code alignment`.

---

## When this must land

C-002 is the C-track closeout deliverable. Dependencies:
- **B-001 6-state lifecycle** — DONE (c1a65df on main, 2026-06-14)
- **B-001 reviewer action + DELETE** — in progress (B-001 c3); C-002 can start in parallel, but the `running → queued` and `review → queued` state-machine edges land with B-001 c3
- **D-002 security review** — in progress; C-002 retry-DoS surface should be cross-referenced

C-002 can be developed against a feature branch (`feat/C002-recovery-system`) once B-001 c3 lands. If you want to start the docs pre-work now (commit 1 only) on a separate docs branch, that's fine.

---

## What I (Leader) will do

- Review the audit doc when ready
- Cross-ref your spec-drift items against A-001's 12 items + B-001's drift fix pattern
- Pre-stage the F-D002-004 IDOR finding for any recovery endpoint you add (it'll read `X-Project-ID` the same way as A-001 / B-001 / B-002)
- Coordinate with Guardian (D-002) for the security review of retry-DoS + idempotency

## What Guardian (D-002) will do

- Review retry-DoS surface — can a malicious task trigger runaway recovery (max attempts × backoff = forever)?
- Review idempotency of recovery — does double-call of `HandleAgentFailure` exploit anything?
- Review container-cleanup race — what if the runtime reports back during `kill` (e.g. `worker.go:7` `Result` write after we've already moved the execution to `queued`)?
- Add `service/orchestrator.go` + the new `service/recovery.go` (when it lands) to the D-002 review checklist

---

## Open questions for Builder (please answer in the audit doc)

1. Max attempts default + config key name
2. Backoff policy (exponential vs. fixed) + base interval
3. Dead-letter store shape (in-memory ring? Persisted table? Hybrid?)
4. Health probe protocol (last-seen heartbeat preferred)
5. Container cleanup policy (kill-only recommended)
6. Manual recovery endpoint — `POST /v1/agents/:id/recover`? Or `POST /v1/executions/:id/retry`? Or both?
7. Idempotency key (WorkerID recommended)
8. State-machine edges to add — `running → queued` and `review → queued` only? Or more?
9. WorkerCancelled — keep and use, or delete as dead code?
10. Recovery-reassignment path — re-use `AssignmentService.AssignTaskToAgent` (recommended) or a new `RecoveryReassign` method?
