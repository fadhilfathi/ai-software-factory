package model

// Agent is the canonical shape used across the store / service / handler
// layers and the JSON wire format (api-spec.md §1).
//
// Migration reference: src/db/migrations/016_agent_registry.sql (Sprint 4
// TASK-402). The DB shape and the Go struct must stay in sync — see
// data-model.md §1.

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AgentStatus is the lifecycle state of an agent. The string values
// MUST match the DB CHECK constraint in 016_agent_registry.sql; the
// service layer rejects any value outside this set on write.
type AgentStatus string

const (
	// AgentInitializing is the default on create. Transitions to
	// idle after the agent's first heartbeat (TASK-405 / execution
	// tracking) or to error if startup fails.
	AgentInitializing AgentStatus = "initializing"

	// AgentIdle means the agent is ready and will accept assignments.
	AgentIdle AgentStatus = "idle"

	// AgentBusy means the agent has at least one open execution.
	AgentBusy AgentStatus = "busy"

	// AgentPaused means the agent is suspended by an operator; the
	// assignment engine (TASK-404) skips paused agents in auto-routing.
	AgentPaused AgentStatus = "paused"

	// AgentError means the agent's last execution failed and the
	// agent is in a failed state until an operator resumes it.
	AgentError AgentStatus = "error"

	// AgentRetired is the soft-delete state. retired_at is non-null.
	// The list endpoint excludes retired agents by default
	// (pass ?include_retired=true to include).
	AgentRetired AgentStatus = "retired"
)

// AllAgentStatuses is the canonical lifecycle state set, used to
// validate incoming filter values and to seed the api-spec's
// error-table examples. Keep in sync with the DB CHECK.
var AllAgentStatuses = []AgentStatus{
	AgentInitializing,
	AgentIdle,
	AgentBusy,
	AgentPaused,
	AgentError,
	AgentRetired,
}

// IsValidAgentStatus reports whether s is in the lifecycle state set.
func IsValidAgentStatus(s AgentStatus) bool {
	for _, v := range AllAgentStatuses {
		if v == s {
			return true
		}
	}
	return false
}

// Agent is the canonical agent record. The JSON tags drive the wire
// format (api-spec.md §1.1). The DB column-to-field mapping is in
// store/postgres/agent_store.go.
type Agent struct {
	ID uuid.UUID `json:"id"`

	// ProjectID is FK -> projects.id. Required on create.
	// Immutable on update (the api-spec does not allow it on PUT).
	ProjectID uuid.UUID `json:"project_id"`

	// Name is a human-friendly label. Required, ≤ 80 chars per
	// api-spec.md §1.1, unique within ProjectID.
	Name string `json:"name"`

	// Role is free-text per the data-model.md / api-spec. Default
	// conventions: "developer", "reviewer", "qa", "devops", etc.
	// Required on create; mutable on update.
	Role string `json:"role"`

	// Status is the lifecycle state. Defaults to "initializing" on
	// create. The DB CHECK rejects any value outside AllAgentStatuses.
	Status AgentStatus `json:"status"`

	// Capabilities is the denormalised cache of the agent's granted
	// capability names. Mirrored from agent_capabilities in the
	// same transaction by the store's SetCapabilities method. Stored
	// as JSONB in the DB and emitted as a JSON array on the wire.
	// Required on create (≥ 1 element per api-spec.md §1.1).
	Capabilities []string `json:"capabilities"`

	// LastActiveAt is the most recent activity timestamp. Updated by
	// the execution engine (TASK-405). Nullable: nil on a freshly
	// created agent that has not yet had a heartbeat.
	LastActiveAt *time.Time `json:"last_active_at"`

	// Metadata is free-form per-agent metadata. Optional on create.
	// Stored as JSONB in the DB and emitted as a JSON object on the
	// wire. Mutated via PUT.
	Metadata json.RawMessage `json:"metadata"`

	// Version is the optimistic-concurrency counter. Starts at 1 on
	// create. Bumped on every successful Update / SetCapabilities.
	// The api-spec.md §1.4 PUT contract requires clients to send
	// the current version; mismatch returns 409 VERSION_CONFLICT.
	Version int `json:"version,omitempty"`

	// RetiredAt is non-nil iff Status == AgentRetired. The list
	// endpoint excludes retired agents by default. The DB has a
	// partial index `idx_agents_project_status ... WHERE retired_at
	// IS NULL` for the active-agent hot path.
	RetiredAt *time.Time `json:"retired_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AgentFilter is the parameter object passed to AgentStore.List.
// Mirrors the api-spec.md §1.2 query parameters.
type AgentFilter struct {
	ProjectID      uuid.UUID   // required (api-spec.md §1.2)
	Status         AgentStatus // optional, exact match
	Capability     string      // optional, "agents declaring this capability"
	IncludeRetired bool        // default false; the spec excludes retired
	Cursor         string      // optional opaque cursor
	Limit          int         // default 50, max 200
}

// AgentListResult is the page envelope returned by AgentStore.List.
// The service layer maps this to the api-spec.md §0.7 response shape.
type AgentListResult struct {
	Data       []*Agent
	NextCursor string
	HasMore    bool
}

// AgentCapability is the per-capability view returned by
// GET /v1/agents/:id/capabilities (api-spec.md §1.6). The proficiency
// field is nullable per data-model.md §3. GrantedAt records when the
// capability was assigned to the agent. GrantedBy is the user who
// performed the assignment, or nil for system-granted capabilities.
type AgentCapability struct {
	Name         string     `json:"name"`
	DisplayName  string     `json:"display_name"`
	Category     string     `json:"category"`
	Proficiency  *int       `json:"proficiency,omitempty"`
	GrantedAt    time.Time  `json:"granted_at"`
	GrantedBy    *uuid.UUID `json:"granted_by,omitempty"`
}

// Capability is the catalog row returned by GET /v1/capabilities
// (api-spec.md §2.1). Note: this is a separate type from
// AgentCapability because the catalog row does not carry
// proficiency / granted_at — those are agent-level.
type Capability struct {
	ID          uuid.UUID `json:"id,omitempty"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Category    string    `json:"category"`
	Description string    `json:"description,omitempty"`
	Version     int       `json:"version"`
}

// CapabilityFilter is the parameter object for CapabilityStore.List.
type CapabilityFilter struct {
	Category string
	Cursor   string
	Limit    int
}

// CapabilityListResult is the page envelope for the catalog.
type CapabilityListResult struct {
	Data       []Capability
	NextCursor string
	HasMore    bool
}
