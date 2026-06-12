package handler

import (
	"net/http"

	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/gin-gonic/gin"
)

// ReviewHandler handles code review endpoints.
type ReviewHandler struct {
	svc *service.ReviewService
}

func NewReviewHandler(svc *service.ReviewService) *ReviewHandler {
	return &ReviewHandler{svc: svc}
}

type createReviewRequest struct {
	ProjectID     string `json:"project_id"`
	CommitSHA     string `json:"commit_sha"`
	ReviewerType  string `json:"reviewer_type"`
	TargetAgentID string `json:"target_agent_id,omitempty"`
}

type reviewResponse struct {
	ID            string `json:"id"`
	ProjectID     string `json:"project_id"`
	CommitSHA     string `json:"commit_sha"`
	TargetAgentID string `json:"target_agent_id,omitempty"`
	Status        string `json:"status"`
	ReviewerType  string `json:"reviewer_type"`
	ReviewerID    string `json:"reviewer_id,omitempty"`
	Reviewer      string `json:"reviewer,omitempty"`

	Result  string            `json:"result,omitempty"`
	Score   float64           `json:"score,omitempty"`
	Issues  []reviewIssue     `json:"issues,omitempty"`
	Metrics *reviewMetrics    `json:"metrics,omitempty"`
}

type reviewIssue struct {
	Severity   string `json:"severity"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

type reviewMetrics struct {
	Complexity    int     `json:"complexity"`
	MaxComplexity int     `json:"max_complexity"`
	TestCoverage  float64 `json:"test_coverage"`
	Duplications  int     `json:"duplications"`
	LintErrors    int     `json:"lint_errors"`
	SASTFindings  int     `json:"sast_findings"`
}

// Create handles POST /reviews.
func (h *ReviewHandler) Create(c *gin.Context) {
	var req createReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	review, svcErr := h.svc.CreateReview(service.CreateReviewRequest{
		ProjectID:     req.ProjectID,
		CommitSHA:     req.CommitSHA,
		ReviewerType:  req.ReviewerType,
		TargetAgentID: req.TargetAgentID,
	})
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	writeJSON(c, http.StatusCreated, reviewResponse{
		ID:            review.ID.String(),
		ProjectID:     review.ProjectID.String(),
		CommitSHA:     review.CommitSHA,
		TargetAgentID: review.TargetAgentID.String(),
		Status:        string(review.Status),
		ReviewerType:  review.ReviewerType,
		Reviewer:      review.Reviewer,
	})
}

// Get handles GET /reviews/{id}.
func (h *ReviewHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Review ID is required")
		return
	}

	review, svcErr := h.svc.GetReview(id)
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	issues := make([]reviewIssue, len(review.Issues))
	for i, iss := range review.Issues {
		issues[i] = reviewIssue{
			Severity:   iss.Severity,
			File:       iss.File,
			Line:       iss.Line,
			Message:    iss.Message,
			Suggestion: iss.Suggestion,
		}
	}

	var metrics *reviewMetrics
	if review.Metrics != nil {
		metrics = &reviewMetrics{
			Complexity:    review.Metrics.Complexity,
			MaxComplexity: review.Metrics.MaxComplexity,
			TestCoverage:  review.Metrics.TestCoverage,
			Duplications:  review.Metrics.Duplications,
			LintErrors:    review.Metrics.LintErrors,
			SASTFindings:  review.Metrics.SASTFindings,
		}
	}

	writeJSON(c, http.StatusOK, reviewResponse{
		ID:            review.ID.String(),
		ProjectID:     review.ProjectID.String(),
		CommitSHA:     review.CommitSHA,
		TargetAgentID: review.TargetAgentID.String(),
		Status:        string(review.Status),
		ReviewerType:  review.ReviewerType,
		ReviewerID:    review.ReviewerID.String(),
		Reviewer:      review.Reviewer,
		Result:        string(review.Result),
		Score:         review.Score,
		Issues:        issues,
		Metrics:       metrics,
	})
}
