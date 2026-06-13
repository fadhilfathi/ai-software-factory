package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DeliverableService is the consumer-side interface the
// DeliverableHandler depends on. Defined in the handler package
// (not the service package) so the handler can be tested with a
// hand-rolled mock without exporting an interface from the
// service implementation. The real *service.DeliverableService
// satisfies this interface structurally (Go's structural
// typing — both types have the same method set).
//
// Matches the AssignmentHandler.AssignmentService + ExecutionHandler.ExecutionService
// patterns approved by the Lead in TASK-404 and TASK-405.
type DeliverableService interface {
	CreateDeliverable(
		ctx context.Context,
		req service.CreateDeliverableRequest,
	) (*model.Deliverable, *service.Error)

	GetDeliverable(
		ctx context.Context,
		id uuid.UUID,
	) (*model.Deliverable, *service.Error)

	ListDeliverables(
		ctx context.Context,
		filter model.DeliverableFilter,
	) (*model.DeliverableListResult, *service.Error)

	UpdateDeliverable(
		ctx context.Context,
		id uuid.UUID,
		req service.UpdateDeliverableRequest,
	) (*model.Deliverable, *service.Error)

	ListDeliverableVersions(
		ctx context.Context,
		deliverableID uuid.UUID,
	) ([]*model.DeliverableVersion, *service.Error)
}

// maxDeliverableRequestBytes caps the entire request body for
// POST /v1/deliverables and PUT /v1/deliverables/:id. It is set
// with a small headroom (8 KiB) above MaxDeliverableContentBytes
// (1 MiB) so the JSON envelope (title, ids, version, etc.) has
// room without letting the envelope itself be abused as an attack
// vector. The service additionally re-checks the parsed content
// size against model.MaxDeliverableContentBytes — the handler
// cap is the first line of defence, the service cap is the
// second (defence-in-depth per F-023).
const maxDeliverableRequestBytes int64 = model.MaxDeliverableContentBytes + 8*1024

// DeliverableHandler is the Sprint 4 (TASK-406) HTTP layer
// for /v1/deliverables. All routes require auth (the router
// wraps them with the auth middleware).
type DeliverableHandler struct {
	svc DeliverableService
}

func NewDeliverableHandler(svc DeliverableService) *DeliverableHandler {
	return &DeliverableHandler{svc: svc}
}

// ----------------------------------------------------------------------------
// Request bodies
// ----------------------------------------------------------------------------

type createDeliverableRequest struct {
	TaskID  string `json:"task_id"`
	AgentID string `json:"agent_id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type updateDeliverableRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// deliverableResponse is the JSON shape for a Deliverable. The
// fields match the §6 schema (id, task_id, agent_id, title,
// content, version, created_at, updated_at). Title and content
// are markdown strings; the frontend (TASK-409) handles the
// rendering. updated_at was added in 022 (the Sprint 4 ALTER).
type deliverableResponse struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	AgentID   string `json:"agent_id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Version   int    `json:"version"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// deliverableVersionResponse is the JSON shape for a single
// DeliverableVersion row. CreatedBy is a *string (UUID) and
// omits when nil (system-driven version-creates).
type deliverableVersionResponse struct {
	ID            string  `json:"id"`
	DeliverableID string  `json:"deliverable_id"`
	Version       int     `json:"version"`
	Title         string  `json:"title"`
	Content       string  `json:"content"`
	CreatedAt     string  `json:"created_at"`
	CreatedBy     *string `json:"created_by,omitempty"`
}

// ----------------------------------------------------------------------------
// POST /v1/deliverables
// ----------------------------------------------------------------------------

func (h *DeliverableHandler) Create(c *gin.Context) {
	// F-023 DoS hardening: wrap the request body in
	// http.MaxBytesReader so an oversize body is rejected
	// before it is buffered into memory and before any
	// JSON parsing is attempted. ShouldBindJSON reads via
	// this wrapped body, so a trip returns a
	// *http.MaxBytesError (Go 1.20+) which we detect and
	// map to 413 PAYLOAD_TOO_LARGE. The service layer
	// re-checks the parsed content size as a second line
	// of defence.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxDeliverableRequestBytes)
	var req createDeliverableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if isMaxBytesError(err) {
			writeError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE",
				"Request body exceeds the maximum allowed size")
			return
		}
		writeError(c, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}
	taskID, err := uuid.Parse(req.TaskID)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid task_id")
		return
	}
	agentID, err := uuid.Parse(req.AgentID)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid agent_id")
		return
	}

	d, svcErr := h.svc.CreateDeliverable(c.Request.Context(), service.CreateDeliverableRequest{
		TaskID:    taskID,
		AgentID:   agentID,
		Title:     req.Title,
		Content:   req.Content,
		CreatedBy: userIDFromContext(c),
	})
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}
	writeJSON(c, http.StatusCreated, toDeliverableResponse(d))
}

