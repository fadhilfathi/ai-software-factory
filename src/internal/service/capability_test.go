package service

import (
	"testing"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func newTestCapabilityService() *CapabilityService {
	return NewCapabilityService()
}

func TestTaskRequiresCapability(t *testing.T) {
	svc := newTestCapabilityService()

	tests := []struct {
		taskType string
		expected []string
	}{
		{"feature", []string{"coding", "testing"}},
		{"implementation", []string{"coding", "testing"}},
		{"architecture", []string{"architecture"}},
		{"design", []string{"architecture"}},
		{"bugfix", []string{"coding"}},
		{"review", []string{"testing", "security"}},
		{"test", []string{"testing"}},
		{"qa", []string{"testing"}},
		{"security_audit", []string{"security"}},
		{"deployment", []string{"devops", "architecture"}},
		{"infrastructure", []string{"devops", "architecture"}},
		{"documentation", []string{"documentation"}},
		{"data_pipeline", []string{"data_engineering", "coding"}},
		{"analytics", []string{"data_engineering", "coding"}},
		{"project_management", []string{"project_management"}},
		{"planning", []string{"project_management"}},
		{"unknown_type", []string{"coding"}},
	}

	for _, tt := range tests {
		t.Run(tt.taskType, func(t *testing.T) {
			result := svc.TaskRequiresCapability(tt.taskType)
			assert.ElementsMatch(t, tt.expected, result, "taskType=%q", tt.taskType)
		})
	}
}

func TestAgentHasCapability_AllMatch(t *testing.T) {
	svc := newTestCapabilityService()
	agent := &model.Agent{
		Capabilities: []string{"coding", "testing", "security"},
	}
	assert.True(t, svc.AgentHasCapability(agent, []string{"coding", "testing"}))
}

func TestAgentHasCapability_MissingOne(t *testing.T) {
	svc := newTestCapabilityService()
	agent := &model.Agent{
		Capabilities: []string{"coding"},
	}
	assert.False(t, svc.AgentHasCapability(agent, []string{"coding", "testing"}))
}

func TestAgentHasCapability_EmptyRequired(t *testing.T) {
	svc := newTestCapabilityService()
	agent := &model.Agent{
		Capabilities: nil,
	}
	assert.True(t, svc.AgentHasCapability(agent, []string{}))
}

func TestAgentHasCapability_NoAgentCaps(t *testing.T) {
	svc := newTestCapabilityService()
	agent := &model.Agent{
		Capabilities: []string{},
	}
	assert.False(t, svc.AgentHasCapability(agent, []string{"coding"}))
}

func TestFindCompatibleAgents_AllRequired(t *testing.T) {
	svc := newTestCapabilityService()
	agents := []*model.Agent{
		{ID: uuid.New(), Capabilities: []string{"coding", "testing"}},
		{ID: uuid.New(), Capabilities: []string{"coding"}},
		{ID: uuid.New(), Capabilities: []string{"security"}},
	}

	result := svc.FindCompatibleAgents(agents, []string{"coding", "testing"})
	assert.Len(t, result, 1)
	assert.Equal(t, agents[0].ID, result[0].ID)
}

func TestFindCompatibleAgents_NoMatch(t *testing.T) {
	svc := newTestCapabilityService()
	agents := []*model.Agent{
		{ID: uuid.New(), Capabilities: []string{"security"}},
		{ID: uuid.New(), Capabilities: []string{"documentation"}},
	}

	result := svc.FindCompatibleAgents(agents, []string{"coding", "testing"})
	assert.Len(t, result, 0)
}

func TestFindCompatibleAgents_EmptyRequired(t *testing.T) {
	svc := newTestCapabilityService()
	agents := []*model.Agent{
		{ID: uuid.New(), Capabilities: []string{"coding"}},
		{ID: uuid.New(), Capabilities: []string{"testing"}},
	}

	result := svc.FindCompatibleAgents(agents, []string{})
	assert.Len(t, result, 2)
}

func TestFindCompatibleAgents_NilSlice(t *testing.T) {
	svc := newTestCapabilityService()
	result := svc.FindCompatibleAgents(nil, []string{"coding"})
	assert.Nil(t, result)
}

func TestAssignmentScore_PerfectMatch(t *testing.T) {
	svc := newTestCapabilityService()
	agent := &model.Agent{
		Capabilities: []string{"coding", "testing"},
	}
	score := svc.AssignmentScore(agent, []string{"coding", "testing"})
	// 2 matches × +2 = 4, 0 extras = 0, total = 4
	assert.Equal(t, 4, score)
}

func TestAssignmentScore_ExtraCaps(t *testing.T) {
	svc := newTestCapabilityService()
	agent := &model.Agent{
		Capabilities: []string{"coding", "testing", "security", "devops"},
	}
	score := svc.AssignmentScore(agent, []string{"coding", "testing"})
	// 2 matches × +2 = 4, 2 extras × +1 = 2, total = 6
	assert.Equal(t, 6, score)
}

func TestAssignmentScore_MissingCap(t *testing.T) {
	svc := newTestCapabilityService()
	agent := &model.Agent{
		Capabilities: []string{"coding"},
	}
	score := svc.AssignmentScore(agent, []string{"coding", "testing"})
	// 1 match × +2 = 2, 1 missing × -5 = -5, total = -3
	assert.Equal(t, -3, score)
}

func TestAssignmentScore_EmptyRequired(t *testing.T) {
	svc := newTestCapabilityService()
	agent := &model.Agent{
		Capabilities: []string{"coding", "testing"},
	}
	score := svc.AssignmentScore(agent, []string{})
	// 2 extras × +1 = 2
	assert.Equal(t, 2, score)
}

func TestAssignmentScore_NoCaps(t *testing.T) {
	svc := newTestCapabilityService()
	agent := &model.Agent{
		Capabilities: []string{},
	}
	score := svc.AssignmentScore(agent, []string{"coding"})
	// 0 matches, 1 missing × -5 = -5, 0 extras = 0, total = -5
	assert.Equal(t, -5, score)
}

// --- Capability Model Tests ---

func TestAllCapabilities_ContainsAll(t *testing.T) {
	caps := model.AllCapabilities()
	expected := []model.Capability{
		model.CapArchitecture,
		model.CapCoding,
		model.CapTesting,
		model.CapSecurity,
		model.CapDocumentation,
		model.CapDevOps,
		model.CapProjectMgmt,
		model.CapDataEngineering,
	}
	assert.ElementsMatch(t, expected, caps)
}

func TestValidCapability_Valid(t *testing.T) {
	assert.True(t, model.ValidCapability("coding"))
	assert.True(t, model.ValidCapability("security"))
	assert.True(t, model.ValidCapability("devops"))
}

func TestValidCapability_Invalid(t *testing.T) {
	assert.False(t, model.ValidCapability("unknown"))
	assert.False(t, model.ValidCapability(""))
	assert.False(t, model.ValidCapability("CODING"))
}

func TestRoleCapabilities_AllRoles(t *testing.T) {
	tests := []struct {
		role     string
		expected []model.Capability
	}{
		{"pm", []model.Capability{model.CapProjectMgmt}},
		{"developer", []model.Capability{model.CapCoding, model.CapTesting}},
		{"architect", []model.Capability{model.CapArchitecture, model.CapCoding}},
		{"reviewer", []model.Capability{model.CapTesting, model.CapSecurity}},
		{"qa", []model.Capability{model.CapTesting}},
		{"security", []model.Capability{model.CapSecurity}},
		{"devops", []model.Capability{model.CapDevOps, model.CapArchitecture}},
		{"techwriter", []model.Capability{model.CapDocumentation}},
		{"data_engineer", []model.Capability{model.CapDataEngineering, model.CapCoding}},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			caps, ok := model.RoleCapabilities[tt.role]
			assert.True(t, ok, "role %q should have capabilities", tt.role)
			assert.ElementsMatch(t, tt.expected, caps)
		})
	}
}

func TestDefaultCapabilitiesForRole_Strings(t *testing.T) {
	caps := model.DefaultCapabilitiesForRole("developer")
	assert.Equal(t, []string{"coding", "testing"}, caps)
}

func TestDefaultCapabilitiesForRole_Unknown(t *testing.T) {
	caps := model.DefaultCapabilitiesForRole("unknown_role")
	assert.Nil(t, caps)
}
