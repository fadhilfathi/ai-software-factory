# AI Software Factory — System Architecture Design

## Architecture Overview

The AI Software Factory is a multi-agent platform that orchestrates specialized AI agents to deliver software projects. The architecture follows a microservices pattern with an event-driven communication backbone.

```
┌─────────────────────────────────────────────────────────────────┐
│                        CLIENT LAYER                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│  │ Web App  │  │ Mobile   │  │ CLI      │  │ API      │       │
│  │ (React)  │  │ (Future) │  │ (Future) │  │ Clients  │       │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘       │
│       └──────────────┴──────────────┴──────────────┘            │
└───────────────────────────┬─────────────────────────────────────┘
                            │ HTTPS
┌───────────────────────────┴─────────────────────────────────────┐
│                        API GATEWAY                               │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  Authentication │ Rate Limiting │ Routing │ Load Bal.    │   │
│  └──────────────────────────────────────────────────────────┘   │
└───────────────────────────┬─────────────────────────────────────┘
                            │ Internal Network
┌───────────────────────────┴─────────────────────────────────────┐
│                      SERVICE LAYER                               │
│                                                                  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐          │
│  │ Project  │ │  Agent   │ │  Code    │ │  Review  │          │
│  │ Service  │ │ Orch.    │ │ Service  │ │ Service  │          │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘          │
│       │            │            │            │                   │
│  ┌────┴─────┐ ┌────┴─────┐ ┌────┴─────┐ ┌────┴─────┐          │
│  │    QA    │ │  Deploy  │ │Notifica- │ │  User    │          │
│  │ Service  │ │ Service  │ │tion Svc  │ │ Service  │          │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘          │
│                                                                  │
│  ┌──────────┐ ┌──────────┐                                      │
│  │Analytics │ │ Webhook  │                                      │
│  │ Service  │ │ Service  │                                      │
│  └──────────┘ └──────────┘                                      │
└───────────────────────────┬─────────────────────────────────────┘
                            │
┌───────────────────────────┴─────────────────────────────────────┐
│                     DATA LAYER                                   │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐          │
│  │PostgreSQL│ │  Redis   │ │  S3/Blob │ │ Git Repos│          │
│  │ (Primary)│ │ (Cache)  │ │(Artifacts│ │ (Code)   │          │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘          │
└─────────────────────────────────────────────────────────────────┘
```

## Technology Stack

### Frontend
- **Framework:** Next.js 14+ (React 18)
- **Language:** TypeScript
- **Styling:** Tailwind CSS
- **State Management:** React Query + Zustand
- **Real-time:** Server-Sent Events (SSE) for agent status

### Backend
- **Runtime:** Go 1.22+
- **Framework:** Gin (high performance)
- **Language:** Go
- **API Style:** REST + WebSocket for real-time

### AI/ML Layer
- **LLM Provider:** OpenAI GPT-4 / Anthropic Claude (configurable)
- **Agent Framework:** Custom agent orchestration engine
- **Prompt Management:** Versioned prompt templates
- **Model Routing:** Task-type based model selection

### Data
- **Primary Database:** PostgreSQL 16
- **Cache:** Redis 7
- **Object Storage:** AWS S3 / MinIO (self-hosted)
- **Search:** Elasticsearch (optional, for audit logs)

### Infrastructure
- **Container Runtime:** Docker
- **Orchestration:** Docker Compose (dev) / Kubernetes (prod)
- **CI/CD:** GitHub Actions
- **Monitoring:** Prometheus + Grafana
- **Logging:** ELK Stack or Loki

## Deployment Architecture

### Development
```
Local Machine
├── Docker Compose
│   ├── API Server (port 3001)
│   ├── PostgreSQL (port 5432)
│   ├── Redis (port 6379)
│   └── MinIO (port 9000)
└── Next.js Dev Server (port 3000)
```

### Production
```
Cloud Provider (AWS/GCP/Azure)
├── Load Balancer (ALB/NLB)
├── Kubernetes Cluster
│   ├── API Pods (3+ replicas)
│   ├── Agent Worker Pods (auto-scaling)
│   └── Background Jobs Pod
├── Managed PostgreSQL (RDS/Cloud SQL)
├── Managed Redis (ElastiCache/ Memorystore)
├── Object Storage (S3/GCS)
└── Monitoring Stack
    ├── Prometheus
    ├── Grafana
    └── AlertManager
```

## Data Flow

### Project Creation Flow
```
User → API Gateway → Project Service → PostgreSQL
                                    ↓
                              PM Agent (spawned)
                                    ↓
                              Generates: User Stories, Tasks
                                    ↓
                              Agent Orchestrator → Status Update → User (SSE)
```

### Code Generation Flow
```
Agent Orchestrator → Developer Agent → Code Service → Git Repo
                                    ↓
                              Review Agent (triggered)
                                    ↓
                              Quality Gate Check
                                    ↓
                              Pass? → Merge + Deploy
                              Fail? → Developer Agent (retry with feedback)
```

### Deployment Flow
```
Code Service (merge event) → Deploy Service → CI/CD Pipeline
                                          ↓
                                    Build → Test → Deploy
                                          ↓
                                    Health Check
                                          ↓
                                    Success? → Notify User
                                    Failure? → Rollback + Notify
```

