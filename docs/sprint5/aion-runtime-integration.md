# Aion Agent Runtime Integration (TASK-501)

> Sprint 5 · Developer-01 · branch `feat/sprint-5-task-501-aion-runtime`
> Status: implementation complete; pre-push gate pending; not yet committed.

This document is the canonical reference for the Aion Agent Runtime integration.
It covers the architecture, the JSON-over-stdio protocol envelope, the
dual-mode runtime, the configuration knobs, and the deliberate "in-memory
only" scope cut for Sprint 5.

## 1. Why this exists

Sprint 4's `execution.go` shipped with a hand-rolled `mockExecution`
goroutine that:

- Slept for `cfg.MockSleep()` (default 0)
- Generated a random failure based on `cfg.MockFailureRate`
- Used a `time.Timer` to wait out the delay and then transitioned the row

This was a placeholder. Production needs a real runtime that can:

- Spawn a subprocess that runs an actual Aion worker
- Receive streaming progress + final result from that subprocess over a
  stable protocol
- Survive process crashes, network glitches, and OS-level signal handling

Sprint 5 (TASK-501) replaces the mock goroutine with a small `aion.Runtime`
interface that has two implementations:

- `aion.MockRuntime` (in-process, test/dev mode)
- `aion.ProcessRuntime` (subprocess, production)

The execution service calls `Runtime.Spawn` once per `CreateExecution` and
records the spawned worker in a new `model.Worker` row. A driver goroutine
calls `Runtime.Wait` to block until the worker reports a terminal status,
then drives the existing `ExecutionService.TransitionTo` state machine to
`completed` or `failed`.

## 2. Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  HTTP handler (POST /v1/executions)                          │
└────────────────────────┬────────────────────────────────────┘
                         │ callerProjectID (X-Project-ID)
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  ExecutionService.CreateExecution(ctx, taskID, agentID, pid) │
│                                                              │
│  1. validate cross-tenant (TASK-422, F-016)                  │
│  2. create model.Execution row (status=pending)              │
│  3. if runtime != nil:                                       │
│       a. agent := store.Agents().GetByID(agentID)            │
│       b. spec := aion.WorkerSpec{ExecutionID, TaskID, ...}   │
│       c. handle := runtime.Spawn(ctx, spec)                 │
│       d. exec.AionAgentInstanceID = &newUUID                 │
│       e. store.Executions().UpdateStatus(...)                │
│       f. worker := model.Worker{...} + store.Create(...)     │
│       g. go driveWorker(worker.ID, handle, execID, pid)     │
│  4. return exec                                              │
└────────────────────────┬────────────────────────────────────┘
                         │ WorkerHandle (opaque)
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  aion.Runtime (interface)                                    │
│   ├─ aion.MockRuntime     — in-process, FakeScript-driven   │
│   └─ aion.ProcessRuntime  — os/exec subprocess, JSON stdio   │
│                                                              │
│  Spawn(ctx, WorkerSpec) (WorkerHandle, error)                │
│  Wait(ctx, WorkerHandle)  (WorkerResult, error)              │
│  Cancel(ctx, WorkerHandle) error                             │
│  Close() error                                               │
└────────────────────────┬────────────────────────────────────┘
                         │ JSON message frames on stdout
                         │ {"Type":"started",...} etc.
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  driveWorker goroutine                                       │
│                                                              │
│  1. worker.Status = running; store.Workers().Update(...)    │
│  2. result, err := runtime.Wait(waitCtx, handle)             │
│  3. translate WorkerResult → ExecutionStatus + errMsg        │
│  4. worker.Status = terminal; store.Workers().Update(...)    │
│  5. ExecutionService.TransitionTo(ctx, execID, target, ...)  │
└────────────────────────┬────────────────────────────────────┘
                         │ events.Event
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  events.MemoryBus → TASK-505 (Deliverable Capture)          │
│                     TASK-506 (Monitoring Dashboard)         │
└─────────────────────────────────────────────────────────────┘
```

## 3. The `aion.Runtime` interface

```go
// internal/aion/runtime.go
type Runtime interface {
    Spawn(ctx context.Context, spec WorkerSpec) (WorkerHandle, error)
    Wait(ctx context.Context, h WorkerHandle) (WorkerResult, error)
    Cancel(ctx context.Context, h WorkerHandle) error
    Close() error
}

