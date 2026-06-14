package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ExecutionService is the consumer-side interface the handler
// depends on. *service.ExecutionService implements it. The
// interface is intentionally narrow (the four methods the HTTP
// layer actually calls) so a hand-rolled mock in the test file
// can stand in for the real service without a goroutine, a
// WaitGroup, or env-var handling.
// TASK-422 (F-016): all four methods now take callerProjectID
// uuid.UUID as the last argument. The handler resolves this from
// the X-Project-ID header (see projectIDFromContext) and the
// service enforces the cross-tenant boundary.
type ExecutionService interface {
	CreateExecution(ctx context.Context, taskID, agentID, callerProjectID uuid.UUID) (*model.Execution, error)
	GetExecution(ctx context.Context, id, callerProjectID uuid.UUID) (*model.Execution, error)
	ListExecutions(ctx context.Context, filter model.ExecutionFilter, callerProjectID uuid.UUID) (*model.ExecutionListResult, error)
	UpdateExecutionStatus(ctx context.Context, id uuid.UUID, newStatus model.ExecutionStatus, errorMessage *string, callerProjectID uuid.UUID) (*model.Execution, error)

	// B-001 reviewer action: lands an execution in COMPLETED (accepted=true)
	// or FAILED (accepted=false, reason required). Requires status=review.
	ReviewExecution(ctx context.Context, id uuid.UUID, accepted bool, reason string, projectID uuid.UUID) (*service.ReviewAction, error)

	// B-001 operator cancel: transitions a non-terminal execution to
	// FAILED with error_message='cancelled by operator'.
	CancelExecution(ctx context.Context, id uuid.UUID, projectID uuid.UUID) error
}

// ExecutionHandler is the Sprint 4 (TASK-405) HTTP layer for
// /v1/executions. It is a thin shell that:
//   - parses JSON bodies and query params
//   - calls the service
//   - maps service errors to HTTP status codes
//   - shapes the response envelope
//
// All routes require auth (the router wraps them with the
// auth middleware).
type ExecutionHandler struct {
	svc ExecutionService
	log *zap.Logger
}

// NewExecutionHandler constructs the handler. The svc may be a
// real *service.ExecutionService or a hand-rolled mock — both
// satisfy the local interface.
func NewExecutionHandler(svc ExecutionService, log *zap.Logger) *ExecutionHandler {
	return &ExecutionHandler{svc: svc, log: log}
}

// ----------------------------------------------------------------------------
// Request bodies
// ----------------------------------------------------------------------------

// createExecutionReq is the body for POST /v1/executions. Both
// fields are required UUIDs.
type createExecutionReq struct {
	TaskID  string `json:"task_id"  binding:"required,uuid"`
	AgentID string `json:"agent_id" binding:"required,uuid"`
}

// patchExecutionReq is the body for PATCH /v1/executions/:id.
// At least one of Status / ErrorMessage must be set; both are
// optional and validated independently.
type patchExecutionReq struct {
	Status       *string `json:"status,omitempty"`
	ErrorMessage *string `json:"error_message,omitempty"`
}

// reviewExecutionReq is the body for PATCH /v1/executions/:id/review.
//   - Accepted is the reviewer verdict: true lands the execution in
//     COMPLETED, false lands it in FAILED.
//   - Reason is the rejection reason (required when accepted=false,
//     silently ignored when accepted=true). Free-text, max 1 KiB.
type reviewExecutionReq struct {
	Accepted *bool  `json:"accepted"`
	Reason   string `json:"reason,omitempty"`
}

// ----------------------------------------------------------------------------
// POST /v1/executions
// ----------------------------------------------------------------------------

// Create handles POST /v1/executions. Returns 201 with the
// created execution (status=assigned under B-001) on success,
// 400 on a bad UUID, 404 if the task or agent does not exist,
// 500 otherwise.
//
// TASK-422: requires X-Project-ID header (400 MISSING_PROJECT_HEADER
// if absent). The project ID is forwarded to the service as the
// last argument to the cross-tenant check.
func (h *ExecutionHandler) Create(c *gin.Context) {
	callerProjectID, ok := projectIDFromContext(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "MISSING_PROJECT_HEADER", "X-Project-ID header is required for this request")
		return
	}

	var req createExecutionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()},
		})
		return
	}
	taskID, _ := uuid.Parse(req.TaskID)
	agentID, _ := uuid.Parse(req.AgentID)

	exec, err := h.svc.CreateExecution(c.Request.Context(), taskID, agentID, callerProjectID)
	if err != nil {
		h.mapError(c, err, "create execution")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": exec})
}

// ----------------------------------------------------------------------------
// GET /v1/executions
// ----------------------------------------------------------------------------

