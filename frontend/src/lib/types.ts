// Centralised domain types for the frontend.
// Source of truth: docs/sprint4/api-spec.md (Sprint 4 canonical spec).
// Where the spec is silent, fields are typed defensively (optional) so the
// frontend can degrade gracefully if the backend adds fields later.

/* ---------- Common ---------- */

export type ISODateString = string

export type PageInfo = {
  next_cursor: string | null
  has_more: boolean
}

export type PaginatedResponse<T> = {
  data: T[]
  page_info: PageInfo
}

export type ApiEnvelope<T> = {
  data: T
  request_id?: string
}

/* ---------- Agents ---------- */

/**
 * Legacy status enum used by the Sprint 1-3 UI (StatusBadge.tsx and
 * the /projects, /tasks, /dashboard pages). The official spec doesn't
 * define a Project.status, so this is a frontend-only projection.
 * Derive values from `updated_at` recency when reading from the API
 * (see e.g. `updatedMoreThan30dAgo` in ProjectCard).
 */
export type ProjectStatus = "initializing" | "active" | "in_progress" | "completed" | "archived"

/**
 * Legacy task status enum (kept here for back-compat with the legacy
 * StatusBadge component). The real TaskStatus is defined below in the
 * Tasks section. Pre-Sprint-1 pages still import this name.
 */
export type LegacyTaskStatus = "backlog" | "ready" | "in_progress" | "review" | "done" | "blocked"

/**
 * Legacy agent status enum (renamed to `AgentStatus` in Sprint 4).
 * Kept as a separate type so the legacy badge that lives in
 * StatusBadge.tsx still works.
 */
export type AgentStatus_ = "spawning" | "idle" | "working" | "completed" | "failed"

/**
 * Free-form role / type label shown in the agent badge. The spec §1.1
 * doesn't constrain this — any string is acceptable, but the UI
 * shortens the most common labels to colored dots (see AgentBadge).
 */
export type AgentType =
  | "pm"
  | "architect"
  | "developer"
  | "reviewer"
  | "qa"
  | "devops"
  | "security"
  | "techwriter"
  | (string & {})

/** Status enum from spec §1.1 / §1.2. */
export type AgentStatus =
  | "initializing"
  | "idle"
  | "busy"
  | "paused"
  | "error"
  | "retired"

export const AGENT_STATUSES: AgentStatus[] = [
  "initializing",
  "idle",
  "busy",
  "paused",
  "error",
  "retired",
]

/**
 * Agent metadata payload. Free-form per spec §1.1, but the UI conventionally
 * uses these keys for display (model, provider, etc.). The backend stores
 * metadata as JSONB and returns it as an object.
 */
export type AgentMetadata = {
  model?: string
  provider?: string
  type?: string
  description?: string
  [key: string]: unknown
}

/**
 * Spec §1.1 / §1.3 response shape. `capabilities` is a string array of
 * capability names; for the rich shape (with category, proficiency) call
 * GET /v1/agents/:id/capabilities (see AgentCapability below).
 */
export type Agent = {
  id: string
  project_id: string
  name: string
  role: string
  status: AgentStatus
  capabilities: string[]
  last_active_at: string | null
  metadata: AgentMetadata
  created_at: ISODateString
  updated_at: ISODateString
  // Optional fields. Some come from spec §1.4 (version) or the Lead's brief;
  // they may or may not be present depending on backend shape.
  version?: number
  description?: string
  retired_at?: string | null
  // Derived/computed fields (per Lead's brief). Optional — populated by the
  // backend when available; UI falls back to defaults otherwise.
  active_assignments?: number
  tasks_completed?: number
  success_rate?: number
  uptime_seconds?: number
}

/** Spec §1.1 POST body. */
export type CreateAgentPayload = {
  project_id: string
  name: string
  role: string
  capabilities: string[]
  metadata?: AgentMetadata
}

/**
 * Spec §1.4 PUT body. "only role, capabilities, and metadata are mutable.
 * name, project_id, status are NOT mutable here".
 * `version` is the expected current version (optimistic concurrency); on
 * mismatch the backend returns 409 VERSION_CONFLICT.
 */
export type UpdateAgentPayload = {
  role?: string
  capabilities?: string[]
  metadata?: AgentMetadata
  version?: number
}

