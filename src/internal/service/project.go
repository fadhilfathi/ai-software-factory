package service

import (
	"context"
	"errors"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/fadhilfathi/AI-Software-Factory/internal/validation"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ProjectService struct {
	store store.Store
	log   *zap.Logger
}

func NewProjectService(s store.Store, log *zap.Logger) *ProjectService {
	return &ProjectService{store: s, log: log}
}

type CreateProjectRequest struct {
	Name        string
	Description string
	Template    string
	OwnerID     uuid.UUID
}

type UpdateProjectRequest struct {
	Name        string
	Description string
	Status      model.ProjectStatus
}

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
		OwnerID:     req.OwnerID,
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

func (s *ProjectService) GetProject(ctx context.Context, id uuid.UUID) (*model.Project, *Error) {
	project, err := s.store.Projects().GetByID(id)
	if err != nil {
		return nil, notFound("Project not found")
	}
	return project, nil
}

func (s *ProjectService) ListProjects(ctx context.Context, filter store.ProjectFilter) ([]*model.Project, *store.Pagination, *Error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 20
	}

	projects, total, err := s.store.Projects().List(filter)
	if err != nil {
		s.log.Error("failed to list projects", zap.Error(err))
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

func (s *ProjectService) UpdateProject(ctx context.Context, id uuid.UUID, req UpdateProjectRequest) (*model.Project, *Error) {
	project, err := s.store.Projects().GetByID(id)
	if err != nil {
		return nil, notFound("Project not found")
	}

	if req.Name != "" {
		project.Name = req.Name
	}
	if req.Description != "" {
		project.Description = req.Description
	}
	if req.Status != "" {
		project.Status = req.Status
	}
	project.UpdatedAt = time.Now().UTC()

	if err := s.store.Projects().Update(project); err != nil {
		s.log.Error("failed to update project", zap.Error(err))
		return nil, internalError("Failed to update project")
	}
	return project, nil
}

func (s *ProjectService) DeleteProject(ctx context.Context, id uuid.UUID) *Error {
	if err := s.store.Projects().Delete(id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return notFound("Project not found")
		}
		return internalError("Failed to delete project")
	}
	return nil
}

func (s *ProjectService) DecomposeProject(ctx context.Context, id uuid.UUID) *Error {
	project, err := s.store.Projects().GetByID(id)
	if err != nil {
		return notFound("Project not found")
	}
	s.log.Info("Triggering decomposition for project", zap.String("project_id", project.ID.String()))
	return nil
}
