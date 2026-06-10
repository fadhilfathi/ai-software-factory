package service

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/example/project/internal/model"
	"github.com/example/project/internal/store"
	"go.uber.org/zap"
)

// Services aggregates all domain services.
type Services struct {
	Auth       *AuthService
	User       *UserService
	Project    *ProjectService
	Agent      *AgentService
	Task       *TaskService
	Code       *CodeService
	Review     *ReviewService
	Deployment *DeploymentService
	Webhook    *WebhookService
}

// New creates all domain services backed by the given store and logger.
func New(s store.Store, log *zap.Logger) *Services {
	return &Services{
		Auth:       NewAuthService(s, log),
		User:       NewUserService(s, log),
		Project:    NewProjectService(s, log),
		Agent:      NewAgentService(s, log),
		Task:       NewTaskService(s, log),
		Code:       NewCodeService(s, log),
		Review:     NewReviewService(s, log),
		Deployment: NewDeploymentService(s, log),
		Webhook:    NewWebhookService(s, log),
	}
}

// generateID creates a unique prefixed identifier.
func generateID(prefix string) string {
	b := make([]byte, 8)
	rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}

// validTransitions maps allowed status transitions for tasks.
var taskStatusTransitions = map[model.TaskStatus][]model.TaskStatus{
	model.TaskBacklog:    {model.TaskTodo},
	model.TaskTodo:       {model.TaskInProgress},
	model.TaskInProgress: {model.TaskReview},
	model.TaskReview:     {model.TaskDone},
}

// validTaskPriorities is the set of allowed task priority values.
var validTaskPriorities = []string{
	string(model.PriorityLow),
	string(model.PriorityMedium),
	string(model.PriorityHigh),
	string(model.PriorityCritical),
}

// validAgentTypes is the set of allowed agent type values.
var validAgentTypes = []string{
	string(model.AgentPM),
	string(model.AgentDev),
	string(model.AgentReviewer),
	string(model.AgentDevOps),
}

// validDeploymentEnvironments is the set of allowed environment values.
var validDeploymentEnvironments = []string{
	string(model.EnvDevelopment),
	string(model.EnvStaging),
	string(model.EnvProduction),
}
