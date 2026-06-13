package handler

// HTTP-level tests for AssignmentHandler (TASK-404, Sprint 4).
//
// Strategy: drive Gin with a real router and a hand-rolled
// mockAssignmentService (the handler depends on the
// service.AssignmentService struct, not an interface, so we cannot
// use a testify mock — we use a hand-rolled shim that records
// calls and returns canned responses). Covers:
//   - POST /v1/tasks/:id/assign  (api-spec.md §3.1)
//   - GET  /v1/tasks/:id/history

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAssignmentService is a hand-rolled shim that satisfies the
// service.AssignmentService call sites. The real service is a
// struct (concrete type) so the testify mock pattern doesn't apply
// cleanly; the shim is simpler and matches the existing
// pattern in the codebase for non-interface services.
type mockAssignmentService struct {
	mu sync.Mutex

	// Response to return from AssignTaskToAgent.
	assignResult *service.AssignmentResult
	assignErr    *service.Error

	// Last arguments captured from AssignTaskToAgent.
	lastCtx             context.Context
	lastTaskID          uuid.UUID
	lastAgentID         uuid.UUID
	lastNotes           string
	lastAssignedBy      *uuid.UUID
	lastCapsRequired    []string
	lastCallerProjectID uuid.UUID
	assignCallCount     int

	// Response to return from ListAssignmentHistory.
	listResult []*model.AssignmentEvent
	listErr    *service.Error

	listCallCount int
}

func (m *mockAssignmentService) AssignTaskToAgent(
	ctx context.Context,
	taskID uuid.UUID,
	agentID uuid.UUID,
	notes string,
	assignedBy *uuid.UUID,
	capabilitiesRequired []string,
	callerProjectID uuid.UUID,
) (*service.AssignmentResult, *service.Error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastCtx = ctx
	m.lastTaskID = taskID
	m.lastAgentID = agentID
	m.lastNotes = notes
	m.lastAssignedBy = assignedBy
	m.lastCapsRequired = capabilitiesRequired
	m.lastCallerProjectID = callerProjectID
	m.assignCallCount++
	return m.assignResult, m.assignErr
}

func (m *mockAssignmentService) ListAssignmentHistory(ctx context.Context, taskID uuid.UUID, callerProjectID uuid.UUID) ([]*model.AssignmentEvent, *service.Error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastCallerProjectID = callerProjectID
	m.listCallCount++
	return m.listResult, m.listErr
}

// newAssignmentTestRouter wires Gin with the AssignmentHandler
// routes and a captured mock service. The user_id middleware
// stub is configurable so we can test both authenticated
// (user_id set) and anonymous (user_id absent) request paths.
func newAssignmentTestRouter(t *testing.T, withUserID string) (*gin.Engine, *mockAssignmentService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("request_id", "test-rid-assign-001")
		if withUserID != "" {
			c.Set("user_id", withUserID)
		}
		c.Next()
	})
	m := &mockAssignmentService{}
	h := NewAssignmentHandler(m)
	v1 := r.Group("/v1")
	{
		v1.POST("/tasks/:id/assign", h.AssignTask)
		v1.GET("/tasks/:id/history", h.ListHistory)
	}
	return r, m
}

func doAssignmentRequest(r *gin.Engine, method, path, projectID string, body interface{}) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if projectID != "" {
		req.Header.Set("X-Project-ID", projectID)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ---- POST /v1/tasks/:id/assign ---------------------------------------

func TestAssignmentHandler_Assign_Success(t *testing.T) {
	userID := uuid.New()
	r, m := newAssignmentTestRouter(t, userID.String())
	projectID := uuid.New()
	taskID := uuid.New()
	agentID := uuid.New()
	now := time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)

	agentIDPtr := agentID
	userIDPtr := userID
	m.assignResult = &service.AssignmentResult{
		Task: &model.Task{
			ID: taskID, ProjectID: uuid.New(), Title: "x", Status: model.TaskInProgress,
			AssigneeID: agentID, UpdatedAt: now,
		},
		// Event.Notes is set to "first assignment" to simulate
		// what the service would return after persisting the
		// request's notes value (F-017 fix). The handler no
		// longer mutates the response; the value in the body
		// is whatever the service returns.
		Event: &model.AssignmentEvent{
			ID: uuid.New(), AssignmentID: uuid.New(), TaskID: taskID, AgentID: &agentIDPtr,
			AssignedBy: &userIDPtr, AssignedAt: now,
			Action: model.AssignmentActionAssign,
			Notes:  "first assignment",
		},
		Idempotent: false,
	}

	body := map[string]interface{}{
		"agent_id":              agentID.String(),
		"capabilities_required": []string{"coding"},
		"notes":                 "first assignment",
	}
	w := doAssignmentRequest(r, http.MethodPost, "/v1/tasks/"+taskID.String()+"/assign", projectID.String(), body)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data, ok := resp["data"].(map[string]interface{})
	require.True(t, ok, "expected data envelope")
	task, ok := data["task"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, agentID.String(), task["assignee_id"])
	event, ok := data["event"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "assign", event["action"])
	assert.Equal(t, "first assignment", event["notes"])

	// Verify captured args.
	assert.Equal(t, taskID, m.lastTaskID)
	assert.Equal(t, agentID, m.lastAgentID)
	// F-017: the handler must pass req.Notes through to the
	// service so the row lands in the DB.
	assert.Equal(t, "first assignment", m.lastNotes)
	require.NotNil(t, m.lastAssignedBy)
	assert.Equal(t, userID, *m.lastAssignedBy)
	assert.Equal(t, []string{"coding"}, m.lastCapsRequired)
	assert.Equal(t, 1, m.assignCallCount)
}

