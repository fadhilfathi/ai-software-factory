# Agent Orchestration Design — Sprint 4 (Canonical)

> **Status:** Canonical design, owned by Analyst-01 (TASK-401).
> **Audience:** Developer-01 (TASK-402/403/404/405/406), Developer-02 (TASK-407/408/409/410).
> **Scope:** Foundation for the Agent Orchestration Engine. ~9 downstream tasks depend on this doc.

This is the **canonical** orchestration design. Companion docs in this folder add detail:

- `data-model.md` — Postgres schema for the entities described here.
- `api-spec.md` — HTTP surface for the entities described here.
- `agent-orchestration-guide.md` — Long-form rationale, state-machine narrative, examples.
- `code-service-logic.md`, `review-service-design.md`, `security-audit-design.md` — Sprint 4 sub-engine deep dives.
- `quality-gates.md`, `code-review-specs.md` — Cross-cutting concerns.

---

## 1. Agent Entity

An **Agent** is a long-lived, named, role-bearing execution unit that can be assigned work inside a project. Agents are **not** ephemeral: once created they persist, change state, and accumulate history.

### 1.1 Fields

| Field             | Type        | Required | Notes                                                                                                  |
|-------------------|-------------|----------|--------------------------------------------------------------------------------------------------------|
| `id`              | UUID        | yes      | Primary key, server-generated.                                                                         |
| `project_id`      | UUID        | yes      | FK → `projects.id`. An agent always belongs to exactly one project.                                    |
| `name`            | string      | yes      | Human-readable, unique per project. 1–80 chars.                                                        |
| `role`            | string      | yes      | Free-form role label, e.g. `Backend Developer`, `Security Reviewer`. 1–80 chars.                       |
| `status`          | enum        | yes      | Lifecycle state. See §2.                                                                               |
| `capabilities`    | string[]    | yes      | Capability names this agent can perform. Validated against the `capabilities` catalog. Min 1.          |
| `last_active_at`  | timestamptz | no       | Updated on every state transition. `NULL` until first activation.                                      |
| `metadata`        | jsonb       | no       | Free-form: model name, version, tool allow-list, notes. Default `{}`.                                  |
| `created_at`      | timestamptz | yes      | Server-set.                                                                                            |
| `updated_at`      | timestamptz | yes      | Server-maintained.                                                                                     |
| `retired_at`      | timestamptz | no       | Set when the agent enters `retired`. Never cleared.                                                    |

> **Note:** Existing Sprint 1–3 schema already has `agents` with a JSONB `metadata` blob and an inline `capabilities` array. Sprint 4 **adds** a normalized `agent_capabilities` join table for relational queries; the inline `capabilities` array on the agent is kept in sync as a denormalized cache. See `data-model.md` §3.

### 1.2 Invariants

- An agent's `project_id` is **immutable**. Cross-project moves are modeled as retire + recreate.
- An agent in state `retired` cannot be assigned new work. Existing assignments complete.
- `name` is unique per `project_id` (`UNIQUE(project_id, name)`).
- An agent must always have ≥ 1 capability. Removing the last capability is rejected unless the agent is being retired.

---

## 2. Lifecycle States

The Agent has six states. The canonical state machine is described here; deeper rationale and worked examples are in `agent-orchestration-guide.md`.

```
        ┌──────────────────┐
        │   initializing   │  (transient, on creation)
        └────────┬─────────┘
                 │ first heartbeat / ready
                 ▼
        ┌──────────────────┐  ◀─────────────┐
        │       idle       │                │
        └────────┬─────────┘                │
                 │ task assigned            │ task completed / failed-fast
                 ▼                          │
        ┌──────────────────┐                │
        │       busy       │ ───────────────┘
        └────────┬─────────┘
                 │ operator / leader request
                 ▼
        ┌──────────────────┐
        │      paused      │   (mid-task, not consuming events)
        └────────┬─────────┘
                 │ resume
                 ▼
             (back to busy)
                 │ unrecoverable failure / panic
                 ▼
        ┌──────────────────┐
        │       error      │   (terminal for the run; operator decides next step)
        └────────┬─────────┘
                 │ operator: retire
                 ▼
        ┌──────────────────┐
        │      retired     │   (terminal; soft-deleted)
        └──────────────────┘
```

