package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
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
		ID:              uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		ProjectID:       uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Title:           "Implement feature X",
		Description:     "Detailed description of feature X",
		Priority:        PriorityHigh,
		Status:          TaskInProgress,
		AssigneeAgentID: "agent-789",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	assert.Equal(t, uuid.MustParse("11111111-1111-1111-1111-111111111111"), task.ID)
	assert.Equal(t, uuid.MustParse("22222222-2222-2222-2222-222222222222"), task.ProjectID)
	assert.Equal(t, "Implement feature X", task.Title)
	assert.Equal(t, "Detailed description of feature X", task.Description)
	assert.Equal(t, PriorityHigh, task.Priority)
	assert.Equal(t, TaskInProgress, task.Status)
	assert.Equal(t, "agent-789", task.AssigneeAgentID)
	assert.Equal(t, now, task.CreatedAt)
	assert.Equal(t, now, task.UpdatedAt)
}

func TestTaskMinimalFields(t *testing.T) {
	task := Task{
		ID:        uuid.MustParse("77777777-7777-7777-7777-777777777777"),
		ProjectID: uuid.MustParse("88888888-8888-8888-8888-888888888888"),
		Title:     "Minimal task",
		Priority:  PriorityMedium,
		Status:    TaskTodo,
	}
	assert.Equal(t, uuid.MustParse("77777777-7777-7777-7777-777777777777"), task.ID)
	assert.Equal(t, uuid.MustParse("88888888-8888-8888-8888-888888888888"), task.ProjectID)
	assert.Equal(t, "Minimal task", task.Title)
	assert.Equal(t, PriorityMedium, task.Priority)
	assert.Equal(t, TaskTodo, task.Status)
	assert.Empty(t, task.Description)
	assert.Empty(t, task.Type)
	assert.Nil(t, task.AcceptanceCriteria)
	assert.Zero(t, task.EstimatedHours)
	assert.Empty(t, task.AssigneeAgentID)
}