package handler

// Agent HTTP handlers (TASK-402, Sprint 4).
//
// Wire-up: docs/sprint4/api-spec.md §1 (1.1-1.6) and §2.1.
//
// All handlers:
//   - Read project_id from the URL path or the auth context.
//   - Wire query parameters with api-spec defaults applied by the
//     service layer (cursor pagination: default 50, max 200; default
//     list excludes retired; pass ?include_retired=true to include).
//   - Map service errors to the api-spec.md §0.4 error envelope
//     `{"error": {"code", "message", "details"}, "request_id"}`.

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AgentHandler is the Gin-bound agent HTTP handler. It depends on the
// AgentService interface, not the concrete impl, so it can be tested
// with a mock service.
type AgentHandler struct {
	svc service.AgentService
}

// NewAgentHandler wires the handler. The svc parameter is the
// interface (service.AgentService), not the pointer — the existing
// router registers it as a pointer for symmetry with other handlers.
func NewAgentHandler(svc service.AgentService) *AgentHandler {
	return &AgentHandler{svc: svc}
}

// --- Request / response shapes (the JSON wire format) ------------------

// createAgentRequest is the body of POST /v1/agents. ProjectID is
// pulled from the URL or auth context, not the body, so the same
// shape works for all 5 endpoints.
type createAgentRequest struct {
	Name         string          `json:"name"`
	Role         string          `json:"role"`
	Capabilities []string        `json:"capabilities"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
}

// updateAgentRequest is the body of PUT /v1/agents/:id. Pointer
// fields encode "absent" vs "present" so partial updates work
// cleanly.
type updateAgentRequest struct {
	Role         *string             `json:"role,omitempty"`
	Status       *model.AgentStatus  `json:"status,omitempty"`
	Capabilities *[]string           `json:"capabilities,omitempty"`
	Metadata     json.RawMessage     `json:"metadata,omitempty"`
	Version      *int                `json:"version"`
}

// agentResponse is the canonical wire format for a single agent
// (api-spec.md §1.1 + §1.3). Every field is exposed; the service
// returns a model.Agent and we map to this struct so internal
// renames do not leak into the API.
type agentResponse struct {
	ID           string          `json:"id"`
	ProjectID    string          `json:"project_id"`
	Name         string          `json:"name"`
	Role         string          `json:"role"`
	Status       string          `json:"status"`
	Capabilities []string        `json:"capabilities"`
	LastActiveAt *string         `json:"last_active_at,omitempty"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
	Version      int             `json:"version"`
	RetiredAt    *string         `json:"retired_at,omitempty"`
	CreatedAt    string          `json:"created_at"`
	UpdatedAt    string          `json:"updated_at"`
}

func toAgentResponse(a *model.Agent) agentResponse {
	r := agentResponse{
		ID:           a.ID.String(),
		ProjectID:    a.ProjectID.String(),
		Name:         a.Name,
		Role:         a.Role,
		Status:       string(a.Status),
		Capabilities: a.Capabilities,
		Version:      a.Version,
		CreatedAt:    a.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
		UpdatedAt:    a.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
	}
	if a.LastActiveAt != nil {
		s := a.LastActiveAt.UTC().Format("2006-01-02T15:04:05.000Z07:00")
		r.LastActiveAt = &s
	}
	if a.RetiredAt != nil {
		s := a.RetiredAt.UTC().Format("2006-01-02T15:04:05.000Z07:00")
		r.RetiredAt = &s
	}
	if len(a.Metadata) > 0 {
		r.Metadata = a.Metadata
	}
	return r
}

// --- POST /v1/agents ---------------------------------------------------

func (h *AgentHandler) Create(c *gin.Context) {
	var req createAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, &service.Error{
			Status:  400,
			Code:    "VALIDATION_ERROR",
			Message: "Invalid JSON body: " + err.Error(),
		})
		return
	}
	projectID, ok := projectIDFromContext(c)
	if !ok {
		respondError(c, &service.Error{
			Status:  400,
			Code:    "VALIDATION_ERROR",
			Message: "project_id is required",
		})
		return
	}

	agent, apiErr := h.svc.CreateAgent(c.Request.Context(), service.CreateAgentRequest{
		ProjectID:    projectID,
		Name:         strings.TrimSpace(req.Name),
		Role:         strings.TrimSpace(req.Role),
		Capabilities: req.Capabilities,
		Metadata:     req.Metadata,
	})
	if apiErr != nil {
		respondError(c, apiErr)
		return
	}
	c.JSON(http.StatusCreated, toAgentResponse(agent))
}

// --- GET /v1/agents ----------------------------------------------------

