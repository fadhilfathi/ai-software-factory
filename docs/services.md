# AI Software Factory — Microservice Architecture

> **Document Version**: 1.0  
> **Last Updated**: 2026-06-10  
> **Status**: Approved  
> **Owner**: Architecture Team

---

## Overview

The AI Software Factory is built as a distributed system of specialized microservices, each owning a distinct domain of the software delivery lifecycle. Services communicate via well-defined APIs and asynchronous messaging, enabling independent scaling, deployment, and evolution.

### Design Principles

1. **Single Responsibility** — Each service owns one business capability end-to-end
2. **Data Ownership** — Services own their data; no shared databases
3. **Async-First Communication** — Event-driven for decoupling; sync only for queries
4. **Observability by Default** — Structured logging, metrics, tracing on every service
5. **Failure Isolation** — Circuit breakers, bulkheads, graceful degradation
6. **API Versioning** — Explicit versioning; backward compatibility guaranteed

---

## Service Catalog

| Service | Port | Protocol | Primary Responsibility |
|---------|------|----------|------------------------|
| API Gateway | 8080 | HTTP/gRPC | Request routing, auth, rate limiting, load balancing |
| Project Service | 8001 | gRPC | Project lifecycle, metadata, progress tracking |
| Agent Orchestrator | 8002 | gRPC + NATS | Agent spawning, coordination, task distribution |
| Code Service | 8003 | gRPC | Code generation, refactoring, analysis |
| Review Service | 8004 | gRPC | Code review, quality gates, standards enforcement |
| QA Service | 8005 | gRPC | Test generation, execution, bug tracking |
| Deploy Service | 8006 | gRPC | CI/CD pipelines, deployments, rollbacks |
| Notification Service | 8007 | gRPC + NATS | Multi-channel notifications, preferences |
| User Service | 8008 | gRPC | Authentication, authorization, profiles, teams |
| Analytics Service | 8009 | gRPC | Metrics, dashboards, reporting, insights |

---

## 1. API Gateway

### Responsibility
- Single entry point for all external clients (web, mobile, CLI)
- Authentication termination (JWT validation, API keys)
- Rate limiting and quota enforcement
- Request/response transformation
- Load balancing and service discovery
- SSL termination
- Request logging and audit trail

### API Surface

#### REST Endpoints
```
GET    /v1/healthz                       # Health check
POST   /v1/auth/login                # User login
POST   /v1/auth/refresh              # Token refresh
POST   /v1/projects                  # Create project
GET    /v1/projects                  # List projects
GET    /v1/projects/{id}             # Get project details
GET    /v1/projects/{id}/status      # Project status
GET    /v1/projects/{id}/events      # Project event stream (SSE)
POST   /v1/projects/{id}/features    # Add feature request
GET    /v1/projects/{id}/artifacts   # List project artifacts
GET    /v1/projects/{id}/artifacts/{artifact_id}  # Download artifact
```

#### WebSocket
```
WS /v1/ws/projects/{id}              # Real-time project updates
```

### Data Ownership
- None (stateless proxy)
- Caches: service registry, rate limit counters (Redis)

### Communication Patterns
- **Sync**: Routes requests to downstream services via gRPC
- **Async**: Publishes `gateway.request.received` to NATS for audit

### Scaling Strategy
- Horizontal: Stateless, scale behind load balancer
- Target: < 10ms p99 latency overhead
- Auto-scale on: request rate, CPU, connection count

---

## 2. Project Service

### Responsibility
- Project lifecycle management (Intake → Done)
- Metadata and progress tracking
- Task decomposition (PM Agent interface)
- Backlog and sprint management
- Project-level configuration and templates

### API Surface
```
POST   /v1/projects                  # Create project
GET    /v1/projects                  # List projects
GET    /v1/projects/{id}             # Get project details
PUT    /v1/projects/{id}             # Update project
DELETE /v1/projects/{id}             # Delete project
POST   /v1/projects/{id}/decompose   # Trigger task decomposition
```

---

## 3. Code Service

### Responsibility
- Code generation from natural language or formal specs (OpenAPI, etc.)
- Incremental code modifications (features, fixes, refactoring)
- Multi-language support (MVP: TS, Python, Go, Rust)
- Infrastructure as Code (IaC) generation
- Project file management and static analysis
- Integration with Git for commits and branch management

### API Surface
```
POST   /v1/code/generate             # Request code generation for a task
GET    /v1/code/{projectId}/files/*path  # Retrieve file content
POST   /v1/code/{projectId}/commits  # Create a new commit
GET    /v1/code/{projectId}/diff     # Get diff between commits/branches
GET    /v1/code/{projectId}/analysis # Static analysis and complexity report
```

### Data Ownership
- Code generation requests and statuses
- Project file metadata and cached content
- Analysis reports

---

## 4. Review Service

### Responsibility
- Automated code review (Correctness, Security, Performance)
- Standards and conventions enforcement
- Quality gate management for PRs and code batches
- Orchestration of human-in-the-loop review workflows
- Architecture Decision Record (ADR) review and management

### API Surface
```
POST   /v1/reviews                   # Request review for a commit
GET    /v1/reviews/{id}              # Get review findings and status
POST   /v1/reviews/{id}/comments     # Add inline comments
PATCH  /v1/reviews/{id}/status       # Approve or request changes
GET    /v1/reviews/project/{projectId} # List project reviews
```

### Data Ownership
- Review results, findings, and inline comments
- Quality gate configurations
- Review analytics and history

---

## 5. Agent Orchestrator

### Responsibility
- Agent lifecycle management (spawn, monitor, shutdown)
- Task distribution and parallel execution coordination
- Inter-agent communication bus management
- Agent health and performance monitoring
- Context isolation and propagation between agent runs
