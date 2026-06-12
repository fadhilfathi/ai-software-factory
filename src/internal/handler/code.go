package handler

import (
	"net/http"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/gin-gonic/gin"
)

// CodeHandler handles code generation and file management endpoints.
type CodeHandler struct {
	svc *service.CodeService
}

func NewCodeHandler(svc *service.CodeService) *CodeHandler {
	return &CodeHandler{svc: svc}
}

type generateCodeRequest struct {
	ProjectID     string   `json:"project_id"`
	TaskID        string   `json:"task_id"`
	Specification string   `json:"specification"`
	Files         []string `json:"files"`
}

type generateCodeResponse struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	EstimatedTime int    `json:"estimated_time"`
}

// Generate handles POST /code/generate.
func (h *CodeHandler) Generate(c *gin.Context) {
	var req generateCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	result, svcErr := h.svc.GenerateCode(c.Request.Context(), service.GenerateCodeRequest{
		ProjectID:     req.ProjectID,
		TaskID:        req.TaskID,
		Specification: req.Specification,
		Files:         req.Files,
	})
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	writeJSON(c, http.StatusAccepted, generateCodeResponse{
		ID:            result.ID,
		Status:        string(result.Status),
		EstimatedTime: result.EstimatedTime,
	})
}

type fileResponse struct {
	Path         string `json:"path"`
	Content      string `json:"content"`
	Language     string `json:"language"`
	Size         int    `json:"size"`
	LastModified string `json:"last_modified"`
	ModifiedBy   string `json:"modified_by"`
}

// GetFile handles GET /code/{projectId}/files/{path...}.
func (h *CodeHandler) GetFile(c *gin.Context) {
	projectID := c.Param("projectId")
	filePath := c.Param("path")
	if projectID == "" || filePath == "" {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "projectId and file path are required")
		return
	}

	file, svcErr := h.svc.GetFile(c.Request.Context(), projectID, filePath)
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	writeJSON(c, http.StatusOK, fileResponse{
		Path:         file.Path,
		Content:      file.Content,
		Language:     file.Language,
		Size:         file.Size,
		LastModified: file.LastModified.Format(time.RFC3339),
		ModifiedBy:   file.ModifiedBy,
	})
}

type commitFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type createCommitRequest struct {
	Branch  string       `json:"branch"`
	Message string       `json:"message"`
	Files   []commitFile `json:"files"`
}

type commitResponse struct {
	SHA       string `json:"sha"`
	Message   string `json:"message"`
	Author    string `json:"author"`
	CreatedAt string `json:"created_at"`
}

// CreateCommit handles POST /code/{projectId}/commits.
func (h *CodeHandler) CreateCommit(c *gin.Context) {
	projectID := c.Param("projectId")
	if projectID == "" {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Project ID is required")
		return
	}

	var req createCommitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	modelFiles := make([]model.CommitFile, len(req.Files))
	for i, f := range req.Files {
		modelFiles[i] = model.CommitFile{Path: f.Path, Content: f.Content}
	}

	commit, svcErr := h.svc.CreateCommit(c.Request.Context(), service.CreateCommitRequest{
		ProjectID: projectID,
		Branch:    req.Branch,
		Message:   req.Message,
		Files:     modelFiles,
	})
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	writeJSON(c, http.StatusCreated, commitResponse{
		SHA:       commit.SHA,
		Message:   commit.Message,
		Author:    commit.Author,
		CreatedAt: commit.CreatedAt.Format(time.RFC3339),
	})
}
