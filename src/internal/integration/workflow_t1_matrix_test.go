package integration_test

// T1.matrix — table-driven coverage layer for the T1 happy-path
// and cross-tenant blocks. Extends the scaffolding in
// workflow_t1_test.go (commit 2) and workflow_cross_tenant_test.go
// (commit 3) with an explicit assertion matrix per step.
//
// Each row in the table is a self-contained test case:
//   - method, path, body, projectID — the request
//   - expectedStatus, expectedCode   — wire-level expectations
//   - validate                       — optional deep check on the
//                                       parsed response (state
//                                       machine, version
//                                       increment, store state)
//
// The benefit over the per-step subtests in commits 2/3: the
// matrix is data-driven, easy to extend with negative cases
// (e.g., 400 on missing required fields, 409 on optimistic-lock
// conflict), and lives in one place for cross-referencing with
// the api-spec.
//
// See docs/reset/workflow-e2e-spec.md §1 (T1) and §2 (CT) for
// the contract that this matrix verifies.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// t1Step is one row of the T1 happy-path matrix.
type t1Step struct {
	name           string
	method         string
	path           func(state *t1State) string
	body           func(state *t1State) map[string]any
	projectID      func(state *t1State) string
	expectedStatus int
	validate       func(t *testing.T, w *httptest.ResponseRecorder, state *t1State)
}

// t1State is the running state across the T1 sequence.
// Populated as steps run; consumed by later steps.
type t1State struct {
	env         *IntegrationTestEnv
	projectID   string
	agent       *model.Agent
	task        *model.Task
	assignment  *model.Assignment
	exec        *model.Execution
	deliverable struct {
		ID      uuid.UUID
		Version int
	}
}

// crossStep is one row of the cross-tenant matrix. Each row
// expects a 404 (or 200-with-empty-data for CT6) when an
// attacker in project B targets a resource in project A.
type crossStep struct {
	name           string
	method         string
	path           func(env *crossTenantEnv) string
	body           func(env *crossTenantEnv) map[string]any
	projectID      func(env *crossTenantEnv) string // attacker's project
	expectedStatus int
	expectedEmpty  bool // for list endpoints
}

// TestIntegration_Workflow_T1_Matrix is the table-driven version
// of the 15-step T1 happy path. Runs the matrix in order so
// each step's `validate` can assert on state captured by earlier
// steps (e.g., step 1.7 asserts on the assignment.ID captured in
// step 1.6).
func TestIntegration_Workflow_T1_Matrix(t *testing.T) {
	env := newIntegrationRouter(t, store.NewMemoryStore())
	state := &t1State{
		env:       env,
		projectID: env.ProjectID,
	}

	for _, step := range t1Matrix() {
		step := step
		t.Run("matrix_"+step.name, func(t *testing.T) {
			t.Parallel()
			var body any
			if step.body != nil {
				body = step.body(state)
			}
			var pid string
			if step.projectID != nil {
				pid = step.projectID(state)
			} else {
				pid = state.projectID
			}
			w := doRequest(t, state.env, step.method, step.path(state), pid, body)
			require.Equal(t, step.expectedStatus, w.Code,
				"%s: expected %d got %d: %s",
				step.name, step.expectedStatus, w.Code, w.Body.String())
			if step.validate != nil {
				step.validate(t, w, state)
			}
		})
	}
}

// TestIntegration_Workflow_CrossTenant_Matrix is the table-driven
// version of the 6 cross-tenant negative tests. Each row's
// `expectedStatus` is 404 (or 200+empty for CT6).
func TestIntegration_Workflow_CrossTenant_Matrix(t *testing.T) {
	env := newCrossTenantEnv(t)

	for _, step := range crossMatrix() {
		step := step
		t.Run("matrix_"+step.name, func(t *testing.T) {
			t.Parallel()
			var body any
			if step.body != nil {
				body = step.body(&env)
			}
			pid := step.projectID(&env)
			w := doRequest(t, env.router, step.method, step.path(&env), pid, body)
			require.Equal(t, step.expectedStatus, w.Code,
				"%s: expected %d got %d: %s",
				step.name, step.expectedStatus, w.Code, w.Body.String())
			if step.expectedEmpty {
				// CT6 list variant: response must be
				// 200 with data: [].
				var list []model.Execution
				require.NoError(t, json.Unmarshal(parseData(t, w), &list),
					"%s: empty-list response must parse cleanly", step.name)
				assert.Empty(t, list,
					"%s: cross-tenant list must be empty (data leak otherwise)", step.name)
			}
		})
	}
}

