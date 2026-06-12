package service

import (
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/fadhilfathi/AI-Software-Factory/internal/validation"
	"go.uber.org/zap"
)

// AgentService handles agent lifecycle operations.
type AgentService struct {
	store store.Store
	log   *zap.Logger
}

func NewAgentService(s store.Store, log *zap.Logger) *AgentService {
	return &AgentService{store: s, log: log}
}

// SpawnAgentRequest carries agent spawn input.
type SpawnAgentRequest struct {
	ProjectID string
	Type      string
	Config    *model.AgentConfig
}

// SpawnAgent creates and starts a new agent in a project.
func (s *AgentService) SpawnAgent(req SpawnAgentRequest) (*model.Agent, *Error) {
	var errs validation.Errors
	validation.NotEmpty(req.ProjectID, "project_id", "Project ID", &errs)
	validation.NotEmpty(req.Type, "type", "Agent type", &errs)
	validation.AllowedStrings(req.Type, validAgentTypes, "type", "Agent type", &errs)
	if errs.HasErrors() {
		return nil, validationError(errs)
	}

	// Verify project exists
	if _, err := s.store.Projects().GetByID(req.ProjectID); err != nil {
		return nil, notFound("Project not found")
	}

	now := time.Now().UTC()
	agent := &model.Agent{
		ID:        generateID("agent"),
		Type:      model.AgentType(req.Type),
		Status:    model.AgentSpawning,
		ProjectID: req.ProjectID,
		Config:    req.Config,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.store.Agents().Create(agent); err != nil {
		s.log.Error("failed to create agent", zap.Error(err))
		return nil, internalError("Failed to spawn agent")
	}

	// Set to idle after creation
	agent.Status = model.AgentIdle
	s.store.Agents().Update(agent)

	// Update project active agent count
	if project, err := s.store.Projects().GetByID(req.ProjectID); err == nil {
		project.ActiveAgents++
		s.store.Projects().Update(project)
	}

	return agent, nil
}

// ListAgents returns paginated agents with optional project filter.
func (s *AgentService) ListAgents(projectID string, page, limit int) ([]*model.Agent, *store.Pagination, *Error) {
	filter := store.AgentFilter{
		ProjectID: projectID,
		Page:      page,
		Limit:     limit,
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 20
	}

	agents, total, err := s.store.Agents().List(filter)
	if err != nil {
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

// AssignTaskRequest carries the input for assigning a task to an agent.
type AssignTaskRequest struct {
	TaskID   string
	Priority string
	Context  map[string]interface{}
}

// AssignTask assigns a task to an agent.
func (s *AgentService) AssignTask(agentID string, req AssignTaskRequest) (*model.Agent, *Error) {
	var errs validation.Errors
	validation.NotEmpty(agentID, "agent_id", "Agent ID", &errs)
	validation.NotEmpty(req.TaskID, "task_id", "Task ID", &errs)
	if errs.HasErrors() {
		return nil, validationError(errs)
	}

	agent, err := s.store.Agents().GetByID(agentID)
	if err != nil {
		return nil, notFound("Agent not found")
	}

	task, err := s.store.Tasks().GetByID(req.TaskID)
	if err != nil {
		return nil, notFound("Task not found")
	}

	// Update agent state
	agent.Status = model.AgentWorking
	agent.CurrentTask = req.TaskID
	agent.UpdatedAt = time.Now().UTC()
	s.store.Agents().Update(agent)

	// Update task state
	task.Status = model.TaskInProgress
	task.AssigneeAgentID = agentID
	task.UpdatedAt = time.Now().UTC()
	s.store.Tasks().Update(task)

	return agent, nil
}
