package handler

import (
	"net/http"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DeliverableHandler struct {
	svc *service.DeliverableService
}

func NewDeliverableHandler(svc *service.DeliverableService) *DeliverableHandler {
	return &DeliverableHandler{svc: svc}
}

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

type deliverableResponse struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	AgentID   string `json:"agent_id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Version   int    `json:"version"`
	CreatedAt string `json:"created_at"`
}

func (h *DeliverableHandler) Create(c *gin.Context) {
	var req createDeliverableRequest
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

	d, svcErr := h.svc.CreateDeliverable(c.Request.Context(), service.CreateDeliverableRequest{
		TaskID:  taskID,
		AgentID: agentID,
		Title:   req.Title,
		Content: req.Content,
	})
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	writeJSON(c, http.StatusCreated, toDeliverableResponse(d))
}

func (h *DeliverableHandler) List(c *gin.Context) {
	var taskID uuid.UUID
	if t := c.Query("task_id"); t != "" {
		taskID, _ = uuid.Parse(t)
	}
	var agentID uuid.UUID
	if a := c.Query("agent_id"); a != "" {
		agentID, _ = uuid.Parse(a)
	}

	deliverables, svcErr := h.svc.ListDeliverables(c.Request.Context(), taskID, agentID)
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	data := make([]deliverableResponse, len(deliverables))
	for i, d := range deliverables {
		data[i] = toDeliverableResponse(d)
	}

	writeJSON(c, http.StatusOK, data)
}

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

func (h *DeliverableHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid Deliverable ID")
		return
	}

	var req updateDeliverableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	d, svcErr := h.svc.UpdateDeliverable(c.Request.Context(), id, service.UpdateDeliverableRequest{
		Title:   req.Title,
		Content: req.Content,
	})
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	writeJSON(c, http.StatusOK, toDeliverableResponse(d))
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
	}
}
