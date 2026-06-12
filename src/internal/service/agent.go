package service

// Agent service (TASK-402, Sprint 4).
//
// Maps the api-spec.md §1 + §2.1 endpoints to the storage layer:
//   - POST   /v1/agents            → CreateAgent
//   - GET    /v1/agents            → ListAgents (cursor pagination, filters)
//   - GET    /v1/agents/:id        → GetAgent
//   - PUT    /v1/agents/:id        → UpdateAgent (optimistic concurrency)
//   - DELETE /v1/agents/:id        → RetireAgent (soft delete)
//   - GET    /v1/agents/:id/capabilities → ListAgentCapabilities
//   - GET    /v1/capabilities      → ListCapabilities (catalog)
//
// All methods return a *Error (see service/errors.go) on failure; the
// handler layer maps the error to the api-spec error envelope.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
)

// AgentService is the interface the handler depends on. It exists so
// the handler can be tested with a mock; the concrete implementation
// is *agentService below.
type AgentService interface {
	CreateAgent(ctx context.Context, req CreateAgentRequest) (*model.Agent, *Error)
	GetAgent(ctx context.Context, id uuid.UUID) (*model.Agent, *Error)
	ListAgents(ctx context.Context, req ListAgentsRequest) (*ListAgentsResult, *Error)
	UpdateAgent(ctx context.Context, id uuid.UUID, req UpdateAgentRequest) (*model.Agent, *Error)
	RetireAgent(ctx context.Context, id uuid.UUID, force bool) *Error
	ListAgentCapabilities(ctx context.Context, id uuid.UUID) ([]*model.AgentCapability, *Error)
	ListCapabilities(ctx context.Context, req ListCapabilitiesRequest) (*ListCapabilitiesResult, *Error)
}

// ============================================================================
// Request / result types (handler-friendly shapes, decoupled from
// model.Agent so future field renames are a one-line change)
// ============================================================================

// CreateAgentRequest is the body of POST /v1/agents. ProjectID is
// pulled from the URL (or the auth context) by the handler; the
// service receives it explicitly here so the interface is testable
// in isolation.
type CreateAgentRequest struct {
	ProjectID    uuid.UUID
	Name         string
	Role         string
	Capabilities []string
	Metadata     json.RawMessage
}

// ListAgentsRequest is the parameter object for ListAgents. The
// handler fills this from the query string; the service applies the
// api-spec defaults.
type ListAgentsRequest struct {
	ProjectID      uuid.UUID
	Status         string
	Capability     string
	IncludeRetired bool
	Cursor         string
	Limit          int
}

// ListAgentsResult is the response shape the handler maps onto the
// api-spec.md §1.2 envelope.
type ListAgentsResult struct {
	Data       []*model.Agent
	NextCursor string
	HasMore    bool
}

// UpdateAgentRequest holds the partial-update fields. Pointer-typed
// fields encode "absent" (do not change) vs "present" (replace).
type UpdateAgentRequest struct {
	Role         *string
	Status       *model.AgentStatus
	Capabilities *[]string
	Metadata     json.RawMessage
	Version      *int
}

// ListCapabilitiesRequest is the parameter object for
// ListCapabilities (catalog read).
type ListCapabilitiesRequest struct {
	Category string
	Cursor   string
	Limit    int
}

// ListCapabilitiesResult is the response shape for the catalog read.
type ListCapabilitiesResult struct {
	Data       []model.CapabilityRow
	NextCursor string
	HasMore    bool
}

// ============================================================================
// Concrete implementation
// ============================================================================

type agentService struct {
	store       store.Store
	capabilities store.CapabilityStore
	now         func() time.Time
}

// NewAgentService wires the service. The capabilities substore is
// pulled out at construction so the validation path is one map
// lookup, not a function call into the Store interface.
func NewAgentService(s store.Store) AgentService {
	return &agentService{
		store:        s,
		capabilities: s.Capabilities(),
		now:          time.Now,
	}
}

// ============================================================================
// Create
// ============================================================================

