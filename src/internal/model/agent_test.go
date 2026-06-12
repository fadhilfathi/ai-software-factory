package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAgentTypeConstants(t *testing.T) {
	assert.Equal(t, AgentType("pm"), AgentPM)
	assert.Equal(t, AgentType("architect"), AgentArch)
	assert.Equal(t, AgentType("developer"), AgentDev)
	assert.Equal(t, AgentType("reviewer"), AgentReviewer)
	assert.Equal(t, AgentType("qa"), AgentQA)
	assert.Equal(t, AgentType("devops"), AgentDevOps)
}

func TestAgentStatusConstants(t *testing.T) {
	assert.Equal(t, AgentStatus("spawning"), AgentSpawning)
	assert.Equal(t, AgentStatus("idle"), AgentIdle)
	assert.Equal(t, AgentStatus("working"), AgentWorking)
	assert.Equal(t, AgentStatus("completed"), AgentCompleted)
	assert.Equal(t, AgentStatus("failed"), AgentFailed)
}

func TestAgentStructFields(t *testing.T) {
	now := time.Now().UTC()
	agent := Agent{
		ID:            uuid.New(),
		Name:          "Test Agent",
		Type:          "developer",
		Role:          "developer",
		Model:         "gpt-4",
		Provider:      "openai",
		Capabilities:  []string{"code_implementation"},
		Status:        AgentWorking,
		ProjectID:     "proj-123",
		Config:        json.RawMessage(`{"temperature": 0.7}`),
		CurrentTaskID: "task-456",
		TasksDone:     5,
		Uptime:        3600,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	assert.Equal(t, "Test Agent", agent.Name)
	assert.Equal(t, "developer", agent.Type)
	assert.Equal(t, "developer", agent.Role)
	assert.Equal(t, "gpt-4", agent.Model)
	assert.Equal(t, "openai", agent.Provider)
	assert.Equal(t, []string{"code_implementation"}, agent.Capabilities)
	assert.Equal(t, AgentWorking, agent.Status)
	assert.Equal(t, "proj-123", agent.ProjectID)
	assert.Equal(t, "task-456", agent.CurrentTaskID)
	assert.Equal(t, 5, agent.TasksDone)
	assert.Equal(t, 3600, agent.Uptime)
	assert.NotNil(t, agent.Config)
	assert.Equal(t, now, agent.CreatedAt)
	assert.Equal(t, now, agent.UpdatedAt)
}

func TestAgentDefaultValues(t *testing.T) {
	agent := Agent{
		Name: "Default Agent",
		Type: "pm",
	}

	assert.Equal(t, "", agent.Model)
	assert.Nil(t, agent.Config)
	assert.Equal(t, AgentStatus(""), agent.Status)
	assert.Empty(t, agent.ProjectID)
	assert.Empty(t, agent.CurrentTaskID)
}

func TestAgentTypeCapabilities(t *testing.T) {
	assert.Equal(t, []AgentCapability{CapRequirementAnalysis, CapTaskDecomposition}, AgentTypeCapabilities[AgentPM])
	assert.Equal(t, []AgentCapability{CapSystemDesign, CapAPIDesign}, AgentTypeCapabilities[AgentArch])
	assert.Equal(t, []AgentCapability{CapCodeImplementation}, AgentTypeCapabilities[AgentDev])
	assert.Equal(t, []AgentCapability{CapCodeReview, CapSecurityScan}, AgentTypeCapabilities[AgentReviewer])
	assert.Equal(t, []AgentCapability{CapTestPlanning, CapTestExecution}, AgentTypeCapabilities[AgentQA])
	assert.Equal(t, []AgentCapability{CapCICD, CapDeployment, CapInfrastructure}, AgentTypeCapabilities[AgentDevOps])
}

func TestDefaultCapabilitiesForType(t *testing.T) {
	assert.Equal(t, []string{"requirement_analysis", "task_decomposition"}, DefaultCapabilitiesForType("pm"))
	assert.Equal(t, []string{"system_design", "api_design"}, DefaultCapabilitiesForType("architect"))
	assert.Equal(t, []string{"code_implementation"}, DefaultCapabilitiesForType("developer"))
	assert.Equal(t, []string{"code_review", "security_scan"}, DefaultCapabilitiesForType("reviewer"))
	assert.Equal(t, []string{"test_planning", "test_execution"}, DefaultCapabilitiesForType("qa"))
	assert.Equal(t, []string{"ci_cd", "deployment", "infrastructure"}, DefaultCapabilitiesForType("devops"))
	assert.Nil(t, DefaultCapabilitiesForType("unknown"))
}

func TestAgentRunStruct(t *testing.T) {
	now := time.Now().UTC()
	started := now
	completed := now.Add(30 * time.Second)
	run := AgentRun{
		ID:          uuid.New(),
		AgentID:     uuid.New(),
		TaskID:      uuid.New(),
		Status:      RunCompleted,
		Input:       "Generate login API",
		Output:      "Created login.go, login_test.go",
		StartedAt:   &started,
		CompletedAt: &completed,
		CreatedAt:   now,
	}

	assert.Equal(t, RunCompleted, run.Status)
	assert.Equal(t, "Generate login API", run.Input)
	assert.Equal(t, "Created login.go, login_test.go", run.Output)
	assert.NotNil(t, run.StartedAt)
	assert.NotNil(t, run.CompletedAt)
}

func TestAgentRunStatusConstants(t *testing.T) {
	assert.Equal(t, AgentRunStatus("pending"), RunPending)
	assert.Equal(t, AgentRunStatus("running"), RunRunning)
	assert.Equal(t, AgentRunStatus("completed"), RunCompleted)
	assert.Equal(t, AgentRunStatus("failed"), RunFailed)
	assert.Equal(t, AgentRunStatus("cancelled"), RunCancelled)
}

func TestAssignmentStruct(t *testing.T) {
	now := time.Now().UTC()
	assignment := Assignment{
		ID:                  uuid.New(),
		AgentID:             uuid.New(),
		TaskID:              uuid.New(),
		Status:              "assigned",
		EstimatedCompletion: now.Add(time.Hour),
		CreatedAt:           now,
	}
	assert.Equal(t, "assigned", assignment.Status)
	assert.Equal(t, now.Add(time.Hour), assignment.EstimatedCompletion)
}

func TestAgentCapabilityConstants(t *testing.T) {
	assert.Equal(t, AgentCapability("requirement_analysis"), CapRequirementAnalysis)
	assert.Equal(t, AgentCapability("task_decomposition"), CapTaskDecomposition)
	assert.Equal(t, AgentCapability("system_design"), CapSystemDesign)
	assert.Equal(t, AgentCapability("api_design"), CapAPIDesign)
	assert.Equal(t, AgentCapability("code_implementation"), CapCodeImplementation)
	assert.Equal(t, AgentCapability("code_review"), CapCodeReview)
	assert.Equal(t, AgentCapability("security_scan"), CapSecurityScan)
	assert.Equal(t, AgentCapability("test_planning"), CapTestPlanning)
	assert.Equal(t, AgentCapability("test_execution"), CapTestExecution)
	assert.Equal(t, AgentCapability("ci_cd"), CapCICD)
	assert.Equal(t, AgentCapability("deployment"), CapDeployment)
	assert.Equal(t, AgentCapability("infrastructure"), CapInfrastructure)
}

func TestAgentConfigStruct(t *testing.T) {
	config := AgentConfig{
		Model:       "gpt-4",
		Temperature: 0.7,
		Provider:    "openai",
	}
	assert.Equal(t, "gpt-4", config.Model)
	assert.Equal(t, 0.7, config.Temperature)
	assert.Equal(t, "openai", config.Provider)
}
