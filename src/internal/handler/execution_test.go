package handler

// HTTP-level tests for ExecutionHandler (TASK-405, Sprint 4; B-001 Sprint 6).
//
// Strategy: drive Gin with a real router, a stub auth middleware
// (so we can test the "no user_id" → 401 case), and a real
// service.ExecutionService backed by an in-memory store. The
// mock goroutine is configured to fire immediately (zero sleep)
// so tests don't have to wait. Covers:
//   - POST   /v1/executions
//   - GET    /v1/executions
//   - GET    /v1/executions/:id
//   - PATCH  /v1/executions/:id
//   - PATCH  /v1/executions/:id/review (B-001 reviewer action)
//   - DELETE /v1/executions/:id       (B-001 operator cancel)
//   - auth middleware (401)

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/aion"
	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// newExecutionTestRouter wires a Gin engine with the 4 execution
// routes and a stub auth middleware. When withUserID is empty,
// the middleware short-circuits with 401 — that is how we test
// the "missing auth" path. When withUserID is set, the
// middleware stashes the user_id for downstream handlers (the
// execution handler does not currently read it, but the route
// is auth-gated and we want the real shape).
func newExecutionTestRouter(t *testing.T, withUserID string) (*gin.Engine, *service.ExecutionService, store.Store) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("request_id", "test-rid-exec-001")
		if withUserID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "authentication required"},
			})
			return
		}
		c.Set("user_id", withUserID)
		c.Next()
	})

	// Real service backed by an in-memory store. Mock goroutine
	// is configured to fire immediately and never fail.
	s := store.NewMemoryStore()
	cfg := &service.ExecutionServiceConfig{
		MockSleep:       func() time.Duration { return 0 },
		MockFailureRate: 0.0,
	}
	svc := service.NewExecutionService(s, zap.NewNop(), cfg, aion.NewMockRuntime()) // TASK-501: in-process mock for handler tests
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = svc.Shutdown(ctx)
	})

	h := NewExecutionHandler(svc, zap.NewNop())
	v1 := r.Group("/v1")
	{
		v1.POST("/executions", h.Create)
		v1.GET("/executions", h.List)
		v1.GET("/executions/:id", h.GetByID)
		v1.PATCH("/executions/:id", h.Patch)
		// B-001 reviewer action + operator cancel.
		v1.PATCH("/executions/:id/review", h.Review)
		v1.DELETE("/executions/:id", h.Cancel)
	}
	return r, svc, s
}

// seedExecTaskAndAgent creates a task and an agent in the store
// so CreateExecution's validation passes.
// TASK-422: returns (taskID, agentID, projectID) so callers can pass
// task.ProjectID as callerProjectID to the service AND set the
// X-Project-ID header on every handler request.
func seedExecTaskAndAgent(t *testing.T, s store.Store) (uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	ctx := context.Background()

	projectID := uuid.New()
	task := &model.Task{
		ID:        uuid.New(),
		ProjectID: projectID,
		Title:     "exec-handler-test-" + uuid.NewString()[:8],
		Status:    model.TaskOpen,
		Priority:  model.PriorityNormal,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, s.Tasks().Create(task))

	agentSvc := service.NewAgentService(s)
	created, apiErr := agentSvc.CreateAgent(ctx, service.CreateAgentRequest{
		ProjectID:    task.ProjectID,
		Name:         "agent-" + uuid.NewString()[:8],
		Role:         "developer",
		Capabilities: []string{"coding"},
	})
	require.Nil(t, apiErr)
	return task.ID, created.ID, projectID
}

