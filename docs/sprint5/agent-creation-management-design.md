# TASK-507: Agent Creation Management (Sprint 5)

## 0. Design decisions

- **AgentFactory** is a new package (`src/internal/agentfactory/`) that wraps `aion.Runtime` with Aion-specific defaults (TokenRouter / MiniMax-M3 / YOLO), tracks every spawned agent (PID, start time, model, provider, permission mode, role), and SIGTERMs all tracked PIDs on `Shutdown()`.
- **ExecutionService integration is deferred** to a follow-up branch. The current `service/execution.go` `createExecution` path keeps its hard-coded `Provider: "anthropic"`, `PermissionMode: "default"` defaults and `nil` PID. Future Sprint 5 work (TASK-432 POST `/v1/tasks/:id/execute` handler, TASK-508 recovery) can switch to `AgentFactory` without touching the runtime abstraction.
- **In-process default** (`aion.NewProcessRuntime` + `aion.MockRuntime` for tests). Sprint 5 reality: no `aion` binary on PATH, so `MockRuntime` is the canonical test impl. The subprocess backend is wired in `service.go` and exercised by the Sprint 4 ci.yml e2e step.
- **Sidecar mode (TASK-507 mention in brief §6.7) is out of scope** for Sprint 5. Sidecar will be a Sprint 6+ task that adds a second runtime backend.
- **PID tracking** uses string parsing of the `proc-<pid>-<uuid>` handle format (TASK-501 ProcessRuntime convention). Sprint 6 can swap this for an interface method on `aion.WorkerHandle` if the format changes.
- **Graceful shutdown** integrates with the existing `gracefulShutdown` wiring in `src/cmd/main.go`. Sequence: `agentFactory.Shutdown(ctx)` (SIGTERM all tracked) → `executionService.Shutdown(ctx)` (drain in-flight + `runtime.Close`).

## 1. Scope

### In scope
- `src/internal/agentfactory/agent_factory.go` — `AgentFactory`, `AgentHandle`, `Config`, `ParsePIDFromHandle`
- `src/internal/agentfactory/agent_factory_test.go` — unit tests with `aion.MockRuntime`
- `docs/sprint5/agent-creation-management-design.md` (this file)
- Integration with `src/cmd/main.go` `gracefulShutdown` for SIGTERM wiring (deferred to follow-up branch; the package itself is the prep deliverable)
- Standalone `AgentFactory` is constructable from main.go and from future code paths (TASK-432 endpoint, TASK-508 recovery)

### Out of scope (flagged for follow-up)
- **ExecutionService `createExecution` switch to AgentFactory** — separate branch `feat/sprint-5-task-507-execution-integration` (or rolled into TASK-432). Current `Provider: "anthropic"` / `PermissionMode: "default"` hard-codes stay in place; AgentFactory is dormant until a call site adopts it.
- **POST `/v1/tasks/:id/execute` handler** — TASK-432. Will consume `AgentFactory.SpawnAgent` as its primary spawn path.
- **Sidecar mode** — Sprint 6+ dedicated task.
- **Aion SDK import** (Q3) — Sprint 6+ dedicated task; the `aion.Runtime` interface is already in its final Sprint 6+ shape.
- **Integration test in `src/internal/integration/agentfactory_test.go`** — will land with TASK-510 (Tester-01) when they wire the dispatch-engine integration test.

## 2. Public API

