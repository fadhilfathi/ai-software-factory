package model

import (
	"time"

	"github.com/google/uuid"
)

// CodeGenStatus represents the lifecycle state of a code generation request.
type CodeGenStatus string

const (
	CodeGenQueued     CodeGenStatus = "queued"
	CodeGenGenerating CodeGenStatus = "generating"
	CodeGenCompleted  CodeGenStatus = "completed"
	CodeGenFailed     CodeGenStatus = "failed"
)

// CodeGenRequest represents a request to generate or modify code.
type CodeGenRequest struct {
	ID            uuid.UUID     `json:"id"`
	ProjectID     uuid.UUID     `json:"project_id"`
	TaskID        uuid.UUID     `json:"task_id"`
	Specification string        `json:"specification"`
	Files         []string      `json:"files"`
	Status        CodeGenStatus `json:"status"`
	ExecutionID   uuid.UUID     `json:"execution_id,omitempty"`
	Output        string        `json:"output,omitempty"`
	EstimatedTime int           `json:"estimated_time"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

// ProjectFile represents a file within a project's codebase.
type ProjectFile struct {
	ProjectID    uuid.UUID `json:"project_id"`
	Path         string    `json:"path"`
	Content      string    `json:"content"`
	Language     string    `json:"language"`
	Size         int       `json:"size"`
	LastModified time.Time `json:"last_modified"`
	ModifiedBy   string    `json:"modified_by"`
}

// Commit represents a commit to the project's codebase.
type Commit struct {
	SHA       string    `json:"sha"`
	ProjectID uuid.UUID `json:"project_id"`
	Branch    string    `json:"branch"`
	Message   string    `json:"message"`
	Author    string    `json:"author"`
	Files     []CommitFile `json:"files"`
	CreatedAt time.Time `json:"created_at"`
}

// CommitFile represents a single file change within a commit.
type CommitFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}
