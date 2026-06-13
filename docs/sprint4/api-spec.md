# API Specification — Sprint 4 (Canonical)

> **Status:** Canonical HTTP spec for the Agent Orchestration Engine, owned by Analyst-01 (TASK-401).
> **Audience:** Developer-01 (handlers for TASK-402/403/404/405/406), Developer-02 (UI for TASK-407/408/409/410), Tester-01 (TASK-411), Security-01 (TASK-412).
> **Base URL:** `/v1` (mounted under the existing Gin router; see `src/internal/router/router.go`).
> **Auth:** Bearer JWT or API key, validated by existing middleware (see `docs/api-spec.md` §Authentication). Sprint 4 inherits; does not add new auth flows.

This doc covers the **Sprint 4 surface only** — agent registry, capabilities, assignment, execution, deliverable management. Auth, project, and task CRUD are covered by the pre-existing `docs/api-spec.md`.

---

## 0. Conventions

### 0.1 Content type

All requests and responses use `application/json` and UTF-8.

### 0.2 Common headers

| Header                | Required | Notes                                                                |
|-----------------------|----------|----------------------------------------------------------------------|
| `Authorization`       | yes      | `Bearer <token>` (JWT or API key)                                    |
| `Content-Type`        | yes (writes) | `application/json`                                                |
| `X-Request-ID`        | no       | Client-supplied request id; echoed in `request_id` of every response |
| `Idempotency-Key`     | recommended for `POST` | `POST /v1/executions` and `POST /v1/deliverables` honor it |

### 0.3 Timestamps

All timestamps are RFC 3339 UTC, e.g. `2026-06-12T08:15:30Z`.

