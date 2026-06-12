package handler

// HTTP-level tests for the CapabilityHandler (TASK-403, Sprint 4).
//
// Mirrors the structure of agent_test.go: real Gin engine, mock
// AgentService (testify). The CapabilityHandler depends on the
// AgentService interface (not the concrete type), and that interface
// is already fully implemented by the existing mockAgentService in
// agent_test.go. So we reuse it and just instantiate the new
// CapabilityHandler.
// Covers:
//   - GET /v1/agents/:id/capabilities  (api-spec.md §1.6)
//   - GET /v1/capabilities             (api-spec.md §2.1)
//
// The two routes were moved off AgentHandler in TASK-403 (see
// router.go). The new tests live here so the new file structure
// matches the wire layout.

import (
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

// newCapabilityTestRouter wires Gin with the CapabilityHandler
// routes and the shared mockAgentService. The X-Project-ID header
// is set by doCapabilityRequest so the catalog endpoint (which is
// project-scoped) can be exercised.
func newCapabilityTestRouter(t *testing.T) (*gin.Engine, *mockAgentService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("request_id", "test-rid-cap-001")
		c.Next()
	})
	m := &mockAgentService{}
	h := NewCapabilityHandler(m)
	v1 := r.Group("/v1")
	{
		v1.GET("/agents/:id/capabilities", h.ListAgentCapabilities)
		v1.GET("/capabilities", h.ListCatalogCapabilities)
	}
	return r, m
}

func doCapabilityRequest(r *gin.Engine, method, path, projectID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if projectID != "" {
		req.Header.Set("X-Project-ID", projectID)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ---- GET /v1/agents/:id/capabilities --------------------------------

func TestCapabilityHandler_ListAgentCapabilities_Success(t *testing.T) {
	r, m := newCapabilityTestRouter(t)
	agentID := uuid.New()
	projectID := uuid.New()

	now := time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)
	prof := 7
	grantedBy := uuid.New()
	m.On("ListAgentCapabilities", mock.Anything, agentID).Return(
		[]*model.AgentCapability{
			{Name: "coding", DisplayName: "Coding", Category: "coding",
				Proficiency: &prof, GrantedAt: now, GrantedBy: &grantedBy},
			{Name: "testing", DisplayName: "Testing", Category: "testing",
				GrantedAt: now},
		},
		(*service.Error)(nil))

	w := doCapabilityRequest(r, http.MethodGet, "/v1/agents/"+agentID.String()+"/capabilities", projectID.String())
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response body did not unmarshal: %v\nbody=%s", err, w.Body.String())
	}
	data, ok := resp["data"].([]interface{})
	if !ok {
		t.Fatalf("expected data array, got %#v", resp["data"])
	}
	assert.Equal(t, 2, len(data))
	first := data[0].(map[string]interface{})
	assert.Equal(t, "coding", first["name"])
	assert.Equal(t, "Coding", first["display_name"])
	assert.Equal(t, float64(7), first["proficiency"])
	assert.Equal(t, grantedBy.String(), first["granted_by"])
	// proficiency omitted on the second entry (nil)
	second := data[1].(map[string]interface{})
	_, hasProf := second["proficiency"]
	assert.False(t, hasProf, "proficiency must be omitted when nil")
	m.AssertExpectations(t)
}

func TestCapabilityHandler_ListAgentCapabilities_NotFound(t *testing.T) {
	r, m := newCapabilityTestRouter(t)
	agentID := uuid.New()
	projectID := uuid.New()

	m.On("ListAgentCapabilities", mock.Anything, agentID).Return(
		([]*model.AgentCapability)(nil),
		&service.Error{Status: 404, Code: "NOT_FOUND", Message: "Agent not found"})

	w := doCapabilityRequest(r, http.MethodGet, "/v1/agents/"+agentID.String()+"/capabilities", projectID.String())
	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response body did not unmarshal: %v\nbody=%s", err, w.Body.String())
	}
	errBlock := resp["error"].(map[string]interface{})
	assert.Equal(t, "NOT_FOUND", errBlock["code"])
	m.AssertExpectations(t)
}

func TestCapabilityHandler_ListAgentCapabilities_InvalidID(t *testing.T) {
	r, _ := newCapabilityTestRouter(t)
	w := doCapabilityRequest(r, http.MethodGet, "/v1/agents/not-a-uuid/capabilities", uuid.New().String())
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---- GET /v1/capabilities --------------------------------------------

func TestCapabilityHandler_ListCatalogCapabilities_Success(t *testing.T) {
	r, m := newCapabilityTestRouter(t)
	projectID := uuid.New()

	m.On("ListCapabilities", mock.Anything, mock.MatchedBy(func(req service.ListCapabilitiesRequest) bool {
		return req.Category == "coding" && req.Limit == 0
	})).Return(&service.ListCapabilitiesResult{
		Data: []model.CapabilityRow{
			{Name: "coding", DisplayName: "Coding", Category: "coding", Version: 1},
		},
		HasMore: false,
	}, (*service.Error)(nil))

	w := doCapabilityRequest(r, http.MethodGet, "/v1/capabilities?category=coding", projectID.String())
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response body did not unmarshal: %v\nbody=%s", err, w.Body.String())
	}
	assert.Contains(t, resp, "data")
	assert.Contains(t, resp, "pagination")
	pg := resp["pagination"].(map[string]interface{})
	assert.Equal(t, false, pg["has_more"])
	m.AssertExpectations(t)
}

func TestCapabilityHandler_ListCatalogCapabilities_WithLimit(t *testing.T) {
	r, m := newCapabilityTestRouter(t)
	projectID := uuid.New()

	m.On("ListCapabilities", mock.Anything, mock.MatchedBy(func(req service.ListCapabilitiesRequest) bool {
		return req.Category == "" && req.Limit == 10
	})).Return(&service.ListCapabilitiesResult{
		Data:    []model.CapabilityRow{},
		HasMore: false,
	}, (*service.Error)(nil))

	w := doCapabilityRequest(r, http.MethodGet, "/v1/capabilities?limit=10", projectID.String())
	assert.Equal(t, http.StatusOK, w.Code)
	m.AssertExpectations(t)
}
