package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// newTestCapabilityService wires a CapabilityService with a nil
// store. Use newTestCapabilityServiceWithStore (or a hand-rolled
// store mock) when the test needs to exercise the live-store
// validation seam (ValidateAgentHasCapabilities).
func newTestCapabilityService() *CapabilityService {
	return NewCapabilityService(nil, zap.NewNop())
}

// fixedGrantedAt is the timestamp used by the capability-grant
// fixtures in the TASK-403 ValidateAgentHasCapabilities tests. A
// fixed value keeps assertions deterministic without time.Now
// race-conditions.
var fixedGrantedAt = time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)

// --- Pre-Sprint-4 helper tests --------------------------------------
// These cover CapabilitiesForRole, TaskRequiresCapability,
// AgentHasCapability, FindCompatibleAgents, AssignmentScore and
// the model.Capability constants. They have not changed in TASK-403.

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
		model.CapLeadership,
	}
	assert.ElementsMatch(t, expected, caps)
}

func TestAssignableCapabilities_ExcludesLeadership(t *testing.T) {
	// Per the TASK-403 brief, the 5 assignable caps are the
	// public assignable surface for task-assignment constraints.
	// Leadership is in the catalog (migration 016 seed) but
	// reserved for the Leader and must never appear in a task's
	// required_capabilities list.
	caps := model.AssignableCapabilities()
	assert.ElementsMatch(t, []model.Capability{
		model.CapArchitecture, model.CapCoding, model.CapTesting,
		model.CapSecurity, model.CapDevOps,
	}, caps)
	for _, c := range caps {
		assert.NotEqual(t, model.CapLeadership, c, "leadership must not be in the assignable set")
	}
}

func TestIsAssignableCapability(t *testing.T) {
	assert.True(t, model.IsAssignableCapability("coding"))
	assert.True(t, model.IsAssignableCapability("testing"))
	assert.True(t, model.IsAssignableCapability("architecture"))
	assert.True(t, model.IsAssignableCapability("security"))
	assert.True(t, model.IsAssignableCapability("devops"))
	assert.False(t, model.IsAssignableCapability("leadership"))
	assert.False(t, model.IsAssignableCapability("documentation"))
	assert.False(t, model.IsAssignableCapability("project_management"))
	assert.False(t, model.IsAssignableCapability("data_engineering"))
	assert.False(t, model.IsAssignableCapability("unknown"))
	assert.False(t, model.IsAssignableCapability(""))
}

