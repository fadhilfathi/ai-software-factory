package model

import "time"

// Environment represents the deployment target.
type Environment string

const (
	EnvDevelopment Environment = "development"
	EnvStaging     Environment = "staging"
	EnvProduction  Environment = "production"
)

// DeploymentStatus represents the lifecycle state of a deployment.
type DeploymentStatus string

const (
	DeployQueued      DeploymentStatus = "queued"
	DeployBuilding    DeploymentStatus = "building"
	DeployTesting     DeploymentStatus = "testing"
	DeployDeploying   DeploymentStatus = "deploying"
	DeployCompleted   DeploymentStatus = "completed"
	DeployFailed      DeploymentStatus = "failed"
	DeployRollingBack DeploymentStatus = "rolling_back"
)

// DeploymentStep represents a phase within a deployment pipeline.
type DeploymentStep struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Duration int    `json:"duration"`
}

// Deployment represents a deployment of a project to an environment.
type Deployment struct {
	ID            string           `json:"id"`
	ProjectID     string           `json:"project_id"`
	Environment   Environment      `json:"environment"`
	Branch        string           `json:"branch"`
	Status        DeploymentStatus `json:"status"`
	URL           string           `json:"url,omitempty"`
	EstimatedTime int              `json:"estimated_time,omitempty"`
	Steps         []DeploymentStep `json:"steps,omitempty"`
	RollbackFrom  string           `json:"rollback_from,omitempty"`
	RollbackTo    string           `json:"rollback_to,omitempty"`
	StartedAt     *time.Time       `json:"started_at,omitempty"`
	CompletedAt   *time.Time       `json:"completed_at,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}
