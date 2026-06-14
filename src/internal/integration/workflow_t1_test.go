package integration_test

// T1 — full 15-step happy-path E2E flow (Sprint 6 D-003).
// See docs/reset/workflow-e2e-spec.md §1 for the contract.
//
// This file is the SCAFFOLDING commit. Each of the 15 steps is a
// t.Run subtest that performs the real HTTP call against the wired
// router + service + in-memory store + aion MockRuntime. The asserts
// in this commit cover the must-pass checks (status codes, basic
// field presence, state-machine transitions, version increment).
// The deep table-driven coverage in commit 4 (D-003-NN) will
// extend each subtest with the full per-step matrix.
//
// Go toolchain is not installed on this host; CI is the source of
// truth for "go test -race -shuffle=on ./internal/integration/..."

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_Workflow_T1_HappyPath exercises the full
// Project → Task → Assignment → Execution → Deliverable → Done flow
// end-to-end, sub-step by sub-step, in sequence. Each sub-step
// runs in its own t.Run so a single failure points to the
// specific hop.
//
// The harness uses a fixed user_id (test middleware) and a
// single phantom project_id (no projects table). X-Project-ID
// is set on project-scoped reads/writes per the handler's
// `projectIDFromContext` priority rules.
func TestIntegration_Workflow_T1_HappyPath(t *testing.T) {
	env := newIntegrationRouter(t, store.NewMemoryStore())
	projectID := env.ProjectID

	// Captured across sub-steps for sequential flow.
	var (
		agentID  uuid.UUID
		taskID   uuid.UUID
		execID   uuid.UUID
		delivID  uuid.UUID
		firstAssignID uuid.UUID
	)

	// Step 1.1 — POST /v1/agents. Create the agent that will run
	// the lifecycle. Capabilities must include "coding" so the
	// assignment match in step 1.6 succeeds.
	t.Run("1.1_create_agent", func(t *testing.T) {
		w := doRequest(t, env, http.MethodPost, "/v1/agents", projectID,
			map[string]any{
				"project_id":   projectID,
				"name":         "T1 Lifecycle Agent",
				"role":         "developer",
				"capabilities": []string{"coding", "testing", "review"},
			})
		require.Equal(t, http.StatusCreated, w.Code,
			"1.1 create agent: %s", w.Body.String())
		var agent model.Agent
		require.NoError(t, json.Unmarshal(parseData(t, w), &agent))
		agentID = agent.ID
		require.NotEqual(t, uuid.Nil, agentID, "agent.ID must be set")
		assert.Equal(t, "T1 Lifecycle Agent", agent.Name, "name echo")
		assert.Equal(t, "developer", agent.Role, "role echo")
		// 3 capabilities sent, 3 expected back.
		assert.Len(t, agent.Capabilities, 3, "capability count")
	})

	// Step 1.2 — GET /v1/agents/:id. The agent must be
	// retrievable by ID from the same project.
	t.Run("1.2_get_agent", func(t *testing.T) {
		w := doRequest(t, env, http.MethodGet, "/v1/agents/"+agentID.String(), projectID, nil)
		require.Equal(t, http.StatusOK, w.Code,
			"1.2 get agent: %s", w.Body.String())
		var agent model.Agent
		require.NoError(t, json.Unmarshal(parseData(t, w), &agent))
		assert.Equal(t, "T1 Lifecycle Agent", agent.Name)
		assert.Equal(t, agentID, agent.ID)
	})

	// Step 1.3 — PUT /v1/agents/:id. Replace the agent's
	// capabilities. Requires optimistic-locking `version` field
	// in the request body. Version must increment.
	t.Run("1.3_update_agent_capabilities", func(t *testing.T) {
		// Read current version first.
		w := doRequest(t, env, http.MethodGet, "/v1/agents/"+agentID.String(), projectID, nil)
		var current model.Agent
		require.NoError(t, json.Unmarshal(parseData(t, w), &current))

		// Replace capabilities (add "deployment").
		newCaps := []string{"coding", "testing", "review", "deployment"}
		w = doRequest(t, env, http.MethodPut, "/v1/agents/"+agentID.String(), projectID,
			map[string]any{
				"capabilities": newCaps,
				"version":      current.Version,
			})
		require.Equal(t, http.StatusOK, w.Code,
			"1.3 update caps: %s", w.Body.String())
		var updated model.Agent
		require.NoError(t, json.Unmarshal(parseData(t, w), &updated))
		assert.Equal(t, current.Version+1, updated.Version, "version must increment by 1")
		assert.Len(t, updated.Capabilities, 4, "capability count after add")
	})

	// Step 1.4 — GET /v1/agents/:id/capabilities. Returns the
	// row view (display name, category, etc.) for the 4
	// capabilities set in 1.3.
	t.Run("1.4_list_agent_capabilities", func(t *testing.T) {
		w := doRequest(t, env, http.MethodGet,
			"/v1/agents/"+agentID.String()+"/capabilities", projectID, nil)
		require.Equal(t, http.StatusOK, w.Code,
			"1.4 list caps: %s", w.Body.String())
		var caps []model.AgentCapabilityView
		require.NoError(t, json.Unmarshal(parseData(t, w), &caps))
		assert.Len(t, caps, 4, "should have 4 capability rows")
		// Spot-check the new "deployment" capability is present
		// with its display name.
		var found bool
		for _, c := range caps {
			if c.Name == "deployment" {
				found = true
				assert.NotEmpty(t, c.DisplayName, "display_name must be set for deployment")
				assert.NotEmpty(t, c.Category, "category must be set for deployment")
			}
		}
		assert.True(t, found, "deployment capability should be in the list")
	})

	// Step 1.5 — POST /v1/projects/:id/tasks. Create the task
	// that the agent will work on. Default priority is `medium`
	// per the handler.
	t.Run("1.5_create_task", func(t *testing.T) {
		w := doRequest(t, env, http.MethodPost,
			"/v1/projects/"+projectID+"/tasks", "",
			map[string]any{
				"title":       "T1 Lifecycle Task",
				"description": "Full lifecycle validation task for D-003",
				"priority":    "high",
			})
		require.Equal(t, http.StatusCreated, w.Code,
			"1.5 create task: %s", w.Body.String())
		var task model.Task
		require.NoError(t, json.Unmarshal(parseData(t, w), &task))
		taskID = task.ID
		require.NotEqual(t, uuid.Nil, taskID, "task.ID must be set")
		assert.Equal(t, "T1 Lifecycle Task", task.Title)
	})

	// Step 1.6 — POST /v1/tasks/:id/assign. First assign, must
	// be idempotent=false. The service creates an Assignment row
	// (status=active) and an AssignmentEvent row (action=assign).
	t.Run("1.6_assign_task", func(t *testing.T) {
		w := doRequest(t, env, http.MethodPost,
			"/v1/tasks/"+taskID.String()+"/assign", "",
			map[string]any{
				"agent_id":              agentID.String(),
				"capabilities_required": []string{"coding"},
				"notes":                 "T1 first assignment",
			})
		require.Equal(t, http.StatusOK, w.Code,
			"1.6 first assign: %s", w.Body.String())
		var resp struct {
			Assignment model.Assignment     `json:"assignment"`
			Event      model.AssignmentEvent `json:"event"`
			Idempotent bool                  `json:"idempotent"`
		}
		require.NoError(t, json.Unmarshal(parseData(t, w), &resp))
		assert.False(t, resp.Idempotent, "first assign must not be idempotent")
		assert.Equal(t, "active", string(resp.Assignment.Status),
			"assignment should be active after first assign")
		assert.Equal(t, agentID, resp.Assignment.AgentID)
		firstAssignID = resp.Assignment.ID
		require.NotEqual(t, uuid.Nil, firstAssignID, "assignment.ID must be set")
	})

	// Step 1.7 — Re-POST the same assign. The service detects
	// the (task_id, agent_id, capabilities) tuple is unchanged
	// and returns the existing assignment with idempotent=true.
	// No new AssignmentEvent is written.
	t.Run("1.7_idempotent_reassign", func(t *testing.T) {
		w := doRequest(t, env, http.MethodPost,
			"/v1/tasks/"+taskID.String()+"/assign", "",
			map[string]any{
				"agent_id":              agentID.String(),
				"capabilities_required": []string{"coding"},
				"notes":                 "T1 re-assignment (should be idempotent)",
			})
		require.Equal(t, http.StatusOK, w.Code,
			"1.7 reassign: %s", w.Body.String())
		var resp struct {
			Assignment model.Assignment     `json:"assignment"`
			Event      model.AssignmentEvent `json:"event"`
			Idempotent bool                  `json:"idempotent"`
		}
		require.NoError(t, json.Unmarshal(parseData(t, w), &resp))
		assert.True(t, resp.Idempotent, "re-POST must be idempotent")
		assert.Equal(t, firstAssignID, resp.Assignment.ID,
			"idempotent re-POST must return the same assignment.ID")
	})

	// Step 1.8 — GET /v1/tasks/:id/history. Must return exactly
	// 1 event (from 1.6; 1.7 was idempotent and did not write a
	// new event).
	t.Run("1.8_list_assignment_history", func(t *testing.T) {
		w := doRequest(t, env, http.MethodGet,
			"/v1/tasks/"+taskID.String()+"/history", "", nil)
		require.Equal(t, http.StatusOK, w.Code,
			"1.8 list history: %s", w.Body.String())
		var history []model.AssignmentEvent
		require.NoError(t, json.Unmarshal(parseData(t, w), &history))
		assert.Len(t, history, 1,
			"exactly 1 event after 1.6 (1.7 was idempotent — no new event)")
		if len(history) >= 1 {
			assert.Equal(t, "assign", history[0].Action,
				"the event action should be 'assign'")
		}
	})

	// Step 1.9 — POST /v1/executions. Create an execution for
	// the (task, agent) pair. B-001 lifecycle: initial status is
	// `assigned`. The worker is NOT auto-spawned in Sprint 6
	// (see B-001 brief — the in-process mock is started by the
	// runtime on a 1.10 PATCH).
	t.Run("1.9_create_execution", func(t *testing.T) {
		w := doRequest(t, env, http.MethodPost, "/v1/executions", "",
			map[string]any{
				"task_id":  taskID.String(),
				"agent_id": agentID.String(),
			})
		require.Equal(t, http.StatusCreated, w.Code,
			"1.9 create execution: %s", w.Body.String())
		var exec model.Execution
		require.NoError(t, json.Unmarshal(parseData(t, w), &exec))
		execID = exec.ID
		require.NotEqual(t, uuid.Nil, execID, "execution.ID must be set")
		// B-001 6-state lifecycle: initial state after create is "assigned".
		assert.Equal(t, model.ExecutionStatusAssigned, exec.Status,
			"initial execution status must be 'assigned' per B-001 lifecycle")
	})

	// Step 1.10 — PATCH /v1/executions/:id {status: running}.
	// Transitions assigned → running. Triggers the runtime to
	// start the mock worker.
	t.Run("1.10_patch_to_running", func(t *testing.T) {
		w := doRequest(t, env, http.MethodPatch,
			"/v1/executions/"+execID.String(), "",
			map[string]any{"status": "running"})
		require.Equal(t, http.StatusOK, w.Code,
			"1.10 patch running: %s", w.Body.String())
		var exec model.Execution
		require.NoError(t, json.Unmarshal(parseData(t, w), &exec))
		assert.Equal(t, model.ExecutionStatusRunning, exec.Status,
			"status should be 'running' after PATCH")
	})

	// Step 1.10a — Wait for the mock runtime to auto-transition
	// the execution from running to review. The MockRuntime's
	// default script has Delay=0, so the worker should call
	// driveWorker almost immediately. Poll the GET endpoint
	// for up to 2 seconds.
	t.Run("1.10a_mock_transitions_to_review", func(t *testing.T) {
		require.Eventually(t, func() bool {
			w := doRequest(t, env, http.MethodGet,
				"/v1/executions/"+execID.String(), "", nil)
			if w.Code != http.StatusOK {
				return false
			}
			var exec model.Execution
			if err := json.Unmarshal(parseData(t, w), &exec); err != nil {
				return false
			}
			return exec.Status == model.ExecutionStatusReview
		}, 2*time.Second, 25*time.Millisecond,
			"mock should auto-transition execution to 'review' within 2s")
	})

	// Step 1.11 — PATCH {status: completed}. The execution
	// moves from review to the terminal completed state.
	t.Run("1.11_patch_to_completed", func(t *testing.T) {
		w := doRequest(t, env, http.MethodPatch,
			"/v1/executions/"+execID.String(), "",
			map[string]any{"status": "completed"})
		require.Equal(t, http.StatusOK, w.Code,
			"1.11 patch completed: %s", w.Body.String())
		var exec model.Execution
		require.NoError(t, json.Unmarshal(parseData(t, w), &exec))
		assert.Equal(t, model.ExecutionStatusCompleted, exec.Status,
			"status should be 'completed' after PATCH")
	})

	// Step 1.12 — GET /v1/agents/:id. After the execution
	// completed, the agent's LastActiveAt must be non-zero
	// and >= the execution's StartedAt.
	t.Run("1.12_agent_last_active_at", func(t *testing.T) {
		w := doRequest(t, env, http.MethodGet,
			"/v1/agents/"+agentID.String(), projectID, nil)
		require.Equal(t, http.StatusOK, w.Code,
			"1.12 get agent: %s", w.Body.String())
		var agent model.Agent
		require.NoError(t, json.Unmarshal(parseData(t, w), &agent))
		assert.False(t, agent.LastActiveAt.IsZero(),
			"LastActiveAt must be set after execution completed")
	})

	// Step 1.13 — POST /v1/deliverables. Create the deliverable
	// tied to the (task, agent) pair. The service writes both
	// the deliverables row (Version=1) and a corresponding
	// deliverable_versions row.
	t.Run("1.13_create_deliverable", func(t *testing.T) {
		w := doRequest(t, env, http.MethodPost, "/v1/deliverables", "",
			map[string]any{
				"task_id":  taskID.String(),
				"agent_id": agentID.String(),
				"title":    "T1 Deliverable v1",
				"content":  "Initial deliverable content for T1 happy path.",
			})
		require.Equal(t, http.StatusCreated, w.Code,
			"1.13 create deliverable: %s", w.Body.String())
		var deliv struct {
			ID      uuid.UUID `json:"id"`
			TaskID  uuid.UUID `json:"task_id"`
			AgentID uuid.UUID `json:"agent_id"`
			Title   string    `json:"title"`
			Version int       `json:"version"`
		}
		require.NoError(t, json.Unmarshal(parseData(t, w), &deliv))
		delivID = deliv.ID
		require.NotEqual(t, uuid.Nil, delivID, "deliverable.ID must be set")
		assert.Equal(t, taskID, deliv.TaskID, "task_id echo")
		assert.Equal(t, agentID, deliv.AgentID, "agent_id echo")
		assert.Equal(t, 1, deliv.Version, "first deliverable version must be 1")
	})

	// Step 1.14 — PUT /v1/deliverables/:id. Update the
	// deliverable's title and content. The service writes a new
	// deliverable_versions row (Version=2) and updates the
	// head row in `deliverables`.
	t.Run("1.14_update_deliverable", func(t *testing.T) {
		w := doRequest(t, env, http.MethodPut,
			"/v1/deliverables/"+delivID.String(), "",
			map[string]any{
				"title":   "T1 Deliverable v2",
				"content": "Updated deliverable content for T1 happy path.",
			})
		require.Equal(t, http.StatusOK, w.Code,
			"1.14 update deliverable: %s", w.Body.String())
		var deliv struct {
			ID      uuid.UUID `json:"id"`
			Version int       `json:"version"`
		}
		require.NoError(t, json.Unmarshal(parseData(t, w), &deliv))
		assert.Equal(t, delivID, deliv.ID, "deliverable.ID stable on update")
		assert.Equal(t, 2, deliv.Version,
			"version must increment to 2 after PUT")
	})

	// Step 1.15 — GET /v1/deliverables/:id/versions. Returns the
	// full version history, newest first. Must be exactly 2
	// rows (the v1 from 1.13 and v2 from 1.14).
	t.Run("1.15_list_deliverable_versions", func(t *testing.T) {
		w := doRequest(t, env, http.MethodGet,
			"/v1/deliverables/"+delivID.String()+"/versions", "", nil)
		require.Equal(t, http.StatusOK, w.Code,
			"1.15 list versions: %s", w.Body.String())
		var versions []model.DeliverableVersion
		require.NoError(t, json.Unmarshal(parseData(t, w), &versions))
		assert.Len(t, versions, 2,
			"exactly 2 versions after 1.13 (v1) and 1.14 (v2)")
		// Versions are newest-first; v2 should be first.
		if len(versions) >= 2 {
			assert.Equal(t, 2, versions[0].Version, "newest version is 2")
			assert.Equal(t, 1, versions[1].Version, "oldest version is 1")
			assert.Equal(t, "T1 Deliverable v2", versions[0].Title,
				"v2 title echo")
			assert.Equal(t, "T1 Deliverable v1", versions[1].Title,
				"v1 title preserved")
		}
	})
}
