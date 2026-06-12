package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestProjectStatusConstants(t *testing.T) {
	assert.Equal(t, ProjectStatus("initializing"), ProjectInitializing)
	assert.Equal(t, ProjectStatus("in_progress"), ProjectInProgress)
	assert.Equal(t, ProjectStatus("completed"), ProjectCompleted)
	assert.Equal(t, ProjectStatus("archived"), ProjectArchived)
}

func TestProjectStructFields(t *testing.T) {
	now := time.Now().UTC()
	id := uuid.New()
	ownerID := uuid.New()
	project := Project{
		ID:          id,
		Name:        "Test Project",
		Description: "A test project",
		OwnerID:     ownerID,
		Status:      ProjectInProgress,
		Template:    "go-api",
		Progress:    50,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, id, project.ID)
	assert.Equal(t, "Test Project", project.Name)
	assert.Equal(t, "A test project", project.Description)
	assert.Equal(t, ownerID, project.OwnerID)
	assert.Equal(t, ProjectInProgress, project.Status)
	assert.Equal(t, "go-api", project.Template)
	assert.Equal(t, 50, project.Progress)
	assert.Equal(t, now, project.CreatedAt)
	assert.Equal(t, now, project.UpdatedAt)
}

func TestProjectJSONSerialization(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	id := uuid.New()
	ownerID := uuid.New()
	project := Project{
		ID:          id,
		Name:        "JSON Test",
		Description: "Testing JSON",
		OwnerID:     ownerID,
		Status:      ProjectCompleted,
		Template:    "react-app",
		Progress:    100,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, id, project.ID)
	assert.Equal(t, ProjectCompleted, project.Status)
}

func TestProjectStatusStringValues(t *testing.T) {
	tests := []struct {
		status   ProjectStatus
		expected string
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
