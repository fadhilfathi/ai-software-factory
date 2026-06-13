package store

import (
	"context"
	"errors"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/google/uuid"
)

// Sentinel errors for store operations.
var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrConflict      = errors.New("conflict")
)

// UserStore defines persistence operations for users.
type UserStore interface {
	Create(user *model.User) error
	GetByID(id uuid.UUID) (*model.User, error)
	GetByEmail(email string) (*model.User, error)
	List() ([]*model.User, error)
	Update(user *model.User) error
	// CheckProjectAccess returns true if the user has access to the project
	CheckProjectAccess(userID, projectID uuid.UUID) bool
}

// ProjectStore defines persistence operations for projects.
type ProjectStore interface {
	Create(project *model.Project) error
	GetByID(id uuid.UUID) (*model.Project, error)
	List(filter ProjectFilter) ([]*model.Project, int, error)
	Update(project *model.Project) error
	Delete(id uuid.UUID) error
}

// ProjectFilter holds optional query parameters for listing projects.
type ProjectFilter struct {
	Status  model.ProjectStatus
	OwnerID uuid.UUID
	Search  string
	Page    int
	Limit   int
}

// AgentStore defines persistence operations for agents (api-spec.md §1).
//
// All methods take a context for cancellation. The store is the
// single source of truth for keeping agents.capabilities (JSONB cache)
// in sync with the agent_capabilities join table (data-model.md §3
// invariant): SetCapabilities is the only allowed write path for
// capabilities and updates both the join and the cache in one
// transaction.
type AgentStore interface {
	// Create inserts a new agent. Sets ID/Version/CreatedAt/UpdatedAt
	// on the supplied model.Agent. Returns ErrAlreadyExists on a
	// (project_id, name) collision.
	Create(ctx context.Context, agent *model.Agent) error

	// GetByID returns the agent with the given UUID, or ErrNotFound.
	GetByID(ctx context.Context, id uuid.UUID) (*model.Agent, error)

	// List returns a cursor-paginated page of agents matching the
	// filter. See model.AgentFilter for the parameter fields. The
	// store honours the partial index `idx_agents_project_status
	// ... WHERE retired_at IS NULL` for the active-agent path and
	// falls back to a full scan when IncludeRetired is true.
	List(ctx context.Context, filter model.AgentFilter) (*model.AgentListResult, error)

	// Update applies a partial update. The store bumps Version on
	// success and sets UpdatedAt. The caller is responsible for
	// optimistic-concurrency: pass the row's current Version and
	// the store will fail with ErrConflict on mismatch.
	Update(ctx context.Context, agent *model.Agent) error

	// SoftDelete transitions the agent to status=retired with
	// retired_at=NOW() and bumps the version. Returns ErrNotFound
	// if the id is unknown. The api-spec.md §1.5 ?force=true path
	// is the handler's concern; the store always does a soft delete.
	SoftDelete(ctx context.Context, id uuid.UUID) error

	// SetCapabilities is the canonical write path for the agent's
	// capability list (data-model.md §3). It (1) upserts the
	// agent_capabilities join rows, (2) updates the
	// agents.capabilities JSONB cache, (3) bumps Version, all in a
	// single transaction. An empty names slice is a no-op except
	// for the version bump.
	SetCapabilities(ctx context.Context, agentID uuid.UUID, names []string) error

	// ListCapabilitiesByAgent returns the agent's granted
	// capabilities with proficiency/granted_at metadata. Mirrors
	// api-spec.md §1.6.
	ListCapabilitiesByAgent(ctx context.Context, agentID uuid.UUID) ([]*model.AgentCapabilityView, error)
}

// CapabilityStore defines persistence operations for the capabilities
// catalog (api-spec.md §2.1). The catalog is read-only via the API;
// writes are managed out-of-band (data-model.md §2 + the 016 seed).
type CapabilityStore interface {
	GetByName(ctx context.Context, name string) (*model.CapabilityRow, error)
	List(ctx context.Context, filter model.CapabilityFilter) (*model.CapabilityListResult, error)
	// Exists is the seam used by AgentService.Create / Update to
	// validate incoming capability names against the catalog. A
	// missing capability yields an ErrNotFound (the service maps
	// this to the api-spec's CAPABILITY_NOT_FOUND error).
	Exists(ctx context.Context, name string) (bool, error)
}