### 0.4 Standard error response

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request parameters",
    "details": [
      { "field": "name", "message": "Name is required" }
    ]
  },
  "request_id": "req_abc123"
}
```

### 0.5 Common status codes

| Code | When                                                              |
|------|-------------------------------------------------------------------|
| 200  | OK — read or update with body                                      |
| 201  | Created — `POST` that creates a new row                            |
| 202  | Accepted — async work started (e.g. `POST /v1/tasks/:id/assign` when async) |
| 204  | No body — `DELETE` success                                         |
| 400  | Validation error (malformed body, missing required field)          |
| 401  | Missing or invalid `Authorization`                                 |
| 403  | Authenticated but not permitted                                   |
| 404  | Resource not found                                                 |
| 409  | Conflict — see error code for specifics (e.g. `AGENT_BUSY`, `NO_AGENT_AVAILABLE`, `CAPABILITY_NOT_FOUND`) |
| 422  | Semantic validation failed (e.g. agent has no capabilities)       |
| 500  | Internal server error                                              |

### 0.6 Standard error codes (Sprint 4)

| `code`                        | Used by                              | Typical `http`  |
|-------------------------------|--------------------------------------|-----------------|
| `VALIDATION_ERROR`            | All                                  | 400             |
| `UNAUTHENTICATED`             | All                                  | 401             |
| `FORBIDDEN`                   | All                                  | 403             |
| `NOT_FOUND`                   | All GET-by-id                        | 404             |
| `AGENT_NOT_FOUND`             | Agent endpoints                      | 404             |
| `TASK_NOT_FOUND`              | Assignment endpoint                  | 404             |
| `CAPABILITY_NOT_FOUND`        | Agent create / assign                | 422             |
| `AGENT_BUSY`                  | Assign with explicit agent that is busy-without-capacity | 409 |
| `NO_AGENT_AVAILABLE`          | Assign auto-route, no candidate      | 409             |
| `PROJECT_MISMATCH`            | Agent or task created in wrong project | 403           |
| `INVALID_STATE_TRANSITION`    | Pause/Resume/Retire illegal          | 409             |
| `VERSION_CONFLICT`            | Optimistic concurrency on PUT        | 409             |
| `RATE_LIMITED`                | (reserved)                           | 429             |

### 0.7 Pagination

List endpoints (`GET /v1/agents`, `GET /v1/executions`, `GET /v1/deliverables`) use cursor pagination:

```
?limit=50&cursor=eyJ0IjoxNzE4MTI0MDAwfQ
```

Response:

```json
{
  "data": [ ... ],
  "page_info": {
    "next_cursor": "eyJ0IjoxNzE4MTI1MDAwfQ",
    "has_more": true
  }
}
```

Default `limit` = 50, max = 200.

---

## 1. Agents

### 1.1 `POST /v1/agents` — create an agent

**Body**

```json
{
  "project_id":   "9c2b1f7e-1c2b-4f1f-9d3c-3b3b6c2c0001",
  "name":         "Backend Coder A",
  "role":         "Backend Developer",
  "capabilities": ["coding", "testing"],
  "metadata":     { "model": "gpt-4o", "version": "2026-05-12" }
}
```

**Validation rules**

- `project_id`, `name`, `role`, `capabilities` are required.
- `name` ≤ 80 chars, unique within `project_id`.
- `capabilities` must be ≥ 1 element, and **every** name must exist in the `capabilities` catalog (see `GET /v1/capabilities`).
- `metadata` is optional and must be a JSON object.

**Response 201**

```json
{
  "data": {
    "id":             "7a1c8a8e-3a2f-4b3a-9c2a-2e5b5e7d1111",
    "project_id":     "9c2b1f7e-1c2b-4f1f-9d3c-3b3b6c2c0001",
    "name":           "Backend Coder A",
    "role":           "Backend Developer",
    "status":         "initializing",
    "capabilities":   ["coding", "testing"],
    "last_active_at": null,
    "metadata":       { "model": "gpt-4o", "version": "2026-05-12" },
    "created_at":     "2026-06-12T08:00:00Z",
    "updated_at":     "2026-06-12T08:00:00Z"
  },
  "request_id": "req_abc123"
}
```

**Errors**

| Status | `code`                | Cause                                                |
|--------|-----------------------|------------------------------------------------------|
| 400    | `VALIDATION_ERROR`    | Missing field or wrong type                          |
| 422    | `CAPABILITY_NOT_FOUND`| A name in `capabilities` is not in the catalog       |
| 409    | `NOT_FOUND` (project) | `project_id` does not exist                          |

---

### 1.2 `GET /v1/agents` — list agents

**Query**

| Param        | Type    | Notes                                                                 |
|--------------|---------|-----------------------------------------------------------------------|
| `project_id` | UUID    | Required.                                                              |
| `status`     | string  | Filter: `initializing`, `idle`, `busy`, `paused`, `error`, `retired`.  |
| `capability` | string  | Filter to agents declaring this capability.                           |
| `limit`      | int     | Default 50, max 200.                                                  |
| `cursor`     | string  | Opaque.                                                               |

**Response 200**

```json
{
  "data": [
    { "id": "7a1c...", "name": "Backend Coder A", "status": "idle", ... },
    { "id": "8b2d...", "name": "Security Reviewer", "status": "busy", ... }
  ],
  "page_info": { "next_cursor": "eyJ0...", "has_more": true }
}
```

By default, agents with `status = 'retired'` are **excluded**. Pass `?include_retired=true` to include them.

---

### 1.3 `GET /v1/agents/:id` — fetch one agent

**Response 200** — same shape as `POST` response.

**Errors**

| Status | `code`             |
|--------|--------------------|
| 404    | `AGENT_NOT_FOUND`  |

---

### 1.4 `PUT /v1/agents/:id` — update agent metadata

**Body** — partial update; only `role`, `capabilities`, and `metadata` are mutable. `name`, `project_id`, `status` are NOT mutable here (status changes go through §1.5; name/project_id are immutable).

```json
{
  "role":         "Senior Backend Developer",
  "capabilities": ["coding", "testing", "devops"],
  "metadata":     { "model": "gpt-4o", "version": "2026-06-01" },
  "version":      3
}
```

**Optimistic concurrency:** `version` is an integer returned by GETs. The update succeeds only if the stored version matches. On mismatch, returns `409 VERSION_CONFLICT`.

**Side effects:**

- If `capabilities` is changed, `agent_capabilities` is rewritten in the same transaction and `agents.capabilities[]` cache is updated.

**Response 200** — full agent record (same as `POST` response).

**Errors**

| Status | `code`                    |
|--------|---------------------------|
| 400    | `VALIDATION_ERROR`        |
| 404    | `AGENT_NOT_FOUND`         |
| 409    | `VERSION_CONFLICT`        |
| 422    | `CAPABILITY_NOT_FOUND`    |

---

### 1.5 `DELETE /v1/agents/:id` — retire an agent (soft delete)

Soft delete. Sets `status = 'retired'`, `retired_at = NOW()`. Existing assignments continue; the agent is filtered out of `POST /v1/tasks/:id/assign` auto-routing.

**Response 204** — empty body.

**Errors**

| Status | `code`                       |
|--------|------------------------------|
| 404    | `AGENT_NOT_FOUND`            |
| 409    | `INVALID_STATE_TRANSITION`   | (e.g. agent is currently the only one holding a `leadership` capability — use `?force=true` to override, which logs to `agent_state_events.reason = 'forced_retire'`)

---

### 1.6 `GET /v1/agents/:id/capabilities` — list capabilities for an agent

**Response 200**

```json
{
  "data": [
    {
      "name":         "coding",
      "display_name": "Coding",
      "category":     "coding",
      "proficiency":  4,
      "granted_at":   "2026-06-12T08:00:00Z"
    },
    {
      "name":         "testing",
      "display_name": "Testing",
      "category":     "testing",
      "proficiency":  3,
      "granted_at":   "2026-06-12T08:00:00Z"
    }
  ],
  "request_id": "req_abc123"
}
```

**Errors**

| Status | `code`            |
|--------|-------------------|
| 404    | `AGENT_NOT_FOUND` |

---

## 2. Capabilities

### 2.1 `GET /v1/capabilities` — list the capability catalog

**Query**

| Param     | Type   | Notes                                  |
|-----------|--------|----------------------------------------|
| `category`| string | Filter to one category.                |
| `limit`   | int    | Default 100 (catalog is small).        |
| `cursor`  | string | Opaque.                                |

**Response 200**

```json
{
  "data": [
    { "name": "architecture", "display_name": "Architecture", "category": "architecture", "version": 1 },
    { "name": "coding",       "display_name": "Coding",       "category": "coding",       "version": 1 },
    { "name": "devops",       "display_name": "DevOps",       "category": "devops",       "version": 1 },
    { "name": "leadership",   "display_name": "Leader",       "category": "leadership",   "version": 1 },
    { "name": "security",     "display_name": "Security",     "category": "security",     "version": 1 },
    { "name": "testing",      "display_name": "Testing",      "category": "testing",      "version": 1 }
  ],
  "page_info": { "next_cursor": null, "has_more": false }
}
```

---

## 3. Assignment

### 3.1 `POST /v1/tasks/:id/assign` — assign a task to an agent

**Body** — all fields optional; the engine auto-routes when no `agent_id` is given.

```json
{
  "agent_id":  "7a1c8a8e-3a2f-4b3a-9c2a-2e5b5e7d1111",     // optional; if absent, auto-route
  "strategy":  "least_recently_active",                      // optional, default above
  "reason":    "leader dispatched after planning"            // optional free-text
}
```

**Behavior** (rules engine — see `agent-orchestration-design.md` §4)

1. Project scope: `task.project_id == agent.project_id`. Else `403 PROJECT_MISMATCH`.
2. If `agent_id` is **not** given: auto-route using the rules in `agent-orchestration-design.md` §4.2.
3. If `agent_id` is given: skip capability check, but require `status ∈ {idle, busy with capacity}` and project scope.
4. Persist `assignments` row (one per task, partial unique index on `is_current = TRUE`).
5. Update `tasks.assigned_agent_id`, `tasks.assigned_at`, and `agents.status` (idle → busy) in the same transaction.
6. Emit an `agent_state_events` row and an `assignment_events` row.

**Response 200** (or 202 if execution engine picks the work up async)

```json
{
  "data": {
    "assignment_id": "be4f7a8e-9c2a-4b3a-9c2a-2e5b5e7d2222",
    "task_id":       "5d3c2a8e-1c2b-4f1f-9d3c-3b3b6c2c0007",
    "agent": {
      "id":     "7a1c8a8e-3a2f-4b3a-9c2a-2e5b5e7d1111",
      "name":   "Backend Coder A",
      "status": "busy"
    },
    "strategy":         "least_recently_active",
    "selected_reason":  "rule_pass",
    "candidates_considered": 3,
    "rules_evaluated":  [
      { "rule": "project_scope",     "passed": true },
      { "rule": "capability_match",  "passed": true, "evidence": { "required": "coding", "agent_has": ["coding","testing"] } },
      { "rule": "availability",      "passed": true, "evidence": { "status": "idle" } },
      { "rule": "not_overcommitted", "passed": true, "evidence": { "open_executions": 0, "max": 1 } }
    ],
    "assigned_at": "2026-06-12T08:05:00Z"
  },
  "request_id": "req_abc123"
}
```

**Errors**

| Status | `code`                  | Cause                                                           |
|--------|-------------------------|-----------------------------------------------------------------|
| 400    | `VALIDATION_ERROR`      | Bad body                                                        |
| 404    | `TASK_NOT_FOUND`        | Task id does not exist                                          |
| 403    | `PROJECT_MISMATCH`      | `agent_id` belongs to a different project                       |
| 409    | `NO_AGENT_AVAILABLE`    | Auto-route, zero candidates; response includes a `hint` array   |
| 409    | `AGENT_BUSY`            | Explicit `agent_id` that is busy and at capacity                |
| 409    | `INVALID_STATE_TRANSITION` | Explicit `agent_id` whose `status` is `paused`, `error`, or `retired` |
| 422    | `CAPABILITY_NOT_FOUND`  | The task's `required_capability` is not in the catalog          |

**409 NO_AGENT_AVAILABLE — extended body**

```json
{
  "error": {
    "code": "NO_AGENT_AVAILABLE",
    "message": "No agent in this project can take the task",
    "details": {
      "required_capability": "security",
      "candidates_considered": 0,
      "hint": [
        "No agent in project 9c2b... declares the 'security' capability.",
        "Create one with POST /v1/agents and capability 'security'."
      ]
    }
  },
  "request_id": "req_abc123"
}
```

---

## 4. Executions

### 4.1 `POST /v1/executions` — start an execution

Starts execution for an already-assigned task. In **MOCK** mode (Sprint 4), the execution is recorded as `running` and a synthetic `output` is generated within 5 s, transitioning to `succeeded` (or `failed` if `force_fail: true` is set — for testing).

**Body**

```json
{
  "task_id":  "5d3c2a8e-1c2b-4f1f-9d3c-3b3b6c2c0007",
  "agent_id": "7a1c8a8e-3a2f-4b3a-9c2a-2e5b5e7d1111",
  "input":    { "prompt": "Implement POST /v1/things", "context": { "...": "..." } },
  "force_fail": false
}
```

`agent_id` is optional; if omitted, the engine looks up the current `assignments` row for `task_id`.

`force_fail` is a Sprint 4 testing affordance: when `true`, the mock execution transitions to `failed` with a synthetic `error` message. Ignored in production builds (always `false` from non-test callers).

`Idempotency-Key` header is honored: re-submitting with the same key returns the original execution.

**Response 201**

```json
{
  "data": {
    "id":           "9f0e7d6c-5b4a-3210-fedc-ba0987654321",
    "task_id":      "5d3c2a8e-1c2b-4f1f-9d3c-3b3b6c2c0007",
    "agent_id":     "7a1c8a8e-3a2f-4b3a-9c2a-2e5b5e7d1111",
    "status":       "running",
    "started_at":   "2026-06-12T08:05:01Z",
    "completed_at": null,
    "output":       {},
    "error":        null
  },
  "request_id": "req_abc123"
}
```

**Errors**

| Status | `code`                  | Cause                                              |
|--------|-------------------------|----------------------------------------------------|
| 400    | `VALIDATION_ERROR`      | Missing `task_id` or `input`                       |
| 404    | `TASK_NOT_FOUND`        | Task id does not exist                             |
| 404    | `AGENT_NOT_FOUND`       | Explicit `agent_id` does not exist                 |
| 409    | `AGENT_BUSY`            | Agent already has a running execution              |
| 409    | `INVALID_STATE_TRANSITION` | No current assignment for this task / agent is not `busy` |

---

### 4.2 `GET /v1/executions` — list executions

**Query**

| Param        | Type    | Notes                                       |
|--------------|---------|---------------------------------------------|
| `task_id`    | UUID    | Filter to a task.                           |
| `agent_id`   | UUID    | Filter to an agent.                         |
| `status`     | string  | `pending`, `running`, `succeeded`, `failed`, `cancelled`. |
| `project_id` | UUID    | Filter via denormalized join.               |
| `limit`      | int     | Default 50, max 200.                        |
| `cursor`     | string  | Opaque.                                     |

**Response 200**

```json
{
  "data": [
    {
      "id":           "9f0e7d6c-5b4a-3210-fedc-ba0987654321",
      "task_id":      "5d3c2a8e-...",
      "agent_id":     "7a1c8a8e-...",
      "status":       "succeeded",
      "started_at":   "2026-06-12T08:05:01Z",
      "completed_at": "2026-06-12T08:05:08Z",
      "output":       { "summary": "Mocked success" },
      "error":        null
    }
  ],
  "page_info": { "next_cursor": null, "has_more": false }
}
```

---

### 4.3 `PATCH /v1/executions/:id` — control an execution (cancel / mark failed)

Used by the **MOCK** engine to update execution status, and by the UI to cancel a running execution. In Sprint 5 (real Hermes) this endpoint will be admin-only.

**Body** — exactly one of:

```json
{ "status": "cancelled", "reason": "Operator cancelled" }
```

```json
{ "status": "failed",    "error": "Mock failure injected" }
```

When `status` is set to a terminal value (`succeeded`, `failed`, `cancelled`), the engine:

- Sets `completed_at = NOW()`.
- Updates the related agent's status: `busy → idle` (on `succeeded` / `cancelled`) or `busy → error` (on `failed`).
- Emits an `agent_state_events` row.

**Response 200** — full execution record.

**Errors**

| Status | `code`                    | Cause                                          |
|--------|---------------------------|------------------------------------------------|
| 400    | `VALIDATION_ERROR`        | Missing `status` or unknown value              |
| 404    | `NOT_FOUND` (execution)   | Id does not exist                              |
| 409    | `INVALID_STATE_TRANSITION`| E.g. PATCH `cancelled` on an already-terminal execution |

---

## 5. Deliverables

### 5.1 `POST /v1/deliverables` — create a deliverable

**Body**

```json
{
  "task_id":      "5d3c2a8e-1c2b-4f1f-9d3c-3b3b6c2c0007",
  "agent_id":     "7a1c8a8e-3a2f-4b3a-9c2a-2e5b5e7d1111",
  "kind":         "code",
  "title":        "Implement POST /v1/things",
  "description":  "Adds the things endpoint with validation.",
  "content":      "package things\n\nfunc Handler() http.HandlerFunc { ... }",
  "metadata":     { "language": "go", "lines": 47 }
}
```

`agent_id` is optional; if omitted, the engine resolves it from the current assignment for the task.

`Idempotency-Key` header is honored.

**Response 201**

```json
{
  "data": {
    "id":             "2b3c4d5e-6f70-8190-a1b2-c3d4e5f60718",
    "task_id":        "5d3c2a8e-1c2b-4f1f-9d3c-3b3b6c2c0007",
    "agent_id":       "7a1c8a8e-3a2f-4b3a-9c2a-2e5b5e7d1111",
    "project_id":     "9c2b1f7e-1c2b-4f1f-9d3c-3b3b6c2c0001",
    "kind":           "code",
    "title":          "Implement POST /v1/things",
    "description":    "Adds the things endpoint with validation.",
    "content":        "package things\n\nfunc Handler() http.HandlerFunc { ... }",
    "metadata":       { "language": "go", "lines": 47 },
    "latest_version": 1,
    "created_at":     "2026-06-12T08:10:00Z",
    "updated_at":     "2026-06-12T08:10:00Z"
  },
  "request_id": "req_abc123"
}
```

**Errors**

| Status | `code`                  | Cause                                                    |
|--------|-------------------------|----------------------------------------------------------|
| 400    | `VALIDATION_ERROR`      | Missing required field                                   |
| 404    | `TASK_NOT_FOUND`        | Task id does not exist                                   |
| 404    | `AGENT_NOT_FOUND`       | Explicit agent id does not exist                         |
| 409    | `INVALID_STATE_TRANSITION` | Task is not in a state that allows deliverables        |

---

### 5.2 `GET /v1/deliverables` — list deliverables

**Query**

| Param        | Type    | Notes                                                |
|--------------|---------|------------------------------------------------------|
| `project_id` | UUID    | Filter to a project.                                 |
| `task_id`    | UUID    | Filter to a task.                                    |
| `agent_id`   | UUID    | Filter to an agent.                                  |
| `kind`       | string  | `code`, `doc`, `design`, `test_report`, `config`, `other`. |
| `limit`      | int     | Default 50, max 200.                                 |
| `cursor`     | string  | Opaque.                                              |

**Response 200** — `data` array of deliverables (same shape as `POST` response, without `request_id`).

---

### 5.3 `PUT /v1/deliverables/:id` — update a deliverable (creates a new version)

`PUT` is **always** treated as "create a new version". The previous content is preserved in `deliverable_versions`; the current `content`/`metadata` on the deliverable is overwritten and `latest_version` is incremented.

**Body** — `content` is required; everything else optional.

```json
{
  "content":        "package things\n\nfunc Handler() http.HandlerFunc { /* v2 */ }",
  "metadata":       { "language": "go", "lines": 60 },
  "title":          "Implement POST /v1/things (v2)",
  "description":    "Adds input validation and error envelope.",
  "change_summary": "Add validation + error envelope per TASK-403 review",
  "version":        1
}
```

`version` is the **expected** current `latest_version` (optimistic concurrency). Returns `409 VERSION_CONFLICT` if the stored version is different.

**Response 200**

```json
{
  "data": {
    "id":             "2b3c4d5e-6f70-8190-a1b2-c3d4e5f60718",
    "latest_version": 2,
    "content":        "package things\n\nfunc Handler() http.HandlerFunc { /* v2 */ }",
    "updated_at":     "2026-06-12T08:15:00Z",
    "...":            "..."
  },
  "request_id": "req_abc123"
}
```

**Errors**

| Status | `code`                  | Cause                                            |
|--------|-------------------------|--------------------------------------------------|
| 400    | `VALIDATION_ERROR`      | Missing `content`                                |
| 404    | `NOT_FOUND` (deliverable) | Id does not exist                              |
| 409    | `VERSION_CONFLICT`      | Stale `version` field                            |

---

### 5.4 `GET /v1/deliverables/:id/versions` — list version history

Returns rows from `deliverable_versions`, newest first.

**Query**

| Param   | Type   | Notes                                |
|---------|--------|--------------------------------------|
| `limit` | int    | Default 50, max 200.                 |
| `cursor`| string | Opaque.                              |

**Response 200**

```json
{
  "data": [
    {
      "version":        2,
      "content":        "package things\n\nfunc Handler() http.HandlerFunc { /* v2 */ }",
      "metadata":       { "language": "go", "lines": 60 },
      "change_summary": "Add validation + error envelope",
      "created_by":     "7a1c8a8e-3a2f-4b3a-9c2a-2e5b5e7d1111",
      "created_at":     "2026-06-12T08:15:00Z"
    },
    {
      "version":        1,
      "content":        "package things\n\nfunc Handler() http.HandlerFunc { ... }",
      "metadata":       { "language": "go", "lines": 47 },
      "change_summary": null,
      "created_by":     "7a1c8a8e-3a2f-4b3a-9c2a-2e5b5e7d1111",
      "created_at":     "2026-06-12T08:10:00Z"
    }
  ],
  "page_info": { "next_cursor": null, "has_more": false }
}
```

**Errors**

| Status | `code`                    |
|--------|---------------------------|
| 404    | `NOT_FOUND` (deliverable) |

---

## 6. Endpoints NOT in Sprint 4 scope

For clarity, the following endpoints are **explicitly out of scope** for Sprint 4 (deferred to later sprints or covered by existing `docs/api-spec.md`):

- `POST /v1/capabilities` / `PATCH /v1/capabilities/:id` / `DELETE /v1/capabilities/:id` — catalog CRUD. Sprint 4 ships the seed and a read endpoint; admin UI for the catalog is post-Sprint 4.
- `POST /v1/projects`, `POST /v1/tasks`, etc. — covered by `docs/api-spec.md` (Sprint 1–3).
- Auth endpoints — covered by `docs/api-spec.md`.
- Real Hermes integration endpoints — Sprint 5.

---

## 7. Cross-references

- Design rationale for the entities behind these endpoints: [`agent-orchestration-design.md`](./agent-orchestration-design.md).
- Postgres tables touched by these endpoints: [`data-model.md`](./data-model.md).
- Existing high-level API conventions (auth, error envelope, pagination header): [`../api-spec.md`](../api-spec.md).
- Sprint 4 test plan: see `quality-gates.md` and the TASK-411 acceptance criteria.
