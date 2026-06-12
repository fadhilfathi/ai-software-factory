package service

import (
	"context"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type AssignmentService struct {
	store     store.Store
	capSvc    *CapabilityService
	log       *zap.Logger
}

func NewAssignmentService(s store.Store, capSvc *CapabilityService, log *zap.Logger) *AssignmentService {
	return &AssignmentService{store: s, capSvc: capSvc, log: log}
}

func (s *AssignmentService) AssignTaskToAgent(ctx context.Context, taskID uuid.UUID, agentID uuid.UUID) (*model.Execution, *Error) {
	task, err := s.store.Tasks().GetByID(taskID)
	if err != nil {
		return nil, notFound("Task not found")
	}

	agent, err := s.store.Agents().GetByID(agentID)
	if err != nil {
		return nil, notFound("Agent not found")
	}

	if agent.Status != model.AgentIdle {
		return nil, conflict("Agent is not idle")
	}

	requiredCaps := s.capSvc.CapabilitiesForRole(agent.Role)
	if len(requiredCaps) == 0 {
		if len(agent.Capabilities) == 0 {
			return nil, unprocessableEntity("CAPABILITY_MISMATCH", "Agent has no capabilities")
		}
		requiredCaps = agent.Capabilities
	}

	agents := []*model.Agent{agent}
	compatible := s.capSvc.FindCompatibleAgents(agents, requiredCaps)
	if len(compatible) == 0 {
		return nil, unprocessableEntity("CAPABILITY_MISMATCH", "Agent lacks required capabilities for this task")
	}

	now := time.Now().UTC()
	execution := &model.Execution{
		ExecutionID: uuid.New(),
		TaskID:      taskID,
		AgentID:     agentID,
		Status:      model.ExecRunning,
		StartedAt:   &now,
		CreatedAt:   now,
	}

	if err := s.store.Executions().Create(execution); err != nil {
		s.log.Error("failed to create execution", zap.Error(err))
		return nil, internalError("Failed to create execution")
	}

	task.Status = model.TaskInProgress
	task.AssigneeAgentID = agentID.String()
	task.UpdatedAt = now
	if err := s.store.Tasks().Update(task); err != nil {
		s.log.Error("failed to update task", zap.Error(err))
		return nil, internalError("Failed to update task assignment")
	}

	agent.Status = model.AgentWorking
	agent.UpdatedAt = now
	if err := s.store.Agents().Update(agent); err != nil {
		s.log.Error("failed to update agent", zap.Error(err))
		return nil, internalError("Failed to update agent status")
	}

	return execution, nil
}
