package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestEnvironmentConstants(t *testing.T) {
	assert.Equal(t, Environment("development"), EnvDevelopment)
	assert.Equal(t, Environment("staging"), EnvStaging)
	assert.Equal(t, Environment("production"), EnvProduction)
}

func TestDeploymentStatusConstants(t *testing.T) {
	assert.Equal(t, DeploymentStatus("queued"), DeployQueued)
	assert.Equal(t, DeploymentStatus("building"), DeployBuilding)
	assert.Equal(t, DeploymentStatus("testing"), DeployTesting)
	assert.Equal(t, DeploymentStatus("deploying"), DeployDeploying)
	assert.Equal(t, DeploymentStatus("completed"), DeployCompleted)
	assert.Equal(t, DeploymentStatus("failed"), DeployFailed)
	assert.Equal(t, DeploymentStatus("rolling_back"), DeployRollingBack)
}

func TestDeploymentStepStruct(t *testing.T) {
	step := DeploymentStep{
		Name:     "Build",
		Status:   "completed",
		Duration: 45,
	}
	assert.Equal(t, "Build", step.Name)
	assert.Equal(t, "completed", step.Status)
	assert.Equal(t, 45, step.Duration)
}

func TestDeploymentStruct(t *testing.T) {
	now := time.Now().UTC()
	startedAt := now.Add(-10 * time.Minute)
	completedAt := now

	deployment := Deployment{
		ID:            uuid.MustParse("00000000-0000-0000-0000-000000000050"),
		ProjectID:     uuid.MustParse("00000000-0000-0000-0000-000000000051"),
		Environment:   EnvProduction,
		Branch:        "main",
		Status:        DeployCompleted,
		URL:           "https://app.example.com",
		EstimatedTime: 300,
		Steps: []DeploymentStep{
			{Name: "Build", Status: "completed", Duration: 60},
			{Name: "Test", Status: "completed", Duration: 120},
			{Name: "Deploy", Status: "completed", Duration: 45},
		},
		RollbackFrom: "abc123",
		RollbackTo:   "def456",
		StartedAt:    &startedAt,
		CompletedAt:  &completedAt,
		CreatedAt:    now.Add(-15 * time.Minute),
		UpdatedAt:    now,
	}

	assert.Equal(t, uuid.MustParse("00000000-0000-0000-0000-000000000050"), deployment.ID)
	assert.Equal(t, uuid.MustParse("00000000-0000-0000-0000-000000000051"), deployment.ProjectID)
	assert.Equal(t, EnvProduction, deployment.Environment)
	assert.Equal(t, "main", deployment.Branch)
	assert.Equal(t, DeployCompleted, deployment.Status)
	assert.Equal(t, "https://app.example.com", deployment.URL)
	assert.Equal(t, 300, deployment.EstimatedTime)
	assert.Len(t, deployment.Steps, 3)
	assert.Equal(t, "Build", deployment.Steps[0].Name)
	assert.Equal(t, "abc123", deployment.RollbackFrom)
	assert.Equal(t, "def456", deployment.RollbackTo)
	assert.Equal(t, startedAt, *deployment.StartedAt)
	assert.Equal(t, completedAt, *deployment.CompletedAt)
}

func TestDeploymentWithNilTimestamps(t *testing.T) {
	deployment := Deployment{
		ID:          uuid.MustParse("00000000-0000-0000-0000-000000000052"),
		ProjectID:   uuid.MustParse("00000000-0000-0000-0000-000000000053"),
		Environment: EnvStaging,
		Branch:      "feature",
		Status:      DeployQueued,
	}
	assert.Nil(t, deployment.StartedAt)
	assert.Nil(t, deployment.CompletedAt)
	assert.Nil(t, deployment.Steps)
}