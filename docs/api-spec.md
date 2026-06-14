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

## The Capability System (A-002)

The **capability system** is the canonical registry of skills an Agent can declare, the seam that matches tasks to agents by those skills, and the default skill profile each role ships with. It is implemented in A-002 and replaces the scattered capability references that previously lived inside the §Agents section.

The catalog is **global** (not per-project): the same nine capability names are valid in every project so the UI can render consistent filters and the assignment engine can match deterministically.

### The Capability Catalog

The catalog is a fixed set of nine names seeded at boot by migration `016_seed_capability_catalog`. Operators can extend the catalog via `POST /v1/capabilities` (admin-only; deferred to Sprint 7 — see below).

| Name                  | Display Name         | Purpose                                                                        |
|-----------------------|----------------------|--------------------------------------------------------------------------------|
| `architecture`        | Architecture         | Produces system design, ADRs, dependency choices.                              |
| `coding`              | Coding               | Writes source code and the tests that live in the source tree.                 |
| `testing`             | Testing              | Runs the test suite, reports coverage, files bug reports.                      |
| `security`            | Security             | Threat modeling, code audit, secret scanning.                                  |
| `devops`              | DevOps               | Builds, deploys, infra-as-code, monitoring.                                    |
| `documentation`       | Documentation        | Writes user-facing docs, READMEs, API references, runbooks.                    |
| `project_management`  | Project Management   | Plans, decomposes work, manages dependencies, tracks progress.                 |
| `data_engineering`    | Data Engineering     | Designs schemas, builds data pipelines, analytics.                             |
| `leadership`          | Leader               | **Reserved.** Owns the assignment workflow; never appears on a task constraint.|

### Assignable vs Reserved

Eight of the nine catalog capabilities are **assignable** — they may appear in a task's `required_capabilities` list and the validation seam will match agents against them. `leadership` is the one exception: it is reserved for the Lead agent role and is rejected by the validation seam with `CAPABILITY_NOT_ASSIGNABLE` if it appears on a task.

```go
// model/capability.go
func AssignableCapabilities() []Capability {
    return []Capability{
        CapArchitecture, CapCoding, CapTesting, CapSecurity, CapDevOps,
        CapDocumentation, CapProjectMgmt, CapDataEngineering,
    }
}
```

### List Catalog

```
GET /v1/capabilities

Response (200):
{
  "data": [
    {
      "name":         "architecture",
      "display_name": "Architecture",
      "category":     "architecture",
      "description":  "Produces system design, ADRs, dependency choices."
    },
    {
      "name":         "leadership",
      "display_name": "Leader",
      "category":     "leadership",
      "description":  "Reserved. Owns the assignment workflow.",
      "reserved":     true
    },
    ...
  ]
}

Errors:
- 401 UNAUTHENTICATED — missing or invalid bearer token.
```

The response is the canonical row view (`model.AgentCapabilityView`). The optional `reserved` flag is `true` only for `leadership`; the other eight omit it. The list is sorted by `name` ascending and is not paginated (catalog size is bounded to the seed set plus any custom capabilities added at runtime).

### List Agent Capabilities

```
GET /v1/agents/:id/capabilities

Response (200):
{
  "data": {
    "agent_id":     "a1b2c3d4-...",
    "agent_type":   "developer",
    "role":         "Backend Developer",
    "capabilities": [
      "coding", "testing", "documentation"
    ],
    "is_assignable": true
  }
}

Errors:
- 401 UNAUTHENTICATED
- 403 CROSS_TENANT_BLOCKED — :id belongs to a different project.
- 404 AGENT_NOT_FOUND
```

The `:id` path parameter is the agent UUID. The response includes `is_assignable: false` only if the agent is the Leader (so callers can render a banner saying " this agent cannot be assigned a task" without checking the role string client-side). For full agent CRUD see the §Agents section; this endpoint is a thin read-only slice.

### The Validation Seam (TASK-403)

When a client submits a task assignment that includes a `required_capabilities` list (via `POST /v1/tasks/:taskId/assign` — see §Task Assignment), the service layer validates the list against the catalog and the assignable set before attempting to match an agent.

