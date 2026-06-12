# Sprint 4 — Agent Orchestration API

> All endpoints are prefixed with `/v1`. Base URL: `http://localhost:8080/v1`

---

## Agents

### Create Agent

```
POST /v1/agents
Content-Type: application/json

{
  "name": "Code Assistant",
  "type": "developer",
  "role": "developer",
  "model": "gpt-4",
  "provider": "openai",
  "capabilities": ["code_implementation"]
}
```

**Response (201):**
```json
{
  "id": "a1b2c3d4-...",
  "name": "Code Assistant",
  "type": "developer",
  "role": "developer",
  "model": "gpt-4",
  "provider": "openai",
  "capabilities": ["code_implementation"],
  "status": "idle",
  "created_at": "2026-06-12T10:00:00Z",
  "updated_at": "2026-06-12T10:00:00Z"
}
```

**Validation:**
- `name` — required
- `role` — required
- `capabilities` — optional; if omitted, defaults from `AgentTypeCapabilities` mapping are used

### List Agents

```
GET /v1/agents?status=idle&role=developer&page=1&limit=20
```

**Response:**
```json
{
  "data": [
    {
      "id": "a1b2c3d4-...",
      "name": "Code Assistant",
      "type": "developer",
      "role": "developer",
      "model": "gpt-4",
      "provider": "openai",
      "capabilities": ["code_implementation"],
      "status": "idle",
      "created_at": "2026-06-12T10:00:00Z",
      "updated_at": "2026-06-12T10:00:00Z"
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

**Query parameters:**
| Name | Type | Description |
|------|------|-------------|
| `status` | string | Filter by status: `idle`, `working`, `spawning`, `completed`, `failed` |
| `role` | string | Filter by role: `pm`, `architect`, `developer`, `reviewer`, `qa`, `devops` |
| `page` | int | Page number (default: 1) |
| `limit` | int | Items per page (default: 20, max: 100) |

### Get Agent

```
GET /v1/agents/:id
```

**Response:**
```json
{
  "id": "a1b2c3d4-...",
  "name": "Code Assistant",
  "type": "developer",
  "role": "developer",
  "model": "gpt-4",
  "provider": "openai",
  "capabilities": ["code_implementation"],
  "status": "idle",
  "created_at": "2026-06-12T10:00:00Z",
  "updated_at": "2026-06-12T10:00:00Z"
}
```

### Update Agent

```
PUT /v1/agents/:id
Content-Type: application/json

{
  "name": "Senior Code Assistant",
  "model": "gpt-4-turbo",
  "capabilities": ["code_implementation", "code_review"],
  "status": "idle"
}
```

**Response:**
```json
{
  "id": "a1b2c3d4-...",
  "name": "Senior Code Assistant",
  "model": "gpt-4-turbo",
  "capabilities": ["code_implementation", "code_review"],
  "status": "idle",
  ...
}
```

All fields are optional. Only provided fields are updated.

### Delete Agent

```
DELETE /v1/agents/:id
```

**Response: 204 No Content**

---

## Task Assignment

### Assign Task to Agent

```
POST /v1/tasks/:taskId/assign
Content-Type: application/json

{
  "agent_id": "a1b2c3d4-..."
}
```

**Response:**
```json
{
  "execution_id": "e5f6a7b8-...",
  "task_id": "task-uuid-here",
  "agent_id": "a1b2c3d4-...",
  "status": "running",
  "started_at": "2026-06-12T10:00:00Z"
}
```

**Flow:**
1. Validates task and agent exist
2. Checks agent is in `idle` status
3. Matches agent capabilities against required capabilities for the task type
4. Creates an execution record with status `running`
5. Updates task status to `in_progress` and sets `assignee_id`
6. Updates agent status to `working`

**Error responses:**
| Status | Code | Reason |
|--------|------|--------|
| 404 | `NOT_FOUND` | Task or agent not found |
| 409 | `CONFLICT` | Agent is not idle |
| 422 | `CAPABILITY_MISMATCH` | Agent lacks required capabilities |

---

## Executions

### Create Execution

```
POST /v1/executions
Content-Type: application/json