| State          | Set by               | Cleared by              | Assignable? | Notes |
|----------------|----------------------|-------------------------|-------------|-------|
| `initializing` | `POST /v1/agents`    | First successful health-check or first task assignment | No | Set at row insert, max 30 s. |
| `idle`         | Agent runtime        | `POST /v1/tasks/:id/assign` success                | Yes | Default steady state. |
| `busy`         | Assignment service   | Execution terminal event (success / fail / cancel)  | No | Holds 0..N executions (typically 1). |
| `paused`       | Leader / operator    | Leader / operator resume                            | No | Mid-execution pause; resumable. |
| `error`        | Execution service    | Operator clears → `idle`, or operator retires → `retired` | No | Agent is alive but last task failed. |
| `retired`      | Operator             | (terminal)                                           | No | Soft-deleted; excluded from listings by default. |

### 2.1 Who transitions

| From → To          | Trigger                                        | Actor / Service                |
|--------------------|------------------------------------------------|--------------------------------|
| (none) → `initializing` | Agent row insert                          | `agent.Service.Create`         |
| `initializing` → `idle` | Health-check OK, or first task assigned | `agent.Service` / `assignment.Service` |
| `idle` → `busy`    | Successful task assignment                      | `assignment.Service.Assign`    |
| `busy` → `paused`  | Leader or operator request                     | `agent.Service.Pause` (or `PATCH /v1/agents/:id` with `status: "paused"`) |
| `paused` → `busy`  | Resume                                         | `agent.Service.Resume`         |
| `busy` → `idle`    | Execution terminal (succeeded)                 | `execution.Service` (event)    |
| `busy` → `error`   | Execution terminal (failed / panicked)         | `execution.Service` (event)    |
| `error` → `idle`   | Operator clears the error                      | `agent.Service.ClearError`     |
| `error` → `retired` | Operator retires                              | `agent.Service.Retire` (or `DELETE /v1/agents/:id`) |
| any → `retired`    | Operator retires                               | `agent.Service.Retire`         |

> All transitions are written to an audit log (table `agent_state_events`, see `data-model.md` §6) so that the team dashboard (TASK-410) can show history.

---

## 3. Capability Entity

A **Capability** is a named, cataloged skill that an Agent can declare. Capabilities are global (not per-project): the catalog is shared so the UI can render filters and the assignment engine can match deterministically.

### 3.1 Fields

| Field        | Type        | Notes                                                                                                |
|--------------|-------------|------------------------------------------------------------------------------------------------------|
| `id`         | UUID        | Primary key.                                                                                         |
| `name`       | string      | Stable machine name, e.g. `coding`, `testing`. Unique. Lowercase, `[a-z][a-z0-9-]{0,40}`.            |
| `display_name` | string    | Human label, e.g. `Coding`. 1–80 chars.                                                              |
| `category`   | enum/string | One of: `architecture`, `coding`, `testing`, `security`, `devops`, `leadership`.                     |
| `description`| string      | Optional, ≤ 500 chars.                                                                               |
| `version`    | int         | Monotonically increasing; bumped when the capability definition changes. Default 1.                  |
| `created_at` | timestamptz | Server-set.                                                                                          |
| `updated_at` | timestamptz | Server-maintained.                                                                                   |

### 3.2 The standard capability catalog (seed data)

These six categories cover everything the platform supports this sprint. The system is open to extension via `POST /v1/capabilities` (admin-only) but the seed set is fixed:

| Name            | Display Name   | Category     | Purpose                                                                |
|-----------------|----------------|--------------|------------------------------------------------------------------------|
| `architecture`  | Architecture   | architecture | Produces system design, ADR, dependency choices.                        |
| `coding`        | Coding         | coding       | Writes source code, tests in the source tree.                          |
| `testing`       | Testing        | testing      | Runs the test suite, reports coverage, files bug reports.              |
| `security`      | Security       | security     | Threat modeling, code audit, secret scanning.                          |
| `devops`        | DevOps         | devops       | Builds, deploys, infra-as-code, monitoring.                            |
| `leadership`    | Leader         | leadership   | Plans, decomposes work, dispatches tasks, resolves conflicts. Reserved for the **Lead** agent role. |

> **Why a separate `leadership` capability:** the Leader agent is structurally different — it does not own deliverables, it owns the assignment workflow itself. Encoding it as a capability lets the assignment engine express "the only agent that can take a planning task is the Leader" via the same rules engine as everything else.

