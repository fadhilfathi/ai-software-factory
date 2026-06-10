package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/example/project/internal/model"
	"github.com/example/project/internal/service"
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

type projectResponse struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Description   string        `json:"description,omitempty"`
	Status        string        `json:"status"`
	Progress      int           `json:"progress,omitempty"`
	ActiveAgents  int           `json:"active_agents,omitempty"`
	AgentsSpawned []string      `json:"agents_spawned,omitempty"`
	Artifacts     []interface{} `json:"artifacts,omitempty"`
	Agents        []interface{} `json:"agents,omitempty"`
	CreatedAt     string        `json:"created_at"`
}

// Create handles POST /projects.
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	project, svcErr := h.svc.CreateProject(service.CreateProjectRequest{
		Name:        req.Name,
		Description: req.Description,
		Template:    req.Template,
	})
	if svcErr != nil {
		writeServiceError(w, svcErr)
		return
	}

	writeJSON(w, http.StatusCreated, toProjectResponse(project))
}

// List handles GET /projects.
func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	projects, pagination, svcErr := h.svc.ListProjects(status, page, limit)
	if svcErr != nil {
		writeServiceError(w, svcErr)
		return
	}

	items := make([]projectResponse, 0, len(projects))
	for _, p := range projects {
		items = append(items, toProjectResponse(p))
	}

	writeJSON(w, http.StatusOK, PaginatedResponse{
		Data:       items,
		Pagination: Pagination{Page: pagination.Page, Limit: pagination.Limit, Total: pagination.Total, Pages: pagination.Pages},
	})
}

// Get handles GET /projects/{id}.
func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Project ID is required")
		return
	}

	project, svcErr := h.svc.GetProject(id)
	if svcErr != nil {
		writeServiceError(w, svcErr)
		return
	}

	writeJSON(w, http.StatusOK, toProjectResponse(project))
}

func toProjectResponse(p *model.Project) projectResponse {
	return projectResponse{
		ID:            p.ID,
		Name:          p.Name,
		Description:   p.Description,
		Status:        string(p.Status),
		Progress:      p.Progress,
		ActiveAgents:  p.ActiveAgents,
		AgentsSpawned: p.AgentsSpawned,
		Artifacts:     p.Artifacts,
		Agents:        p.Agents,
		CreatedAt:     p.CreatedAt.Format(time.RFC3339),
	}
}
