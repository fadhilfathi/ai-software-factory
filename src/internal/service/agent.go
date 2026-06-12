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

type AgentService struct {
	store store.Store
	log   *zap.Logger
}

func NewAgentService(s store.Store, log *zap.Logger) *AgentService {
	return &AgentService{store: s, log: log}
}

type CreateAgentRequest struct {
	Name         string
	Type         string
	Role         string
	Model        string
	Provider     string
	Capabilities []string
	ProjectID    string
	Config       []byte
}

type UpdateAgentRequest struct {
	Name          string
	Type          string
	Role          string
	Model         string
	Provider      string
	Capabilities  []string
	Status        model.AgentStatus
	ProjectID     string
	Config        []byte
	CurrentTaskID string
	TasksDone     *int
	Uptime        *int
}

func (s *AgentService) CreateAgent(ctx context.Context, req CreateAgentRequest) (*model.Agent, *Error) {
	var errs validation.Errors
	validation.NotEmpty(req.Name, "name", "Name", &errs)
	validation.NotEmpty(req.Role, "role", "Role", &errs)
	if errs.HasErrors() {
		return nil, validationError(errs)
	}

	caps := req.Capabilities
	if len(caps) == 0 {
		caps = model.DefaultCapabilitiesForType(req.Type)
	}
	if caps == nil {
		caps = []string{}
	}

	now := time.Now().UTC()
	agent := &model.Agent{
		ID:           uuid.New(),
		Name:         req.Name,
		Type:         req.Type,
		Role:         req.Role,
		Model:        req.Model,
		Provider:     req.Provider,
		Capabilities: caps,
		ProjectID:    req.ProjectID,
		Config:       req.Config,
		Status:       model.AgentIdle,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.store.Agents().Create(agent); err != nil {
		s.log.Error("failed to create agent", zap.Error(err))
		return nil, internalError("Failed to create agent")
	}

	return agent, nil
}

func (s *AgentService) GetAgent(ctx context.Context, id uuid.UUID) (*model.Agent, *Error) {
	agent, err := s.store.Agents().GetByID(id)
	if err != nil {
		return nil, notFound("Agent not found")
	}
	return agent, nil
}

func (s *AgentService) ListAgents(ctx context.Context, filter store.AgentFilter) ([]*model.Agent, *store.Pagination, *Error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 20
	}

	agents, total, err := s.store.Agents().List(filter)
	if err != nil {
		s.log.Error("failed to list agents", zap.Error(err))
		return nil, nil, internalError("Failed to list agents")
	}

	pages := (total + filter.Limit - 1) / filter.Limit
	pagination := &store.Pagination{
		Page:  filter.Page,
		Limit: filter.Limit,
		Total: total,
		Pages: pages,
	}

	return agents, pagination, nil
}

func (s *AgentService) UpdateAgent(ctx context.Context, id uuid.UUID, req UpdateAgentRequest) (*model.Agent, *Error) {
	agent, err := s.store.Agents().GetByID(id)
	if err != nil {
		return nil, notFound("Agent not found")
	}

	if req.Name != "" {
		agent.Name = req.Name
	}
	if req.Type != "" {
		agent.Type = req.Type
	}
	if req.Role != "" {
		agent.Role = req.Role
	}
	if req.Model != "" {
		agent.Model = req.Model
	}
	if req.Provider != "" {
		agent.Provider = req.Provider
	}
	if req.Capabilities != nil {
		agent.Capabilities = req.Capabilities
	}
	if req.Status != "" {
		agent.Status = req.Status
	}
	if req.ProjectID != "" {
		agent.ProjectID = req.ProjectID
	}
	if req.Config != nil {
		agent.Config = req.Config
	}
	if req.CurrentTaskID != "" {
		agent.CurrentTaskID = req.CurrentTaskID
	}
	if req.TasksDone != nil {
		agent.TasksDone = *req.TasksDone
	}
	if req.Uptime != nil {
		agent.Uptime = *req.Uptime
	}
	agent.UpdatedAt = time.Now().UTC()

	if err := s.store.Agents().Update(agent); err != nil {
		s.log.Error("failed to update agent", zap.Error(err))
		return nil, internalError("Failed to update agent")
	}

	return agent, nil
}

// Heartbeat updates an agent's uptime and last seen timestamp.
func (s *AgentService) Heartbeat(ctx context.Context, id uuid.UUID) *Error {
	agent, err := s.store.Agents().GetByID(id)
	if err != nil {
		return notFound("Agent not found")
	}

	agent.Uptime += 30 // Assuming 30s heartbeat interval
	agent.UpdatedAt = time.Now().UTC()

	if err := s.store.Agents().Update(agent); err != nil {
		s.log.Error("failed to update heartbeat", zap.Error(err), zap.String("agent_id", id.String()))
		return internalError("Failed to update heartbeat")
	}

	return nil
}

// AssignTask assigns a task to an agent and updates its status.
func (s *AgentService) AssignTask(ctx context.Context, agentID uuid.UUID, taskID uuid.UUID) *Error {
	agent, err := s.store.Agents().GetByID(agentID)
	if err != nil {
		return notFound("Agent not found")
	}

	if agent.Status == model.AgentWorking {
		return conflict("Agent is already working on another task")
	}

	agent.CurrentTaskID = taskID.String()
	agent.Status = model.AgentWorking
	agent.UpdatedAt = time.Now().UTC()

	if err := s.store.Agents().Update(agent); err != nil {
		s.log.Error("failed to assign task", zap.Error(err), zap.String("agent_id", agentID.String()))
		return internalError("Failed to assign task")
	}

	return nil
}

// ReportCompletion reports that an agent has completed its current task.
func (s *AgentService) ReportCompletion(ctx context.Context, agentID uuid.UUID) *Error {
	agent, err := s.store.Agents().GetByID(agentID)
	if err != nil {
		return notFound("Agent not found")
	}

	agent.TasksDone++
	agent.CurrentTaskID = ""
	agent.Status = model.AgentIdle
	agent.UpdatedAt = time.Now().UTC()

	if err := s.store.Agents().Update(agent); err != nil {
		s.log.Error("failed to report completion", zap.Error(err), zap.String("agent_id", agentID.String()))
		return internalError("Failed to report completion")
	}

	return nil
}

func (s *AgentService) DeleteAgent(ctx context.Context, id uuid.UUID) *Error {
	if err := s.store.Agents().Delete(id); err != nil {
		return notFound("Agent not found")
	}
	return nil
}