```go
// internal/agentfactory/agent_factory.go

package agentfactory

import (
    "context"
    "github.com/fadhilfathi/AI-Software-Factory/internal/aion"
    "github.com/fadhilfathi/AI-Software-Factory/internal/model"
    "github.com/google/uuid"
    "go.uber.org/zap"
)

// DefaultAionModel is the default Aion model when agent.Runtime is empty.
const DefaultAionModel = "MiniMax-M3"

// DefaultAionProvider is the default Aion provider when agent.Runtime is empty.
const DefaultAionProvider = "TokenRouter"

// DefaultAionPermissionMode is the default Aion permission mode when agent.Runtime is empty.
const DefaultAionPermissionMode = "YOLO"

// DefaultShutdownGracePeriod is the default time given to tracked agents
// to exit gracefully after SIGTERM.
const DefaultShutdownGracePeriod = 5 * time.Second

// Sentinel errors.
var (
    ErrAlreadyShutdown = errors.New("agent factory already shut down")
    ErrAgentNotTracked = errors.New("agent not tracked")
    ErrNilAgent        = errors.New("agent is nil")
    ErrNilRuntime      = errors.New("runtime is nil")
)

// Config configures an AgentFactory. Zero values use the Aion
// defaults (MiniMax-M3, TokenRouter, YOLO).
type Config struct {
    DefaultModel          string
    DefaultProvider       string
    DefaultPermissionMode string
    ShutdownGracePeriod   time.Duration
    Logger                *zap.Logger
}

// AgentHandle is a tracked reference to a spawned Aion agent process.
type AgentHandle struct {
    AgentID        uuid.UUID
    ExecutionID    uuid.UUID
    ProjectID      uuid.UUID
    Role           string
    Model          string
    Provider       string
    PermissionMode string
    WorkerHandle   aion.WorkerHandle
    PID            int           // 0 for mock runtime
    StartedAt      time.Time
}

// AgentFactory spawns Aion agent subprocesses and tracks them for
// graceful shutdown.
type AgentFactory struct { /* ... */ }

// New creates a new AgentFactory wrapping the given aion.Runtime.
func New(runtime aion.Runtime, cfg Config) (*AgentFactory, error)

// SpawnAgent spawns an Aion agent process for the given agent and
// returns a tracked handle. Subsequent Shutdown() will SIGTERM the
// underlying process.
func (f *AgentFactory) SpawnAgent(
    ctx context.Context,
    agent *model.Agent,
    executionID uuid.UUID,
    input string,
) (*AgentHandle, error)

// Get returns the tracked handle for the given agent ID, or false.
func (f *AgentFactory) Get(agentID uuid.UUID) (AgentHandle, bool)

// Tracked returns a snapshot of all currently tracked agent handles.
func (f *AgentFactory) Tracked() []AgentHandle

// TrackedCount returns the number of currently tracked agents.
func (f *AgentFactory) TrackedCount() int

// Shutdown sends SIGTERM to all tracked agent subprocesses and closes
// the underlying runtime. Safe to call multiple times; only the first
// call has effect.
func (f *AgentFactory) Shutdown(ctx context.Context) error

// ParsePIDFromHandle extracts the PID from a process-runtime handle
// ("proc-<pid>-<uuid>"). Returns 0 and a typed error if the handle
// is not a process handle (e.g., mock runtime uses "mock-<n>-<uuid>").
func ParsePIDFromHandle(h aion.WorkerHandle) (int, error)
```

## 3. Lifecycle

```
                                +-----------------+
                                |   AgentFactory  |
                                |     .New()      |
                                +--------+--------+
                                         |
                                         v
+-----------------+  SpawnAgent  +--------+--------+  Shutdown   +-----------------+
|  caller code    |------------->|  tracked map    |------------>|  SIGTERM all    |
|  (TASK-432 etc) |              |  agent_id ->    |             |  tracked PIDs   |
+-----------------+              |  AgentHandle    |             +--------+--------+
                                 +-----------------+                      |
                                         ^                                v
                                         |                       +--------+--------+
                                         |                       |  runtime.Close() |
                                         |                       +-----------------+
                                         |
                                         +-- Get / Tracked / TrackedCount (observability)
```

`SpawnAgent` is non-blocking w.r.t. runtime.Wait - the caller is responsible for waiting on the underlying handle (or for the future `driveWorker` goroutine to do it). The factory's tracking is purely for SIGTERM wiring.