// ----------------------------------------------------------------------------
// GET /v1/deliverables
// ----------------------------------------------------------------------------

func (h *DeliverableHandler) List(c *gin.Context) {
	filter := model.DeliverableFilter{}

	if raw := c.Query("task_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid task_id")
			return
		}
		filter.TaskID = id
	}
	if raw := c.Query("agent_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid agent_id")
			return
		}
		filter.AgentID = id
	}
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid limit")
			return
		}
		filter.Limit = n
	}
	if raw := c.Query("cursor"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid cursor")
			return
		}
		filter.Cursor = id
	}

	result, svcErr := h.svc.ListDeliverables(c.Request.Context(), filter)
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}
	writeJSON(c, http.StatusOK, result)
}

// ----------------------------------------------------------------------------
// GET /v1/deliverables/:id
// ----------------------------------------------------------------------------

func (h *DeliverableHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid Deliverable ID")
		return
	}
	d, svcErr := h.svc.GetDeliverable(c.Request.Context(), id)
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}
	writeJSON(c, http.StatusOK, toDeliverableResponse(d))
}

// ----------------------------------------------------------------------------
// PUT /v1/deliverables/:id  (append-only version-create)
// ----------------------------------------------------------------------------

func (h *DeliverableHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid Deliverable ID")
		return
	}
	// F-023 DoS hardening: see Create() — same wrap + same
	// 413 mapping.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxDeliverableRequestBytes)
	var req updateDeliverableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if isMaxBytesError(err) {
			writeError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE",
				"Request body exceeds the maximum allowed size")
			return
		}
		writeError(c, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	d, svcErr := h.svc.UpdateDeliverable(c.Request.Context(), id, service.UpdateDeliverableRequest{
		Title:     req.Title,
		Content:   req.Content,
		UpdatedBy: userIDFromContext(c),
	})
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}
	writeJSON(c, http.StatusOK, toDeliverableResponse(d))
}

// ----------------------------------------------------------------------------
// GET /v1/deliverables/:id/versions
// ----------------------------------------------------------------------------

// ListVersions returns the immutable history of a deliverable,
// ordered by version DESC. 404 if the deliverable itself
// doesn't exist. The response is a plain JSON array (not a
// paginated list) — version counts per deliverable are
// expected to be small in Sprint 4 (typically < 100).
func (h *DeliverableHandler) ListVersions(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid Deliverable ID")
		return
	}
	versions, svcErr := h.svc.ListDeliverableVersions(c.Request.Context(), id)
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}
	resp := make([]deliverableVersionResponse, len(versions))
	for i, v := range versions {
		resp[i] = toDeliverableVersionResponse(v)
	}
	writeJSON(c, http.StatusOK, resp)
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

// userIDFromContext extracts the user_id stashed by the auth
// middleware (post-TASK-418). Returns nil when the middleware
// did not set it (system-driven requests, e.g. background
// cron jobs). The user_id is a string in the middleware; we
// parse it as a UUID for the service.
func userIDFromContext(c *gin.Context) *uuid.UUID {
	raw, ok := c.Get("user_id")
	if !ok {
		return nil
	}
	s, ok := raw.(string)
	if !ok || s == "" {
		return nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil
	}
	return &id
}

func toDeliverableResponse(d *model.Deliverable) deliverableResponse {
	return deliverableResponse{
		ID:        d.ID.String(),
		TaskID:    d.TaskID.String(),
		AgentID:   d.AgentID.String(),
		Title:     d.Title,
		Content:   d.Content,
		Version:   d.Version,
		CreatedAt: d.CreatedAt.Format(time.RFC3339),
		UpdatedAt: d.UpdatedAt.Format(time.RFC3339),
	}
}

func toDeliverableVersionResponse(v *model.DeliverableVersion) deliverableVersionResponse {
	r := deliverableVersionResponse{
		ID:            v.ID.String(),
		DeliverableID: v.DeliverableID.String(),
		Version:       v.Version,
		Title:         v.Title,
		Content:       v.Content,
		CreatedAt:     v.CreatedAt.Format(time.RFC3339),
	}
	if v.CreatedBy != nil {
		s := v.CreatedBy.String()
		r.CreatedBy = &s
	}
	return r
}