// doExecutionRequest fires a single request and returns the
// response recorder. projectID == uuid.Nil → omit the
// X-Project-ID header (used to assert 400 MISSING_PROJECT_HEADER).
// Otherwise the header is set to projectID.String().
func doExecutionRequest(r *gin.Engine, method, path string, projectID uuid.UUID, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if projectID != uuid.Nil {
		req.Header.Set("X-Project-ID", projectID.String())
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ----------------------------------------------------------------------------
// POST /v1/executions
// ----------------------------------------------------------------------------

func TestExecutionHandler_Create_201(t *testing.T) {
	r, _, s := newExecutionTestRouter(t, uuid.NewString())
	taskID, agentID, projectID := seedExecTaskAndAgent(t, s)

	body := map[string]any{"task_id": taskID.String(), "agent_id": agentID.String()}
	w := doExecutionRequest(r, http.MethodPost, "/v1/executions", projectID, body)
	require.Equal(t, http.StatusCreated, w.Code)

	var resp struct {
		Data model.Execution `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, uuid.Nil, resp.Data.ExecutionID)
	assert.Equal(t, taskID, resp.Data.TaskID)
	assert.Equal(t, agentID, resp.Data.AgentID)
	assert.Equal(t, model.ExecutionStatusAssigned, resp.Data.Status)
}

func TestExecutionHandler_Create_400_BadUUID(t *testing.T) {
	r, _, _ := newExecutionTestRouter(t, uuid.NewString())
	body := map[string]any{"task_id": "not-a-uuid", "agent_id": uuid.NewString()}
	w := doExecutionRequest(r, http.MethodPost, "/v1/executions", uuid.New(), body)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	body = map[string]any{"task_id": uuid.NewString(), "agent_id": "still-not-a-uuid"}
	w = doExecutionRequest(r, http.MethodPost, "/v1/executions", uuid.New(), body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExecutionHandler_Create_404(t *testing.T) {
	r, _, s := newExecutionTestRouter(t, uuid.NewString())
	_, agentID, projectA := seedExecTaskAndAgent(t, s)

	// Task does not exist.
	body := map[string]any{"task_id": uuid.NewString(), "agent_id": agentID.String()}
	w := doExecutionRequest(r, http.MethodPost, "/v1/executions", projectA, body)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "TASK_NOT_FOUND")

	// Agent does not exist.
	taskID, _, projectB := seedExecTaskAndAgent(t, s)
	body = map[string]any{"task_id": taskID.String(), "agent_id": uuid.NewString()}
	w = doExecutionRequest(r, http.MethodPost, "/v1/executions", projectB, body)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "AGENT_NOT_FOUND")
}

// ----------------------------------------------------------------------------
// GET /v1/executions/:id
// ----------------------------------------------------------------------------

func TestExecutionHandler_GetByID_Success(t *testing.T) {
	r, svc, s := newExecutionTestRouter(t, uuid.NewString())
	taskID, agentID, projectID := seedExecTaskAndAgent(t, s)
	exec, err := svc.CreateExecution(context.Background(), taskID, agentID, projectID)
	require.NoError(t, err)

	w := doExecutionRequest(r, http.MethodGet, "/v1/executions/"+exec.ExecutionID.String(), projectID, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data model.Execution `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, exec.ExecutionID, resp.Data.ExecutionID)
}

func TestExecutionHandler_GetByID_404(t *testing.T) {
	r, _, _ := newExecutionTestRouter(t, uuid.NewString())
	w := doExecutionRequest(r, http.MethodGet, "/v1/executions/"+uuid.NewString(), uuid.New(), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "EXECUTION_NOT_FOUND")
}

// ----------------------------------------------------------------------------
// GET /v1/executions (list)
// ----------------------------------------------------------------------------

func TestExecutionHandler_List_WithFilters(t *testing.T) {
	// Inline router setup so we can slow the mock worker.
	// The default newExecutionTestRouter uses MockSleep=0,
	// which lets the worker race ahead and drive all 4
	// executions to a terminal state before the test can
	// issue the status=pending filter. With MockSleep=500ms
	// the executions stay in pending long enough for the
	// filter to see at least one row.
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("request_id", "test-rid-exec-list-001")
		c.Set("user_id", uuid.NewString())
		c.Next()
	})
	st := store.NewMemoryStore()
	cfg := &service.ExecutionServiceConfig{
		MockSleep:       func() time.Duration { return 500 * time.Millisecond },
		MockFailureRate: 0.0,
	}
	// Slow the production-runtime path too so the worker
	// stays in pending/running long enough for the test to
	// observe at least one pending row.
	runtime := aion.NewMockRuntime()
	runtime.SetDefaultScript(aion.FakeScript{Delay: 500 * time.Millisecond})
	svc := service.NewExecutionService(st, zap.NewNop(), cfg, runtime)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = svc.Shutdown(ctx)
	})
	h := NewExecutionHandler(svc, zap.NewNop())
	v1 := r.Group("/v1")
	{
		v1.POST("/executions", h.Create)
		v1.GET("/executions", h.List)
		v1.GET("/executions/:id", h.GetByID)
		v1.PATCH("/executions/:id", h.Patch)
		// B-001 reviewer action + operator cancel.
		v1.PATCH("/executions/:id/review", h.Review)
		v1.DELETE("/executions/:id", h.Cancel)
	}
	s := st
	taskA, agentA, projectA := seedExecTaskAndAgent(t, s)
	taskB, agentB, _ := seedExecTaskAndAgent(t, s)

	// Re-link taskB/agentB to projectA so all 4 executions land
	// in the same project (the original filter counts hold).
	// Cross-project semantics are covered by the new tests at the
	// bottom of this file (TASK-422).
	require.NoError(t, s.Tasks().Update(&model.Task{ID: taskB, ProjectID: projectA, Title: "relinked", Status: model.TaskOpen, Priority: model.PriorityNormal, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}))
	// Re-link agentB to projectA. The store's Update checks
	// optimistic-concurrency via the Version field, so we
	// read the current row first to keep Version in sync.
	// Without this, Update sees Version=0 and rejects the
	// call as a version conflict.
	curAgent, err := s.Agents().GetByID(context.Background(), agentB)
	require.NoError(t, err)
	curAgent.ProjectID = projectA
	curAgent.Name = "relinked-b"
	require.NoError(t, s.Agents().Update(context.Background(), curAgent))

	// Seed: 2 on (taskA, agentA), 1 on (taskA, agentB), 1 on (taskB, agentA).
	// Use a fresh local name inside the for loop to avoid
	// shadowing the outer err from the curAgent fetch above.
	for i := 0; i < 2; i++ {
		_, errSeed := svc.CreateExecution(context.Background(), taskA, agentA, projectA)
		require.NoError(t, errSeed)
	}
	_, err = svc.CreateExecution(context.Background(), taskA, agentB, projectA)
	require.NoError(t, err)
	_, err = svc.CreateExecution(context.Background(), taskB, agentA, projectA)
	require.NoError(t, err)

	// No filter: all 4.
	w := doExecutionRequest(r, http.MethodGet, "/v1/executions", projectA, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data model.ExecutionListResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Data.Items, 4)

	// task_id filter
	w = doExecutionRequest(r, http.MethodGet, "/v1/executions?task_id="+taskA.String(), projectA, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Data.Items, 3)

	// agent_id filter
	w = doExecutionRequest(r, http.MethodGet, "/v1/executions?agent_id="+agentA.String(), projectA, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Data.Items, 3)

	// status=assigned filter (B-001: 6-state lifecycle, 'pending' is gone)
	w = doExecutionRequest(r, http.MethodGet, "/v1/executions?status=assigned", projectA, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.GreaterOrEqual(t, len(resp.Data.Items), 1, "expected at least one assigned row right after create")

	// status=garbage → 400
	w = doExecutionRequest(r, http.MethodGet, "/v1/executions?status=garbage", projectA, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_EXECUTION_STATUS")
}

// ----------------------------------------------------------------------------
// PATCH /v1/executions/:id
// ----------------------------------------------------------------------------

func TestExecutionHandler_Patch_200(t *testing.T) {
	// Inline router setup so we can slow the mock worker. The
	// default newExecutionTestRouter uses MockSleep=0, which
	// lets the worker race ahead and drive the execution to a
	// terminal state before the test can PATCH a forward status.
	// With MockSleep=500ms the test has a clear window to issue
	// the PATCHes in order.
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("request_id", "test-rid-exec-001")
		c.Set("user_id", uuid.NewString())
		c.Next()
	})
	st := store.NewMemoryStore()
	cfg := &service.ExecutionServiceConfig{
		MockSleep:       func() time.Duration { return 500 * time.Millisecond },
		MockFailureRate: 0.0,
	}
	// Slow the production-runtime path too: register a 500ms-delay
	// default script on the mock runtime so the worker stays
	// in pending/running state long enough for the test's PATCH
	// to win the race. Without this, the worker terminates in
	// microseconds and the PATCH sees a terminal execution.
	runtime := aion.NewMockRuntime()
	runtime.SetDefaultScript(aion.FakeScript{Delay: 500 * time.Millisecond})
	svc := service.NewExecutionService(st, zap.NewNop(), cfg, runtime)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = svc.Shutdown(ctx)
	})
	h := NewExecutionHandler(svc, zap.NewNop())
	v1 := r.Group("/v1")
	{
		v1.POST("/executions", h.Create)
		v1.GET("/executions", h.List)
		v1.GET("/executions/:id", h.GetByID)
		v1.PATCH("/executions/:id", h.Patch)
		// B-001 reviewer action + operator cancel.
		v1.PATCH("/executions/:id/review", h.Review)
		v1.DELETE("/executions/:id", h.Cancel)
	}
	s := st
	taskID, agentID, projectID := seedExecTaskAndAgent(t, s)
	exec, err := svc.CreateExecution(context.Background(), taskID, agentID, projectID)
	require.NoError(t, err)

	body := map[string]any{"status": "running"}
	w := doExecutionRequest(r, http.MethodPatch, "/v1/executions/"+exec.ExecutionID.String(), projectID, body)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data model.Execution `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, model.ExecutionStatusRunning, resp.Data.Status)

	// B-001 6-state lifecycle: running → review → completed is the
	// only path into 'completed' (the reviewer action lives in B-001 c3).
	body = map[string]any{"status": "review"}
	w = doExecutionRequest(r, http.MethodPatch, "/v1/executions/"+exec.ExecutionID.String(), projectID, body)
	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, model.ExecutionStatusReview, resp.Data.Status)

	body = map[string]any{"status": "completed"}
	w = doExecutionRequest(r, http.MethodPatch, "/v1/executions/"+exec.ExecutionID.String(), projectID, body)
	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, model.ExecutionStatusCompleted, resp.Data.Status)
	require.NotNil(t, resp.Data.CompletedAt)
}

func TestExecutionHandler_Patch_409(t *testing.T) {
	r, svc, s := newExecutionTestRouter(t, uuid.NewString())
	taskID, agentID, projectID := seedExecTaskAndAgent(t, s)
	exec, err := svc.CreateExecution(context.Background(), taskID, agentID, projectID)
	require.NoError(t, err)

	// assigned → running is valid; running → assigned is not.
	_, err = svc.UpdateExecutionStatus(context.Background(), exec.ExecutionID, model.ExecutionStatusRunning, nil, projectID)
	require.NoError(t, err)

	body := map[string]any{"status": "assigned"}
	w := doExecutionRequest(r, http.MethodPatch, "/v1/executions/"+exec.ExecutionID.String(), projectID, body)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_STATE_TRANSITION")
}

func TestExecutionHandler_Patch_404(t *testing.T) {
	r, _, _ := newExecutionTestRouter(t, uuid.NewString())
	body := map[string]any{"status": "running"}
	w := doExecutionRequest(r, http.MethodPatch, "/v1/executions/"+uuid.NewString(), uuid.New(), body)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "EXECUTION_NOT_FOUND")
}

// ----------------------------------------------------------------------------
// Auth
// ----------------------------------------------------------------------------

func TestExecutionHandler_MissingAuth_401(t *testing.T) {
	// No user_id → middleware short-circuits with 401 before
	// the handler is even called.
	r, _, _ := newExecutionTestRouter(t, "")
	w := doExecutionRequest(r, http.MethodGet, "/v1/executions", uuid.New(), nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "UNAUTHORIZED")
}
// ----------------------------------------------------------------------------
// Cross-tenant (F-016, TASK-422)
// ----------------------------------------------------------------------------

func TestExecutionHandler_MissingProjectHeader_400(t *testing.T) {
	r, _, s := newExecutionTestRouter(t, uuid.NewString())
	taskID, agentID, _ := seedExecTaskAndAgent(t, s)

	body := map[string]any{"task_id": taskID.String(), "agent_id": agentID.String()}
	w := doExecutionRequest(r, http.MethodPost, "/v1/executions", uuid.Nil, body)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "MISSING_PROJECT_HEADER")

	w = doExecutionRequest(r, http.MethodGet, "/v1/executions/"+uuid.NewString(), uuid.Nil, nil)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "MISSING_PROJECT_HEADER")

	w = doExecutionRequest(r, http.MethodGet, "/v1/executions", uuid.Nil, nil)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "MISSING_PROJECT_HEADER")

	w = doExecutionRequest(r, http.MethodPatch, "/v1/executions/"+uuid.NewString(), uuid.Nil, body)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "MISSING_PROJECT_HEADER")
}

func TestExecutionHandler_Create_CrossTenant_404(t *testing.T) {
	r, _, s := newExecutionTestRouter(t, uuid.NewString())
	taskID, agentID, _ := seedExecTaskAndAgent(t, s)

	body := map[string]any{"task_id": taskID.String(), "agent_id": agentID.String()}
	// Send a DIFFERENT project in the header than the one the
	// task/agent belong to.
	w := doExecutionRequest(r, http.MethodPost, "/v1/executions", uuid.New(), body)
	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "CROSS_TENANT_BLOCKED")
}

func TestExecutionHandler_GetByID_CrossTenant_404(t *testing.T) {
	r, svc, s := newExecutionTestRouter(t, uuid.NewString())
	taskID, agentID, projectID := seedExecTaskAndAgent(t, s)
	exec, err := svc.CreateExecution(context.Background(), taskID, agentID, projectID)
	require.NoError(t, err)

	w := doExecutionRequest(r, http.MethodGet, "/v1/executions/"+exec.ExecutionID.String(), uuid.New(), nil)
	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "CROSS_TENANT_BLOCKED")
}

func TestExecutionHandler_Patch_CrossTenant_404(t *testing.T) {
	r, svc, s := newExecutionTestRouter(t, uuid.NewString())
	taskID, agentID, projectID := seedExecTaskAndAgent(t, s)
	exec, err := svc.CreateExecution(context.Background(), taskID, agentID, projectID)
	require.NoError(t, err)

	body := map[string]any{"status": "running"}
	w := doExecutionRequest(r, http.MethodPatch, "/v1/executions/"+exec.ExecutionID.String(), uuid.New(), body)
	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "CROSS_TENANT_BLOCKED")
}
// ----------------------------------------------------------------------------
// PATCH /v1/executions/:id/review (B-001 reviewer action)
// ----------------------------------------------------------------------------

// seedExecutionInReview creates an execution, drives it through
// assigned->running->review (via the service), and returns the
// final row. Used by the reviewer-action tests so they start from
// the correct state without depending on the mock goroutine's timing.
func seedExecutionInReview(t *testing.T, svc *service.ExecutionService, s store.Store) (uuid.UUID, uuid.UUID, *model.Execution) {
	t.Helper()
	ctx := context.Background()
	taskID, agentID, projectID := seedExecTaskAndAgent(t, s)
	exec, err := svc.CreateExecution(ctx, taskID, agentID, projectID)
	require.NoError(t, err)
	updated, err := svc.UpdateExecutionStatus(ctx, exec.ExecutionID, model.ExecutionStatusRunning, nil, projectID)
	require.NoError(t, err)
	updated, err = svc.UpdateExecutionStatus(ctx, exec.ExecutionID, model.ExecutionStatusReview, nil, projectID)
	require.NoError(t, err)
	require.Equal(t, model.ExecutionStatusReview, updated.Status)
	return taskID, projectID, updated
}

func TestExecutionHandler_Review_Accept_200(t *testing.T) {
	r, svc, s := newExecutionTestRouter(t, uuid.NewString())
	_, projectID, exec := seedExecutionInReview(t, svc, s)

	body := map[string]any{"accepted": true}
	w := doExecutionRequest(r, http.MethodPatch, "/v1/executions/"+exec.ExecutionID.String()+"/review", projectID, body)
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

	var resp struct {
		Data struct {
			ID   uuid.UUID             `json:"id"`
			From model.ExecutionStatus `json:"from"`
			To   model.ExecutionStatus `json:"to"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, exec.ExecutionID, resp.Data.ID)
	assert.Equal(t, model.ExecutionStatusReview, resp.Data.From)
	assert.Equal(t, model.ExecutionStatusCompleted, resp.Data.To)

	// Confirm the row is now in COMPLETED via the service.
	final, err := svc.GetExecution(context.Background(), exec.ExecutionID, projectID)
	require.NoError(t, err)
	assert.Equal(t, model.ExecutionStatusCompleted, final.Status)
	assert.NotNil(t, final.CompletedAt)
}