/** Spec §1.2 query params. */
export type AgentListFilters = {
  project_id?: string
  status?: AgentStatus | "all"
  capability?: string | string[]
  /**
   * Free-text role filter. Not in the Sprint 4 spec §1.4 filter list,
   * but the agents index page lets the user filter by role label
   * anyway — the backend either supports it or silently ignores it.
   */
  role?: string
  search?: string
  sort?: "name" | "-name" | "last_active_at" | "-last_active_at" | "created_at" | "-created_at"
  cursor?: string
  /** Accepts number or string for legacy callers (the wire format coerces). */
  limit?: number | string
  include_retired?: boolean
}

/** Spec §1.6 response item. */
export type AgentCapability = {
  name: string
  display_name: string
  category: string
  proficiency: number // 1-5
  granted_at: ISODateString
}

/** Spec §2.1 response item. */
export type CapabilityCatalogItem = {
  name: string
  display_name: string
  category: string
  version: number
}

export type CapabilityCategory =
  | "architecture"
  | "coding"
  | "testing"
  | "security"
  | "devops"
  | "leadership"

/** Spec §2.1 query params. */
export type CapabilityListFilters = {
  category?: CapabilityCategory
  cursor?: string
  limit?: number
}

/* ---------- Executions ---------- */

export type ExecutionStatus =
  | "pending"
  | "running"
  | "succeeded"
  | "failed"
  | "cancelled"

export type Execution = {
  id: string
  task_id: string
  agent_id: string
  status: ExecutionStatus
  started_at: ISODateString
  completed_at: ISODateString | null
  output?: Record<string, unknown>
  error?: string | null
}

export type ExecutionListFilters = {
  task_id?: string
  agent_id?: string
  project_id?: string
  status?: ExecutionStatus
  cursor?: string
  /** Accepts number or string for legacy callers (the wire format coerces). */
  limit?: number | string
}

/* ---------- Deliverables ---------- */

export type DeliverableKind =
  | "code"
  | "doc"
  | "design"
  | "test_report"
  | "config"
  | "other"

/**
 * Per TASK-406 / Lead's brief (2026-06-12):
 *   { id, task_id, agent_id, title, content, version, created_at, updated_at }
 *
 * `version` is the latest version integer on the row (the backend
 * returns the current version inline; full history lives in
 * `DeliverableVersion` and is fetched separately).
 *
 * `kind`, `description`, `metadata` are optional forward-compat fields
 * — the TASK-409 list view uses `kind` to color chips but never
 * assumes it exists, and the older agent detail page reads
 * `latest_version` / `description` for back-compat. Both sides stay
 * happy regardless of which fields the backend sends.
 */
export type Deliverable = {
  id: string
  task_id: string
  agent_id: string
  project_id: string
  title: string
  content: string
  /** Current version of this row (matches `latest_version` when both present). */
  version: number
  created_at: ISODateString
  /** Sprint 4 brief lists this; the current backend may not yet emit it. */
  updated_at?: ISODateString
  /** Alias for `version`, present on older data. */
  latest_version?: number
  kind?: DeliverableKind
  description?: string
  metadata?: Record<string, unknown>
}

/**
 * Per TASK-406 / Lead's brief (2026-06-12):
 *   { id, deliverable_id, version, title, content, created_at, created_by? }
 *
 * `created_by` is the user-id (UUID, nullable) that pushed the version
 * via the append-only PUT /v1/deliverables/:id flow.
 */
export type DeliverableVersion = {
  id: string
  deliverable_id: string
  version: number
  title: string
  content: string
  created_at: ISODateString
  /** UUID of the user who created this version. Nullable in the schema. */
  created_by?: string | null
}

export type DeliverableListFilters = {
  project_id?: string
  task_id?: string
  agent_id?: string
  kind?: DeliverableKind
  cursor?: string
  limit?: number | string
}

/* ---------- Activity / history ---------- */

/**
 * History row (e.g. agent_state_events or assignment_events). The exact
 * shape is documented in docs/sprint4/data-model.md (migration 020) — not
 * in api-spec.md. Typed defensively here; the UI uses a generic timeline
 * shape that only needs `id`, `type`, `at`, `title`, `description`.
 */
export type ActivityEvent = {
  id: string
  type:
    | "agent_state_change"
    | "capability_assigned"
    | "capability_revoked"
    | "task_assigned"
    | "task_unassigned"
    | "execution_started"
    | "execution_completed"
    | "execution_failed"
    | "deliverable_created"
    | "deliverable_updated"
  at: ISODateString
  title: string
  description?: string
  agent_id?: string
  actor_id?: string
  metadata?: Record<string, unknown>
}

/* ---------- Metrics ---------- */

/**
 * The /v1/agents/metrics endpoint shape is not in api-spec.md (Lead's brief
 * referenced §1.7 but no such section exists). Typed defensively; the hook
 * returns whatever the backend sends and the UI tolerates missing fields.
 */
