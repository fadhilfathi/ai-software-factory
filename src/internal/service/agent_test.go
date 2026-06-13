package service

// Unit tests for AgentService (TASK-402, Sprint 4).
//
// Test strategy: drive the service through a real memory store so
// the integration is exercised end-to-end (capability existence
// check, soft-delete state machine, optimistic concurrency, etc.).
// The Postgres path is covered by an integration test in a follow-up
// sprint task (see infra-validation.md "Outstanding / deferred").

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestService wires a service backed by a fresh memory store.
// The 6 canonical capabilities are pre-seeded by NewMemoryStore.
func newTestService(t *testing.T) (AgentService, store.Store) {
	t.Helper()
	s := store.NewMemoryStore()
	return NewAgentService(s), s
}

func ptrStr(s string) *string    { return &s }
func ptrInt(i int) *int          { return &i }
func ptrStatus(s model.AgentStatus) *model.AgentStatus { return &s }

func TestAgentService_Create_Success(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	agent, apiErr := svc.CreateAgent(ctx, CreateAgentRequest{
		ProjectID:    uuid.New(),
		Name:         "alpha-001",
		Role:         "developer",
		Capabilities: []string{"coding", "testing"},
	})
	require.Nil(t, apiErr)
	require.NotNil(t, agent)

	assert.Equal(t, "alpha-001", agent.Name)
	assert.Equal(t, "developer", agent.Role)
	assert.Equal(t, model.AgentInitializing, agent.Status)
	assert.Equal(t, 1, agent.Version)
	assert.ElementsMatch(t, []string{"coding", "testing"}, agent.Capabilities)
	assert.False(t, agent.CreatedAt.IsZero())
}

func TestAgentService_Create_ValidationErrors(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	projectID := uuid.New()

	tests := []struct {
		name string
		req  CreateAgentRequest
		code string
	}{
		{
			name: "empty name",
			req:  CreateAgentRequest{ProjectID: projectID, Name: "  ", Role: "developer", Capabilities: []string{"coding"}},
			code: "VALIDATION_ERROR",
		},
		{
			name: "empty role",
			req:  CreateAgentRequest{ProjectID: projectID, Name: "alpha", Role: "", Capabilities: []string{"coding"}},
			code: "VALIDATION_ERROR",
		},
		{
			name: "no capabilities",
			req:  CreateAgentRequest{ProjectID: projectID, Name: "alpha", Role: "developer", Capabilities: []string{}},
			code: "VALIDATION_ERROR",
		},
		{
			name: "unknown capability",
			req:  CreateAgentRequest{ProjectID: projectID, Name: "alpha", Role: "developer", Capabilities: []string{"coding", "nope"}},
			code: "CAPABILITY_NOT_FOUND",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, apiErr := svc.CreateAgent(ctx, tc.req)
			require.NotNil(t, apiErr)
			assert.Equal(t, tc.code, apiErr.Code)
		})
	}
}

func TestAgentService_Create_AlreadyExists(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	projectID := uuid.New()

	first, apiErr := svc.CreateAgent(ctx, CreateAgentRequest{
		ProjectID: projectID, Name: "dup", Role: "developer", Capabilities: []string{"coding"},
	})
	require.Nil(t, apiErr)
	require.NotNil(t, first)

	// Same (project_id, name) collision in the same project.
	_, apiErr = svc.CreateAgent(ctx, CreateAgentRequest{
		ProjectID: projectID, Name: "dup", Role: "developer", Capabilities: []string{"coding"},
	})
	require.NotNil(t, apiErr)
	assert.Equal(t, "ALREADY_EXISTS", apiErr.Code)
}

func TestAgentService_Get_NotFound(t *testing.T) {
	svc, _ := newTestService(t)
	_, apiErr := svc.GetAgent(context.Background(), uuid.New(), uuid.New())
	require.NotNil(t, apiErr)
	assert.Equal(t, 404, apiErr.Status)
}

