package model

import (
	"time"

	"github.com/google/uuid"
)

// TaskPriority represents the urgency level of a task.
type TaskPriority string

const (
	PriorityLow      TaskPriority = "low"
	PriorityMedium   TaskPriority = "medium"
	PriorityHigh     TaskPriority = "high"
	PriorityCritical TaskPriority = "critical"
)

// TaskStatus represents the lifecycle state of a task (Kanban column).
type TaskStatus string

const (
	TaskBacklog    TaskStatus = "backlog"
	TaskReady      TaskStatus = "ready"
	TaskInProgress TaskStatus = "in_progress"
	TaskReview     TaskStatus = "review"
	TaskDone       TaskStatus = "done"
	TaskBlocked    TaskStatus = "blocked"
)

// Task represents a unit of work within a project.
type Task struct {
	ID              uuid.UUID    `json:"id"`
	ProjectID       uuid.UUID    `json:"project_id"`
	Title           string       `json:"title"`
	Description     string       `json:"description,omitempty"`
	Status          TaskStatus   `json:"status"`
	Priority        TaskPriority `json:"priority"`
	AssigneeID      uuid.UUID    `json:"assignee_id,omitempty"`
	AssigneeAgentID string       `json:"assignee_agent_id,omitempty"`
	Position        int          `json:"position,omitempty"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}
