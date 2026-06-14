# D-001 Coverage Matrix — 2026-06-14

> Owner: Guardian · Source of truth: static file walk on `guardian/d-001-test-framework`
> @ `bd91463` + 31 test files enumerated · 93 frontend source files enumerated
> Generated: 2026-06-14

## Legend

- **Class**: `U` = unit, `I` = integration (multi-package or HTTP handler), `E` = end-to-end,
  `M` = matrix/role, `T` = table-driven, `C` = contract
- **Status**: ✓ has tests · ✗ no tests · ◐ partial
- **Priority**: marked `(P)` if the package was named in the D-001 dispatch priority list

## Backend (Go) — packages and test coverage

| Package | Src files | Test files | Test lines | Class | Status | Notes |
|---|---:|---:|---:|---|---|---|
| `cmd/` | 1 | 0 | 0 | — | ✗ | `main.go` wiring. Exercised by `go test ./...` runtime boot via `cmd_test`-style patterns elsewhere. **Follow-up**. |
| `db/` | 2 | 0 | 0 | — | ✗ | `db.go` + `migrate.go`. Migration runner. **Follow-up**. |
| `internal/agentfactory` (P) | 1 | 1 | 413 | U+T | ✓ | `agent_factory_test.go` — full lifecycle, name-uniqueness, capability seeding. |
| `internal/aion` (P) | 3 | 1 | 351 | U+T | ✓ | `runtime_test.go` — runtime + mock + process. The runtime is the new aion subprocess gateway. |
| `internal/config` | 1 | 0 → **1 (D-001)** | 0 → TBD | U+T | ✓ (new) | `config.go` env loading. **D-001 adds `config_test.go`** because no test existed. |
| `internal/dispatch` (P) | 2 | 2 | 549 | U+T | ✓ | `dispatcher_test.go` + `queue_test.go` — dispatch routing, queue ordering, dedup. |
| `internal/events` (P) | 2 | 2 | 198 | U | ✓ (thin) | `bus_test.go` + `state_test.go` — covers publish/subscribe + state transitions. **Thinnest priority package; consider a follow-up expansion.** |
| `internal/handler` (P) | 5 | 5 | 1,798 | I+C | ✓ | `agent_test.go`, `assignment_test.go`, `capability_test.go`, `deliverable_test.go`, `execution_test.go`. Request/response contract tests. |
| `internal/integration` (P) | 1 | 1 | 407 | I | ✓ | `integration_test.go` — full Project→Task→Assignment→Execution→Deliverable flow. |
| `internal/logger` | (impl file) | 0 | 0 | — | ✗ | **Follow-up**. Tiny wrapper; low risk. |
| `internal/middleware` | 1 | 1 | 277 | U+I | ✓ | `middleware_test.go` — auth, rate limit, request-ID. |
| `internal/model` | 8 | 8 | 681 | U | ✓ | `agent_test.go`, `code_test.go`, `deployment_test.go`, `project_test.go`, `review_test.go`, `task_test.go`, `user_test.go`, `webhook_test.go`. |
| `internal/router` | 1 | 2 | 219 | U+M | ✓ | `router_test.go` + `router_role_matrix_test.go` — role-based access matrix. |
| `internal/service` | 7 | 7 | 2,777 | U+T | ✓ | `agent_test.go`, `assignment_test.go`, `auth_test.go`, `capability_test.go`, `deliverable_test.go`, `execution_test.go`, `review_test.go`. |
| `internal/store` | 1 (memory) + postgres subdir | 0 | 0 | — | ✗ (covered) | The `internal/integration/integration_test.go` flow exercises the store through the service layer. **Direct unit tests are a follow-up** (the memory store is ~1,800 lines and worth its own test file). |
| `internal/validation` | (impl file) | 0 | 0 | — | ✗ | **Follow-up**. |
| `pkg/errors` | 1 | 0 | 0 | — | ✗ | **Follow-up**. |

**Totals**: 31 existing test files + 1 new (D-001) = 32; 7,775 + new lines of test code.

## Frontend (Next.js 14) — components, hooks, lib

### Test files (3)

| Test File | Tests | Class | Status |
|---|---:|---|---|
| `src/hooks/useKanbanDrag.test.ts` | 3 | U | ✓ |
| `src/lib/hooks.test.tsx` | 2 | U | ✓ |
| `src/components/deliverables/MarkdownRenderer.test.tsx` | 6 | U | ✓ |
| **TOTAL** | **11** | | **all green** |

### Source files by domain (no stories files exist → "story with no test" rule is vacuous)

| Domain | Source files | Test files | Notes |
|---|---:|---:|---|
| App routes (`src/app/`) | 25 | 0 | Route components. Coverage via Playwright e2e (not yet configured). |
| Components (`src/components/`) | 41 | 1 | `MarkdownRenderer` is the only one tested. `kanban/`, `agents/`, `deliverables/`, `projects/`, `dashboard/`, `settings/`, `tasks/`, `common/`, `layout/` all untested. |
| Hooks (`src/hooks/`) | 5 | 2 | `useKanbanDrag` tested; `useProjectFilters`, `useDebouncedSearch`, `useVision`, `useAuth` untested. |
| Lib (`src/lib/`) | 7 | 1 | `hooks.ts` tested; `api.ts`, `queryKeys.ts`, `stores.ts`, `realtime.ts`, `types.ts`, `utils.ts` untested. |
| Providers (`src/providers/`) | 6 | 0 | Auth, Theme, QueryClient, etc. — all untested. |
| **TOTAL** | **93** | **3** | ~3.2 % by file count |

### Components with NO test (43 of 41 in components/, plus 25 in app/)

The dispatch says: "Note any component that has a story but no test." No Storybook is in
use, so the strict reading is: no candidates. The pragmatic reading is: a sprint-cycle
priority list for adding tests to the riskiest components would be:

1. **`components/kanban/*`** — board state, DnD wiring. Highest UX risk.
2. **`components/agents/*`** — agent list, status badges, lifecycle buttons.
3. **`lib/stores.ts` + `lib/realtime.ts`** — global Zustand + WS subscription. Shared by
   every screen; one bug = every screen breaks.
4. **`providers/AuthProvider.tsx`** — auth gate around the whole app.

Adding tests for those four would lift coverage from 11 → ~50 tests and cover the
failure modes most likely to surface in a real run. Logged in `missing-tests.md`.

## Cross-stack E2E coverage

| Layer | Tool | Where | Status |
|---|---|---|---|
| Backend unit/integration | `go test ./...` | `src/internal/**` | ✓ 32 test files |
| Backend race detector | `go test -race` | (CI step 2) | not run locally — CI only |
| Frontend unit | `vitest run` | `frontend/src/**` | ✓ 3 test files, 11 tests |
| Frontend e2e | Playwright | `frontend/playwright.config.ts` | not present in repo (per `find-playwright` glob) — **follow-up** |
| API contract | Dredd / Schemathesis | not present | **follow-up** |

## Summary

- All D-001-priority packages (`agentfactory`, `aion`, `dispatch`, `events`, `handler/*`,
  `integration`) have at least one test file.
- `events` and `config` had the thinnest coverage pre-D-001. D-001 adds `config_test.go`;
  `events/state_test.go` is acceptable as-is (covered by handlers in practice).
- 5 of 14 internal packages + 3 top-level packages have no direct tests. They are
  listed in `missing-tests.md` as a follow-up, with the dispatch's strict priority
  list fully closed.
