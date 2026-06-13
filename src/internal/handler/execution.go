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
type ExecutionService interface {
	CreateExecution(ctx context.Context, taskID, agentID uuid.UUID) (*model.Execution, error)
	GetExecution(ctx context.Context, id uuid.UUID) (*model.Execution, error)
	ListExecutions(ctx context.Context, filter model.ExecutionFilter) (*model.ExecutionListResult, error)
	UpdateExecutionStatus(ctx context.Context, id uuid.UUID, newStatus model.ExecutionStatus, errorMessage *string) (*model.Execution, error)
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

// ----------------------------------------------------------------------------
// POST /v1/executions
// ----------------------------------------------------------------------------

// Create handles POST /v1/executions. Returns 201 with the
// created execution (status=pending) on success, 400 on a bad
// UUID, 404 if the task or agent does not exist, 500 otherwise.
func (h *ExecutionHandler) Create(c *gin.Context) {
	var req createExecutionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()},
		})
		return
	}
	taskID, _ := uuid.Parse(req.TaskID)
	agentID, _ := uuid.Parse(req.AgentID)

	exec, err := h.svc.CreateExecution(c.Request.Context(), taskID, agentID)
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
//   - status    (string, optional; one of pending/running/completed/failed)
//   - limit     (int,    optional; default 50, max 200)
//   - cursor    (UUID,   optional; pass the NextCursor from a previous page)
//
// Returns 200 with an ExecutionListResult envelope. Unknown query
// params are ignored (forward compatibility). Bad UUIDs in task_id,
// agent_id, or cursor return 400.
func (h *ExecutionHandler) List(c *gin.Context) {
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
					"message": "status must be one of pending/running/completed/failed",
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

	result, err := h.svc.ListExecutions(c.Request.Context(), filter)
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
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "BAD_REQUEST", "message": "id must be a UUID"},
		})
		return
	}
	exec, err := h.svc.GetExecution(c.Request.Context(), id)
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
					"message": "status must be one of pending/running/completed/failed",
				},
			})
			return
		}
	}

	updated, err := h.svc.UpdateExecutionStatus(c.Request.Context(), id, newStatus, req.ErrorMessage)
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
	switch {
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
