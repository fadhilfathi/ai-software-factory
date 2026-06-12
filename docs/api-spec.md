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

### Spawn Agent
```
POST /v1/agents/spawn

Request:
{
  "project_id": "proj_abc123",
  "type": "developer",
  "config": {
    "model": "gpt-4",
    "temperature": 0.3
  }
}

Response (201):
{
  "id": "agent_dev_001",
  "type": "developer",
  "status": "spawning",
  "project_id": "proj_abc123"
}
```

### List Active Agents
```
GET /v1/agents?project_id=proj_abc123

Response:
{
  "data": [
    {
      "id": "agent_dev_001",
      "type": "developer",
      "status": "working",
      "current_task": "task_001",
      "tasks_completed": 5,
      "uptime": 3600
    }
  ]
}
```

### Assign Task to Agent
```
POST /v1/agents/:id/assign

Request:
{
  "task_id": "task_001",
  "priority": "high",
  "context": {
    "files": ["src/api/users.ts", "src/models/user.ts"]
  }
}

Response:
{
  "id": "agent_dev_001",
  "task_id": "task_001",
  "status": "working",
  "estimated_completion": "2026-06-10T11:00:00Z"
}
```

---

## Code

### Generate Code
```
POST /v1/code/generate

Request:
{
  "project_id": "proj_abc123",
  "task_id": "task_001",
  "specification": "Implement JWT authentication with Gin...",
  "files": ["src/auth/login.ts", "src/auth/register.ts"]
}

Response (202):
{
  "id": "code_gen_001",
  "status": "generating",
  "estimated_time": 300
}
```

### Get File Content
```
GET /v1/code/:projectId/files/*path

Response:
{
  "path": "src/auth/login.ts",
  "content": "import jwt from 'jsonwebtoken'...",
  "language": "typescript",
  "size": 2048,
  "last_modified": "2026-06-10T10:45:00Z",
  "modified_by": "agent_dev_001"
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
    { "path": "src/auth/login.ts", "content": "..." },
    { "path": "src/auth/register.ts", "content": "..." }
  ]
}

Response (201):
{
  "sha": "abc123def456",
  "message": "feat: implement JWT authentication",
  "author": "agent_dev_001",
  "created_at": "2026-06-10T10:50:00Z"
}
```

---

## Reviews

### Create Review Request
```
POST /v1/reviews

Request:
{
  "project_id": "proj_abc123",
  "commit_sha": "abc123def456",
  "reviewer_type": "automated"
}

Response (201):
{
  "id": "review_001",
  "status": "in_progress",
  "reviewer": "review_agent_001"
}
```

### Get Review Results
```
GET /v1/reviews/:id

Response:
{
  "id": "review_001",
  "status": "completed",
  "result": "approved",
  "score": 8.5,
  "issues": [
    {
      "severity": "warning",
      "file": "src/auth/login.ts",
      "line": 42,
      "message": "Consider adding rate limiting to login endpoint",
      "suggestion": "Add gin-rate-limit middleware"
    }
  ],
  "metrics": {
    "complexity": "low",
    "test_coverage": 92,
    "duplications": 0
  }
}
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