`Shutdown` is idempotent (`sync.Once` guarded), serializes the SIGTERM loop, and closes the underlying runtime last. The caller's `ctx` bounds the SIGTERM loop; agents that don't exit within `cfg.ShutdownGracePeriod` (default 5s) are not forcibly killed (the assumption is the runtime's `Close()` will SIGKILL the subprocess group).

## 4. Configuration

`Config` is a struct with zero-value defaults:

| Field                  | Default                   | Notes |
|------------------------|---------------------------|-------|
| `DefaultModel`         | `MiniMax-M3`              | Used when `agent.Runtime` is empty. |
| `DefaultProvider`      | `TokenRouter`             | Used when `agent.Runtime` is empty. |
| `DefaultPermissionMode`| `YOLO`                    | Used when `agent.Runtime` is empty. |
| `ShutdownGracePeriod`  | 5s                        | Currently advisory only; SIGTERM is best-effort. |
| `Logger`               | `zap.NewNop()`            | Test convenience. |

`agent.Runtime` is a `json.RawMessage` field on `model.Agent` (added in TASK-501). When non-empty, the factory attempts to unmarshal it into `{model, provider, permission_mode}` overrides:

```json
{"model": "claude-opus-4-7", "provider": "TokenRouter", "permission_mode": "YOLO"}
```

Missing keys fall through to the factory defaults. Malformed JSON is silently ignored (the spawn proceeds with factory defaults).

## 5. PID extraction

`aion.WorkerHandle` is `type WorkerHandle string` (defined type, not alias - see `src/internal/aion/runtime.go`). The ProcessRuntime writes handles in the format `proc-<pid>-<uuid>` (see `src/internal/aion/process.go`).

`ParsePIDFromHandle` parses this format:

```go
func ParsePIDFromHandle(h aion.WorkerHandle) (int, error) {
    s := string(h)
    const prefix = "proc-"
    if !strings.HasPrefix(s, prefix) {
        return 0, fmt.Errorf("not a process handle: %q", s)
    }
    rest := s[len(prefix):]
    dashIdx := strings.Index(rest, "-")
    if dashIdx < 0 {
        return 0, fmt.Errorf("malformed process handle: %q", s)
    }
    pid, err := strconv.Atoi(rest[:dashIdx])
    if err != nil {
        return 0, fmt.Errorf("malformed PID in handle %q: %w", s, err)
    }
    return pid, nil
}
```

For mock-runtime handles (`"mock-<n>-<uuid>"`), the prefix doesn't match and `ParsePIDFromHandle` returns `(0, error)`. The factory treats PID 0 as "no real process" and skips SIGTERM in `Shutdown`.

**Future refactor (Sprint 6+):** add a `PID() int` method to `aion.WorkerHandle` (changing it from `type WorkerHandle string` to `type WorkerHandle interface { String() string; PID() int }`) so ProcessRuntime and MockRuntime can both expose a PID without string parsing.

## 6. Cross-tenant / F-016 invariants

AgentFactory itself does NOT do project scoping. The caller is responsible for passing a `model.Agent` that has already been project-checked (e.g., `s.store.Agents().GetByID(ctx, agentID, callerProjectID)`). The factory only knows:

- `agent.ID` (uuid.UUID) - for tracking
- `agent.ProjectID` (uuid.UUID) - recorded on the handle for observability
- `agent.Role` (string) - recorded on the handle for observability

Future call sites (TASK-432 endpoint, TASK-508 recovery) MUST do the F-016 cross-tenant check before calling `SpawnAgent`. The factory will not re-check.

## 7. Graceful shutdown integration (deferred to follow-up)

`src/cmd/main.go` `gracefulShutdown` currently does:
1. `httpSrv.Shutdown(ctx)` - drain in-flight HTTP
2. `svc.Execution.Shutdown(execCtx)` - cancel service ctx + close runtime

The follow-up integration inserts step 1.5: `svc.AgentFactory.Shutdown(execCtx)` (or a top-level `agentFactory.Shutdown(execCtx)` if the factory is constructed outside the service graph).

The order matters:
- **First** `AgentFactory.Shutdown`: sends SIGTERM to all tracked agents. Agents that don't exit within the grace period are abandoned (the runtime's `Close()` will SIGKILL the subprocess group in step 3).
- **Then** `ExecutionService.Shutdown`: cancels the service-level stop ctx (drains in-flight mock goroutines) and calls `runtime.Close()` (which SIGKILLs the subprocess group as a backstop).

This gives us a two-stage shutdown: cooperative SIGTERM (with grace) → hard SIGKILL via the runtime. The 5s default grace period matches the existing `cmd/server/main.go` shutdown timeout.

## 8. Observability

