package handler

// HTTP-level tests for the agent handler (TASK-402, Sprint 4).
//
// Strategy: drive Gin with a real router and a mock AgentService
// (the handler depends on the interface, not the concrete type).
// Each test asserts the response status, body, and that the
// service was called with the right arguments.

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
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Error aliases service.Error so test files can use *Error locally
// (matches production code that uses *service.Error).
type Error = service.Error

// mockAgentService is a hand-rolled mock that returns canned
// responses. It mirrors the service.AgentService interface exactly.
type mockAgentService struct {
	mock.Mock
}

func (m *mockAgentService) CreateAgent(ctx context.Context, req service.CreateAgentRequest) (*model.Agent, *Error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Get(1).(*Error)
	}
	return args.Get(0).(*model.Agent), args.Get(1).(*Error)
}
func (m *mockAgentService) GetAgent(ctx context.Context, id uuid.UUID, callerProjectID uuid.UUID) (*model.Agent, *Error) {
	args := m.Called(ctx, id, callerProjectID)
	if args.Get(0) == nil {
		return nil, args.Get(1).(*Error)
	}
	return args.Get(0).(*model.Agent), args.Get(1).(*Error)
}
func (m *mockAgentService) ListAgents(ctx context.Context, req service.ListAgentsRequest) (*service.ListAgentsResult, *Error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Get(1).(*Error)
	}
	return args.Get(0).(*service.ListAgentsResult), args.Get(1).(*Error)
}
func (m *mockAgentService) UpdateAgent(ctx context.Context, id uuid.UUID, callerProjectID uuid.UUID, req service.UpdateAgentRequest) (*model.Agent, *Error) {
	args := m.Called(ctx, id, callerProjectID, req)
	if args.Get(0) == nil {
		return nil, args.Get(1).(*Error)
	}
	return args.Get(0).(*model.Agent), args.Get(1).(*Error)
}
func (m *mockAgentService) RetireAgent(ctx context.Context, id uuid.UUID, callerProjectID uuid.UUID, force bool) *Error {
	args := m.Called(ctx, id, callerProjectID, force)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*Error)
}
func (m *mockAgentService) ListAgentCapabilities(ctx context.Context, id uuid.UUID, callerProjectID uuid.UUID) ([]*model.AgentCapabilityView, *Error) {
	args := m.Called(ctx, id, callerProjectID)
	if args.Get(0) == nil {
		return nil, args.Get(1).(*Error)
	}
	return args.Get(0).([]*model.AgentCapabilityView), args.Get(1).(*Error)
}
func (m *mockAgentService) ListCapabilities(ctx context.Context, req service.ListCapabilitiesRequest) (*service.ListCapabilitiesResult, *Error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Get(1).(*Error)
	}
	return args.Get(0).(*service.ListCapabilitiesResult), args.Get(1).(*Error)
}

// newTestRouter wires Gin with the agent handler routes and a
// captured mock service. The X-Project-ID header is set on every
// test request by the helper below.
func newTestRouter(t *testing.T) (*gin.Engine, *mockAgentService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Set a stable request_id so error envelope assertions are
	// deterministic.
	r.Use(func(c *gin.Context) {
		c.Set("request_id", "test-rid-001")
		c.Next()
	})
	m := &mockAgentService{}
	h := NewAgentHandler(m)
	v1 := r.Group("/v1")
	{
		v1.POST("/agents", h.Create)
		v1.GET("/agents", h.List)
		v1.GET("/agents/:id", h.Get)
		v1.PUT("/agents/:id", h.Update)
		v1.DELETE("/agents/:id", h.Delete)
		// NOTE (TASK-403): /v1/agents/:id/capabilities and
		// /v1/capabilities are now wired in capability_test.go on
		// a CapabilityHandler. They were moved out of AgentHandler
		// by the TASK-403 brief so the agent file stays focused on
		// the agent CRUD endpoints.
	}
	return r, m
}

