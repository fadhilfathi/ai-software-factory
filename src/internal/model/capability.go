package model

// Capability is a named skill or competency an agent can possess.
type Capability string

const (
	CapArchitecture      Capability = "architecture"
	CapCoding            Capability = "coding"
	CapTesting           Capability = "testing"
	CapSecurity          Capability = "security"
	CapDocumentation     Capability = "documentation"
	CapDevOps            Capability = "devops"
	CapProjectMgmt       Capability = "project_management"
	CapDataEngineering   Capability = "data_engineering"
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

// RoleCapabilities maps agent role strings to their default capability set.
var RoleCapabilities = map[string][]Capability{
	"pm":             {CapProjectMgmt},
	"developer":      {CapCoding, CapTesting},
	"architect":      {CapArchitecture, CapCoding},
	"reviewer":       {CapTesting, CapSecurity},
	"qa":             {CapTesting},
	"security":       {CapSecurity},
	"devops":         {CapDevOps, CapArchitecture},
	"techwriter":     {CapDocumentation},
	"data_engineer":  {CapDataEngineering, CapCoding},
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
