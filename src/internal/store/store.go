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
	Status model.ProjectStatus
	Page   int
	Limit  int
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
	Page      int
	Limit     int
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
	ProjectID uuid.UUID
	Status    model.TaskStatus
	Page      int
	Limit     int
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
	Tasks() TaskStore
	Code() CodeStore
	Reviews() ReviewStore
	Deployments() DeploymentStore
	Webhooks() WebhookStore
}
