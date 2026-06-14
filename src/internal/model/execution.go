package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ExecutionStatus is the lifecycle state of a single execution attempt.
// Allowed transitions are enforced in the service layer:
//
//	queued    → assigned, failed
//	assigned  → running, failed, queued (operator/recovery return)
//	running   → review, failed
//	review    → completed, failed  (the only path into completed)
//	completed → (terminal)
//	failed    → (terminal)
//
// The corresponding TEXT CHECK constraint lives in
// 008_create_executions.sql (the original table), 024_create_executions.sql
// (TASK-501 re-creation), and 028_extend_executions_lifecycle.sql (B-001
// extension from 4 to 6 states: added queued, review; renamed pending to
// assigned; removed the direct pending → completed edge in favour of
// the review state).
type ExecutionStatus string

const (
	ExecutionStatusQueued    ExecutionStatus = "queued"
	ExecutionStatusAssigned  ExecutionStatus = "assigned"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusReview    ExecutionStatus = "review"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
)

// AllExecutionStatuses returns every defined status value. Useful for
// validation, swagger generation, and the in-flight index.
func AllExecutionStatuses() []ExecutionStatus {
	return []ExecutionStatus{
		ExecutionStatusQueued,
		ExecutionStatusAssigned,
		ExecutionStatusRunning,
		ExecutionStatusReview,
		ExecutionStatusCompleted,
		ExecutionStatusFailed,
	}
}

// IsValidExecutionStatus returns true if s is one of the six defined
// statuses. Empty string and unknown values return false.
func IsValidExecutionStatus(s ExecutionStatus) bool {
	switch s {
	case ExecutionStatusQueued, ExecutionStatusAssigned,
		ExecutionStatusRunning, ExecutionStatusReview,
		ExecutionStatusCompleted, ExecutionStatusFailed:
		return true
	default:
		return false
	}
}

// ErrInvalidExecutionStatus is the typed sentinel returned by
// service-level status validation. The handler layer maps this to
// 400 INVALID_EXECUTION_STATUS.
var ErrInvalidExecutionStatus = errors.New("invalid execution status")

// Execution is a single attempt by an agent to run a task. The
// in-memory and postgres stores persist exactly these fields. The
// `ExecutionID` field name (rather than plain `ID`) is preserved from
// the original Sprint 1/2 model to avoid a wider refactor; callers
// that need an opaque identifier should treat it as the primary key.
//
// ErrorMessage is populated only when Status transitions to
// ExecutionStatusFailed. It is nil for queued/assigned/running/review/
// completed rows.
//
// AionAgentInstanceID is the optional Aion agent-process instance
// identifier populated by TASK-501's aion.Runtime.Spawn path. It is
// nil for executions created via the legacy mock-goroutine path
// (Sprint 4 default) and non-nil for executions spawned by the
// aion.Runtime (Sprint 5 default). Callers can use it to correlate
// the execution row with the child process described by the
// corresponding model.Worker (Worker.PID is derived from this for
// process-mode runtimes).
type Execution struct {
	ExecutionID         uuid.UUID
	TaskID              uuid.UUID
	AgentID             uuid.UUID // required; B-001 keeps the Sprint 5 contract that CreateExecution always carries an agent
	Status              ExecutionStatus
	StartedAt           time.Time
	CompletedAt         *time.Time
	ErrorMessage        *string
	AionAgentInstanceID *uuid.UUID // TASK-501; nil for legacy paths
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// ExecutionFilter is the input to ListExecutions. The zero value is a
// "first page, no filter" request: returns the most recent executions
// across all tasks/agents, default page size 50, max 200.
//
// Cursor-based keyset pagination: when a previous response returns a
// NextCursor, the caller passes that cursor back as Cursor to fetch
// the next page. The cursor is the ExecutionID of the last row in the
// previous page; the next page contains rows whose ExecutionID <
// cursor (UUIDs are 16 bytes, lexicographic order is well-defined),
// ordered by (started_at DESC, execution_id DESC) for stability.
type ExecutionFilter struct {
	TaskID  uuid.UUID
	AgentID uuid.UUID
	Status  ExecutionStatus
	Cursor  uuid.UUID // empty = no cursor (first page)
	Limit   int       // 0 = use default (50); clamped to [1, 200]
}

// ExecutionListResult is the output of ListExecutions. NextCursor is
// empty when there are no more pages; the caller can use it as a
// signal to stop paginating.
type ExecutionListResult struct {
	Items      []*Execution
	NextCursor uuid.UUID
}

// DefaultExecutionLimit and MaxExecutionLimit bound the per-page size.
// We pick 50 as the default (a page of dashboard rows) and 200 as the
// hard cap (a single API call should never return more).
const (
	DefaultExecutionLimit = 50
	MaxExecutionLimit     = 200
)