## Agent Orchestration Pattern

### Agent Lifecycle
```
┌─────────┐     ┌─────────┐     ┌─────────┐     ┌─────────┐
│  Spawn  │────▶│ Assign  │────▶│ Execute │────▶│  Review │
│         │     │  Task   │     │  Task   │     │ Output  │
└─────────┘     └─────────┘     └─────────┘     └─────────┘
                                      │              │
                                      │    ┌─────────┴─────────┐
                                      │    │                   │
                                      ▼    ▼                   ▼
                                   ┌─────────┐          ┌─────────┐
                                   │  Retry  │          │ Complete│
                                   │ (if     │          │         │
                                   │ failed) │          │         │
                                   └─────────┘          └─────────┘
```

### Task Decomposition
```
Project Request
       │
       ▼
┌─────────────┐
│  PM Agent   │
│ (Decompose) │
└──────┬──────┘
       │
       ├──▶ User Story 1 ──▶ Task 1.1 ──▶ Developer Agent
       │                    Task 1.2 ──▶ Developer Agent
       │
       ├──▶ User Story 2 ──▶ Task 2.1 ──▶ Developer Agent
       │                    Task 2.2 ──▶ Architect Agent
       │
       └──▶ User Story 3 ──▶ Task 3.1 ──▶ Developer Agent
```

## Message Passing / Event System

### Event Types
| Event | Producer | Consumer | Description |
|-------|----------|----------|-------------|
| `project.created` | Project Service | PM Agent | New project needs decomposition |
| `task.assigned` | Agent Orch. | Developer Agent | Task ready for implementation |
| `code.committed` | Code Service | Review Agent | New code needs review |
| `review.approved` | Review Agent | Deploy Service | Code approved for deployment |
| `deploy.completed` | Deploy Service | QA Agent | Deployment ready for testing |
| `test.passed` | QA Agent | User | Tests pass, feature ready |
| `agent.failed` | Agent Worker | Agent Orch. | Agent needs retry or escalation |

### Event Bus
- **Technology:** Redis Streams (lightweight) or Apache Kafka (scale)
- **Pattern:** Publish-Subscribe with consumer groups
- **Retention:** 7 days for replay capability
- **Ordering:** Per-project event ordering guaranteed

## Storage Strategy

### Code Repositories
- **Location:** GitHub/GitLab (hosted) or Gitea (self-hosted)
- **Pattern:** One repository per project
- **Branching:** main → develop → feature branches
- **Protection:** main branch requires review approval

### Artifacts
- **Location:** S3-compatible object storage
- **Types:** Build artifacts, deployment packages, reports
- **Lifecycle:** 30-day retention for builds, permanent for releases
- **Access:** Pre-signed URLs for temporary access

### Logs & Audit
- **Location:** Elasticsearch or Loki
- **Retention:** 90 days hot, 1 year cold
- **Indexing:** By project, agent, timestamp
- **Search:** Full-text search across all logs

## Security Architecture

### Authentication Flow
```
User → Login (OAuth/Email) → Auth Service → JWT Token
                                                    │
                                              ┌─────┴─────┐
                                              │ Access +  │
                                              │ Refresh   │
                                              │ Tokens    │
                                              └───────────┘
```

### Network Security
- All external traffic via HTTPS (TLS 1.3)
- Internal service mesh (Istio/Linkerd optional)
- Network policies restrict inter-service communication
- Secrets managed via HashiCorp Vault or cloud KMS

### Agent Security
- Agents run in isolated containers
- Limited filesystem access (only project workspace)
- No network access except approved APIs
- Resource limits (CPU, memory, execution time)
- Output sanitization before user display

## Scalability Approach

### Horizontal Scaling
- **API Servers:** Stateless, scale behind load balancer
- **Agent Workers:** Independent scaling based on queue depth
- **Database:** Read replicas for query-heavy operations
- **Cache:** Redis Cluster for distributed caching

### Auto-Scaling Rules
- CPU > 70% → Scale up API servers
- Queue depth > 50 → Scale up agent workers
- Memory > 80% → Scale up database
- Connections > 80% → Scale up connection pool

## Key Architectural Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Database | PostgreSQL | ACID compliance, JSON support, proven reliability |
| Cache | Redis | Performance, pub/sub for events, session storage |
| Agent Runtime | Go | Consistent with API, goroutines/channels for high-concurrency agent loops |
| Communication | REST + SSE | REST for commands, SSE for real-time updates |
| Storage | S3-compatible | Scalable, cost-effective, widely supported |
| Container | Docker | Standard, portable, Kubernetes-native |

## Trade-offs

1. **Monolith vs Microservices:** Chose microservices for independent scaling and deployment, accepting operational complexity
2. **SQL vs NoSQL:** Chose PostgreSQL for data integrity, accepting slightly lower write throughput
3. **Self-hosted vs Managed:** Chose managed services for production, self-hosted for development
4. **Synchronous vs Async:** Chose async agent execution for resilience, accepting eventual consistency
5. **Single Agent vs Multi-Agent:** Chose multi-agent for specialization, accepting coordination overhead
