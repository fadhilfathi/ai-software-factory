package integration_test

// T1.cross — 6 cross-tenant negative tests (D-003 CT1..CT6).
// See docs/reset/workflow-e2e-spec.md §2 for the contract.
//
// Replays F-013/14/15/16 from docs/sprint4/security-report.md and
// F-D002-004 from docs/reset/security-review.md. Verifies the
// service-layer path-implied fix (TASK-419..422) holds at the wire
// level: an attacker in project B cannot read or write resources
// in project A.
//
// Setup: two phantom project_ids (A = victim, B = attacker), one
// user (test middleware bypasses auth). The attacker uses
// X-Project-ID=B on every request.

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// crossTenantEnv wires two project_ids and a victim (project A)
// agent + task, so each CT subtest can target a specific
// cross-tenant hole.
type crossTenantEnv struct {
	router    *IntegrationTestEnv
	projectA  string
	projectB  string
	agentA    uuid.UUID
	agentB    uuid.UUID
	taskA     uuid.UUID
	assignA   uuid.UUID
}

func newCrossTenantEnv(t *testing.T) crossTenantEnv {
	t.Helper()
	r := newIntegrationRouter(t, store.NewMemoryStore())
	projectA := r.ProjectID
	// Pre-create projectB in the same store so the cross-tenant attacker
	// setup can create an agent + task under B. The earlier random uuid
	// never hit s.Projects().Create() and 404'd on every handler call.
	projectBUUID := uuid.New()
	ownerUUID, _ := uuid.Parse("11111111-1111-1111-1111-111111111111")
	projB := &model.Project{
		ID:      projectBUUID,
		Name:    "Test Project B",
		OwnerID: ownerUUID,
		Status:  model.ProjectInProgress,
	}
	if err := r.Store.Projects().Create(projB); err != nil {
		t.Fatalf("setup: pre-create projectB: %v", err)
	}
	projectB := projectBUUID.String()

	// Create victim agent in A.
	w := doRequest(t, r, http.MethodPost, "/v1/agents", projectA,
		map[string]any{
			"project_id":   projectA,
			"name":         "CT victim agent",
			"role":         "developer",
			"capabilities": []string{"coding"},
		})
	require.Equal(t, http.StatusCreated, w.Code,
		"setup: create agent in A: %s", w.Body.String())
	var agentA model.Agent
	require.NoError(t, json.Unmarshal(parseData(t, w), &agentA))

	// Create attacker agent in B.
	w = doRequest(t, r, http.MethodPost, "/v1/agents", projectB,
		map[string]any{
			"project_id":   projectB,
			"name":         "CT attacker agent",
			"role":         "developer",
			"capabilities": []string{"coding"},
		})
	require.Equal(t, http.StatusCreated, w.Code,
		"setup: create agent in B: %s", w.Body.String())
	var agentB model.Agent
	require.NoError(t, json.Unmarshal(parseData(t, w), &agentB))

	// Create victim task in A.
	w = doRequest(t, r, http.MethodPost,
		"/v1/projects/"+projectA+"/tasks", "",
		map[string]any{
			"title":    "CT victim task",
			"priority": "high",
		})
	require.Equal(t, http.StatusCreated, w.Code,
		"setup: create task in A: %s", w.Body.String())
	var taskA model.Task
	require.NoError(t, json.Unmarshal(parseData(t, w), &taskA))

	// Assign the victim task to the victim agent (legit flow
	// within project A). Capture the assignment ID for CT3.
	w = doRequest(t, r, http.MethodPost,
		"/v1/tasks/"+taskA.ID.String()+"/assign", "",
		map[string]any{
			"agent_id":              agentA.ID.String(),
			"capabilities_required": []string{"coding"},
		})
	require.Equal(t, http.StatusOK, w.Code,
		"setup: assign task in A: %s", w.Body.String())
	var assignResp struct {
		Assignment model.Assignment `json:"assignment"`
	}
	require.NoError(t, json.Unmarshal(parseData(t, w), &assignResp))

	return crossTenantEnv{
		router:   r,
		projectA: projectA,
		projectB: projectB,
		agentA:   agentA.ID,
		agentB:   agentB.ID,
		taskA:    taskA.ID,
		assignA:  assignResp.Assignment.ID,
	}
}