func TestExecutionHandler_Review_Reject_200(t *testing.T) {
	r, svc, s := newExecutionTestRouter(t, uuid.NewString())
	_, projectID, exec := seedExecutionInReview(t, svc, s)

	body := map[string]any{"accepted": false, "reason": "output is not what the spec asks for"}
	w := doExecutionRequest(r, http.MethodPatch, "/v1/executions/"+exec.ExecutionID.String()+"/review", projectID, body)
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

	var resp struct {
		Data struct {
			To model.ExecutionStatus `json:"to"`
	} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, model.ExecutionStatusFailed, resp.Data.To)

	final, err := svc.GetExecution(context.Background(), exec.ExecutionID, projectID)
	require.NoError(t, err)
	assert.Equal(t, model.ExecutionStatusFailed, final.Status)
	require.NotNil(t, final.ErrorMessage)
	assert.Equal(t, "output is not what the spec asks for", *final.ErrorMessage)
}

func TestExecutionHandler_Review_MissingReason_400(t *testing.T) {
	r, svc, s := newExecutionTestRouter(t, uuid.NewString())
	_, projectID, exec := seedExecutionInReview(t, svc, s)

	// accepted=false but no reason -> 400 VALIDATION_ERROR.
	body := map[string]any{"accepted": false}
	w := doExecutionRequest(r, http.MethodPatch, "/v1/executions/"+exec.ExecutionID.String()+"/review", projectID, body)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "VALIDATION_ERROR")

	// The execution should still be in REVIEW (no transition happened).
	final, err := svc.GetExecution(context.Background(), exec.ExecutionID, projectID)
	require.NoError(t, err)
	assert.Equal(t, model.ExecutionStatusReview, final.Status)
}

