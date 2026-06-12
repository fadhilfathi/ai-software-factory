package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AgentType defines the role of an agent within a project.
type AgentType string

const (
	AgentPM       AgentType = "pm"
	AgentDev      AgentType = "developer"
	AgentReviewer AgentType = "reviewer"
	AgentDevOps   AgentType = "devops"
)

// AgentStatus represents the current state of an agent.
type AgentStatus string

const (
	AgentSpawning  AgentStatus = "spawning"
	AgentIdle      AgentStatus = "idle"
	AgentWorking   AgentStatus = "working"
	AgentCompleted AgentStatus = "completed"
	AgentFailed    AgentStatus = "failed"
)

// Agent represents an AI agent operating within a project.
type Agent struct {
	ID           uuid.UUID   `json:"id"`
	Type         AgentType   `json:"type"`
	Status       AgentStatus `json:"status"`
	ProjectID    uuid.UUID   `json:"project_id,omitempty"`
	Config       json.RawMessage `json:"config,omitempty"`
	CurrentTaskID  uuid.UUID   `json:"current_task_id,omitempty"`
	TasksDone    int         `json:"tasks_completed,omitempty"`
	Uptime       int         `json:"uptime,omitempty"` // seconds
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// Assignment represents a task assigned to an agent.
type Assignment struct {
	ID                  uuid.UUID `json:"id"`
	AgentID             uuid.UUID `json:"agent_id"`
	TaskID              uuid.UUID `json:"task_id"`
	Status              string    `json:"status"`
	EstimatedCompletion time.Time `json:"estimated_completion,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
}
