# AI Software Factory — Service Architecture

## Overview

The AI Software Factory is decomposed into 10 microservices, each with a clear responsibility boundary. Services communicate via REST APIs and asynchronous events.

## Service Definitions

### 1. API Gateway

**Responsibility:** Single entry point for all client requests. Handles routing, authentication, rate limiting, and request/response transformation.

**API Surface:**
- `POST /api/v1/auth/login` — User login
- `POST /api/v1/auth/register` — User registration
- `GET /api/v1/projects` — List projects
- `POST /api/v1/projects` — Create project
- `GET /api/v1/projects/:id` — Get project details
- `WebSocket /ws/projects/:id` — Real-time project updates

**Data Ownership:** None (stateless proxy)

**Communication:** Forwards requests to appropriate backend services

**Deployment:** 3+ replicas behind load balancer

**Scaling:** Horizontal, based on request throughput

---

### 2. Project Service

**Responsibility:** Manages project lifecycle, metadata, and status tracking.

**API Surface:**
- `POST /api/v1/projects` — Create project
- `GET /api/v1/projects/:id` — Get project
- `PATCH /api/v1/projects/:id` — Update project
- `DELETE /api/v1/projects/:id` — Archive project
- `GET /api/v1/projects/:id/status` — Get project status
- `GET /api/v1/projects/:id/timeline` — Get project timeline

**Data Ownership:** `projects`, `project_members`, `project_artifacts`

**Communication:**
- Publishes: `project.created`, `project.updated`, `project.archived`
- Subscribes: `task.completed` (updates project progress)

**Deployment:** 2 replicas

**Scaling:** Vertical (database-bound)

---

### 3. Agent Orchestrator

**Responsibility:** Manages agent lifecycle, task assignment, and coordination between agents.

**API Surface:**
- `POST /api/v1/agents/spawn` — Spawn new agent
- `GET /api/v1/agents` — List active agents
- `GET /api/v1/agents/:id` — Get agent status
- `POST /api/v1/agents/:id/assign` — Assign task to agent
- `POST /api/v1/agents/:id/terminate` — Terminate agent
- `GET /api/v1/agents/:id/activity` — Get agent activity log

**Data Ownership:** `agents`, `agent_tasks`, `agent_activity_log`

**Communication:**
- Publishes: `agent.spawned`, `task.assigned`, `task.completed`, `agent.failed`
- Subscribes: `project.created` (spawns PM agent), `review.completed` (triggers next step)

**Deployment:** 2 replicas + background worker

**Scaling:** Horizontal, based on active agent count

---

### 4. Code Service

**Responsibility:** Manages code generation, storage, versioning, and repository interactions.

**API Surface:**
- `POST /api/v1/code/generate` — Generate code from spec
- `GET /api/v1/code/:projectId/files` — List project files
- `GET /api/v1/code/:projectId/files/:path` — Get file content
- `POST /api/v1/code/:projectId/commits` — Create commit
- `GET /api/v1/code/:projectId/branches` — List branches
- `POST /api/v1/code/:projectId/merge` — Merge branch

**Data Ownership:** `code_artifacts`, `code_reviews` (metadata only)

**Communication:**
- Publishes: `code.committed`, `code.merged`, `code.conflict`
- Subscribes: `task.assigned` (generates code for implementation tasks)

**Deployment:** 2 replicas

**Scaling:** Horizontal, based on commit throughput

---

### 5. Review Service

**Responsibility:** Performs automated code review, enforces quality standards, and manages review workflows.

**API Surface:**
- `POST /api/v1/reviews` — Create review request
- `GET /api/v1/reviews/:id` — Get review results
- `POST /api/v1/reviews/:id/approve` — Approve review
- `POST /api/v1/reviews/:id/reject` — Reject with feedback
- `GET /api/v1/reviews/project/:projectId` — List project reviews

**Data Ownership:** `reviews`, `review_comments`, `quality_metrics`

**Communication:**
- Publishes: `review.approved`, `review.rejected`, `review.completed`
- Subscribes: `code.committed` (triggers review)

**Deployment:** 2 replicas

**Scaling:** Horizontal, based on review queue depth

---

### 6. QA Service

**Responsibility:** Creates test plans, executes tests, tracks coverage, and manages test infrastructure.

**API Surface:**
- `POST /api/v1/qa/test-plans` — Create test plan
- `POST /api/v1/qa/runs` — Trigger test run
- `GET /api/v1/qa/runs/:id` — Get test results
- `GET /api/v1/qa/coverage/:projectId` — Get coverage report
- `POST /api/v1/qa/environments` — Provision test environment