func TestValidCapability_Valid(t *testing.T) {
	assert.True(t, model.ValidCapability("coding"))
	assert.True(t, model.ValidCapability("security"))
	assert.True(t, model.ValidCapability("devops"))
	assert.True(t, model.ValidCapability("leadership"))
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
		{"leader", []model.Capability{model.CapLeadership}},
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

// --- TASK-403: ValidateAgentHasCapabilities tests -------------------

// mockCapabilityStore is a hand-rolled mock for store.Store that
// exposes only the surface ValidateAgentHasCapabilities touches:
// Agents().ListCapabilitiesByAgent. The mock returns canned
// capability grants for a given agent ID.
type mockCapabilityStore struct {
	store.Store
	agents *mockCapabilityAgentStore
}

func (m *mockCapabilityStore) Agents() store.AgentStore { return m.agents }

type mockCapabilityAgentStore struct {
	store.AgentStore
	grants     map[uuid.UUID][]*model.AgentCapabilityView
	errOnRead  error
	notFound   bool
	readCalled int
	lastAgent  uuid.UUID
}

func (m *mockCapabilityAgentStore) ListCapabilitiesByAgent(ctx context.Context, agentID uuid.UUID) ([]*model.AgentCapabilityView, error) {
	m.readCalled++
	m.lastAgent = agentID
	if m.notFound {
		return nil, store.ErrNotFound
	}
	if m.errOnRead != nil {
		return nil, m.errOnRead
	}
	return m.grants[agentID], nil
}

func newCapAgentStore(grants map[uuid.UUID][]*model.AgentCapabilityView) *mockCapabilityAgentStore {
	return &mockCapabilityAgentStore{grants: grants}
}

func newMockCapStore(grants map[uuid.UUID][]*model.AgentCapabilityView) *mockCapabilityStore {
	agents := newCapAgentStore(grants)
	return &mockCapabilityStore{agents: agents}
}

func TestValidateAgentHasCapabilities_AllPresent(t *testing.T) {
	agentID := uuid.New()
	grants := map[uuid.UUID][]*model.AgentCapabilityView{
		agentID: {
			{Name: "coding", Category: "coding", GrantedAt: fixedGrantedAt},
			{Name: "testing", Category: "testing", GrantedAt: fixedGrantedAt},
		},
	}
	s := NewCapabilityService(newMockCapStore(grants), zap.NewNop())
	err := s.ValidateAgentHasCapabilities(context.Background(), agentID, []string{"coding", "testing"})
	assert.NoError(t, err)
}

func TestValidateAgentHasCapabilities_OneMissing(t *testing.T) {
	agentID := uuid.New()
	grants := map[uuid.UUID][]*model.AgentCapabilityView{
		agentID: {
			{Name: "coding", Category: "coding", GrantedAt: fixedGrantedAt},
		},
	}
	s := NewCapabilityService(newMockCapStore(grants), zap.NewNop())
	err := s.ValidateAgentHasCapabilities(context.Background(), agentID, []string{"coding", "testing"})
	assert.Error(t, err, "missing 'testing' must surface an error")

	var svcErr *Error
	if assert.True(t, errors.As(err, &svcErr), "must be a *Error") {
		assert.Equal(t, "CAPABILITY_MISMATCH", string(svcErr.Code))
		assert.Equal(t, 409, svcErr.Status)
		// details must include the missing capability name
		if assert.NotNil(t, svcErr.Details) {
			found := false
			for _, d := range svcErr.Details {
				if d.Field == "required_capabilities" && strings.Contains(d.Message, "testing") {
					found = true
					break
				}
			}
			assert.True(t, found, "details should mention the missing 'testing' cap")
		}
	}
}

func TestValidateAgentHasCapabilities_EmptyList(t *testing.T) {
	agentID := uuid.New()
	grants := map[uuid.UUID][]*model.AgentCapabilityView{
		agentID: {
			{Name: "coding", Category: "coding", GrantedAt: fixedGrantedAt},
		},
	}
	// Track whether the store was consulted. It must NOT be
	// consulted for an empty required list.
	agents := newCapAgentStore(grants)
	s := NewCapabilityService(&mockCapabilityStore{agents: agents}, zap.NewNop())
	err := s.ValidateAgentHasCapabilities(context.Background(), agentID, []string{})
	assert.NoError(t, err)
	assert.Equal(t, 0, agents.readCalled, "store must not be read for empty required list")

	// nil slice behaves the same way.
	err = s.ValidateAgentHasCapabilities(context.Background(), agentID, nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, agents.readCalled, "store must not be read for nil required list")
}

func TestValidateAgentHasCapabilities_AgentNotFound(t *testing.T) {
	agentID := uuid.New()
	agents := newCapAgentStore(nil)
	agents.notFound = true
	s := NewCapabilityService(&mockCapabilityStore{agents: agents}, zap.NewNop())
	err := s.ValidateAgentHasCapabilities(context.Background(), agentID, []string{"coding"})
	assert.Error(t, err)

	var svcErr *Error
	if assert.True(t, errors.As(err, &svcErr), "must be a *Error") {
		assert.Equal(t, "NOT_FOUND", string(svcErr.Code))
		assert.Equal(t, 404, svcErr.Status)
	}
}

func TestValidateAgentHasCapabilities_StoreError(t *testing.T) {
	agentID := uuid.New()
	agents := newCapAgentStore(nil)
	agents.errOnRead = errors.New("connection reset")
	s := NewCapabilityService(&mockCapabilityStore{agents: agents}, zap.NewNop())
	err := s.ValidateAgentHasCapabilities(context.Background(), agentID, []string{"coding"})
	assert.Error(t, err)

	// Store errors collapse to a generic INTERNAL 500 envelope so
	// we don't leak driver-level error strings to clients.
	var svcErr *Error
	if assert.True(t, errors.As(err, &svcErr), "must be a *Error") {
		assert.Equal(t, "INTERNAL", string(svcErr.Code))
		assert.Equal(t, 500, svcErr.Status)
	}
}

func TestValidateAgentHasCapabilities_PassesContext(t *testing.T) {
	// Confirms the context.Context parameter is plumbed through
	// to the store call (defensive — guards against future
	// refactors accidentally dropping it).
	agentID := uuid.New()
	grants := map[uuid.UUID][]*model.AgentCapabilityView{
		agentID: {{Name: "coding", Category: "coding", GrantedAt: fixedGrantedAt}},
	}
	agents := newCapAgentStore(grants)
	s := NewCapabilityService(&mockCapabilityStore{agents: agents}, zap.NewNop())
	err := s.ValidateAgentHasCapabilities(context.Background(), agentID, []string{"coding"})
	assert.NoError(t, err)
	assert.Equal(t, 1, agents.readCalled)
	assert.Equal(t, agentID, agents.lastAgent)
}
