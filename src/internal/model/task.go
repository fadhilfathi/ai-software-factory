package model

import "time"

// TaskPriority represents the urgency level of a task.
type TaskPriority string

const (
	PriorityLow     TaskPriority = "low"
	PriorityMedium  TaskPriority = "medium"
	PriorityHigh    TaskPriority = "high"
	PriorityCritical TaskPriority = "critical"
)

// TaskStatus represents the lifecycle state of a task.
type TaskStatus string

const (
	TaskBacklog TaskStatus = "backlog"
	TaskTodo    TaskStatus = "todo"
	TaskInProgress TaskStatus = "in_progress"
	TaskReview  TaskStatus = "review"
	TaskDone    TaskStatus = "done"
)

// Task represents a unit of work within a project.
type Task struct {
	ID                 string       `json:"id"`
	ProjectID          string       `json:"project_id"`
	Title              string       `json:"title"`
	Description        string       `json:"description,omitempty"`
	Type               string       `json:"type,omitempty"`
	AcceptanceCriteria []string     `json:"acceptance_criteria,omitempty"`
	Priority           TaskPriority `json:"priority"`
	Status             TaskStatus   `json:"status"`
	EstimatedHours     int          `json:"estimated_hours,omitempty"`
	AssigneeAgentID    string       `json:"assignee_agent_id,omitempty"`
	CreatedAt          time.Time    `json:"created_at"`
	UpdatedAt          time.Time    `json:"updated_at"`
}