func TestExecutionHandler_Review_MissingAccepted_400(t *testing.T) {
	r, svc, s := newExecutionTestRouter(t, uuid.NewString())
	_, projectID, exec := seedExecutionInReview(t, svc, s)

	// No `accepted` field at all -> 400 VALIDATION_ERROR.
	body := map[string]any{"reason": "missing accepted"}
	w := doExecutionRequest(r, http.MethodPatch, "/v1/executions/"+exec.ExecutionID.String()+"/review", projectID, body)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "accepted is required")
}

func TestExecutionHandler_Review_WrongState_409(t *testing.T) {
	r, svc, s := newExecutionTestRouter(t, uuid.NewString())
	ctx := context.Background()
	taskID, agentID, projectID := seedExecTaskAndAgent(t, s)
	exec, err := svc.CreateExecution(ctx, taskID, agentID, projectID)
	require.NoError(t, err)
	// Row is in ASSIGNED. Review requires REVIEW.
	body := map[string]any{"accepted": true}
	w := doExecutionRequest(r, http.MethodPatch, "/v1/executions/"+exec.ExecutionID.String()+"/review", projectID, body)
	require.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_STATE_TRANSITION")
}

func TestExecutionHandler_Review_CrossTenant_404(t *testing.T) {
	r, svc, s := newExecutionTestRouter(t, uuid.NewString())
	_, _, exec := seedExecutionInReview(t, svc, s)

	body := map[string]any{"accepted": true}
	w := doExecutionRequest(r, http.MethodPatch, "/v1/executions/"+exec.ExecutionID.String()+"/review", uuid.New(), body)
	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "CROSS_TENANT_BLOCKED")
}

