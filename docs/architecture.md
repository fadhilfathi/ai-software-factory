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
│  │   Task   │ │  Deploy  │ │Notifica- │ │  User    │          │
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
- **Framework:** Next.js 16 (React 19)
- **Language:** TypeScript
- **Styling:** Tailwind CSS 4
- **State Management:** React Query + Zustand
- **Drag-and-Drop:** @dnd-kit (Kanban board)
- **Real-time:** Server-Sent Events (SSE) for agent status

### Backend
- **Runtime:** Go 1.25+
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

---

## Sprint 3 Service Layer

The Sprint 3 implementation added two core services with full CRUD + state-machine support.

### Project Service

**File:** `src/internal/service/project.go`

The `ProjectService` wraps the `store.Store` interface and provides business logic:

| Method | Description | Validation |
|--------|-------------|------------|
| `CreateProject` | Create a new project with `initializing` status | `name` required |
| `GetProject` | Fetch a project by UUID | Returns `NOT_FOUND` if missing |
| `ListProjects` | Paginated list with optional status filter | Page defaults to 1, limit to 20 |
| `UpdateProject` | Partial update (name, description, status) | Only provided fields applied |
| `DeleteProject` | Remove a project by UUID | Returns `NOT_FOUND` if missing |

### Task Service

**File:** `src/internal/service/task.go`

The `TaskService` manages tasks within a project, including Kanban status transitions:

| Method | Description | Validation |
|--------|-------------|------------|
| `CreateTask` | Create task with `backlog` status and `medium` default priority | `title` required, `project_id` must exist |
| `GetTask` | Fetch a task by UUID | Returns `NOT_FOUND` if missing |
| `ListProjectTasks` | Paginated list with optional status filter | Page defaults to 1, limit to 20 |
| `UpdateTask` | Partial update (title, description, priority, assignee) | Only provided fields applied |
| `DeleteTask` | Remove a task by UUID | Returns `NOT_FOUND` if missing |
| `UpdateTaskStatus` | Kanban state-machine transition | Validated against transition map |

### Status Transition State Machine

**File:** `src/internal/service/service.go`

```go
taskStatusTransitions = map[model.TaskStatus][]model.TaskStatus{
    model.TaskBacklog:    {model.TaskReady, model.TaskBlocked},
    model.TaskReady:      {model.TaskInProgress, model.TaskBlocked},
    model.TaskInProgress: {model.TaskReview, model.TaskBlocked},
    model.TaskReview:     {model.TaskDone, model.TaskBlocked},
    model.TaskDone:       {model.TaskBlocked},
    model.TaskBlocked:    {model.TaskBacklog, model.TaskReady,
                           model.TaskInProgress, model.TaskReview, model.TaskDone},
}
```

Invalid transitions return HTTP `422 Unprocessable Entity` with code `INVALID_TRANSITION`.

### Services Composition

**File:** `src/internal/service/service.go`

The `Services` struct composes all service instances:

| Service | Store Interface | Description |
|---------|----------------|-------------|
| `AuthService` | UserStore | JWT authentication |
| `UserService` | UserStore | User profile management |
| `ProjectService` | ProjectStore | Project CRUD |
| `TaskService` | TaskStore | Task CRUD + Kanban status |
| `AgentService` | AgentStore | Agent lifecycle |
| `CodeService` | CodeStore | Code generation |
| `ReviewService` | ReviewStore | Code review |
| `DeploymentService` | DeploymentStore | Deployment management |
| `WebhookService` | WebhookStore | Webhook registration |

---

## Sprint 3 Store Layer

### In-Memory Store (Fallback)

**File:** `src/internal/store/memory.go`

Implements all `Store` interfaces using `sync.RWMutex`-protected maps. Used when `DB_HOST` is not set. All data is ephemeral — resets on server restart.

```
NewMemoryStore() → Store
  ├─ Users()      → memoryUserStore
  ├─ Projects()   → memoryProjectStore
  ├─ Agents()     → memoryAgentStore
  ├─ Tasks()      → memoryTaskStore
  ├─ Code()       → memoryCodeStore
  ├─ Reviews()    → memoryReviewStore
  ├─ Deployments()→ memoryDeploymentStore
  └─ Webhooks()   → memoryWebhookStore
```

### PostgreSQL Store

**File:** `src/internal/store/postgres/store.go`

Wraps a `pgx/v5` connection pool. Uses the in-memory store as a fallback for stores not yet migrated (User, Agent, Code, Review, Deployment, Webhook). Projects and Tasks are backed by PostgreSQL.

```
NewStore(pool) → Store
  ├─ Projects() → postgresProjectStore  (PostgreSQL)
  ├─ Tasks()    → postgresTaskStore     (PostgreSQL)
  └─ Others()   → memoryStore           (fallback)
```

**Auto-selection logic** in `src/cmd/main.go`:
```go
if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
    pool, _ := db.Connect(ctx, config)
    db.RunMigrations(ctx, pool, "db/migrations")
    st = postgres.NewStore(pool)  // PostgreSQL + fallback
} else {
    st = store.NewMemoryStore()   // In-memory only
}
```

### Store Interface

**File:** `src/internal/store/store.go`