```
Rejection responses (409 Conflict):

  required_capabilities contains a name that is not in the catalog at all:
  {
    "error": "CAPABILITY_NOT_IN_CATALOG",
    "message": "Capability 'rocket-science' is not in the capability catalog.",
    "details": { "unknown_capability": "rocket-science" }
  }

  required_capabilities contains 'leadership':
  {
    "error": "CAPABILITY_NOT_ASSIGNABLE",
    "message": "Capability 'leadership' is reserved and may not appear on a task.",
    "details": { "reserved_capability": "leadership" }
  }

  No agent in the project has all the required capabilities:
  {
    "error": "CAPABILITY_MISMATCH",
    "message": "No agent in this project has all required capabilities.",
    "details": {
      "required_capabilities": ["data_engineering", "coding"],
      "candidates": [
        { "agent_id": "...", "missing": ["data_engineering"] }
      ]
    }
  }
```

The seam lives in `service.NewAssignmentService().validateCapabilities()`. The three error codes are mapped to `409 Conflict` because the request was syntactically valid but cannot be satisfied with the current project roster — the client may resolve the conflict by adjusting the task, adding an agent, or splitting the work.

### Role × Default Capabilities

When an agent is created without an explicit `capabilities` list, the service seeds it from the `role` string using this table (`model.DefaultCapabilitiesForRole`):

| Role             | Default capabilities                          |
|------------------|-----------------------------------------------|
| `pm`             | `project_management`                          |
| `developer`      | `coding`, `testing`                           |
| `architect`      | `architecture`, `coding`                      |
| `reviewer`       | `testing`, `security`                         |
| `qa`             | `testing`                                     |
| `security`       | `security`                                    |
| `devops`         | `devops`, `architecture`                      |
| `techwriter`     | `documentation`                               |
| `data_engineer`  | `data_engineering`, `coding`                  |
| `leader`         | `leadership`                                  |

Unknown roles yield an empty default list; the caller is then required to supply `capabilities` explicitly or the create call fails with `400 MISSING_FIELD`.

### Agent Type × Default Capabilities (internal)

In addition to the user-facing role-to-caps map above, the model exposes a parallel `agent_type`-keyed map (`model.DefaultCapabilitiesForType`) used by routing and reporting rather than by the validation seam. The names in this map are the original 12-cap Sprint 4 design set, kept as full capability constants so the canonical capability set is self-describing for tests, documentation, and downstream reporting:

| Agent Type   | Default capabilities                                         |
|--------------|--------------------------------------------------------------|
| `pm`         | `requirement_analysis`, `task_decomposition`                 |
| `arch`       | `system_design`, `api_design`                                |
| `dev`        | `code_implementation`                                        |
| `reviewer`   | `code_review`, `security_scan`                               |
| `qa`         | `test_planning`, `test_execution`                            |
| `devops`     | `ci_cd`, `deployment`, `infrastructure`                      |

The two maps are deliberately separate: `RoleCapabilities` is keyed by the free-form `role` string on the agent struct (used for seeding on create); `AgentTypeCapabilities` is keyed by the closed `AgentType` enum (used by routing and the monitoring dashboard). They never overlap by name — the user-facing nine and the agent-type twelve live in disjoint namespaces.

### Custom Capabilities (Sprint 7, Deferred)

Custom capability creation is the only catalog mutation surface and is **deferred to Sprint 7**:

```
POST /v1/capabilities       (admin-only)

Request:
{
  "name":         "mobile-ios",
  "display_name": "Mobile (iOS)",
  "category":     "coding",
  "description":  "Swift / SwiftUI / UIKit. Default to coding category."
}

Response (201): { same shape as catalog list rows }
```

Reserved names: `__system__*` (double-underscore system prefix) and any of the nine seed names. The validation seam and the role/type default maps will pick up the new name on the next request without a restart because both `ValidCapability` and `AssignableCapabilities` read from the catalog at request time.

### Endpoints Implemented by A-002

The following endpoints are wired, tested, and shipped as part of the Capability System (A-002):

- `GET  /v1/capabilities` — list the full capability catalog (this section).
- `GET  /v1/agents/:id/capabilities` — list capabilities declared by a single agent (this section; also referenced from §Agents for discoverability).

The validation seam (TASK-403) is exposed as part of `POST /v1/tasks/:taskId/assign` (see §Task Assignment) and surfaces the three `CAPABILITY_*` error codes documented above.

---

## The Assignment Engine (A-003)

The **assignment engine** is the workflow that wires a task to an agent and records the action in an append-only history. It is implemented in A-003 and replaces the placeholder §Task Assignment section that previously lived between §Capabilities and §Executions.

