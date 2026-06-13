package model

// AgentType is the closed enumeration of agent types supported by the
// factory. It is distinct from the open Role string on the Agent struct
// (see agent.go) — Role is a free-form classifier used in capability
// assignment, while AgentType is a stable identifier used for routing,
// reporting, and the AgentTypeCapabilities map in capability.go.
type AgentType string

// Known agent types. The six values are the Sprint 4 design set.
const (
	AgentPM       AgentType = "pm"
	AgentArch     AgentType = "architect"
	AgentDev      AgentType = "developer"
	AgentReviewer AgentType = "reviewer"
	AgentQA       AgentType = "qa"
	AgentDevOps   AgentType = "devops"
)

// AllAgentTypes lists the six known agent types in the order they are
// declared above. Used by validation, fixtures, and report generation.
var AllAgentTypes = []AgentType{
	AgentPM,
	AgentArch,
	AgentDev,
	AgentReviewer,
	AgentQA,
	AgentDevOps,
}

// IsValidAgentType reports whether the given string is one of the
// known agent types.
func IsValidAgentType(typeName string) bool {
	for _, t := range AllAgentTypes {
		if string(t) == typeName {
			return true
		}
	}
	return false
}