```
Store
├─ Users()      → UserStore      (Create, GetByID, GetByEmail, List, Update, CheckProjectAccess)
├─ Projects()   → ProjectStore   (Create, GetByID, List, Update, Delete)
├─ Agents()     → AgentStore     (Create, GetByID, List, Update, Delete)
├─ Tasks()      → TaskStore      (Create, GetByID, List, Update, Delete)
├─ Code()       → CodeStore      (CodeGen + File + Commit operations)
├─ Reviews()    → ReviewStore    (Create, GetByID, ListByProject, Update)
├─ Deployments()→ DeploymentStore(Create, GetByID, ListByProject, Update)
└─ Webhooks()   → WebhookStore   (Create, GetByID, List, Update, Delete)
```

---

## Sprint 3 Frontend Pages

### Next.js App Router Structure

```
frontend/src/app/
├── projects/
│   ├── page.tsx              # Project list with status filter + pagination
│   ├── new/page.tsx          # Create project form
│   └── [id]/
│       ├── page.tsx          # Project detail with task summary + task list
│       ├── edit/page.tsx     # Edit project form
│       └── board/page.tsx    # Kanban board with drag-and-drop
├── dashboard/page.tsx        # Dashboard metrics
├── agents/page.tsx           # Agent list
├── tasks/page.tsx            # Task overview
└── settings/page.tsx         # Settings
```

### React Query Integration

**File:** `frontend/src/lib/hooks.ts`

All API operations use `@tanstack/react-query` v5 for caching, background refetching, and optimistic updates.

| Hook | Endpoint | Cache Strategy |
|------|----------|---------------|
| `useProjects` | `GET /v1/projects` | Stale-while-revalidate |
| `useProject` | `GET /v1/projects/:id` | Cache by ID |
| `useCreateProject` | `POST /v1/projects` | Invalidates list |
| `useUpdateProject` | `PUT /v1/projects/:id` | Invalidates list + detail |
| `useDeleteProject` | `DELETE /v1/projects/:id` | Invalidates list |
| `useTasks` | `GET /v1/projects/:projectId/tasks` | Cache by project |
| `useCreateTask` | `POST /v1/projects/:projectId/tasks` | Invalidates task list |
| `useUpdateTaskStatus` | `PATCH /v1/tasks/:id/status` | **Optimistic update** with rollback |
| `useDeleteTask` | `DELETE /v1/tasks/:id` | Invalidates task list |

### Kanban Board Components

**File:** `frontend/src/components/kanban/`

```
kanban/
├── KanbanBoard.tsx    # DndContext + Column layout + DragOverlay
├── KanbanColumn.tsx   # SortableContext per column + column header
├── TaskCard.tsx       # Draggable task card with priority badge
└── AddTaskDialog.tsx  # Inline task creation dialog
```

Built with `@dnd-kit/core` and `@dnd-kit/sortable`. Uses `PointerSensor` with 5px activation distance to prevent accidental drags. Optimistic UI updates the task status immediately on drop, with rollback on API error.

---

## Deployment Architecture

### Development
```
Local Machine
├── Docker Compose
│   ├── API Server (port 8080)
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

---

## Key Architectural Decisions (Sprint 3 Additions)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Store Architecture | Dual PostgreSQL + in-memory | Allows development without Docker; production uses PostgreSQL. Auto-selected at startup via `DB_HOST` env var. |
| Kanban State Machine | Go service layer with explicit map | Testable, no hidden state, clear transition rules in a single file. |
| Drag-and-Drop | @dnd-kit | Lightweight (3KB gzip), accessible, React-first design. |
| API Integration | React Query v5 | Automatic caching, background refetch, optimistic updates with rollback. |
| Status Codes | 201/204 for creates/deletes | REST best practices. `PATCH /status` returns `422` for invalid transitions. |

---

## Data Flow

### Project Creation Flow
```
User → Projects Page (form) → useCreateProject → POST /v1/projects
                                                    ↓
                                              ProjectHandler.Create
                                                    ↓
                                              ProjectService.CreateProject
                                                    ↓
                                              ProjectStore.Create (PostgreSQL / memory)
                                                    ↓
                                              Response (201) → React Query cache invalidation
```

### Kanban Drag-and-Drop Flow
```
User drags task card → DndContext.onDragEnd
                        ↓
                  handleStatusChange(taskId, newStatus)
                        ↓
                  useUpdateTaskStatus.mutate({ id, status })
                        ↓
                  Optimistic UI update (immediate)
                   ┌────┴────┐
                   │         │
              PATCH /v1/tasks/:id/status
                   │         │
               Success    Failure
                   │         │
              Cache sync   Rollback UI
```

---

## Security Architecture

For a comprehensive formalization of the platform's security controls, identity management, and compliance standards, see the [Security Architecture](./security.md) document.

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

---

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

---

## Trade-offs

1. **Monolith vs Microservices:** Chose microservices for independent scaling and deployment, accepting operational complexity
2. **SQL vs NoSQL:** Chose PostgreSQL for data integrity, accepting slightly lower write throughput
3. **Self-hosted vs Managed:** Chose managed services for production, self-hosted for development
4. **Synchronous vs Async:** Chose async agent execution for resilience, accepting eventual consistency
5. **Single Agent vs Multi-Agent:** Chose multi-agent for specialization, accepting coordination overhead
6. **In-Memory vs Persistent:** Chose dual store for developer experience, accepting the need to handle two storage backends