Conceptually there are **two tables**:

  - `assignments` (migration 019) is the current-state projection: at most one row per task with `status = 'active'`. The `uq_assignments_one_active_per_task` partial unique index enforces this at the DB layer; a race between two concurrent POSTs surfaces as a unique-constraint violation that the service maps to 409.
  - `assignment_events` (migration 020) is the append-only history. Every state change writes one row. Rows are linked back to the `assignments` row that caused them via `assignment_events.assignment_id → assignments.id`.

The service writes to both inside a single transaction (`s.store.WithTx`) so the two are always consistent. `task.AssigneeID` is updated **outside** the transaction (it is its own table) — if that single-row update fails the assignment is already committed and a Sprint 5+ reconciliation job can backfill.

### The Assignment Lifecycle (status enum)

Four lifecycle values are persisted as TEXT with a CHECK constraint in migration 019:

| Status       | When it is set                                                       | `completed_at` |
|--------------|----------------------------------------------------------------------|----------------|
| `active`     | The row represents the current "who is assigned right now".         | `NULL`         |
| `superseded` | The row was active and has been replaced by a newer assignment.      | set to now     |
| `completed`  | The row finished its lifecycle because the assigned task was completed (TASK-405 drives this transition). | set to now     |
| `cancelled`  | The row was explicitly cancelled (e.g. an admin override or the Sprint 5+ `DELETE /v1/tasks/:id/assign` endpoint). | set to now     |

`model.AllAssignmentStatuses()` mirrors this set; the service uses `model.IsValidAssignmentStatus` as defence in depth (the DB CHECK constraint is the canonical enforcement).

### The Action Verbs (action enum)

Three action verbs are persisted in `assignment_events.action` (migration 020) with a CHECK constraint:

| Action     | When it is written                                                                |
|------------|-----------------------------------------------------------------------------------|
| `assign`   | First-time assignment. `task.AssigneeID` was unset before this event.             |
| `reassign` | `task.AssigneeID` was set to a different agent before this event.                 |
| `unassign` | `task.AssigneeID` was set before this event and is unset after. Reserved for the Sprint 5+ `DELETE /v1/tasks/:id/assign` endpoint; the enum is committed now so history rows from that endpoint do not need a schema migration. |

`model.AllAssignmentActions()` mirrors this set; the service uses `model.IsValidAssignmentAction` for the same defence-in-depth reason.

### Assign Task to Agent

```
POST /v1/tasks/:id/assign
X-Project-ID: <project UUID>          # REQUIRED, F-014 cross-tenant safety

Request:
{
  "agent_id":              "a1b2c3d4-...",     # required, UUID
  "capabilities_required": ["coding", "testing"],  # optional; when non-empty it is persisted to task.RequiredCapabilities and used as the validation seam constraint set. When empty, the task's existing required_capabilities is preserved (not nulled).
  "notes":                 "manual dispatch to backend owner"  # optional, free-text ≤ 1 KiB, audit-trail only
}

Response (200):
{
  "data": {
    "task": { ... updated task with assignee_id set ... },
    "event": {
      "id":            "ev-uuid",
      "assignment_id": "a-uuid",
      "task_id":       "t-uuid",
      "agent_id":      "ag-uuid",   # null for unassign events (Sprint 5+)
      "assigned_by":   "u-uuid",    # null for system-initiated assignments
      "assigned_at":   "2026-06-14T10:00:00Z",
      "action":        "assign",    # one of: assign, reassign, unassign
      "notes":         "..."
    },
    "assignment": {
      "id":           "a-uuid",
      "task_id":      "t-uuid",
      "agent_id":     "ag-uuid",
      "assigned_at":  "2026-06-14T10:00:00Z",
      "completed_at": null,         # omitted when null
      "status":       "active"      # one of: active, superseded, completed, cancelled
    },
    "idempotent": false              # true on re-POST of the same agent_id; no new event written
  }
}
```

Errors:
- `400 VALIDATION_ERROR` — bad UUID in path or body, missing `agent_id`, malformed JSON.
- `400 MISSING_PROJECT_HEADER` — `X-Project-ID` header absent.
- `404 NOT_FOUND` — task or agent does not exist.
- `404 CROSS_TENANT_BLOCKED` (F-014) — `task.ProjectID != callerProjectID` OR `agent.ProjectID != callerProjectID` OR `task.ProjectID != agent.ProjectID`. The response is a 404 (not a 403) so the cross-tenant probe does not leak existence.
- `409 CAPABILITY_MISMATCH` — the agent lacks at least one of the required capabilities (after the validation seam rejects, see §The Capability System (A-002)).
- `409 Agent is not idle` — the agent is in `busy`, `paused`, `error`, or `retired` lifecycle state.
- `409 Assignment race` — a concurrent POST beat this one to the partial unique index; the client may retry.
- `500 INTERNAL` — store error during the transaction.

