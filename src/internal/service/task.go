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

type TaskService struct {
	store store.Store
	log   *zap.Logger
}

func NewTaskService(s store.Store, log *zap.Logger) *TaskService {
	return &TaskService{store: s, log: log}
}

type CreateTaskRequest struct {
	ProjectID   uuid.UUID
	Title       string
	Description string
	Priority    model.TaskPriority
}

type UpdateTaskRequest struct {
	Title       string
	Description string
	Priority    model.TaskPriority
	AssigneeID  uuid.UUID
}

func (s *TaskService) CreateTask(ctx context.Context, req CreateTaskRequest) (*model.Task, *Error) {
	var errs validation.Errors
	validation.NotEmpty(req.Title, "title", "Title", &errs)
	if errs.HasErrors() {
		return nil, validationError(errs)
	}

	if _, err := s.store.Projects().GetByID(req.ProjectID); err != nil {
		return nil, notFound("Project not found")
	}

	priority := req.Priority
	if priority == "" {
		priority = model.PriorityMedium
	}

	now := time.Now().UTC()
	task := &model.Task{
		ID:          uuid.New(),
		ProjectID:   req.ProjectID,
		Title:       req.Title,
		Description: req.Description,
		Status:      model.TaskBacklog,
		Priority:    priority,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.Tasks().Create(task); err != nil {
		s.log.Error("failed to create task", zap.Error(err))
		return nil, internalError("Failed to create task")
	}

	return task, nil
}

func (s *TaskService) GetTask(ctx context.Context, id uuid.UUID) (*model.Task, *Error) {
	task, err := s.store.Tasks().GetByID(id)
	if err != nil {
		return nil, notFound("Task not found")
	}
	return task, nil
}

func (s *TaskService) ListProjectTasks(ctx context.Context, projectID uuid.UUID, status model.TaskStatus, page, limit int) ([]*model.Task, *store.Pagination, *Error) {
	filter := store.TaskFilter{
		ProjectID: projectID,
		Status:    status,
		Page:      page,
		Limit:     limit,
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 20
	}

	tasks, total, err := s.store.Tasks().List(filter)
	if err != nil {
		s.log.Error("failed to list tasks", zap.Error(err))
		return nil, nil, internalError("Failed to list tasks")
	}

	pages := (total + filter.Limit - 1) / filter.Limit
	pagination := &store.Pagination{
		Page:  filter.Page,
		Limit: filter.Limit,
		Total: total,
		Pages: pages,
	}

	return tasks, pagination, nil
}

func (s *TaskService) UpdateTask(ctx context.Context, id uuid.UUID, req UpdateTaskRequest) (*model.Task, *Error) {
	task, err := s.store.Tasks().GetByID(id)
	if err != nil {
		return nil, notFound("Task not found")
	}

	if req.Title != "" {
		task.Title = req.Title
	}
	if req.Description != "" {
		task.Description = req.Description
	}
	if req.Priority != "" {
		task.Priority = req.Priority
	}
	if req.AssigneeID != uuid.Nil {
		task.AssigneeID = req.AssigneeID
	}
	task.UpdatedAt = time.Now().UTC()

	if err := s.store.Tasks().Update(task); err != nil {
		s.log.Error("failed to update task", zap.Error(err))
		return nil, internalError("Failed to update task")
	}

	return task, nil
}

func (s *TaskService) DeleteTask(ctx context.Context, id uuid.UUID) *Error {
	if err := s.store.Tasks().Delete(id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return notFound("Task not found")
		}
		return internalError("Failed to delete task")
	}
	return nil
}

func (s *TaskService) UpdateTaskStatus(ctx context.Context, id uuid.UUID, newStatus model.TaskStatus) (*model.Task, *Error) {
	task, err := s.store.Tasks().GetByID(id)
	if err != nil {
		return nil, notFound("Task not found")
	}

	allowed, ok := taskStatusTransitions[task.Status]
	if !ok {
		return nil, unprocessableEntity("INVALID_TRANSITION", "Invalid status transition from "+string(task.Status))
	}

	valid := false
	for _, a := range allowed {
		if a == newStatus {
			valid = true
			break
		}
	}
	if !valid {
		return nil, unprocessableEntity("INVALID_TRANSITION", "Cannot transition from "+string(task.Status)+" to "+string(newStatus))
	}

	task.Status = newStatus
	task.UpdatedAt = time.Now().UTC()

	if err := s.store.Tasks().Update(task); err != nil {
		s.log.Error("failed to update task status", zap.Error(err))
		return nil, internalError("Failed to update task status")
	}

	return task, nil
}
