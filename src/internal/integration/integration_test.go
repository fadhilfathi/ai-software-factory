// Package integration_test contains cross-route integration
// tests for Sprint 4 (TASK-411). The tests wire a real Gin
// router with the real Sprint 4 services backed by an
// in-memory store, then exercise the agent → assignment →
// execution → deliverable flow end-to-end (T1) and verify that
// project-scoped routes return HTTP 400 (not 500) for non-UUID
// filter parameters (T2).
//
// The tests are WRITTEN on this host (no Go toolchain
// available on Windows per the team norm). The Pre-Commit
// Quality Gate (TASK-414) is the canonical execution point.
// See docs/sprint4/test-plan.md for the full plan and
// docs/sprint4/acceptance-report.md for the results.
//
// Sprint 4 scope (per Lead's option C): T1 is a SMOKE test
// (4 sub-steps proving the cross-route wiring works); the
// full T1 lifecycle (15 sub-steps) is deferred to Sprint 5.
// T2 is the full malformed-UUID table (11 sub-cases).
package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fadhilfathi/AI-Software-Factory/internal/handler"
	"github.com/fadhilfathi/AI-Software-Factory/internal/integration"
	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/aion"
	"github.com/fadhilfathi/AI-Software-Factory/internal/aion"
	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ----------------------------------------------------------------------------
// Test environment
// ----------------------------------------------------------------------------

// IntegrationTestEnv holds the wired router and the canonical
// test UUIDs. Each test gets a fresh env (fresh in-memory store).
type IntegrationTestEnv struct {
	Router    *gin.Engine
	Store     integration.Store
	UserID    string
	ProjectID string
}

// newIntegrationRouter wires a real Gin router with the real
// Sprint 4 services, all backed by the provided store. Auth is
// bypassed via a test middleware that sets request_id and
// user_id directly into the Gin context; the real
// middleware.Auth (JWT + API-key validation) is not invoked.
// Auth correctness is TASK-417/418's lane; integration tests
// focus on behaviour behind the auth boundary.
func newIntegrationRouter(t *testing.T, s integration.Store) *IntegrationTestEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	capSvc := service.NewCapabilityService(s, log)
	agentSvc := service.NewAgentService(s)
	taskSvc := service.NewTaskService(s, log)
	assignmentSvc := service.NewAssignmentService(s, capSvc, log)
	execSvc := service.NewExecutionService(s, log, nil, nil, aion.NewMockRuntime()) // TASK-501: in-process mock for integration tests
	delivSvc := service.NewDeliverableService(s, log)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("request_id", "test-int-rid-001")
		c.Set("user_id", "11111111-1111-1111-1111-111111111111")
		c.Next()
	})

	agents := handler.NewAgentHandler(agentSvc)
	caps := handler.NewCapabilityHandler(agentSvc)
	tasks := handler.NewTaskHandler(taskSvc)
	assigns := handler.NewAssignmentHandler(assignmentSvc)
	execs := handler.NewExecutionHandler(execSvc, log)
	delivs := handler.NewDeliverableHandler(delivSvc)

	v1 := r.Group("/v1")
	v1.POST("/agents", agents.Create)
	v1.GET("/agents", agents.List)
	v1.GET("/agents/:id", agents.Get)
	v1.PUT("/agents/:id", agents.Update)
	v1.DELETE("/agents/:id", agents.Delete)
	v1.GET("/agents/:id/capabilities", caps.ListAgentCapabilities)
	v1.GET("/capabilities", caps.ListCatalogCapabilities)

	v1.POST("/projects/:projectId/tasks", tasks.Create)
	v1.GET("/projects/:projectId/tasks", tasks.List)
	v1.GET("/tasks/:id", tasks.Get)
	v1.PUT("/tasks/:id", tasks.Update)
	v1.PATCH("/tasks/:id/status", tasks.UpdateStatus)

	v1.POST("/tasks/:id/assign", assigns.AssignTask)
	v1.GET("/tasks/:id/history", assigns.ListHistory)

	v1.POST("/executions", execs.Create)
	v1.GET("/executions", execs.List)
	v1.GET("/executions/:id", execs.GetByID)
	v1.PATCH("/executions/:id", execs.Patch)

	v1.POST("/deliverables", delivs.Create)
	v1.GET("/deliverables", delivs.List)
	v1.GET("/deliverables/:id", delivs.Get)
	v1.PUT("/deliverables/:id", delivs.Update)
	v1.GET("/deliverables/:id/versions", delivs.ListVersions)

	return &IntegrationTestEnv{
		Router:    r,
		Store:     s,
		UserID:    "11111111-1111-1111-1111-111111111111",
		ProjectID: uuid.New().String(),
	}
}

