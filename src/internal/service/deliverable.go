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

type DeliverableService struct {
	store store.Store
	log   *zap.Logger
}

func NewDeliverableService(s store.Store, log *zap.Logger) *DeliverableService {
	return &DeliverableService{store: s, log: log}
}

type CreateDeliverableRequest struct {
	TaskID  uuid.UUID
	AgentID uuid.UUID
	Title   string
	Content string
}

type UpdateDeliverableRequest struct {
	Title   string
	Content string
}

func (s *DeliverableService) CreateDeliverable(ctx context.Context, req CreateDeliverableRequest) (*model.Deliverable, *Error) {
	var errs validation.Errors
	validation.NotEmpty(req.Title, "title", "Title", &errs)
	if errs.HasErrors() {
		return nil, validationError(errs)
	}

	if _, err := s.store.Tasks().GetByID(req.TaskID); err != nil {
		return nil, notFound("Task not found")
	}
	if _, err := s.store.Agents().GetByID(req.AgentID); err != nil {
		return nil, notFound("Agent not found")
	}

	now := time.Now().UTC()
	d := &model.Deliverable{
		ID:        uuid.New(),
		TaskID:    req.TaskID,
		AgentID:   req.AgentID,
		Title:     req.Title,
		Content:   req.Content,
		Version:   1,
		CreatedAt: now,
	}

	if err := s.store.Deliverables().Create(d); err != nil {
		s.log.Error("failed to create deliverable", zap.Error(err))
		return nil, internalError("Failed to create deliverable")
	}

	return d, nil
}

func (s *DeliverableService) GetDeliverable(ctx context.Context, id uuid.UUID) (*model.Deliverable, *Error) {
	d, err := s.store.Deliverables().GetByID(id)
	if err != nil {
		return nil, notFound("Deliverable not found")
	}
	return d, nil
}

func (s *DeliverableService) ListDeliverables(ctx context.Context, taskID uuid.UUID, agentID uuid.UUID) ([]*model.Deliverable, *Error) {
	switch {
	case taskID != uuid.Nil:
		deliverables, err := s.store.Deliverables().ListByTask(taskID)
		if err != nil {
			s.log.Error("failed to list deliverables by task", zap.Error(err))
			return nil, internalError("Failed to list deliverables")
		}
		return deliverables, nil
	case agentID != uuid.Nil:
		deliverables, err := s.store.Deliverables().ListByAgent(agentID)
		if err != nil {
			s.log.Error("failed to list deliverables by agent", zap.Error(err))
			return nil, internalError("Failed to list deliverables")
		}
		return deliverables, nil
	default:
		return nil, validationSingle("filter", "Provide task_id or agent_id to filter")
	}
}

func (s *DeliverableService) UpdateDeliverable(ctx context.Context, id uuid.UUID, req UpdateDeliverableRequest) (*model.Deliverable, *Error) {
	d, err := s.store.Deliverables().GetByID(id)
	if err != nil {
		return nil, notFound("Deliverable not found")
	}

	d.Title = req.Title
	d.Content = req.Content
	d.Version++

	if err := s.store.Deliverables().Update(d); err != nil {
		s.log.Error("failed to update deliverable", zap.Error(err))
		return nil, internalError("Failed to update deliverable")
	}

	return d, nil
}
