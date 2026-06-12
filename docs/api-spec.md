# AI Software Factory — API Specification

## Base URL
```
Production: https://api.ai-software-factory.com/v1
Development: http://localhost:3001/v1
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

## Projects

### Create Project
```
POST /projects

Request:
{
  "name": "E-commerce Platform",
  "description": "Build a modern e-commerce platform with React frontend and Go backend...",
  "template": "web-app"
}

Response (201):
{
  "id": "proj_abc123",
  "name": "E-commerce Platform",
  "status": "initializing",
  "created_at": "2026-06-10T10:00:00Z",
  "agents_spawned": ["pm"]
}
```

### List Projects
```
GET /projects?status=active&page=1&limit=20

Response:
{
  "data": [
    {
      "id": "proj_abc123",
      "name": "E-commerce Platform",
      "status": "in_progress",
      "progress": 45,
      "active_agents": 3,
      "created_at": "2026-06-10T10:00:00Z"
    }
  ],
  "pagination": { ... }
}
```

### Get Project Details
```
GET /projects/:id

Response:
{
  "id": "proj_abc123",
  "name": "E-commerce Platform",
  "description": "...",
  "status": "in_progress",
  "progress": 45,
  "artifacts": [
    { "type": "vision", "status": "complete" },
    { "type": "architecture", "status": "in_progress" },
    { "type": "user_stories", "status": "complete" }
  ],
  "agents": [
    { "id": "agent_001", "type": "pm", "status": "idle" },
    { "id": "agent_002", "type": "architect", "status": "working" }
  ],
  "created_at": "2026-06-10T10:00:00Z"
}
```

---

## Agents

### Spawn Agent
```
POST /agents/spawn

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
GET /agents?project_id=proj_abc123

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
POST /agents/:id/assign

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

## Tasks

### Create Task
```
POST /projects/:projectId/tasks

Request:
{
  "title": "Implement user authentication API",
  "description": "Create JWT-based authentication with login, register, refresh endpoints",
  "type": "implementation",
  "acceptance_criteria": [
    "POST /api/auth/login returns JWT token",
    "POST /api/auth/register creates new user",
    "POST /api/auth/refresh rotates tokens"
  ],
  "priority": "must_have",
  "estimated_hours": 8
}

Response (201):
{
  "id": "task_001",
  "title": "Implement user authentication API",
  "status": "backlog",
  "created_at": "2026-06-10T10:00:00Z"
}
```

### Update Task Status
```
PATCH /tasks/:id

Request:
{
  "status": "in_progress",
  "assignee_agent_id": "agent_dev_001"
}

Response:
{
  "id": "task_001",
  "status": "in_progress",
  "assignee_agent_id": "agent_dev_001",
  "updated_at": "2026-06-10T10:30:00Z"
}
```

---

## Code

### Generate Code
```
POST /code/generate

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
GET /code/:projectId/files/src/auth/login.ts

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
POST /code/:projectId/commits

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
POST /reviews

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
GET /reviews/:id

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
POST /deployments

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
GET /deployments/:id

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
POST /deployments/:id/rollback

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
POST /users/register

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
GET /users/me

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
POST /webhooks

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
