package handler

import (
	"net/http"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AssignmentHandler struct {
	svc *service.AssignmentService
}

func NewAssignmentHandler(svc *service.AssignmentService) *AssignmentHandler {
	return &AssignmentHandler{svc: svc}
}

type assignTaskRequest struct {
	AgentID string `json:"agent_id"`
}

type executionResponse struct {
	ExecutionID string  `json:"execution_id"`
	TaskID      string  `json:"task_id"`
	AgentID     string  `json:"agent_id"`
	Status      string  `json:"status"`
	StartedAt   *string `json:"started_at,omitempty"`
}

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

	agentID, err := uuid.Parse(req.AgentID)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid agent_id format")
		return
	}

	exec, svcErr := h.svc.AssignTaskToAgent(c.Request.Context(), id, agentID)
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	resp := executionResponse{
		ExecutionID: exec.ExecutionID.String(),
		TaskID:      exec.TaskID.String(),
		AgentID:     exec.AgentID.String(),
		Status:      string(exec.Status),
	}
	if exec.StartedAt != nil {
		formatted := exec.StartedAt.Format(time.RFC3339)
		resp.StartedAt = &formatted
	}

	writeJSON(c, http.StatusOK, resp)
}