**Data Ownership:** `test_plans`, `test_runs`, `test_results`, `coverage_reports`

**Communication:**
- Publishes: `test.passed`, `test.failed`, `test.completed`
- Subscribes: `deploy.completed` (triggers test execution)

**Deployment:** 2 replicas + test runner workers

**Scaling:** Horizontal, based on test queue depth

---

### 7. Deploy Service

**Responsibility:** Manages CI/CD pipelines, deployments, rollbacks, and environment management.

**API Surface:**
- `POST /api/v1/deployments` — Trigger deployment
- `GET /api/v1/deployments/:id` — Get deployment status
- `POST /api/v1/deployments/:id/rollback` — Rollback deployment
- `GET /api/v1/environments` — List environments
- `POST /api/v1/environments` — Create environment
- `GET /api/v1/deployments/:id/logs` — Get deployment logs

**Data Ownership:** `deployments`, `environments`, `deployment_logs`

**Communication:**
- Publishes: `deploy.started`, `deploy.completed`, `deploy.failed`
- Subscribes: `review.approved` (triggers deployment), `test.failed` (triggers rollback)

**Deployment:** 2 replicas + deployment runner workers

**Scaling:** Horizontal, based on deployment frequency

---

### 8. Notification Service

**Responsibility:** Sends notifications via email, Slack, webhooks, and in-app channels.

**API Surface:**
- `POST /api/v1/notifications` — Send notification
- `GET /api/v1/notifications` — List user notifications
- `PATCH /api/v1/notifications/:id/read` — Mark as read
- `PUT /api/v1/notifications/preferences` — Update preferences

**Data Ownership:** `notifications`, `notification_preferences`, `notification_log`

**Communication:**
- Subscribes: All completion/failure events across services
- Publishes: `notification.sent`, `notification.failed`

**Deployment:** 1 replica + background worker

**Scaling:** Horizontal, based on notification volume

---

### 9. User Service

**Responsibility:** Manages user authentication, authorization, profiles, and team management.

**API Surface:**
- `POST /api/v1/users/register` — Register user
- `POST /api/v1/users/login` — Login
- `GET /api/v1/users/me` — Get current user profile
- `PUT /api/v1/users/me` — Update profile
- `POST /api/v1/teams` — Create team
- `POST /api/v1/teams/:id/members` — Add team member

**Data Ownership:** `users`, `teams`, `team_members`, `roles`, `permissions`

**Communication:**
- Publishes: `user.created`, `user.updated`, `team.member_added`
- Subscribes: None (foundation service)

**Deployment:** 2 replicas

**Scaling:** Vertical (authentication-bound)

---

### 10. Analytics Service

**Responsibility:** Collects metrics, generates reports, and provides dashboards.

**API Surface:**
- `GET /api/v1/analytics/projects/:id` — Project analytics
- `GET /api/v1/analytics/agents` — Agent performance
- `GET /api/v1/analytics/velocity` — Team velocity
- `GET /api/v1/analytics/quality` — Quality metrics
- `POST /api/v1/analytics/reports` — Generate report

**Data Ownership:** `analytics_events`, `reports`, `dashboards`

**Communication:**
- Subscribes: All events (for metric collection)
- Publishes: `report.generated`

**Deployment:** 1 replica + background processor

**Scaling:** Horizontal, based on query volume

## Inter-Service Communication Patterns

### Synchronous (REST)
- Client → Gateway → Service (for user-facing operations)
- Service → Service (for immediate responses needed)

### Asynchronous (Event Bus)
- Service → Event Bus → Service (for fire-and-forget operations)
- Used for: status updates, triggering downstream processes, notifications

### Event Flow Matrix
```
PM Agent decomposes → task.created → Developer Agent picks up
Developer Agent codes → code.committed → Review Agent reviews
Review Agent approves → review.approved → Deploy Service deploys
Deploy Service deploys → deploy.completed → QA Service tests
QA Service tests → test.passed → User notified
```

## Service Dependencies

```
User Service (foundation)
    │
    ├──▶ Project Service
    │       │
    │       ├──▶ Agent Orchestrator
    │       │       │
    │       │       ├──▶ Code Service
    │       │       │       │
    │       │       │       ├──▶ Review Service
    │       │       │       │       │
    │       │       │       │       └──▶ Deploy Service
    │       │       │       │               │
    │       │       │       │               └──▶ QA Service
    │       │       │       │
    │       │       │       └──▶ Notification Service
    │       │       │
    │       │       └──▶ Notification Service
    │       │
    │       └──▶ Analytics Service
    │
    └──▶ Analytics Service
```