// t1Matrix returns the 15-step T1 happy-path matrix. The matrix
// is intentionally a function (not a package var) so it can
// reference the helpers below without init-ordering issues.
func t1Matrix() []t1Step {
	return []t1Step{
		// 1.1 — create agent
		{
			name:   "1.1_create_agent",
			method: http.MethodPost,
			path:   func(s *t1State) string { return "/v1/agents" },
			body: func(s *t1State) map[string]any {
				return map[string]any{
					"project_id":   s.projectID,
					"name":         "T1 Matrix Agent",
					"role":         "developer",
					"capabilities": []string{"coding", "testing", "review"},
				}
			},
			projectID:      nil,
			expectedStatus: http.StatusCreated,
			validate: func(t *testing.T, w *httptest.ResponseRecorder, s *t1State) {
				var agent model.Agent
				require.NoError(t, json.Unmarshal(parseData(t, w), &agent))
				s.agent = &agent
				assert.Equal(t, "T1 Matrix Agent", agent.Name)
				assert.Len(t, agent.Capabilities, 3, "capability count from create")
			},
		},
		// 1.2 — read agent
		{
			name:   "1.2_get_agent",
			method: http.MethodGet,
			path:   func(s *t1State) string { return "/v1/agents/" + s.agent.ID.String() },
			body:   nil,
			projectID:      nil,
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, w *httptest.ResponseRecorder, s *t1State) {
				var got model.Agent
				require.NoError(t, json.Unmarshal(parseData(t, w), &got))
				assert.Equal(t, s.agent.ID, got.ID, "id stable")
				assert.Equal(t, "T1 Matrix Agent", got.Name, "name stable")
			},
		},
		// 1.3 — replace capabilities
		{
			name:   "1.3_update_capabilities",
			method: http.MethodPut,
			path:   func(s *t1State) string { return "/v1/agents/" + s.agent.ID.String() },
			body: func(s *t1State) map[string]any {
				return map[string]any{
					"capabilities": []string{"coding", "testing", "review", "deployment"},
					"version":      s.agent.Version,
				}
			},
			projectID:      nil,
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, w *httptest.ResponseRecorder, s *t1State) {
				var got model.Agent
				require.NoError(t, json.Unmarshal(parseData(t, w), &got))
				assert.Equal(t, s.agent.Version+1, got.Version, "version increments by 1")
				s.agent = &got
			},
		},
		// 1.4 — list capabilities
		{
			name:   "1.4_list_capabilities",
			method: http.MethodGet,
			path:   func(s *t1State) string { return "/v1/agents/" + s.agent.ID.String() + "/capabilities" },
			body:   nil,
			projectID:      nil,
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, w *httptest.ResponseRecorder, s *t1State) {
				var caps []model.AgentCapabilityView
				require.NoError(t, json.Unmarshal(parseData(t, w), &caps))
				assert.Len(t, caps, 4, "4 capabilities after 1.3 add")
			},
		},
		// 1.5 — create task
		{
			name:   "1.5_create_task",
			method: http.MethodPost,
			path:   func(s *t1State) string { return "/v1/projects/" + s.projectID + "/tasks" },
			body: func(s *t1State) map[string]any {
				return map[string]any{
					"title":       "T1 Matrix Task",
					"description": "Table-driven task",
					"priority":    "high",
				}
			},
			projectID:      func(s *t1State) string { return "" }, // no X-Project-ID on task create
			expectedStatus: http.StatusCreated,
			validate: func(t *testing.T, w *httptest.ResponseRecorder, s *t1State) {
				var task model.Task
				require.NoError(t, json.Unmarshal(parseData(t, w), &task))
				s.task = &task
				assert.Equal(t, "T1 Matrix Task", task.Title)
			},
		},
		// 1.6 — assign (first time, idempotent=false)
		{
			name:   "1.6_assign",
			method: http.MethodPost,
			path:   func(s *t1State) string { return "/v1/tasks/" + s.task.ID.String() + "/assign" },
			body: func(s *t1State) map[string]any {
				return map[string]any{
					"agent_id":              s.agent.ID.String(),
					"capabilities_required": []string{"coding"},
					"notes":                 "T1 matrix assign",
				}
			},
			projectID:      func(s *t1State) string { return "" },
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, w *httptest.ResponseRecorder, s *t1State) {
				var resp struct {
					Assignment model.Assignment     `json:"assignment"`
					Event      model.AssignmentEvent `json:"event"`
					Idempotent bool                  `json:"idempotent"`
				}
				require.NoError(t, json.Unmarshal(parseData(t, w), &resp))
				assert.False(t, resp.Idempotent, "first assign not idempotent")
				s.assignment = &resp.Assignment
			},
		},
		// 1.7 — re-assign (idempotent=true)
		{
			name:   "1.7_idempotent_reassign",
			method: http.MethodPost,
			path:   func(s *t1State) string { return "/v1/tasks/" + s.task.ID.String() + "/assign" },
			body: func(s *t1State) map[string]any {
				return map[string]any{
					"agent_id":              s.agent.ID.String(),
					"capabilities_required": []string{"coding"},
					"notes":                 "T1 matrix reassign (idempotent)",
				}
			},
			projectID:      func(s *t1State) string { return "" },
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, w *httptest.ResponseRecorder, s *t1State) {
				var resp struct {
					Assignment model.Assignment     `json:"assignment"`
					Event      model.AssignmentEvent `json:"event"`
					Idempotent bool                  `json:"idempotent"`
				}
				require.NoError(t, json.Unmarshal(parseData(t, w), &resp))
				assert.True(t, resp.Idempotent, "re-assign is idempotent")
				assert.Equal(t, s.assignment.ID, resp.Assignment.ID, "stable assignment.ID")
			},
		},
		// 1.8 — assignment history
		{
			name:   "1.8_assignment_history",
			method: http.MethodGet,
			path:   func(s *t1State) string { return "/v1/tasks/" + s.task.ID.String() + "/history" },
			body:   nil,
			projectID:      func(s *t1State) string { return "" },
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, w *httptest.ResponseRecorder, s *t1State) {
				var history []model.AssignmentEvent
				require.NoError(t, json.Unmarshal(parseData(t, w), &history))
				assert.Len(t, history, 1, "1 event after 1.6 (1.7 idempotent)")
			},
		},
		// 1.9 — create execution
		{
			name:   "1.9_create_execution",
			method: http.MethodPost,
			path:   func(s *t1State) string { return "/v1/executions" },
			body: func(s *t1State) map[string]any {
				return map[string]any{
					"task_id":  s.task.ID.String(),
					"agent_id": s.agent.ID.String(),
				}
			},
			projectID:      func(s *t1State) string { return "" },
			expectedStatus: http.StatusCreated,
			validate: func(t *testing.T, w *httptest.ResponseRecorder, s *t1State) {
				var exec model.Execution
				require.NoError(t, json.Unmarshal(parseData(t, w), &exec))
				s.exec = &exec
				assert.Equal(t, model.ExecutionStatusAssigned, exec.Status,
					"initial state per B-001 lifecycle is 'assigned'")
			},
		},
		// 1.10 — patch to running
		{
			name:   "1.10_patch_running",
			method: http.MethodPatch,
			path:   func(s *t1State) string { return "/v1/executions/" + s.exec.ExecutionID.String() },
			body: func(s *t1State) map[string]any {
				return map[string]any{"status": "running"}
			},
			projectID:      func(s *t1State) string { return "" },
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, w *httptest.ResponseRecorder, s *t1State) {
				var exec model.Execution
				require.NoError(t, json.Unmarshal(parseData(t, w), &exec))
				assert.Equal(t, model.ExecutionStatusRunning, exec.Status)
			},
		},
		// 1.10a — wait for mock to transition
		{
			name:   "1.10a_mock_to_review",
			method: http.MethodGet,
			path:   func(s *t1State) string { return "/v1/executions/" + s.exec.ExecutionID.String() },
			body:   nil,
			projectID:      func(s *t1State) string { return "" },
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, w *httptest.ResponseRecorder, s *t1State) {
				// The matrix row's HTTP call is the initial
				// read. The actual wait + assert happens in
				// the closure below using require.Eventually.
				var exec model.Execution
				require.NoError(t, json.Unmarshal(parseData(t, w), &exec))
				require.Eventually(t, func() bool {
					w2 := doRequest(t, s.env, http.MethodGet,
						"/v1/executions/"+s.exec.ExecutionID.String(), "", nil)
					if w2.Code != http.StatusOK {
						return false
					}
					var e model.Execution
					if err := json.Unmarshal(parseData(t, w2), &e); err != nil {
						return false
					}
					return e.Status == model.ExecutionStatusReview
				}, 2*time.Second, 25*time.Millisecond,
					"1.10a: mock should reach 'review' within 2s")
			},
		},
		// 1.11 — patch to completed
		{
			name:   "1.11_patch_completed",
			method: http.MethodPatch,
			path:   func(s *t1State) string { return "/v1/executions/" + s.exec.ExecutionID.String() },
			body: func(s *t1State) map[string]any {
				return map[string]any{"status": "completed"}
			},
			projectID:      func(s *t1State) string { return "" },
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, w *httptest.ResponseRecorder, s *t1State) {
				var exec model.Execution
				require.NoError(t, json.Unmarshal(parseData(t, w), &exec))
				assert.Equal(t, model.ExecutionStatusCompleted, exec.Status)
			},
		},
		// 1.12 — agent LastActiveAt
		{
			name:   "1.12_agent_last_active_at",
			method: http.MethodGet,
			path:   func(s *t1State) string { return "/v1/agents/" + s.agent.ID.String() },
			body:   nil,
			projectID:      nil,
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, w *httptest.ResponseRecorder, s *t1State) {
				var got model.Agent
				require.NoError(t, json.Unmarshal(parseData(t, w), &got))
				assert.False(t, got.LastActiveAt.IsZero(),
					"LastActiveAt set after execution completed")
			},
		},
		// 1.13 — create deliverable (version 1)
		{
			name:   "1.13_create_deliverable",
			method: http.MethodPost,
			path:   func(s *t1State) string { return "/v1/deliverables" },
			body: func(s *t1State) map[string]any {
				return map[string]any{
					"task_id":  s.task.ID.String(),
					"agent_id": s.agent.ID.String(),
					"title":    "T1 Matrix Deliverable v1",
					"content":  "Initial deliverable content for T1 matrix.",
				}
			},
			projectID:      func(s *t1State) string { return "" },
			expectedStatus: http.StatusCreated,
			validate: func(t *testing.T, w *httptest.ResponseRecorder, s *t1State) {
				var deliv struct {
					ID      uuid.UUID `json:"id"`
					Version int       `json:"version"`
				}
				require.NoError(t, json.Unmarshal(parseData(t, w), &deliv))
				s.deliverable.ID = deliv.ID
				s.deliverable.Version = deliv.Version
				assert.Equal(t, 1, deliv.Version, "first deliverable version is 1")
			},
		},
		// 1.14 — update deliverable (version 2)
		{
			name:   "1.14_update_deliverable",
			method: http.MethodPut,
			path:   func(s *t1State) string { return "/v1/deliverables/" + s.deliverable.ID.String() },
			body: func(s *t1State) map[string]any {
				return map[string]any{
					"title":   "T1 Matrix Deliverable v2",
					"content": "Updated deliverable content for T1 matrix.",
				}
			},
			projectID:      func(s *t1State) string { return "" },
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, w *httptest.ResponseRecorder, s *t1State) {
				var deliv struct {
					ID      uuid.UUID `json:"id"`
					Version int       `json:"version"`
				}
				require.NoError(t, json.Unmarshal(parseData(t, w), &deliv))
				assert.Equal(t, 2, deliv.Version, "version increments to 2")
				s.deliverable.Version = deliv.Version
			},
		},
		// 1.15 — list versions (2 rows)
		{
			name:   "1.15_list_deliverable_versions",
			method: http.MethodGet,
			path:   func(s *t1State) string { return "/v1/deliverables/" + s.deliverable.ID.String() + "/versions" },
			body:   nil,
			projectID:      func(s *t1State) string { return "" },
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, w *httptest.ResponseRecorder, s *t1State) {
				var versions []model.DeliverableVersion
				require.NoError(t, json.Unmarshal(parseData(t, w), &versions))
				assert.Len(t, versions, 2, "2 versions after 1.13 + 1.14")
				if len(versions) >= 2 {
					assert.Equal(t, 2, versions[0].Version, "newest is v2")
					assert.Equal(t, 1, versions[1].Version, "oldest is v1")
				}
			},
		},
	}
}

