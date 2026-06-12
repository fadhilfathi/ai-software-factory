package service

import (
	"context"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ExecutionService struct {
	store store.Store
	log   *zap.Logger
}

func NewExecutionService(s store.Store, log *zap.Logger) *ExecutionService {
	return &ExecutionService{store: s, log: log}
}

func (s *ExecutionService) CreateExecution(ctx context.Context, taskID uuid.UUID, agentID uuid.UUID) (*model.Execution, *Error) {
	if _, err := s.store.Tasks().GetByID(taskID); err != nil {
		return nil, notFound("Task not found")
	}
	if _, err := s.store.Agents().GetByID(agentID); err != nil {
		return nil, notFound("Agent not found")
	}

	now := time.Now().UTC()
	exec := &model.Execution{
		ExecutionID: uuid.New(),
		TaskID:      taskID,
		AgentID:     agentID,
		Status:      model.ExecRunning,
		StartedAt:   &now,
		CreatedAt:   now,
	}

	if err := s.store.Executions().Create(exec); err != nil {
		s.log.Error("failed to create execution", zap.Error(err))
		return nil, internalError("Failed to create execution")
	}

	return exec, nil
}

func (s *ExecutionService) GetExecution(ctx context.Context, id uuid.UUID) (*model.Execution, *Error) {
	exec, err := s.store.Executions().GetByID(id)
	if err != nil {
		return nil, notFound("Execution not found")
	}
	return exec, nil
}

func (s *ExecutionService) ListExecutions(ctx context.Context, taskID uuid.UUID, agentID uuid.UUID, page, limit int) ([]*model.Execution, *store.Pagination, *Error) {
	filter := store.ExecutionFilter{
		TaskID:  taskID,
		AgentID: agentID,
		Page:    page,
		Limit:   limit,
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 20
	}

	execs, total, err := s.store.Executions().List(filter)
	if err != nil {
		s.log.Error("failed to list executions", zap.Error(err))
		return nil, nil, internalError("Failed to list executions")
	}

	pages := (total + filter.Limit - 1) / filter.Limit
	pagination := &store.Pagination{
		Page:  filter.Page,
		Limit: filter.Limit,
		Total: total,
		Pages: pages,
	}

	return execs, pagination, nil
}

func (s *ExecutionService) UpdateExecutionStatus(ctx context.Context, id uuid.UUID, status model.ExecutionStatus) (*model.Execution, *Error) {
	exec, err := s.store.Executions().GetByID(id)
	if err != nil {
		return nil, notFound("Execution not found")
	}

	if !isValidExecutionTransition(exec.Status, status) {
		return nil, unprocessableEntity("INVALID_TRANSITION", "Cannot transition from "+string(exec.Status)+" to "+string(status))
	}

	exec.Status = status
	now := time.Now().UTC()
	if status == model.ExecCompleted || status == model.ExecFailed {
		exec.CompletedAt = &now
	}

	if err := s.store.Executions().Update(exec); err != nil {
		s.log.Error("failed to update execution", zap.Error(err))
		return nil, internalError("Failed to update execution")
	}

	return exec, nil
}

func (s *ExecutionService) CompleteExecution(ctx context.Context, id uuid.UUID) (*model.Execution, *Error) {
	return s.UpdateExecutionStatus(ctx, id, model.ExecCompleted)
}

func isValidExecutionTransition(current, next model.ExecutionStatus) bool {
	transitions := map[model.ExecutionStatus][]model.ExecutionStatus{
		model.ExecPending:   {model.ExecRunning},
		model.ExecRunning:   {model.ExecCompleted, model.ExecFailed},
		model.ExecCompleted: {},
		model.ExecFailed:    {},
	}
	allowed, ok := transitions[current]
	if !ok {
		return false
	}
	for _, a := range allowed {
		if a == next {
			return true
		}
	}
	return false
}