// TestAssignmentHandler_Assign_NoInMemoryNotesMutation is the
// F-017 regression: if the service returns an event with empty
// Notes (e.g., the caller did not pass notes), the response
// must reflect that — the handler must not synthesise notes
// after the fact.
func TestAssignmentHandler_Assign_NoInMemoryNotesMutation(t *testing.T) {
	userID := uuid.New()
	r, m := newAssignmentTestRouter(t, userID.String())
	projectID := uuid.New()
	taskID := uuid.New()
	agentID := uuid.New()
	now := time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)

	agentIDPtr := agentID
	userIDPtr := userID
	// Service returns an event with NO notes (notes:"").
	m.assignResult = &service.AssignmentResult{
		Task: &model.Task{
			ID: taskID, AssigneeID: agentID, UpdatedAt: now,
		},
		Event: &model.AssignmentEvent{
			ID: uuid.New(), AssignmentID: uuid.New(), TaskID: taskID, AgentID: &agentIDPtr,
			AssignedBy: &userIDPtr, AssignedAt: now,
			Action: model.AssignmentActionAssign,
			// Notes intentionally empty.
		},
		Idempotent: false,
	}

	// Request has notes="would-be-ignored" — the handler must
	// pass it to the service but the response must reflect
	// what the service returned (i.e., empty Notes).
	body := map[string]interface{}{
		"agent_id": agentID.String(),
		"notes":    "would-be-ignored",
	}
	w := doAssignmentRequest(r, http.MethodPost, "/v1/tasks/"+taskID.String()+"/assign", projectID.String(), body)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]interface{})
	event := data["event"].(map[string]interface{})
	assert.Equal(t, "", event["notes"], "handler must not inject notes into the response after the fact")
	// The handler did pass the notes through to the service;
	// the service chose to persist empty. That is correct.
	assert.Equal(t, "would-be-ignored", m.lastNotes)
}

func TestAssignmentHandler_Assign_Idempotent(t *testing.T) {
	userID := uuid.New()
	r, m := newAssignmentTestRouter(t, userID.String())
	projectID := uuid.New()
	taskID := uuid.New()
	agentID := uuid.New()

	m.assignResult = &service.AssignmentResult{
		Task: &model.Task{ID: taskID, AssigneeID: agentID},
		// Event nil — the service returns nil for idempotent.
		Idempotent: true,
	}

	body := map[string]interface{}{"agent_id": agentID.String()}
	w := doAssignmentRequest(r, http.MethodPost, "/v1/tasks/"+taskID.String()+"/assign", projectID.String(), body)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, true, data["idempotent"])
	assert.Nil(t, data["event"])
}

func TestAssignmentHandler_Assign_TaskNotFound(t *testing.T) {
	userID := uuid.New()
	r, m := newAssignmentTestRouter(t, userID.String())
	projectID := uuid.New()
	m.assignErr = &service.Error{
		Status: 404, Code: "NOT_FOUND", Message: "Task not found",
	}

	body := map[string]interface{}{"agent_id": uuid.New().String()}
	w := doAssignmentRequest(r, http.MethodPost, "/v1/tasks/"+uuid.New().String()+"/assign", projectID.String(), body)
	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	errBlock := resp["error"].(map[string]interface{})
	assert.Equal(t, "NOT_FOUND", errBlock["code"])
}