// TestIntegration_Workflow_CrossTenant_Blocks is the cross-tenant
// negative test suite. Each sub-test sets X-Project-ID to the
// attacker's project (B) and attempts an operation against a
// resource owned by project A. The expected response is a 404
// (or 200-with-empty-data for list endpoints), NOT a data leak.
//
// F-013/14/15/16 from Sprint 4 and F-D002-004 from D-002 are
// re-validated at the wire level.
func TestIntegration_Workflow_CrossTenant_Blocks(t *testing.T) {
	env := newCrossTenantEnv(t)
	projectA := env.projectA
	projectB := env.projectB

	// CT1 — Attacker (B) reads agent in A. Expect 404.
	// Replays F-013 (cross-tenant agent read).
	t.Run("CT1_cross_tenant_agent_read", func(t *testing.T) {
		w := doRequest(t, env.router, http.MethodGet,
			"/v1/agents/"+env.agentA.String(), projectB, nil)
		require.Equal(t, http.StatusNotFound, w.Code,
			"CT1: GET agent from project A with X-Project-ID=B must 404, got %d: %s",
			w.Code, w.Body.String())
		// Also confirm the victim agent in A is still
		// accessible to its owner (sanity check).
		wOk := doRequest(t, env.router, http.MethodGet,
			"/v1/agents/"+env.agentA.String(), projectA, nil)
		require.Equal(t, http.StatusOK, wOk.Code,
			"CT1 sanity: GET agent from project A with X-Project-ID=A must 200, got %d: %s",
			wOk.Code, wOk.Body.String())
	})

	// CT2 — Attacker (B) updates agent in A. Expect 404.
	// Replays F-014 (cross-tenant agent write).
	t.Run("CT2_cross_tenant_agent_update", func(t *testing.T) {
		w := doRequest(t, env.router, http.MethodPut,
			"/v1/agents/"+env.agentA.String(), projectB,
			map[string]any{
				"capabilities": []string{"hacked"},
				"version":      1,
			})
		require.Equal(t, http.StatusNotFound, w.Code,
			"CT2: PUT agent from project A with X-Project-ID=B must 404, got %d: %s",
			w.Code, w.Body.String())
		// Sanity: the agent's capabilities are unchanged.
		wOk := doRequest(t, env.router, http.MethodGet,
			"/v1/agents/"+env.agentA.String(), projectA, nil)
		var agent model.Agent
		require.NoError(t, json.Unmarshal(parseData(t, wOk), &agent))
		assert.NotContains(t, agent.Capabilities, "hacked",
			"CT2 sanity: attacker must not have been able to inject 'hacked' capability")
	})

	// CT3 — Attacker (B) tries to assign a task in A to an
	// agent in B. Service must reject because task.ProjectID
	// (A) != callerProjectID (B). Replays F-015.
	t.Run("CT3_cross_tenant_assign", func(t *testing.T) {
		w := doRequest(t, env.router, http.MethodPost,
			"/v1/tasks/"+env.taskA.String()+"/assign", projectB,
			map[string]any{
				"agent_id":              env.agentB.String(),
				"capabilities_required": []string{"coding"},
				"notes":                 "CT3 cross-tenant attempt",
			})
		require.Equal(t, http.StatusNotFound, w.Code,
			"CT3: assign task in A from X-Project-ID=B must 404, got %d: %s",
			w.Code, w.Body.String())
		// Sanity: history of the victim task is unchanged.
		wOk := doRequest(t, env.router, http.MethodGet,
			"/v1/tasks/"+env.taskA.String()+"/history", "", nil)
		var history []model.AssignmentEvent
		require.NoError(t, json.Unmarshal(parseData(t, wOk), &history))
		assert.Len(t, history, 1, "CT3 sanity: only the legit setup event must remain")
	})

	// CT4 — Attacker (B) creates an execution for a task in A.
	// Service must reject because the task belongs to A. Replays
	// F-D002-004 (the execution-create surface in D-002).
	t.Run("CT4_cross_tenant_execution_create", func(t *testing.T) {
		w := doRequest(t, env.router, http.MethodPost,
			"/v1/executions", projectB,
			map[string]any{
				"task_id":  env.taskA.String(),
				"agent_id": env.agentB.String(),
			})
		require.Equal(t, http.StatusNotFound, w.Code,
			"CT4: create execution for task in A from X-Project-ID=B must 404, got %d: %s",
			w.Code, w.Body.String())
	})

	// CT5 — Attacker (B) creates a deliverable for a task in A
	// (from an agent in A). Service must reject. Replays F-016.
	t.Run("CT5_cross_tenant_deliverable_create", func(t *testing.T) {
		w := doRequest(t, env.router, http.MethodPost,
			"/v1/deliverables", projectB,
			map[string]any{
				"task_id":  env.taskA.String(),
				"agent_id": env.agentA.String(),
				"title":    "CT5 cross-tenant attempt",
				"content":  "should not be persisted",
			})
		require.Equal(t, http.StatusNotFound, w.Code,
			"CT5: create deliverable for task in A from X-Project-ID=B must 404, got %d: %s",
			w.Code, w.Body.String())
	})

	// CT6 — Attacker (B) lists executions for a task in A. The
	// list endpoint filters by callerProjectID, so the response
	// must be 200 with data: [] (empty), NOT a data leak. This
	// is the LIST variant of F-013.
	t.Run("CT6_cross_tenant_executions_list_empty", func(t *testing.T) {
		// First, a sanity check: create a legit execution in A
		// and confirm the A owner can see it.
		w := doRequest(t, env.router, http.MethodPost,
			"/v1/executions", "",
			map[string]any{
				"task_id":  env.taskA.String(),
				"agent_id": env.agentA.String(),
			})
		require.Equal(t, http.StatusCreated, w.Code,
			"CT6 setup: create execution in A: %s", w.Body.String())

		// Owner in A: must see 1 execution.
		wOwner := doRequest(t, env.router, http.MethodGet,
			"/v1/executions?task_id="+env.taskA.String(), "", nil)
		require.Equal(t, http.StatusOK, wOwner.Code,
			"CT6 sanity owner: %s", wOwner.Body.String())
		var ownerList []model.Execution
		require.NoError(t, json.Unmarshal(parseData(t, wOwner), &ownerList))
		assert.Len(t, ownerList, 1, "CT6 sanity: owner must see 1 execution")

		// Attacker in B: must see an EMPTY list (not the
		// victim's execution).
		wAttacker := doRequest(t, env.router, http.MethodGet,
			"/v1/executions?task_id="+env.taskA.String(), projectB, nil)
		require.Equal(t, http.StatusOK, wAttacker.Code,
			"CT6 attacker: %s", wAttacker.Body.String())
		var attackerList []model.Execution
		require.NoError(t, json.Unmarshal(parseData(t, wAttacker), &attackerList))
		assert.Empty(t, attackerList,
			"CT6: attacker in B must see 0 executions for task in A (data leak otherwise)")
	})
}
