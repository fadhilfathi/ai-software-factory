package store

import (
	"errors"

	"github.com/example/project/internal/model"
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
	GetByID(id string) (*model.User, error)
	GetByEmail(email string) (*model.User, error)
	List() ([]*model.User, error)
	Update(user *model.User) error
}

// ProjectStore defines persistence operations for projects.
type ProjectStore interface {
	Create(project *model.Project) error
	GetByID(id string) (*model.Project, error)
	List(filter ProjectFilter) ([]*model.Project, int, error)
	Update(project *model.Project) error
	Delete(id string) error
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
	GetByID(id string) (*model.Agent, error)
	List(filter AgentFilter) ([]*model.Agent, int, error)
	Update(agent *model.Agent) error
	Delete(id string) error
}

// AgentFilter holds optional query parameters for listing agents.
type AgentFilter struct {
	ProjectID string
	Page      int
	Limit     int
}

// TaskStore defines persistence operations for tasks.
type TaskStore interface {
	Create(task *model.Task) error
	GetByID(id string) (*model.Task, error)
	List(filter TaskFilter) ([]*model.Task, int, error)
	Update(task *model.Task) error
	Delete(id string) error
}

// TaskFilter holds optional query parameters for listing tasks.
type TaskFilter struct {
	ProjectID string
	Status    model.TaskStatus
	Page      int
	Limit     int
}

// CodeStore defines persistence operations for code generation and commits.
type CodeStore interface {
	CreateCodeGen(req *model.CodeGenRequest) error
	GetCodeGenByID(id string) (*model.CodeGenRequest, error)
	ListCodeGenByProject(projectID string) ([]*model.CodeGenRequest, error)
	UpdateCodeGen(req *model.CodeGenRequest) error

	SaveFile(file *model.ProjectFile) error
	GetFile(projectID, path string) (*model.ProjectFile, error)
	ListFiles(projectID string) ([]*model.ProjectFile, error)

	CreateCommit(commit *model.Commit) error
	GetCommit(projectID, sha string) (*model.Commit, error)
	ListCommits(projectID string) ([]*model.Commit, error)
}

// ReviewStore defines persistence operations for reviews.
type ReviewStore interface {
	Create(review *model.Review) error
	GetByID(id string) (*model.Review, error)
	ListByProject(projectID string) ([]*model.Review, error)
	Update(review *model.Review) error
}

// DeploymentStore defines persistence operations for deployments.
type DeploymentStore interface {
	Create(deployment *model.Deployment) error
	GetByID(id string) (*model.Deployment, error)
	ListByProject(projectID string) ([]*model.Deployment, error)
	Update(deployment *model.Deployment) error
}

// WebhookStore defines persistence operations for webhooks.
type WebhookStore interface {
	Create(webhook *model.Webhook) error
	GetByID(id string) (*model.Webhook, error)
	List() ([]*model.Webhook, error)
	Update(webhook *model.Webhook) error
	Delete(id string) error
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
