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
| API Gateway | 8000 | HTTP/gRPC | Request routing, auth, rate limiting, load balancing |
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
GET    /api/v1/health                    # Health check
POST   /api/v1/auth/login                # User login
POST   /api/v1/auth/refresh              # Token refresh
POST   /api/v1/projects                  # Create project
GET    /api/v1/projects                  # List projects
GET    /api/v1/projects/{id}             # Get project details
GET    /api/v1/projects/{id}/status      # Project status
GET    /api/v1/projects/{id}/events      # Project event stream (SSE)
POST   /api/v1/projects/{id}/features    # Add feature request
GET    /api/v1/projects/{id}/artifacts   # List project artifacts
GET    /api/v1/projects/{id}/artifacts/{artifact_id}  # Download artifact
```

#### WebSocket
```
WS /api/v1/ws/projects/{id}              # Real-time project updates
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
