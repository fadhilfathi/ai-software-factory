package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAgentTypeConstants(t *testing.T) {
	assert.Equal(t, AgentType("pm"), AgentPM)
	assert.Equal(t, AgentType("developer"), AgentDev)
	assert.Equal(t, AgentType("reviewer"), AgentReviewer)
	assert.Equal(t, AgentType("devops"), AgentDevOps)
}

func TestAgentStatusConstants(t *testing.T) {
	assert.Equal(t, AgentStatus("spawning"), AgentSpawning)
	assert.Equal(t, AgentStatus("idle"), AgentIdle)
	assert.Equal(t, AgentStatus("working"), AgentWorking)
	assert.Equal(t, AgentStatus("completed"), AgentCompleted)
	assert.Equal(t, AgentStatus("failed"), AgentFailed)
}

func TestAgentConfigStruct(t *testing.T) {
	config := AgentConfig{
		Model:       "gpt-4",
		Temperature: 0.7,
	}

	assert.Equal(t, "gpt-4", config.Model)
	assert.Equal(t, 0.7, config.Temperature)
}

func TestAgentConfigEmptyValues(t *testing.T) {
	config := AgentConfig{}
	assert.Equal(t, "", config.Model)
	assert.Equal(t, 0.0, config.Temperature)
}

func TestAgentStructFields(t *testing.T) {
	now := time.Now().UTC()
	config := &AgentConfig{Model: "claude-3", Temperature: 0.5}
	agent := Agent{
		ID:          "agent-123",
		Type:        AgentDev,
		Status:      AgentWorking,
		ProjectID:   "proj-456",
		Config:      config,
		CurrentTaskID: "task-789",
		TasksDone:   5,
		Uptime:      3600,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, "agent-123", agent.ID)
	assert.Equal(t, AgentDev, agent.Type)
	assert.Equal(t, AgentWorking, agent.Status)
	assert.Equal(t, "proj-456", agent.ProjectID)
	assert.Equal(t, config, agent.Config)
	assert.Equal(t, "task-789", agent.CurrentTaskID)
	assert.Equal(t, 5, agent.TasksDone)
	assert.Equal(t, 3600, agent.Uptime)
	assert.Equal(t, now, agent.CreatedAt)
	assert.Equal(t, now, agent.UpdatedAt)
}

func TestAgentWithNilConfig(t *testing.T) {
	agent := Agent{
		ID:     "agent-no-config",
		Type:   AgentPM,
		Status: AgentIdle,
	}
	assert.Nil(t, agent.Config)
}

func TestAssignmentStruct(t *testing.T) {
	now := time.Now().UTC()
	assignment := Assignment{
		ID:                  "assign-123",
		AgentID:             "agent-1",
		TaskID:              "task-1",
		Status:              "assigned",
		EstimatedCompletion: now.Add(time.Hour),
		CreatedAt:           now,
	}

	assert.Equal(t, "assign-123", assignment.ID)
	assert.Equal(t, "agent-1", assignment.AgentID)
	assert.Equal(t, "task-1", assignment.TaskID)
	assert.Equal(t, "assigned", assignment.Status)
	assert.Equal(t, now.Add(time.Hour), assignment.EstimatedCompletion)
	assert.Equal(t, now, assignment.CreatedAt)
}