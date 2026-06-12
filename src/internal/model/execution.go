package model

import (
	"time"

	"github.com/google/uuid"
)

// ExecutionStatus represents the state of an execution record.
type ExecutionStatus string

const (
	ExecPending   ExecutionStatus = "pending"
	ExecRunning   ExecutionStatus = "running"
	ExecCompleted ExecutionStatus = "completed"
	ExecFailed    ExecutionStatus = "failed"
)

// Execution records a single execution of an agent on a task.
type Execution struct {
	ExecutionID uuid.UUID      `json:"execution_id"`
	TaskID      uuid.UUID      `json:"task_id"`
	AgentID     uuid.UUID      `json:"agent_id"`
	Status      ExecutionStatus `json:"status"`
	StartedAt   *time.Time     `json:"started_at,omitempty"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}