// crossMatrix returns the 6-row cross-tenant matrix.
func crossMatrix() []crossStep {
	return []crossStep{
		// CT1 — read agent in A from B
		{
			name:   "CT1_cross_tenant_agent_read",
			method: http.MethodGet,
			path:   func(e *crossTenantEnv) string { return "/v1/agents/" + e.agentA.String() },
			body:   nil,
			projectID:      func(e *crossTenantEnv) string { return e.projectB },
			expectedStatus: http.StatusNotFound,
		},
		// CT2 — update agent in A from B
		{
			name:   "CT2_cross_tenant_agent_update",
			method: http.MethodPut,
			path:   func(e *crossTenantEnv) string { return "/v1/agents/" + e.agentA.String() },
			body: func(e *crossTenantEnv) map[string]any {
				return map[string]any{
					"capabilities": []string{"hacked"},
					"version":      1,
				}
			},
			projectID:      func(e *crossTenantEnv) string { return e.projectB },
			expectedStatus: http.StatusNotFound,
		},
		// CT3 — assign task in A from B
		{
			name:   "CT3_cross_tenant_assign",
			method: http.MethodPost,
			path:   func(e *crossTenantEnv) string { return "/v1/tasks/" + e.taskA.String() + "/assign" },
			body: func(e *crossTenantEnv) map[string]any {
				return map[string]any{
					"agent_id":              e.agentB.String(),
					"capabilities_required": []string{"coding"},
					"notes":                 "CT3 cross-tenant attempt",
				}
			},
			projectID:      func(e *crossTenantEnv) string { return e.projectB },
			expectedStatus: http.StatusNotFound,
		},
		// CT4 — create execution for task in A from B
		{
			name:   "CT4_cross_tenant_execution_create",
			method: http.MethodPost,
			path:   func(e *crossTenantEnv) string { return "/v1/executions" },
			body: func(e *crossTenantEnv) map[string]any {
				return map[string]any{
					"task_id":  e.taskA.String(),
					"agent_id": e.agentB.String(),
				}
			},
			projectID:      func(e *crossTenantEnv) string { return e.projectB },
			expectedStatus: http.StatusNotFound,
		},
		// CT5 — create deliverable for task in A from B
		{
			name:   "CT5_cross_tenant_deliverable_create",
			method: http.MethodPost,
			path:   func(e *crossTenantEnv) string { return "/v1/deliverables" },
			body: func(e *crossTenantEnv) map[string]any {
				return map[string]any{
					"task_id":  e.taskA.String(),
					"agent_id": e.agentA.String(),
					"title":    "CT5 cross-tenant attempt",
					"content":  "should not be persisted",
				}
			},
			projectID:      func(e *crossTenantEnv) string { return e.projectB },
			expectedStatus: http.StatusNotFound,
		},
		// CT6 — list executions for task in A from B (200 + empty)
		{
			name:   "CT6_cross_tenant_executions_list_empty",
			method: http.MethodGet,
			path:   func(e *crossTenantEnv) string { return "/v1/executions?task_id=" + e.taskA.String() },
			body:   nil,
			projectID:      func(e *crossTenantEnv) string { return e.projectB },
			expectedStatus: http.StatusOK,
			expectedEmpty:  true,
		},
	}
}
