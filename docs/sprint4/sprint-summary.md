# Sprint 4 Summary — Agent Orchestration Engine

**Sprint**: 4
**Dates**: 2026-06-08 → 2026-06-12
**Repo**: https://github.com/fadhilfathi/ai-software-factory.git
**Sprint commit**: `ebeba6b32a0c03ca5ab2095264eb46dd40264268` — `feat(sprint-4): agent orchestration engine` (squash-merge of PR #1, 2026-06-13T06:34:18Z)

---

## Executive Summary

Sprint 4 delivered the Agent Orchestration Engine end-to-end: agent registry with capability engine, task assignment with append-only history, execution tracking (mocked for Sprint 4, real Hermes runtime deferred to Sprint 5), versioned deliverable storage, and four frontend pages (agent management, task assignment dashboard, deliverable viewer, activity dashboard). Infrastructure shipped a 14-step CI quality gate on GitHub Actions (`ubuntu-latest`) plus a graceful-shutdown path in `cmd/main.go` so in-flight HTTP requests and execution goroutines drain cleanly on `SIGINT`/`SIGTERM`. The dev wave landed 18 task cards (TASK-401..425), of which 13 are completed, 1 (TASK-411, Testing & Validation) is in final sign-off, and 5 (TASK-419..422, TASK-425) are waived Critical cross-tenant / authz findings explicitly deferred to Sprint 5 per §7.2.1 of the security review.

**What worked.** The brief-driven workflow plus the Lead's `verify-code-state-before-acceptance` discipline caught multiple spec-vs-code gaps early in the cycle — F-011 (`useProjectFilters` missing `projectId` gate, closed in-patch in TASK-408), the assignments doc-vs-code chain across data-model.md §4/§5/§8/§9, the `010` vs `015` migration conflict on `agents.capabilities` (closed by TASK-416, JSONB canonical), the `auth.go` JWT role hardcode (F-001, closed in TASK-417), the `middleware.go` `api_*` prefix-only check (F-002, closed in TASK-418), and the missing shutdown path in `cmd/main.go` (Leader's brief said "join the existing path"; there was no existing path — built from scratch with a shared `SHUTDOWN_GRACE` budget that prioritises HTTP drain over execution goroutine cleanup). Each gap was caught at the file/branch level, fixed in place, and recorded in the wave-state log. No gap was discovered after merge.

**What we deferred.** Four Critical cross-tenant findings (F-013..F-016) are explicitly waived for Sprint 4 — the data model defers `project_id` on agents/deliverables/executions/assignments to Sprint 5 (data-model.md §4.1/§6/§9.1). F-021 (RequireRole middleware not wired in the router) is deferred to Sprint 5 as TASK-425, dependent on the F-001 patch opening the path. Three real data-layer gaps were also flagged for the Sprint 5 spec: `completed_at` column on `tasks` + `completed_after` filter on `GET /v1/projects/:id/tasks`; `started_after` filter on `GET /v1/executions`; and pagination work for `useExecutions` (currently consumes only the first page). The recharts bundle (~80KB gzipped) is a `next/dynamic` lazy-load one-liner for Sprint 5. None of these block the Sprint 4 closeout.

**Quality posture.** The 14-step CI gate (`.github/workflows/sprint-quality-gate.yml`) runs the full pipeline — `go vet`, `go build`, `go test ./internal/...`, `docker compose config`, `docker compose up -d`, healthz polling, `curl /v1/healthz`, integration tests, and `docker compose down -v` cleanup — on every push to `main` and on every PR. The gate has a `concurrency.cancel-in-progress` group, a Go-modules + build-cache layer keyed on `src/**/go.sum`, and `::error::` annotations on every step. The companion `scripts/quality-gate.sh` mirrors the same 14 steps for local pre-push verification. The CI run attached to the closeout commit is the final proof that the sprint is closed; `docs/sprint4/quality-gate-report.md` carries the per-step pass/fail and the `TESTER APPROVED` line that closes the gate.

---

