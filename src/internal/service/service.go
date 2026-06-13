package service

import (
	"github.com/fadhilfathi/AI-Software-Factory/internal/config"
	"github.com/fadhilfathi/AI-Software-Factory/internal/aion"
	"github.com/fadhilfathi/AI-Software-Factory/internal/events"
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

	// Bus (TASK-503 Sprint 5, minimal): the in-process event bus,
	// instantiated in main.go and threaded through here. The
	// ExecutionService does NOT yet read it (the publish-on-
	// transition refactor is Sprint 6 per Lead's dispatch
	// 2026-06-14); it is exposed so future TASK-501/TASK-505/
	// TASK-506 code can subscribe and publish without having
	// to add a new constructor parameter. nil is allowed at
	// construction time but main.go always wires a real bus.
	Bus events.Bus
}

// New constructs the full service container. The bus argument
// (TASK-503, Sprint 5 minimal) is stored on Services.Bus for
// future publishers; the ExecutionService does not read it yet.
// See the struct comment on Bus.
func New(s store.Store, apiKeys store.APIKeyStore, log *zap.Logger, cfg *config.Config, bus events.Bus) *Services {
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
		Execution:    NewExecutionService(s, log, nil, aion.NewProcessRuntime(aion.ProcessRuntimeConfig{})), // nil cfg → DefaultExecutionServiceConfig; aion.NewProcessRuntime wires the TASK-501 subprocess runtime (AION_BINARY etc. env vars read inside)
		Deliverable:  NewDeliverableService(s, log),
		AuditLog:     NewAuditLogService(s, log),
		Orchestrator: orch,
		Sandbox:      sandbox,
		Log:          log,
		Bus:          bus,
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
