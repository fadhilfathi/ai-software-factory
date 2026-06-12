package model

import (
	"time"

	"github.com/google/uuid"
)

// Deliverable represents an artifact produced by an agent for a task.
type Deliverable struct {
	ID        uuid.UUID `json:"id"`
	TaskID    uuid.UUID `json:"task_id"`
	AgentID   uuid.UUID `json:"agent_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}
