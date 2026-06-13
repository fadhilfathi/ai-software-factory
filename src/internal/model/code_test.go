package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCodeGenStatusConstants(t *testing.T) {
	assert.Equal(t, CodeGenStatus("queued"), CodeGenQueued)
	assert.Equal(t, CodeGenStatus("generating"), CodeGenGenerating)
	assert.Equal(t, CodeGenStatus("completed"), CodeGenCompleted)
	assert.Equal(t, CodeGenStatus("failed"), CodeGenFailed)
}

func TestCodeGenRequestStruct(t *testing.T) {
	now := time.Now().UTC()
	req := CodeGenRequest{
		ID:            "codegen-123",
		ProjectID:     "proj-456",
		TaskID:        "task-789",
		Specification: "Create a REST API",
		Files:         []string{"main.go", "handler.go"},
		Status:        CodeGenGenerating,
		EstimatedTime: 120,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	assert.Equal(t, "codegen-123", req.ID)
	assert.Equal(t, "proj-456", req.ProjectID)
	assert.Equal(t, "task-789", req.TaskID)
	assert.Equal(t, "Create a REST API", req.Specification)
	assert.Equal(t, []string{"main.go", "handler.go"}, req.Files)
	assert.Equal(t, CodeGenGenerating, req.Status)
	assert.Equal(t, 120, req.EstimatedTime)
	assert.Equal(t, now, req.CreatedAt)
	assert.Equal(t, now, req.UpdatedAt)
}

func TestProjectFileStruct(t *testing.T) {
	now := time.Now().UTC()
	file := ProjectFile{
		ProjectID:    "proj-1",
		Path:         "src/main.go",
		Content:      "package main\n\nfunc main() {}\n",
		Language:     "go",
		Size:         30,
		LastModified: now,
		ModifiedBy:   "agent-123",
	}

	assert.Equal(t, "proj-1", file.ProjectID)
	assert.Equal(t, "src/main.go", file.Path)
	assert.Equal(t, "package main\n\nfunc main() {}\n", file.Content)
	assert.Equal(t, "go", file.Language)
	assert.Equal(t, 30, file.Size)
	assert.Equal(t, now, file.LastModified)
	assert.Equal(t, "agent-123", file.ModifiedBy)
}

func TestCommitStruct(t *testing.T) {
	now := time.Now().UTC()
	commit := Commit{
		SHA:       "abc123def456",
		ProjectID: "proj-1",
		Branch:    "main",
		Message:   "Initial commit",
		Author:    "agent-123",
		Files: []CommitFile{
			{Path: "main.go", Content: "package main\n"},
			{Path: "go.mod", Content: "module test\n"},
		},
		CreatedAt: now,
	}

	assert.Equal(t, "abc123def456", commit.SHA)
	assert.Equal(t, "proj-1", commit.ProjectID)
	assert.Equal(t, "main", commit.Branch)
	assert.Equal(t, "Initial commit", commit.Message)
	assert.Equal(t, "agent-123", commit.Author)
	assert.Len(t, commit.Files, 2)
	assert.Equal(t, "main.go", commit.Files[0].Path)
	assert.Equal(t, "package main\n", commit.Files[0].Content)
	assert.Equal(t, now, commit.CreatedAt)
}

func TestCommitFileStruct(t *testing.T) {
	file := CommitFile{
		Path:    "test.go",
		Content: "package test\n",
	}
	assert.Equal(t, "test.go", file.Path)
	assert.Equal(t, "package test\n", file.Content)
}