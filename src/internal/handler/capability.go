package handler

// Capability HTTP handlers (TASK-403, Sprint 4).
//
// These were previously hosted on AgentHandler; the TASK-403 brief
// moves them into a dedicated file so the AgentHandler stays focused
// on the agent CRUD endpoints (api-spec.md §1.1-1.5). The wire-level
// URL paths and request/response shapes are unchanged.
//
// Wire-up:
//   - GET /v1/agents/:id/capabilities  ->  api-spec.md §1.6
//   - GET /v1/capabilities             ->  api-spec.md §2.1
//
// Cursor pagination contract (api-spec.md §0.3):
//   - default limit = 50
//   - max limit     = 200
//   - opaque cursor base64 in the response pagination block

import (
	"net/http"
	"strconv"

	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CapabilityHandler is the Gin-bound capability HTTP handler. It
// depends on the AgentService interface for the agent capability
// listing and the CapabilityService for the catalog read. The catalog
// read method is on AgentService.ListCapabilities in this codebase
// (set up in TASK-402) — the CapabilityService interface itself is
// used by AssignmentService as the validation seam and does not
// expose any HTTP-facing method.
type CapabilityHandler struct {
	agents service.AgentService
}

// NewCapabilityHandler wires the handler with the agent service.
// CapabilityService is reachable via the AssignmentService validation
// seam (TASK-403) and does not need its own handler binding.
func NewCapabilityHandler(agents service.AgentService) *CapabilityHandler {
	return &CapabilityHandler{agents: agents}
}

// --- GET /v1/agents/:id/capabilities ----------------------------------

// ListAgentCapabilities is the per-agent read (api-spec.md §1.6).
// Returns the granted capabilities with proficiency and granted_at.
// Returns 404 if the agent is unknown (the service maps
// store.ErrNotFound to a notFound *Error).
func (h *CapabilityHandler) ListAgentCapabilities(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, &service.Error{
			Status:  http.StatusBadRequest, Code: "VALIDATION_ERROR", Message: "Invalid agent id",
		})
		return
	}
	callerProjectID, ok := projectIDFromContext(c)
	if !ok || callerProjectID == uuid.Nil {
		respondError(c, &service.Error{
			Status:  http.StatusBadRequest, Code: "MISSING_PROJECT_HEADER", Message: "X-Project-ID header is required for this request",
		})
		return
	}
	caps, apiErr := h.agents.ListAgentCapabilities(c.Request.Context(), id, callerProjectID)
	if apiErr != nil {
		respondError(c, apiErr)
		return
	}
	out := make([]gin.H, 0, len(caps))
	for _, ac := range caps {
		entry := gin.H{
			"name":         ac.Name,
			"display_name": ac.DisplayName,
			"category":     ac.Category,
			"granted_at":   ac.GrantedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
		}
		if ac.Proficiency != nil {
			entry["proficiency"] = *ac.Proficiency
		}
		if ac.GrantedBy != nil {
			entry["granted_by"] = ac.GrantedBy.String()
		}
		out = append(out, entry)
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

// --- GET /v1/capabilities ----------------------------------------------

// ListCatalogCapabilities is the catalog read (api-spec.md §2.1).
// Query parameters:
//   - category: filter to one of the 6 categories
//     (architecture / coding / testing / security / devops / leadership)
//   - cursor:   opaque cursor from a previous page
//   - limit:    1-200, default 50
func (h *CapabilityHandler) ListCatalogCapabilities(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	res, apiErr := h.agents.ListCapabilities(c.Request.Context(), service.ListCapabilitiesRequest{
		Category: c.Query("category"),
		Cursor:   c.Query("cursor"),
		Limit:    limit,
	})
	if apiErr != nil {
		respondError(c, apiErr)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": res.Data,
		"pagination": gin.H{
			"has_more":    res.HasMore,
			"next_cursor": res.NextCursor,
		},
	})
}
