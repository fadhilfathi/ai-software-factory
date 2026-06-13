package model

// Capability is a named skill or competency an agent can possess.
type Capability string

const (
	CapArchitecture    Capability = "architecture"
	CapCoding          Capability = "coding"
	CapTesting         Capability = "testing"
	CapSecurity        Capability = "security"
	CapDocumentation   Capability = "documentation"
	CapDevOps          Capability = "devops"
	CapProjectMgmt     Capability = "project_management"
	CapDataEngineering Capability = "data_engineering"
	// CapLeadership is reserved for the Leader agent and is part of the
	// catalog (seeded in migration 016) but is NOT in the assignable set.
	// Per Analyst-01's design (Sprint 4 design docs), only the 5
	// assignable capabilities (architecture, coding, testing, security,
	// devops) can be used as task-assignment constraints.
	CapLeadership Capability = "leadership"

	// Agent-type default capabilities. Used by AgentTypeCapabilities to
	// express the capability set each agent type is responsible for. The
	// 12 names here were the original Sprint 4 design (Analyst-01,
	// TBD) before being collapsed into the current 9-capability catalog.
	// They are kept as full capability constants so the canonical
	// capability set is still self-describing for tests, documentation,
	// and downstream reporting. None of these appear in AssignableCapabilities()
	// above because task-assignment constraints are still restricted to the
	// 5 assignable ones (see comment on CapLeadership).
	CapRequirementAnalysis Capability = "requirement_analysis"
	CapTaskDecomposition   Capability = "task_decomposition"
	CapSystemDesign        Capability = "system_design"
	CapAPIDesign           Capability = "api_design"
	CapCodeImplementation  Capability = "code_implementation"
	CapCodeReview          Capability = "code_review"
	CapSecurityScan        Capability = "security_scan"
	CapTestPlanning        Capability = "test_planning"
	CapTestExecution       Capability = "test_execution"
	CapCICD                Capability = "ci_cd"
	CapDeployment          Capability = "deployment"
	CapInfrastructure      Capability = "infrastructure"
)

// AllCapabilities returns every valid capability constant.
func AllCapabilities() []Capability {
	return []Capability{
		CapArchitecture,
		CapCoding,
		CapTesting,
		CapSecurity,
		CapDocumentation,
		CapDevOps,
		CapProjectMgmt,
		CapDataEngineering,
		CapLeadership,
	}
}

// ValidCapability checks whether a string is a known capability.
func ValidCapability(cap string) bool {
	for _, c := range AllCapabilities() {
		if string(c) == cap {
			return true
		}
	}
	return false
}

// AssignableCapabilities returns the subset of capabilities that may be used
// as task-assignment constraints. Leadership is intentionally excluded — it
// is reserved for the Leader and never appears in a task's
// required_capabilities list. See Analyst-01's design (Sprint 4 design
// docs, §3) and the TASK-403 brief.
func AssignableCapabilities() []Capability {
	return []Capability{
		CapArchitecture,
		CapCoding,
		CapTesting,
		CapSecurity,
		CapDevOps,
	}
}

// IsAssignableCapability reports whether a capability name is in the
// assignable set (i.e. may appear in a task's required_capabilities).
// Used by the service layer to reject requests that name a non-assignable
// capability (e.g. "leadership") before they hit the validation seam.
func IsAssignableCapability(cap string) bool {
	for _, c := range AssignableCapabilities() {
		if string(c) == cap {
			return true
		}
	}
	return false
}

// RoleCapabilities maps agent role strings to their default capability set.
var RoleCapabilities = map[string][]Capability{
	"pm":            {CapProjectMgmt},
	"developer":     {CapCoding, CapTesting},
	"architect":     {CapArchitecture, CapCoding},
	"reviewer":      {CapTesting, CapSecurity},
	"qa":            {CapTesting},
	"security":      {CapSecurity},
	"devops":        {CapDevOps, CapArchitecture},
	"techwriter":    {CapDocumentation},
	"data_engineer": {CapDataEngineering, CapCoding},
	"leader":        {CapLeadership},
}

// DefaultCapabilitiesForRole returns the default capability strings for a role.
func DefaultCapabilitiesForRole(role string) []string {
	caps, ok := RoleCapabilities[role]
	if !ok {
		return nil
	}
	strs := make([]string, len(caps))
	for i, c := range caps {
		strs[i] = string(c)
	}
	return strs
}

// AgentCapability is a type alias for Capability. The name is kept for
// call-site readability when the value is consumed in an agent-type
// context (see AgentTypeCapabilities below). The row view used by stores
// and services is AgentCapabilityView — see agent.go.
type AgentCapability = Capability

// AgentTypeCapabilities maps an AgentType to its canonical default
// capability set. The 12 capability names referenced here are the
// Sprint 4 design-set (see comments on the CapRequirementAnalysis
// group above); they are not assignable in the task-assignment sense
// but they describe the per-type default skill profile. Lookups for
// unknown agent types return nil.
//
// This map is parallel to RoleCapabilities but keyed by AgentType
// (defined in agent_type.go) rather than by raw role string. Both maps
// exist because the Role field on the Agent struct is a free-form
// string (see agent.go) while the AgentType enum is a closed set of
// values used for routing and reporting.
var AgentTypeCapabilities = map[AgentType][]AgentCapability{
	AgentPM:       {CapRequirementAnalysis, CapTaskDecomposition},
	AgentArch:     {CapSystemDesign, CapAPIDesign},
	AgentDev:      {CapCodeImplementation},
	AgentReviewer: {CapCodeReview, CapSecurityScan},
	AgentQA:       {CapTestPlanning, CapTestExecution},
	AgentDevOps:   {CapCICD, CapDeployment, CapInfrastructure},
}

// DefaultCapabilitiesForType returns the default capability strings for
// an agent type, in the same []string form as DefaultCapabilitiesForRole
// so callers can use the two interchangeably. Returns nil for unknown
// agent types so tests and call sites can use the result directly.
func DefaultCapabilitiesForType(typeName string) []string {
	caps, ok := AgentTypeCapabilities[AgentType(typeName)]
	if !ok {
		return nil
	}
	strs := make([]string, len(caps))
	for i, c := range caps {
		strs[i] = string(c)
	}
	return strs
}
