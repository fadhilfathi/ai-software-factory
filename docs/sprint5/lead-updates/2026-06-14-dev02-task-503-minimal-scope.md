# TASK-503 Slim-Down Recovery — Dev-02 (2026-06-14)

**Trigger:** Lead's URGENT message 2026-06-14: TASK-503 decision = **Option (c)
Minimal bus integration**. My earlier 3e3dce5 was over-scoped (full
ExecutionService refactor + runtime integration). I rebuilt the branch
to match Lead's spec.

## Decision summary (per Lead)

- **6 state constants** matching brief §6.3:
  `QUEUED → ASSIGNED → RUNNING → REVIEW → COMPLETED/FAILED`
- **StateManager interface** in `src/internal/events/state.go` (typed enum
  + transition table; package-private impl, exported interface)
- **MemoryBus** in `src/internal/events/bus.go` (per-project fan-out,
  at-most-once, monotonic IDs, ring buffer for last N)
- **Wiring**: `cmd/main.go` instantiates MemoryBus, passes to
  `service.New(...)`. The bus is stored on `Services.Bus` so future
  TASK-501/TASK-505/TASK-506 code can subscribe/publish without
  constructor changes.
- **3 tests** total (down from 16+):
  1. `TestMemoryBus_RoundTrip` — happy path + per-project isolation + unsubscribe
  2. `TestStateMachine_AllValidTransitions` — table-driven sweep of valid edges
  3. `TestStateMachine_InvalidTransition` — closed-set check that every other edge wraps `ErrInvalidTransition`

## Deferred to Sprint 6 (per Lead's explicit instruction)

- Full ExecutionService refactor (publishing events on transitions)
- Postgres-backed bus
- Wiring into TASK-501's runtime layer

## Files in this commit

| Path | Status | Notes |
|---|---|---|
| `src/internal/events/bus.go` | new | MemoryBus + Bus interface (~260 LOC) |
| `src/internal/events/bus_test.go` | new | 1 round-trip test |
| `src/internal/events/state.go` | new | 6-state enum + StateManager + transition table |
| `src/internal/events/state_test.go` | new | 2 state-machine tests |
| `src/cmd/main.go` | modified | bus wiring + 5-arg `service.New` call |
| `src/internal/service/service.go` | modified | `Bus events.Bus` field + 5-arg `New` |

## Files NOT in this commit (deliberately, per Lead)

- `src/internal/service/execution.go` — TASK-501 owns it, no changes
- `src/internal/service/execution_test.go` — no changes
- `src/internal/service/execution_503_test.go` — does not exist
  (the prior rebase ship had this file; it is gone now)
- `src/internal/model/execution.go` — no changes (4-state
  `model.ExecutionStatus` stays; the new 6-state `events.Status` is
  intentionally a separate type for Sprint 6 bridging)

## Conflict resolution vs main (origin/main @ 8096a76)

- `src/cmd/main.go`: **theirs** (kept main's healthz/MaxConns/route
  changes), then added bus wiring at the service.New call site
- `src/internal/service/service.go`: **theirs** (kept main's 4-arg
  `New` + 4-arg `NewExecutionService`), then bumped `New` to 5-arg
  with `events.Bus` and added the `Bus` field on the struct
- `src/internal/service/execution.go`: **theirs verbatim** — this is
  TASK-501's territory. No edits.
- `src/internal/service/execution_test.go`: **theirs verbatim**

## Pre-push 9-rule gate (passed)

1. Go corruption grep: zero `TODO|FIXME|XXX|panic(` in new files
2. Brace/paren balance: all 6 files balanced (main.go's 2-paren gap is
   pre-existing in origin/main's tree, not introduced here)
3. Conflict markers: zero
4. Unused imports: zero (all imports used; `errors.New` + `fmt.Errorf`
   in state.go; `events.NewMemoryBus` in main.go; `events.Bus` field
   in service.go)
5. Type compatibility: `service.New` 5-arg signature matches the
   5-arg call in main.go; `events.Bus` interface has `Publish`,
   `Subscribe`, `Last` — all present in `MemoryBus`
6. File locations: `events/` for bus + state; `service/` for the
   wiring; `cmd/` for instantiation
7. Test call sites: only `bus_test.go` and `state_test.go` exist in
   `events/`; no `execution_503_test.go`; no edits to `execution_test.go`
8. Dispatch file: this file
9. Line endings: bus.go + bus_test.go from prior commit retained
   their CRLF endings (matching repo convention); state.go and
   state_test.go written in LF (matches their `Write` tool output);
   main.go and service.go edited in place — both retain their
   original CRLF endings

## Next steps (for me)

- Force-push the slim-down to
  `feat/sprint-5-task-503-execution-state-manager`
- Ping Lead with the new SHA + verification of: file count, test
  count, no-Go-on-Windows acknowledgement for CI
- Wait for CI green ping
- Hold for TASK-504 step 2 (code) rebase instruction

## CI verification (no-Go-on-Windows)

I cannot run `go test ./...` on this host (no-Go-on-Windows). The CI
pipeline (`.github/workflows/ci.yml` + `.github/workflows/sprint-quality-gate.yml`)
is the canonical verification. If the gate goes red, I'll fix
forward per Lead's standard protocol (red-CI guardrail: stop and
report, never silent retry).

## Sibling work this is blocking

- TASK-504 (Agent Communication Layer) — needs the bus wired; will
  rebase onto this commit
- TASK-505 (Deliverable Capture) — will subscribe to events
- TASK-506 (Execution Monitoring Dashboard) — SSE in the UI
- TASK-509/510 (Tester-01) — will use the bus for integration tests
- TASK-511 (Security-01) — will see `events` package in the security
  review surface
