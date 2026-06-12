package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectStatusConstants(t *testing.T) {
	assert.Equal(t, ProjectStatus("initializing"), ProjectInitializing)
	assert.Equal(t, ProjectStatus("in_progress"), ProjectInProgress)
	assert.Equal(t, ProjectStatus("completed"), ProjectCompleted)
	assert.Equal(t, ProjectStatus("archived"), ProjectArchived)
}

func TestProjectStructFields(t *testing.T) {
	now := time.Now().UTC()
	project := Project{
		ID:            "proj-123",
		Name:          "Test Project",
		Description:   "A test project",
		Status:        ProjectInProgress,
		Template:      "go-api",
		Progress:      50,
		ActiveAgents:  2,
		AgentsSpawned: []string{"agent-1", "agent-2"},
		Artifacts:     []interface{}{"artifact1"},
		Agents:        []interface{}{"agent1"},
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	assert.Equal(t, "proj-123", project.ID)
	assert.Equal(t, "Test Project", project.Name)
	assert.Equal(t, "A test project", project.Description)
	assert.Equal(t, ProjectInProgress, project.Status)
	assert.Equal(t, "go-api", project.Template)
	assert.Equal(t, 50, project.Progress)
	assert.Equal(t, 2, project.ActiveAgents)
	assert.Equal(t, []string{"agent-1", "agent-2"}, project.AgentsSpawned)
	assert.Equal(t, now, project.CreatedAt)
	assert.Equal(t, now, project.UpdatedAt)
}

func TestProjectJSONSerialization(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	project := Project{
		ID:          "proj-456",
		Name:        "JSON Test",
		Description: "Testing JSON",
		Status:      ProjectCompleted,
		Template:    "react-app",
		Progress:    100,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Test that the struct can be used with JSON marshaling
	// (actual marshaling tested in handler tests)
	assert.NotEmpty(t, project.ID)
	assert.Equal(t, ProjectCompleted, project.Status)
}

func TestProjectStatusStringValues(t *testing.T) {
	tests := []struct {
		status     ProjectStatus
		expected   string
	}{
		{ProjectInitializing, "initializing"},
		{ProjectInProgress, "in_progress"},
		{ProjectCompleted, "completed"},
		{ProjectArchived, "archived"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, string(tc.status))
		})
	}
}