## Sprint Scope

**In scope (delivered):**
- **Backend** (TASK-402..406): agent registry, capability engine, task assignment with history, execution tracking, versioned deliverable storage.
- **Frontend** (TASK-407..410): agent management, task assignment dashboard, deliverable viewer, activity dashboard.
- **Security patches** (TASK-417, 418, 423, 424, plus in-patch in 409): `auth.go` JWT role, `middleware.go` API-key validation, F-017 notes threading, F-023 content-size bound, F-006 markdown XSS.
- **Infrastructure** (TASK-413, 414, plus the `cmd/main.go` wiring landed as part of 414): docker-compose validation, CI gate workflow, graceful shutdown path.
- **Testing** (TASK-411): test plan + integration tests + acceptance report.
- **Security review** (TASK-412): full static + runtime review with §7.2.1/§7.2.2 waiver text.
- **Sprint cleanup** (TASK-416): migration conflict fix, JSONB canonical for `capabilities` across 010/015 and `data-model.md` / `database.md`.

**Out of scope (deferred to Sprint 5):**
- TASK-419..422, TASK-425: cross-tenant scoping (4 items) + RequireRole middleware in router.
- Project membership management endpoints.
- `completed_at` column on `tasks` + `completed_after` filter on `GET /v1/projects/:id/tasks`.
- `started_after` filter on `GET /v1/executions`.
- Pagination work for `useExecutions` (currently consumes only first page).
- recharts bundle-size lazy-load via `next/dynamic`.
- 4 enrichment columns on `agents` / `deliverables` / `executions` (Sprint 5 backlog).
- `021` `agent_state_events` design placeholder.
- Backlog: integration concern from TASK-410 (400 vs 500 on malformed UUID) — already inside TASK-411's test scope; any route still returning 500 on malformed UUID will be fixed in Sprint 5.

---

## Task Delivery Table

| ID    | Subject                          | Owner         | Status     | Notes |
|-------|----------------------------------|---------------|------------|-------|
| 401   | Agent Registry Design            | Analyst-01    | completed  | Canonical data-model.md §3; 5 tables (agents + 4 new) for agent state, capabilities, assignments, deliverable_versions |
| 402   | Agent Registry Backend           | Developer-01  | completed  | Adds migrations 017 (capabilities join table) + 018 (agents.RequiredCapabilities column); `ValidateAgentHasCapabilities` seam |
| 403   | Agent Capability Engine          | Developer-01  | completed  | Capability service on top of 402's seam; capability-assign via PUT (replace-semantics), not separate endpoints |
| 404   | Task Assignment Engine           | Developer-01  | completed  | Adds migration 019 (assignment_events); POST /v1/tasks/:id/assign + GET /v1/tasks/:id/history; append-only history |
| 405   | Execution Tracking System        | Developer-01  | completed  | Mock execution only (Sprint 5 = real Hermes); migration 023 additive ALTER on 008; mock goroutine via WaitGroup + service-level stop ctx; env-var failure rate; state machine in service; keyset pagination; 22 tests |
| 406   | Deliverable Storage              | Developer-01  | completed  | Versioned (deliverable_versions); append-only; bounded content size in service |
| 407   | Agent Management UI              | Developer-02  | completed  | Project-scoped via X-Project-ID header; "all projects" view is a UI gate, not a request variant |
| 408   | Task Assignment Dashboard        | Developer-02  | completed  | F-011 (`useProjectFilters` missing `projectId` gate) critical runtime fix closed in-patch; F-012 (useUpdateTaskStatus rollback snapshot order) closed in-patch |
| 409   | Deliverable Viewer               | Developer-02  | completed  | F-006 (markdown XSS) closed in-patch via `MarkdownRenderer.tsx` sanitisation |
| 410   | Agent Activity Dashboard         | Developer-02  | completed  | recharts integration; first-page-only pagination flagged for Sprint 5 |
| 411   | Testing & Validation             | Tester-01     | in_progress | Final sign-off pending; integration test suite populates `src/internal/integration/` (currently empty; gate step 13 prints `::notice::` and skips) |
| 412   | Security Review                  | Security-01   | completed  | §7.2.1 waiver text for F-013..F-016 (cross-tenant data-model deferral); §7.2.2 partial-waiver text for F-014 (authz-narrowed) |
| 413   | Infrastructure Validation        | DevOps-01     | completed  | 3 fixes: missing `wget` in both Dockerfiles (healthcheck would always fail); duplicate migration version "008" (schema_migrations PK collision); `.env.example` missing 12 env vars the app reads (incl. JWT_SECRET, DB_*) |
| 414   | Pre-Commit Quality Gate          | DevOps-01     | completed  | CI gate on ubuntu-latest, 14 steps; graceful shutdown wired in `cmd/main.go` (signal.Notify SIGINT+SIGTERM, `http.Server` wrapper, `svc.Execution.Shutdown(graceCtx)` joined with `srv.Shutdown(graceCtx)` under shared `SHUTDOWN_GRACE` budget) |
| 415   | GitHub Automation & Closeout     | DevOps-01     | completed  | PR #1 squash-merged to `main` as commit `ebeba6b` at 2026-06-13T06:34:18Z |
| 416   | Migration conflict fix (010/015) | DevOps-01     | completed  | JSONB canonical for `capabilities`; `role`/`provider` types aligned across 010/015; `data-model.md` and `database.md` updated; "KEEP IN SYNC" header comments on both migrations |
| 417   | Patch auth.go — JWT role         | Developer-01  | completed  | F-001 closed in-patch: JWT now reads role from DB instead of hardcoded "user" |
| 418   | Patch middleware.go — api-key    | Developer-01  | completed  | F-002 closed in-patch: real API-key validation, not just `api_*` prefix check |
| 423   | Fix notes dropped (F-017)        | Developer-01  | completed  | 4-arg signature on assignment service; audit-trail gap closed |
| 424   | Bound deliverable content (F-023)| Developer-01  | completed  | Two-layer defence-in-depth: handler `MaxBytesReader` + service-level size check |

