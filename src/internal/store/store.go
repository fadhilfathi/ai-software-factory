package store

import (
	"errors"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/google/uuid"
)

// Sentinel errors for store operations.
var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrConflict      = errors.New("conflict")
)

// UserStore defines persistence operations for users.
type UserStore interface {
	Create(user *model.User) error
	GetByID(id uuid.UUID) (*model.User, error)
	GetByEmail(email string) (*model.User, error)
	List() ([]*model.User, error)
	Update(user *model.User) error
	// CheckProjectAccess returns true if the user has access to the project
	CheckProjectAccess(userID, projectID uuid.UUID) bool
}

// ProjectStore defines persistence operations for projects.
type ProjectStore interface {
	Create(project *model.Project) error
	GetByID(id uuid.UUID) (*model.Project, error)
	List(filter ProjectFilter) ([]*model.Project, int, error)
	Update(project *model.Project) error
	Delete(id uuid.UUID) error
}

// ProjectFilter holds optional query parameters for listing projects.
type ProjectFilter struct {
	Status  model.ProjectStatus
	OwnerID uuid.UUID
	Search  string
	Page    int
	Limit   int
}

// AgentStore defines persistence operations for agents.
type AgentStore interface {
	Create(agent *model.Agent) error
	GetByID(id uuid.UUID) (*model.Agent, error)
	List(filter AgentFilter) ([]*model.Agent, int, error)
	Update(agent *model.Agent) error
	Delete(id uuid.UUID) error
}

// AgentFilter holds optional query parameters for listing agents.
type AgentFilter struct {
	ProjectID uuid.UUID
	Status    model.AgentStatus
	Type      string
	Role      string
	Page      int
	Limit     int
}

// ExecutionStore defines persistence operations for executions.
type ExecutionStore interface {
	Create(exec *model.Execution) error
	GetByID(id uuid.UUID) (*model.Execution, error)
	List(filter ExecutionFilter) ([]*model.Execution, int, error)
	Update(exec *model.Execution) error
}

// ExecutionFilter holds optional query parameters for listing executions.
type ExecutionFilter struct {
	TaskID  uuid.UUID
	AgentID uuid.UUID
	Status  string
	Page    int
	Limit   int
}

// AgentRunStore defines persistence operations for agent runs.
type AgentRunStore interface {
	Create(run *model.AgentRun) error
	GetByID(id uuid.UUID) (*model.AgentRun, error)
	List(filter AgentRunFilter) ([]*model.AgentRun, int, error)
	Update(run *model.AgentRun) error
}

// AgentRunFilter holds optional query parameters for listing agent runs.
type AgentRunFilter struct {
	AgentID uuid.UUID
	TaskID  uuid.UUID
	Status  string
	Page    int
	Limit   int
}

// DeliverableStore defines persistence operations for deliverables.
type DeliverableStore interface {
	Create(d *model.Deliverable) error
	GetByID(id uuid.UUID) (*model.Deliverable, error)
	Update(d *model.Deliverable) error
	ListByTask(taskID uuid.UUID) ([]*model.Deliverable, error)
	ListByAgent(agentID uuid.UUID) ([]*model.Deliverable, error)
}

// TaskStore defines persistence operations for tasks.
type TaskStore interface {
	Create(task *model.Task) error
	GetByID(id uuid.UUID) (*model.Task, error)
	List(filter TaskFilter) ([]*model.Task, int, error)
	Update(task *model.Task) error
	Delete(id uuid.UUID) error
}

// TaskFilter holds optional query parameters for listing tasks.
type TaskFilter struct {
	ProjectID  uuid.UUID
	Status     model.TaskStatus
	AssigneeID uuid.UUID
	Page       int
	Limit      int
}

// CodeStore defines persistence operations for code generation and commits.
type CodeStore interface {
	CreateCodeGen(req *model.CodeGenRequest) error
	GetCodeGenByID(id uuid.UUID) (*model.CodeGenRequest, error)
	ListCodeGenByProject(projectID uuid.UUID) ([]*model.CodeGenRequest, error)
	UpdateCodeGen(req *model.CodeGenRequest) error

	SaveFile(file *model.ProjectFile) error
	GetFile(projectID uuid.UUID, path string) (*model.ProjectFile, error)
	ListFiles(projectID uuid.UUID) ([]*model.ProjectFile, error)

	CreateCommit(commit *model.Commit) error
	GetCommit(projectID uuid.UUID, sha string) (*model.Commit, error)
	ListCommits(projectID uuid.UUID) ([]*model.Commit, error)
}

// ReviewStore defines persistence operations for reviews.
type ReviewStore interface {
	Create(review *model.Review) error
	GetByID(id uuid.UUID) (*model.Review, error)
	ListByProject(projectID uuid.UUID) ([]*model.Review, error)
	Update(review *model.Review) error

	CreateComment(comment *model.ReviewComment) error
	ListComments(reviewID uuid.UUID) ([]*model.ReviewComment, error)
}

// DeploymentStore defines persistence operations for deployments.
type DeploymentStore interface {
	Create(deployment *model.Deployment) error
	GetByID(id uuid.UUID) (*model.Deployment, error)
	ListByProject(projectID uuid.UUID) ([]*model.Deployment, error)
	Update(deployment *model.Deployment) error
}

// WebhookStore defines persistence operations for webhooks.
type WebhookStore interface {
	Create(webhook *model.Webhook) error
	GetByID(id uuid.UUID) (*model.Webhook, error)
	List() ([]*model.Webhook, error)
	Update(webhook *model.Webhook) error
	Delete(id uuid.UUID) error
}

// AuditLogStore defines persistence operations for audit logs.
type AuditLogStore interface {
	Create(log *model.AuditLog) error
	List(filter AuditLogFilter) ([]*model.AuditLog, int, error)
}

// AuditLogFilter holds optional query parameters for listing audit logs.
type AuditLogFilter struct {
	EntityType string
	EntityID   uuid.UUID
	UserID     uuid.UUID
	Page       int
	Limit      int
}

// TokenStore defines operations for managing short-lived tokens (e.g. refresh tokens).
type TokenStore interface {
	Set(key string, userID uuid.UUID, ttl int) error
	Get(key string) (uuid.UUID, error)
	Delete(key string) error
}

// Pagination holds pagination metadata for list responses.
type Pagination struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
	Pages int `json:"pages"`
}

// Store combines all individual stores.
type Store interface {
	Users() UserStore
	Projects() ProjectStore
	Agents() AgentStore
	AgentRuns() AgentRunStore
	Executions() ExecutionStore
	Deliverables() DeliverableStore
	Tasks() TaskStore
	Code() CodeStore
	Reviews() ReviewStore
	Deployments() DeploymentStore
	Webhooks() WebhookStore
	AuditLogs() AuditLogStore
	Tokens() TokenStore
}
