package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AuditLog represents a record of a change in the system.
type AuditLog struct {
	ID         uuid.UUID       `json:"id"`
	EntityType string          `json:"entity_type"`
	EntityID   uuid.UUID       `json:"entity_id"`
	Action     string          `json:"action"`
	UserID     *uuid.UUID      `json:"user_id,omitempty"`
	Changes    json.RawMessage `json:"changes"`
	CreatedAt  time.Time       `json:"created_at"`
}
