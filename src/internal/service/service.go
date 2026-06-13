package service

import (
	"github.com/fadhilfathi/AI-Software-Factory/internal/config"
	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"go.uber.org/zap"
)

type Services struct {
	Auth         AuthService
	User         *UserService
	Project      *ProjectService
	Agent        AgentService
	Task         *TaskService
	Code         *CodeService
	Review       *ReviewService
	Deployment   *DeploymentService
	Webhook      *WebhookService
	Assignment   *AssignmentService
	Execution    *ExecutionService
	Deliverable  *DeliverableService
	AuditLog     *AuditLogService
	Orchestrator AgentOrchestrator
	Sandbox      *SandboxService
	Log          *zap.Logger
}

func New(s store.Store, apiKeys store.APIKeyStore, log *zap.Logger, cfg *config.Config) *Services {
	// CapabilityService: takes a store.Store so the TASK-403
	// ValidateAgentHasCapabilities seam can read the live
	// agent_capabilities join table.
	capSvc := NewCapabilityService(s, log)

	// DeliverableSvc is constructed lazily in New() because it has
	// no store dependency in the pre-Sprint-4 code path; TASK-406
	// will likely change that.
	orch, err := NewAgentOrchestrator(s, cfg, log)
	if err != nil {
		log.Warn("failed to initialize agent orchestrator", zap.Error(err))
	}

	sandbox, err := NewSandboxService(cfg, log)
	if err != nil {
		log.Warn("failed to initialize sandbox service", zap.Error(err))
	}
	
	return &Services{
		Auth:         NewAuthService(s, apiKeys, log, cfg.Auth.JWTSecret),
		User:         NewUserService(s, log),
		Project:      NewProjectService(s, log),
		Agent:        NewAgentService(s),
		Task:         NewTaskService(s, log),
		Code:         NewCodeService(s, log),
		Review:       NewReviewService(s, orch, log),
		Deployment:   NewDeploymentService(s, log),
		Webhook:      NewWebhookService(s, log),
		Assignment:   NewAssignmentService(s, capSvc, log),
		Execution:    NewExecutionService(s, log, nil), // nil cfg → DefaultExecutionServiceConfig (env-var-driven failure rate)
		Deliverable:  NewDeliverableService(s, log),
		AuditLog:     NewAuditLogService(s, log),
		Orchestrator: orch,
		Sandbox:      sandbox,
		Log:          log,
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
