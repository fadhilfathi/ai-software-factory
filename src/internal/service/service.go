package service

import (
	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"go.uber.org/zap"
)

type Services struct {
	Auth        AuthService
	User        *UserService
	Project     *ProjectService
	Agent       *AgentService
	Task        *TaskService
	Code        *CodeService
	Review      *ReviewService
	Deployment  *DeploymentService
	Webhook     *WebhookService
	Assignment  *AssignmentService
	Execution   *ExecutionService
	Deliverable *DeliverableService
	AuditLog    *AuditLogService
}

func New(s store.Store, log *zap.Logger, jwtSecret string) *Services {
	capSvc := NewCapabilityService()
	return &Services{
		Auth:        NewAuthService(s, log, jwtSecret),
		User:        NewUserService(s, log),
		Project:     NewProjectService(s, log),
		Agent:       NewAgentService(s, log),
		Task:        NewTaskService(s, log),
		Code:        NewCodeService(s, log),
		Review:      NewReviewService(s, log),
		Deployment:  NewDeploymentService(s, log),
		Webhook:     NewWebhookService(s, log),
		Assignment:  NewAssignmentService(s, capSvc, log),
		Execution:   NewExecutionService(s, log),
		Deliverable: NewDeliverableService(s, log),
		AuditLog:    NewAuditLogService(s, log),
	}
}

var taskStatusTransitions = map[model.TaskStatus][]model.TaskStatus{
	model.TaskBacklog:    {model.TaskReady, model.TaskBlocked},
	model.TaskReady:      {model.TaskInProgress, model.TaskBlocked},
	model.TaskInProgress: {model.TaskReview, model.TaskBlocked},
	model.TaskReview:     {model.TaskDone, model.TaskBlocked},
	model.TaskDone:       {model.TaskBlocked},
	model.TaskBlocked:    {model.TaskBacklog, model.TaskReady, model.TaskInProgress, model.TaskReview, model.TaskDone},
}

var validTaskPriorities = []string{
	string(model.PriorityLow),
	string(model.PriorityMedium),
	string(model.PriorityHigh),
	string(model.PriorityCritical),
}

var validDeploymentEnvironments = []string{
	string(model.EnvDevelopment),
	string(model.EnvStaging),
	string(model.EnvProduction),
}
