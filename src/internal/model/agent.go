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
	AgentArch     AgentType = "architect"
	AgentDev      AgentType = "developer"
	AgentReviewer AgentType = "reviewer"
	AgentQA       AgentType = "qa"
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

// AgentCapability describes a specific ability an agent type can perform.
type AgentCapability string

const (
	CapRequirementAnalysis AgentCapability = "requirement_analysis"
	CapTaskDecomposition   AgentCapability = "task_decomposition"
	CapSystemDesign        AgentCapability = "system_design"
	CapAPIDesign           AgentCapability = "api_design"
	CapCodeImplementation  AgentCapability = "code_implementation"
	CapCodeReview          AgentCapability = "code_review"
	CapSecurityScan        AgentCapability = "security_scan"
	CapTestPlanning        AgentCapability = "test_planning"
	CapTestExecution       AgentCapability = "test_execution"
	CapCICD                AgentCapability = "ci_cd"
	CapDeployment          AgentCapability = "deployment"
	CapInfrastructure      AgentCapability = "infrastructure"
)

// AgentTypeCapabilities maps each agent type to its built-in capabilities.
var AgentTypeCapabilities = map[AgentType][]AgentCapability{
	AgentPM:       {CapRequirementAnalysis, CapTaskDecomposition},
	AgentArch:     {CapSystemDesign, CapAPIDesign},
	AgentDev:      {CapCodeImplementation},
	AgentReviewer: {CapCodeReview, CapSecurityScan},
	AgentQA:       {CapTestPlanning, CapTestExecution},
	AgentDevOps:   {CapCICD, CapDeployment, CapInfrastructure},
}

// AgentConfig holds runtime configuration for an agent instance.
type AgentConfig struct {
	Model       string  `json:"model,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	Provider    string  `json:"provider,omitempty"`
}

// DefaultCapabilitiesForType returns the default capability set for an agent type.
func DefaultCapabilitiesForType(t string) []string {
	caps, ok := AgentTypeCapabilities[AgentType(t)]
	if !ok {
		return nil
	}
	strCaps := make([]string, len(caps))
	for i, c := range caps {
		strCaps[i] = string(c)
	}
	return strCaps
}

// Agent represents an AI agent operating within a project.
type Agent struct {
	ID            uuid.UUID        `json:"id"`
	Name          string           `json:"name"`
	Type          string           `json:"type"`
	Role          string           `json:"role"`
	Model         string           `json:"model"`
	Provider      string           `json:"provider"`
	Capabilities  []string         `json:"capabilities"`
	Status        AgentStatus      `json:"status"`
	ProjectID     string           `json:"project_id,omitempty"`
	Config        json.RawMessage  `json:"config,omitempty"`
	CurrentTaskID string           `json:"current_task_id,omitempty"`
	TasksDone     int              `json:"tasks_completed,omitempty"`
	Uptime        int              `json:"uptime,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

// AgentRunStatus represents the state of a single agent execution.
type AgentRunStatus string

const (
	RunPending   AgentRunStatus = "pending"
	RunRunning   AgentRunStatus = "running"
	RunCompleted AgentRunStatus = "completed"
	RunFailed    AgentRunStatus = "failed"
	RunCancelled AgentRunStatus = "cancelled"
)

// AgentRun records a single execution of an agent on a task.
type AgentRun struct {
	ID          uuid.UUID      `json:"id"`
	AgentID     uuid.UUID      `json:"agent_id"`
	TaskID      uuid.UUID      `json:"task_id,omitempty"`
	Status      AgentRunStatus `json:"status"`
	Input       string         `json:"input,omitempty"`
	Output      string         `json:"output,omitempty"`
	Error       string         `json:"error,omitempty"`
	StartedAt   *time.Time     `json:"started_at,omitempty"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
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
