package model

import (
	"time"

	"github.com/google/uuid"
)

// Role defines the user's authorization level.
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
	RoleViewer Role = "viewer"
)

// User represents a registered user in the system.
type User struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	PasswordHash  string    `json:"-"` // never serialized
	Name      string    `json:"name"`
	Role      Role      `json:"role"`
	Teams     []string  `json:"teams,omitempty"`
	Projects  []string  `json:"projects,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