func TestAgentService_List_Pagination(t *testing.T) {
	svc, st := newTestService(t)
	ctx := context.Background()
	projectID := uuid.New()

	// Seed 7 agents in the same project; request 3 at a time.
	for i := 0; i < 7; i++ {
		_, err := svc.CreateAgent(ctx, CreateAgentRequest{
			ProjectID:    projectID,
			Name:         "agent-" + string(rune('a'+i)),
			Role:         "developer",
			Capabilities: []string{"coding"},
		})
		require.Nil(t, err)
	}

	page1, apiErr := svc.ListAgents(ctx, ListAgentsRequest{
		ProjectID: projectID,
		Limit:     3,
	})
	require.Nil(t, apiErr)
	assert.Len(t, page1.Data, 3)
	assert.True(t, page1.HasMore)
	assert.NotEmpty(t, page1.NextCursor)

	page2, apiErr := svc.ListAgents(ctx, ListAgentsRequest{
		ProjectID: projectID,
		Limit:     3,
		Cursor:    page1.NextCursor,
	})
	require.Nil(t, apiErr)
	assert.Len(t, page2.Data, 3)
	assert.True(t, page2.HasMore)

	page3, apiErr := svc.ListAgents(ctx, ListAgentsRequest{
		ProjectID: projectID,
		Limit:     3,
		Cursor:    page2.NextCursor,
	})
	require.Nil(t, apiErr)
	assert.Len(t, page3.Data, 1)
	assert.False(t, page3.HasMore)

	// IDs across pages are all distinct.
	seen := make(map[string]struct{}, 7)
	for _, p := range []*ListAgentsResult{page1, page2, page3} {
		for _, a := range p.Data {
			_, dup := seen[a.ID.String()]
			assert.False(t, dup, "duplicate id across pages: %s", a.ID)
			seen[a.ID.String()] = struct{}{}
		}
	}

	// Default exclude-retired behaviour: retired agents are not in
	// the list. Retire one and re-list.
	retireTarget := page1.Data[0].ID
	apiErr = svc.RetireAgent(ctx, retireTarget, projectID, false)
	require.Nil(t, apiErr)
	listed, apiErr := svc.ListAgents(ctx, ListAgentsRequest{ProjectID: projectID, Limit: 50})
	require.Nil(t, apiErr)
	for _, a := range listed.Data {
		assert.NotEqual(t, retireTarget, a.ID, "retired agent must not appear in default list")
	}

	// include_retired=true brings it back.
	included, apiErr := svc.ListAgents(ctx, ListAgentsRequest{ProjectID: projectID, Limit: 50, IncludeRetired: true})
	require.Nil(t, apiErr)
	found := false
	for _, a := range included.Data {
		if a.ID == retireTarget {
			found = true
		}
	}
	assert.True(t, found, "retired agent must appear when include_retired=true")

	_ = st
}

func TestAgentService_Update_OptimisticConcurrency(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	created, apiErr := svc.CreateAgent(ctx, CreateAgentRequest{
		ProjectID: uuid.New(), Name: "alpha", Role: "developer", Capabilities: []string{"coding"},
	})
	require.Nil(t, apiErr)

	// First update succeeds.
	updated, apiErr := svc.UpdateAgent(ctx, created.ID, created.ProjectID, UpdateAgentRequest{
		Role:    ptrStr("reviewer"),
		Version: ptrInt(created.Version),
	})
	require.Nil(t, apiErr)
	assert.Equal(t, "reviewer", updated.Role)
	assert.Equal(t, created.Version+1, updated.Version)

	// Re-using the OLD version is a 409 VERSION_CONFLICT.
	_, apiErr = svc.UpdateAgent(ctx, created.ID, created.ProjectID, UpdateAgentRequest{
		Role:    ptrStr("qa"),
		Version: ptrInt(created.Version),
	})
	require.NotNil(t, apiErr)
	assert.Equal(t, "VERSION_CONFLICT", apiErr.Code)
}