type WorkerSpec struct {
    ExecutionID    uuid.UUID
    TaskID         uuid.UUID
    AgentID        uuid.UUID
    ProjectID      uuid.UUID
    Model          string
    Provider       string
    PermissionMode string
    Attempt        int
}

type WorkerHandle interface {
    String() string
}

type WorkerResult struct {
    Status       WorkerStatus // pending|running|completed|failed|cancelled
    Result       json.RawMessage
    ErrorMessage string
    Progress     []ProgressEvent // optional, currently unused in Sprint 5
}

type WorkerStatus string

const (
    WorkerPending   WorkerStatus = "pending"
    WorkerRunning   WorkerStatus = "running"
    WorkerCompleted WorkerStatus = "completed"
    WorkerFailed    WorkerStatus = "failed"
    WorkerCancelled WorkerStatus = "cancelled"
)
```

### 3.1 `model.Agent.Runtime` shape

When the execution service builds a `WorkerSpec`, it reads the model
identifier from `model.Agent.Runtime` (a `json.RawMessage` field added in
TASK-501). Expected shape:

```json
{
  "model": "sonnet",
  "provider": "anthropic",
  "permission_mode": "default",
  "extra": {
    "max_tokens": 8000,
    "temperature": 0.2
  }
}
```

The execution service parses `"model"` (or nested `"runtime.model"`) and
falls back to `"sonnet"` when nothing is set. The full `Runtime` blob is
forwarded to the subprocess via `--runtime` as a JSON-encoded flag in a
follow-up (Sprint 6+); Sprint 5 only threads the model identifier and
hardcodes `provider="anthropic"` + `permission_mode="default"`.

## 4. JSON-over-stdio envelope (subprocess mode)

The `aion.ProcessRuntime` (subprocess mode) speaks a stable JSON-line
protocol over the child's stdout. The schema is intentionally minimal and
is designed to be forward-compatible:

```jsonc
// On spawn, the runtime writes ONE initial frame to stdout:
{
  "type": "started",
  "execution_id": "uuid",
  "pid": 12345,
  "at": "RFC3339"
}

// Zero or more progress frames (TASK-506 will consume these):
{
  "type": "progress",
  "execution_id": "uuid",
  "step": 3,
  "total": 10,
  "message": "writing tests",
  "at": "RFC3339"
}

