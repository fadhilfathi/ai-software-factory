package service

import (
	"time"

	"github.com/example/project/internal/model"
	"github.com/example/project/internal/store"
	"github.com/example/project/internal/validation"
	"go.uber.org/zap"
)

// TaskService handles task management operations.
type TaskService struct {
	store store.Store
	log   *zap.Logger
}

func NewTaskService(s store.Store, log *zap.Logger) *TaskService {
	return &TaskService{store: s, log: log}
}

// CreateTaskRequest carries task creation input.
type CreateTaskRequest struct {
	ProjectID          string
	Title              string
	Description        string
	Type               string
	AcceptanceCriteria []string
	Priority           string
	EstimatedHours     int
}

// CreateTask creates a new task in the backlog of a project.
func (s *TaskService) CreateTask(req CreateTaskRequest) (*model.Task, *Error) {
	var errs validation.Errors
	validation.NotEmpty(req.ProjectID, "project_id", "Project ID", &errs)
	validation.NotEmpty(req.Title, "title", "Title", &errs)
	validation.MaxLength(req.Title, 256, "title", "Title", &errs)
	if req.Priority != "" {
		validation.AllowedStrings(req.Priority, validTaskPriorities, "priority", "Priority", &errs)
	}
	validation.MinValue(req.EstimatedHours, 0, "estimated_hours", "Estimated hours", &errs)
	if errs.HasErrors() {
		return nil, validationError(errs)
	}

	// Verify project exists
	if _, err := s.store.Projects().GetByID(req.ProjectID); err != nil {
		return nil, notFound("Project not found")
	}

	priority := model.PriorityMedium
	if req.Priority != "" {
		priority = model.TaskPriority(req.Priority)
	}

	now := time.Now().UTC()
	task := &model.Task{
		ID:                 generateID("task"),
		ProjectID:          req.ProjectID,
		Title:              req.Title,
		Description:        req.Description,
		Type:               req.Type,
		AcceptanceCriteria: req.AcceptanceCriteria,
		Priority:           priority,
		Status:             model.TaskBacklog,
		EstimatedHours:     req.EstimatedHours,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := s.store.Tasks().Create(task); err != nil {
		s.log.Error("failed to create task", zap.Error(err))
		return nil, internalError("Failed to create task")
	}

	return task, nil
}

// UpdateTaskStatus updates a task's status with transition validation.
func (s *TaskService) UpdateTaskStatus(id, newStatus, assigneeAgentID string) (*model.Task, *Error) {
	var errs validation.Errors
	validation.NotEmpty(id, "id", "Task ID", &errs)
	validation.NotEmpty(newStatus, "status", "Status", &errs)
	if errs.HasErrors() {
		return nil, validationError(errs)
	}

	task, err := s.store.Tasks().GetByID(id)
	if err != nil {
		return nil, notFound("Task not found")
	}

	target := model.TaskStatus(newStatus)
	allowed, ok := taskStatusTransitions[task.Status]
	if !ok {
		return nil, validationSingle("status", "Invalid status transition from "+string(task.Status))
	}

	valid := false
	for _, a := range allowed {
		if a == target {
			valid = true
			break
		}
	}
	if !valid {
		return nil, validationSingle("status", "Cannot transition from "+string(task.Status)+" to "+newStatus)
	}

	task.Status = target
	task.AssigneeAgentID = assigneeAgentID
	task.UpdatedAt = time.Now().UTC()

	if err := s.store.Tasks().Update(task); err != nil {
		return nil, internalError("Failed to update task")
	}

	return task, nil
}

// ListProjectTasks returns paginated tasks for a project.
func (s *TaskService) ListProjectTasks(projectID string, status string, page, limit int) ([]*model.Task, *store.Pagination, *Error) {
	filter := store.TaskFilter{
		ProjectID: projectID,
		Status:    model.TaskStatus(status),
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
