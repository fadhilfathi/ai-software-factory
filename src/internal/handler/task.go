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

type TaskHandler struct {
	svc *service.TaskService
}

func NewTaskHandler(svc *service.TaskService) *TaskHandler {
	return &TaskHandler{svc: svc}
}

type createTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
}

type updateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	AssigneeID  string `json:"assignee_id"`
}

type updateTaskStatusRequest struct {
	Status string `json:"status"`
}

type taskResponse struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	AssigneeID  string `json:"assignee_id,omitempty"`
	Position    int    `json:"position,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func (h *TaskHandler) Create(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid Project ID")
		return
	}

	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	priority := model.PriorityMedium
	if req.Priority != "" {
		priority = model.TaskPriority(req.Priority)
	}

	task, svcErr := h.svc.CreateTask(c.Request.Context(), service.CreateTaskRequest{
		ProjectID:   projectID,
		Title:       req.Title,
		Description: req.Description,
		Priority:    priority,
	})
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	writeJSON(c, http.StatusCreated, toTaskResponse(task))
}

func (h *TaskHandler) List(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid Project ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	status := model.TaskStatus(c.Query("status"))

	tasks, pagination, svcErr := h.svc.ListProjectTasks(c.Request.Context(), projectID, status, page, limit)
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	data := make([]taskResponse, len(tasks))
	for i, t := range tasks {
		data[i] = toTaskResponse(t)
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

func (h *TaskHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid Task ID")
		return
	}

	task, svcErr := h.svc.GetTask(c.Request.Context(), id)
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	writeJSON(c, http.StatusOK, toTaskResponse(task))
}

func (h *TaskHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid Task ID")
		return
	}

	var req updateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	svcReq := service.UpdateTaskRequest{
		Title:       req.Title,
		Description: req.Description,
	}
	if req.Priority != "" {
		svcReq.Priority = model.TaskPriority(req.Priority)
	}
	if req.AssigneeID != "" {
		parsed, err := uuid.Parse(req.AssigneeID)
		if err != nil {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid assignee_id format")
			return
		}
		svcReq.AssigneeID = parsed
	}

	task, svcErr := h.svc.UpdateTask(c.Request.Context(), id, svcReq)
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	writeJSON(c, http.StatusOK, toTaskResponse(task))
}

func (h *TaskHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid Task ID")
		return
	}

	if svcErr := h.svc.DeleteTask(c.Request.Context(), id); svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *TaskHandler) UpdateStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid Task ID")
		return
	}

	var req updateTaskStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	if req.Status == "" {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Status is required")
		return
	}

	task, svcErr := h.svc.UpdateTaskStatus(c.Request.Context(), id, model.TaskStatus(req.Status))
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	writeJSON(c, http.StatusOK, toTaskResponse(task))
}

func toTaskResponse(t *model.Task) taskResponse {
	resp := taskResponse{
		ID:          t.ID.String(),
		ProjectID:   t.ProjectID.String(),
		Title:       t.Title,
		Description: t.Description,
		Status:      string(t.Status),
		Priority:    string(t.Priority),
		Position:    t.Position,
		CreatedAt:   t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   t.UpdatedAt.Format(time.RFC3339),
	}
	if t.AssigneeID != uuid.Nil {
		resp.AssigneeID = t.AssigneeID.String()
	}
	return resp
}
