package handler

// HTTP-level tests for DeliverableHandler (TASK-406, Sprint 4).
//
// Strategy: drive Gin with a real router, a stub auth middleware
// (so we can test the "no user_id" → 401 case), and a real
// service.DeliverableService backed by an in-memory store.
// Covers the 5 routes:
//   - POST   /v1/deliverables
//   - GET    /v1/deliverables
//   - GET    /v1/deliverables/:id
//   - PUT    /v1/deliverables/:id
//   - GET    /v1/deliverables/:id/versions
// and the auth middleware (401).

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

// newDeliverableTestRouter wires a Gin engine with the 5
// deliverable routes and a stub auth middleware. When
// withUserID is empty, the middleware short-circuits with
// 401. When set, the middleware stashes the user_id for
// downstream handlers (the deliverable handler reads it as
// the version created_by).
func newDeliverableTestRouter(t *testing.T, withUserID string) (*gin.Engine, *service.DeliverableService, store.Store) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("request_id", "test-rid-deliv-001")
		if withUserID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "authentication required"},
			})
			return
		}
		c.Set("user_id", withUserID)
		c.Next()
	})

	s := store.NewMemoryStore()
	svc := service.NewDeliverableService(s, zap.NewNop())
	h := NewDeliverableHandler(svc)
	v1 := r.Group("/v1")
	{
		v1.POST("/deliverables", h.Create)
		v1.GET("/deliverables", h.List)
		v1.GET("/deliverables/:id", h.Get)
		v1.PUT("/deliverables/:id", h.Update)
		v1.GET("/deliverables/:id/versions", h.ListVersions)
	}
	return r, svc, s
}

