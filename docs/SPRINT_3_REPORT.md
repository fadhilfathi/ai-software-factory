# Sprint 3 Completion Report

## Overview
Sprint 3 delivered the core project management and Kanban workflow module. All 12 tasks (TASK-201 through TASK-212) are complete. The sprint established a working backend CRUD API for projects and tasks, a PostgreSQL-backed store with in-memory fallback, a Kanban board UI with drag-and-drop, and full frontend-backend integration via React Query.

---

## Participants

| Role | Contribution |
|------|-------------|
| Data Engineer | Domain models, PostgreSQL schema, migrations |
| Backend Developer | Project + Task API handlers, service layer, store layer |
| Frontend Developer | Project pages, Kanban board, drag-and-drop, task list |
| QA | Test strategy, acceptance criteria validation |
| Security | Threat model review, auth middleware audit |
| DevOps | Docker Compose validation, environment configuration |
| Tech Writer | Sprint report, setup guide, API spec, architecture updates |

---

## Task Reports

### TASK-201: Project Domain Model
1. **Objective**: Define the `Project` model with UUID primary key, status lifecycle, and typed fields.
2. **Deliverables**: `src/internal/model/project.go` with `ProjectStatus` enum (`initializing`, `in_progress`, `completed`, `archived`) and `Project` struct.
3. **Dependencies**: TASK-101 (Repository Structure).
4. **Status**: Complete. Model includes `ID`, `Name`, `Description`, `OwnerID`, `Status`, `Template`, `Progress`, `CreatedAt`, `UpdatedAt`.

### TASK-202: Task Domain Model
1. **Objective**: Define the `Task` model with Kanban status states and priority levels.
2. **Deliverables**: `src/internal/model/task.go` with `TaskStatus` enum (`backlog`, `ready`, `in_progress`, `review`, `done`, `blocked`) and `TaskPriority` enum (`low`, `medium`, `high`, `critical`).
3. **Dependencies**: TASK-201.
4. **Status**: Complete.

### TASK-203: Project API (CRUD)
1. **Objective**: Implement full CRUD endpoints for projects.
2. **Deliverables**: `POST /v1/projects`, `GET /v1/projects`, `GET /v1/projects/:id`, `PUT /v1/projects/:id`, `DELETE /v1/projects/:id`.
3. **Dependencies**: TASK-201, TASK-102.
4. **Status**: Complete. Endpoints return `201 Created` for create, `200 OK` for get/update/list, `204 No Content` for delete. Paginated responses follow the `{ data, pagination }` contract.

### TASK-204: Task API (CRUD)
1. **Objective**: Implement full CRUD endpoints for tasks scoped to a project.
2. **Deliverables**: `POST /v1/projects/:projectId/tasks`, `GET /v1/projects/:projectId/tasks`, `GET /v1/tasks/:id`, `PUT /v1/tasks/:id`, `DELETE /v1/tasks/:id`.
3. **Dependencies**: TASK-202, TASK-203.
4. **Status**: Complete. Task creation defaults status to `backlog` and priority to `medium`.

### TASK-205: Kanban Status Transition API
1. **Objective**: Implement a state-machine-based status transition endpoint for Kanban workflow.
2. **Deliverables**: `PATCH /v1/tasks/:id/status` with validation against `taskStatusTransitions` map in service layer.
3. **Dependencies**: TASK-204.
4. **Status**: Complete. Transition rules: `backlog → ready/blocked`, `ready → in_progress/blocked`, `in_progress → review/blocked`, `review → done/blocked`, `done → blocked`, `blocked → any state`. Invalid transitions return `422 Unprocessable Entity` with code `INVALID_TRANSITION`.

### TASK-206: Project Management UI
1. **Objective**: Build project list, detail, create, and edit pages.
2. **Deliverables**: `frontend/src/app/projects/` containing list (`page.tsx`), detail (`[id]/page.tsx`), create (`new/page.tsx`), edit (`[id]/edit/page.tsx`) pages.
3. **Dependencies**: TASK-104, TASK-203.
4. **Status**: Complete. Pages use `useProject`, `useProjects`, `useCreateProject`, `useUpdateProject`, `useDeleteProject` hooks. Project list supports filtering by status and pagination. Detail page shows task summary counts grouped by status.

### TASK-207: Kanban Board UI
1. **Objective**: Build an interactive Kanban board with drag-and-drop.
2. **Deliverables**: `frontend/src/components/kanban/` containing `KanbanBoard.tsx`, `KanbanColumn.tsx`, `TaskCard.tsx`, `AddTaskDialog.tsx`. Uses `@dnd-kit/core` and `@dnd-kit/sortable` for drag-and-drop.
3. **Dependencies**: TASK-104, TASK-205.
4. **Status**: Complete. Six columns (Backlog, Ready, In Progress, Review, Done, Blocked). Tasks are draggable between columns using `PointerSensor` with 5px activation distance. `DragOverlay` provides visual feedback. Add-task dialog for inline creation.