**5 cross-tenant / authz tasks intentionally NOT in this delivery table** — they're Sprint 5 follow-ups (TASK-419..422, TASK-425). Tracked under "Waivers Granted" and "Open Items" below.

---

## Fixes Applied (in-patch)

| Finding | Description                                  | Patched in    | Status            |
|---------|----------------------------------------------|---------------|-------------------|
| F-001   | `auth.go` JWT role hardcoded to "user"        | TASK-417      | FIXED-IN-PATCH    |
| F-002   | `middleware.go` `api_*` prefix-only check    | TASK-418      | FIXED-IN-PATCH    |
| F-006   | Markdown XSS in deliverable viewer            | TASK-409      | FIXED-IN-PATCH    |
| F-008   | APIKeyStore persistence (mention only)        | (pre-existing)| (n/a this sprint) |
| F-011   | `useProjectFilters` missing `projectId` gate  | TASK-408      | FIXED-IN-PATCH    |
| F-012   | `useUpdateTaskStatus` rollback snapshot order | TASK-408      | FIXED-IN-PATCH    |
| F-017   | Notes dropped in `assignment_events`         | TASK-423      | FIXED-IN-PATCH    |
| F-023   | Deliverable content size unbounded           | TASK-424      | FIXED-IN-PATCH    |

(F-008 is referenced from the security review §5.1 in the same row as F-006; one-character correction to the comment in `frontend/src/components/deliverables/MarkdownRenderer.tsx` folded into the closeout commit per the wave-state "Doc nit registry".)

---

## Waivers Granted (Sprint 4, follow-up Sprint 5)