func TestAssignmentHandler_Assign_AgentNotFound(t *testing.T) {
	userID := uuid.New()
	r, m := newAssignmentTestRouter(t, userID.String())
	projectID := uuid.New()
	m.assignErr = &service.Error{
		Status: 404, Code: "NOT_FOUND", Message: "Agent not found",
	}

	body := map[string]interface{}{"agent_id": uuid.New().String()}
	w := doAssignmentRequest(r, http.MethodPost, "/v1/tasks/"+uuid.New().String()+"/assign", projectID.String(), body)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAssignmentHandler_Assign_CapabilityMismatch(t *testing.T) {
	userID := uuid.New()
	r, m := newAssignmentTestRouter(t, userID.String())
	projectID := uuid.New()
	m.assignErr = &service.Error{
		Status: 409, Code: "CAPABILITY_MISMATCH", Message: "agent does not hold all required capabilities",
	}

	body := map[string]interface{}{"agent_id": uuid.New().String()}
	w := doAssignmentRequest(r, http.MethodPost, "/v1/tasks/"+uuid.New().String()+"/assign", projectID.String(), body)
	assert.Equal(t, http.StatusConflict, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	errBlock := resp["error"].(map[string]interface{})
	assert.Equal(t, "CAPABILITY_MISMATCH", errBlock["code"])
}

func TestAssignmentHandler_Assign_MissingAgentID(t *testing.T) {
	userID := uuid.New()
	r, m := newAssignmentTestRouter(t, userID.String())
	projectID := uuid.New()
	body := map[string]interface{}{} // no agent_id
	w := doAssignmentRequest(r, http.MethodPost, "/v1/tasks/"+uuid.New().String()+"/assign", projectID.String(), body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, m.assignCallCount, "service must not be called on bad request")
}

func TestAssignmentHandler_Assign_InvalidAgentID(t *testing.T) {
	userID := uuid.New()
	r, m := newAssignmentTestRouter(t, userID.String())
	projectID := uuid.New()
	body := map[string]interface{}{"agent_id": "not-a-uuid"}
	w := doAssignmentRequest(r, http.MethodPost, "/v1/tasks/"+uuid.New().String()+"/assign", projectID.String(), body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, m.assignCallCount)
}

func TestAssignmentHandler_Assign_InvalidTaskID(t *testing.T) {
	userID := uuid.New()
	r, m := newAssignmentTestRouter(t, userID.String())
	projectID := uuid.New()
	body := map[string]interface{}{"agent_id": uuid.New().String()}
	w := doAssignmentRequest(r, http.MethodPost, "/v1/tasks/not-a-uuid/assign", projectID.String(), body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, m.assignCallCount)
}

func TestAssignmentHandler_Assign_AnonymousCaller_AssignedByNil(t *testing.T) {
	// No user_id in middleware context — system-initiated caller.
	r, m := newAssignmentTestRouter(t, "")
	projectID := uuid.New()
	taskID := uuid.New()
	agentID := uuid.New()

	m.assignResult = &service.AssignmentResult{
		Task: &model.Task{ID: taskID, AssigneeID: agentID},
		Event: &model.AssignmentEvent{
			ID: uuid.New(), AssignmentID: uuid.New(), TaskID: taskID,
			AgentID: func() *uuid.UUID { id := agentID; return &id }(),
			// AssignedBy intentionally nil
			AssignedAt: time.Now().UTC(),
			Action:     model.AssignmentActionAssign,
		},
	}

	body := map[string]interface{}{"agent_id": agentID.String()}
	w := doAssignmentRequest(r, http.MethodPost, "/v1/tasks/"+taskID.String()+"/assign", projectID.String(), body)
	assert.Equal(t, http.StatusOK, w.Code)

	// Service must have been called with assignedBy == nil.
	assert.Nil(t, m.lastAssignedBy, "anonymous caller must propagate nil assigned_by")
}

// ---- GET /v1/tasks/:id/history ---------------------------------------

func TestAssignmentHandler_History_Success(t *testing.T) {
	userID := uuid.New()
	r, m := newAssignmentTestRouter(t, userID.String())
	projectID := uuid.New()
	taskID := uuid.New()
	now := time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)

	a := uuid.New()
	agentA := a
	agentB := uuid.New()
	m.listResult = []*model.AssignmentEvent{
		{ID: uuid.New(), AssignmentID: uuid.New(), TaskID: taskID, AgentID: &agentB, AssignedAt: now, Action: model.AssignmentActionReassign},
		{ID: uuid.New(), AssignmentID: uuid.New(), TaskID: taskID, AgentID: &agentA, AssignedAt: now.Add(-time.Hour), Action: model.AssignmentActionAssign},
	}

	w := doAssignmentRequest(r, http.MethodGet, "/v1/tasks/"+taskID.String()+"/history", projectID.String(), nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].([]interface{})
	assert.Equal(t, 2, len(data))
	meta := resp["meta"].(map[string]interface{})
	assert.Equal(t, float64(2), meta["count"])
}

func TestAssignmentHandler_History_EmptyForNewTask(t *testing.T) {
	userID := uuid.New()
	r, m := newAssignmentTestRouter(t, userID.String())
	projectID := uuid.New()
	m.listResult = []*model.AssignmentEvent{}

	w := doAssignmentRequest(r, http.MethodGet, "/v1/tasks/"+uuid.New().String()+"/history", projectID.String(), nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].([]interface{})
	assert.Equal(t, 0, len(data))
}

func TestAssignmentHandler_History_TaskNotFound(t *testing.T) {
	userID := uuid.New()
	r, m := newAssignmentTestRouter(t, userID.String())
	projectID := uuid.New()
	m.listErr = &service.Error{Status: 404, Code: "NOT_FOUND", Message: "Task not found"}

	w := doAssignmentRequest(r, http.MethodGet, "/v1/tasks/"+uuid.New().String()+"/history", projectID.String(), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAssignmentHandler_History_InvalidTaskID(t *testing.T) {
	userID := uuid.New()
	r, m := newAssignmentTestRouter(t, userID.String())
	projectID := uuid.New()
	w := doAssignmentRequest(r, http.MethodGet, "/v1/tasks/not-a-uuid/history", projectID.String(), nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, m.listCallCount)
}

// ---- F-014 cross-tenant handler tests (Sprint 5) -----------------

// TestAssignmentHandler_Assign_CrossTenant: mock returns 404
// CROSS_TENANT_BLOCKED, handler should pass it through.
func TestAssignmentHandler_Assign_CrossTenant(t *testing.T) {
	userID := uuid.New()
	r, m := newAssignmentTestRouter(t, userID.String())
	projectID := uuid.New()
	var assignedBy uuid.UUID = uuid.New()
	_ = assignedBy
	m.assignErr = &service.Error{Status: 404, Code: "CROSS_TENANT_BLOCKED", Message: "blocked"}

	body := map[string]interface{}{"agent_id": uuid.New().String()}
	w := doAssignmentRequest(r, http.MethodPost, "/v1/tasks/"+uuid.New().String()+"/assign", projectID.String(), body)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, 1, m.assignCallCount)
	assert.Equal(t, projectID, m.lastCallerProjectID)
}

// TestAssignmentHandler_Assign_MissingProjectHeader: empty X-Project-ID
// returns 400 MISSING_PROJECT_HEADER without calling the service.
func TestAssignmentHandler_Assign_MissingProjectHeader(t *testing.T) {
	userID := uuid.New()
	r, m := newAssignmentTestRouter(t, userID.String())

	body := map[string]interface{}{"agent_id": uuid.New().String()}
	w := doAssignmentRequest(r, http.MethodPost, "/v1/tasks/"+uuid.New().String()+"/assign", "", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, m.assignCallCount)
}

// TestAssignmentHandler_History_CrossTenant: mock returns 404
// CROSS_TENANT_BLOCKED, handler should pass it through.
func TestAssignmentHandler_History_CrossTenant(t *testing.T) {
	userID := uuid.New()
	r, m := newAssignmentTestRouter(t, userID.String())
	projectID := uuid.New()
	m.listErr = &service.Error{Status: 404, Code: "CROSS_TENANT_BLOCKED", Message: "blocked"}

	w := doAssignmentRequest(r, http.MethodGet, "/v1/tasks/"+uuid.New().String()+"/history", projectID.String(), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, 1, m.listCallCount)
	assert.Equal(t, projectID, m.lastCallerProjectID)
}

// TestAssignmentHandler_History_MissingProjectHeader: empty X-Project-ID
// returns 400 MISSING_PROJECT_HEADER without calling the service.
func TestAssignmentHandler_History_MissingProjectHeader(t *testing.T) {
	userID := uuid.New()
	r, m := newAssignmentTestRouter(t, userID.String())

	w := doAssignmentRequest(r, http.MethodGet, "/v1/tasks/"+uuid.New().String()+"/history", "", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, m.listCallCount)
}
