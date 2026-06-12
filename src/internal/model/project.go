package model

import (
	"time"

	"github.com/google/uuid"
)

// ProjectStatus represents the lifecycle state of a project.
type ProjectStatus string

const (
	ProjectInitializing ProjectStatus = "initializing"
	ProjectInProgress   ProjectStatus = "in_progress"
	ProjectCompleted    ProjectStatus = "completed"
	ProjectArchived     ProjectStatus = "archived"
)

// Project represents a software project within the factory.
type Project struct {
	ID          uuid.UUID     `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	OwnerID     uuid.UUID     `json:"owner_id"`
	Status      ProjectStatus `json:"status"`
	Template     string        `json:"template,omitempty"`
	Progress     int           `json:"progress,omitempty"`
	ActiveAgents int           `json:"active_agents,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}