// ExecutionStore defines persistence operations for executions
// (api-spec.md §5). Executions are append-mostly: the only PATCH is
// the status transition (see UpdateStatus). There is no optimistic
// concurrency version column for Sprint 4 — the service layer is
// expected to serialise PATCH calls per execution (single-writer
// per row is fine for the in-flight "pending → running → terminal"
// path).
type ExecutionStore interface {
	// Create inserts a new execution row. The store does NOT
	// honour a caller-supplied ExecutionID; the id is generated
	// by the store (or the caller pre-populated it; the service
	// populates it before calling Create).
	Create(ctx context.Context, exec *model.Execution) error

	// GetByID returns the execution by primary key, or
	// ErrNotFound on miss.
	GetByID(ctx context.Context, id uuid.UUID) (*model.Execution, error)

	// List returns a keyset-paginated page of executions
	// matching the filter (TaskID, AgentID, Status, Cursor,
	// Limit). See model.ExecutionFilter for the cursor
	// semantics. The result wraps the page and a NextCursor
	// that the caller passes back to fetch the next page.
	List(ctx context.Context, filter model.ExecutionFilter) (*model.ExecutionListResult, error)

	// UpdateStatus is the only PATCH path. It transitions
	// execution `id` to `newStatus` (one of the four defined
	// statuses) and optionally sets ErrorMessage when newStatus
	// is 'failed'. The store updates UpdatedAt to now() and
	// sets CompletedAt to now() when newStatus is terminal
	// (completed/failed).
	//
	// The service layer is responsible for state-transition
	// validation; the store does NOT enforce transitions
	// itself (no SQL trigger, no constraint). On miss,
	// returns ErrNotFound.
	UpdateStatus(ctx context.Context, id uuid.UUID, newStatus model.ExecutionStatus, errorMessage *string) (*model.Execution, error)
}

// AgentRunStore defines persistence operations for agent runs.
type AgentRunStore interface {
	Create(run *model.AgentRun) error
	GetByID(id uuid.UUID) (*model.AgentRun, error)
	List(filter AgentRunFilter) ([]*model.AgentRun, int, error)
	Update(run *model.AgentRun) error
}

// AgentRunFilter holds optional query parameters for listing agent runs.
type AgentRunFilter struct {
	AgentID uuid.UUID
	TaskID  uuid.UUID
	Status  string
	Page    int
	Limit   int
}

// DeliverableStore defines persistence operations for deliverables.
//
// Two-table design (data-model.md §6): the `deliverables` table
// holds the *current* state of each deliverable; the
// `deliverable_versions` table holds the *immutable history*.
//
// The Update method on this store is IN-PLACE — it overwrites
// the deliverable's title/content/version/updated_at columns.
// The append-only invariant (one new row in
// deliverable_versions per PUT) is enforced by the service layer
// using WithTx to coordinate: a PUT does
//   (1) DeliverableStore.Update — update the main row
//   (2) DeliverableVersionStore.Insert — append a history row
// in a single transaction. The store does NOT do both internally
// (the brief: "either add InsertVersion to DeliverableStore or
// add a separate DeliverableVersionStore interface"; we chose
// the separate interface for cleaner SoC).
type DeliverableStore interface {
	// Create inserts a new deliverable. The store does NOT
	// honour a caller-supplied ID; the id is generated by the
	// store (the service pre-populates it before calling
	// Create).
	Create(ctx context.Context, d *model.Deliverable) error

	// GetByID returns the deliverable by primary key, or
	// ErrNotFound on miss.
	GetByID(ctx context.Context, id uuid.UUID) (*model.Deliverable, error)

	// List returns a keyset-paginated page of deliverables
	// matching the filter (TaskID, AgentID, Cursor, Limit).
	// See model.DeliverableFilter for cursor semantics.
	List(ctx context.Context, filter model.DeliverableFilter) (*model.DeliverableListResult, error)

	// Update applies a new state to an existing deliverable
	// (in-place on the main row). The service layer
	// coordinates with DeliverableVersionStore.Insert to
	// maintain the append-only history invariant.
	Update(ctx context.Context, d *model.Deliverable) error
}