// List returns a cursor-paginated page. Query parameters:
//   - status: exact-match on the lifecycle state
//   - capability: filter to agents declaring this capability
//   - include_retired: "true" to include retired agents (default false)
//   - cursor: opaque cursor from a previous page
//   - limit: 1-200, default 50
func (h *AgentHandler) List(c *gin.Context) {
	projectID, ok := projectIDFromContext(c)
	if !ok {
		respondError(c, &service.Error{
			Status:  400,
			Code:    "VALIDATION_ERROR",
			Message: "project_id is required",
		})
		return
	}

	limit, _ := strconv.Atoi(c.Query("limit"))
	includeRetired := strings.EqualFold(c.Query("include_retired"), "true")

	res, apiErr := h.svc.ListAgents(c.Request.Context(), service.ListAgentsRequest{
		ProjectID:      projectID,
		Status:         c.Query("status"),
		Capability:     c.Query("capability"),
		IncludeRetired: includeRetired,
		Cursor:         c.Query("cursor"),
		Limit:          limit,
	})
	if apiErr != nil {
		respondError(c, apiErr)
		return
	}

	data := make([]agentResponse, 0, len(res.Data))
	for _, a := range res.Data {
		data = append(data, toAgentResponse(a))
	}

	body := gin.H{
		"data": data,
		"pagination": gin.H{
			"has_more":    res.HasMore,
			"next_cursor": res.NextCursor,
		},
	}
	c.JSON(http.StatusOK, body)
}

// --- GET /v1/agents/:id -----------------------------------------------

func (h *AgentHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, &service.Error{
			Status:  400, Code: "VALIDATION_ERROR", Message: "Invalid agent id",
		})
		return
	}
	callerProjectID, ok := projectIDFromContext(c)
	if !ok || callerProjectID == uuid.Nil {
		respondError(c, &service.Error{
			Status:  400, Code: "MISSING_PROJECT_HEADER", Message: "X-Project-ID header is required for this request",
		})
		return
	}
	agent, apiErr := h.svc.GetAgent(c.Request.Context(), id, callerProjectID)
	if apiErr != nil {
		respondError(c, apiErr)
		return
	}
	c.JSON(http.StatusOK, toAgentResponse(agent))
}

// --- PUT /v1/agents/:id -----------------------------------------------

func (h *AgentHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, &service.Error{
			Status:  400, Code: "VALIDATION_ERROR", Message: "Invalid agent id",
		})
		return
	}
	callerProjectID, ok := projectIDFromContext(c)
	if !ok || callerProjectID == uuid.Nil {
		respondError(c, &service.Error{
			Status:  400, Code: "MISSING_PROJECT_HEADER", Message: "X-Project-ID header is required for this request",
		})
		return
	}
	var req updateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, &service.Error{
			Status:  400, Code: "VALIDATION_ERROR", Message: "Invalid JSON body: " + err.Error(),
		})
		return
	}
	agent, apiErr := h.svc.UpdateAgent(c.Request.Context(), id, callerProjectID, service.UpdateAgentRequest{
		Role:         req.Role,
		Status:       req.Status,
		Capabilities: req.Capabilities,
		Metadata:     req.Metadata,
		Version:      req.Version,
	})
	if apiErr != nil {
		respondError(c, apiErr)
		return
	}
	c.JSON(http.StatusOK, toAgentResponse(agent))
}

// --- DELETE /v1/agents/:id --------------------------------------------

// Delete is the soft-delete path (api-spec.md §1.5). The
// ?force=true query parameter is accepted but currently behaves
// identically to the soft path; future schema-level hard delete can
// be added without breaking the wire format.
func (h *AgentHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, &service.Error{
			Status:  400, Code: "VALIDATION_ERROR", Message: "Invalid agent id",
		})
		return
	}
	callerProjectID, ok := projectIDFromContext(c)
	if !ok || callerProjectID == uuid.Nil {
		respondError(c, &service.Error{
			Status:  400, Code: "MISSING_PROJECT_HEADER", Message: "X-Project-ID header is required for this request",
		})
		return
	}
	force := strings.EqualFold(c.Query("force"), "true")
	apiErr := h.svc.RetireAgent(c.Request.Context(), id, callerProjectID, force)
	if apiErr != nil {
		respondError(c, apiErr)
		return
	}
	c.Status(http.StatusNoContent)
}

// --- helpers ----------------------------------------------------------

// projectIDFromContext extracts the project_id from the Gin context.
// Priority order:
//   1. Header "X-Project-ID" — the recommended cross-cutting source
//      for multi-tenant agent operations.
//   2. URL parameter "project_id" — for any future route nested under
//      /v1/projects/:project_id/agents/....
//
// The "ok" return is false when no source is available; the handler
// responds 400 with a clear message in that case.
func projectIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	if s := c.GetHeader("X-Project-ID"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			return id, true
		}
	}
	if s := c.Param("project_id"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			return id, true
		}
	}
	return uuid.Nil, false
}

// respondError is the canonical error envelope writer. It uses the
// api-spec.md §0.4 shape `{"error": {"code", "message", "details"},
// "request_id"}` and pulls the request_id from the Gin context
// (set by the request-id middleware in cmd/main.go).
func respondError(c *gin.Context, e *service.Error) {
	body := gin.H{
		"error": gin.H{
			"code":    e.Code,
			"message": e.Message,
		},
	}
	if len(e.Details) > 0 {
		body["error"].(gin.H)["details"] = e.Details
	}
	if v, exists := c.Get("request_id"); exists {
		body["request_id"] = v
	}
	c.JSON(e.Status, body)
}
