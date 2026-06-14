package handler

// HTTP-level test for TaskHandler D7 envelope consistency
// (Sprint 6, TASK-427).
//
// Strategy: drive Gin with a real router, stub auth middleware,
// and a real service.TaskService backed by an in-memory store.
// One test only — assert POST /v1/projects/:id/tasks returns
// {data: {...}} envelope.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTaskTestRouter(t *testing.T, withUserID string) (*gin.Engine, uuid.UUID) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("request_id", "test-rid-task-001")
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
	svc := service.NewTaskService(s, zap.NewNop())
	h := NewTaskHandler(svc)
	v1 := r.Group("/v1")
	{
		v1.POST("/projects/:id/tasks", h.Create)
	}
	// Pre-create the project the test will use. The 404 cascade hit
	// TestTaskHandler_Create_DataEnvelope because the test was using
	// uuid.New() for projectID without ever creating the project row.
	userUUID, _ := uuid.Parse(withUserID)
	projectID := uuid.New()
	proj := &model.Project{
		ID:      projectID,
		Name:    "Test Project",
		OwnerID: userUUID,
		Status:  model.ProjectInProgress,
	}
	if err := s.Projects().Create(proj); err != nil {
		t.Fatalf("setup: pre-create test project: %v", err)
	}
	return r, projectID
}

func doTaskRequestAs(r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
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

// ---- D7 envelope consistency (Sprint 6, TASK-427) ------------------

func TestTaskHandler_Create_DataEnvelope(t *testing.T) {
	r, projectID := newTaskTestRouter(t, "test-user-001")

	body := map[string]any{
		"title":       "Envelope test task",
		"description": "verify {data: ...} envelope on POST /v1/projects/:id/tasks",
		"priority":    "normal",
	}
	w := doTaskRequestAs(r, http.MethodPost, "/v1/projects/"+projectID.String()+"/tasks", body)
	require.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data, ok := resp["data"].(map[string]any)
	assert.True(t, ok, "expected top-level 'data' envelope, got body=%s", w.Body.String())
	assert.Equal(t, "Envelope test task", data["title"])
	assert.Equal(t, projectID.String(), data["project_id"])
}