export type AgentMetrics = {
  metrics: {
    total_agents?: number
    active_agents?: number
    total_tasks?: number
    success_rate?: number
    avg_uptime_seconds?: number
    [key: string]: unknown
  }
  by_role?: Array<{
    role: string
    count: number
    avg_success_rate?: number
    [key: string]: unknown
  }>
  pagination?: PageInfo
}

/* ---------- Project ---------- */

export type Project = {
  id: string
  name: string
  slug?: string
  description?: string
  created_at?: ISODateString
  updated_at?: ISODateString
}

export type ProjectListResponse = {
  data: Project[]
  page_info?: PageInfo
}

/* ---------- Tasks ---------- */

/**
 * Task entity. Source of truth: docs/api-spec.md (Sprint 1-3) §tasks.
 * `position` is a per-status ordinal the Kanban board uses for ordering.
 */
export type TaskStatus =
  | "backlog"
  | "ready"
  | "in_progress"
  | "review"
  | "done"
  | "blocked"

export const TASK_STATUSES: TaskStatus[] = [
  "backlog",
  "ready",
  "in_progress",
  "review",
  "done",
  "blocked",
]

export type TaskPriority = "low" | "medium" | "high" | "critical"

export const TASK_PRIORITIES: TaskPriority[] = [
  "low",
  "medium",
  "high",
  "critical",
]

export type Task = {
  id: string
  project_id: string
  title: string
  description?: string
  status: TaskStatus
  priority: TaskPriority
  assignee_id?: string | null
  position: number
  /** Optional. Some pages show it as a read-only chip strip. */
  required_capabilities?: string[]
  created_at: ISODateString
  updated_at: ISODateString
}

/** Filter for GET /v1/projects/:projectId/tasks (Sprint 1-3 spec §tasks). */
export type TaskListFilters = {
  project_id?: string
  status?: TaskStatus
  priority?: TaskPriority
  assignee_id?: string
  search?: string
  page?: number
  limit?: number | string
  sort?: "created_at" | "-created_at" | "position" | "-position"
}

/** Body for POST /v1/projects/:projectId/tasks. */
export type CreateTaskPayload = {
  projectId: string
  title: string
  description?: string
  priority?: TaskPriority
}

/** Body for PUT /v1/tasks/:id. */
export type UpdateTaskPayload = {
  title?: string
  description?: string
  priority?: TaskPriority
  assignee_id?: string | null
}

/** Body for PATCH /v1/tasks/:id/status. */
export type UpdateTaskStatusPayload = {
  id: string
  status: TaskStatus
}

/* ---------- Assignment ---------- */

/**
 * Assignment event row (e.g. assignment_events table). The exact shape is
 * documented in docs/sprint4/data-model.md (assignment_events table).
 * Typed defensively — backend may add fields (e.g. before/after state,
 * strategy, etc.) without a frontend contract update.
 */
export type AssignmentEventType = "assign" | "reassign" | "release" | "unassign"

export type AssignmentEvent = {
  id: string
  task_id: string
  assignment_id?: string
  agent_id?: string | null
  agent_name?: string
  project_id?: string
  event_type: AssignmentEventType
  notes?: string
  /** Whoever triggered the event (user id or "system"). */
  actor_id?: string
  created_at: ISODateString
}

/** Response envelope for GET /v1/tasks/:id/history (Lead's brief, TASK-404). */
export type TaskHistoryResponse = {
  data: AssignmentEvent[]
  meta: {
    count: number
    server_time: ISODateString
  }
}

/**
 * Body for POST /v1/tasks/:id/assign.
 *
 * Per Lead's TASK-408 brief (2026-06-12), the shape is:
 *   { agent_id, capabilities_required?, notes? }
 *
 * NOTE: this differs from the Sprint 1-3 spec body (`{ agent_id }`) and
 * the Sprint 4 spec body (`{ agent_id?, strategy?, reason? }`). The
 * brief is the most recent and authoritative source for TASK-408. If
 * the backend rejects unknown fields, this is a real gap.
 */
export type AssignTaskPayload = {
  taskId: string
  agent_id: string
  capabilities_required?: string[]
  notes?: string
}

/**
 * Response from POST /v1/tasks/:id/assign. Per Lead's brief:
 *   { data: { task, event, idempotent } }
 * The outer envelope may be omitted in some responses; we accept both.
 */
export type AssignTaskResult = {
  task: Task
  event: AssignmentEvent | null
  idempotent: boolean
}