// DeliverableVersionStore defines persistence operations for the
// `deliverable_versions` table (the immutable history of
// deliverable title/content changes). The service uses this
// interface in coordination with DeliverableStore.Update to
// enforce the append-only invariant: every PUT writes a new
// row here AND updates the main deliverable row in a single
// transaction (see WithTx in service/deliverable.go).
type DeliverableVersionStore interface {
	// Insert appends a new version row. The service
	// pre-computes the (deliverable_id, version) tuple; the
	// store does NOT increment version automatically. On a
	// duplicate (deliverable_id, version), returns
	// ErrAlreadyExists (mapped from pg 23505 in the postgres
	// impl).
	Insert(ctx context.Context, v *model.DeliverableVersion) error

	// ListVersions returns all versions for a deliverable,
	// ordered by version DESC. Returns an empty slice (not
	// an error) when the deliverable has no versions; the
	// service validates the deliverable exists and 404s on
	// miss.
	ListVersions(ctx context.Context, deliverableID uuid.UUID) ([]*model.DeliverableVersion, error)
}

// TaskStore defines persistence operations for tasks.
type TaskStore interface {
	Create(task *model.Task) error
	GetByID(id uuid.UUID) (*model.Task, error)
	List(filter TaskFilter) ([]*model.Task, int, error)
	Update(task *model.Task) error
	Delete(id uuid.UUID) error
}

// TaskFilter holds optional query parameters for listing tasks.
type TaskFilter struct {
	ProjectID  uuid.UUID
	Status     model.TaskStatus
	AssigneeID uuid.UUID
	Page       int
	Limit      int
}

// CodeStore defines persistence operations for code generation and commits.
type CodeStore interface {
	CreateCodeGen(req *model.CodeGenRequest) error
	GetCodeGenByID(id uuid.UUID) (*model.CodeGenRequest, error)
	ListCodeGenByProject(projectID uuid.UUID) ([]*model.CodeGenRequest, error)
	UpdateCodeGen(req *model.CodeGenRequest) error

	SaveFile(file *model.ProjectFile) error
	GetFile(projectID uuid.UUID, path string) (*model.ProjectFile, error)
	ListFiles(projectID uuid.UUID) ([]*model.ProjectFile, error)

	CreateCommit(commit *model.Commit) error
	GetCommit(projectID uuid.UUID, sha string) (*model.Commit, error)
	ListCommits(projectID uuid.UUID) ([]*model.Commit, error)
}

// ReviewStore defines persistence operations for reviews.
type ReviewStore interface {
	Create(review *model.Review) error
	GetByID(id uuid.UUID) (*model.Review, error)
	ListByProject(projectID uuid.UUID) ([]*model.Review, error)
	Update(review *model.Review) error

	CreateComment(comment *model.ReviewComment) error
	ListComments(reviewID uuid.UUID) ([]*model.ReviewComment, error)
}

// DeploymentStore defines persistence operations for deployments.
type DeploymentStore interface {
	Create(deployment *model.Deployment) error
	GetByID(id uuid.UUID) (*model.Deployment, error)
	ListByProject(projectID uuid.UUID) ([]*model.Deployment, error)
	Update(deployment *model.Deployment) error
}

// WebhookStore defines persistence operations for webhooks.
type WebhookStore interface {
	Create(webhook *model.Webhook) error
	GetByID(id uuid.UUID) (*model.Webhook, error)
	List() ([]*model.Webhook, error)
	Update(webhook *model.Webhook) error
	Delete(id uuid.UUID) error
}

// AuditLogStore defines persistence operations for audit logs.
type AuditLogStore interface {
	Create(log *model.AuditLog) error
	List(filter AuditLogFilter) ([]*model.AuditLog, int, error)
}

// AuditLogFilter holds optional query parameters for listing audit logs.
type AuditLogFilter struct {
	EntityType string
	EntityID   uuid.UUID
	UserID     uuid.UUID
	Page       int
	Limit      int
}

// TokenStore defines operations for managing short-lived tokens (e.g. refresh tokens).
type TokenStore interface {
	Set(key string, userID uuid.UUID, ttl int) error
	Get(key string) (uuid.UUID, error)
	Delete(key string) error
}

// Pagination holds pagination metadata for list responses.
type Pagination struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
	Pages int `json:"pages"`
}