// seedDelivTaskAndAgent is the in-memory-store equivalent of
// the helper used in service tests, but lives in the handler
// package because the handler doesn't import the service test
// internals.
func seedDelivTaskAndAgent(t *testing.T, s store.Store) (uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	task := &model.Task{
		ID:        uuid.New(),
		ProjectID: uuid.New(),
		Title:     "deliv-handler-test-" + uuid.NewString()[:8],
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
	return task.ID, created.ID, task.ProjectID
}

// doDelivRequestAs sets the X-Project-ID header to the supplied projectID.
// Pass uuid.Nil to skip the header (used by tests that exercise a different
// failure mode, e.g. MissingAuth_401 or the new MissingProjectHeader tests).
func doDelivRequestAs(r *gin.Engine, method, path string, body any, projectID uuid.UUID) *httptest.ResponseRecorder {
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

// doDelivRequest is the no-header variant. Kept for the MissingAuth_401 test,
// which short-circuits before the deliverable handler reads the header.
func doDelivRequest(r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	return doDelivRequestAs(r, method, path, body, uuid.Nil)
}

// ----------------------------------------------------------------------------
// POST /v1/deliverables
// ----------------------------------------------------------------------------

func TestDeliverableHandler_Create_201(t *testing.T) {
	r, _, s := newDeliverableTestRouter(t, uuid.NewString())
	taskID, agentID, projectID := seedDelivTaskAndAgent(t, s)

	body := map[string]any{
		"task_id": taskID.String(), "agent_id": agentID.String(),
		"title": "First deliverable", "content": "# Hello",
	}
	w := doDelivRequestAs(r, http.MethodPost, "/v1/deliverables", body, projectID)
	require.Equal(t, http.StatusCreated, w.Code)

	var resp struct {
		Data deliverableResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, uuid.Nil, resp.Data.ID)
	assert.Equal(t, "First deliverable", resp.Data.Title)
	assert.Equal(t, 1, resp.Data.Version)
	assert.Equal(t, taskID.String(), resp.Data.TaskID)
	assert.Equal(t, agentID.String(), resp.Data.AgentID)
}

func TestDeliverableHandler_Create_400_BadUUID(t *testing.T) {
	r, _, _ := newDeliverableTestRouter(t, uuid.NewString())

	body := map[string]any{
		"task_id": "not-a-uuid", "agent_id": uuid.NewString(),
		"title": "x", "content": "y",
	}
	// BadUUID happens before the X-Project-ID check, so we still need a valid project header.
	projectID := uuid.New()
	w := doDelivRequestAs(r, http.MethodPost, "/v1/deliverables", body, projectID)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	body = map[string]any{
		"task_id": uuid.NewString(), "agent_id": "still-not-a-uuid",
		"title": "x", "content": "y",
	}
	w = doDelivRequestAs(r, http.MethodPost, "/v1/deliverables", body, projectID)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeliverableHandler_Create_404(t *testing.T) {
	r, _, s := newDeliverableTestRouter(t, uuid.NewString())
	_, agentID, projectID := seedDelivTaskAndAgent(t, s)

	// Task does not exist.
	body := map[string]any{
		"task_id": uuid.NewString(), "agent_id": agentID.String(),
		"title": "x", "content": "y",
	}
	w := doDelivRequestAs(r, http.MethodPost, "/v1/deliverables", body, projectID)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ----------------------------------------------------------------------------
// GET /v1/deliverables/:id
// ----------------------------------------------------------------------------

func TestDeliverableHandler_Get_200_And_404(t *testing.T) {
	r, svc, s := newDeliverableTestRouter(t, uuid.NewString())
	taskID, agentID, projectID := seedDelivTaskAndAgent(t, s)

	d, svcErr := svc.CreateDeliverable(context.Background(), service.CreateDeliverableRequest{
		TaskID: taskID, AgentID: agentID, Title: "x", Content: "y",
	}, projectID)
	require.Nil(t, svcErr)

	// 200
	w := doDelivRequestAs(r, http.MethodGet, "/v1/deliverables/"+d.ID.String(), nil, projectID)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data deliverableResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, d.ID.String(), resp.Data.ID)

	// 404
	w = doDelivRequestAs(r, http.MethodGet, "/v1/deliverables/"+uuid.NewString(), nil, projectID)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ----------------------------------------------------------------------------
// GET /v1/deliverables (list)
// ----------------------------------------------------------------------------

func TestDeliverableHandler_List_WithFilters(t *testing.T) {
	r, svc, s := newDeliverableTestRouter(t, uuid.NewString())
	taskA, agentA, projectID := seedDelivTaskAndAgent(t, s)
	taskB, _, _ := seedDelivTaskAndAgent(t, s)

	for i := 0; i < 3; i++ {
		_, svcErr := svc.CreateDeliverable(context.Background(), service.CreateDeliverableRequest{
			TaskID: taskA, AgentID: agentA, Title: "a-" + uuid.NewString()[:6], Content: "x",
		}, projectID)
		require.Nil(t, svcErr)
	}
	_, svcErr := svc.CreateDeliverable(context.Background(), service.CreateDeliverableRequest{
		TaskID: taskB, AgentID: agentA, Title: "b", Content: "x",
	}, projectID)
	require.Nil(t, svcErr)

	// task_id filter → 3
	w := doDelivRequestAs(r, http.MethodGet, "/v1/deliverables?task_id="+taskA.String(), nil, projectID)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data model.DeliverableListResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Data.Items, 3)

	// agent_id filter → 4
	w = doDelivRequestAs(r, http.MethodGet, "/v1/deliverables?agent_id="+agentA.String(), nil, projectID)
	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Data.Items, 4)

	// No filter → 400 (service requires at least one of
	// task_id/agent_id).
	w = doDelivRequestAs(r, http.MethodGet, "/v1/deliverables", nil, projectID)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ----------------------------------------------------------------------------
// PUT /v1/deliverables/:id
// ----------------------------------------------------------------------------

func TestDeliverableHandler_PUT_200_v1ToV2(t *testing.T) {
	r, svc, s := newDeliverableTestRouter(t, uuid.NewString())
	taskID, agentID, projectID := seedDelivTaskAndAgent(t, s)
	d, svcErr := svc.CreateDeliverable(context.Background(), service.CreateDeliverableRequest{
		TaskID: taskID, AgentID: agentID, Title: "v1", Content: "v1 body",
	}, projectID)
	require.Nil(t, svcErr)

	body := map[string]any{"title": "v2", "content": "v2 body"}
	w := doDelivRequestAs(r, http.MethodPut, "/v1/deliverables/"+d.ID.String(), body, projectID)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data deliverableResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp.Data.Version)
	assert.Equal(t, "v2", resp.Data.Title)
	assert.Equal(t, "v2 body", resp.Data.Content)
}

func TestDeliverableHandler_PUT_404(t *testing.T) {
	r, _, _ := newDeliverableTestRouter(t, uuid.NewString())
	body := map[string]any{"title": "x", "content": "y"}
	projectID := uuid.New()
	w := doDelivRequestAs(r, http.MethodPut, "/v1/deliverables/"+uuid.NewString(), body, projectID)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ----------------------------------------------------------------------------
// GET /v1/deliverables/:id/versions
// ----------------------------------------------------------------------------

func TestDeliverableHandler_ListVersions_200(t *testing.T) {
	r, svc, s := newDeliverableTestRouter(t, uuid.NewString())
	taskID, agentID, projectID := seedDelivTaskAndAgent(t, s)
	d, svcErr := svc.CreateDeliverable(context.Background(), service.CreateDeliverableRequest{
		TaskID: taskID, AgentID: agentID, Title: "v1", Content: "v1",
	}, projectID)
	require.Nil(t, svcErr)
	_, svcErr = svc.UpdateDeliverable(context.Background(), d.ID, service.UpdateDeliverableRequest{
		Title: "v2", Content: "v2",
	}, projectID)
	require.Nil(t, svcErr)

	w := doDelivRequestAs(r, http.MethodGet, "/v1/deliverables/"+d.ID.String()+"/versions", nil, projectID)
	require.Equal(t, http.StatusOK, w.Code)

	var versions []deliverableVersionResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &versions))
	require.Len(t, versions, 2)
	// DESC ordering: v2 first, v1 second.
	assert.Equal(t, 2, versions[0].Version)
	assert.Equal(t, 1, versions[1].Version)

	// 404 when the deliverable itself doesn't exist.
	w = doDelivRequestAs(r, http.MethodGet, "/v1/deliverables/"+uuid.NewString()+"/versions", nil, projectID)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ----------------------------------------------------------------------------
// Auth
// ----------------------------------------------------------------------------

func TestDeliverableHandler_MissingAuth_401(t *testing.T) {
	r, _, _ := newDeliverableTestRouter(t, "")
	w := doDelivRequest(r, http.MethodGet, "/v1/deliverables", nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ----------------------------------------------------------------------------
// F-023: DoS hardening — oversize request body returns 413
// ----------------------------------------------------------------------------

// doDelivRequestRaw is the raw-body variant of doDelivRequest.
// F-023 needs to push a 10 MiB body, and JSON-encoding a 10
// MiB map[string]any first is wasteful. This helper writes the
// raw bytes directly to the request, sets Content-Length
// explicitly, and skips JSON encoding.
// doDelivRequestRaw sets the X-Project-ID header to the supplied projectID.
// Pass uuid.Nil to skip the header.
func doDelivRequestRaw(r *gin.Engine, method, path string, contentType string, body []byte, projectID uuid.UUID) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	req.ContentLength = int64(len(body))
	if projectID != uuid.Nil {
		req.Header.Set("X-Project-ID", projectID.String())
	}
	r.ServeHTTP(w, req)
	return w
}

// TestDeliverableHandler_Create_OversizedRequest_413 is the
// F-023 handler-layer cap. The request body exceeds
// maxDeliverableRequestBytes (model.MaxDeliverableContentBytes
// + 8 KiB headroom), so http.MaxBytesReader trips and the
// handler returns 413 PAYLOAD_TOO_LARGE. The body is shaped
// as JSON so the response envelope shape is testable. The
// service is never reached — that is the point of the
// handler-layer cap.
func TestDeliverableHandler_Create_OversizedRequest_413(t *testing.T) {
	r, _, _ := newDeliverableTestRouter(t, uuid.NewString())

	// 10 MiB body — well over maxDeliverableRequestBytes
	// (1 MiB + 8 KiB). The content field is the only thing
	// that needs to be huge; the rest of the envelope is
	// small.
	oversize := bytes.Repeat([]byte("A"), 10*1024*1024)
	projectID := uuid.New()
	payload := []byte(`{"task_id":"` + uuid.NewString() + `","agent_id":"` + uuid.NewString() + `","title":"oversize","content":"`)
	payload = append(payload, oversize...)
	payload = append(payload, []byte(`"}`)...)

	w := doDelivRequestRaw(r, http.MethodPost, "/v1/deliverables",
		"application/json", payload, projectID)
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code,
		"oversize body must return 413, got %d (body=%s)", w.Code, truncate(w.Body.Bytes(), 256))

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	errObj := resp["error"].(map[string]interface{})
	assert.Equal(t, "PAYLOAD_TOO_LARGE", errObj["code"])
}

// TestDeliverableHandler_Update_OversizedRequest_413 covers
// the same trip on the PUT path. Same envelope, same cap.
// As with Create, the service is never reached — the 413
// happens in the handler before any service call.
func TestDeliverableHandler_Update_OversizedRequest_413(t *testing.T) {
	r, _, _ := newDeliverableTestRouter(t, uuid.NewString())
	id := uuid.New()
	projectID := uuid.New()
	oversize := bytes.Repeat([]byte("A"), 10*1024*1024)
	payload := []byte(`{"title":"v2-oversize","content":"`)
	payload = append(payload, oversize...)
	payload = append(payload, []byte(`"}`)...)

	w := doDelivRequestRaw(r, http.MethodPut, "/v1/deliverables/"+id.String(),
		"application/json", payload, projectID)
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code,
		"oversize PUT body must return 413, got %d (body=%s)", w.Code, truncate(w.Body.Bytes(), 256))

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	errObj := resp["error"].(map[string]interface{})
	assert.Equal(t, "PAYLOAD_TOO_LARGE", errObj["code"])
}

// TestDeliverableHandler_Create_AtTheCapBody_Succeeds pins
// the boundary at the handler layer: a body exactly at
// maxDeliverableRequestBytes is accepted, even though the
// content field is at the service cap. The 8 KiB envelope
// headroom is enough for the JSON shape used here.
func TestDeliverableHandler_Create_AtTheCapBody_Succeeds(t *testing.T) {
	r, _, s := newDeliverableTestRouter(t, uuid.NewString())
	taskID, agentID, projectID := seedDelivTaskAndAgent(t, s)

	// Build a JSON body whose `content` is exactly 1 MiB.
	content := strings.Repeat("B", int(model.MaxDeliverableContentBytes))
	body := map[string]interface{}{
		"task_id":  taskID.String(),
		"agent_id": agentID.String(),
		"title":    "at-cap",
		"content":  content,
	}
	w := doDelivRequestAs(r, http.MethodPost, "/v1/deliverables", body, projectID)
	assert.Equal(t, http.StatusCreated, w.Code,
		"at-cap body must succeed, got %d (body=%s)", w.Code, truncate(w.Body.Bytes(), 256))
}

// truncate returns the first n bytes of b for log output, so
// giant error bodies don't blow up the test log.
func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "...(truncated)"
}