### TASK-208: API Integration (Frontend)
1. **Objective**: Wire frontend React Query hooks to backend API endpoints.
2. **Deliverables**: `frontend/src/lib/api.ts` (fetch wrapper with auth interceptor, auto-refresh on 401), `frontend/src/lib/hooks.ts` (React Query hooks), `frontend/src/lib/types.ts` (TypeScript types matching Go models), `frontend/src/lib/queryKeys.ts` (centralized cache key factory).
3. **Dependencies**: TASK-203, TASK-204, TASK-205, TASK-206, TASK-207.
4. **Status**: Complete. Optimistic updates for task status change with rollback on error.

### TASK-209: QA Validation
1. **Objective**: Validate all Sprint 3 deliverables against acceptance criteria.
2. **Deliverables**: Test strategy review, acceptance criteria check on project/task CRUD and Kanban transitions.
3. **Dependencies**: All TASK-201 through TASK-208.
4. **Status**: Complete. All acceptance criteria met.
5. **Key Findings**: Paginated responses correctly wrapped, status transitions properly rejected, CORS configured for local dev.

### TASK-210: Security Review
1. **Objective**: Review auth middleware, input validation, and API security.
2. **Deliverables**: Auth middleware audit, validation layer review, threat model update.
3. **Dependencies**: All TASK-201 through TASK-208.
4. **Status**: Complete. Auth middleware properly protects private routes. Input validation rejects empty project names and malformed IDs. Public routes whitelist maintained.

### TASK-211: Docker Validation
1. **Objective**: Verify Docker Compose stack starts correctly with all Sprint 3 additions.
2. **Deliverables**: Validated `docker-compose.yml` with backend, frontend, and PostgreSQL services.
3. **Dependencies**: TASK-203, TASK-204, TASK-205, TASK-206, TASK-207.
4. **Status**: Complete. Services build and pass health checks. In-memory store works when `DB_HOST` is not set.

### TASK-212: Sprint Documentation
1. **Objective**: Generate Sprint 3 report, update setup guide, API spec, and architecture docs.
2. **Deliverables**: This report, `docs/setup-guide.md`, updated `docs/api-spec.md`, updated `docs/architecture.md`.
3. **Dependencies**: All TASK-201 through TASK-211.
4. **Status**: Complete.

---

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Database Strategy | PostgreSQL with in-memory fallback | When `DB_HOST` env var is set, use PostgreSQL via `pgx/v5`; otherwise fall back to in-memory maps. Enables development without Docker. |
| Kanban State Machine | Service layer with explicit transition map | `taskStatusTransitions` map in `src/internal/service/service.go` defines allowed transitions. Controller-free, testable state logic. |
| Drag-and-Drop Library | @dnd-kit | Lightweight, accessible, React-first. `PointerSensor` with 5px activation distance prevents accidental drags. |
| API Versioning | URL path prefix `/v1` | Consistent with existing pattern. All project/task endpoints under `/v1`. |
| Pagination | Page/Limit with total count | `PaginatedResponse` envelope returned by all list endpoints with `page`, `limit`, `total`, `pages`. |

---

## Architecture (Sprint 3 Additions)

```
┌─────────────────────┐     ┌──────────────────────┐
│   Frontend (Next.js) │     │   Backend (Go/Gin)    │
│                      │     │                        │
│  projects/page.tsx   │────▶│  /v1/projects          │
│  projects/[id]/*     │     │  /v1/projects/:id      │
│  projects/[id]/board │     │  /v1/projects/:id/tasks│
│                      │     │  /v1/tasks/:id         │
│  React Query hooks   │     │  /v1/tasks/:id/status  │
│  (lib/hooks.ts)      │     │                        │
│                      │     │  ┌──────────────────┐  │
│  @dnd-kit Kanban     │     │  │  Service Layer    │  │
│  (components/kanban/)│     │  │  project.go       │  │
└─────────────────────┘     │  │  task.go           │  │
                            │  └────────┬─────────┘  │
                            │           │             │
                            │  ┌────────┴─────────┐  │
                            │  │  Store Layer      │  │
                            │  │  memory.go (fallback)│
                            │  │  postgres/        │  │
                            │  │  ├ project_store   │  │
                            │  │  └ task_store      │  │
                            │  └──────────────────┘  │
                            └────────────────────────┘
```

---

**Sprint Status**: SUCCESS — All 12 tasks completed, all acceptance criteria met.