// List handles GET /v1/executions. Query params:
//   - task_id   (UUID, optional)
//   - agent_id  (UUID, optional)
//   - status    (string, optional; one of queued/assigned/running/review/completed/failed)
//   - limit     (int,    optional; default 50, max 200)
//   - cursor    (UUID,   optional; pass the NextCursor from a previous page)
//
// Returns 200 with an ExecutionListResult envelope. Unknown query
// params are ignored (forward compatibility). Bad UUIDs in task_id,
// agent_id, or cursor return 400.
func (h *ExecutionHandler) List(c *gin.Context) {
	callerProjectID, ok := projectIDFromContext(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "MISSING_PROJECT_HEADER", "X-Project-ID header is required for this request")
		return
	}

	filter := model.ExecutionFilter{}

	if raw := c.Query("task_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{"code": "BAD_REQUEST", "message": "task_id must be a UUID"},
			})
			return
		}
		filter.TaskID = id
	}
	if raw := c.Query("agent_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{"code": "BAD_REQUEST", "message": "agent_id must be a UUID"},
			})
			return
		}
		filter.AgentID = id
	}
	if raw := c.Query("status"); raw != "" {
		st := model.ExecutionStatus(raw)
		if !model.IsValidExecutionStatus(st) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "INVALID_EXECUTION_STATUS",
					"message": "status must be one of queued/assigned/running/review/completed/failed",
				},
			})
			return
		}
		filter.Status = st
	}
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{"code": "BAD_REQUEST", "message": "limit must be a non-negative integer"},
			})
			return
		}
		filter.Limit = n
	}
	if raw := c.Query("cursor"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{"code": "BAD_REQUEST", "message": "cursor must be a UUID"},
			})
			return
		}
		filter.Cursor = id
	}

	result, err := h.svc.ListExecutions(c.Request.Context(), filter, callerProjectID)
	if err != nil {
		h.mapError(c, err, "list executions")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

// ----------------------------------------------------------------------------
// GET /v1/executions/:id
// ----------------------------------------------------------------------------

// GetByID handles GET /v1/executions/:id. Returns 200 with the
// execution, 400 on a bad UUID, 404 on miss.
func (h *ExecutionHandler) GetByID(c *gin.Context) {
	callerProjectID, ok := projectIDFromContext(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "MISSING_PROJECT_HEADER", "X-Project-ID header is required for this request")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "BAD_REQUEST", "message": "id must be a UUID"},
		})
		return
	}
	exec, err := h.svc.GetExecution(c.Request.Context(), id, callerProjectID)
	if err != nil {
		h.mapError(c, err, "get execution")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": exec})
}

// ----------------------------------------------------------------------------
// PATCH /v1/executions/:id
// ----------------------------------------------------------------------------

// Patch handles PATCH /v1/executions/:id. Body is patchExecutionReq.
// Returns 200 with the updated execution, 400 on a bad body /
// bad status, 404 on miss, 409 on a disallowed state transition.
func (h *ExecutionHandler) Patch(c *gin.Context) {
	callerProjectID, ok := projectIDFromContext(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "MISSING_PROJECT_HEADER", "X-Project-ID header is required for this request")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "BAD_REQUEST", "message": "id must be a UUID"},
		})
		return
	}
	var req patchExecutionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()},
		})
		return
	}
	if req.Status == nil && req.ErrorMessage == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "at least one of status, error_message is required",
			},
		})
		return
	}
	var newStatus model.ExecutionStatus
	if req.Status != nil {
		newStatus = model.ExecutionStatus(*req.Status)
		if !model.IsValidExecutionStatus(newStatus) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "INVALID_EXECUTION_STATUS",
					"message": "status must be one of queued/assigned/running/review/completed/failed",
				},
			})
			return
		}
	}

	updated, err := h.svc.UpdateExecutionStatus(c.Request.Context(), id, newStatus, req.ErrorMessage, callerProjectID)
	if err != nil {
		// 404 is mapped first; 409 INVALID_STATE_TRANSITION is
		// mapped second; everything else is 500.
		h.mapError(c, err, "update execution")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": updated})
}

// ----------------------------------------------------------------------------
// Error mapping
// ----------------------------------------------------------------------------