### Idempotency

Re-POSTing the same `agent_id` to the same task is a no-op: the service returns the existing state with `idempotent: true` and `event: null`. The `assignments` table is not mutated, the `assignment_events` table is not appended, and `task.UpdatedAt` is not bumped. This matches the F-017 brief and the api-spec.md §3.1 contract.

The idempotency check happens after the cross-tenant and capability-validation guards, so a re-POST from the wrong project still returns `404 CROSS_TENANT_BLOCKED` (the safety checks run first).

### List Assignment History

```
GET /v1/tasks/:id/history
X-Project-ID: <project UUID>          # REQUIRED, F-014 cross-tenant safety

Response (200):
{
  "data": [
    {
      "id":            "ev-uuid-1",   # newest first (ORDER BY assigned_at DESC)
      "assignment_id": "a-uuid-1",
      "task_id":       "t-uuid",
      "agent_id":      "ag-uuid-1",
      "assigned_by":   "u-uuid",
      "assigned_at":   "2026-06-14T10:00:00Z",
      "action":        "reassign",
      "notes":         "..."
    },
    ...
  ],
  "meta": {
    "count":       3,
    "server_time": "2026-06-14T10:05:00Z"
  }
}
```

Errors:
- `400 VALIDATION_ERROR` — bad UUID in path.
- `400 MISSING_PROJECT_HEADER` — `X-Project-ID` header absent.
- `404 NOT_FOUND` — task does not exist. The brief prefers 404 over an empty `data: []` so the UI can distinguish "no history yet" from "no such task".
- `404 CROSS_TENANT_BLOCKED` (F-014) — `task.ProjectID != callerProjectID`.
- `500 INTERNAL` — store error during the read.

The response is **the full history in one call** — not paginated, no cursor, no offset. Assignment history per task is bounded (~10s of events in normal use), and the spec for A-003 does not introduce pagination for it. The `meta` envelope provides `count` and `server_time` so the client can render a footer without a second round-trip. If the count grows past a soft limit (TBD by Ops), a future Sprint can add a cursor without a wire break.

### Cross-Tenant Safety (F-014)

Both endpoints in this section require the `X-Project-ID` header. The service performs a triple-check before any state mutation:

  1. `callerProjectID` (from header) must not be `uuid.Nil` — missing header returns `400 MISSING_PROJECT_HEADER`.
  2. `task.ProjectID == callerProjectID` — otherwise `404 CROSS_TENANT_BLOCKED`.
  3. `agent.ProjectID == callerProjectID` (for `POST /v1/tasks/:id/assign`) — otherwise `404 CROSS_TENANT_BLOCKED`.
  4. `task.ProjectID == agent.ProjectID` (defensive) — otherwise `404 CROSS_TENANT_BLOCKED`.

Per the F-014 brief, the rejection is a `404` (not a `403`) so a cross-tenant probe does not leak existence. The wire-level `404` carries the `CROSS_TENANT_BLOCKED` code in the body so a legitimate caller can distinguish it from a real `NOT_FOUND`.

### Endpoints Implemented by A-003

The following endpoints are wired, tested, and shipped as part of the Assignment Engine (A-003):

- `POST /v1/tasks/:id/assign` — wire a task to an agent, append an event, return the new state.
- `GET  /v1/tasks/:id/history` — read the append-only history for a task.

The TASK-403 capability validation seam (see §The Capability System (A-002)) is exposed through `POST /v1/tasks/:id/assign` and surfaces the three `CAPABILITY_*` error codes (`CAPABILITY_NOT_IN_CATALOG`, `CAPABILITY_NOT_ASSIGNABLE`, `CAPABILITY_MISMATCH`).

---

## The Execution Engine (B-001)

An execution records the lifecycle of an agent working on a task. The
execution state machine is the spine of B-001, C-002 (Recovery), and the
runtime worker pool §aion (Runtime).

### Lifecycle

6 states per the brief:

```
QUEUED → ASSIGNED → RUNNING → REVIEW → COMPLETED
                   \         /             /
                    \       /             /
                     \     /             /
                      \   /             /
                       \ /             /
                        FAILED <———————————
```