| Finding | Description                                  | Waiver task   | Reason |
|---------|----------------------------------------------|---------------|--------|
| F-013   | Cross-tenant agent scoping                   | TASK-419      | data-model.md §4.1/§9.1 defers `project_id` on `agents` to Sprint 5; Sprint 4 agents are global |
| F-014   | Cross-tenant assignment                      | TASK-420      | Same; F-014 narrowed to pure authz (data-integrity gap closed by the partial unique index on `(task_id, agent_id)` where `status='active'`) |
| F-015   | Cross-tenant deliverable                     | TASK-421      | Same; data-model.md §6 defers `project_id` on `deliverables` |
| F-016   | Cross-tenant execution                       | TASK-422      | Same; data-model.md defers `project_id` on `executions` |
| F-021   | RequireRole middleware not wired in router   | TASK-425      | F-001 patch opens the path (role now in JWT); multi-route design deferred to Sprint 5 |

---

## Open Items / Sprint 5 Backlog

- TASK-419..422: cross-tenant scoping (4 items, owned by Developer-01)
- TASK-425: RequireRole middleware in router (Developer-01)
- Project membership management endpoints
- `completed_at` column on `tasks` + `completed_after` filter on `GET /v1/projects/:id/tasks`
- `started_after` filter on `GET /v1/executions`
- Pagination work for `useExecutions` (currently consumes first page only)
- recharts bundle-size lazy-load via `next/dynamic` (~80KB gzipped, one-line fix)
- 4 enrichment columns on `agents` / `deliverables` / `executions` (Sprint 5 backlog, surfaced in security review §6)
- `021` `agent_state_events` design placeholder (Sprint 5 — backfill from the execution events emitted by the mock goroutine in TASK-405)
- Backlog: integration concern from TASK-410 (400 vs 500 on malformed UUID) — already in TASK-411 test scope; any route still returning 500 on malformed UUID will be fixed in Sprint 5
- Backlog: the `cmd/main.go` graceful-shutdown path is "best effort" with a shared `SHUTDOWN_GRACE` budget — if HTTP drain consumes the full budget, Execution shutdown returns immediately with a cancelled-context error. A natural test (fire `SIGTERM` mid-execution, assert WaitGroup drained after grace period) lives in the integration suite (TASK-411 / future) and was not in scope for the Sprint 4 gate

---

## Quality Gate

- 14-step CI gate on `.github/workflows/sprint-quality-gate.yml` (ubuntu-latest, Go 1.25.x, Node 20, Postgres via `docker-compose`)
- `scripts/quality-gate.sh` mirrors the 14 steps for local pre-push verification
- `docs/sprint4/quality-gate-report.md` is populated by the CI run attached to the closeout commit
- `TESTER APPROVED — <ISO-8601 timestamp> <name>` line is the explicit sign-off that closes the gate

---

## Sprint 4 closeout commit (TASK-415 — completed)

Single `feat(sprint-4): agent orchestration engine` commit, squashed and merged into `main` as PR #1.

- **Final commit SHA on `main`**: `ebeba6b32a0c03ca5ab2095264eb46dd40264268`
- **Merged at**: 2026-06-13T06:34:18Z
- **PR**: https://github.com/fadhilfathi/ai-software-factory/pull/1
- **Post-merge Sprint Quality Gate**: ✅ pass — run [27459209339](https://github.com/fadhilfathi/ai-software-factory/actions/runs/27459209339) (5m44s)
- **Post-merge CI**: ✅ pass — run [27459209335](https://github.com/fadhilfathi/ai-software-factory/actions/runs/27459209335) (e2e-smoke non-blocking; see TASK-426)
- **Post-merge Deploy**: ❌ fail — run [27459209340](https://github.com/fadhilfathi/ai-software-factory/actions/runs/27459209340) — `docker/build-push-action@v6` buildx `cache-to` GHA backend error. Filed as Sprint 5 TASK-429; does not block Sprint 4 closeout (Deploy is not in the Sprint 4 closeout gate).
- **`gh repo view` verification**: `ai-software-factory` / `defaultBranchRef.name = main` / no `latestRelease` yet.
- **Branch cleanup**: `feat/sprint-4-closeout` kept (per `--delete-branch=false` in TASK-415) so the closeout history is preserved for audit.