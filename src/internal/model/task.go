package model

import (
	"time"

	"github.com/google/uuid"
)

// TaskPriority represents the urgency level of a task.
type TaskPriority string


// TaskStatus represents the lifecycle state of a task (Kanban column).
type TaskStatus string

const (
	TaskBacklog    TaskStatus = "backlog"
	TaskTodo       TaskStatus = "todo"
	TaskReady      TaskStatus = "ready"
	TaskInProgress TaskStatus = "in_progress"
	TaskReview     TaskStatus = "review"
	TaskDone       TaskStatus = "done"
	TaskBlocked    TaskStatus = "blocked"
	TaskOpen       TaskStatus = "open"
)

const (
	PriorityLow      TaskPriority = "low"
	PriorityMedium   TaskPriority = "medium"
	PriorityHigh     TaskPriority = "high"
	PriorityCritical TaskPriority = "critical"
	PriorityNormal   TaskPriority = "medium" // alias for PriorityMedium (legacy code)
)
type Task struct {
	ID              uuid.UUID         `json:"id"`
	ProjectID       uuid.UUID         `json:"project_id"`
	Title           string            `json:"title"`
	Description     string            `json:"description,omitempty"`
	Status          TaskStatus        `json:"status"`
	Priority        TaskPriority      `json:"priority"`
	AssigneeID      uuid.UUID         `json:"assignee_id,omitempty"`
	AssigneeAgentID string            `json:"assignee_agent_id,omitempty"`
	// RequiredCapabilities is the set of capabilities an agent must hold to be
	// assigned to this task. Populated by the POST /v1/tasks/:id/assign endpoint
	// (TASK-404) and enforced by CapabilityService.ValidateAgentHasCapabilities
	// (TASK-403). An empty/nil slice means "no capability constraint" — every
	// agent is eligible. Persisted as a JSONB column (migration 018) so the
	// shape can grow later (e.g. {name, min_proficiency}) without another
	// column-add migration.
	RequiredCapabilities []string          `json:"required_capabilities,omitempty"`
	Position             int               `json:"position,omitempty"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
}
