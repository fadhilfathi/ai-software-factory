package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ExecutionHandler struct {
	svc *service.ExecutionService
}

func NewExecutionHandler(svc *service.ExecutionService) *ExecutionHandler {
	return &ExecutionHandler{svc: svc}
}

type createExecutionRequest struct {
	TaskID  string `json:"task_id"`
	AgentID string `json:"agent_id"`
}

type updateExecutionStatusRequest struct {
	Status string `json:"status"`
}

type executionResponse struct {
	ExecutionID string  `json:"execution_id"`
	TaskID      string  `json:"task_id"`
	AgentID     string  `json:"agent_id"`
	Status      string  `json:"status"`
	StartedAt   *string `json:"started_at,omitempty"`
	CompletedAt *string `json:"completed_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

func (h *ExecutionHandler) Create(c *gin.Context) {
	var req createExecutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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

	exec, svcErr := h.svc.CreateExecution(c.Request.Context(), taskID, agentID)
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	writeJSON(c, http.StatusCreated, toExecutionResponse(exec))
}

func (h *ExecutionHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	var taskID uuid.UUID
	if t := c.Query("task_id"); t != "" {
		taskID, _ = uuid.Parse(t)
	}
	var agentID uuid.UUID
	if a := c.Query("agent_id"); a != "" {
		agentID, _ = uuid.Parse(a)
	}

	execs, pagination, svcErr := h.svc.ListExecutions(c.Request.Context(), taskID, agentID, page, limit)
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	data := make([]executionResponse, len(execs))
	for i, e := range execs {
		data[i] = toExecutionResponse(e)
	}

	writeJSON(c, http.StatusOK, PaginatedResponse{
		Data: data,
		Pagination: Pagination{
			Page:  pagination.Page,
			Limit: pagination.Limit,
			Total: pagination.Total,
			Pages: pagination.Pages,
		},
	})
}

func (h *ExecutionHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid Execution ID")
		return
	}

	exec, svcErr := h.svc.GetExecution(c.Request.Context(), id)
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	writeJSON(c, http.StatusOK, toExecutionResponse(exec))
}

func (h *ExecutionHandler) UpdateStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid Execution ID")
		return
	}

	var req updateExecutionStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	if req.Status == "" {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Status is required")
		return
	}

	exec, svcErr := h.svc.UpdateExecutionStatus(c.Request.Context(), id, model.ExecutionStatus(req.Status))
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	writeJSON(c, http.StatusOK, toExecutionResponse(exec))
}

func toExecutionResponse(e *model.Execution) executionResponse {
	resp := executionResponse{
		ExecutionID: e.ExecutionID.String(),
		TaskID:      e.TaskID.String(),
		AgentID:     e.AgentID.String(),
		Status:      string(e.Status),
		CreatedAt:   e.CreatedAt.Format(time.RFC3339),
	}
	if e.StartedAt != nil {
		formatted := e.StartedAt.Format(time.RFC3339)
		resp.StartedAt = &formatted
	}
	if e.CompletedAt != nil {
		formatted := e.CompletedAt.Format(time.RFC3339)
		resp.CompletedAt = &formatted
	}
	return resp
}
