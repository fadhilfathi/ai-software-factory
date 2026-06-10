package handler

import (
	"encoding/json"
	"net/http"

	"github.com/example/project/internal/service"
)

// ReviewHandler handles code review endpoints.
type ReviewHandler struct {
	svc *service.ReviewService
}

func NewReviewHandler(svc *service.ReviewService) *ReviewHandler {
	return &ReviewHandler{svc: svc}
}

type createReviewRequest struct {
	ProjectID    string `json:"project_id"`
	CommitSHA    string `json:"commit_sha"`
	ReviewerType string `json:"reviewer_type"`
}

type reviewResponse struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Reviewer string `json:"reviewer,omitempty"`

	Result   string            `json:"result,omitempty"`
	Score    float64           `json:"score,omitempty"`
	Issues   []reviewIssue     `json:"issues,omitempty"`
	Metrics  *reviewMetrics    `json:"metrics,omitempty"`
}

type reviewIssue struct {
	Severity   string `json:"severity"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

type reviewMetrics struct {
	Complexity    string  `json:"complexity"`
	TestCoverage  float64 `json:"test_coverage"`
	Duplications  int     `json:"duplications"`
}

// Create handles POST /reviews.
func (h *ReviewHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	review, svcErr := h.svc.CreateReview(service.CreateReviewRequest{
		ProjectID:    req.ProjectID,
		CommitSHA:    req.CommitSHA,
		ReviewerType: req.ReviewerType,
	})
	if svcErr != nil {
		writeServiceError(w, svcErr)
		return
	}

	writeJSON(w, http.StatusCreated, reviewResponse{
		ID:       review.ID,
		Status:   string(review.Status),
		Reviewer: review.Reviewer,
	})
}

// Get handles GET /reviews/{id}.
func (h *ReviewHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Review ID is required")
		return
	}

	review, svcErr := h.svc.GetReview(id)
	if svcErr != nil {
		writeServiceError(w, svcErr)
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
			TestCoverage:  review.Metrics.TestCoverage,
			Duplications: review.Metrics.Duplications,
		}
	}

	writeJSON(w, http.StatusOK, reviewResponse{
		ID:      review.ID,
		Status:  string(review.Status),
		Result:  string(review.Result),
		Score:   review.Score,
		Issues:  issues,
		Metrics: metrics,
	})
}
