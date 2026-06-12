# Release Notes

## Template Guide

This file documents production releases of the **AI Software Factory** platform.
Each release section follows a standard structure:

```
## [v<major>.<minor>.<patch>] — <YYYY-MM-DD>

### Highlights
Top 2–3 things a reader should know.

### What's New
New features and capabilities.

### Improvements
Enhancements to existing features (performance, UX, reliability).

### Bug Fixes
Issues resolved in this release.

### Breaking Changes
Changes that require action when upgrading. Include migration path or
reference the [Migration Guide](./MIGRATION_GUIDE.md).

### Deprecations
Features that are still supported but will be removed in a future release.

### Known Issues
Open issues that affect this release.

### Components / Versions
Component-level version table for operators.

### Upgrade Notes
Link to specific migration section or steps.
```

---

## v1.0.0 — 2026-06-10

### Highlights

- **First public release** of the AI Software Factory platform — turn a software
  idea into a deployed project by orchestrating a team of specialised AI agents.
- **Production-ready stack** — Go API, Next.js frontend, PostgreSQL, Docker
  Compose orchestration, and GitHub Actions CI/CD.
- **End-to-end documentation** — 260+ line README, 1240-line user guide,
  1583-line developer guide, deployment guide, monitoring guide, architecture
  docs, threat model, wireframes, and design system.

### What's New

#### Platform
- **Project Lifecycle Management** — Full project lifecycle from natural-language
  intake through analysis, planning, implementation, review, testing, and
  deployment. Each stage has entry/exit criteria with automated or human gate
  approval.
- **Agent Orchestration** — Orchestration engine that spawns, coordinates, and
  monitors six specialised agent types: PM, Architect, Developer, QA, Reviewer,
  and DevOps. Agents operate in parallel where dependencies permit.
- **Real-time Dashboard** — Per-project dashboard showing current stage, %
  complete, active agents, recent events, velocity metrics, and cycle time.
- **Quality Gates** — Automated checks at each stage transition with full audit
  trail. Gate history is immutable and rollback is supported with reason
  capture.

#### API (v1)
- 9 resource categories covering the full platform surface: Authentication,
  Projects, Agents, Tasks, Code Generation, Code Reviews, Deployments, Users,
  and Webhooks.
- JWT bearer tokens for interactive sessions and API key (`ak_` prefix)
  authentication for service-to-service automation.
- Paginated, filterable, and searchable list endpoints.
- Consistent error-response format across all resources with HTTP status codes
  and machine-readable error codes.

#### Frontend
- Next.js 14 (React 18) application with TypeScript and Tailwind CSS.
- Interactive dashboard, Kanban task board, agent performance metrics,
  deployment tracking, and settings pages.
- SSE-based real-time provider for live agent status and project event updates.
- Responsive design with mobile-optimised navigation and dark/light theme
  support.

#### Backend
- Go 1.22+ RESTful API with structured logging (`log/slog`), panic recovery,
  CORS, request ID injection, and per-user/per-IP rate limiting.
- PostgreSQL-backed data layer with migration management.
- Dependency-tracking for task scheduling and agent coordination.

#### DevOps
- **Docker Compose** — Single-command `docker compose up -d --build` to run the
  full stack. Health-check dependency ordering ensures correct startup sequence.
- **Dockerfiles** — Multi-stage Go build (distroless runtime) and Node.js frontend
  production image with layer caching optimisation.
- **CI/CD** — GitHub Actions `ci.yml` (lint → test → build → e2e smoke) on
  every push/PR; `deploy.yml` (build & push to GHCR → SSH deploy → health
  validation) on push to `main`.
- **Scripts** — `build.sh`, `deploy.sh`, and `healthcheck.sh` for common
  operations.

#### Documentation
- Complete README with quick start, architecture diagram, installation options,
  and project structure.
- User Guide with API reference, frontend features, 7 tutorials, and
  troubleshooting.
- Developer Guide with API contracts, architecture deep-dive, contributing
  standards, and operations reference.
- Deployment Guide covering 3 deployment paths with rollback procedures.
- Environment Setup, Monitoring Guide, Architecture Document, Threat Model,
  Design System, Wireframes, and full Requirements Catalogue.

### Improvements

- *N/A — Initial release.*

### Bug Fixes

- *N/A — Initial release.*

### Breaking Changes

- **None.** This is the initial release. No upgrade path from a previous version
  exists. See the [Migration Guide](./MIGRATION_GUIDE.md) for notes on adopting
  this release from a pre-release or prototype setup.

### Deprecations

- **None.** All APIs and features in v1.0.0 are stable and supported.

### Known Issues

- **Single-host deployment only.** The current Docker Compose-based deployment
  runs on a single VM. Multi-host scaling and Kubernetes support are planned
  for a future release.
- **No built-in authentication UI.** User registration and login are available
  via the API but the frontend login flow is minimal. A full auth UI is planned.
- **Agent quality is model-dependent.** The platform orchestrates agents, but
  code quality, review accuracy, and planning coherence depend on the underlying
  LLM provider and model configuration.
- **No persistent agent memory.** Agents operate per-project with no cross-
  project learning or memory. Session context is limited to the current project.

### Components

| Component | Technology | Version | Notes |
|-----------|-----------|---------|-------|
| API Server | Go | 1.22+ | RESTful backend on `:8080` |
| Frontend | Next.js 14 / React 18 | 20+ (LTS) | TypeScript + Tailwind CSS |
| Database | PostgreSQL | 16 | Primary data store |
| Cache | Redis | 7+ | Session cache (future use) |
| Container Runtime | Docker | 24+ | Build and orchestration |
| Orchestration | Docker Compose | 2.24+ | Multi-service orchestration |
| CI/CD | GitHub Actions | — | CI + deploy workflows |
| Image Registry | GitHub Container Registry | — | Production image storage |

### Upgrade Notes

See the [Migration Guide](./MIGRATION_GUIDE.md) for full details. For this
release (v1.0.0), the recommended path for new installations is:

```bash
git clone https://github.com/fadhilfathi/AI-Software-Factory.git
cd AI-Software-Factory
cp .env.example .env
docker compose up -d --build
```

### Resources

- [Repository](https://github.com/fadhilfathi/AI-Software-Factory)
- [README](./README.md)
- [User Guide](./docs/user-guide.md)
- [Developer Guide](./docs/developer-guide.md)
- [Deployment Guide](./docs/deployment-guide.md)
- [Changelog](./CHANGELOG.md)
- [Migration Guide](./MIGRATION_GUIDE.md)
