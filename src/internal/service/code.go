package service

import (
	"context"
	"strings"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/fadhilfathi/AI-Software-Factory/internal/validation"
	"go.uber.org/zap"
)

// CodeService handles code generation and file management.
type CodeService struct {
	store store.Store
	log   *zap.Logger
}

func NewCodeService(s store.Store, log *zap.Logger) *CodeService {
	return &CodeService{store: s, log: log}
}

// getUserID extracts the user ID from context
func (s *CodeService) getUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}

// checkProjectAccess verifies the user has access to the project
func (s *CodeService) checkProjectAccess(ctx context.Context, projectID string) bool {
	userID, ok := s.getUserID(ctx)
	if !ok {
		return false
	}
	return s.store.Users().CheckProjectAccess(userID, projectID)
}

// GenerateCodeRequest carries code generation input.
type GenerateCodeRequest struct {
	ProjectID     string
	TaskID        string
	Specification string
	Files         []string
}

// GenerateCode creates a code generation request.
func (s *CodeService) GenerateCode(ctx context.Context, req GenerateCodeRequest) (*model.CodeGenRequest, *Error) {
	if !s.checkProjectAccess(ctx, req.ProjectID) {
		return nil, notFound("Project not found")
	}

	var errs validation.Errors
	validation.NotEmpty(req.ProjectID, "project_id", "Project ID", &errs)
	validation.NotEmpty(req.Specification, "specification", "Specification", &errs)
	if errs.HasErrors() {
		return nil, validationError(errs)
	}

	// Verify project exists
	if _, err := s.store.Projects().GetByID(req.ProjectID); err != nil {
		return nil, notFound("Project not found")
	}

	now := time.Now().UTC()
	codeGen := &model.CodeGenRequest{
		ID:            generateID("code"),
		ProjectID:     req.ProjectID,
		TaskID:        req.TaskID,
		Specification: req.Specification,
		Files:         req.Files,
		Status:        model.CodeGenQueued,
		EstimatedTime: 300,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.store.Code().CreateCodeGen(codeGen); err != nil {
		s.log.Error("failed to create code gen request", zap.Error(err))
		return nil, internalError("Failed to create code generation request")
	}

	return codeGen, nil
}

// GetFile returns a specific file from a project.
func (s *CodeService) GetFile(ctx context.Context, projectID, path string) (*model.ProjectFile, *Error) {
	if !s.checkProjectAccess(ctx, projectID) {
		return nil, notFound("File not found")
	}
	file, err := s.store.Code().GetFile(projectID, path)
	if err != nil {
		return nil, notFound("File not found")
	}
	return file, nil
}

// CreateCommitRequest carries commit creation input.
type CreateCommitRequest struct {
	ProjectID string
	Branch    string
	Message   string
	Files     []model.CommitFile
}

// CreateCommit creates a new commit for a project.
func (s *CodeService) CreateCommit(ctx context.Context, req CreateCommitRequest) (*model.Commit, *Error) {
	if !s.checkProjectAccess(ctx, req.ProjectID) {
		return nil, notFound("Project not found")
	}

	var errs validation.Errors
	validation.NotEmpty(req.ProjectID, "project_id", "Project ID", &errs)
	validation.NotEmpty(req.Message, "message", "Commit message", &errs)
	if errs.HasErrors() {
		return nil, validationError(errs)
	}

	now := time.Now().UTC()
	commit := &model.Commit{
		SHA:       generateID("sha")[4:12], // short SHA
		ProjectID: req.ProjectID,
		Branch:    req.Branch,
		Message:   req.Message,
		Author:    "agent_dev_001",
		Files:     req.Files,
		CreatedAt: now,
	}

	if err := s.store.Code().CreateCommit(commit); err != nil {
		s.log.Error("failed to create commit", zap.Error(err))
		return nil, internalError("Failed to create commit")
	}

	// Save files
	for _, f := range req.Files {
		file := &model.ProjectFile{
			ProjectID:    req.ProjectID,
			Path:         f.Path,
			Content:      f.Content,
			Language:     detectLanguage(f.Path),
			Size:         len(f.Content),
			LastModified: now,
			ModifiedBy:   commit.Author,
		}
		s.store.Code().SaveFile(file)
	}

	return commit, nil
}

func detectLanguage(path string) string {
	p := strings.ToLower(path)
	if strings.HasSuffix(p, ".go") {
		return "go"
	}
	if strings.HasSuffix(p, ".ts") || strings.HasSuffix(p, ".tsx") {
		return "typescript"
	}
	if strings.HasSuffix(p, ".js") || strings.HasSuffix(p, ".jsx") {
		return "javascript"
	}
	if strings.HasSuffix(p, ".py") {
		return "python"
	}
	if strings.HasSuffix(p, ".rs") {
		return "rust"
	}
	if strings.HasSuffix(p, ".md") {
		return "markdown"
	}
	if strings.HasSuffix(p, ".css") {
		return "css"
	}
	if strings.HasSuffix(p, ".html") || strings.HasSuffix(p, ".htm") {
		return "html"
	}
	if strings.HasSuffix(p, ".yml") || strings.HasSuffix(p, ".yaml") {
		return "yaml"
	}
	if strings.HasSuffix(p, ".json") {
		return "json"
	}
	if strings.HasSuffix(p, ".toml") {
		return "toml"
	}
	return "text"
}
