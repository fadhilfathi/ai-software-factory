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

// ProjectHandler handles project CRUD endpoints.
type ProjectHandler struct {
	svc *service.ProjectService
}

func NewProjectHandler(svc *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

type createProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Template    string `json:"template"`
}

type updateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type projectResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

// Create handles POST /projects.
func (h *ProjectHandler) Create(c *gin.Context) {
	var req createProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	project, svcErr := h.svc.CreateProject(c.Request.Context(), service.CreateProjectRequest{
		Name:        req.Name,
		Description: req.Description,
		Template:    req.Template,
	})
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	writeJSON(c, http.StatusCreated, toProjectResponse(project))
}

// Update handles PATCH /projects/{id}.
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

	project, svcErr := h.svc.UpdateProject(c.Request.Context(), id, service.UpdateProjectRequest{
		Name:        req.Name,
		Description: req.Description,
	})
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	writeJSON(c, http.StatusOK, toProjectResponse(project))
}

// Delete handles DELETE /projects/{id}.
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

// Decompose handles POST /projects/{id}/decompose.
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

// Get handles GET /projects/{id}.
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

func toProjectResponse(p *model.Project) projectResponse {
	return projectResponse{
		ID:          p.ID.String(),
		Name:        p.Name,
		Description: p.Description,
		Status:      string(p.Status),
		CreatedAt:   p.CreatedAt.Format(time.RFC3339),
	}
}
