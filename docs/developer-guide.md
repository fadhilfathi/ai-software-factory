# AI Software Factory — Developer Guide

> **Document Version**: 1.1  
> **Last Updated**: 2026-06-12  
> **Applies To**: API v1  
> **Repository**: `github.com/example/project`

---

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Quick Start for Developers](#quick-start-for-developers)
- [Architecture Overview](#architecture-overview)
- [API Reference](#api-reference)
  - [Authentication](#authentication)
  - [Projects](#projects)
  - [Agents](#agents)
  - [Tasks](#tasks)
  - [Code Generation](#code-generation)
  - [Code Reviews](#code-reviews)
  - [Deployments](#deployments)
  - [Users](#users)
  - [Webhooks](#webhooks)
- [Frontend Architecture](#frontend-architecture)
- [Backend Architecture](#backend-architecture)
- [Database Schema](#database-schema)
- [Contributing Guide](#contributing-guide)
  - [Development Workflow](#development-workflow)
  - [Code Standards](#code-standards)
  - [Testing](#testing)
  - [Pull Request Process](#pull-request-process)
- [Operations](#operations)
  - [Building and Running](#building-and-running)
  - [Logging and Monitoring](#logging-and-monitoring)
  - [Debugging](#debugging)
- [Appendix: Error Codes](#appendix-error-codes)

---

## Overview

The AI Software Factory is a multi-agent software development platform. It orchestrates
specialized AI agents (PM, Architect, Developer, Reviewer, QA) to autonomously build
software projects from a user's description.

This guide is for **developers integrating with the API**, **contributors working on the
codebase**, and **operators deploying the system**. It covers the internal architecture,
protocol-level API contracts, and contribution workflows.

If you are an end user looking for feature documentation and tutorials, see the
[User Guide](./user-guide.md).

---

## Prerequisites

| Tool       | Version    | Purpose                        |
|------------|------------|--------------------------------|
| Go             | 1.25+      | Backend development       |
| Node.js        | 22+ (LTS)  | Frontend development             |
| Docker     | 24+        | Containerized development      |
| Docker Compose | 2.24+  | Multi-service orchestration    |
| PostgreSQL | 16         | Primary database               |
| Git        | 2.40+      | Version control                |

### Recommended Tools

- **curl** or **HTTPie** — API testing
- **jq** — JSON response parsing
- **pgAdmin** or **DBeaver** — database inspection
- **Prometheus + Grafana** — metrics (production)

---

## Quick Start for Developers

### 1. Clone the Repository

```bash
git clone <repository-url> ai-software-factory
cd ai-software-factory
```

### 2. Start with Docker Compose

```bash
# Create .env from defaults
cp .env.example .env

# Build and start all services
docker compose up -d --build

# Verify health
curl http://localhost:8080/v1/healthz
# → {"status":"ok"}
```

Services start on these ports:

| Service   | Port  |
|-----------|-------|
| API       | 8080  |
| Frontend  | 3000  |
| PostgreSQL| 5432  |

### 3. Authenticate and Try the API

```bash
# Create a user
curl -s -X POST http://localhost:8080/v1/users/register \
  -H "Content-Type: application/json" \
  -d '{"email":"dev@example.com","password":"secure123","name":"Dev User"}' | jq .

# Login to get a JWT
TOKEN=$(curl -s -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"dev@example.com","password":"secure123"}' | jq -r '.access_token')

# Create a project
curl -s -X POST http://localhost:8080/v1/projects \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Hello World","description":"A test project","template":"web-app"}' | jq .
```

### 4. Run Without Docker (Native)

**Backend:**

```bash
cd src/
go build -o bin/api ./cmd/main.go

# Set config (defaults work for local dev)
export PORT=8080
export LOG_LEVEL=debug

./bin/api
```

**Frontend:**

```bash
cd frontend/
npm install
npm run dev    # → http://localhost:3000
```

---

## Architecture Overview

### Service Layout

```
┌─────────────────────────────────────────────────────────┐
│                    CLIENT LAYER                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │ Web App  │  │ CLI      │  │ API      │              │
│  │ (Next.js)│  │ (Future) │  │ Clients  │              │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘              │
│       └──────────────┴──────────────┘                   │
└───────────────────────┬─────────────────────────────────┘
                        │ HTTPS
┌───────────────────────┴─────────────────────────────────┐
│                    API GATEWAY                            │
│  Auth · Rate Limiting · Routing · TLS Termination        │
└───────────────────────┬─────────────────────────────────┘
                        │ Internal Network
┌───────────────────────┴─────────────────────────────────┐
│                    SERVICE LAYER                          │
│                                                          │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │
│  │ Project  │ │  Agent   │ │  Code    │ │  Review  │  │
│  │ Service  │ │  Orch.   │ │ Service  │ │ Service  │  │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │
│  │    QA    │ │  Deploy  │ │Notifica- │ │  User    │  │
│  │ Service  │ │ Service  │ │tion Svc  │ │ Service  │  │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘  │
│  ┌──────────┐ ┌──────────┐                              │
│  │Analytics │ │ Webhook  │                              │
│  │ Service  │ │ Service  │                              │
│  └──────────┘ └──────────┘                              │
└───────────────────────┬─────────────────────────────────┘
                        │
┌───────────────────────┴─────────────────────────────────┐
│                     DATA LAYER                            │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │
│  │PostgreSQL│ │  Redis   │ │  S3/MinIO│ │ Gitea/Git│  │
│  │ (Primary)│ │ (Cache)  │ │(Artifacts│ │ (Code)   │  │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘  │
└─────────────────────────────────────────────────────────┘
```

### Technology Stack

| Layer        | Technology                              |
|-------------|------------------------------------------|
| Backend      | Go 1.25, Gin Framework                   |
| Frontend     | Next.js 16, React 19, TypeScript 5      |
| Styling      | Tailwind CSS 4                          |
| State Mgmt   | TanStack React Query, Zustand           |
| Drag & Drop  | dnd-kit                                 |
| Database     | PostgreSQL 16                           |
| Cache        | Redis 7                                 |
| Container    | Docker, Docker Compose                  |
| CI/CD        | GitHub Actions                          |
| Monitoring   | Prometheus + Grafana                    |
| Logging      | Structured (zap) / ELK                  |

### Communication Patterns

- **Command (sync):** REST over HTTP (Gin) — creates, reads, and mutates resources
- **Events (async):** Scheduled or agent-initiated — project status changes, build completions
- **Real-time (SSE):** Event streams for live agent status updates (planned)

---

## API Reference

### Base URL

```
Production: https://api.ai-software-factory.com/v1
Development: http://localhost:8080/v1
```

### Versioning

The API version is embedded in the URL path (`/v1/`, `/v2/`). Breaking changes
increment the major version. Non-breaking additions (new endpoints, new fields) are
added to the current version.

**Deprecation header:** `Sunset: Sat, 01 Jan 2027 00:00:00 GMT`

### Common Headers

| Header           | When       | Description                       |
|-----------------|------------|-----------------------------------|
| `Authorization` | Always     | `Bearer <jwt_or_api_key>`         |
| `Content-Type`  | POST/PATCH | `application/json`                |
| `X-Request-ID`  | Response   | Unique request identifier         |
| `X-RateLimit-Limit` | Response | Requests/hour quota              |
| `X-RateLimit-Remaining` | Response | Remaining quota                |
| `X-RateLimit-Reset` | Response | Unix timestamp of quota reset   |

### Response Envelope

All responses use a consistent JSON envelope:

```json
{
  "data": { ... },
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150,
    "pages": 8
  }
}
```

For single-resource responses, `data` is an object. For list endpoints, `data` is an
array. Pagination is included on list endpoints and omitted on single-resource ones.

### Error Response Format

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

---

### Authentication

#### POST /auth/login

Exchange credentials for a JWT access token.

**Request:**

```json
{
  "email": "user@example.com",
  "password": "securepassword"
}
```

**Response (200):**

```json
{
  "access_token": "eyJhbG...",
  "refresh_token": "dGhpcyBpcyBhIHJlZnJl...",
  "expires_in": 86400
}
```

| Field           | Type   | Description                          |
|-----------------|--------|--------------------------------------|
| `access_token`  | string | JWT for API authentication (24h)     |
| `refresh_token` | string | Token to obtain a new access token   |
| `expires_in`    | int    | Seconds until access token expiry    |

**Error codes:**

| HTTP Status | Code              | Meaning                    |
|-------------|-------------------|----------------------------|
| 400         | VALIDATION_ERROR  | Missing email or password  |
| 401         | UNAUTHORIZED      | Invalid credentials        |

#### POST /auth/refresh

Obtain a new access token using a refresh token.

**Request:**

```json
{
  "refresh_token": "dGhpcyBpcyBhIHJlZnJl..."
}
```

**Response (200):**

Same shape as `/auth/login`.

#### API Key Authentication

Alternatively, authenticate with a static API key:

```http
Authorization: Bearer ak_123...cdef
```

API keys are generated per user in the dashboard and do not expire.

---

### Projects

#### POST /projects

Create a new project.

**Request:**

```json
{
  "name": "E-commerce Platform",
  "description": "Build a modern e-commerce platform with React frontend and Go backend",
  "template": "web-app"
}
```

| Field         | Type   | Required | Description                                |
|---------------|--------|----------|--------------------------------------------|
| `name`        | string | Yes      | Project display name (3–255 chars)         |
| `description` | string | No       | Full project specification                  |
| `template`    | string | No       | Project template: `web-app`, `api`, `cli`  |

**Response (201):**

```json
{
  "id": "proj_abc123",
  "name": "E-commerce Platform",
  "status": "initializing",
  "created_at": "2026-06-10T10:00:00Z",
  "agents_spawned": ["pm"]
}
```

**Status lifecycle:** `initializing` → `in_progress` → `completed` | `failed`

| HTTP Status | Code              | Meaning                          |
|-------------|-------------------|----------------------------------|
| 201         | —                 | Project created successfully     |
| 400         | VALIDATION_ERROR  | Missing or invalid name          |
| 401         | UNAUTHORIZED      | Invalid or missing auth token    |
| 429         | RATE_LIMITED      | Quota exceeded                   |

#### GET /projects

List projects with optional filters.

**Query parameters:**

| Parameter | Type   | Default | Description                |
|-----------|--------|---------|----------------------------|
| `status`  | string | —       | Filter: `active`, `completed`, `failed` |
| `page`    | int    | 1       | Page number                |
| `limit`   | int    | 20      | Items per page (max 100)   |
| `sort`    | string | `created_at` | Sort field            |
| `order`   | string | `desc`  | Sort order: `asc`, `desc`  |

**Response (200):**

```json
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
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150,
    "pages": 8
  }
}
```

#### GET /projects/:id

Get detailed project information.

**Response (200):**

```json
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

| HTTP Status | Code           | Meaning                    |
|-------------|----------------|----------------------------|
| 200         | —              | Project found              |
| 404         | NOT_FOUND      | Project ID does not exist  |

---

### Agents

#### POST /agents/spawn

Spawn a new AI agent for a project.

**Request:**

```json
{
  "project_id": "proj_abc123",
  "type": "developer",
  "config": {
    "model": "gpt-4",
    "temperature": 0.3
  }
}
```

| Field          | Type   | Required | Description                              |
|----------------|--------|----------|------------------------------------------|
| `project_id`   | string | Yes      | Target project ID                        |
| `type`         | string | Yes      | Agent role: `pm`, `architect`, `developer`, `reviewer`, `qa` |
| `config.model` | string | No       | LLM model override (default: system-configured) |
| `config.temperature` | float | No  | Model temperature 0.0–1.0 (default: 0.3) |

**Response (201):**

```json
{
  "id": "agent_dev_001",
  "type": "developer",
  "status": "spawning",
  "project_id": "proj_abc123"
}
```

**Agent lifecycle:** `spawning` → `idle` → `working` → `completed` | `failed`

#### GET /agents

List active agents.

**Query parameters:**

| Parameter    | Type   | Description          |
|-------------|--------|----------------------|
| `project_id` | string | Filter by project    |
| `status`     | string | Filter by status     |

**Response (200):**

```json
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
  ],
  "pagination": { ... }
}
```

#### POST /agents/:id/assign

Assign a task to an agent.

**Request:**

```json
{
  "task_id": "task_001",
  "priority": "high",
  "context": {
    "files": ["src/api/users.ts", "src/models/user.ts"]
  }
}
```

| Field             | Type   | Required | Description                          |
|-------------------|--------|----------|--------------------------------------|
| `task_id`         | string | Yes      | Task to assign                       |
| `priority`        | string | No       | `low`, `medium`, `high`, `critical`  |
| `context.files`   | array  | No       | Relevant file paths for context      |

**Response (200):**

```json
{
  "id": "agent_dev_001",
  "task_id": "task_001",
  "status": "working",
  "estimated_completion": "2026-06-10T11:00:00Z"
}
```

---

### Tasks

#### POST /projects/:projectId/tasks

Create a task within a project.

**Request:**

```json
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
```

| Field                | Type   | Required | Description                                    |
|----------------------|--------|----------|------------------------------------------------|
| `title`              | string | Yes      | Task title (3–500 chars)                       |
| `description`        | string | No       | Detailed task description                      |
| `type`               | string | Yes      | `implementation`, `bugfix`, `refactor`, `test`, `documentation` |
| `acceptance_criteria`| array  | No       | List of pass/fail conditions                   |
| `priority`           | string | No       | `must_have`, `should_have`, `could_have`, `wont_have` |
| `estimated_hours`    | float  | No       | Estimated effort in hours                      |

**Response (201):**

```json
{
  "id": "task_001",
  "title": "Implement user authentication API",
  "status": "backlog",
  "created_at": "2026-06-10T10:00:00Z"
}
```

**Task lifecycle:** `backlog` → `ready` → `in_progress` → `review` → `done` | `blocked` | `cancelled`

#### PATCH /tasks/:id

Update task status (e.g., move through workflow).

**Request:**

```json
{
  "status": "in_progress",
  "assignee_agent_id": "agent_dev_001"
}
```

| Field               | Type   | Required | Description                    |
|---------------------|--------|----------|--------------------------------|
| `status`            | string | Yes      | New status (see lifecycle)     |
| `assignee_agent_id` | string | No       | Agent to assign                |

**Response (200):**

```json
{
  "id": "task_001",
  "status": "in_progress",
  "assignee_agent_id": "agent_dev_001",
  "updated_at": "2026-06-10T10:30:00Z"
}
```

---

### Code Generation

#### POST /code/generate

Generate code for a task asynchronously.

**Request:**

```json
{
  "project_id": "proj_abc123",
  "task_id": "task_001",
  "specification": "Implement JWT authentication with Gin...",
  "files": ["src/auth/login.ts", "src/auth/register.ts"]
}
```

| Field           | Type   | Required | Description                        |
|-----------------|--------|----------|------------------------------------|
| `project_id`    | string | Yes      | Project ID                         |
| `task_id`       | string | Yes      | Task ID                            |
| `specification` | string | Yes      | Implementation specification       |
| `files`         | array  | No       | Target file paths                  |

**Response (202):**

```json
{
  "id": "code_gen_001",
  "status": "generating",
  "estimated_time": 300
}
```

The generation is asynchronous. Poll or set up a webhook to be notified when complete.

#### GET /code/:projectId/files/{path}

Retrieve generated file content.

**Response (200):**

```json
{
  "path": "src/auth/login.ts",
  "content": "import jwt from 'jsonwebtoken'...",
  "language": "typescript",
  "size": 2048,
  "last_modified": "2026-06-10T10:45:00Z",
  "modified_by": "agent_dev_001"
}
```

The `{path}` parameter supports wildcard paths (`src/auth/*.ts`).

| HTTP Status | Code           | Meaning                       |
|-------------|----------------|-------------------------------|
| 200         | —              | File found                    |
| 404         | FILE_NOT_FOUND | File does not exist in project|

#### POST /code/:projectId/commits

Create a commit with generated code.

**Request:**

```json
{
  "branch": "feature/auth",
  "message": "feat: implement JWT authentication",
  "files": [
    { "path": "src/auth/login.ts", "content": "..." },
    { "path": "src/auth/register.ts", "content": "..." }
  ]
}
```

**Response (201):**

```json
{
  "sha": "abc123def456",
  "message": "feat: implement JWT authentication",
  "author": "agent_dev_001",
  "created_at": "2026-06-10T10:50:00Z"
}
```

| HTTP Status | Code             | Meaning                   |
|-------------|------------------|---------------------------|
| 201         | —                | Commit created            |
| 400         | VALIDATION_ERROR | Missing branch or message |
| 409         | CONFLICT         | Branch has diverged       |

---

### Code Reviews

#### POST /reviews

Request a code review.

**Request:**

```json
{
  "project_id": "proj_abc123",
  "commit_sha": "abc123def456",
  "reviewer_type": "automated"
}
```

| Field           | Type   | Required | Description                             |
|-----------------|--------|----------|-----------------------------------------|
| `project_id`    | string | Yes      | Project ID                              |
| `commit_sha`    | string | Yes      | Commit SHA to review                    |
| `reviewer_type` | string | No       | `automated` (default), `human`, `both`  |

**Response (201):**

```json
{
  "id": "review_001",
  "status": "in_progress",
  "reviewer": "review_agent_001"
}
```

#### GET /reviews/:id

Get review results.

**Response (200):**

```json
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
    "duplications": 0,
    "lint_errors": 0
  }
}
```

**Result values:** `approved`, `changes_requested`, `rejected`

| HTTP Status | Code           | Meaning                  |
|-------------|----------------|--------------------------|
| 200         | —              | Review found             |
| 404         | NOT_FOUND      | Review ID does not exist |

**Issue severity levels:** `critical`, `error`, `warning`, `info`

---

### Deployments

#### POST /deployments

Trigger a deployment asynchronously.

**Request:**

```json
{
  "project_id": "proj_abc123",
  "environment": "staging",
  "branch": "main"
}
```

| Field         | Type   | Required | Description                              |
|---------------|--------|----------|------------------------------------------|
| `project_id`  | string | Yes      | Project ID                               |
| `environment` | string | Yes      | `development`, `staging`, `production`   |
| `branch`      | string | Yes      | Git branch to deploy                     |

**Response (202):**

```json
{
  "id": "deploy_001",
  "status": "queued",
  "environment": "staging",
  "estimated_time": 600
}
```

#### GET /deployments/:id

Get deployment status and details.

**Response (200):**

```json
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

**Deployment lifecycle:** `queued` → `building` → `testing` → `deploying` → `completed` | `failed` | `rolled_back`

#### POST /deployments/:id/rollback

Rollback a deployment to the previous version.

**Response:**

```json
{
  "id": "deploy_002",
  "status": "rolling_back",
  "rollback_from": "deploy_001",
  "rollback_to": "deploy_000"
}
```

---

### Users

#### POST /users/register

Create a new user account.

**Request:**

```json
{
  "email": "user@example.com",
  "password": "securepassword123",
  "name": "John Doe"
}
```

| Field      | Type   | Required | Constraints                  |
|------------|--------|----------|------------------------------|
| `email`    | string | Yes      | Valid email format, unique   |
| `password` | string | Yes      | Min 8 chars, 1 uppercase, 1 number |
| `name`     | string | Yes      | 1–255 chars                  |

**Response (201):**

```json
{
  "id": "user_001",
  "email": "user@example.com",
  "name": "John Doe",
  "created_at": "2026-06-10T10:00:00Z"
}
```

| HTTP Status | Code              | Meaning                    |
|-------------|-------------------|----------------------------|
| 201         | —                 | User created               |
| 400         | VALIDATION_ERROR  | Invalid fields             |
| 409         | CONFLICT          | Email already exists       |

#### GET /users/me

Get the authenticated user's profile.

**Response (200):**

```json
{
  "id": "user_001",
  "email": "user@example.com",
  "name": "John Doe",
  "role": "admin",
  "teams": ["team_001"],
  "projects": ["proj_abc123"]
}
```

| Field      | Type   | Description                     |
|------------|--------|---------------------------------|
| `role`     | string | `user`, `admin`                 |
| `teams`    | array  | Team IDs the user belongs to    |
| `projects` | array  | Project IDs the user owns       |

---

### Webhooks

#### POST /webhooks

Register a webhook to receive event notifications.

**Request:**

```json
{
  "url": "https://your-server.com/webhook",
  "events": ["project.completed", "deploy.failed"],
  "secret": "your_webhook_secret"
}
```

| Field    | Type   | Required | Description                           |
|----------|--------|----------|---------------------------------------|
| `url`    | string | Yes      | HTTPS endpoint to receive payloads    |
| `events` | array  | Yes      | Event types to subscribe to           |
| `secret` | string | No       | HMAC secret for payload signing       |

**Available events:**

| Event                  | Description                        |
|------------------------|------------------------------------|
| `project.completed`    | Project finished                   |
| `project.failed`       | Project failed                     |
| `agent.status_change`  | Agent state transition             |
| `task.completed`       | Task marked done                   |
| `code.generated`       | Code generation completed          |
| `review.completed`     | Review completed                   |
| `deploy.completed`     | Deployment succeeded               |
| `deploy.failed`        | Deployment failed                  |

**Response (201):**

```json
{
  "id": "wh_001",
  "url": "https://your-server.com/webhook",
  "events": ["project.completed", "deploy.failed"],
  "active": true,
  "created_at": "2026-06-10T10:00:00Z"
}
```

#### Webhook Payload Format

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

**Verification:** Compute HMAC-SHA256 of the raw request body using your webhook
secret. Compare to the `signature` header value. Example:

```go
// Go verification
func VerifyWebhook(body []byte, signature, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(body)
    expected := hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(signature))
}
```

---

### Rate Limits

| Plan       | Requests/Hour | Concurrent Agents |
|------------|---------------|-------------------|
| Free       | 100           | 1                 |
| Pro        | 1,000         | 5                 |
| Enterprise | 10,000        | 20                |

Rate limits are enforced per API key or user. Headers are returned on every response:

```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1718035200
```

When exceeded, the API returns `429 Too Many Requests` with code `RATE_LIMITED`.

---

## Frontend Architecture

### Directory Layout

```
frontend/
├── src/
│   ├── app/              # Next.js App Router pages
│   │   ├── page.tsx      # Dashboard / project list
│   │   ├── layout.tsx    # Root layout
│   │   └── projects/
│   │       └── [id]/
│   │           ├── page.tsx      # Project detail
│   │           ├── board/        # Kanban board view
│   │           └── settings/     # Project settings
│   ├── components/       # Shared React components
│   ├── hooks/            # Custom React hooks
│   ├── lib/              # API client, utilities
│   └── styles/           # Global CSS / Tailwind
├── public/               # Static assets
├── next.config.ts
├── package.json
└── tsconfig.json
```

### Key Dependencies

| Package                  | Purpose                         |
|--------------------------|---------------------------------|
| `next`                   | React framework (SSR/SSG)       |
| `react` / `react-dom`    | UI library                      |
| `@tanstack/react-query`  | Server state management         |
| `@dnd-kit/core`          | Drag-and-drop Kanban board      |
| `tailwindcss`            | Utility-first CSS               |
| `clsx`                   | Conditional class names         |

### API Client

The frontend communicates with the backend through a typed API client in `src/lib/api.ts`.
It handles:

- JWT token storage and automatic refresh
- Request normalization (auth headers, Content-Type)
- Error handling and retry logic
- Response type inference

---

## Backend Architecture

### Directory Layout

```
src/
├── cmd/
│   └── main.go           # HTTP server entry point
├── internal/
│   ├── config/           # Configuration loader (env vars)
│   │   └── config.go
│   ├── handler/          # HTTP handlers (one file per resource)
│   │   ├── auth.go       # Login, refresh
│   │   ├── project.go    # CRUD projects
│   │   ├── agent.go      # Spawn, list, assign agents
│   │   ├── task.go       # Create, update tasks
│   │   ├── code.go       # Generate code, files, commits
│   │   ├── review.go     # Create, get reviews
│   │   ├── deployment.go # Trigger, status, rollback
│   │   ├── user.go       # Register, profile
│   │   ├── webhook.go    # Register webhooks
│   │   └── types.go      # Shared types (APIResponse, ErrorResponse)
│   ├── logger/           # Structured logging (zap)
│   │   └── logger.go
│   ├── middleware/        # HTTP middleware chain
│   │   └── middleware.go  # CORS, RequestID, Recovery, Logger, Auth
│   ├── model/            # Domain struct definitions
│   ├── router/           # Route registration
│   │   └── router.go
│   └── service/          # Business logic layer
├── pkg/
│   └── errors/           # Custom error types
│       └── errors.go
├── db/                   # Database migrations, seeds
├── Dockerfile
└── go.mod
```

### Package Responsibilities

| Package      | Responsibility                                        |
|--------------|------------------------------------------------------|
| `cmd`        | Bootstrap: config → logger → router → serve          |
| `handler`    | Gin layer: parse request, validate, call service, format response |
| `service`    | Business logic: orchestration, validation rules      |
| `model`      | Domain types (Project, Agent, Task, etc.)            |
| `middleware` | Gin middleware chain: CORS → RequestID → Recovery → Logger → Auth |
| `router`     | Gin engine and route group definitions               |
| `config`     | Environment variable parsing and defaults            |

### Middleware Chain

Requests pass through Gin middleware in this order (outer → inner):

1. **CORS** — Permissive headers for development (configurable for production)
2. **RequestID** — Attaches `X-Request-ID` to every response
3. **Recovery** — Catches panics, returns 500 with a logged stack trace
4. **Logger** — Logs method, path, remote address, status, and duration
5. **RateLimit** — Enforces requests/hour quotas
6. **Auth** — Validates JWT (`Bearer eyJ...`) or API Key (`Bearer ak_...`).
   Skips auth for healthz, login, and register endpoints.

### Configuration

All configuration is via environment variables:

| Variable      | Default    | Description                     |
|---------------|------------|---------------------------------|
| `PORT`        | `8080`     | HTTP listen port                |
| `LOG_LEVEL`   | `info`     | `debug`, `info`, `warn`, `error` |
| `DB_HOST`     | `localhost`| PostgreSQL host                 |
| `DB_PORT`     | `5432`     | PostgreSQL port                 |
| `DB_USER`     | `postgres` | Database user                   |
| `DB_PASSWORD` | `postgres` | Database password               |
| `DB_NAME`     | `project`  | Database name                   |

### In-Memory Store (Development)

Currently, the API uses an in-memory data store (see `internal/service/memory.go`).
The PostgreSQL schema and Docker Compose service are ready — migrating to the
database-backed store requires swapping the store implementation via
the service factory.

---

## Database Schema

### Entity Relationships

```
users ──< projects ──< agents
  │                     │
  │              ┌──────┘
  │              ▼
  ├──< team_members >── teams
  │
  ▼
tasks ──< code_artifacts
  │
  ├──< reviews
  │
  └──< deployments
```

### Core Table Reference

| Table            | Key Columns                                      | Purpose                    |
|------------------|--------------------------------------------------|----------------------------|
| `users`          | id, email, password_hash, role                   | User accounts              |
| `projects`       | id, name, status, owner_id (FK), config (JSONB)  | Project lifecycle          |
| `agents`         | id, project_id (FK), type, status, config (JSONB)| AI agent instances         |
| `tasks`          | id, project_id (FK), title, status, priority     | Work items                 |
| `code_artifacts` | id, task_id (FK), file_path, content, language   | Generated source files     |
| `reviews`        | id, task_id (FK), commit_sha, score, issues (JSONB)| Code reviews             |
| `deployments`    | id, project_id (FK), environment, status, url    | Deployment history         |
| `webhook_config` | id, project_id (FK), url, events (JSONB), secret | Webhook subscriptions      |

Full DDL and index definitions: see `docs/database.md`.

### Data Retention

| Table            | Hot Data | Cold Data    | Deletion      |
|------------------|----------|--------------|---------------|
| audit_logs       | 90 days  | 1 year (S3)  | After 1 year  |
| code_artifacts   | Current  | Git history  | Never         |
| deployments       | Last 50  | Archived     | After 180 days|

---

## Contributing Guide

### Development Workflow

We use trunk-based development with short-lived feature branches.

```
main  ────●─────────●─────────●─────────
           \         /         /
feature/   └─●─●─●─┘
```

#### Branch Naming

| Pattern                     | Example                        |
|-----------------------------|--------------------------------|
| `feat/<short-description>`  | `feat/github-oauth`            |
| `fix/<short-description>`   | `fix/rate-limit-overcount`     |
| `refactor/<description>`    | `refactor/agent-service`       |
| `docs/<description>`        | `docs/api-rate-limits`         |
| `test/<description>`        | `test/auth-flows`               |

#### Commit Message Convention

```
<type>(<scope>): <short summary>

<optional body>
```

Types: `feat`, `fix`, `refactor`, `docs`, `test`, `chore`, `perf`

Examples:

```
feat(api): add rate limit headers to all responses
fix(agent): handle LLM timeout with retry
docs(api): document webhook event types
test(auth): add refresh token expiry test
```

### Code Standards

#### Go Backend

- **Formatting:** Run `gofmt` (or `go fmt ./...`) before every commit
- **Linting:** `go vet ./...` — zero warnings before PR
- **Imports:** Standard library → third-party → internal (grouped with blank lines)
- **Error handling:** Wrap errors with context using `fmt.Errorf("...: %w", err)`.
  Never use `log.Fatal` outside `main.go`.
- **Naming:** Follow Go conventions — `camelCase` for unexported, `PascalCase` for
  exported. Acronyms are all-caps: `HTTP`, `API`, `ID`.
- **Handler pattern:** Each handler file exports a struct with methods (e.g.,
  `ProjectHandler` with `Create`, `List`, `Get`). All input validation happens
  in the handler; business logic lives in `service/`.

#### TypeScript Frontend

- **Formatting:** Prettier with default config
- **Linting:** `npm run lint` (ESLint) — zero warnings before PR
- **Types:** Strict TypeScript — avoid `any`. Use `interface` for public API
  shapes, `type` for unions and utilities.
- **Components:** Prefer function components with hooks. One component per file.
- **Naming:** `PascalCase` for components, `camelCase` for functions and variables.
  Files match the exported name: `ProjectCard.tsx`.

#### General

- **No commented-out code** — delete it. Git history preserves it.
- **No dead code** — if it's not used, remove it.
- **Document public APIs** — every exported function and type needs a doc comment.
- **Keep functions small** — under 40 lines where possible.
- **Async operations** — use context cancellation; never ignore the returned
  context's Done channel.

### Testing

#### Backend Tests

```bash
cd src/

# Run all tests
go test ./... -v -count=1

# Run with race detector
go test ./... -race -count=1

# Run a specific package
go test ./internal/service/... -v

# Run a single test
go test ./internal/handler/... -run TestProjectCreate -v

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

**Test requirements:**

- **Unit tests** for all service-layer functions
- **Handler tests** using `httptest` for request/response validation
- **No external dependencies** in unit tests — mock the store interface
- Minimum **70% coverage** on new code (enforced by CI)

#### Frontend Tests

```bash
cd frontend/

# Run tests
npm test

# Run with coverage
npm test -- --coverage
```

**Test requirements:**

- **Component tests** for all page-level and shared components
- **Hook tests** for custom hooks
- **Integration tests** for key user flows (project creation, task assignment)

#### API Integration Tests

```bash
# Start the API server
./scripts/test.sh
```

The test script starts the API, runs a suite of curl-based integration tests, and
reports pass/fail per endpoint.

**What integration tests cover:**

- Authentication flows (login, refresh, invalid tokens)
- CRUD operations on each resource
- Error cases (validation, not found, conflicts)
- Pagination edge cases
- Asynchronous operation polling

### Pull Request Process

#### Before Submitting

1. Pull the latest `main` and rebase your branch
2. Run the full test suite locally
3. Run linters (`go vet`, `npm run lint`)
4. Build the project (`go build`, `npm run build`)
5. Add or update documentation for any API changes

#### PR Template

```markdown
## Description
Brief description of what this PR does.

## Related Issue
Closes #ISSUE_NUMBER

## Type of Change
- [ ] feat: new feature
- [ ] fix: bug fix
- [ ] refactor: code improvement
- [ ] docs: documentation
- [ ] test: test addition/fix

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No new warnings
- [ ] Tests added for new functionality
```

#### Review Process

1. At least **one approving review** required for merge
2. All **CI checks** must pass (tests, lint, build)
3. No merge commits — use **squash merge** or **rebase merge**
4. Reviewers check for:
   - Correctness and edge cases
   - Test coverage
   - Documentation impact
   - Performance implications
   - Security (auth, input validation, SQL injection)

#### After Merge

- Delete the feature branch
- Monitor CI/CD deployment
- Update any dependent tasks or issues

---

## Operations

### Building and Running

#### Docker Compose (Recommended for Dev)

```bash
docker compose up -d --build
```

#### Native Build

**Backend:**

```bash
cd src/
CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/api ./cmd/main.go
PORT=8080 LOG_LEVEL=debug ./bin/api
```

**Frontend:**

```bash
cd frontend/
npm run build
npm start
```

#### Production Build

The Dockerfile uses a multi-stage build:

```dockerfile
# Stage 1: Build
FROM golang:1.25-alpine AS builder
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /build/api ./cmd/main.go

# Stage 2: Runtime (minimal)
FROM alpine:3.20
COPY --from=builder /build/api /app/api
USER appuser
ENTRYPOINT ["/app/api"]
```

The binary is statically linked and runs as a non-root user.

### Logging and Monitoring

#### Structured Logging

The backend uses `go.uber.org/zap` for structured logging:

```
{"level":"info","time":"2026-06-10T12:00:00Z","msg":"request completed",
 "method":"GET","path":"/v1/projects","status":200,"duration":45}
```

Set `LOG_LEVEL=debug` for verbose output during development.

#### Health Check

The API exposes a health check endpoint:

```bash
curl http://localhost:8080/v1/healthz
# → {"status":"ok"}
```

The Docker Compose setup uses this for container health checks with
30s intervals and 3 retries.

#### Metrics (Planned)

- Prometheus metrics at `/v1/metrics`
- Request count, latency (p50/p95/p99), error rate per endpoint
- Active agent count, queue depth
- LLM token usage

### Debugging

#### Common Issues

| Symptom                  | Likely Cause                           | Fix                              |
|--------------------------|----------------------------------------|----------------------------------|
| `401 Unauthorized`       | Missing or expired JWT                 | Re-authenticate, refresh token   |
| `429 Too Many Requests`  | Rate limit exceeded                    | Wait for reset or upgrade plan   |
| Connection refused       | Services not started (Docker)          | `docker compose ps` to check     |
| `404 NOT_FOUND`          | Wrong resource ID or path              | Verify ID, check API version     |
| Slow responses           | No database connection (using memory)  | Check DB_HOST, verify PostgreSQL |

#### Debug Mode

```bash
# Start API in debug mode
LOG_LEVEL=debug PORT=8080 ./bin/api
```

This enables:
- Full request/response logging
- Stack traces on errors
- SQL query logging (when database is connected)

#### Database Inspection

```bash
# Connect to PostgreSQL via Docker
docker compose exec db psql -U postgres -d project

# List tables
\l
\dt
\d+ projects

# Query
SELECT id, name, status FROM projects;
```

---

## Appendix: Error Codes

| HTTP Code | Code                    | Description                    |
|-----------|------------------------|--------------------------------|
| 400       | VALIDATION_ERROR       | Request body failed validation |
| 401       | UNAUTHORIZED           | Missing or invalid auth token  |
| 403       | FORBIDDEN              | Insufficient permissions       |
| 404       | NOT_FOUND              | Resource does not exist        |
| 409       | CONFLICT               | Resource conflict (duplicate)  |
| 422       | UNPROCESSABLE_ENTITY   | Semantic validation failure    |
| 429       | RATE_LIMITED           | API quota exceeded             |
| 500       | INTERNAL_ERROR         | Unexpected server error        |
| 502       | BAD_GATEWAY            | Upstream service unavailable   |
| 503       | SERVICE_UNAVAILABLE    | System in maintenance          |

All errors return the standard envelope:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Human-readable description",
    "details": [
      { "field": "email", "message": "Invalid email format" }
    ]
  },
  "request_id": "req_abc123"
}
```

The `request_id` is included in every error response and correlates with server-side
logs for debugging.

---

> **Document Version**: 1.0 | **Last Updated**: 2026-06-10 | **Applies To**: API v1
>
> For user-oriented documentation, see the [User Guide](./user-guide.md).
> For the API specification, see [API Spec](./api-spec.md).
> For the platform security architecture, see [Security Architecture](./security.md).