// ----------------------------------------------------------------------------
// DELETE /v1/executions/:id (B-001 operator cancel)
// ----------------------------------------------------------------------------

func TestExecutionHandler_Cancel_204_FromAssigned(t *testing.T) {
	r, svc, s := newExecutionTestRouter(t, uuid.NewString())
	ctx := context.Background()
	taskID, agentID, projectID := seedExecTaskAndAgent(t, s)
	exec, err := svc.CreateExecution(ctx, taskID, agentID, projectID)
	require.NoError(t, err)

	w := doExecutionRequest(r, http.MethodDelete, "/v1/executions/"+exec.ExecutionID.String(), projectID, nil)
	require.Equal(t, http.StatusNoContent, w.Code)

	final, err := svc.GetExecution(ctx, exec.ExecutionID, projectID)
	require.NoError(t, err)
	assert.Equal(t, model.ExecutionStatusFailed, final.Status)
	require.NotNil(t, final.ErrorMessage)
	assert.Equal(t, "cancelled by operator", *final.ErrorMessage)
}

func TestExecutionHandler_Cancel_204_FromRunning(t *testing.T) {
	r, svc, s := newExecutionTestRouter(t, uuid.NewString())
	ctx := context.Background()
	taskID, agentID, projectID := seedExecTaskAndAgent(t, s)
	exec, err := svc.CreateExecution(ctx, taskID, agentID, projectID)
	require.NoError(t, err)
	updated, err := svc.UpdateExecutionStatus(ctx, exec.ExecutionID, model.ExecutionStatusRunning, nil, projectID)
	require.NoError(t, err)
	require.Equal(t, model.ExecutionStatusRunning, updated.Status)

	w := doExecutionRequest(r, http.MethodDelete, "/v1/executions/"+exec.ExecutionID.String(), projectID, nil)
	require.Equal(t, http.StatusNoContent, w.Code)

	final, err := svc.GetExecution(ctx, exec.ExecutionID, projectID)
	require.NoError(t, err)
	assert.Equal(t, model.ExecutionStatusFailed, final.Status)
}

func TestExecutionHandler_Cancel_409_FromCompleted(t *testing.T) {
	r, svc, s := newExecutionTestRouter(t, uuid.NewString())
	_, projectID, exec := seedExecutionInReview(t, svc, s)
	// Land it in COMPLETED first.
	_, err := svc.ReviewExecution(context.Background(), exec.ExecutionID, true, "", projectID)
	require.NoError(t, err)

	w := doExecutionRequest(r, http.MethodDelete, "/v1/executions/"+exec.ExecutionID.String(), projectID, nil)
	require.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_STATE_TRANSITION")
}

func TestExecutionHandler_Cancel_CrossTenant_404(t *testing.T) {
	r, svc, s := newExecutionTestRouter(t, uuid.NewString())
	ctx := context.Background()
	taskID, agentID, projectID := seedExecTaskAndAgent(t, s)
	exec, err := svc.CreateExecution(ctx, taskID, agentID, projectID)
	require.NoError(t, err)

	w := doExecutionRequest(r, http.MethodDelete, "/v1/executions/"+exec.ExecutionID.String(), uuid.New(), nil)
	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "CROSS_TENANT_BLOCKED")
}