// =============================================================================
// TASK-421 (F-015) cross-tenant + missing-header handler tests
// =============================================================================
//
// Each test exercises one of the 5 deliverable routes under
// (a) cross-tenant access (different X-Project-ID) and
// (b) missing X-Project-ID header.
//
// Expected responses (per docs/sprint4/security-review.md §5.1.1):
//   - cross-tenant       → 404 + code CROSS_TENANT_BLOCKED
//   - missing header     → 400 + code MISSING_PROJECT_HEADER

// Cross-tenant: CreateDeliverable, GetDeliverable, UpdateDeliverable,
// ListDeliverableVersions (all return 404).
// ListDeliverables is in the same-project control test (returns 200)
// plus a cross-tenant test (returns 404).

func TestDeliverableHandler_Create_CrossTenant_Returns404(t *testing.T) {
	r, _, s := newDeliverableTestRouter(t, uuid.NewString())
	taskID, agentID, projectID := seedDelivTaskAndAgent(t, s)

	// Caller in a different project asks to create against a task they don't own.
	otherProjectID := uuid.New()
	body := map[string]any{
		"task_id":  taskID.String(),
		"agent_id": agentID.String(),
		"title":    "cross-tenant",
		"content":  "should not stick",
	}
	w := doDelivRequestAs(r, http.MethodPost, "/v1/deliverables", body, otherProjectID)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assertCode(t, w.Body.Bytes(), "CROSS_TENANT_BLOCKED")
}