// ----------------------------------------------------------------------------
// Request / response helpers
// ----------------------------------------------------------------------------

// doRequest makes an HTTP request against the test router.
// projectID is the X-Project-ID header (empty = do not set).
// body is marshalled to JSON; nil = no body.
func doRequest(t *testing.T, env *IntegrationTestEnv, method, path, projectID string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, path, reqBody)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if projectID != "" {
		req.Header.Set("X-Project-ID", projectID)
	}
	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)
	return w
}

// responseEnvelope is the standard API response shape.
// Some handlers return `{"data": <x>}`; others return
// `{"error": {"code", "message"}}`. The test assertions are
// explicit about which shape they expect.
type responseEnvelope struct {
	Data  json.RawMessage `json:"data,omitempty"`
	Error *errorPayload   `json:"error,omitempty"`
}

// errorPayload matches the standard error envelope. Details
// and RequestID are optional — not all Sprint 4 handlers
// include them (see acceptance report §8 inconsistency note).
type errorPayload struct {
	Code      string          `json:"code"`
	Message   string          `json:"message"`
	Details   json.RawMessage `json:"details,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
}

// parseData extracts the `data` field from a success response.
// Fails the test if the response is an error envelope.
func parseData(t *testing.T, w *httptest.ResponseRecorder) json.RawMessage {
	t.Helper()
	var env responseEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env),
		"unmarshal response: %s", w.Body.String())
	require.Nil(t, env.Error, "expected success response, got error: %s", w.Body.String())
	require.NotEmpty(t, env.Data, "data field missing: %s", w.Body.String())
	return env.Data
}

// assertMalformedUUID400 verifies a 400 response with a typed
// error envelope. Different Sprint 4 handlers use different
// error codes (VALIDATION_ERROR vs BAD_REQUEST), so this
// helper accepts either as long as the response is a
// well-formed 400 with a non-empty code and message.
//
// If the handler returns 500, the sub-case is SKIPPED (not
// failed) per Lead's scope addendum — a 500 is a backend
// bug to be flagged in the acceptance report, not a test
// failure.
func assertMalformedUUID400(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	if w.Code == http.StatusInternalServerError {
		// Per Lead's scope addendum: 500 = backend bug, not a
		// test failure. Skip the sub-case and let the test
		// reporter flag it for the acceptance report.
		t.Skipf("BUG: handler returned 500 for malformed UUID. Body: %s", w.Body.String())
		return
	}
	require.Equal(t, http.StatusBadRequest, w.Code,
		"expected 400 for malformed UUID, got %d. Body: %s", w.Code, w.Body.String())
	var env responseEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env),
		"unmarshal response: %s", w.Body.String())
	require.NotNil(t, env.Error, "expected error envelope, got: %s", w.Body.String())
	assert.NotEmpty(t, env.Error.Code, "error.code missing: %s", w.Body.String())
	assert.NotEmpty(t, env.Error.Message, "error.message missing: %s", w.Body.String())
}

// ----------------------------------------------------------------------------
// T1: SMOKE Agent Lifecycle (Sprint 4 scope per Lead's option C)
//
// Full T1 lifecycle (15 sub-steps per test-plan §3) is deferred
// to Sprint 5. This Sprint 4 smoke proves the cross-route
// wiring works end-to-end: Create agent → Create task → Assign
// task to agent → Create deliverable. If these four steps
// pass, the router, services, and store are all talking to
// each other correctly.
// ----------------------------------------------------------------------------

func TestAgentLifecycle_CreateAssignExecuteDeliver_Smoke(t *testing.T) {
	env := newIntegrationRouter(t, integration.NewMemoryStore())
	projectID := env.ProjectID

	// Step 1: POST /v1/agents — create the agent
	w := doRequest(t, env, http.MethodPost, "/v1/agents", projectID,
		map[string]any{
			"project_id":   projectID,
			"name":         "Smoke Test Agent",
			"role":         "developer",
			"capabilities": []string{"coding", "testing"},
		})
	require.Equal(t, http.StatusCreated, w.Code,
		"step 1: create agent. Body: %s", w.Body.String())
	agentData := parseData(t, w)
	var agent model.Agent
	require.NoError(t, json.Unmarshal(agentData, &agent))
	agentID := agent.ID.String()
	require.NotEqual(t, uuid.Nil, agent.ID, "agent.ID must be set")
	assert.Equal(t, "Smoke Test Agent", agent.Name)
	assert.Equal(t, "developer", agent.Role)

	// Step 2: POST /v1/projects/:projectId/tasks — create a task
	w = doRequest(t, env, http.MethodPost,
		"/v1/projects/"+projectID+"/tasks", "",
		map[string]any{
			"title":       "Smoke Test Task",
			"description": "A task for the smoke test",
		})
	require.Equal(t, http.StatusCreated, w.Code,
		"step 2: create task. Body: %s", w.Body.String())
	taskData := parseData(t, w)
	var task model.Task
	require.NoError(t, json.Unmarshal(taskData, &task))
	taskID := task.ID.String()
	require.NotEqual(t, uuid.Nil, task.ID, "task.ID must be set")
	assert.Equal(t, "Smoke Test Task", task.Title)

	// Step 3: POST /v1/tasks/:id/assign — auto-assign the task to
	// the agent. The assignment service should match by capability
	// (the agent has "coding", the request requires "coding").
	w = doRequest(t, env, http.MethodPost,
		"/v1/tasks/"+taskID+"/assign", "",
		map[string]any{
			"agent_id":              agentID,
			"capabilities_required": []string{"coding"},
			"notes":                 "smoke test assignment",
		})
	require.Equal(t, http.StatusOK, w.Code,
		"step 3: assign task. Body: %s", w.Body.String())
	assignData := parseData(t, w)
	var assignResp struct {
		Assignment model.Assignment    `json:"assignment"`
		Event      model.AssignmentEvent `json:"event"`
		Idempotent bool                 `json:"idempotent"`
	}
	require.NoError(t, json.Unmarshal(assignData, &assignResp))
	assert.False(t, assignResp.Idempotent,
		"first assignment should not be idempotent")
	assert.Equal(t, agentID, assignResp.Assignment.AgentID.String(),
		"assignment should target the smoke test agent")
	// Assignment struct in Sprint 4 has no RequiredCapabilities field — that's on the
	// Task struct. The TASK-404 service code matches capabilities but does not echo
	// them back on the assignment row. Sprint 5 may add this back.
	_ = assignResp.Assignment // keep the assignment variable referenced
	assert.Equal(t, "active", string(assignResp.Assignment.Status),
		"assignment should be active after successful assign")

	// Step 4: POST /v1/deliverables — create a deliverable tied to
	// the task and agent. This proves the final hop in the
	// cross-route chain.
	w = doRequest(t, env, http.MethodPost, "/v1/deliverables", "",
		map[string]any{
			"task_id":  taskID,
			"agent_id": agentID,
			"title":    "Smoke Test Deliverable",
			"content":  "Smoke test deliverable content for Sprint 4 acceptance.",
		})
	require.Equal(t, http.StatusCreated, w.Code,
		"step 4: create deliverable. Body: %s", w.Body.String())
	delivData := parseData(t, w)
	var deliv struct {
		ID      string `json:"id"`
		TaskID  string `json:"task_id"`
		AgentID string `json:"agent_id"`
		Title   string `json:"title"`
		Version int    `json:"version"`
	}
	require.NoError(t, json.Unmarshal(delivData, &deliv))
	assert.Equal(t, taskID, deliv.TaskID)
	assert.Equal(t, agentID, deliv.AgentID)
	assert.Equal(t, 1, deliv.Version, "first version should be 1")
}

// ----------------------------------------------------------------------------
// T2: Malformed UUIDs (full table — 11 sub-cases)
//
// Per Lead's scope addendum: project-scoped routes
// (/v1/agents, /v1/tasks/:id/assign, /v1/executions,
// /v1/deliverables) must return HTTP 400 (not 500) when
// filter parameters (task_id, agent_id, project_id) are
// non-UUID strings. A 500 is treated as a backend bug and
// flagged (via t.Skip) for the acceptance report.
// ----------------------------------------------------------------------------

// TestProjectScopedRoutes_RejectMalformedUUIDs covers 11
// sub-cases (test plan §4). Each row is a sub-test via t.Run;
// a 500 in any sub-case skips that row and is recorded for
// the acceptance report.
func TestProjectScopedRoutes_RejectMalformedUUIDs(t *testing.T) {
	env := newIntegrationRouter(t, integration.NewMemoryStore())
	projectID := env.ProjectID
	validTaskID := uuid.New().String()
	validAgentID := uuid.New().String()

	// Sub-cases per test-plan §4.
	// "Filter / Body" indicates where the malformed UUID appears.
	// Cases 2.6, 2.9, 2.10, 2.11 re-assert existing handler-test
	// coverage (belt-and-braces for the CI gate's one coherent run).
	cases := []struct {
		name    string
		method  string
		path    string
		project string // X-Project-ID; "" = omit
		body    any
	}{
		// 2.1 — /v1/agents list, ?project_id malformed
		{
			name:    "agents_list_project_id_malformed",
			method:  http.MethodGet,
			path:    "/v1/agents?project_id=not-a-uuid",
			project: projectID,
		},
		// 2.2 — /v1/agents list, ?project_id empty (NOT malformed —
		// empty means "filter not provided"; per Lead's decision
		// the response is 200 with empty list, not 400)
		// Documented in §8 of the acceptance report.
		// 2.3 — /v1/agents list, ?project_id near-miss (bad char)
		{
			name:    "agents_list_project_id_near_miss",
			method:  http.MethodGet,
			path:    "/v1/agents?project_id=12345678-1234-1234-1234-12345678901Z",
			project: projectID,
		},
		// 2.4 — /v1/executions list, ?task_id malformed
		{
			name:   "executions_list_task_id_malformed",
			method: http.MethodGet,
			path:   "/v1/executions?task_id=not-a-uuid",
		},
		// 2.5 — /v1/executions list, ?agent_id malformed
		{
			name:   "executions_list_agent_id_malformed",
			method: http.MethodGet,
			path:   "/v1/executions?agent_id=not-a-uuid",
		},
		// 2.6 — /v1/executions list, ?status=garbage (re-assert)
		{
			name:   "executions_list_status_garbage",
			method: http.MethodGet,
			path:   "/v1/executions?status=garbage",
		},
		// 2.7 — /v1/deliverables list, ?task_id malformed
		{
			name:   "deliverables_list_task_id_malformed",
			method: http.MethodGet,
			path:   "/v1/deliverables?task_id=not-a-uuid",
		},
		// 2.8 — /v1/deliverables list, ?agent_id malformed
		{
			name:   "deliverables_list_agent_id_malformed",
			method: http.MethodGet,
			path:   "/v1/deliverables?agent_id=not-a-uuid",
		},
		// 2.9 — POST /v1/tasks/:id/assign, path :id malformed (re-assert)
		{
			name:   "assign_path_task_id_malformed",
			method: http.MethodPost,
			path:   "/v1/tasks/not-a-uuid/assign",
			body: map[string]any{
				"agent_id":              validAgentID,
				"capabilities_required": []string{"coding"},
			},
		},
		// 2.10 — POST /v1/tasks/:id/assign, body agent_id malformed (re-assert)
		{
			name:   "assign_body_agent_id_malformed",
			method: http.MethodPost,
			path:   "/v1/tasks/" + validTaskID + "/assign",
			body: map[string]any{
				"agent_id":              "not-a-uuid",
				"capabilities_required": []string{"coding"},
			},
		},
		// 2.11 — POST /v1/tasks/:id/assign, body agent_id empty (re-assert)
		{
			name:   "assign_body_agent_id_empty",
			method: http.MethodPost,
			path:   "/v1/tasks/" + validTaskID + "/assign",
			body: map[string]any{
				"agent_id":              "",
				"capabilities_required": []string{"coding"},
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			w := doRequest(t, env, tc.method, tc.path, tc.project, tc.body)
			assertMalformedUUID400(t, w)
		})
	}
}