// mapError centralises the service-error → HTTP-status translation.
// It is shared by every method so the error envelope stays
// consistent. We deliberately do NOT leak the wrapped error
// message to the client; the error_code is the public API.
func (h *ExecutionHandler) mapError(c *gin.Context, err error, op string) {
	// TASK-422 (F-016): cross-tenant is mapped FIRST so the
	// 404 path is unambiguous (an execution that doesn't exist
	// in the caller's project looks the same to the caller as
	// one that doesn't exist at all). Returning 404 (not 403)
	// avoids leaking the existence of resources in other
	// projects.
	switch {
	case errors.Is(err, service.ErrCrossTenantBlocked):
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "CROSS_TENANT_BLOCKED",
				"message": "the referenced resource is not in your project",
			},
		})
		return
	case errors.Is(err, service.ErrTaskNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"code": "TASK_NOT_FOUND", "message": "the referenced task does not exist"},
		})
		return
	case errors.Is(err, service.ErrAgentNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"code": "AGENT_NOT_FOUND", "message": "the referenced agent does not exist"},
		})
		return
	case errors.Is(err, service.ErrExecutionNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"code": "EXECUTION_NOT_FOUND", "message": "the requested execution does not exist"},
		})
		return
	case errors.Is(err, service.ErrInvalidStateTransition):
		c.JSON(http.StatusConflict, gin.H{
			"error": gin.H{
				"code":    "INVALID_STATE_TRANSITION",
				"message": "the requested status transition is not allowed for this execution",
			},
		})
		return
	}
	h.log.Error(op, zap.Error(err))
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": gin.H{"code": "INTERNAL", "message": "internal server error"},
	})
}
// ----------------------------------------------------------------------------
// PATCH /v1/executions/:id/review (B-001 reviewer action)
// ----------------------------------------------------------------------------

// reviewExecutionResponse is the success response shape for
// PATCH /v1/executions/:id/review. The api-spec §The Execution Engine
// B-001 spec asks for { data: { id, from, to, at } }.
type reviewExecutionResponse struct {
	Data *service.ReviewAction `json:"data"`
}

// Review handles PATCH /v1/executions/:id/review (B-001).
//
// Wire-level contract:
//   - 200 on success: body { data: { id, from: 'review', to: 'completed'|'failed', at } }
//   - 400 on bad UUID, missing/non-boolean `accepted`, missing reason when accepted=false, reason > 1 KiB
//   - 404 EXECUTION_NOT_FOUND if the execution doesn't exist
//   - 404 CROSS_TENANT_BLOCKED if the caller's project doesn't own the execution (F-014)
//   - 409 INVALID_STATE_TRANSITION if the execution is not in 'review'
//   - 500 on store error
//
// The reviewer is the ONLY path into COMPLETED. This is the user-facing
// endpoint that pairs with the runtime's driveWorker handoff (which lands
// in REVIEW). See docs/reset/audit-prep-B-001.md.
func (h *ExecutionHandler) Review(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid Execution ID")
		return
	}
	callerProjectID, ok := projectIDFromContext(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "MISSING_PROJECT_HEADER", "X-Project-ID header is required for this request")
		return
	}
	var req reviewExecutionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}
	if req.Accepted == nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "accepted is required")
		return
	}
	if !*req.Accepted && req.Reason == "" {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "reason is required when accepted=false")
		return
	}
	// Reason length cap (parallel to the assignment-notes cap from A-003).
	// 1 KiB keeps the audit trail compact.
	if len(req.Reason) > int(model.MaxAssignmentNotesBytes) {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "reason exceeds 1 KiB (1024 bytes)")
		return
	}
	action, svcErr := h.svc.ReviewExecution(
		c.Request.Context(),
		id,
		*req.Accepted,
		req.Reason,
		callerProjectID,
	)
	if svcErr != nil {
		h.mapError(c, svcErr, "review execution")
		return
	}
	c.JSON(http.StatusOK, reviewExecutionResponse{Data: action})
}

// ----------------------------------------------------------------------------
// DELETE /v1/executions/:id (B-001 operator cancel)
// ----------------------------------------------------------------------------

// Cancel handles DELETE /v1/executions/:id (B-001 operator cancel).
//
// Wire-level contract:
//   - 204 No Content on success
//   - 400 on bad UUID
//   - 404 EXECUTION_NOT_FOUND if the execution doesn't exist
//   - 404 CROSS_TENANT_BLOCKED if the caller's project doesn't own the execution (F-014)
//   - 409 INVALID_STATE_TRANSITION if the execution is already in a terminal state
//   - 500 on store error
//
// The router enforces admin/operator-role on this route (see
// internal/router/router.go), so the cross-tenant check in the
// service is the last line of defense, not the first.
func (h *ExecutionHandler) Cancel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid Execution ID")
		return
	}
	callerProjectID, ok := projectIDFromContext(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "MISSING_PROJECT_HEADER", "X-Project-ID header is required for this request")
		return
	}
	if svcErr := h.svc.CancelExecution(c.Request.Context(), id, callerProjectID); svcErr != nil {
		h.mapError(c, svcErr, "cancel execution")
		return
	}
	c.Status(http.StatusNoContent)
}
