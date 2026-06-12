package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AgentHandler struct {
	svc *service.AgentService
}

func NewAgentHandler(svc *service.AgentService) *AgentHandler {
	return &AgentHandler{svc: svc}
}

type createAgentRequest struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Role         string   `json:"role"`
	Model        string   `json:"model"`
	Provider     string   `json:"provider"`
	Capabilities []string `json:"capabilities"`
}

type updateAgentRequest struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Role         string   `json:"role"`
	Model        string   `json:"model"`
	Provider     string   `json:"provider"`
	Capabilities []string `json:"capabilities"`
	Status       string   `json:"status"`
}

type agentResponse struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	Role          string   `json:"role"`
	Model         string   `json:"model"`
	Provider      string   `json:"provider"`
	Capabilities  []string `json:"capabilities"`
	Status        string   `json:"status"`
	ProjectID     string   `json:"project_id,omitempty"`
	CurrentTaskID string   `json:"current_task_id,omitempty"`
	TasksDone     int      `json:"tasks_done"`
	Uptime        int      `json:"uptime"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

func (h *AgentHandler) Create(c *gin.Context) {
	var req createAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	agent, svcErr := h.svc.CreateAgent(c.Request.Context(), service.CreateAgentRequest{
		Name:         req.Name,
		Type:         req.Type,
		Role:         req.Role,
		Model:        req.Model,
		Provider:     req.Provider,
		Capabilities: req.Capabilities,
	})
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	writeJSON(c, http.StatusCreated, toAgentResponse(agent))
}

func (h *AgentHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	filter := store.AgentFilter{
		Page:  page,
		Limit: limit,
	}
	if s := c.Query("status"); s != "" {
		filter.Status = model.AgentStatus(s)
	}
	if r := c.Query("role"); r != "" {
		filter.Role = r
	}
	if t := c.Query("type"); t != "" {
		filter.Type = t
	}
	if pid := c.Query("project_id"); pid != "" {
		if u, err := uuid.Parse(pid); err == nil {
			filter.ProjectID = u
		}
	}

	agents, pagination, svcErr := h.svc.ListAgents(c.Request.Context(), filter)
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	data := make([]agentResponse, len(agents))
	for i, a := range agents {
		data[i] = toAgentResponse(a)
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

func (h *AgentHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid Agent ID")
		return
	}

	agent, svcErr := h.svc.GetAgent(c.Request.Context(), id)
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	writeJSON(c, http.StatusOK, toAgentResponse(agent))
}

func (h *AgentHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid Agent ID")
		return
	}

	var req updateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	svcReq := service.UpdateAgentRequest{
		Name:         req.Name,
		Type:         req.Type,
		Role:         req.Role,
		Model:        req.Model,
		Provider:     req.Provider,
		Capabilities: req.Capabilities,
	}
	if req.Status != "" {
		svcReq.Status = model.AgentStatus(req.Status)
	}

	agent, svcErr := h.svc.UpdateAgent(c.Request.Context(), id, svcReq)
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	writeJSON(c, http.StatusOK, toAgentResponse(agent))
}

func (h *AgentHandler) Heartbeat(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid Agent ID")
		return
	}

	if svcErr := h.svc.Heartbeat(c.Request.Context(), id); svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *AgentHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid Agent ID")
		return
	}

	if svcErr := h.svc.DeleteAgent(c.Request.Context(), id); svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	c.Status(http.StatusNoContent)
}

func toAgentResponse(a *model.Agent) agentResponse {
	return agentResponse{
		ID:            a.ID.String(),
		Name:          a.Name,
		Type:          a.Type,
		Role:          a.Role,
		Model:         a.Model,
		Provider:      a.Provider,
		Capabilities:  a.Capabilities,
		Status:        string(a.Status),
		ProjectID:     a.ProjectID,
		CurrentTaskID: a.CurrentTaskID,
		TasksDone:     a.TasksDone,
		Uptime:        a.Uptime,
		CreatedAt:     a.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     a.UpdatedAt.Format(time.RFC3339),
	}
}
