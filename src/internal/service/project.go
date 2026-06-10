package service

import (
	"time"

	"github.com/example/project/internal/model"
	"github.com/example/project/internal/store"
	"github.com/example/project/internal/validation"
	"go.uber.org/zap"
)

// ProjectService handles project CRUD operations.
type ProjectService struct {
	store store.Store
	log   *zap.Logger
}

func NewProjectService(s store.Store, log *zap.Logger) *ProjectService {
	return &ProjectService{store: s, log: log}
}

// CreateProjectRequest carries project creation input.
type CreateProjectRequest struct {
	Name        string
	Description string
	Template    string
}

// CreateProject creates a new project with initializing status.
func (s *ProjectService) CreateProject(req CreateProjectRequest) (*model.Project, *Error) {
	var errs validation.Errors
	validation.NotEmpty(req.Name, "name", "Name", &errs)
	validation.MaxLength(req.Name, 128, "name", "Name", &errs)
	if errs.HasErrors() {
		return nil, validationError(errs)
	}

	now := time.Now().UTC()
	project := &model.Project{
		ID:            generateID("proj"),
		Name:          req.Name,
		Description:   req.Description,
		Status:        model.ProjectInitializing,
		Template:      req.Template,
		AgentsSpawned: []string{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.store.Projects().Create(project); err != nil {
		s.log.Error("failed to create project", zap.Error(err))
		return nil, internalError("Failed to create project")
	}

	// Auto-spawn PM agent
	pmAgent := &model.Agent{
		ID:        generateID("agent"),
		Type:      model.AgentPM,
		Status:    model.AgentIdle,
		ProjectID: project.ID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.store.Agents().Create(pmAgent); err != nil {
		s.log.Warn("failed to auto-spawn PM agent", zap.Error(err))
	} else {
		project.AgentsSpawned = append(project.AgentsSpawned, pmAgent.ID)
		project.ActiveAgents = 1
		s.store.Projects().Update(project)
	}

	return project, nil
}

// GetProject returns a project by ID.
func (s *ProjectService) GetProject(id string) (*model.Project, *Error) {
	project, err := s.store.Projects().GetByID(id)
	if err != nil {
		return nil, notFound("Project not found")
	}
	return project, nil
}

// ListProjects returns paginated projects with optional status filter.
func (s *ProjectService) ListProjects(status string, page, limit int) ([]*model.Project, *store.Pagination, *Error) {
	filter := store.ProjectFilter{
		Status: model.ProjectStatus(status),
		Page:   page,
		Limit:  limit,
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 20
	}

	projects, total, err := s.store.Projects().List(filter)
	if err != nil {
		return nil, nil, internalError("Failed to list projects")
	}

	pages := (total + filter.Limit - 1) / filter.Limit
	pagination := &store.Pagination{
		Page:  filter.Page,
		Limit: filter.Limit,
		Total: total,
		Pages: pages,
	}

	return projects, pagination, nil
}