func TestAgentService_Update_RetiredAgent(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	created, apiErr := svc.CreateAgent(ctx, CreateAgentRequest{
		ProjectID: uuid.New(), Name: "alpha", Role: "developer", Capabilities: []string{"coding"},
	})
	require.Nil(t, apiErr)
	apiErr = svc.RetireAgent(ctx, created.ID, created.ProjectID, false)
	require.Nil(t, apiErr)

	_, apiErr = svc.UpdateAgent(ctx, created.ID, created.ProjectID, UpdateAgentRequest{
		Role:    ptrStr("reviewer"),
		Version: ptrInt(2),
	})
	require.NotNil(t, apiErr)
	assert.Equal(t, "ALREADY_RETIRED", apiErr.Code)
}

func TestAgentService_Update_CapabilityRewrite(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	created, apiErr := svc.CreateAgent(ctx, CreateAgentRequest{
		ProjectID: uuid.New(), Name: "alpha", Role: "developer", Capabilities: []string{"coding"},
	})
	require.Nil(t, apiErr)

	caps := []string{"coding", "testing", "security"}
	updated, apiErr := svc.UpdateAgent(ctx, created.ID, created.ProjectID, UpdateAgentRequest{
		Capabilities: &caps,
		Version:      ptrInt(created.Version),
	})
	require.Nil(t, apiErr)
	assert.ElementsMatch(t, caps, updated.Capabilities)
	assert.Greater(t, updated.Version, created.Version)
}

func TestAgentService_Retire_NotFound(t *testing.T) {
	svc, _ := newTestService(t)
	apiErr := svc.RetireAgent(context.Background(), uuid.New(), uuid.New(), false)
	require.NotNil(t, apiErr)
	assert.Equal(t, 404, apiErr.Status)
}

func TestAgentService_ListAgentCapabilities(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	created, apiErr := svc.CreateAgent(ctx, CreateAgentRequest{
		ProjectID: uuid.New(), Name: "alpha", Role: "developer",
		Capabilities: []string{"coding", "testing"},
	})
	require.Nil(t, apiErr)

	caps, apiErr := svc.ListAgentCapabilities(ctx, created.ID, created.ProjectID)
	require.Nil(t, apiErr)
	assert.Len(t, caps, 2)
	names := []string{caps[0].Name, caps[1].Name}
	assert.Contains(t, names, "coding")
	assert.Contains(t, names, "testing")
}

func TestAgentService_ListCapabilities_UnknownCategory(t *testing.T) {
	svc, _ := newTestService(t)
	_, apiErr := svc.ListCapabilities(context.Background(), ListCapabilitiesRequest{
		Category: "marketing",
	})
	require.NotNil(t, apiErr)
	assert.Equal(t, "VALIDATION_ERROR", apiErr.Code)
}

func TestAgentService_Create_MetadataDefault(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	agent, apiErr := svc.CreateAgent(ctx, CreateAgentRequest{
		ProjectID: uuid.New(), Name: "alpha", Role: "developer", Capabilities: []string{"coding"},
	})
	require.Nil(t, apiErr)
	assert.JSONEq(t, `{}`, string(agent.Metadata))
}

func TestAgentService_Create_CustomMetadata(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	agent, apiErr := svc.CreateAgent(ctx, CreateAgentRequest{
		ProjectID: uuid.New(), Name: "alpha", Role: "developer", Capabilities: []string{"coding"},
		Metadata: json.RawMessage(`{"region":"us-east-1"}`),
	})
	require.Nil(t, apiErr)
	assert.JSONEq(t, `{"region":"us-east-1"}`, string(agent.Metadata))
}

// ---- F-013 cross-tenant (Sprint 5) -----------------------------

// TestAgentService_Get_CrossTenant asserts that an authenticated
// caller from projectB cannot read an agent owned by projectA —
// 404 with code CROSS_TENANT_BLOCKED (security-review.md §5.1 F-013).
func TestAgentService_Get_CrossTenant(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	projectA := uuid.New()
	projectB := uuid.New()

	created, apiErr := svc.CreateAgent(ctx, CreateAgentRequest{
		ProjectID: projectA, Name: "alpha", Role: "developer", Capabilities: []string{"coding"},
	})
	require.Nil(t, apiErr)

	_, apiErr = svc.GetAgent(ctx, created.ID, projectB)
	require.NotNil(t, apiErr)
	assert.Equal(t, 404, apiErr.Status)
	assert.Equal(t, "CROSS_TENANT_BLOCKED", apiErr.Code)

	// control: same project works
	agent, apiErr := svc.GetAgent(ctx, created.ID, projectA)
	require.Nil(t, apiErr)
	assert.Equal(t, created.ID, agent.ID)
}

