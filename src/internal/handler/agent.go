package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/example/project/internal/model"
	"github.com/example/project/internal/service"
)

// AgentHandler handles agent lifecycle endpoints.
type AgentHandler struct {
	svc *service.AgentService
}

func NewAgentHandler(svc *service.AgentService) *AgentHandler {
	return &AgentHandler{svc: svc}
}

type spawnAgentRequest struct {
	ProjectID string `json:"project_id"`
	Type      string `json:"type"`
	Config    *struct {
		Model       string  `json:"model"`
		Temperature float64 `json:"temperature"`
	} `json:"config,omitempty"`
}

type agentResponse struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	ProjectID   string `json:"project_id,omitempty"`
	CurrentTask string `json:"current_task,omitempty"`
	TasksDone   int    `json:"tasks_completed,omitempty"`
	Uptime      int    `json:"uptime,omitempty"`
}

// Spawn handles POST /agents/spawn.
func (h *AgentHandler) Spawn(w http.ResponseWriter, r *http.Request) {
	var req spawnAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	var cfg *model.AgentConfig
	if req.Config != nil {
		cfg = &model.AgentConfig{
			Model:       req.Config.Model,
			Temperature: req.Config.Temperature,
		}
	}

	agent, svcErr := h.svc.SpawnAgent(service.SpawnAgentRequest{
		ProjectID: req.ProjectID,
		Type:      req.Type,
		Config:    cfg,
	})
	if svcErr != nil {
		writeServiceError(w, svcErr)
		return
	}

	writeJSON(w, http.StatusCreated, agentResponse{
		ID:        agent.ID,
		Type:      string(agent.Type),
		Status:    string(agent.Status),
		ProjectID: agent.ProjectID,
	})
}

// List handles GET /agents.
func (h *AgentHandler) List(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	agents, pagination, svcErr := h.svc.ListAgents(projectID, page, limit)
	if svcErr != nil {
		writeServiceError(w, svcErr)
		return
	}

	items := make([]agentResponse, 0, len(agents))
	for _, a := range agents {
		items = append(items, agentResponse{
			ID:          a.ID,
			Type:        string(a.Type),
			Status:      string(a.Status),
			ProjectID:   a.ProjectID,
			CurrentTask: a.CurrentTask,
			TasksDone:   a.TasksDone,
			Uptime:      a.Uptime,
		})
	}

	writeJSON(w, http.StatusOK, PaginatedResponse{
		Data:       items,
		Pagination: Pagination{Page: pagination.Page, Limit: pagination.Limit, Total: pagination.Total, Pages: pagination.Pages},
	})
}

type assignTaskRequest struct {
	TaskID   string                 `json:"task_id"`
	Priority string                 `json:"priority"`
	Context  map[string]interface{} `json:"context,omitempty"`
}

type assignTaskResponse struct {
	ID                  string `json:"id"`
	TaskID              string `json:"task_id"`
	Status              string `json:"status"`
	EstimatedCompletion string `json:"estimated_completion,omitempty"`
}

// AssignTask handles POST /agents/{id}/assign.
func (h *AgentHandler) AssignTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Agent ID is required")
		return
	}

	var req assignTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	agent, svcErr := h.svc.AssignTask(id, service.AssignTaskRequest{
		TaskID:   req.TaskID,
		Priority: req.Priority,
		Context:  req.Context,
	})
	if svcErr != nil {
		writeServiceError(w, svcErr)
		return
	}

	writeJSON(w, http.StatusOK, assignTaskResponse{
		ID:     agent.ID,
		TaskID: req.TaskID,
		Status: string(agent.Status),
	})
}