| State     | Meaning                                                                    | Agent assigned? |
|-----------|----------------------------------------------------------------------------|-----------------|
| `queued`  | The task is in the dispatch queue. No agent has been picked yet.          | No (NULL)       |
| `assigned`| An agent has been picked. The worker pool is preparing to spawn it.       | Yes             |
| `running` | The worker is actively running (aion instance is up, script is executing).| Yes             |
| `review`  | The worker reported `WorkerStatusCompleted`. Awaiting reviewer decision.  | Yes             |
| `completed`| Terminal. Reviewer accepted the deliverables.                             | Yes             |
| `failed`  | Terminal. Either the worker errored, the reviewer rejected, or the queue   | (last assigned) |
|           | could not place the task.                                                  |                 |

### Valid transitions

- `queued` → `assigned` (queue dispatcher picks an agent)
- `queued` → `failed` (queue gives up: capability mismatch persists past deadline, no eligible agent, etc.)
- `assigned` → `running` (worker pool starts the aion instance)
- `assigned` → `failed` (worker pool cannot start the instance: aion runtime unavailable, etc.)
- `assigned` → `queued` (operator or recovery returns the execution to the queue: agent retracted, dispatcher re-runs the eligibility check, the new pick may be a different agent)
- `running` → `review` (worker reports `WorkerStatusCompleted` and emits a deliverable; runtime calls `MarkReview(ctx, execID, agentID, projectID, message)`)
- `running` → `failed` (worker reports `WorkerStatusError` or panics; runtime calls `MarkFailed(ctx, execID, agentID, projectID, reason)`)
- `review` → `completed` (reviewer accepts: PATCH `/v1/executions/:id/review` with `{ accepted: true }`)
- `review` → `failed` (reviewer rejects: PATCH `/v1/executions/:id/review` with `{ accepted: false, reason: "..." }`)

### Terminal states

`completed` and `failed` are terminal. No transitions out. The
PATCH `/v1/executions/:id/status` and `/review` endpoints reject
writes to a terminal state with `409 INVALID_STATE_TRANSITION`.

### Worker status § execution status mapping

The runtime worker pool (see §aion Runtime, the `WorkerStatus`
enum in `model/aion.go`) reports its own state. The execution
service translates worker events into execution transitions:

| Worker event                  | Runtime call to service           | New execution status |
|-------------------------------|------------------------------------|----------------------|
| `WorkerStatusStarting`        | (none; execution is already        | `running`            |
|                               | `assigned` and the runtime is      |                      |
|                               | about to call `MarkRunning`)       |                      |
| `WorkerStatusRunning`         | `MarkRunning(ctx, execID, ...)`    | `running` (idempotent same-state) |
| `WorkerStatusCompleted`       | `MarkReview(ctx, execID, ...)`     | `review` (NOT `completed`; reviewer decides) |
| `WorkerStatusError`           | `MarkFailed(ctx, execID, ..., err)`| `failed`             |
| `WorkerStatusPanicked`        | `MarkFailed(ctx, execID, ..., panic msg)` | `failed`       |

The runtime → service handoff is the spine of commit 3. The runtime
does not call `MarkCompleted` directly; the reviewer action is
the only path into `completed`.

### List Executions
```
GET /v1/executions?task_id=<uuid>&agent_id=<uuid>&status=<queued|assigned|running|review|completed|failed>&page=1&limit=20
```

Query parameters:

- `task_id` (optional, uuid) — filter by task.
- `agent_id` (optional, uuid) — filter by assigned agent. Rows in
  the `queued` state have `agent_id IS NULL` and are excluded by
  an `agent_id` filter (the filter is `agent_id = $1`, not the
  semantics of "any agent". Document this in the client UI as
  "the queue is not visible on the agent filter").
- `status` (optional, enum) — one of the 6 lifecycle values.
  Invalid values return 400 `VALIDATION_ERROR` with a per-field
  detail on the offending value.
- `page` (optional, int, default 1).
- `limit` (optional, int, default 20, max 100).

Response (200):

```json
{
  "data": [
    {
      "id": "uuid",
      "task_id": "uuid",
      "agent_id": "uuid | null",
      "project_id": "uuid",
      "status": "queued | assigned | running | review | completed | failed",
      "created_at": "RFC 3339",
      "started_at": "RFC 3339 | null",
      "completed_at": "RFC 3339 | null",
      "failure_reason": "string | null"
    }
  ],
  "meta": { "page": 1, "limit": 20, "count": 42, "server_time": "RFC 3339" }
}
```

### Get Execution Details
```
GET /v1/executions/:id
```

Response (200):