func doRequest(r *gin.Engine, method, path, projectID string, body interface{}) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	if projectID != "" {
		req.Header.Set("X-Project-ID", projectID)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ---- POST /v1/agents -------------------------------------------------

func TestAgentHandler_Create_Success(t *testing.T) {
	r, m := newTestRouter(t)
	projectID := uuid.New()
	createdID := uuid.New()

	m.On("CreateAgent", mock.Anything, mock.MatchedBy(func(req service.CreateAgentRequest) bool {
		return req.ProjectID == projectID && req.Name == "alpha" && len(req.Capabilities) == 1
	})).Return(&model.Agent{
		ID: createdID, ProjectID: projectID, Name: "alpha", Role: "developer",
		Status: model.AgentInitializing, Capabilities: []string{"coding"},
		Version: 1, CreatedAt: parseTime("2026-06-12T10:00:00Z"),
		UpdatedAt: parseTime("2026-06-12T10:00:00Z"),
	}, (*service.Error)(nil))

	body := map[string]interface{}{
		"name":         "alpha",
		"role":         "developer",
		"capabilities": []string{"coding"},
	}
	w := doRequest(r, http.MethodPost, "/v1/agents", projectID.String(), body)

	assert.Equal(t, http.StatusCreated, w.Code)
	m.AssertExpectations(t)
	var resp map[string]interface{}
	requireUnmarshal(t, w, &resp)
	data, ok := resp["data"].(map[string]interface{})
	assert.True(t, ok, "expected top-level 'data' envelope, got body=%s", w.Body.String())
	assert.Equal(t, createdID.String(), data["id"])
	assert.Equal(t, "alpha", data["name"])
	assert.Equal(t, float64(1), data["version"])
	assert.Equal(t, "initializing", data["status"])
}

func TestAgentHandler_Create_ValidationError(t *testing.T) {
	r, m := newTestRouter(t)
	projectID := uuid.New()

	m.On("CreateAgent", mock.Anything, mock.Anything).Return(
		(*model.Agent)(nil),
		&service.Error{Status: 422, Code: "CAPABILITY_NOT_FOUND", Message: "unknown capability"},
	)

	body := map[string]interface{}{
		"name":         "alpha",
		"role":         "developer",
		"capabilities": []string{"nope"},
	}
	w := doRequest(r, http.MethodPost, "/v1/agents", projectID.String(), body)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	var resp map[string]interface{}
	requireUnmarshal(t, w, &resp)
	errBlock := resp["error"].(map[string]interface{})
	assert.Equal(t, "CAPABILITY_NOT_FOUND", errBlock["code"])
	assert.Equal(t, "test-rid-001", resp["request_id"])
}

// ---- GET /v1/agents --------------------------------------------------

func TestAgentHandler_List_Success(t *testing.T) {
	r, m := newTestRouter(t)
	projectID := uuid.New()

	m.On("ListAgents", mock.Anything, mock.MatchedBy(func(req service.ListAgentsRequest) bool {
		return req.ProjectID == projectID && req.Limit == 0 && !req.IncludeRetired
	})).Return(&service.ListAgentsResult{
		Data: []*model.Agent{
			{ID: uuid.New(), ProjectID: projectID, Name: "a", Role: "developer",
				Status: model.AgentIdle, Capabilities: []string{"coding"}, Version: 1,
				CreatedAt: parseTime("2026-06-12T10:00:00Z"),
				UpdatedAt: parseTime("2026-06-12T10:00:00Z")},
		},
		HasMore: false,
	}, (*service.Error)(nil))

	w := doRequest(r, http.MethodGet, "/v1/agents", projectID.String(), nil)
	assert.Equal(t, http.StatusOK, w.Code)
	m.AssertExpectations(t)
}

// ---- GET /v1/agents/:id ---------------------------------------------

func TestAgentHandler_Get_Success(t *testing.T) {
	r, m := newTestRouter(t)
	id := uuid.New()
	projectID := uuid.New()

	m.On("GetAgent", mock.Anything, id, projectID).Return(
		&model.Agent{ID: id, ProjectID: projectID, Name: "alpha", Role: "developer",
			Status: model.AgentIdle, Capabilities: []string{"coding"}, Version: 1,
			CreatedAt: parseTime("2026-06-12T10:00:00Z"),
			UpdatedAt: parseTime("2026-06-12T10:00:00Z")},
		(*service.Error)(nil))

	w := doRequest(r, http.MethodGet, "/v1/agents/"+id.String(), projectID.String(), nil)
	assert.Equal(t, http.StatusOK, w.Code)
	m.AssertExpectations(t)
}

func TestAgentHandler_Get_NotFound(t *testing.T) {
	r, m := newTestRouter(t)
	id := uuid.New()
	projectID := uuid.New()

	m.On("GetAgent", mock.Anything, id, projectID).Return(
		(*model.Agent)(nil),
		&service.Error{Status: 404, Code: "NOT_FOUND", Message: "Agent not found"})

	w := doRequest(r, http.MethodGet, "/v1/agents/"+id.String(), projectID.String(), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
	m.AssertExpectations(t)
}

func TestAgentHandler_Get_InvalidID(t *testing.T) {
	r, _ := newTestRouter(t)
	w := doRequest(r, http.MethodGet, "/v1/agents/not-a-uuid", uuid.New().String(), nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---- PUT /v1/agents/:id ---------------------------------------------

func TestAgentHandler_Update_VersionConflict(t *testing.T) {
	r, m := newTestRouter(t)
	id := uuid.New()
	projectID := uuid.New()

	m.On("UpdateAgent", mock.Anything, id, projectID, mock.Anything).Return(
		(*model.Agent)(nil),
		&service.Error{Status: 409, Code: "VERSION_CONFLICT", Message: "stale version"})

	body := map[string]interface{}{"role": "qa", "version": 1}
	w := doRequest(r, http.MethodPut, "/v1/agents/"+id.String(), projectID.String(), body)
	assert.Equal(t, http.StatusConflict, w.Code)
	m.AssertExpectations(t)
}

// ---- DELETE /v1/agents/:id ------------------------------------------

func TestAgentHandler_Delete_Success(t *testing.T) {
	r, m := newTestRouter(t)
	id := uuid.New()
	projectID := uuid.New()

	m.On("RetireAgent", mock.Anything, id, projectID, false).Return((*service.Error)(nil))

	w := doRequest(r, http.MethodDelete, "/v1/agents/"+id.String(), projectID.String(), nil)
	assert.Equal(t, http.StatusNoContent, w.Code)
	m.AssertExpectations(t)
}

func TestAgentHandler_Delete_NotFound(t *testing.T) {
	r, m := newTestRouter(t)
	id := uuid.New()
	projectID := uuid.New()

	m.On("RetireAgent", mock.Anything, id, projectID, false).Return(
		&service.Error{Status: 404, Code: "NOT_FOUND", Message: "Agent not found"})

	w := doRequest(r, http.MethodDelete, "/v1/agents/"+id.String(), projectID.String(), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
	m.AssertExpectations(t)
}


// ---- F-013 cross-tenant handler tests (Sprint 5) ------------------

func TestAgentHandler_Get_CrossTenant(t *testing.T) {
	r, m := newTestRouter(t)
	id := uuid.New()
	projectID := uuid.New()

	m.On("GetAgent", mock.Anything, id, projectID).Return(
		(*model.Agent)(nil),
		&service.Error{Status: 404, Code: "CROSS_TENANT_BLOCKED", Message: "blocked"})

	w := doRequest(r, http.MethodGet, "/v1/agents/"+id.String(), projectID.String(), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
	m.AssertExpectations(t)
}

func TestAgentHandler_Update_CrossTenant(t *testing.T) {
	r, m := newTestRouter(t)
	id := uuid.New()
	projectID := uuid.New()

	m.On("UpdateAgent", mock.Anything, id, projectID, mock.Anything).Return(
		(*model.Agent)(nil),
		&service.Error{Status: 404, Code: "CROSS_TENANT_BLOCKED", Message: "blocked"})

	body := map[string]interface{}{"role": "qa", "version": 1}
	w := doRequest(r, http.MethodPut, "/v1/agents/"+id.String(), projectID.String(), body)
	assert.Equal(t, http.StatusNotFound, w.Code)
	m.AssertExpectations(t)
}

func TestAgentHandler_Delete_CrossTenant(t *testing.T) {
	r, m := newTestRouter(t)
	id := uuid.New()
	projectID := uuid.New()

	m.On("RetireAgent", mock.Anything, id, projectID, false).Return(
		&service.Error{Status: 404, Code: "CROSS_TENANT_BLOCKED", Message: "blocked"})

	w := doRequest(r, http.MethodDelete, "/v1/agents/"+id.String(), projectID.String(), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
	m.AssertExpectations(t)
}

func TestAgentHandler_Get_MissingProjectHeader(t *testing.T) {
	r, _ := newTestRouter(t)
	id := uuid.New()

	// pass empty projectID header — the handler should reject with 400
	w := doRequest(r, http.MethodGet, "/v1/agents/"+id.String(), "", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAgentHandler_Update_MissingProjectHeader(t *testing.T) {
	r, _ := newTestRouter(t)
	id := uuid.New()

	body := map[string]interface{}{"role": "qa", "version": 1}
	w := doRequest(r, http.MethodPut, "/v1/agents/"+id.String(), "", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAgentHandler_Delete_MissingProjectHeader(t *testing.T) {
	r, _ := newTestRouter(t)
	id := uuid.New()

	w := doRequest(r, http.MethodDelete, "/v1/agents/"+id.String(), "", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
// ---- D7 envelope consistency (Sprint 6, TASK-427) ------------------

func TestAgentHandler_Create_DataEnvelope(t *testing.T) {
	r, m := newTestRouter(t)
	projectID := uuid.New()
	createdID := uuid.New()

	m.On("CreateAgent", mock.Anything, mock.Anything).Return(&model.Agent{
		ID: createdID, ProjectID: projectID, Name: "alpha", Role: "developer",
		Status: model.AgentInitializing, Capabilities: []string{"coding"},
		Version: 1, CreatedAt: parseTime("2026-06-12T10:00:00Z"),
		UpdatedAt: parseTime("2026-06-12T10:00:00Z"),
	}, (*service.Error)(nil))

	body := map[string]interface{}{"name": "alpha", "role": "developer", "capabilities": []string{"coding"}}
	w := doRequest(r, http.MethodPost, "/v1/agents", projectID.String(), body)
	assert.Equal(t, http.StatusCreated, w.Code)
	m.AssertExpectations(t)

	var resp map[string]interface{}
	requireUnmarshal(t, w, &resp)
	data, ok := resp["data"].(map[string]interface{})
	assert.True(t, ok, "expected top-level 'data' envelope, got body=%s", w.Body.String())
	assert.Equal(t, createdID.String(), data["id"])
}

func TestAgentHandler_Get_DataEnvelope(t *testing.T) {
	r, m := newTestRouter(t)
	id := uuid.New()
	projectID := uuid.New()

	m.On("GetAgent", mock.Anything, id, projectID).Return(&model.Agent{
		ID: id, ProjectID: projectID, Name: "alpha", Role: "developer",
		Status: model.AgentActive, Capabilities: []string{"coding"},
		Version: 1, CreatedAt: parseTime("2026-06-12T10:00:00Z"),
		UpdatedAt: parseTime("2026-06-12T10:00:00Z"),
	}, (*service.Error)(nil))

	w := doRequest(r, http.MethodGet, "/v1/agents/"+id.String(), projectID.String(), nil)
	assert.Equal(t, http.StatusOK, w.Code)
	m.AssertExpectations(t)

	var resp map[string]interface{}
	requireUnmarshal(t, w, &resp)
	data, ok := resp["data"].(map[string]interface{})
	assert.True(t, ok, "expected top-level 'data' envelope, got body=%s", w.Body.String())
	assert.Equal(t, id.String(), data["id"])
}

func TestAgentHandler_Update_DataEnvelope(t *testing.T) {
	r, m := newTestRouter(t)
	id := uuid.New()
	projectID := uuid.New()

	m.On("UpdateAgent", mock.Anything, id, projectID, mock.Anything).Return(&model.Agent{
		ID: id, ProjectID: projectID, Name: "alpha", Role: "qa",
		Status: model.AgentActive, Capabilities: []string{"testing"},
		Version: 2, CreatedAt: parseTime("2026-06-12T10:00:00Z"),
		UpdatedAt: parseTime("2026-06-12T11:00:00Z"),
	}, (*service.Error)(nil))

	body := map[string]interface{}{"role": "qa", "version": 1}
	w := doRequest(r, http.MethodPut, "/v1/agents/"+id.String(), projectID.String(), body)
	assert.Equal(t, http.StatusOK, w.Code)
	m.AssertExpectations(t)

	var resp map[string]interface{}
	requireUnmarshal(t, w, &resp)
	data, ok := resp["data"].(map[string]interface{})
	assert.True(t, ok, "expected top-level 'data' envelope, got body=%s", w.Body.String())
	assert.Equal(t, id.String(), data["id"])
	assert.Equal(t, "qa", data["role"])
}
// ---- helper utilities ------------------------------------------------

func requireUnmarshal(t *testing.T, w *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), v); err != nil {
		t.Fatalf("response body did not unmarshal: %v\nbody=%s", err, w.Body.String())
	}
}

// parseTime is a small helper; the tests use a stable timestamp
// rather than time.Now() so assertions are deterministic.
func parseTime(s string) time.Time {
	tt, _ := time.Parse(time.RFC3339, s)
	return tt
}
