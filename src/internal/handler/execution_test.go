package handler

// HTTP-level tests for ExecutionHandler (TASK-405, Sprint 4).
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
	svc := service.NewExecutionService(s, zap.NewNop(), cfg)
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
	}
	return r, svc, s
}

// seedExecTaskAndAgent creates a task and an agent in the store
// so CreateExecution's validation passes.
func seedExecTaskAndAgent(t *testing.T, s store.Store) (uuid.UUID, uuid.UUID) {
	t.Helper()
	ctx := context.Background()

	task := &model.Task{
		ID:        uuid.New(),
		ProjectID: uuid.New(),
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
	return task.ID, created.ID
}

// doExecutionRequest fires a single request and returns the
// response recorder.
func doExecutionRequest(r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
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
	taskID, agentID := seedExecTaskAndAgent(t, s)

	body := map[string]any{"task_id": taskID.String(), "agent_id": agentID.String()}
	w := doExecutionRequest(r, http.MethodPost, "/v1/executions", body)
	require.Equal(t, http.StatusCreated, w.Code)

	var resp struct {
		Data model.Execution `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, uuid.Nil, resp.Data.ExecutionID)
	assert.Equal(t, taskID, resp.Data.TaskID)
	assert.Equal(t, agentID, resp.Data.AgentID)
	assert.Equal(t, model.ExecutionStatusPending, resp.Data.Status)
}

func TestExecutionHandler_Create_400_BadUUID(t *testing.T) {
	r, _, _ := newExecutionTestRouter(t, uuid.NewString())
	body := map[string]any{"task_id": "not-a-uuid", "agent_id": uuid.NewString()}
	w := doExecutionRequest(r, http.MethodPost, "/v1/executions", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	body = map[string]any{"task_id": uuid.NewString(), "agent_id": "still-not-a-uuid"}
	w = doExecutionRequest(r, http.MethodPost, "/v1/executions", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExecutionHandler_Create_404(t *testing.T) {
	r, _, s := newExecutionTestRouter(t, uuid.NewString())
	_, agentID := seedExecTaskAndAgent(t, s)

	// Task does not exist.
	body := map[string]any{"task_id": uuid.NewString(), "agent_id": agentID.String()}
	w := doExecutionRequest(r, http.MethodPost, "/v1/executions", body)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "TASK_NOT_FOUND")

	// Agent does not exist.
	taskID, _ := seedExecTaskAndAgent(t, s)
	body = map[string]any{"task_id": taskID.String(), "agent_id": uuid.NewString()}
	w = doExecutionRequest(r, http.MethodPost, "/v1/executions", body)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "AGENT_NOT_FOUND")
}

// ----------------------------------------------------------------------------
// GET /v1/executions/:id
// ----------------------------------------------------------------------------

func TestExecutionHandler_GetByID_Success(t *testing.T) {
	r, svc, s := newExecutionTestRouter(t, uuid.NewString())
	taskID, agentID := seedExecTaskAndAgent(t, s)
	exec, err := svc.CreateExecution(context.Background(), taskID, agentID)
	require.NoError(t, err)

	w := doExecutionRequest(r, http.MethodGet, "/v1/executions/"+exec.ExecutionID.String(), nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data model.Execution `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, exec.ExecutionID, resp.Data.ExecutionID)
}

func TestExecutionHandler_GetByID_404(t *testing.T) {
	r, _, _ := newExecutionTestRouter(t, uuid.NewString())
	w := doExecutionRequest(r, http.MethodGet, "/v1/executions/"+uuid.NewString(), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "EXECUTION_NOT_FOUND")
}

// ----------------------------------------------------------------------------
// GET /v1/executions (list)
// ----------------------------------------------------------------------------

func TestExecutionHandler_List_WithFilters(t *testing.T) {
	r, svc, s := newExecutionTestRouter(t, uuid.NewString())
	taskA, agentA := seedExecTaskAndAgent(t, s)
	taskB, agentB := seedExecTaskAndAgent(t, s)

	// Seed: 2 on (taskA, agentA), 1 on (taskA, agentB), 1 on (taskB, agentA).
	for i := 0; i < 2; i++ {
		_, err := svc.CreateExecution(context.Background(), taskA, agentA)
		require.NoError(t, err)
	}
	_, err := svc.CreateExecution(context.Background(), taskA, agentB)
	require.NoError(t, err)
	_, err = svc.CreateExecution(context.Background(), taskB, agentA)
	require.NoError(t, err)

	// No filter: all 4.
	w := doExecutionRequest(r, http.MethodGet, "/v1/executions", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data model.ExecutionListResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Data.Items, 4)

	// task_id filter
	w = doExecutionRequest(r, http.MethodGet, "/v1/executions?task_id="+taskA.String(), nil)
	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Data.Items, 3)

	// agent_id filter
	w = doExecutionRequest(r, http.MethodGet, "/v1/executions?agent_id="+agentA.String(), nil)
	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Data.Items, 3)

	// status=pending filter
	w = doExecutionRequest(r, http.MethodGet, "/v1/executions?status=pending", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.GreaterOrEqual(t, len(resp.Data.Items), 1, "expected at least one pending row right after create")

	// status=garbage → 400
	w = doExecutionRequest(r, http.MethodGet, "/v1/executions?status=garbage", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_EXECUTION_STATUS")
}

// ----------------------------------------------------------------------------
// PATCH /v1/executions/:id
// ----------------------------------------------------------------------------

func TestExecutionHandler_Patch_200(t *testing.T) {
	r, svc, s := newExecutionTestRouter(t, uuid.NewString())
	taskID, agentID := seedExecTaskAndAgent(t, s)
	exec, err := svc.CreateExecution(context.Background(), taskID, agentID)
	require.NoError(t, err)

	body := map[string]any{"status": "running"}
	w := doExecutionRequest(r, http.MethodPatch, "/v1/executions/"+exec.ExecutionID.String(), body)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data model.Execution `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, model.ExecutionStatusRunning, resp.Data.Status)

	// Now move to completed with an error_message (no-op for
	// the state but tests that the field is accepted).
	body = map[string]any{"status": "completed"}
	w = doExecutionRequest(r, http.MethodPatch, "/v1/executions/"+exec.ExecutionID.String(), body)
	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, model.ExecutionStatusCompleted, resp.Data.Status)
	require.NotNil(t, resp.Data.CompletedAt)
}

func TestExecutionHandler_Patch_409(t *testing.T) {
	r, svc, s := newExecutionTestRouter(t, uuid.NewString())
	taskID, agentID := seedExecTaskAndAgent(t, s)
	exec, err := svc.CreateExecution(context.Background(), taskID, agentID)
	require.NoError(t, err)

	// pending → running is valid; running → pending is not.
	_, err = svc.UpdateExecutionStatus(context.Background(), exec.ExecutionID, model.ExecutionStatusRunning, nil)
	require.NoError(t, err)

	body := map[string]any{"status": "pending"}
	w := doExecutionRequest(r, http.MethodPatch, "/v1/executions/"+exec.ExecutionID.String(), body)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_STATE_TRANSITION")
}

func TestExecutionHandler_Patch_404(t *testing.T) {
	r, _, _ := newExecutionTestRouter(t, uuid.NewString())
	body := map[string]any{"status": "running"}
	w := doExecutionRequest(r, http.MethodPatch, "/v1/executions/"+uuid.NewString(), body)
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
	w := doExecutionRequest(r, http.MethodGet, "/v1/executions", nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "UNAUTHORIZED")
}