func TestDeliverableHandler_Create_MissingProjectHeader_Returns400(t *testing.T) {
	r, _, s := newDeliverableTestRouter(t, uuid.NewString())
	taskID, agentID, _ := seedDelivTaskAndAgent(t, s)

	body := map[string]any{
		"task_id":  taskID.String(),
		"agent_id": agentID.String(),
		"title":    "missing-header",
		"content":  "x",
	}
	// uuid.Nil → no X-Project-ID header.
	w := doDelivRequestAs(r, http.MethodPost, "/v1/deliverables", body, uuid.Nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assertCode(t, w.Body.Bytes(), "MISSING_PROJECT_HEADER")
}

func TestDeliverableHandler_Get_CrossTenant_Returns404(t *testing.T) {
	r, svc, s := newDeliverableTestRouter(t, uuid.NewString())
	taskID, agentID, projectID := seedDelivTaskAndAgent(t, s)

	d, svcErr := svc.CreateDeliverable(context.Background(), service.CreateDeliverableRequest{
		TaskID:  taskID,
		AgentID: agentID,
		Title:   "get-cross-tenant",
		Content: "y",
	}, projectID)
	require.Nil(t, svcErr)

	otherProjectID := uuid.New()
	w := doDelivRequestAs(r, http.MethodGet, "/v1/deliverables/"+d.ID.String(), nil, otherProjectID)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assertCode(t, w.Body.Bytes(), "CROSS_TENANT_BLOCKED")
}

func TestDeliverableHandler_Get_MissingProjectHeader_Returns400(t *testing.T) {
	r, svc, s := newDeliverableTestRouter(t, uuid.NewString())
	taskID, agentID, projectID := seedDelivTaskAndAgent(t, s)

	d, svcErr := svc.CreateDeliverable(context.Background(), service.CreateDeliverableRequest{
		TaskID:  taskID,
		AgentID: agentID,
		Title:   "get-missing-header",
		Content: "y",
	}, projectID)
	require.Nil(t, svcErr)

	w := doDelivRequestAs(r, http.MethodGet, "/v1/deliverables/"+d.ID.String(), nil, uuid.Nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assertCode(t, w.Body.Bytes(), "MISSING_PROJECT_HEADER")
}

func TestDeliverableHandler_List_CrossTenant_TaskFilter_Returns404(t *testing.T) {
	r, _, s := newDeliverableTestRouter(t, uuid.NewString())
	taskID, _, _ := seedDelivTaskAndAgent(t, s)

	otherProjectID := uuid.New()
	w := doDelivRequestAs(r, http.MethodGet, "/v1/deliverables?task_id="+taskID.String(), nil, otherProjectID)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assertCode(t, w.Body.Bytes(), "CROSS_TENANT_BLOCKED")
}

func TestDeliverableHandler_List_CrossTenant_AgentFilter_Returns404(t *testing.T) {
	r, _, s := newDeliverableTestRouter(t, uuid.NewString())
	_, agentID, _ := seedDelivTaskAndAgent(t, s)

	otherProjectID := uuid.New()
	w := doDelivRequestAs(r, http.MethodGet, "/v1/deliverables?agent_id="+agentID.String(), nil, otherProjectID)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assertCode(t, w.Body.Bytes(), "CROSS_TENANT_BLOCKED")
}

func TestDeliverableHandler_List_MissingProjectHeader_Returns400(t *testing.T) {
	r, _, s := newDeliverableTestRouter(t, uuid.NewString())
	taskID, _, _ := seedDelivTaskAndAgent(t, s)

	w := doDelivRequestAs(r, http.MethodGet, "/v1/deliverables?task_id="+taskID.String(), nil, uuid.Nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assertCode(t, w.Body.Bytes(), "MISSING_PROJECT_HEADER")
}

func TestDeliverableHandler_Update_CrossTenant_Returns404(t *testing.T) {
	r, svc, s := newDeliverableTestRouter(t, uuid.NewString())
	taskID, agentID, projectID := seedDelivTaskAndAgent(t, s)

	d, svcErr := svc.CreateDeliverable(context.Background(), service.CreateDeliverableRequest{
		TaskID:  taskID,
		AgentID: agentID,
		Title:   "update-cross-tenant",
		Content: "y",
	}, projectID)
	require.Nil(t, svcErr)

	otherProjectID := uuid.New()
	body := map[string]any{"title": "should-not-stick", "content": "z"}
	w := doDelivRequestAs(r, http.MethodPut, "/v1/deliverables/"+d.ID.String(), body, otherProjectID)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assertCode(t, w.Body.Bytes(), "CROSS_TENANT_BLOCKED")
}

func TestDeliverableHandler_Update_MissingProjectHeader_Returns400(t *testing.T) {
	r, svc, s := newDeliverableTestRouter(t, uuid.NewString())
	taskID, agentID, projectID := seedDelivTaskAndAgent(t, s)

	d, svcErr := svc.CreateDeliverable(context.Background(), service.CreateDeliverableRequest{
		TaskID:  taskID,
		AgentID: agentID,
		Title:   "update-missing-header",
		Content: "y",
	}, projectID)
	require.Nil(t, svcErr)

	body := map[string]any{"title": "v2", "content": "z"}
	w := doDelivRequestAs(r, http.MethodPut, "/v1/deliverables/"+d.ID.String(), body, uuid.Nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assertCode(t, w.Body.Bytes(), "MISSING_PROJECT_HEADER")
}

func TestDeliverableHandler_ListVersions_CrossTenant_Returns404(t *testing.T) {
	r, svc, s := newDeliverableTestRouter(t, uuid.NewString())
	taskID, agentID, projectID := seedDelivTaskAndAgent(t, s)

	d, svcErr := svc.CreateDeliverable(context.Background(), service.CreateDeliverableRequest{
		TaskID:  taskID,
		AgentID: agentID,
		Title:   "listversions-cross-tenant",
		Content: "y",
	}, projectID)
	require.Nil(t, svcErr)

	otherProjectID := uuid.New()
	w := doDelivRequestAs(r, http.MethodGet, "/v1/deliverables/"+d.ID.String()+"/versions", nil, otherProjectID)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assertCode(t, w.Body.Bytes(), "CROSS_TENANT_BLOCKED")
}

func TestDeliverableHandler_ListVersions_MissingProjectHeader_Returns400(t *testing.T) {
	r, svc, s := newDeliverableTestRouter(t, uuid.NewString())
	taskID, agentID, projectID := seedDelivTaskAndAgent(t, s)

	d, svcErr := svc.CreateDeliverable(context.Background(), service.CreateDeliverableRequest{
		TaskID:  taskID,
		AgentID: agentID,
		Title:   "listversions-missing-header",
		Content: "y",
	}, projectID)
	require.Nil(t, svcErr)

	w := doDelivRequestAs(r, http.MethodGet, "/v1/deliverables/"+d.ID.String()+"/versions", nil, uuid.Nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assertCode(t, w.Body.Bytes(), "MISSING_PROJECT_HEADER")
}

// assertCode parses an error envelope and asserts the code field.
func assertCode(t *testing.T, body []byte, want string) {
	t.Helper()
	var resp map[string]any
	require.NoError(t, json.Unmarshal(body, &resp))
	errObj, ok := resp["error"].(map[string]any)
	require.True(t, ok, "expected error object in body, got: %s", string(body))
	assert.Equal(t, want, errObj["code"])
}
