package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/middleware"
	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProjectHandler struct {
	svc *service.ProjectService
}

func NewProjectHandler(svc *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

type createProjectRequest struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Template    string      `json:"template"`
	Agents      []uuid.UUID `json:"agents"`
}

type updateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type projectResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	OwnerID      string `json:"owner_id"`
	Status       string `json:"status"`
	Progress     int    `json:"progress,omitempty"`
	ActiveAgents int    `json:"active_agents"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

func (h *ProjectHandler) Create(c *gin.Context) {
	var req createProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	uid, exists := c.Get(middleware.UserIDKey)
	if !exists {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	ownerID, err := uuid.Parse(uid.(string))
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user identity")
		return
	}

	project, svcErr := h.svc.CreateProject(c.Request.Context(), service.CreateProjectRequest{
		Name:        req.Name,
		Description: req.Description,
		Template:    req.Template,
		OwnerID:     ownerID,
		Agents:      req.Agents,
	})
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	writeJSON(c, http.StatusCreated, toProjectResponse(project))
}

func (h *ProjectHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	filter := store.ProjectFilter{
		Page:   page,
		Limit:  limit,
		Search: c.Query("search"),
	}
	if s := c.Query("status"); s != "" {
		filter.Status = model.ProjectStatus(s)
	}

	projects, pagination, svcErr := h.svc.ListProjects(c.Request.Context(), filter)
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	data := make([]projectResponse, len(projects))
	for i, p := range projects {
		data[i] = toProjectResponse(p)
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

func (h *ProjectHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid Project ID")
		return
	}

	project, svcErr := h.svc.GetProject(c.Request.Context(), id)
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	writeJSON(c, http.StatusOK, toProjectResponse(project))
}

func (h *ProjectHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid Project ID")
		return
	}

	var req updateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	svcReq := service.UpdateProjectRequest{
		Name:        req.Name,
		Description: req.Description,
	}
	if req.Status != "" {
		svcReq.Status = model.ProjectStatus(req.Status)
	}

	project, svcErr := h.svc.UpdateProject(c.Request.Context(), id, svcReq)
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	writeJSON(c, http.StatusOK, toProjectResponse(project))
}

func (h *ProjectHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid Project ID")
		return
	}

	if svcErr := h.svc.DeleteProject(c.Request.Context(), id); svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ProjectHandler) Decompose(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid Project ID")
		return
	}

	if svcErr := h.svc.DecomposeProject(c.Request.Context(), id); svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	c.Status(http.StatusAccepted)
}

func toProjectResponse(p *model.Project) projectResponse {
	return projectResponse{
		ID:           p.ID.String(),
		Name:         p.Name,
		Description:  p.Description,
		OwnerID:      p.OwnerID.String(),
		Status:       string(p.Status),
		Progress:     p.Progress,
		ActiveAgents: p.ActiveAgents,
		CreatedAt:    p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    p.UpdatedAt.Format(time.RFC3339),
	}
}
