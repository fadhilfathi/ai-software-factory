# AI Software Factory — API Specification

## Base URL
```
Production: https://api.ai-software-factory.com/v1
Development: http://localhost:8080/v1
```

## Authentication

### JWT Token Flow
```
POST /auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword"
}

Response:
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "dGhpcyBpcyBhIHJlZnJl...",
  "expires_in": 86400
}
```

### API Key Authentication
```
Authorization: Bearer ak_1234567890abcdef
```

## Error Response Format
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request parameters",
    "details": [
      {
        "field": "name",
        "message": "Name is required"
      }
    ]
  },
  "request_id": "req_abc123"
}
```

## Pagination
```
GET /projects?page=1&limit=20&sort=created_at&order=desc

Response:
{
  "data": [...],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150,
    "pages": 8
  }
}
```

---

## Health Check

### GET /healthz
```
GET /v1/healthz

Response:
{
  "status": "ok"
}
```

---

## Projects

### Create Project
```
POST /v1/projects

Request:
{
  "name": "E-commerce Platform",
  "description": "Build a modern e-commerce platform with React frontend and Go backend...",
  "template": "web-app"
}

Response (201):
{
  "id": "3a1b2c3d-...",
  "name": "E-commerce Platform",
  "description": "Build a modern e-commerce platform...",
  "owner_id": "00000000-...",
  "status": "initializing",
  "template": "web-app",
  "progress": 0,
  "created_at": "2026-06-10T10:00:00Z",
  "updated_at": "2026-06-10T10:00:00Z"
}
```

Validation:
- `name` is required (non-empty)
- Default status: `initializing`

### List Projects
```
GET /v1/projects?status=active&page=1&limit=20