// TestAgentService_Update_CrossTenant: projectB caller cannot update
// projectA agent. Returns 404 CROSS_TENANT_BLOCKED, no state change.
func TestAgentService_Update_CrossTenant(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	projectA := uuid.New()
	projectB := uuid.New()

	created, apiErr := svc.CreateAgent(ctx, CreateAgentRequest{
		ProjectID: projectA, Name: "alpha", Role: "developer", Capabilities: []string{"coding"},
	})
	require.Nil(t, apiErr)

	role := ptrStr("reviewer")
	_, apiErr = svc.UpdateAgent(ctx, created.ID, projectB, UpdateAgentRequest{
		Role:    role,
		Version: ptrInt(created.Version),
	})
	require.NotNil(t, apiErr)
	assert.Equal(t, 404, apiErr.Status)
	assert.Equal(t, "CROSS_TENANT_BLOCKED", apiErr.Code)

	// control: re-read with projectA shows role unchanged
	agent, apiErr := svc.GetAgent(ctx, created.ID, projectA)
	require.Nil(t, apiErr)
	assert.Equal(t, "developer", agent.Role)
}

// TestAgentService_Retire_CrossTenant: projectB caller cannot delete
// projectA agent. Returns 404 CROSS_TENANT_BLOCKED, agent still active.
func TestAgentService_Retire_CrossTenant(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	projectA := uuid.New()
	projectB := uuid.New()

	created, apiErr := svc.CreateAgent(ctx, CreateAgentRequest{
		ProjectID: projectA, Name: "alpha", Role: "developer", Capabilities: []string{"coding"},
	})
	require.Nil(t, apiErr)

	apiErr = svc.RetireAgent(ctx, created.ID, projectB, false)
	require.NotNil(t, apiErr)
	assert.Equal(t, 404, apiErr.Status)
	assert.Equal(t, "CROSS_TENANT_BLOCKED", apiErr.Code)

	// control: agent is still active (not soft-deleted)
	agent, apiErr := svc.GetAgent(ctx, created.ID, projectA)
	require.Nil(t, apiErr)
	assert.NotEqual(t, model.AgentRetired, agent.Status)
}

// TestAgentService_ListAgentCapabilities_CrossTenant: projectB caller
// cannot enumerate projectA agent capabilities.
func TestAgentService_ListAgentCapabilities_CrossTenant(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	projectA := uuid.New()
	projectB := uuid.New()

	created, apiErr := svc.CreateAgent(ctx, CreateAgentRequest{
		ProjectID: projectA, Name: "alpha", Role: "developer", Capabilities: []string{"coding", "testing"},
	})
	require.Nil(t, apiErr)

	_, apiErr = svc.ListAgentCapabilities(ctx, created.ID, projectB)
	require.NotNil(t, apiErr)
	assert.Equal(t, 404, apiErr.Status)
	assert.Equal(t, "CROSS_TENANT_BLOCKED", apiErr.Code)

	// control: projectA caller sees the capabilities
	caps, apiErr := svc.ListAgentCapabilities(ctx, created.ID, projectA)
	require.Nil(t, apiErr)
	assert.Len(t, caps, 2)
}

// TestAgentService_MissingProjectHeader: zero UUID caller is rejected
// at the service layer with 400 MISSING_PROJECT_HEADER.
func TestAgentService_MissingProjectHeader(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	projectA := uuid.New()

	created, apiErr := svc.CreateAgent(ctx, CreateAgentRequest{
		ProjectID: projectA, Name: "alpha", Role: "developer", Capabilities: []string{"coding"},
	})
	require.Nil(t, apiErr)

	_, apiErr = svc.GetAgent(ctx, created.ID, uuid.Nil)
	require.NotNil(t, apiErr)
	assert.Equal(t, 400, apiErr.Status)
	assert.Equal(t, "MISSING_PROJECT_HEADER", apiErr.Code)
}