{
  "task_id": "task-uuid-here",
  "agent_id": "a1b2c3d4-..."
}
```

**Response (201):**
```json
{
  "execution_id": "e5f6a7b8-...",
  "task_id": "task-uuid-here",
  "agent_id": "a1b2c3d4-...",
  "status": "running",
  "started_at": "2026-06-12T10:00:00Z",
  "created_at": "2026-06-12T10:00:00Z"
}
```

### List Executions

```
GET /v1/executions?task_id=<uuid>&agent_id=<uuid>&page=1&limit=20
```

**Response:**
```json
{
  "data": [
    {
      "execution_id": "e5f6a7b8-...",
      "task_id": "task-uuid-here",
      "agent_id": "a1b2c3d4-...",
      "status": "running",
      "started_at": "2026-06-12T10:00:00Z",
      "created_at": "2026-06-12T10:00:00Z"
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

**Query parameters:**
| Name | Type | Description |
|------|------|-------------|
| `task_id` | uuid | Filter by task |
| `agent_id` | uuid | Filter by agent |
| `page` | int | Page number |
| `limit` | int | Items per page |

### Get Execution

```
GET /v1/executions/:id
```

**Response:**
```json
{
  "execution_id": "e5f6a7b8-...",
  "task_id": "task-uuid-here",
  "agent_id": "a1b2c3d4-...",
  "status": "running",
  "started_at": "2026-06-12T10:00:00Z",
  "created_at": "2026-06-12T10:00:00Z"
}
```

### Update Execution Status

```
PATCH /v1/executions/:id/status
Content-Type: application/json

{
  "status": "completed"
}
```

**Response:**
```json
{
  "execution_id": "e5f6a7b8-...",
  "task_id": "task-uuid-here",
  "agent_id": "a1b2c3d4-...",
  "status": "completed",
  "started_at": "2026-06-12T10:00:00Z",
  "completed_at": "2026-06-12T11:00:00Z",
  "created_at": "2026-06-12T10:00:00Z"
}
```

**Allowed transitions:**
| From | To |
|------|----|
| `pending` | `running` |
| `running` | `completed`, `failed` |
| `completed` | *(terminal)* |
| `failed` | *(terminal)* |

**Validation:**
- `status` is required (non-empty)
- Invalid transitions return `422 Unprocessable Entity` with code `INVALID_TRANSITION`
- `completed` status is required

---

## Deliverables

### Create Deliverable

```
POST /v1/deliverables
Content-Type: application/json

{
  "task_id": "task-uuid-here",
  "agent_id": "a1b2c3d4-...",
  "title": "Authentication API Design",
  "content": "## API Endpoints\n\n### POST /auth/login\n..."
}
```

**Response (201):**
```json
{
  "id": "d1e2f3a4-...",
  "task_id": "task-uuid-here",
  "agent_id": "a1b2c3d4-...",
  "title": "Authentication API Design",
  "content": "## API Endpoints\n\n### POST /auth/login\n...",
  "version": 1,
  "created_at": "2026-06-12T10:00:00Z"
}
```

**Validation:**
- `task_id` — required, must reference an existing task
- `agent_id` — required, must reference an existing agent
- `title` — required

### List Deliverables

```
GET /v1/deliverables?task_id=<uuid>
```

```
GET /v1/deliverables?agent_id=<uuid>
```

**Response:**
```json
[
  {
    "id": "d1e2f3a4-...",
    "task_id": "task-uuid-here",
    "agent_id": "a1b2c3d4-...",
    "title": "Authentication API Design",
    "content": "## API Endpoints...",
    "version": 1,
    "created_at": "2026-06-12T10:00:00Z"
  }
]
```

**Query parameters (exactly one required):**
| Name | Type | Description |
|------|------|-------------|
| `task_id` | uuid | Filter by task |
| `agent_id` | uuid | Filter by agent |

Returns deliverables sorted by creation date descending. Missing both filters returns `400 VALIDATION_ERROR`.

### Get Deliverable

```
GET /v1/deliverables/:id
```

**Response:**
```json
{
  "id": "d1e2f3a4-...",
  "task_id": "task-uuid-here",
  "agent_id": "a1b2c3d4-...",
  "title": "Authentication API Design",
  "content": "## API Endpoints...",
  "version": 1,
  "created_at": "2026-06-12T10:00:00Z"
}
```

### Update Deliverable

```
PUT /v1/deliverables/:id
Content-Type: application/json

{
  "title": "Authentication API Design v2",
  "content": "## Updated API Endpoints\n..."
}
```

**Response:**
```json
{
  "id": "d1e2f3a4-...",
  "task_id": "task-uuid-here",
  "agent_id": "a1b2c3d4-...",
  "title": "Authentication API Design v2",
  "content": "## Updated API Endpoints\n...",
  "version": 2,
  "created_at": "2026-06-12T10:00:00Z"
}
```

Version auto-increments on each update.

---

## Error Responses

All endpoints follow the standard error format:

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

**Sprint 4 error codes:**
| Code | HTTP Status | Meaning |
|------|-------------|---------|
| `VALIDATION_ERROR` | 400 | Invalid input |
| `INVALID_JSON` | 400 | Malformed request body |
| `NOT_FOUND` | 404 | Resource not found |
| `CONFLICT` | 409 | Resource conflict (e.g. agent not idle) |
| `CAPABILITY_MISMATCH` | 422 | Agent lacks required capabilities |
| `INVALID_TRANSITION` | 422 | Invalid status transition |
| `INTERNAL_ERROR` | 500 | Unexpected server error |