```json
{
  "data": {
    "id": "uuid",
    "task_id": "uuid",
    "agent_id": "uuid | null",
    "project_id": "uuid",
    "status": "queued | assigned | running | review | completed | failed",
    "created_at": "RFC 3339",
    "started_at": "RFC 3339 | null",
    "completed_at": "RFC 3339 | null",
    "failure_reason": "string | null",
    "events": [
      { "at": "RFC 3339", "from": "queued", "to": "assigned", "by": "system | user:<uuid>" },
      { "at": "RFC 3339", "from": "assigned", "to": "running", "by": "system" },
      { "at": "RFC 3339", "from": "running", "to": "review", "by": "system" }
    ]
  }
}
```

### Create Execution
```
POST /v1/executions

Request:
{
  "task_id": "uuid",
  "agent_id": "uuid"   // optional; omit to enqueue without an agent (creates a `queued` row with agent_id NULL)
}
```

Semantics:

- With `agent_id` set: the execution row is created in `assigned`
  state. The runtime worker pool is expected to start the aion
  instance and call `MarkRunning` shortly after.
- With `agent_id` omitted: the execution row is created in
  `queued` state. The dispatch loop is expected to pick the
  agent and call `Assign` (which transitions `queued`
  → `assigned`).

Response (201):

```json
{
  "data": { "id": "uuid", "status": "queued | assigned", "task_id": "uuid", "agent_id": "uuid | null" }
}
```

Error codes:

- 400 `VALIDATION_ERROR` — missing `task_id`, invalid uuid, etc.
- 400 `MISSING_PROJECT_HEADER` — X-Project-ID header is required.
- 404 `NOT_FOUND` — task (or agent, if `agent_id` set) does not exist.
- 404 `CROSS_TENANT_BLOCKED` — task, agent, or caller is in a
  different project than the call header. The 404 (not 403) avoids
  existence leakage, per the F-014 standing pattern.
- 409 `INVALID_STATE_TRANSITION` — a row already exists for this
  task in a non-terminal state; re-create is rejected.

### Update Execution Status (worker/runtime handoff)
```
PATCH /v1/executions/:id/status

Request:
{
  "to": "running | review | failed",
  "reason": "string"   // required for `to: failed`
}
```

This endpoint is the worker/runtime handoff. It accepts only the
transitions that the runtime can legitimately trigger:
`assigned → running`, `running → review`, `running → failed`,
`assigned → failed`. The `completed` transition is NOT accepted here;
use PATCH `/v1/executions/:id/review` instead. The `queued` and
`assigned` transitions are reserved for the queue dispatcher (commit 3).

Response (200):

```json
{
  "data": { "id": "uuid", "from": "assigned", "to": "running", "at": "RFC 3339" }
}
```

Error codes:

- 400 `VALIDATION_ERROR` — invalid `to` value, missing `reason` for `to: failed`.
- 404 `NOT_FOUND` — execution does not exist.
- 404 `CROSS_TENANT_BLOCKED` — caller is in a different project.
- 409 `INVALID_STATE_TRANSITION` — the transition is not allowed from
  the current state. Response includes a `current_status` and
  `requested_status` in the per-field detail so the runtime can
  branch on the actual reason (e.g. "already completed").

### Review Execution
```
PATCH /v1/executions/:id/review

Request:
{
  "accepted": true | false,
  "reason": "string"   // required when accepted is false; optional when true
}
```

The reviewer action is the only path into `completed`. The
execution must be in `review` state; transitions from any other
state return 409 `INVALID_STATE_TRANSITION`.

Response (200):

```json
{
  "data": { "id": "uuid", "from": "review", "to": "completed | failed", "at": "RFC 3339" }
}
```

### Cancel Execution
```
DELETE /v1/executions/:id
```

Operator-only. Transitions a non-terminal execution to `failed`
with `failure_reason = "cancelled by operator"`. The runtime is
expected to stop the worker on its next checkpoint (the C-002
recovery system polls for the `failed` state and frees the agent).

Response (204).

### F-014 cross-tenant invariant

Every endpoint in this section follows the F-014 triple-check:

1. `caller_project_id` (from `X-Project-ID`) must be set (400 `MISSING_PROJECT_HEADER`).
2. The execution row's `project_id` must equal `caller_project_id` (404 `CROSS_TENANT_BLOCKED`).
3. For PATCH `/status` and POST `/executions` with `agent_id` set, the agent's
   `project_id` must also equal `caller_project_id` (defensive triple-check).

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
