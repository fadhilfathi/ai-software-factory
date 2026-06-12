package service

import (
	"context"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/fadhilfathi/AI-Software-Factory/internal/validation"
	"github.com/google/uuid"
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

// CreateProject creates a new project.
func (s *ProjectService) CreateProject(ctx context.Context, req CreateProjectRequest) (*model.Project, *Error) {
	var errs validation.Errors
	validation.NotEmpty(req.Name, "name", "Name", &errs)
	if errs.HasErrors() {
		return nil, validationError(errs)
	}

	now := time.Now().UTC()
	project := &model.Project{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		Status:      model.ProjectInitializing,
		Template:    req.Template,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.Projects().Create(project); err != nil {
		s.log.Error("failed to create project", zap.Error(err))
		return nil, internalError("Failed to create project")
	}

	return project, nil
}

// UpdateProject updates an existing project.
func (s *ProjectService) UpdateProject(ctx context.Context, id uuid.UUID, req UpdateProjectRequest) (*model.Project, *Error) {
	project, err := s.store.Projects().GetByID(id)
	if err != nil {
		return nil, notFound("Project not found")
	}

	project.Name = req.Name
	project.Description = req.Description
	project.UpdatedAt = time.Now().UTC()

	if err := s.store.Projects().Update(project); err != nil {
		return nil, internalError("Failed to update project")
	}
	return project, nil
}

// DeleteProject removes a project.
func (s *ProjectService) DeleteProject(ctx context.Context, id uuid.UUID) *Error {
	if err := s.store.Projects().Delete(id); err != nil {
		return internalError("Failed to delete project")
	}
	return nil
}

// DecomposeProject initiates PM agent task decomposition.
func (s *ProjectService) DecomposeProject(ctx context.Context, id uuid.UUID) *Error {
	project, err := s.store.Projects().GetByID(id)
	if err != nil {
		return notFound("Project not found")
	}
	// Logic to trigger PM Agent (e.g., via Event Bus)
	s.log.Info("Triggering decomposition for project", zap.String("project_id", project.ID.String()))
	return nil
}

// GetProject returns a project by ID.
func (s *ProjectService) GetProject(ctx context.Context, id uuid.UUID) (*model.Project, *Error) {
	project, err := s.store.Projects().GetByID(id)
	if err != nil {
		return nil, notFound("Project not found")
	}
	return project, nil
}

type CreateProjectRequest struct {
	Name        string
	Description string
	Template    string
}

type UpdateProjectRequest struct {
	Name        string
	Description string
}

