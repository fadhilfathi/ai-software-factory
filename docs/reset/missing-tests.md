# D-001 Missing-Critical-Tests Triage — 2026-06-14

> Owner: Guardian · Resolution: 1 added, 5 deferred to follow-up tickets.
> Companion to `coverage-matrix.md`.

## D-001 dispatch priority list — all closed

The dispatch calls out these packages as the must-cover set:

| Package | Test file (pre-D-001) | Lines | D-001 action |
|---|---|---:|---|
| `internal/agentfactory` | `agent_factory_test.go` | 413 | None needed. |
| `internal/aion` | `runtime_test.go` | 351 | None needed. |
| `internal/dispatch` | `dispatcher_test.go`, `queue_test.go` | 549 | None needed. |
| `internal/events` | `bus_test.go`, `state_test.go` | 198 | Acceptable. State transitions covered; bus pub/sub covered. |
| `internal/handler/*` | 5 files | 1,798 | None needed. |
| `internal/integration/` | `integration_test.go` | 407 | None needed. |

**Net result of D-001 on the priority list: zero new test files in the priority packages.**
The priority list is closed as-is. The two smallest (events at 198 lines) are within
"thin but covered" tolerance — adding more tests there would be duplicate coverage of
state-transition behaviour that's already exercised through the handler/service tests.

## D-001 NEW test file (1)

### `internal/config/config_test.go` — NEW

**Why**: The `config` package is the entry point for every required env var
(`DB_HOST`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_PORT`, `JWT_SECRET`). A bug here —
a misspelled fallback, a silent fallthrough from a misformatted int, a CSV parser that
crashes on empty strings — would take the entire backend down at boot. The 220-line
file had no direct tests. D-001 adds ~110 lines of table-driven tests covering:

- `getEnv` fallback semantics (present, empty-treated-as-missing, default used)
- `getEnvInt` integer parsing (valid int, garbage → default, empty → default)
- `getEnvIntRequired` panic on missing and on non-integer
- `getEnvRequired` panic on missing
- `getEnvBool` "true"/"1" → true, "false"/"0" → false, anything else → false, default fallback
- `parseCSV` / `splitCSV` / `trimSpace` edge cases (empty string, single value,
  trailing comma, whitespace, multi-comma)
- `Load()` smoke: a fully-populated env produces a non-nil `*Config` with all
  sections zero-initialised (we never assert the global env, only the contract)

**Verification**: Cannot run `go test ./internal/config` locally (Go not installed —
see `test-baseline.md`). CI step 2 will run it on PR. If CI reports a failure, the
test file will be patched and re-pushed.

## Deferred to follow-up tickets (5 packages + 3 top-level)

These are NOT in the D-001 priority list. They are real coverage gaps, but D-001's
brief is to ship the priority list, not the entire surface. Each is logged here for
Ops/Builder to pick up in a subsequent sprint.

| Package | Why deferred | Suggested follow-up |
|---|---|---|
| `internal/store` (memory) | The store is ~1,800 lines and the surface is huge. Coverage comes through `integration_test.go` via the service layer. A direct unit test is a 200+ line undertaking. | Sprint+1: add `store/memory_test.go` covering CRUD round-trips for each sub-store and the denormalised indexes (assignmentByTask, activeAssignmentByTask, workerByAgent/Execution). |
| `internal/logger` | Tiny wrapper around a stdlib logger. Low bug surface. | Sprint+1: 30-line smoke test for level filtering. |
| `internal/validation` | Lightweight rules; covered by service-layer invalid-input tests. | Sprint+1: 50-line rule table. |
| `internal/cmd/` | `main.go` is wiring, not logic. Boot path is exercised by `go build` + `docker-compose up`. | Skip unless a refactor lands. |
| `internal/db/` | Migration runner is integration-tested by spinning up the schema. | Skip unless a migration bug appears. |
| `pkg/errors` | Tiny error-type file. | Skip. |
| `frontend/src/components/kanban/*` | Highest UX risk area. ~10 components, no tests. | Sprint+1: kanban DnD + status transition tests. |
| `frontend/src/lib/stores.ts` | Global Zustand store. One bug breaks every screen. | Sprint+1: store-reducer tests. |
| `frontend/src/lib/realtime.ts` | WS subscription. Reconnection logic is racey. | Sprint+1: mock-WS tests. |
| `frontend/src/providers/AuthProvider.tsx` | Auth gate; failure = lock-out. | Sprint+1: redirect / refresh tests. |
| Playwright e2e | Not present in repo. | Sprint+1: bootstrap `playwright.config.ts` + 1 happy-path spec. |

## What D-001 does NOT add and why

- **No new integration tests** — `internal/integration/integration_test.go` (407 lines)
  already covers the full Project→Task→Assignment→Execution→Deliverable flow against
  the in-memory store. Adding more would be duplicate coverage.
- **No race-detector tests** — `go test -race` runs in CI step 2. The existing tests
  should pass under `-race`; if any don't, that's a CI signal to fix, not a D-001 task.
- **No new handler tests** — 1,798 lines of handler tests already cover all five
  endpoints.
- **No Makefile** — `scripts/test.sh` already exists and is comprehensive. The
  dispatch allowed "or document `scripts/test.sh`"; documentation is the
  lower-overhead path. If the team wants a `make test` wrapper, that's a separate
  follow-up.