// Exactly one terminal frame:
{
  "type": "result",
  "execution_id": "uuid",
  "result": { ... },          // any JSON, forwarded to model.Worker.Result
  "at": "RFC3339"
}
```

Failure and cancellation use a sibling `"type": "error"` / `"cancelled"`
frame with an `error` field:

```jsonc
{
  "type": "error",
  "execution_id": "uuid",
  "error": "human-readable message",
  "at": "RFC3339"
}
```

Stderr is drained in a separate goroutine and logged at WARN level; it is
**not** part of the protocol. Lines on stderr are treated as operator
diagnostics and never block the protocol.

The full protocol spec lives at `docs/sprint5/aion-protocol.md` (TODO:
flesh out in a follow-up; the stub is in `aion/protocol.go` and the parser
is in `aion/process.go:pumpStdout`).

## 5. Dual-mode configuration

The `aion` package is intentionally a single interface with two
implementations. Selection happens at construction time, not at config time
— the agent-runtime layer picks one based on `AGENT_RUNTIME` env var
(or, in tests, explicit constructor calls).

| Mode           | Constructor                       | Used by                 | Spawn semantics             |
|----------------|-----------------------------------|-------------------------|-----------------------------|
| `mock` (in-process) | `aion.NewMockRuntime()`     | all tests, dev mode     | goroutine + FakeScript      |
| `process` (subprocess) | `aion.NewProcessRuntime(cfg)` | production           | `os/exec` + JSON-over-stdio |

`aion.NewMockRuntime()` is the default in `service.go` because it never
fails on missing binaries and is what the Sprint 4 test infrastructure
already exercises. `aion.NewProcessRuntime(ProcessRuntimeConfig{Binary: cfg.AionBinary, WaitTimeout: ...})` is wired into `service.go` for production. The `AION_E2E=1` env var (set by `tests/e2e/`) is what flips the `service.go` constructor from `MockRuntime` to `ProcessRuntime` in CI.

### 5.1 Env-var table

| Env var                | Default       | Used by            | Effect                                       |
|------------------------|---------------|--------------------|----------------------------------------------|
| `AION_BINARY`          | `aion`        | ProcessRuntime     | Path or PATH-name of the `aion` CLI           |
| `AION_MODEL`           | `sonnet`      | spec.Model         | Default model when agent has no override     |
| `AION_PROVIDER`        | `anthropic`   | spec.Provider      | Default provider when agent has no override  |
| `AION_PERMISSION_MODE` | `default`     | spec.PermissionMode| Default mode when agent has no override      |
| `AION_MAX_CONCURRENT`  | `8`           | (Sprint 6+ DispatchQueue) | Max in-flight workers              |
| `AION_WAIT_TIMEOUT`    | `600`         | ProcessRuntime     | Per-worker timeout in seconds                |
| `AION_E2E`             | unset         | service.go         | When set, wire `ProcessRuntime` not `MockRuntime` |

`cfg.Agent.AionBinary` is the Go-side mirror of `AION_BINARY`; same for the
other five knobs. The execution service does **not** read these env vars
directly — it gets them from `cfg.Agent` populated at startup.

## 6. In-memory only for Sprint 5

`WorkerStore` is a real interface on the `store.Store` parent, but the
postgres implementation is a `fallback.Workers()` delegate for Sprint 5:

```go
// internal/store/postgres/store.go
func (s *postgresStore) Workers() store.WorkerStore { return s.fallback.Workers() }
```

The `fallback` is the in-memory store wired alongside the postgres
connection in `NewPostgresStore`. The reason is that workers are
short-lived (typically < 10 minutes) and are not queryable by the user —
the only consumer is the runtime itself + the upcoming TASK-506 monitoring
dashboard, which reads from the events bus. Persisting workers across
restarts is not needed for Sprint 5.

**Sprint 6 follow-up**: add a `workers` table with the same shape as the
in-memory `model.Worker`, a `postgresWorkerStore` that mirrors the
`memoryWorkerStore` semantics, and a denormalised `(project_id, agent_id)`
index for the monitoring dashboard. The `store.Store` interface and the
`model.Worker` struct are already in their final Sprint 6+ shape, so this
will be additive only.

## 7. State machine mapping

`driveWorker` translates the worker's terminal status to the execution's
state machine status:

| `WorkerResult.Status` | `ExecutionStatus`         | `ErrorMessage`             |
|-----------------------|---------------------------|----------------------------|
| `completed`           | `ExecutionStatusCompleted`| (nil)                      |
| `failed`              | `ExecutionStatusFailed`   | `result.ErrorMessage`      |
| `cancelled`           | `ExecutionStatusFailed`   | `"worker cancelled: " + ...`|
| (other / error)       | `ExecutionStatusFailed`   | driver error string        |

The `WorkerCompleted` → `ExecutionStatusCompleted` path skips the
existing pending → running → completed transition chain; `driveWorker`
goes straight to the terminal status. The reason: the worker's internal
state machine (pending → running → completed) is its own concern, and
the execution's state machine only cares about the terminal status.
`TASK-503` has an `assigned` and `review` state that the runtime
*can* drive to (via the upcoming dispatch queue) but the simple
"spawn, wait, transition" flow used here only needs the terminal step.

## 8. New model fields

### 8.1 `model.Execution.AionAgentInstanceID` (new in TASK-501)

```go
type Execution struct {
    // ... existing fields ...
    AionAgentInstanceID *uuid.UUID // TASK-501; nil for legacy paths
    // ... existing fields ...
}
```

`nil` for executions created via the legacy mock-goroutine path
(`s.runtime == nil`); non-nil for executions spawned by `aion.Runtime`.
Handy for correlating the row with the child process described by the
corresponding `model.Worker` (Worker.PID is derived from this for
process-mode runtimes).

Migration: `db/migrations/025_add_executions_aion_instance_id.sql`
(adding `aion_agent_instance_id UUID NULL`).

### 8.2 `model.Worker` (new in TASK-501)

```go
type Worker struct {
    ID          uuid.UUID
    ExecutionID uuid.UUID
    AgentID     uuid.UUID
    ProjectID   uuid.UUID
    Handle      string   // opaque, runtime-specific
    PID         *int     // nil for in-process runtimes
    Status      WorkerStatus
    Attempt     int
    StartedAt   *time.Time
    CompletedAt *time.Time
    Result      json.RawMessage
    ErrorMessage string
    AionInstanceID *uuid.UUID // mirror of Execution.AionAgentInstanceID
}
```

`Handle` is the runtime-specific opaque handle (e.g. `"mock-1-..."` for
`MockRuntime`, `"proc-12345-..."` for `ProcessRuntime`). `PID` is `*int`
so the in-process runtime can leave it nil; the `ProcessRuntime` populates
it from `cmd.Process.Pid`. `AionInstanceID` mirrors the value on
`model.Execution` for O(1) lookup-by-instance-ID (the upcoming TASK-506
monitoring dashboard uses this).

### 8.3 `model.Agent.Runtime` (new in TASK-501)

```go
type Agent struct {
    // ... existing fields ...
    Metadata json.RawMessage // user-set, free-form
    Runtime  json.RawMessage // TASK-501: Aion runtime config for this agent
    // ... existing fields ...
}
```

Distinct from `Metadata` so a user's `"model": "claude-opus-4"` metadata
entry doesn't leak into the Aion spec. Migration:
`db/migrations/026_add_agents_runtime.sql` (adding `runtime JSONB NULL`).

## 9. Files changed

| File                                                       | Change                                                              |
|------------------------------------------------------------|---------------------------------------------------------------------|
| `src/internal/aion/runtime.go`                             | NEW. Runtime interface, types, errors, validation.                  |
| `src/internal/aion/mock.go`                                | NEW. In-process runtime. SetDefaultScript added in TASK-501.        |
| `src/internal/aion/process.go`                             | NEW. Subprocess runtime + JSON-over-stdio pump.                     |
| `src/internal/aion/runtime_test.go`                        | NEW. 9 test functions, ~400 lines.                                  |
| `src/internal/model/worker.go`                             | NEW. Worker entity + WorkerStatus + IsValidWorkerStatus.             |
| `src/internal/model/execution.go`                          | MODIFIED. `AionAgentInstanceID *uuid.UUID` field added.             |
| `src/internal/model/agent.go`                              | MODIFIED. `Runtime json.RawMessage` field added.                    |
| `src/internal/store/worker_store.go`                       | NEW. WorkerStore interface.                                         |
| `src/internal/store/store.go`                              | MODIFIED. `Workers() WorkerStore` added to Store interface.          |
| `src/internal/store/memory.go`                             | MODIFIED. `memoryWorkerStore` sub-store + `Workers()` accessor.     |
| `src/internal/store/postgres/store.go`                     | MODIFIED. `Workers() WorkerStore` delegates to fallback.            |
| `src/internal/store/postgres/execution_store.go`           | MODIFIED. `aion_agent_instance_id` threaded through INSERT/SELECT/scan. |
| `src/internal/store/postgres/agent_store.go`               | MODIFIED. `runtime` column threaded through INSERT/UPDATE/SELECT/scan. |
| `src/internal/config/config.go`                            | MODIFIED. 6 new Aion fields on AgentConfig + env defaults.          |
| `src/internal/service/execution.go`                        | MODIFIED. `runtime aion.Runtime` field + 5-arg ctor + driveWorker + 2 helpers + 1 ptr helper. |
| `src/internal/service/execution_test.go`                   | MODIFIED. 2 legacy MockGoroutine tests updated to SetDefaultScript.  |
| `src/internal/service/execution_503_test.go`               | MODIFIED. 7 `NewExecutionService` call sites pass `aion.NewMockRuntime()`. |
| `src/internal/service/service.go`                          | MODIFIED. `aion.NewProcessRuntime(...)` wired + aion import.        |
| `src/internal/handler/execution_test.go`                   | MODIFIED. 1 call site + aion import.                                |
| `src/internal/integration/integration_test.go`             | MODIFIED. 1 call site + aion import.                                |
| `src/db/migrations/025_add_executions_aion_instance_id.sql` | NEW. ADD COLUMN aion_agent_instance_id UUID NULL.                   |
| `src/db/migrations/026_add_agents_runtime.sql`             | NEW. ADD COLUMN runtime JSONB NULL.                                 |
| `docs/sprint5/aion-runtime-integration.md`                 | NEW. This file.                                                     |

## 10. Pre-push gate checklist

- [x] `aion.Runtime` interface — all 4 methods implemented by both runtimes
- [x] `WorkerStore` interface — implemented in memory; postgres delegates to fallback
- [x] `model.Worker` and `model.Execution.AionAgentInstanceID` added
- [x] `model.Agent.Runtime` added (separate from Metadata)
- [x] Migrations 025 + 026 additive; no destructive schema changes
- [x] Config env-var table (AION_BINARY / AION_MODEL / etc.) wired
- [x] `service.go` wires `aion.NewProcessRuntime` for production
- [x] All `NewExecutionService` call sites updated to pass a `aion.Runtime` arg
- [x] Legacy MockGoroutine tests updated to use `SetDefaultScript`
- [x] Sprint 5 cross-tenant F-016 invariants preserved (callerProjectID threaded)
- [x] No imports of `internal/aion` from `internal/model` (deliberate, documented in worker.go)
- [ ] `go build ./...` passes (canonical: CI; no-Go-on-Windows host)
- [ ] `go test ./...` passes (canonical: CI; no-Go-on-Windows host)
- [ ] `gofmt -d` is clean
- [ ] `golangci-lint run` is clean
- [ ] `make run` boots and `/v1/executions` round-trip works

## 11. Open questions for the Lead

1. **Where does the `aion` CLI live for the subprocess mode?** I assumed
   `aion` on `$PATH` (or `AION_BINARY` env var). Production will need a
   real binary. Sprint 5 ships the integration without the binary; CI's
   e2e step is the only path that exercises `ProcessRuntime`.
2. **Should `model.Agent.Runtime` be a hard requirement on CreateAgent
   in Sprint 5, or is empty (no override) acceptable?** Current code
   treats empty as "use server defaults". TASK-507 will need to
   resolve this.
3. **Should `model.Execution.AionAgentInstanceID` be exposed on the
   HTTP response?** Today the JSON tag is `omitempty`; clients can't
   see nil values. The TASK-505 deliverable-capture consumer needs
   it to correlate. Default: yes, expose.

## 12. References

- Lead's brief: `docs/sprint5/brief.md` §5 (TASK-501)
- Integration test plan: `docs/sprint5/integration-test-plan.md` §2
  (dual-mode runtime), §4 (JSON-over-stdio envelope)
- Security review: cross-tenant F-016 invariants preserved (TASK-422
  callerProjectID threading continues to apply)
- Sprint 4 closeout: TASK-405 mock-goroutine in `execution.go:290-335`
  (the body that's now replaced by the runtime-driven path)
