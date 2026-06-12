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
