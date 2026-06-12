# Agent Orchestration Guide

> Developer guide for the AI Software Factory agent orchestration engine — models, capabilities, assignment, execution, and deliverables.

---

## Table of Contents

- [Agent Model](#agent-model)
- [Capability System](#capability-system)
- [Task Assignment Flow](#task-assignment-flow)
- [Execution Lifecycle](#execution-lifecycle)
- [Deliverable Storage](#deliverable-storage)
- [Agent Orchestrator](#agent-orchestrator)

---

## Agent Model

### Source: `src/internal/model/agent.go`

Agents represent AI workers that execute tasks within a project. Each agent has a type, role, status, and set of capabilities.

```go
type Agent struct {
    ID           uuid.UUID       `json:"id"`
    Name         string          `json:"name"`
    Type         string          `json:"type"`
    Role         string          `json:"role"`
    Model        string          `json:"model"`
    Provider     string          `json:"provider"`
    Capabilities []string        `json:"capabilities"`
    Status       AgentStatus     `json:"status"`
    ProjectID    string          `json:"project_id,omitempty"`
    Config       json.RawMessage `json:"config,omitempty"`
    CurrentTaskID string         `json:"current_task_id,omitempty"`
    TasksDone    int             `json:"tasks_completed,omitempty"`
    Uptime       int             `json:"uptime,omitempty"`
    CreatedAt    time.Time       `json:"created_at"`
    UpdatedAt    time.Time       `json:"updated_at"`
}
```

### Agent Types (Roles)

| Type | Label | Purpose |
|------|-------|---------|
| `pm` | Project Manager | Requirement analysis, task decomposition |
| `architect` | Architect | System design, API design |
| `developer` | Developer | Code implementation |
| `reviewer` | Reviewer | Code review, security scanning |
| `qa` | QA Engineer | Test planning, test execution |
| `devops` | DevOps Engineer | CI/CD, deployment, infrastructure |

### Status Lifecycle

```
    ┌──────────┐
    │ Spawning │  Agent is being created / initialized
    └────┬─────┘
         │
         ▼
    ┌──────────┐
    │   Idle   │  Agent is ready and waiting for work
    └────┬─────┘
         │
         ▼
    ┌──────────┐      ┌───────────┐
    │ Working  │──────│ Completed │  Agent finished all tasks
    └──────────┘      └───────────┘
         │
         ▼
    ┌──────────┐
    │  Failed  │  Agent encountered an unrecoverable error
    └──────────┘
```

Transitions:
- `spawning` → `idle` (agent initialized)
- `idle` → `working` (task assigned)
- `working` → `idle` (task completed, ready for next)
- `working` → `completed` (agent finished all work)
- `working` → `failed` (error)

---

## Capability System

### Source: `src/internal/service/capability.go`

The capability system provides matching logic between agents and tasks. It uses a scoring algorithm to find the best agent for a given task.

### Available Capabilities (12)

| Capability | Description |
|------------|-------------|
| `requirement_analysis` | Analyzing project requirements |
| `task_decomposition` | Breaking work into tasks |
| `system_design` | Designing system architecture |
| `api_design` | Designing API contracts |
| `code_implementation` | Writing code |
| `code_review` | Reviewing code changes |
| `security_scan` | Security vulnerability scanning |
| `test_planning` | Creating test plans |
| `test_execution` | Running tests |
| `ci_cd` | CI/CD pipeline management |
| `deployment` | Deploying to environments |
| `infrastructure` | Infrastructure provisioning |

### Role-to-Capability Mapping

```go
AgentTypeCapabilities = map[AgentType][]AgentCapability{
    AgentPM:       {"requirement_analysis", "task_decomposition"},
    AgentArch:     {"system_design", "api_design"},
    AgentDev:      {"code_implementation"},
    AgentReviewer: {"code_review", "security_scan"},
    AgentQA:       {"test_planning", "test_execution"},
    AgentDevOps:   {"ci_cd", "deployment", "infrastructure"},
}
```

When an agent is created without specifying capabilities, the default set for its type is used automatically via `DefaultCapabilitiesForType()`.

### Task-to-Capability Mapping

When assigning a task to an agent, the system determines required capabilities based on task type:

| Task Type | Required Capabilities |
|-----------|---------------------|
| `feature`, `implementation` | coding, testing |
| `architecture`, `design` | architecture |
| `bugfix` | coding |
| `review` | testing, security |
| `test`, `qa` | testing |
| `security_audit` | security |
| `deployment`, `infrastructure` | devops, architecture |
| `documentation` | documentation |
| `data_pipeline`, `analytics` | data_engineering, coding |
| `project_management`, `planning` | project_management |
| default | coding |

### Capability Scoring

The `AssignmentScore()` method calculates a numerical match:

| Condition | Score Change |
|-----------|-------------|
| Capability matches required | **+2** |
| Agent has extra capability (breadth bonus) | **+1** |
| Required capability missing | **-5** (disqualifies) |

Higher scores indicate better agent-task fit.

### Compatibility Check

`FindCompatibleAgents()` filters a list of agents to only those possessing ALL required capabilities. Compatible agents must have every capability in the required set.

---

## Task Assignment Flow

### Source: `src/internal/service/assignment.go`

The assignment flow binds a task to an agent and creates an execution record.

```
POST /v1/tasks/{taskId}/assign
Body: { "agent_id": "uuid" }
```

### Step-by-Step Flow

```
┌──────────────┐
│   Request    │  POST /v1/tasks/:id/assign  { agent_id }
└──────┬───────┘
       │
       ▼
┌──────────────┐      ┌──────────────────┐
│ Validate     │──────│ Task not found?   │→ 404 NOT_FOUND
│ Task exists  │      └──────────────────┘
└──────┬───────┘
       │
       ▼
┌──────────────┐      ┌──────────────────┐
│ Validate     │──────│ Agent not found?  │→ 404 NOT_FOUND
│ Agent exists │      └──────────────────┘
└──────┬───────┘
       │
       ▼
┌──────────────┐      ┌──────────────────────┐
│ Check agent  │──────│ Agent not idle?       │→ 409 CONFLICT
│ is idle      │      └──────────────────────┘
└──────┬───────┘
       │
       ▼
┌──────────────┐      ┌───────────────────────────┐
│ Capability   │──────│ Agent lacks capabilities?  │→ 422 CAPABILITY_MISMATCH
│ check        │      └───────────────────────────┘
└──────┬───────┘
       │
       ▼
┌──────────────────┐
│ Create Execution  │  Status: running, StartedAt: now
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Update Task      │  Status: in_progress, assignee: agent ID
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Update Agent     │  Status: working
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Return Execution │  { execution_id, task_id, agent_id, status }
└──────────────────┘
```

---

## Execution Lifecycle

### Source: `src/internal/model/execution.go`, `src/internal/service/execution.go`

An execution records the lifecycle of an agent working on a task.

```go
type Execution struct {
    ExecutionID uuid.UUID       `json:"execution_id"`
    TaskID      uuid.UUID       `json:"task_id"`
    AgentID     uuid.UUID       `json:"agent_id"`
    Status      ExecutionStatus `json:"status"`
    StartedAt   *time.Time      `json:"started_at,omitempty"`
    CompletedAt *time.Time      `json:"completed_at,omitempty"`
    CreatedAt   time.Time       `json:"created_at"`
}
```

### Status State Machine

```
    ┌──────────┐
    │ Pending  │  Execution created but not yet started
    └────┬─────┘
         │
         ▼
    ┌──────────┐
    │ Running  │  Agent is actively working
    └────┬─────┘
       ┌─┴──┐
       │    │
       ▼    ▼
  ┌───────────┐  ┌──────────┐
  │ Completed │  │  Failed  │
  └───────────┘  └──────────┘
```

Allowed transitions:
- `pending` → `running`
- `running` → `completed`
- `running` → `failed`
- `completed` → (terminal — no transitions out)
- `failed` → (terminal — no transitions out)

Invalid transitions return HTTP `422 Unprocessable Entity`.

### API

| Endpoint | Description |
|----------|-------------|
| `POST /v1/executions` | Create execution (sets to running) |
| `GET /v1/executions` | List with optional task_id/agent_id filter |
| `GET /v1/executions/:id` | Get execution details |
| `PATCH /v1/executions/:id/status` | Update status (validated transition) |

Helper method:
- `CompleteExecution(ctx, id)` — shorthand for `UpdateExecutionStatus(ctx, id, "completed")`

---

## Deliverable Storage

### Source: `src/internal/model/deliverable.go`, `src/internal/service/deliverable.go`

Deliverables are artifacts produced by an agent while working on a task. They support version tracking for iterative improvements.

```go
type Deliverable struct {
    ID        uuid.UUID `json:"id"`
    TaskID    uuid.UUID `json:"task_id"`
    AgentID   uuid.UUID `json:"agent_id"`
    Title     string    `json:"title"`
    Content   string    `json:"content"`
    Version   int       `json:"version"`
    CreatedAt time.Time `json:"created_at"`
}
```

### Version Tracking

- **On creation:** Version is set to `1`
- **On update:** Version auto-increments (`d.Version++`)
- Updates modify `Title` and `Content` fields in place
- List endpoints return all deliverables in descending creation order

### API

| Endpoint | Description |
|----------|-------------|
| `POST /v1/deliverables` | Create deliverable (version 1) |
| `GET /v1/deliverables` | List by `task_id` or `agent_id` (exactly one required) |
| `GET /v1/deliverables/:id` | Get single deliverable |
| `PUT /v1/deliverables/:id` | Update (auto-increments version) |

### Usage Pattern

1. Agent completes work → calls `POST /v1/deliverables`
2. Agent refines output → calls `PUT /v1/deliverables/:id` (version increments)
3. Other services query deliverables by task to retrieve the latest artifact

---

## Agent Orchestrator

### Source: `src/internal/service/orchestrator.go`

The `AgentOrchestrator` manages the lifecycle of agent containers. It interfaces with the Docker daemon to spawn and monitor agent processes.

```go
type AgentOrchestrator interface {
    StartMonitoring(ctx context.Context)
    HandleAgentFailure(agentID string) error
    SpawnAgentProcess(ctx context.Context, agent *model.Agent) error
}
```

### Spawning

`SpawnAgentProcess` creates a Docker container with:
- Image: `ai-software-factory-agent:latest`
- Env: `AGENT_ID=<uuid>` injected
- Memory limit: 512 MB
- CPU quota: 0.5 core
- Auto-remove on exit

### Monitoring

`StartMonitoring` runs a periodic health check (every 30 seconds) in a background goroutine. When the context is cancelled, monitoring stops cleanly.

### Failure Handling

`HandleAgentFailure` is a stub for future implementation. It will attempt to recover or reassign tasks from failed agents.

---

## Data Model Relationships

```
Project
    │
    ├── Tasks ───────────┐
    │                     │
    │     POST /tasks/:id/assign
    │         │
    │         ▼
    │    Execution
    │    (task_id, agent_id, status, started_at, completed_at)
    │         │
    │         ▼
    │    Deliverable
    │    (task_id, agent_id, title, content, version)
    │
    └── Agents
        (type, role, capabilities, status)
```
