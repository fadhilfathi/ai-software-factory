package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AssignmentService is the consumer-side interface the AssignmentHandler
// depends on. Defined in the handler package (not the service package)
// so the handler can be tested with a hand-rolled mock without
// exporting an interface from the service implementation. The real
// *service.AssignmentService satisfies this interface structurally
// (Go's nominal typing only applies at the package boundary — both
// types have the same method set).
type AssignmentService interface {
	AssignTaskToAgent(
		ctx context.Context,
		taskID uuid.UUID,
		agentID uuid.UUID,
		notes string,
		assignedBy *uuid.UUID,
		capabilitiesRequired []string,
	) (*service.AssignmentResult, *service.Error)

	ListAssignmentHistory(
		ctx context.Context,
		taskID uuid.UUID,
	) ([]*model.AssignmentEvent, *service.Error)
}

type AssignmentHandler struct {
	svc AssignmentService
}

func NewAssignmentHandler(svc AssignmentService) *AssignmentHandler {
	return &AssignmentHandler{svc: svc}
}

// assignTaskRequest is the request body for POST /v1/tasks/:id/assign
// (TASK-404, api-spec.md §3.1).
//
//   - AgentID is required and must be a UUID.
//   - CapabilitiesRequired is optional. When non-empty it is persisted
//     to task.RequiredCapabilities (migration 018) and used as the
//     constraint set for the capability validation seam. When empty
//     the task's existing required_capabilities is preserved (not
//     nulled) per the brief.
//   - Notes is the optional audit-trail string for the
//     assignment_events row. Free-text, max ~1 KiB to keep the
//     append-only history compact.
type assignTaskRequest struct {
	AgentID              string   `json:"agent_id"`
	CapabilitiesRequired []string `json:"capabilities_required,omitempty"`
	Notes                string   `json:"notes,omitempty"`
}

// assignTaskResponse is the success response shape for
// POST /v1/tasks/:id/assign. Wraps the AssignmentResult DTO so the
// UI can render the updated task and the new event in one fetch.
type assignTaskResponse struct {
	Data *service.AssignmentResult `json:"data"`
}

// AssignTask handles POST /v1/tasks/:id/assign (TASK-404).
//
// Wire-level contract:
//   - 200 on success — body { data: { task, event, idempotent } }
//   - 200 on idempotent re-POST — body { data: { task, event: null, idempotent: true } }
//   - 400 on bad UUID, missing agent_id, malformed JSON
//   - 404 on missing task or agent
//   - 409 on capability mismatch (CAPABILITY_MISMATCH) or agent not idle
//   - 500 on store error
//
// The assignedBy UUID is sourced from c.Get("user_id") (set by the
// auth middleware, TASK-418). System-initiated callers (Sprint 5+
// autobalancer) will be wired through a different path.
func (h *AssignmentHandler) AssignTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid Task ID")
		return
	}

	var req assignTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}
	if req.AgentID == "" {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "agent_id is required")
		return
	}

	agentID, err := uuid.Parse(req.AgentID)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid agent_id format")
		return
	}

	// assignedBy: resolve from the middleware-set user_id. The
	// value is a string (UUID-formatted) per the middleware
	// contract. If absent (system caller), pass nil.
	var assignedBy *uuid.UUID
	if raw, ok := c.Get("user_id"); ok {
		if s, ok := raw.(string); ok && s != "" {
			if parsed, perr := uuid.Parse(s); perr == nil {
				assignedBy = &parsed
			}
		}
	}

	result, svcErr := h.svc.AssignTaskToAgent(
		c.Request.Context(),
		id,
		agentID,
		req.Notes,
		assignedBy,
		req.CapabilitiesRequired,
	)
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	// Notes (if any) are persisted by the service in the same
	// transaction as the assignment_events row (F-017 fix). The
	// service returns the row with Notes already set, so no
	// post-call mutation is needed.

	writeJSON(c, http.StatusOK, assignTaskResponse{Data: result})
}

// ListHistory handles GET /v1/tasks/:id/history (TASK-404).
//
// Wire-level contract:
//   - 200 with { data: [...] } (newest first)
//   - 200 with { data: [] } when the task exists but has no events
//   - 404 on missing task
//   - 500 on store error
//
// The response items are assignment_events rows shaped per
// model.AssignmentEvent (event id, task_id, agent_id, assigned_by,
// assigned_at, action, notes).
func (h *AssignmentHandler) ListHistory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid Task ID")
		return
	}

	events, svcErr := h.svc.ListAssignmentHistory(c.Request.Context(), id)
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	writeJSON(c, http.StatusOK, gin.H{
		"data": events,
		"meta": gin.H{
			"count":       len(events),
			"server_time": time.Now().UTC().Format(time.RFC3339),
		},
	})
}
