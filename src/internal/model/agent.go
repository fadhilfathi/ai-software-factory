package model

import "time"

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

// AgentConfig holds runtime configuration for an agent.
type AgentConfig struct {
	Model       string  `json:"model,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

// Agent represents an AI agent operating within a project.
type Agent struct {
	ID           string      `json:"id"`
	Type         AgentType   `json:"type"`
	Status       AgentStatus `json:"status"`
	ProjectID    string      `json:"project_id,omitempty"`
	Config       *AgentConfig `json:"config,omitempty"`
	CurrentTask  string      `json:"current_task,omitempty"`
	TasksDone    int         `json:"tasks_completed,omitempty"`
	Uptime       int         `json:"uptime,omitempty"` // seconds
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// Assignment represents a task assigned to an agent.
type Assignment struct {
	ID                  string    `json:"id"`
	AgentID             string    `json:"agent_id"`
	TaskID              string    `json:"task_id"`
	Status              string    `json:"status"`
	EstimatedCompletion time.Time `json:"estimated_completion,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
}
