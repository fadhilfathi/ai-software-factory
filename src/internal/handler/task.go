package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/example/project/internal/service"
)

// TaskHandler handles task management endpoints.
type TaskHandler struct {
	svc *service.TaskService
}

func NewTaskHandler(svc *service.TaskService) *TaskHandler {
	return &TaskHandler{svc: svc}
}

type createTaskRequest struct {
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Type               string   `json:"type"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	Priority           string   `json:"priority"`
	EstimatedHours     int      `json:"estimated_hours"`
}

type taskResponse struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// Create handles POST /projects/{projectId}/tasks.
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectId")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Project ID is required")
		return
	}

	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	task, svcErr := h.svc.CreateTask(service.CreateTaskRequest{
		ProjectID:          projectID,
		Title:              req.Title,
		Description:        req.Description,
		Type:               req.Type,
		AcceptanceCriteria: req.AcceptanceCriteria,
		Priority:           req.Priority,
		EstimatedHours:     req.EstimatedHours,
	})
	if svcErr != nil {
		writeServiceError(w, svcErr)
		return
	}

	writeJSON(w, http.StatusCreated, taskResponse{
		ID:        task.ID,
		Title:     task.Title,
		Status:    string(task.Status),
		CreatedAt: task.CreatedAt.Format(time.RFC3339),
	})
}

type updateTaskRequest struct {
	Status          string `json:"status"`
	AssigneeAgentID string `json:"assignee_agent_id,omitempty"`
}

type updateTaskResponse struct {
	ID              string `json:"id"`
	Status          string `json:"status"`
	AssigneeAgentID string `json:"assignee_agent_id,omitempty"`
	UpdatedAt       string `json:"updated_at"`
}

// UpdateStatus handles PATCH /tasks/{id}.
func (h *TaskHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Task ID is required")
		return
	}

	var req updateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	task, svcErr := h.svc.UpdateTaskStatus(id, req.Status, req.AssigneeAgentID)
	if svcErr != nil {
		writeServiceError(w, svcErr)
		return
	}

	writeJSON(w, http.StatusOK, updateTaskResponse{
		ID:              task.ID,
		Status:          string(task.Status),
		AssigneeAgentID: task.AssigneeAgentID,
		UpdatedAt:       task.UpdatedAt.Format(time.RFC3339),
	})
}
