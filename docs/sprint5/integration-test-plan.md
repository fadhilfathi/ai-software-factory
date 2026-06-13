# Sprint 5 — Integration Test Plan (E2E harness design)

**Owner:** Tester-01
**Source brief:** `docs/sprint5/brief.md` §6.10
**Linked matrix:** `docs/sprint5/workflow-validation.md` (TASK-509, this agent)
**Date:** 2026-06-13
**Status:** design draft (Wave 1-3 not yet started; harness is the contract TASK-501..508 must satisfy)

---

## 1. Purpose

Design the E2E test harness for Sprint 5's real agent execution engine. The harness automates the cross-route integration tests (L1 in the workflow-validation matrix) and is the canonical execution point for TASK-513 (CI/CD Quality Gate, step 13).

The harness solves the two problems Lead called out in the kickoff:

1. **How to spawn a fake Aion CLI in tests** — answer: in-process `fakeRuntime` (Mode A, default) + subprocess `realRuntime` (Mode B, gated by `AION_E2E=1`)
2. **How to wait for state transitions without sleep** — answer: state-manager event subscription via `Watch(id) <-chan StateEvent` (per TASK-503 design; falls back to `assert.Eventually` polling until TASK-505/506 land `events.MemoryBus`)

**Verified contract from TASK-501 (commit 5afc2ad on `feat/sprint-5-task-501-aion-runtime`):**
- `aion.Runtime = {Spawn, Wait, Cancel, Close}` — NOT my sketched `{Spawn, Send, Recv, Cancel}`. The fake implements `Wait(handle) WorkerResult` (blocking call that returns the worker's final result) rather than a sync `Send`/`Recv` pair. The handshake pattern is implicit in `Wait()`.
- `WorkerHandle` is an opaque `string` — callers don't introspect; they pass it back to `Wait`/`Cancel`.
- `WorkerSpec` carries `ExecutionID, TaskID, AgentID, ProjectID, Model, Provider, PermissionMode, Attempt` — the project ID is in the spec, not a separate argument.
- `WorkerStatus = pending|running|completed|failed|cancelled` (5 states, distinct from the 6-state `ExecutionStatus` for the state manager).

## 2. Two-mode runtime

### Mode A — `fakeRuntime` (in-process, default)

- Implements `aion.Runtime` directly in Go. **No subprocess.**
- Behavior controlled by a `FakeScript` — a list of `{Result, Delay, Error}` actions tied to `Wait()` calls.
- Honors the same wire shape as TASK-501's `aion.Runtime` (verified at 5afc2ad: `{Spawn, Wait, Cancel, Close}`) so the in-process handler logic is the same as Mode B. Switching to Mode B is a single-line `realRuntime()` call, not a test rewrite.
- Sub-second per test. The default for `go test`.

**Sketch:**
```go
// src/internal/integration/fake_runtime.go
package integration

import (
    "context"
    "encoding/json"
    "testing"
    "time"

    "github.com/fadhilfathi/AI-Software-Factory/internal/aion"
)

type FakeAction struct {
    Kind   string          // "send" | "recv" | "delay" | "error" | "exit"
    From   string          // outgoing message type (for send)
    Body   json.RawMessage // outgoing message body (for send)
    Wait   time.Duration   // for delay
    Exit   int             // for exit
}

type fakeRuntime struct {
    t      testing.TB
    script []FakeAction
    calls  *[]FakeCall
}

func NewFakeRuntime(t testing.TB, script []FakeAction) aion.Runtime {
    return &fakeRuntime{t: t, script: script}
}

// Spawn/Wait/Cancel/Close implemented by indexing into script[] and
// returning the configured action. State held in a map[Handle]int.
```

### Mode B — `realRuntime` (subprocess, gated)

- **In-process Go SDK with CLI shim fallback** (per Lead Q4): the default transport is the Go SDK; when `AION_E2E=1`, the CLI shim path is used — `os/exec` against the `aion` binary.
- Binary path: `aion` (PATH lookup) by default; override via `AION_BINARY` env var.
- Both transports conform to the same `aion.Runtime` interface; tests should treat them as identical.
- Configured with `provider=TokenRouter model=MiniMax-M3 permission_mode=YOLO` per TASK-507 (env vars on the child process).
- Multi-second per test. **Run only when `AION_E2E=1` env var is set.**
- Used in nightly CI / pre-release. **NOT enabled in `sprint-quality-gate.yml` step 13** (per brief §6.13: "mock it for CI; full path for nightly").

**Sketch:**
```go
// src/internal/integration/real_runtime.go
package integration

import (
    "os/exec"
    "testing"

    "github.com/fadhilfathi/AI-Software-Factory/internal/aion"
)

func NewRealRuntime(t testing.TB) (aion.Runtime, error) {
    if os.Getenv("AION_E2E") != "1" {
        t.Skip("AION_E2E=1 not set; skipping subprocess smoke")
    }
    cmd := exec.Command("aion",
        "--provider=TokenRouter",
        "--model=MiniMax-M3",
        "--permission-mode=YOLO",
        "--stdio",
    )
    // Wire stdin/stdout pipes; same JSON-over-stdio protocol as fake.
    return &realRuntime{cmd: cmd}, nil
}
```

## 3. Wait-without-sleep pattern (event subscription)

The state manager (TASK-503) emits state events:
```go
package execution

type StateEvent struct {
    ExecutionID uuid.UUID
    From, To    State  // QUEUED → ASSIGNED → RUNNING → REVIEW → COMPLETED | FAILED
    At          time.Time
    Reason      string
}

func (m *Manager) Watch(id uuid.UUID) <-chan StateEvent
```
The channel is closed when the execution reaches a terminal state (`COMPLETED` or `FAILED`).

**Test pattern:**
```go
// waitForState blocks until the execution reaches `want` or the
// timer fires. Returns the last seen state (useful for the
// timeout error message).
func waitForState(t *testing.T, mgr *execution.Manager, id uuid.UUID, want State) State {
    t.Helper()
    var last State
    timer := time.NewTimer(5 * time.Second)
    defer timer.Stop()
    for {
        select {
        case evt, ok := <-mgr.Watch(id):
            if !ok {
                return last  // closed = terminal
            }
            last = evt.To
            if evt.To == want {
                return evt.To
            }
        case <-timer.C:
            t.Fatalf("timeout waiting for state %s; last seen: %s", want, last)
        }
    }
}
```

**Polling fallback** (for *side-effect* checks, not *primary* state transitions):
```go
assert.Eventually(t, func() bool {
    versions, err := svc.ListDeliverableVersions(t.Context(), taskID)
    return err == nil && len(versions) == 2
}, 2*time.Second, 10*time.Millisecond,
    "deliverable should have 2 versions after the agent's second emit")
```

**No `time.Sleep` in test bodies.** This is a discipline rule, not just a style preference:
- Sleeps mask race conditions in tests (works locally, fails in CI)
- Sleeps make tests slow (cumulative cost across 30+ sub-tests)
- The state manager's `Watch()` gives us a deterministic signal — use it

If a test author finds themselves wanting `time.Sleep`, the right move is to ask Dev-02 (TASK-503 owner) to expose a hook on the state manager, or fall back to `assert.Eventually` with a 2s budget.

## 4. File layout

```
src/internal/integration/
├── store.go                          (Sprint 4 — DB-agnostic Store alias)
├── integration_test.go               (Sprint 4 — T1 Smoke + T2 malformed UUIDs)
├── integration_sprint5_test.go       (Sprint 5 — Mode A, all A*.x + E*.x)
├── aion_subprocess_test.go           (Sprint 5 — Mode B, E5 only, gated by AION_E2E=1)
├── fake_runtime.go                   (Sprint 5 — Mode A implementation)
├── real_runtime.go                   (Sprint 5 — Mode B implementation)
└── sprint5_helpers.go                (Sprint 5 — newIntegrationRouter, waitForState, etc.)
```

**Test bootstrap (extends Sprint 4's `newIntegrationRouter`):**
```go
// newSprint5Router wires the same Gin router + services, but
// uses the fake runtime for execution. Auth bypass unchanged.
func newSprint5Router(t *testing.T, s integration.Store, runtime aion.Runtime) *Sprint5TestEnv {
    t.Helper()
    gin.SetMode(gin.TestMode)
    log := zap.NewNop()

    capSvc := service.NewCapabilityService(s, log)
    agentSvc := service.NewAgentService(s)
    taskSvc := service.NewTaskService(s, log)
    assignmentSvc := service.NewAssignmentService(s, capSvc, log)
    execSvc := service.NewExecutionService(s, log, runtime)  // <-- the only change
    delivSvc := service.NewDeliverableService(s, log)

    r := gin.New()
    r.Use(func(c *gin.Context) {
        c.Set("request_id", "test-int-s5-rid-001")
        c.Set("user_id", "11111111-1111-1111-1111-111111111111")
        c.Next()
    })
    // ... route wiring identical to Sprint 4 ...
    return &Sprint5TestEnv{Router: r, Store: s, /* ... */}
}
```

## 5. Coverage map (TASK-509 matrix → TASK-510 automation)

| Matrix row | Go test | Sub-test | Notes |
|------------|---------|----------|-------|
| A1.1 | `TestStage1_AgentCreation` | `t.Run("valid_body", ...)` | Happy path |
| A1.2 | same | `t.Run("duplicate_name", ...)` | Conditional — see open question 1 |
| A1.3 | same | `t.Run("malformed_capability", ...)` | |
| A1.4 | same | `t.Run("cross_tenant", ...)` | Verifies TASK-419 fix |
| A2.1..A2.5 | `TestStage2_TaskAssignment` | 5 sub-tests | Includes A2.4 cross-tenant (TASK-420) + A2.5 F-017 |
| A3.1 | `TestStage3_Execution` | `t.Run("happy_state_machine", ...)` | Uses `waitForState` |
| A3.2 | same | `t.Run("event_order_monotonic", ...)` | |
| A3.3 | same | `t.Run("illegal_transition_rejected", ...)` | |
| A3.4 | same | `t.Run("cross_tenant", ...)` | Verifies TASK-422 fix |
| A4.1..A4.4 | `TestStage4_Deliverable` | 4 sub-tests | Includes A4.3 F-023 + A4.4 F-006 |
| A5.1..A5.4 | `TestStage5_Recovery` | 4 sub-tests | Each uses a different `FakeScript` |
| E1..E4 | `TestE2E_*` | 4 cross-stage tests | Each one calls into the stage tests above + adds cross-stage assertions |
| E5 | `TestE2E_SubprocessSmoke` (gated) | 1 | `AION_E2E=1` only |

Total automated sub-cases: ~22 (17 A*.x + 4 E*.x + 1 E5).

## 6. CI integration (TASK-513)

`sprint-quality-gate.yml` step 13 — **add** a new sub-step that asserts the Aion runtime is mockable in CI:

```yaml
- name: Assert Aion runtime is mocked (not subprocess)
  run: |
    if [ "$AION_E2E" = "1" ]; then
      echo "::error::AION_E2E must not be set in the gate; nightly only"
      exit 1
    fi
    echo "AION_E2E unset; gate uses Mode A (fakeRuntime). OK"
- name: Integration tests (Sprint 5)
  run: go test -count=1 -timeout 10m ./internal/integration/...
```

A separate nightly cron workflow (`.github/workflows/nightly-sprint5.yml`, **out of scope for Sprint 5** — flagged for TASK-513) would set `AION_E2E=1` and run the Mode B test.

## 7. Acceptance (TASK-510 closeout)

- All 17 A*.x sub-cases + 4 E*.x sub-cases pass in Mode A (`go test -count=1 -timeout 10m ./internal/integration/...`)
- E5 passes in Mode B when `AION_E2E=1` (gated; not run in the gate)
- Zero `time.Sleep` in `src/internal/integration/*.go` (verified by `grep -r 'time.Sleep' src/internal/integration/`)
- The fake runtime honors the JSON-over-stdio protocol (TASK-504's contract) — verified by `TestFakeRuntime_ProtocolConformance`
- `sprint-quality-gate.yml` step 13 includes the Aion-mock assertion
- `docs/sprint5/workflow-validation.md` and this file are consistent (matrix rows map 1:1 to Go sub-tests)

## 8. Risks

1. **The `Runtime` interface design from TASK-501 may differ from my assumption** — I sketched `Runtime{Spawn, Send, Recv, Cancel}`. If TASK-501 settles on a different shape (e.g., async callbacks instead of sync Send/Recv), the fake needs to be rewritten. **Mitigation:** I'll wait for TASK-501 to land, then verify the interface signature and adjust the fake. The fake is a thin shim; the test logic doesn't change.
2. **The state-manager event subscription may not be implemented by TASK-503** — if `Watch()` doesn't exist, the wait-without-sleep pattern falls back to `assert.Eventually` polling. **Mitigation:** polling works; it's just slower (2s timeout per assertion vs sub-second event-driven). I'll add a TODO and revisit when TASK-503 lands.
3. **`aion` CLI may not be available in CI** — Mode A handles this by not spawning a subprocess at all. **Mitigation:** the gate is Mode A only; Mode B is gated by `AION_E2E=1` and is not in the gate.
4. **Multi-clone confusion** — prior session's `integration_full_test.go` is not in this working tree (per the 4-check at session start). **Mitigation:** I ship proof (`git rev-parse HEAD` + `git status -s`) in the report; the new files will be in the working tree as untracked, per the team rule (DevOps-01 owns closeout commit).
5. **Test-runtime debt from Sprint 4 may have regressions** — the 4 backend debts (err.StatusCode, RequiredCapabilities, MockStore.DeliverableVersions, handler test mock) were all triaged as already-resolved. **Mitigation:** verify the integration_full_test.go from Sprint 4 still passes after TASK-510 lands (re-run the smoke sub-case as a regression check).

## 9. Open questions for Lead

**Resolved (Lead answered 2026-06-13):**
1. ✅ `aion` CLI distribution — in-process Go SDK (default transport) + CLI shim fallback (`os/exec` against the `aion` binary; binary path = `aion` (PATH) by default, override via `AION_BINARY` env var). Both conform to the same `aion.Runtime` interface; tests treat them as identical.
5. ✅ Multi-clone `integration_full_test.go` — confirmed: TASK-426 is `completed` on the board; re-writing the Sprint 4 full T1 is out of scope; the Sprint 4 smoke + T2 are the regression baseline.

**Still open:**
2. **Runtime interface signature** — does TASK-501's `Runtime` interface match my sketch (`Spawn, Send, Recv, Cancel`)? Affects the fake's method set.
3. ⏳ **State-manager `Watch()` API** — does it match my sketch (returns `<-chan StateEvent`, closes on terminal)? Affects the wait pattern. State machine is now LOCKED (Decision 4, 2026-06-14): 6 states `QUEUED → ASSIGNED → RUNNING → REVIEW → COMPLETED/FAILED`. The `Watch()` event shape will likely include `From`/`To` fields matching these state constants.
4. **Retry event delivery** — does TASK-508's retry policy emit events on the same `Watch` channel, or a separate one? Affects A5.4's test setup.

## 10. Cross-references

- TASK-509 test matrix: `docs/sprint5/workflow-validation.md`
- TASK-501 Aion runtime interface: `docs/sprint5/aion-runtime-integration.md` (Dev-01's lane; not yet written)
- TASK-503 state machine: depends on Dev-02's design (Wave 2)
- TASK-504 JSON-over-stdio protocol: Dev-02's schema doc (Wave 2)
- TASK-508 retry policy: Dev-02's design (Wave 3)
- TASK-513 CI/CD gate: `.github/workflows/sprint-quality-gate.yml` (extends Sprint 4)
- Sprint 4 integration baseline: `src/internal/integration/integration_test.go` (Sprint 4 smoke + T2; still passing)
