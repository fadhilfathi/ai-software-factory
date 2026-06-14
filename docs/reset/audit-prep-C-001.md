# C-001 Monitoring Dashboard — Pre-Scope Brief

**Owner of C-001:** Builder. Support: Leader (this brief) + Guardian (D-002 reviews the new IDOR surface if the real-time channel touches projects).
**Spec:** `docs/sprint4/agent-orchestration-design.md` §Dashboard + `docs/sprint5/agent-creation-management-design.md`
**Audit precedent:** `docs/reset/audit/A-001-audit.md`, `docs/reset/audit-prep-A-002.md`, `docs/reset/audit-prep-A-003.md`, `docs/reset/audit-prep-B-001.md`

**Critical dependency:** B-001 (Execution Engine) must ship the 6-state lifecycle FIRST. C-001 is downstream — it displays the new states. If C-001 lands first, the new state badges are `skipped` with TODO markers until B-001 ships.

## Deliverables (per the team contract)
- Running Agents view
- Active Tasks view
- Failed Tasks view
- Completed Tasks view

## Existing code floor (READ FIRST)

| File | Purpose | Status |
|------|---------|--------|
| `frontend/src/app/dashboard/page.tsx` | Main dashboard. Uses `useProjects`, `useAgents`, `useExecutions` hooks. 4 project metric cards + 5 agent metric cards + 1 execution list. | Partial. |
| `frontend/src/lib/hooks/` | `useProjects`, `useAgents`, `useExecutions` (and many more). `useProjects` returns a bare array; `useAgents` and `useExecutions` return envelopes per the Sprint 4 / Sprint 1-3 split. | Mixed. Envelope drift. |
| `frontend/src/lib/types.ts` | `AgentStatus` (6 values: idle/busy/initializing/error/retired/paused), `ExecutionStatus` (4 values: pending/running/completed/failed) | Out of date. Needs to grow to 6 execution states after B-001. |
| `frontend/src/lib/api.ts` | API client | Read. |
| `frontend/src/lib/realtime.ts` | RealtimeProvider wiring (Sprint 5) | Read. |
| `frontend/src/components/agents/`, `components/deliverables/`, `components/kanban/`, `components/activity/` | Per-domain UI components | Read; identify which are dashboard-related. |
| `frontend/src/components/shared/MetricCard.tsx`, `components/ui/{Badge,Skeleton,EmptyState,ProgressBar}.tsx` | Reusable UI | Read. |

## What C-001 needs to do (after B-001 ships the 6-state model)

### 1. Add the new execution status badges

Currently `EXEC_STATUS_COLOR` (page.tsx:23) has 3 states: completed (emerald), running (blue), failed (red). After B-001, add:
- `queued` (gray — waiting in the dispatch queue)
- `assigned` (yellow — agent picked, pool preparing)
- `review` (yellow — worker done, awaiting reviewer)

Update `lib/types.ts` `ExecutionStatus` to the 6-value enum. Add a label-mapping table (mirror `AGENT_STATUS_BADGE`) for the human-readable label and color.

### 2. Add new metric cards

The current dashboard has 4 project cards + 5 agent cards + 0 dedicated execution cards (the executions show as a list only). Add a third metrics row for executions:
- **Queued** (count) — the dispatch queue depth. Critical for the operator's view of "how backed up is the system"
- **In Review** (count) — the work that's done but waiting for human review
- **Running** (count) — currently in flight
- **Failed (24h)** (count) — recent failures, with a sparkline if possible

### 3. Add status filters / tabs

The current executions list shows 10 with no filter. Add a tab/segmented-control above the list:
- All
- Queued
- Assigned
- Running
- Review
- Completed
- Failed

The list filters based on the selected tab. Cursor pagination per the existing `useExecutions` hook contract.

### 4. Fix the useProjects envelope mismatch

`useProjects` returns a bare array; `useAgents`/`useExecutions` return envelopes. The page does `(agentsData as {data?: unknown[]})?.data ?? []` to work around it. This is spec drift — make `useProjects` return the same envelope shape as the others. Per the A-001/A-002/A-003 pattern: `{"data": [...], "next_cursor": "..."}`.

### 5. Wire real-time updates

The `RealtimeProvider` exists but I don't see it wired to the dashboard. Hook the metric cards and the executions list to a real-time channel (WebSocket / SSE — verify the existing transport). When an execution transitions state, the dashboard updates without a refetch.

### 6. F-D002-004 IDOR (cross-tenant data leak)

Same X-Project-ID surface as the backend. The `useProjects`, `useAgents`, `useExecutions` hooks should respect the X-Project-ID header. The dashboard's cross-tenant test (per D-003) should verify a project-A user sees only project-A data.

## Suggested PR shape (after B-001 ships)
- Commit 1: `feat(dashboard): extend ExecutionStatus to 6 states with new badges` (mirror A-001/A-002/A-003 spec drift)
- Commit 2: `feat(dashboard): add queued/in-review/running/failed metric cards`
- Commit 3: `feat(dashboard): add status filters/tabs to executions list`
- Commit 4: `fix(dashboard): align useProjects envelope with useAgents/useExecutions`
- Commit 5: `feat(dashboard): wire RealtimeProvider to dashboard updates`
- Commit 6: `docs(audit): C-001 monitoring dashboard audit + pre-push gate`

If 6 commits is too many, fold 5 into 6. The most user-visible wins are 2 and 3.

## When this must land

After B-001 (Execution Engine) ships. C-001 is downstream of B-001's 6-state model — without the model, the new badges are placeholders.

## What I (Leader) will do

- Review the audit doc.
- Cross-ref the dashboard envelope shape with the A-002-11 / A-003-01 standing `{data: ...}` pattern.
- Coordinate with Guardian on the D-002 IDOR check (cross-tenant data leak on the dashboard).

## What Guardian (D-002 / D-003) will do

- Review the F-D002-004 surface on the new metric cards (a project-A user shouldn't see project-B's queued count).
- Add the dashboard cross-tenant test to the D-003 deliverable.

## What Ops (E-001) will do (downstream)

- Add a `docker compose up` smoke that creates executions in various states and confirms the dashboard renders all 6 state badges.
- Update the `validate-infra.py` health check to curl the dashboard page and confirm it loads.
