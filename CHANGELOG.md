# Changelog

All notable changes to the **AI Software Factory** platform are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- *Placeholder for upcoming features.*

---

## [1.1.0] — 2026-06-12

### Added
- **Kanban Module** — Fully integrated Kanban board with drag-and-drop support using `@dnd-kit`.
- **Authentication RBAC** — New `RequireRole` middleware to support role-based access control.
- **Redis Revocation Store** — Persistent store for refresh token revocation (WIP).
- **Agent Badge Component** — Visual indicator for different agent types in the frontend.
- **Task Detail Modal** — Detailed view for tasks accessible from the Kanban board.
- **Agent Registry Backend** — Heartbeat monitoring, task lifecycle management, and performance tracking.
- **Audit Log Infrastructure** — Persistent PostgreSQL store for all security-sensitive actions.

### Changed
- **Backend Framework** — Migrated from standard `net/http` to **Gin** for improved performance and middleware support.
- **Go Version** — Upgraded runtime to **Go 1.25+**.
- **Auth Architecture** — Refactored `AuthService` into an interface for better testability and enhanced `ValidateToken` to verify user status on every request.
- **Frontend Stack** — Upgraded to **Next.js 16** (React 19) and **Tailwind CSS 4**.
- **Documentation** — Comprehensive refinement of `api-spec.md`, `architecture.md`, and `developer-guide.md`. Added `docs/security.md`.

### Fixed
- **Kanban Sync** — Implemented optimistic UI updates with automatic rollback on API failure for smoother drag-and-drop experience.
- **Port Consistency** — Standardized API port to `8080` across all documentation and Docker configurations.

---

## [1.0.0] — 2026-06-10

### Added

#### Core Platform
- **Project Management** — Create projects from natural-language descriptions; automated breakdown into user stories and tasks; full lifecycle from Intake → Analysis → Planning → Implementation → Review → Testing → Deployment → Done.
- **Agent Orchestration Engine** — Spawns and coordinates specialized AI agents (PM, Architect, Developer, Reviewer, QA, DevOps) with dependency-aware parallel execution.
- **Quality Gate System** — Automated and human approval checkpoints at every project stage; gate history is immutable and auditable.
- **Real-time Dashboard** — Per-project progress view with stage indicators, active agent status, velocity metrics, and auto-refresh every 10 seconds.

#### API (v1)
- **Authentication** — JWT bearer tokens for user sessions and API keys (`ak_` prefix) for service-to-service automation. Public routes: health check, login, registration.
- **Projects** — CRUD operations with status filters, pagination, and search. Project metadata includes name, description, complexity, and current stage.
- **Agents** — Spawn, monitor, and control AI agents per project. Each agent reports status (idle/running/complete/failed) and supports manual intervention (pause/resume/cancel).
- **Tasks** — Create, assign, and track work items. Tasks support dependencies, priority, and status transitions. Agent assignment routes tasks to the correct specialist.
- **Code Generation** — Generate code from task descriptions; results include file diffs, commit messages, and branch references.
- **Code Reviews** — Submit code for review, receive annotated feedback, approve or reject changes, and trigger rework cycles.
- **Deployments** — Start deployments from approved builds; track deployment status, health checks, and rollback triggers.
- **Users** — Registration, profile management, role-based access (admin/member/viewer), and team management.
- **Webhooks** — Register event-driven callbacks for project lifecycle, agent status changes, and deployment events.
- **Health Check** — `GET /v1/healthz` endpoint with dependency status (database connectivity, Redis reachability).

#### Frontend
- **Dashboard** — Project list with status indicators, search, and sort.
- **Project Detail Page** — Stage timeline, agent activity feed, and task board.
- **Task Board** — Kanban-style task view with drag-and-drop status transitions.
- **Agent Performance View** — Metrics dashboard per agent (completion rate, cycle time, quality score).
- **New Project Wizard** — Step-by-step creation flow with natural-language intake.
- **Settings** — User profile, API key management, notification preferences.
- **Realtime Provider** — SSE-based live updates for agent status and project events.
- **Responsive Design** — Mobile dashboard (bottom navigation, collapsible panels).
- **Dark/Light Theme Support** — System-preference-aware theming with manual override.