// CreateAgent validates the request, fills in defaults, and inserts
// the new agent. Capability existence is checked against the catalog
// before the insert; a missing capability is a 422 CAPABILITY_NOT_FOUND.
func (s *agentService) CreateAgent(ctx context.Context, req CreateAgentRequest) (*model.Agent, *Error) {
	if err := s.validateAgentName(req.Name); err != nil {
		return nil, err
	}
	if err := s.validateAgentRole(req.Role); err != nil {
		return nil, err
	}
	if len(req.Capabilities) == 0 {
		return nil, validationSingle("capabilities", "At least one capability is required")
	}
	if err := s.validateCapabilitiesExist(ctx, req.Capabilities); err != nil {
		return nil, err
	}

	now := s.now().UTC()
	agent := &model.Agent{
		ID:           uuid.New(),
		ProjectID:    req.ProjectID,
		Name:         req.Name,
		Role:         req.Role,
		Status:       model.AgentInitializing,
		Capabilities: append([]string(nil), req.Capabilities...),
		Metadata:     metadataOrEmpty(req.Metadata),
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.store.Agents().Create(ctx, agent); err != nil {
		if errors.Is(err, store.ErrAlreadyExists) {
			return nil, &Error{
				Status:  409,
				Code:    "ALREADY_EXISTS",
				Message: "An agent with this name already exists in the project",
			}
		}
		return nil, internalError("Failed to create agent")
	}
	return agent, nil
}

// ============================================================================
// Read
// ============================================================================

func (s *agentService) GetAgent(ctx context.Context, id uuid.UUID) (*model.Agent, *Error) {
	a, err := s.store.Agents().GetByID(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return nil, notFound("Agent not found")
	}
	if err != nil {
		return nil, internalError("Failed to fetch agent")
	}
	return a, nil
}

func (s *agentService) ListAgents(ctx context.Context, req ListAgentsRequest) (*ListAgentsResult, *Error) {
	if req.ProjectID == uuid.Nil {
		return nil, validationSingle("project_id", "project_id is required")
	}
	filter := model.AgentFilter{
		ProjectID:      req.ProjectID,
		IncludeRetired: req.IncludeRetired,
		Cursor:         req.Cursor,
		Limit:          req.Limit,
	}
	if req.Status != "" {
		st := model.AgentStatus(req.Status)
		if !model.IsValidAgentStatus(st) {
			return nil, validationSingle("status", "Unknown status: "+req.Status)
		}
		filter.Status = st
	}
	if req.Capability != "" {
		// Validate the capability exists in the catalog. The
		// api-spec.md §1.2 says unknown capability returns
		// 400 VALIDATION_ERROR.
		ok, err := s.capabilities.Exists(ctx, req.Capability)
		if err != nil {
			return nil, internalError("Failed to validate capability filter")
		}
		if !ok {
			return nil, validationSingle("capability", "Unknown capability: "+req.Capability)
		}
		filter.Capability = req.Capability
	}

	page, err := s.store.Agents().List(ctx, filter)
	if err != nil {
		return nil, internalError("Failed to list agents")
	}
	return &ListAgentsResult{
		Data:       page.Data,
		NextCursor: page.NextCursor,
		HasMore:    page.HasMore,
	}, nil
}

// ListAgentCapabilities is the per-agent read. We return
// []model.AgentCapability directly to keep the handler mapping
// trivial.
func (s *agentService) ListAgentCapabilities(ctx context.Context, id uuid.UUID) ([]*model.AgentCapability, *Error) {
	caps, err := s.store.Agents().ListCapabilitiesByAgent(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return nil, notFound("Agent not found")
	}
	if err != nil {
		return nil, internalError("Failed to fetch agent capabilities")
	}
	return caps, nil
}

// ============================================================================
// Update
// ============================================================================

// UpdateAgent applies a partial update with optimistic-concurrency
// (api-spec.md §1.4). The caller must send the current version; a
// mismatch returns 409 VERSION_CONFLICT. Capability rewrites are
// done in the store's SetCapabilities to keep the join + cache
// consistent; a capability rewrite also bumps the version.
func (s *agentService) UpdateAgent(ctx context.Context, id uuid.UUID, req UpdateAgentRequest) (*model.Agent, *Error) {
	existing, apiErr := s.GetAgent(ctx, id)
	if apiErr != nil {
		return nil, apiErr
	}
	if existing.RetiredAt != nil {
		return nil, &Error{
			Status:  409,
			Code:    "ALREADY_RETIRED",
			Message: "Agent is retired; create a new agent instead",
		}
	}

	if req.Version == nil {
		return nil, validationSingle("version", "version is required for optimistic concurrency")
	}
	if *req.Version != existing.Version {
		return nil, &Error{
			Status:  409,
			Code:    "VERSION_CONFLICT",
			Message: fmt.Sprintf("Stale version (have %d, current %d)", *req.Version, existing.Version),
		}
	}

	// Validate any partial-update fields.
	if req.Role != nil {
		if err := s.validateAgentRole(*req.Role); err != nil {
			return nil, err
		}
		existing.Role = *req.Role
	}
	if req.Status != nil {
		if !model.IsValidAgentStatus(*req.Status) {
			return nil, validationSingle("status", "Unknown status: "+string(*req.Status))
		}
		existing.Status = *req.Status
	}
	if req.Metadata != nil {
		existing.Metadata = metadataOrEmpty(req.Metadata)
	}

	// Two write paths: capabilities-change uses SetCapabilities
	// (transactional with the join table); other fields use the
	// straight Update with the version bump.
	if req.Capabilities != nil {
		if len(*req.Capabilities) == 0 {
			return nil, validationSingle("capabilities", "Capabilities cannot be empty")
		}
		if err := s.validateCapabilitiesExist(ctx, *req.Capabilities); err != nil {
			return nil, err
		}
		if err := s.store.Agents().SetCapabilities(ctx, existing.ID, *req.Capabilities); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, notFound("Agent not found")
			}
			return nil, internalError("Failed to update capabilities")
		}
		// Re-read so the returned shape has the new version and
		// refreshed cache.
		fresh, apiErr := s.GetAgent(ctx, existing.ID)
		if apiErr != nil {
			return nil, apiErr
		}
		// Now apply the non-capability fields (if any). Pass the
		// freshly-read version to Update so optimistic
		// concurrency is honoured against the new value.
		fresh.Role = existing.Role
		fresh.Status = existing.Status
		fresh.Metadata = existing.Metadata
		if err := s.store.Agents().Update(ctx, fresh); err != nil {
			if errors.Is(err, store.ErrConflict) {
				return nil, &Error{
					Status:  409,
					Code:    "VERSION_CONFLICT",
					Message: "Concurrent modification; refresh and retry",
				}
			}
			return nil, internalError("Failed to update agent")
		}
		// Re-read one more time to surface the final version.
		return s.GetAgent(ctx, existing.ID)
	}

	// No capabilities change — straight Update.
	if err := s.store.Agents().Update(ctx, existing); err != nil {
		if errors.Is(err, store.ErrConflict) {
			return nil, &Error{
				Status:  409,
				Code:    "VERSION_CONFLICT",
				Message: "Concurrent modification; refresh and retry",
			}
		}
		return nil, internalError("Failed to update agent")
	}
	return s.GetAgent(ctx, existing.ID)
}