### 3.3 Custom capabilities

Operators can add domain-specific capabilities (e.g. `mobile-ios`, `data-engineering`) via `POST /v1/capabilities`. Once added, any agent can declare them. Reserved names: `__system__*`.

### 3.4 Per-agent link

A many-to-many `agent_capabilities` table joins agents to capabilities. This is **redundant** with the `agents.capabilities` text[] column, but the join is what the assignment engine queries. See `data-model.md` §3.

---

## 4. Assignment Rules

The **Assignment Engine** (TASK-404) decides, given a Task, which Agent gets it. The decision is deterministic and explainable: a list of `(rule, passed, evidence)` tuples is returned with the assignment.

### 4.1 Inputs

- `task.project_id`
- `task.required_capability` (string; must be a name from the capability catalog)
- `task.priority` (informational only — does not block, may bias tie-breaking)
- `task.id` (for "is this task already assigned?" check)
- Current agent pool filtered to the same `project_id`

### 4.2 Rules (evaluated in order, all must pass)

| # | Rule                                | Check                                                                                       | On fail            |
|---|-------------------------------------|----------------------------------------------------------------------------------------------|--------------------|
| 1 | Project scope                       | Agent's `project_id == task.project_id`                                                      | Skip agent         |
| 2 | Capability match                    | `task.required_capability` ∈ agent's capabilities (via `agent_capabilities`)                 | Skip agent         |
| 3 | Availability                        | Agent's `status` ∈ {`idle`}                                                                  | Skip agent         |
| 4 | Not over-committed                  | Agent's open execution count < `MAX_CONCURRENT_TASKS` (config; default 1)                    | Skip agent         |
| 5 | Not paused for this task            | No `paused` reason matching this task's tags (forward-compat)                                | Skip agent         |
| 6 | Not retired                         | `retired_at IS NULL`                                                                        | Skip agent         |

### 4.3 Tie-breaking (when multiple agents pass)

Default strategy: **least-recently-active wins** (the agent whose `last_active_at` is oldest, or `NULL` first). This is overridable per request:

```json
POST /v1/tasks/:id/assign
{
  "strategy": "least_recently_active"   // default
  // future: "round_robin", "random", "explicit"
}
```

The response always includes `candidates_considered` (count) and `selected_reason`, so the UI can show *why* this agent was chosen.

### 4.4 Manual override

`POST /v1/tasks/:id/assign` accepts an optional `agent_id` body field. When present, the rules above are **skipped except** for Rule 1 (project scope) and Rule 3 (must be `idle` or `busy` with capacity). The response's `selected_reason` is `"manual_override"`. This is the only way to bypass Rule 2 (capability match).

### 4.5 Failure modes

If no agent passes, the endpoint returns `409 NO_AGENT_AVAILABLE` with a `candidates_considered: 0` and a `hint` describing which rule failed for the closest match. See `api-spec.md` §5.

### 4.6 What the Leader can do

The Leader agent is the only one with the `leadership` capability, but assignment of leadership tasks is **also** routed through the same engine. The Leader is itself an `Agent` row with `status` cycling between `busy` and `idle` like any other. There is no special-case code path.

---

## 5. Cross-references

- Agent/Capability/Execution/Deliverable Postgres schema: [`data-model.md`](./data-model.md)
- HTTP endpoints: [`api-spec.md`](./api-spec.md)
- Deep state-machine narrative: [`agent-orchestration-guide.md`](./agent-orchestration-guide.md)
- How an assignment flows through the system end-to-end: [`agent-orchestration-guide.md` §"Assignment flow"](./agent-orchestration-guide.md)
- Execution lifecycle (separate from agent lifecycle): [`agent-orchestration-guide.md` §"Execution"](./agent-orchestration-guide.md)
- Sprint 4 test plan for assignment rules: see `quality-gates.md` and the TASK-411 acceptance criteria.

---

## 6. Open questions / forward work

- **Multi-project agents:** explicitly out of scope for Sprint 4; tracked for Sprint 6+.
- **Skill levels / confidence:** the catalog is binary (has it / doesn't). Skill levels (junior, senior) are an extension point on `agent_capabilities.proficiency`, deferred.
- **Real Hermes integration:** Sprint 5. Sprint 4 is **MOCK execution** — `execution.Service` returns synthetic outputs.