// Store combines all individual stores.
type Store interface {
	Users() UserStore
	Projects() ProjectStore
	Agents() AgentStore
	Capabilities() CapabilityStore
	AgentRuns() AgentRunStore
	Executions() ExecutionStore
	Deliverables() DeliverableStore
	// DeliverableVersions is the TASK-406 append-only history
	// store. Wired by every concrete Store implementation
	// (memory + postgres).
	DeliverableVersions() DeliverableVersionStore
	Tasks() TaskStore
	Code() CodeStore
	Reviews() ReviewStore
	Deployments() DeploymentStore
	Webhooks() WebhookStore
	AuditLogs() AuditLogStore
	Tokens() TokenStore
	// AssignmentEvents is the TASK-404 append-only history store.
	// Wired by every concrete Store implementation (memory + postgres).
	AssignmentEvents() AssignmentEventStore
	// Assignments is the TASK-404 current-state store.
	Assignments() AssignmentStore
	// WithTx opens a SQL transaction and runs the closure with a
	// Tx view of the store. The closure MUST be the only code
	// path that calls Assignments/AssignmentEvents on the tx-scoped
	// sub-stores. If the closure returns an error, the transaction
	// is rolled back. If it returns nil, the transaction is
	// committed. For the in-memory store, WithTx is a no-op (the
	// mutex already serialises everything).
	WithTx(ctx context.Context, fn func(Tx) error) error
}

// AssignmentEventStore is the append-only history of task
// assignments (migration 020, TASK-404). There is no Update or
// Delete — assignment_events is append-only by contract. The TASK-404
// endpoint POST /v1/tasks/:id/assign calls Append; GET
// /v1/tasks/:id/history calls ListByTask.
type AssignmentEventStore interface {
	// Append writes a new event row. The ID, AssignedAt, and
	// AssignedBy fields are honoured from the input (AssignedAt
	// defaults to time.Now().UTC() if zero). Returns the persisted
	// row (with server-populated defaults applied).
	Append(ctx context.Context, ev *model.AssignmentEvent) (*model.AssignmentEvent, error)

	// ListByTask returns all events for a task, newest first
	// (assigned_at DESC). Returns an empty slice (not an error) if
	// the task has no events. Does NOT check the task's existence —
	// the service layer does that so it can return 404.
	ListByTask(ctx context.Context, taskID uuid.UUID) ([]*model.AssignmentEvent, error)
}

// AssignmentStore is the current-state table for task assignments
// (migration 019, TASK-404). At most one row per task may have
// status='active' (enforced by the partial unique index in 019).
// The TASK-404 endpoint POST /v1/tasks/:id/assign writes to this
// table inside a transaction (see WithTx below), flipping the
// previous active row to 'superseded' and inserting the new active
// row.
type AssignmentStore interface {
	// Create inserts a new assignment row. The status is
	// expected to be 'active' on this code path (the service
	// guards this). The store does NOT auto-set completed_at
	// for the active row — that field stays NULL until the row
	// is flipped to superseded/completed/cancelled.
	Create(ctx context.Context, a *model.Assignment) (*model.Assignment, error)

	// Update mutates an existing row. Used by the service to
	// flip a previous active row to 'superseded' (and set
	// completed_at = now) inside the same transaction as the
	// Create of the new active row.
	Update(ctx context.Context, a *model.Assignment) error

	// GetByID returns the assignment by primary key, or
	// ErrNotFound on miss.
	GetByID(ctx context.Context, id uuid.UUID) (*model.Assignment, error)

	// GetActiveByTask returns the current active assignment for
	// a task (the row with status='active'), or ErrNotFound if
	// no active row exists. Used by AssignTaskToAgent to find
	// the row that needs to be flipped to 'superseded' during
	// a reassignment.
	GetActiveByTask(ctx context.Context, taskID uuid.UUID) (*model.Assignment, error)
}

// Tx is the transactional view of a Store. It exposes the sub-stores
// needed by a transaction and is only valid for the duration of
// the WithTx closure.
//
// The full Store interface is intentionally NOT exposed — only the
// sub-stores that need to be in the same SQL transaction. Other
// sub-stores (Tasks, Agents) can still be accessed via the
// surrounding Store but will be in their own implicit transactions;
// this is fine because they are read-only in the flows that use
// WithTx (Tasks.GetByID/Agents.GetByID are single-row reads outside
// the tx).
//
// TASK-404 wires Assignments + AssignmentEvents (assignment
// current-state + history).
// TASK-406 wires Deliverables + DeliverableVersions (deliverable
// current-state + history).
type Tx interface {
	Assignments() AssignmentStore
	AssignmentEvents() AssignmentEventStore
	Deliverables() DeliverableStore
	DeliverableVersions() DeliverableVersionStore
}