Response:
{
  "data": [
    {
      "id": "3a1b2c3d-...",
      "name": "E-commerce Platform",
      "status": "in_progress",
      "progress": 45,
      "owner_id": "00000000-...",
      "created_at": "2026-06-10T10:00:00Z",
      "updated_at": "2026-06-10T10:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 1,
    "pages": 1
  }
}
```

Query parameters:
- `status` — filter by `initializing`, `in_progress`, `completed`, `archived`
- `page` — page number (default: 1)
- `limit` — items per page (default: 20, max: 100)

### Get Project Details
```
GET /v1/projects/:id

Response:
{
  "id": "3a1b2c3d-...",
  "name": "E-commerce Platform",
  "description": "...",
  "owner_id": "00000000-...",
  "status": "in_progress",
  "template": "web-app",
  "progress": 45,
  "created_at": "2026-06-10T10:00:00Z",
  "updated_at": "2026-06-10T10:30:00Z"
}
```

### Update Project
```
PUT /v1/projects/:id

Request:
{
  "name": "E-commerce Platform v2",
  "description": "Updated description",
  "status": "in_progress"
}

Response:
{
  "id": "3a1b2c3d-...",
  "name": "E-commerce Platform v2",
  "description": "Updated description",
  "status": "in_progress",
  ...
}
```

All fields are optional. Only provided fields are updated.

### Delete Project
```
DELETE /v1/projects/:id

Response: 204 No Content
```

---

## Tasks

Tasks are scoped to a project. Created with status `backlog` by default.

### Create Task
```
POST /v1/projects/:projectId/tasks

Request:
{
  "title": "Implement user authentication API",
  "description": "Create JWT-based authentication with login, register, refresh endpoints",
  "priority": "high"
}

Response (201):
{
  "id": "b4c5d6e7-...",
  "project_id": "3a1b2c3d-...",
  "title": "Implement user authentication API",
  "description": "Create JWT-based authentication...",
  "status": "backlog",
  "priority": "high",
  "created_at": "2026-06-10T10:00:00Z",
  "updated_at": "2026-06-10T10:00:00Z"
}
```

Validation:
- `title` is required (non-empty)
- `priority` defaults to `medium`

### List Tasks for a Project
```
GET /v1/projects/:projectId/tasks?status=backlog&page=1&limit=20

Response:
{
  "data": [
    {
      "id": "b4c5d6e7-...",
      "project_id": "3a1b2c3d-...",
      "title": "Implement user authentication API",
      "status": "backlog",
      "priority": "high",
      "position": 0,
      "created_at": "2026-06-10T10:00:00Z",
      "updated_at": "2026-06-10T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 1,
    "pages": 1
  }
}
```

Query parameters:
- `status` — filter by task status
- `page` — page number (default: 1)
- `limit` — items per page (default: 20, max: 100)

### Get Task Details
```
GET /v1/tasks/:id

Response:
{
  "id": "b4c5d6e7-...",
  "project_id": "3a1b2c3d-...",
  "title": "Implement user authentication API",
  "description": "...",
  "status": "backlog",
  "priority": "high",
  "assignee_id": "",
  "position": 0,
  "created_at": "2026-06-10T10:00:00Z",
  "updated_at": "2026-06-10T10:00:00Z"
}
```

### Update Task
```
PUT /v1/tasks/:id

Request:
{
  "title": "Implement JWT auth",
  "description": "Updated scope description",
  "priority": "critical",
  "assignee_id": "agent_001"
}

Response:
{
  "id": "b4c5d6e7-...",
  "title": "Implement JWT auth",
  "priority": "critical",
  "assignee_id": "agent_001",
  ...
}
```

All fields are optional.

### Delete Task
```
DELETE /v1/tasks/:id

Response: 204 No Content
```

---

## Kanban Status Transitions

### Update Task Status
```
PATCH /v1/tasks/:id/status

Request:
{
  "status": "in_progress"
}

Response:
{
  "id": "b4c5d6e7-...",
  "status": "in_progress",
  "updated_at": "2026-06-10T10:30:00Z",
  ...
}

Error (422) on invalid transition:
{
  "error": {
    "code": "INVALID_TRANSITION",
    "message": "Cannot transition from backlog to done"
  }
}
```

### Status Transition Rules

```
backlog    ──→ ready, blocked
ready      ──→ in_progress, blocked
in_progress ──→ review, blocked
review     ──→ done, blocked
done       ──→ blocked
blocked    ──→ backlog, ready, in_progress, review, done
```

| From \ To | backlog | ready | in_progress | review | done | blocked |
|-----------|---------|-------|-------------|--------|------|---------|
| backlog   | -       | ✔     | ✘           | ✘      | ✘    | ✔       |
| ready     | ✘       | -     | ✔           | ✘      | ✘    | ✔       |
| in_progress | ✘     | ✘     | -           | ✔      | ✘    | ✔       |
| review    | ✘       | ✘     | ✘           | -      | ✔    | ✔       |
| done      | ✘       | ✘     | ✘           | ✘      | -    | ✔       |
| blocked   | ✔       | ✔     | ✔           | ✔      | ✔    | -       |

---

## Shared Response Shapes

### PaginatedResponse
```json
{
  "data": [ ... ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150,
    "pages": 8
  }
}
```

### Project
| Field | Type | Description |
|-------|------|-------------|
| `id` | string (UUID) | Unique identifier |
| `name` | string | Project name |
| `description` | string | Project description (optional) |
| `owner_id` | string (UUID) | Owner user ID |
| `status` | string | `initializing`, `in_progress`, `completed`, `archived` |
| `template` | string | Project template (optional) |
| `progress` | number | Progress percentage (0-100) |
| `created_at` | string (ISO 8601) | Creation timestamp |
| `updated_at` | string (ISO 8601) | Last update timestamp |

### Task
| Field | Type | Description |
|-------|------|-------------|
| `id` | string (UUID) | Unique identifier |
| `project_id` | string (UUID) | Parent project ID |
| `title` | string | Task title |
| `description` | string | Task description (optional) |
| `status` | string | `backlog`, `ready`, `in_progress`, `review`, `done`, `blocked` |
| `priority` | string | `low`, `medium`, `high`, `critical` |
| `assignee_id` | string (UUID) | Assigned agent/user ID (optional) |
| `position` | number | Display order within column |
| `created_at` | string (ISO 8601) | Creation timestamp |
| `updated_at` | string (ISO 8601) | Last update timestamp |

### Error Response
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validation failed",
    "details": [
      { "field": "name", "message": "Name is required" }
    ]
  },
  "request_id": "req_abc123"
}
```

Error codes: `VALIDATION_ERROR`, `UNAUTHORIZED`, `NOT_FOUND`, `CONFLICT`, `INTERNAL_ERROR`, `INVALID_TRANSITION`, `INVALID_JSON`

HTTP status codes: `400`, `401`, `404`, `409`, `422`, `500`

---

## Agents

The Agent Registry is the canonical store for long-lived, role-bearing
execution units inside a project. The wire contract below is the source of
truth for the public API; for deeper rationale see
`docs/sprint4/agent-orchestration-design.md` and `docs/sprint4/data-model.md`.

All agent routes are **project-scoped**: the project is identified by the
`X-Project-ID` request header. Cross-project reads are rejected with
`403 CROSS_TENANT_BLOCKED` (F-013).

### Fields

| Field             | Type        | Notes                                                                                                  |
|-------------------|-------------|--------------------------------------------------------------------------------------------------------|
| `id`              | UUID        | Primary key, server-generated.                                                                         |
| `project_id`      | UUID        | FK → `projects.id`. Immutable for the life of the row.                                                |
| `name`            | string      | 1-80 chars. Unique per `project_id` (`UNIQUE(project_id, name)`).                                     |
| `role`            | string      | Free-form role label (e.g. `Backend Developer`, `Security Reviewer`). 1-80 chars. Not an enum.        |
| `status`          | enum        | Lifecycle state. See "Status" below.                                                                   |
| `capabilities`    | string[]    | Capability names this agent can perform. Validated against the `capabilities` catalog. Min 1.          |
| `last_active_at`  | timestamptz | `NULL` until first activation.                                                                         |
| `metadata`        | jsonb       | Free-form: model name, version, tool allow-list, notes. Default `{}`.                                  |
| `version`         | int         | Server-maintained. Used for optimistic concurrency on `PUT`. Starts at 1.                              |
| `created_at`      | timestamptz | Server-set.                                                                                            |
| `updated_at`      | timestamptz | Server-maintained.                                                                                     |
| `retired_at`      | timestamptz | Set when the agent enters `retired`. Never cleared.                                                    |

### Status

Six values, matching the `agents_status_chk` CHECK constraint in
`db/migrations/016_agent_registry.sql` and `model.AllAgentStatuses`:

| State          | Assignable? | Notes                                                                       |
|----------------|-------------|-----------------------------------------------------------------------------|
| `initializing` | No          | Transient. Set at row insert. Max 30 s before first heartbeat.              |
| `idle`         | Yes         | Default steady state.                                                       |
| `busy`         | No          | Holds 0..N executions (typically 1). Set on successful task assignment.     |
| `paused`       | No          | Mid-execution pause; resumable.                                             |
| `error`        | No          | Agent is alive but last task failed. Operator can clear -> `idle` or retire. |
| `retired`      | No          | Terminal / soft-deleted. Excluded from listings by default.                  |

Transitions and the events that produce them are documented in
`agent-orchestration-design.md` section 2.1.

### Create Agent

```
POST /v1/agents

Request:
{
  "name": "Code Assistant",
  "role": "Backend Developer",
  "capabilities": ["coding", "testing"],
  "metadata": {
    "model": "gpt-4",
    "provider": "openai",
    "notes": "Owns the API package."
  }
}

Response (201):
{
  "id": "a1b2c3d4-...",
  "project_id": "3a1b2c3d-...",
  "name": "Code Assistant",
  "role": "Backend Developer",
  "status": "initializing",
  "capabilities": ["coding", "testing"],
  "metadata": { "model": "gpt-4", "provider": "openai", "notes": "Owns the API package." },
  "last_active_at": null,
  "version": 1,
  "created_at": "2026-06-14T10:00:00Z",
  "updated_at": "2026-06-14T10:00:00Z",
  "retired_at": null
}
```

Validation:
- `name` is required, 1-80 chars, unique per `project_id`.
- `role` is required, 1-80 chars, free-form.
- `capabilities` is required, >= 1 element, every value must exist in the
  `capabilities` catalog (see Capabilities / A-002). The catalog seeds
  `architecture`, `coding`, `testing`, `security`, `devops`. `leadership` is
  cataloged but reserved for the Leader agent and is **not** a valid value on
  this endpoint.
- `metadata` is optional; default `{}`.
- `status` is **not** accepted on create - new agents always start in
  `initializing`.
- `project_id` is taken from the `X-Project-ID` header; agents created with a
  missing or empty project header are rejected with `400 MISSING_PROJECT`.

Errors:
- `400 VALIDATION_ERROR` - name/role/capabilities validation failed.
- `400 MISSING_PROJECT` - `X-Project-ID` header is missing or empty.
- `409 NAME_CONFLICT` - an agent with this `name` already exists in the
  project.
- `413 CAPABILITY_OVERFLOW` - `capabilities` has more than 32 entries.
- `422 UNKNOWN_CAPABILITY` - at least one entry is not in the catalog.

### List Agents

```
GET /v1/agents?status=idle&role=Backend%20Developer&cursor=&limit=20&include_retired=false
```

Response (200):
```
{
  "data": [
    {
      "id": "a1b2c3d4-...",
      "project_id": "3a1b2c3d-...",
      "name": "Code Assistant",
      "role": "Backend Developer",
      "status": "idle",
      "capabilities": ["coding", "testing"],
      "last_active_at": "2026-06-14T10:05:00Z",
      "version": 3,
      "created_at": "2026-06-14T10:00:00Z",
      "updated_at": "2026-06-14T10:05:00Z"
    }
  ],
  "next_cursor": "WyJhMDFiMmMzZC04...",
  "has_more": false
}
```

Query parameters:
- `status` - optional. One of `initializing, idle, busy, paused, error,
  retired`. Invalid value returns `400 VALIDATION_ERROR`.
- `role` - optional. Case-insensitive exact match.
- `capability` - optional, repeatable (`?capability=coding&capability=testing`).
  Filters to agents that declare **all** listed capabilities (AND, not OR).
- `cursor` - optional. Opaque string from a prior response's `next_cursor`.
  Omit on the first page.
- `limit` - optional, default 20, max 100. `400 VALIDATION_ERROR` outside the
  range.
- `include_retired` - optional, default `false`. When `true`, retired agents
  appear in `data` and `status=retired` is implicitly allowed as a filter.

Pagination:
- **Cursor-based.** The cursor encodes the last row's `(updated_at, id)` key
  set; clients must treat it as opaque. There is no `total` count on the
  response (the registry does not maintain a count, to keep writes cheap).
  Use `has_more` to drive paging termination.
- This differs from the Projects section, which uses page-based pagination.
  The two patterns coexist; clients should not assume one style across
  sections.

### Get Agent

```
GET /v1/agents/:id
```

Response (200) - full agent object (same shape as the create response above).

Errors:
- `400 INVALID_ID` - `id` is not a valid UUID.
- `403 CROSS_TENANT_BLOCKED` - the agent exists but belongs to a different
  `project_id` than the `X-Project-ID` header.
- `404 NOT_FOUND` - no agent with this `id`.

### Update Agent

```
PUT /v1/agents/:id

Request:
{
  "name": "Senior Code Assistant",
  "role": "Senior Backend Developer",
  "status": "paused",
  "capabilities": ["coding", "testing", "security"],
  "metadata": { "model": "gpt-4-turbo" },
  "version": 3
}

Response (200):
{
  "id": "a1b2c3d4-...",
  "...": "full agent object with version bumped to 4"
}
```

Validation:
- All fields are optional. At least one updatable field must be present.
- `name` and `role`, if present, are re-validated against the same rules as
  create.
- `capabilities`, if present, replaces the existing list (it is not a
  merge). Must still be >= 1, every value in the catalog, and <= 32 entries.
- `status`, if present, must be a valid transition for the current state
  (see `agent-orchestration-design.md` section 2.1). Invalid transitions
  return `422 INVALID_TRANSITION`.
- `version` is **required**. The server compares the body's `version` to
  the row's current `version`; mismatch returns `409 VERSION_CONFLICT`.
  On success the row's `version` is incremented by 1.
- `project_id` and `id` are immutable and ignored if supplied.

Errors:
- `400 VALIDATION_ERROR` - name/role/capabilities validation failed.
- `403 CROSS_TENANT_BLOCKED` - agent belongs to a different project.
- `404 NOT_FOUND`.
- `409 VERSION_CONFLICT` - `version` in body does not match row.
- `422 INVALID_TRANSITION` - `status` change is not legal from the current
  state.

### Delete Agent (Soft Delete)

```
DELETE /v1/agents/:id

Response: 204 No Content
```

Behavior:
- Sets `status = 'retired'` and `retired_at = NOW()`. The row is **preserved**.
- Existing assignments on this agent continue to run to completion (TASK-404
  contract). New assignments are rejected by the assignment engine (busy/idle
  check).
- Retired agents are excluded from `GET /v1/agents` and `GET /v1/agents/:id`
  by default; use `?include_retired=true` on list to reveal them.
- Idempotent: a second `DELETE` on an already-retired agent also returns 204.

Errors:
- `403 CROSS_TENANT_BLOCKED`.
- `404 NOT_FOUND` - only when the agent truly does not exist; retired agents
  in the same project return 204 (idempotent).

### Capability Endpoints (A-002)

The following endpoints are wired and tested as part of the Capability System
(A-002) and are documented in detail in the Capabilities section. They are
listed here for discoverability:

- `GET /v1/agents/:id/capabilities` - list capabilities declared by a single
  agent.
- `GET /v1/capabilities` - list the full capabilities catalog.

### Sprint 7 (Deferred)

The following endpoints are **not** implemented in A-001 and are deferred to
Sprint 7. They are listed here so the contract surface is not lost:

- `POST /v1/agents/:id/heartbeat` - liveness ping; updates `last_active_at`
  and may transition `initializing -> idle`. Requires the `agent_state_events`
  event-sourcing table (see `data-model.md` section 6) to land first.
- `GET /v1/agents/:id/events` - lifecycle event log for an agent.
  Backed by `agent_state_events`.
- `GET /v1/agents/conflicts` - agents with multiple active assignments or
  status mismatches. Depends on the Assignment Engine (A-003) shipping the
  "active run" signal.

These will be added by the A-002 Capability System (heartbeat piggybacks on
the capability-set read path) and A-003 Assignment Engine (events/conflicts)
in Sprint 7.

---
## Task Assignment

### Assign Task to Agent
```
POST /v1/tasks/:taskId/assign

Request:
{
  "agent_id": "a1b2c3d4-..."
}

Response (200):
{
  "execution_id": "e5f6a7b8-...",
  "task_id": "task-uuid-here",
  "agent_id": "a1b2c3d4-...",
  "status": "running",
  "started_at": "2026-06-12T10:00:00Z"
}
```

Validation:
- Agent must be `idle`.
- Agent capabilities must match task requirements.
- Updates task status to `in_progress` and agent status to `working`.

---

## Executions

An execution records the lifecycle of an agent working on a task.

### List Executions
```
GET /v1/executions?task_id=<uuid>&agent_id=<uuid>&page=1&limit=20
```

### Get Execution Details
```
GET /v1/executions/:id
```

### Update Execution Status
```
PATCH /v1/executions/:id/status

Request:
{
  "status": "completed"
}
```

Status Transitions:
- `pending` → `running`
- `running` → `completed`, `failed`

---

## Deliverables

Deliverables are artifacts produced by an agent while working on a task.

### Create Deliverable
```
POST /v1/deliverables

Request:
{
  "task_id": "task-uuid-here",
  "agent_id": "a1b2c3d4-...",
  "title": "Authentication API Design",
  "content": "..."
}

Response (201):
{
  "id": "d1e2f3a4-...",
  "version": 1,
  "created_at": "2026-06-12T10:00:00Z",
  ...
}
```

### List Deliverables
```
GET /v1/deliverables?task_id=<uuid>
GET /v1/deliverables?agent_id=<uuid>
```

### Update Deliverable
```
PUT /v1/deliverables/:id

Request:
{
  "content": "Updated content..."
}
```
Version auto-increments on each update.

---

## Code

The Code Service manages the codebase, generation requests, and Git operations.

### Generate Code
```
POST /v1/code/generate

Request:
{
  "project_id": "proj-uuid",
  "task_id": "task-uuid",
  "specification": "Implement JWT authentication with Gin...",
  "files": ["internal/handler/auth.go"]
}

Response (202):
{
  "id": "code-gen-uuid",
  "status": "generating",
  "execution_id": "exec-uuid"
}
```

### List Project Files
```
GET /v1/code/:projectId/files
```

### Get File Content
```
GET /v1/code/:projectId/files/*path

Response:
{
  "path": "internal/handler/auth.go",
  "content": "package handler...",
  "language": "go",
  "size": 2048,
  "last_modified": "2026-06-12T10:45:00Z"
}
```

### Create Commit
```
POST /v1/code/:projectId/commits

Request:
{
  "branch": "feature/auth",
  "message": "feat: implement JWT authentication",
  "files": [
    { "path": "internal/handler/auth.go", "content": "..." }
  ]
}
```

### Get Diff
```
GET /v1/code/:projectId/diff?base=sha1&head=sha256
```

### Get Static Analysis
```
GET /v1/code/:projectId/analysis
```
Returns complexity, linting, and maintainability metrics.

---

## Reviews

The Review Service manages the code review lifecycle and quality gates.

### Create Review Request
```
POST /v1/reviews

Request:
{
  "project_id": "proj-uuid",
  "commit_sha": "abc123def456",
  "reviewer_type": "automated"
}

Response (201):
{
  "id": "review-uuid",
  "status": "in_progress",
  "reviewer_id": "agent-uuid"
}
```

### Get Review Details
```
GET /v1/reviews/:id

Response:
{
  "id": "review-uuid",
  "status": "completed",
  "result": "approved",
  "score": 85,
  "issues": [
    {
      "severity": "warning",
      "file": "internal/handler/auth.go",
      "line": 42,
      "message": "Consider adding rate limiting",
      "suggestion": "..."
    }
  ],
  "metrics": {
    "complexity": 8,
    "test_coverage": 92
  }
}
```

### Add Review Comment
```
POST /v1/reviews/:id/comments

Request:
{
  "file": "internal/handler/auth.go",
  "line": 10,
  "content": "Good use of the Gin context."
}
```

### List Review Comments
```
GET /v1/reviews/:id/comments
```

### Update Review Status
```
PATCH /v1/reviews/:id/status

Request:
{
  "status": "approved"
}
```

### List Project Reviews
```
GET /v1/reviews/project/:projectId
```

---

## Deployments

### Trigger Deployment
```
POST /v1/deployments

Request:
{
  "project_id": "proj_abc123",
  "environment": "staging",
  "branch": "main"
}

Response (202):
{
  "id": "deploy_001",
  "status": "queued",
  "environment": "staging",
  "estimated_time": 600
}
```

### Get Deployment Status
```
GET /v1/deployments/:id

Response:
{
  "id": "deploy_001",
  "status": "completed",
  "environment": "staging",
  "url": "https://staging.ai-factory-project.com",
  "started_at": "2026-06-10T11:00:00Z",
  "completed_at": "2026-06-10T11:08:00Z",
  "steps": [
    { "name": "build", "status": "completed", "duration": 120 },
    { "name": "test", "status": "completed", "duration": 180 },
    { "name": "deploy", "status": "completed", "duration": 60 }
  ]
}
```

### Rollback Deployment
```
POST /v1/deployments/:id/rollback

Response:
{
  "id": "deploy_002",
  "status": "rolling_back",
  "rollback_from": "deploy_001",
  "rollback_to": "deploy_000"
}
```

---

## Users

### Register
```
POST /v1/users/register

Request:
{
  "email": "user@example.com",
  "password": "securepassword123",
  "name": "John Doe"
}

Response (201):
{
  "id": "user_001",
  "email": "user@example.com",
  "name": "John Doe",
  "created_at": "2026-06-10T10:00:00Z"
}
```

### Get Profile
```
GET /v1/users/me

Response:
{
  "id": "user_001",
  "email": "user@example.com",
  "name": "John Doe",
  "role": "admin",
  "teams": ["team_001"],
  "projects": ["proj_abc123"]
}
```

---

## Webhooks

### Register Webhook
```
POST /v1/webhooks

Request:
{
  "url": "https://your-server.com/webhook",
  "events": ["project.completed", "deploy.failed"],
  "secret": "your_webhook_secret"
}

Response (201):
{
  "id": "wh_001",
  "url": "https://your-server.com/webhook",
  "events": ["project.completed", "deploy.failed"],
  "active": true,
  "created_at": "2026-06-10T10:00:00Z"
}
```

### Webhook Payload
```json
{
  "event": "project.completed",
  "timestamp": "2026-06-10T12:00:00Z",
  "data": {
    "project_id": "proj_abc123",
    "name": "E-commerce Platform",
    "status": "completed",
    "duration_days": 14
  },
  "signature": "sha256=abc123..."
}
```

---

## Rate Limits

| Plan | Requests/Hour | Concurrent Agents |
|------|--------------|-------------------|
| Free | 100 | 1 |
| Pro | 1,000 | 5 |
| Enterprise | 10,000 | 20 |

Rate limit headers:
```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1718035200
```

---

## Versioning

API version is included in the URL path: `/v1/`, `/v2/`

Breaking changes require a new major version. Non-breaking additions (new endpoints, new fields) are added to the current version.

Deprecation notice: `Sunset: Sat, 01 Jan 2027 00:00:00 GMT`
