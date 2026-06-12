package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTaskPriorityConstants(t *testing.T) {
	assert.Equal(t, TaskPriority("low"), PriorityLow)
	assert.Equal(t, TaskPriority("medium"), PriorityMedium)
	assert.Equal(t, TaskPriority("high"), PriorityHigh)
	assert.Equal(t, TaskPriority("critical"), PriorityCritical)
}

func TestTaskStatusConstants(t *testing.T) {
	assert.Equal(t, TaskStatus("backlog"), TaskBacklog)
	assert.Equal(t, TaskStatus("todo"), TaskTodo)
	assert.Equal(t, TaskStatus("in_progress"), TaskInProgress)
	assert.Equal(t, TaskStatus("review"), TaskReview)
	assert.Equal(t, TaskStatus("done"), TaskDone)
}

func TestTaskStructFields(t *testing.T) {
	now := time.Now().UTC()
	task := Task{
		ID:                 "task-123",
		ProjectID:          "proj-456",
		Title:              "Implement feature X",
		Description:        "Detailed description of feature X",
		Type:               "feature",
		AcceptanceCriteria: []string{"Criteria 1", "Criteria 2"},
		Priority:           PriorityHigh,
		Status:             TaskInProgress,
		EstimatedHours:     8,
		AssigneeAgentID:    "agent-789",
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	assert.Equal(t, "task-123", task.ID)
	assert.Equal(t, "proj-456", task.ProjectID)
	assert.Equal(t, "Implement feature X", task.Title)
	assert.Equal(t, "Detailed description of feature X", task.Description)
	assert.Equal(t, "feature", task.Type)
	assert.Equal(t, []string{"Criteria 1", "Criteria 2"}, task.AcceptanceCriteria)
	assert.Equal(t, PriorityHigh, task.Priority)
	assert.Equal(t, TaskInProgress, task.Status)
	assert.Equal(t, 8, task.EstimatedHours)
	assert.Equal(t, "agent-789", task.AssigneeAgentID)
	assert.Equal(t, now, task.CreatedAt)
	assert.Equal(t, now, task.UpdatedAt)
}

func TestTaskMinimalFields(t *testing.T) {
	task := Task{
		ID:        "task-minimal",
		ProjectID: "proj-1",
		Title:     "Minimal task",
		Priority:  PriorityMedium,
		Status:    TaskTodo,
	}
	assert.Equal(t, "task-minimal", task.ID)
	assert.Equal(t, "proj-1", task.ProjectID)
	assert.Equal(t, "Minimal task", task.Title)
	assert.Equal(t, PriorityMedium, task.Priority)
	assert.Equal(t, TaskTodo, task.Status)
	assert.Empty(t, task.Description)
	assert.Empty(t, task.Type)
	assert.Nil(t, task.AcceptanceCriteria)
	assert.Zero(t, task.EstimatedHours)
	assert.Empty(t, task.AssigneeAgentID)
}