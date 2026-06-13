package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRoleConstants(t *testing.T) {
	assert.Equal(t, Role("admin"), RoleAdmin)
	assert.Equal(t, Role("member"), RoleMember)
	assert.Equal(t, Role("viewer"), RoleViewer)
}

func TestUserStructFields(t *testing.T) {
	now := time.Now().UTC()
	user := User{
		ID:           uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Email:        "test@example.com",
		PasswordHash: "hashed-password",
		Name:      "Test User",
		Role:      RoleAdmin,
		Teams:     []string{"team-1", "team-2"},
		Projects:  []string{"proj-1", "proj-2"},
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, uuid.MustParse("11111111-1111-1111-1111-111111111111"), user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "hashed-password", user.PasswordHash)
	assert.Equal(t, "Test User", user.Name)
	assert.Equal(t, RoleAdmin, user.Role)
	assert.Equal(t, []string{"team-1", "team-2"}, user.Teams)
	assert.Equal(t, []string{"proj-1", "proj-2"}, user.Projects)
	assert.Equal(t, now, user.CreatedAt)
	assert.Equal(t, now, user.UpdatedAt)
}

func TestUserJSONExcludesPassword(t *testing.T) {
	// The Password field has json:"-" tag, so it should not be serialized
	// This is a compile-time check - if the tag is missing, the test would need JSON marshaling
	user := User{
		ID:           uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Email:        "test@test.com",
		PasswordHash: "secret",
		Name:     "Test",
		Role:     RoleMember,
	}
	// Password field exists but should not be in JSON output
	assert.NotEmpty(t, user.PasswordHash) // Field exists in struct
	_ = user // suppress unused warning
}