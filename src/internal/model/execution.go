package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ExecutionStatus is the lifecycle state of a single execution attempt.
// Allowed transitions are enforced in the service layer:
//
//	pending   → running, completed, failed
//	running   → completed, failed
//	completed → (terminal)
//	failed    → (terminal)
//
// The corresponding TEXT CHECK constraint lives in
// 008_create_executions.sql (the original table) and is reused by
// 024_create_executions.sql — no enum is added in Sprint 4.
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
)

// AllExecutionStatuses returns every defined status value. Useful for
// validation, swagger generation, and the in-flight index.
func AllExecutionStatuses() []ExecutionStatus {
	return []ExecutionStatus{
		ExecutionStatusPending,
		ExecutionStatusRunning,
		ExecutionStatusCompleted,
		ExecutionStatusFailed,
	}
}

// IsValidExecutionStatus returns true if s is one of the four defined
// statuses. Empty string and unknown values return false.
func IsValidExecutionStatus(s ExecutionStatus) bool {
	switch s {
	case ExecutionStatusPending, ExecutionStatusRunning,
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
// ExecutionStatusFailed. It is nil for pending/running/completed rows.
type Execution struct {
	ExecutionID  uuid.UUID
	TaskID       uuid.UUID
	AgentID      uuid.UUID
	Status       ExecutionStatus
	StartedAt    time.Time
	CompletedAt  *time.Time
	ErrorMessage *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
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