The factory exposes three read-only accessors:

- `Get(agentID) (AgentHandle, bool)` - single-agent lookup.
- `Tracked() []AgentHandle` - full snapshot. Used by a future `GET /v1/admin/agents` endpoint (TASK-507 follow-up) and by the existing `/v1/executions/:id` response (to surface the PID, role, model, provider, permission mode).
- `TrackedCount() int` - cheap (no allocation) count.

`zap.Logger` is used for structured logging at:
- `Info` on SpawnAgent (agent_id, execution_id, role, model, provider, permission_mode, pid)
- `Debug` on Shutdown when PID is 0 (skip SIGTERM)
- `Info` on Shutdown when SIGTERM sent
- `Warn` on Shutdown when SIGTERM fails (e.g., process already exited)
- `Warn` on post-spawn SIGTERM failure during shutdown race

## 9. Testing

### Unit tests (`agent_factory_test.go`, 16 tests)

- `TestNew_DefaultsApply` - zero-value `Config` produces the Aion defaults.
- `TestNew_NilRuntime` - nil runtime returns `ErrNilRuntime`.
- `TestSpawnAgent_NilAgent` - nil agent returns `ErrNilAgent`.
- `TestSpawnAgent_DefaultsApplied` - agent with empty `Runtime` gets MiniMax-M3/TokenRouter/YOLO.
- `TestSpawnAgent_AgentRuntimeOverrides` - agent with `Runtime` JSON gets the overrides.
- `TestSpawnAgent_AgentRuntimePartialOverrides` - missing fields fall through to defaults.
- `TestSpawnAgent_AgentRuntimeMalformedJSON` - malformed JSON falls through to defaults.
- `TestGet_TrackedAfterSpawn` - `Get` returns the handle after `SpawnAgent`.
- `TestGet_NotFound` - `Get` on unknown agent returns `(AgentHandle{}, false)`.
- `TestTracked_Empty` - fresh factory returns empty slice.
- `TestTracked_MultipleAgents` - multiple spawns are all tracked.
- `TestShutdown_Idempotent` - calling `Shutdown` twice returns nil on the second call.
- `TestShutdown_AlreadyShutdownOnSpawn` - `SpawnAgent` after `Shutdown` returns `ErrAlreadyShutdown`.
- `TestShutdown_ClosesRuntime` - `Shutdown` calls `runtime.Close()`.
- `TestShutdown_ConcurrentSafety` - `Shutdown` is safe to call concurrently with `SpawnAgent`.
- `TestParsePIDFromHandle_Valid` - well-formed handle parses to its PID.
- `TestParsePIDFromHandle_LargePID` - large PIDs (up to int max) parse correctly.
- `TestParsePIDFromHandle_MockHandle` - mock handle returns an error.
- `TestParsePIDFromHandle_MalformedPID` - non-numeric PID returns an error.
- `TestParsePIDFromHandle_NoDashSeparator` - missing dash separator returns an error.
- `TestSpawnAgent_PopulatesStartedAt` - `StartedAt` is in [before, after] window.

Test runtime: `aion.MockRuntime` (in-process, no `aion` binary needed). The real-process SIGTERM delivery is left to E2E (requires Unix signal handling on a CI runner with the `aion` binary installed; not in scope for the no-Go-on-Windows unit-test suite).

## 10. References

- TASK-501 (Sprint 5): `docs/sprint5/aion-runtime-integration.md` (parent runtime)
- TASK-502 (Sprint 5): `docs/sprint5/dispatch-engine-design.md` (consumer; dispatch → runtime)
- TASK-432 (Sprint 5 follow-up): `POST /v1/tasks/:id/execute` - primary consumer of `AgentFactory.SpawnAgent`
- TASK-508 (Sprint 5): Execution Recovery - secondary consumer
- Lead's TASK-507 brief: dispatched via `docs/sprint5/lead-updates/2026-06-14-dev01-task-502-shipped-task-507-go.md` (inline constraints)
- `src/internal/aion/runtime.go` (Runtime interface)
- `src/internal/aion/process.go` (ProcessRuntime handle format)
- `src/internal/model/agent.go` (Agent.Runtime field)
