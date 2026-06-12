package service

import (
	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
)

// CapabilityService provides capability matching and assignment logic.
type CapabilityService struct{}

func NewCapabilityService() *CapabilityService {
	return &CapabilityService{}
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
