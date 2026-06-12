package service

import (
	"context"
	"errors"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// CapabilityService provides capability matching and assignment logic.
//
// The methods on the right (CapabilitiesForRole, TaskRequiresCapability,
// AgentHasCapability, FindCompatibleAgents, AssignmentScore) are the
// original pre-Sprint-4 helpers and operate on a snapshot
// *model.Agent — they are used by AgentOrchestrator for capability
// discovery and have no database dependency.
//
// ValidateAgentHasCapabilities is the Sprint 4 (TASK-403) enforcement
// seam: it reads the live capability grant set from the agent store
// (which mirrors the agent_capabilities join table, migration 017)
// and returns ErrCapabilityMismatch if the agent does not hold every
// requested capability. AssignmentService calls this before persisting
// an assignment (api-spec.md §3.1).
type CapabilityService struct {
	stores store.Store
	log    *zap.Logger
}

// NewCapabilityService wires the live-store dependency required by
// ValidateAgentHasCapabilities. Pre-Sprint-4 callers that only use
// the role-snapshot helpers can still call this constructor (the
// store is only consulted by ValidateAgentHasCapabilities).
func NewCapabilityService(s store.Store, log *zap.Logger) *CapabilityService {
	return &CapabilityService{stores: s, log: log}
}

// CapabilitiesForRole returns the default capabilities for a given role string.
func (s *CapabilityService) CapabilitiesForRole(role string) []string {
	return model.DefaultCapabilitiesForRole(role)
}

// TaskRequiresCapability returns the capabilities needed for a given task type.
func (s *CapabilityService) TaskRequiresCapability(taskType string) []string {
	switch taskType {
	case "feature", "implementation":
		return []string{"coding", "testing"}
	case "architecture", "design":
		return []string{"architecture"}
	case "bugfix":
		return []string{"coding"}
	case "review":
		return []string{"testing", "security"}
	case "test", "qa":
		return []string{"testing"}
	case "security_audit":
		return []string{"security"}
	case "deployment", "infrastructure":
		return []string{"devops", "architecture"}
	case "documentation":
		return []string{"documentation"}
	case "data_pipeline", "analytics":
		return []string{"data_engineering", "coding"}
	case "project_management", "planning":
		return []string{"project_management"}
	default:
		return []string{"coding"}
	}
}

// AgentHasCapability checks if an agent possesses ALL required capabilities.
func (s *CapabilityService) AgentHasCapability(agent *model.Agent, required []string) bool {
	if len(required) == 0 {
		return true
	}
	agentCaps := make(map[string]bool, len(agent.Capabilities))
	for _, c := range agent.Capabilities {
		agentCaps[c] = true
	}
	for _, req := range required {
		if !agentCaps[req] {
			return false
		}
	}
	return true
}

// FindCompatibleAgents filters a list of agents to only those having ALL required capabilities.
func (s *CapabilityService) FindCompatibleAgents(agents []*model.Agent, required []string) []*model.Agent {
	if len(required) == 0 {
		result := make([]*model.Agent, len(agents))
		copy(result, agents)
		return result
	}
	var compatible []*model.Agent
	for _, a := range agents {
		if s.AgentHasCapability(a, required) {
			compatible = append(compatible, a)
		}
	}
	return compatible
}

// AssignmentScore returns a match score between an agent and required capabilities.
// Higher scores indicate a better fit. Scoring:
//   - +2 per directly matching capability
//   - +1 for any extra capability the agent has (breadth bonus)
//   - -5 if any required capability is missing (effectively disqualifies)
func (s *CapabilityService) AssignmentScore(agent *model.Agent, required []string) int {
	if len(required) == 0 {
		return len(agent.Capabilities)
	}

	agentSet := make(map[string]bool, len(agent.Capabilities))
	for _, c := range agent.Capabilities {
		agentSet[c] = true
	}

	score := 0
	for _, req := range required {
		if agentSet[req] {
			score += 2
			delete(agentSet, req)
		} else {
			score -= 5
		}
	}

	for range agentSet {
		score += 1
	}

	return score
}

// ValidateAgentHasCapabilities is the Sprint 4 (TASK-403) enforcement
// seam. It reads the agent's granted capabilities from the store
// (which mirrors the agent_capabilities join table, migration 017) and
// returns ErrCapabilityMismatch (mapped to a 409 with code
// CAPABILITY_MISMATCH per api-spec.md §3.1) if the agent is missing
// one or more of the required capabilities.
//
// Behaviour contract:
//   - Empty/nil required slice: no-op, returns nil. There is no
//     constraint, so every agent is eligible.
//   - Agent not found in the store: returns notFound("agent ...").
//   - Store error: returns the wrapped error and logs it.
//   - Missing capabilities: returns ErrCapabilityMismatch with the
//     missing-names list in Details.
//
// This method is called by AssignmentService.AssignTaskToAgent
// before persisting an assignment. The CapabilityService is also
// safe to call from other services (e.g. the future Sprint 5
// task-creation endpoint that pre-validates an agent pick) — it
// has no side effects.
func (s *CapabilityService) ValidateAgentHasCapabilities(ctx context.Context, agentID uuid.UUID, required []string) error {
	// No-op for an empty required list — every agent is eligible.
	if len(required) == 0 {
		return nil
	}

	agents := s.stores.Agents()

	// Read the agent's granted capability set. The store is the
	// single source of truth for the join table + JSONB cache
	// (data-model.md §3 invariant), so we do NOT trust any
	// pre-computed snapshot on the request.
	granted, err := agents.ListCapabilitiesByAgent(ctx, agentID)
	if err != nil {
		// Map store.ErrNotFound to the service-level notFound
		// envelope so the handler can return a clean 404.
		if errors.Is(err, store.ErrNotFound) {
			return notFound("agent " + agentID.String() + " not found")
		}
		if s.log != nil {
			s.log.Error("capability validation: store error",
				zap.String("agent_id", agentID.String()),
				zap.Strings("required", required),
				zap.Error(err))
		}
		return internalError("failed to load agent capabilities")
	}

	// Build a set of granted capability names for O(1) lookup.
	grantedSet := make(map[string]struct{}, len(granted))
	for _, c := range granted {
		grantedSet[c.Name] = struct{}{}
	}

	// Walk the required list and collect anything missing.
	var missing []string
	for _, req := range required {
		if _, ok := grantedSet[req]; !ok {
			missing = append(missing, req)
		}
	}
	if len(missing) > 0 {
		return capabilityMismatch(missing)
	}
	return nil
}
