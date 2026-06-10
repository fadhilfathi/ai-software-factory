package service

import (
	"time"

	"github.com/example/project/internal/model"
	"github.com/example/project/internal/store"
	"github.com/example/project/internal/validation"
	"go.uber.org/zap"
)

// ReviewService handles code review operations.
type ReviewService struct {
	store store.Store
	log   *zap.Logger
}

func NewReviewService(s store.Store, log *zap.Logger) *ReviewService {
	return &ReviewService{store: s, log: log}
}

// CreateReviewRequest carries review creation input.
type CreateReviewRequest struct {
	ProjectID    string
	CommitSHA    string
	ReviewerType string
}

// CreateReview creates a code review request.
func (s *ReviewService) CreateReview(req CreateReviewRequest) (*model.Review, *Error) {
	var errs validation.Errors
	validation.NotEmpty(req.ProjectID, "project_id", "Project ID", &errs)
	validation.NotEmpty(req.CommitSHA, "commit_sha", "Commit SHA", &errs)
	if errs.HasErrors() {
		return nil, validationError(errs)
	}

	// Verify project exists
	if _, err := s.store.Projects().GetByID(req.ProjectID); err != nil {
		return nil, notFound("Project not found")
	}

	now := time.Now().UTC()
	review := &model.Review{
		ID:           generateID("review"),
		ProjectID:    req.ProjectID,
		CommitSHA:    req.CommitSHA,
		ReviewerType: req.ReviewerType,
		Reviewer:     "review_agent_001",
		Status:       model.ReviewInProgress,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.store.Reviews().Create(review); err != nil {
		s.log.Error("failed to create review", zap.Error(err))
		return nil, internalError("Failed to create review")
	}

	return review, nil
}

// GetReview returns a review by ID.
func (s *ReviewService) GetReview(id string) (*model.Review, *Error) {
	// var errs validation.Errors
	// validation.NotEmpty(id, "id", "Review ID", &errs) // not needed, handler validates
	review, err := s.store.Reviews().GetByID(id)
	if err != nil {
		return nil, notFound("Review not found")
	}
	return review, nil
}