// ============================================================================
// Soft delete
// ============================================================================

// RetireAgent soft-deletes the agent (status=retired, retired_at=NOW).
// The api-spec.md §1.5 ?force=true path is a handler concern; the
// service does the soft delete unconditionally.
func (s *agentService) RetireAgent(ctx context.Context, id uuid.UUID, _ bool) *Error {
	err := s.store.Agents().SoftDelete(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return notFound("Agent not found")
	}
	if err != nil {
		return internalError("Failed to retire agent")
	}
	return nil
}

// ============================================================================
// Capabilities catalog
// ============================================================================

func (s *agentService) ListCapabilities(ctx context.Context, req ListCapabilitiesRequest) (*ListCapabilitiesResult, *Error) {
	if req.Category != "" {
		// The catalog only has the 6 seeded categories (data-model
		// §2). Reject anything else to give the client a clear
		// 400 instead of an empty result.
		switch req.Category {
		case "architecture", "coding", "testing", "security", "devops", "leadership":
		default:
			return nil, validationSingle("category", "Unknown category: "+req.Category)
		}
	}
	page, err := s.capabilities.List(ctx, model.CapabilityFilter{
		Category: req.Category,
		Cursor:   req.Cursor,
		Limit:    req.Limit,
	})
	if err != nil {
		return nil, internalError("Failed to list capabilities")
	}
	return &ListCapabilitiesResult{
		Data:       page.Data,
		NextCursor: page.NextCursor,
		HasMore:    page.HasMore,
	}, nil
}

// ============================================================================
// Helpers
// ============================================================================

// validateAgentName enforces the api-spec.md §1.1 length cap (80
// chars) and a minimum length. The DB also has a UNIQUE
// (project_id, name) constraint as a backstop.
func (s *agentService) validateAgentName(name string) *Error {
	name = strings.TrimSpace(name)
	if name == "" {
		return validationSingle("name", "Name is required")
	}
	if len(name) > 80 {
		return validationSingle("name", "Name must be 80 characters or fewer")
	}
	return nil
}

// validateAgentRole enforces the api-spec.md §1.1 length cap (255
// chars) and a minimum length. Role is free text but the api-spec
// lists reasonable defaults.
func (s *agentService) validateAgentRole(role string) *Error {
	role = strings.TrimSpace(role)
	if role == "" {
		return validationSingle("role", "Role is required")
	}
	if len(role) > 255 {
		return validationSingle("role", "Role must be 255 characters or fewer")
	}
	return nil
}

// validateCapabilitiesExist checks every name against the catalog.
// Any miss returns a 422 CAPABILITY_NOT_FOUND with the missing
// names in the error details. This is the api-spec's recommended
// behaviour for unknown capabilities.
func (s *agentService) validateCapabilitiesExist(ctx context.Context, names []string) *Error {
	if len(names) == 0 {
		return nil
	}
	// De-duplicate to avoid double-checks.
	seen := make(map[string]struct{}, len(names))
	missing := make([]string, 0)
	for _, n := range names {
		if _, dup := seen[n]; dup {
			continue
		}
		seen[n] = struct{}{}
		ok, err := s.capabilities.Exists(ctx, n)
		if err != nil {
			return internalError("Failed to validate capabilities")
		}
		if !ok {
			missing = append(missing, n)
		}
	}
	if len(missing) > 0 {
		return unprocessableEntity("CAPABILITY_NOT_FOUND", "One or more capabilities do not exist in the catalog")
	}
	return nil
}

// metadataOrEmpty normalises nil / empty metadata to the literal
// "{}" so the JSONB column never gets NULL.
func metadataOrEmpty(m json.RawMessage) json.RawMessage {
	if len(m) == 0 {
		return json.RawMessage(`{}`)
	}
	// Defensive: if the caller passed "null", treat as empty.
	s := strings.TrimSpace(string(m))
	if s == "" || s == "null" {
		return json.RawMessage(`{}`)
	}
	return m
}