#### Backend
- **Golang API Server** — RESTful backend using Gin framework, structured logging (`zap`), request-scoped context, and panic recovery middleware.
- **Router** — Path-based routing with middleware chaining (auth, rate limiting, request logging, CORS).
- **Authentication Services** — JWT issuance/validation, API key verification, password hashing (bcrypt), refresh token rotation.
- **User Service** — Registration flow, profile CRUD, role assignment, rate-limit tracking.
- **Webhook Service** — Event subscription management, delivery retry with exponential backoff, delivery audit log.
- **Middleware Stack** — CORS (configurable origins), request ID injection, structured access logging, rate limiting (per-user and per-IP), recovery from panics.
- **Database Migrations** — PostgreSQL schema with migrations for projects, users, agents, tasks, code reviews, deployments, webhooks, and audit log.

#### Frontend Architecture
- **State Management** — React Query (server cache) + Zustand (client state) with typed stores.
- **Realtime Integration** — SSE-based `RealtimeProvider` with automatic reconnection and connection-state indicators.
- **TypeScript Types** — Shared API types, generated from backend schemas.
- **Custom Hooks** — `useDebouncedSearch`, consistent query key factory pattern.
- **Component Library** — Built on Tailwind CSS with design-system tokens (colors, spacing, typography).

#### DevOps & Infrastructure
- **Docker Compose Orchestration** — Single-command startup for API, frontend, and PostgreSQL; health-check dependency ordering; layered `.env` configuration; three operation modes (full stack, native dev with Docker DB, per-service).
- **Dockerfiles** — Multi-stage Go build (distroless runtime, layer caching), Node.js frontend production build with npm cache optimisation.
- **CI/CD Pipeline** — GitHub Actions `ci.yml` (lint → test → build → e2e smoke) on every push and PR; `deploy.yml` (build & push to GHCR → SSH deploy → health validation) on push to `main`.
- **Deployment Scripts** — `build.sh` (Go binary + frontend assets + Docker images), `deploy.sh` (start/stop/restart/update stack with --pull support), `healthcheck.sh` (unified CLI with JSON output and watch mode).
- **Helm Chart (Preview)** — Kubernetes deployment manifests for production-scale deployments (included in `ops/helm/`).

#### Documentation
- **README** — Project overview, architecture diagram, quick start (4 commands), 3 installation options, project structure map, script reference.
- **User Guide** — API feature reference (20 endpoints across 9 categories), authentication docs, frontend features table, configuration reference, 7 step-by-step tutorials, troubleshooting section.
- **Developer Guide** — Complete API reference with request/response schemas for all 9 resource categories, HTTP status codes, error codes, architecture overview with service-layer diagram, full contributing guide (coding standards for Go + TypeScript, testing workflows, PR process), operations section (build, logging, debugging), database schema documentation.
- **Deployment Guide** — Three deployment paths (Docker Compose / CI/CD / manual SSH), rollback procedures, Docker config reference, CI/CD workflow breakdown.
- **Environment Setup Guide** — Prerequisites, repository setup, env vars, first-time setup, Windows notes.
- **Monitoring Guide** — 4-layer health architecture, health check script reference, logging (structured JSON, log levels), Docker monitoring commands, metrics roadmap (Prometheus/Grafana), alerting recommendations.
- **Architecture Document** — System architecture overview, technology stack decisions, component interaction diagrams, data flow documentation.
- **Security Threat Model** — Asset inventory, trust boundaries, threat analysis (STRIDE), security controls mapping, incident response plan.
- **Functional & Non-functional Requirements** — Full requirements catalog (FR-001 through FR-XXX) with MoSCoW priorities, acceptance criteria, dependency mapping.
- **Design System** — Component library documentation, design tokens, UI/UX patterns, accessibility guidelines.
- **Wireframes** — 8 interactive HTML wireframes covering the full user journey (dashboard, projects, tasks, agent performance, settings, mobile).

### Changed

- *N/A — Initial release.*

### Fixed

- *N/A — Initial release.*

### Removed

- *N/A — Initial release.*

### Deprecated

- *N/A — Initial release.*

### Security

- Authentication uses bcrypt for password hashing (work factor 12).
- JWT tokens use HMAC-SHA256 signing with configurable expiration.
- API keys are stored as SHA-256 hashes; full key shown only at creation.
- CORS middleware restricts origins to configured allowlist.
- Rate limiting per user and per IP prevents abuse.
- All database queries use parameterised statements (no SQL injection surface).
- Structured logging avoids leaking secrets (PII redaction in access logs).

---

[Unreleased]: https://github.com/fadhilfathi/AI-Software-Factory/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/fadhilfathi/AI-Software-Factory/releases/tag/v1.0.0
