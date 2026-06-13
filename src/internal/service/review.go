package service

import (
	"context"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/fadhilfathi/AI-Software-Factory/internal/validation"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ReviewService handles code review operations.
type ReviewService struct {
	store        store.Store
	orchestrator AgentOrchestrator
	log          *zap.Logger
	gates        []GateRunner
}

func NewReviewService(s store.Store, orch AgentOrchestrator, log *zap.Logger) *ReviewService {
	svc := &ReviewService{
		store:        s,
		orchestrator: orch,
		log:          log,
	}
	svc.gates = []GateRunner{
		&LinterGate{orchestrator: orch},
		&SASTGate{orchestrator: orch},
		&ComplexityGate{orchestrator: orch},
	}
	return svc
}

// CreateReviewRequest carries review creation input.
type CreateReviewRequest struct {
	ProjectID     string
	CommitSHA     string
	ReviewerType  string
	TargetAgentID string
}

// CreateReview creates a code review request.
func (s *ReviewService) CreateReview(req CreateReviewRequest) (*model.Review, *Error) {
	errs := &validation.Errors{}
	validation.NotEmpty(req.ProjectID, "project_id", "Project ID", errs)
	validation.NotEmpty(req.CommitSHA, "commit_sha", "Commit SHA", errs)
	if errs.HasErrors() {
		return nil, validationError(*errs)
	}

	pID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		errs.Add("project_id", "Invalid Project ID format")
		return nil, validationError(errs)
	}

	var targetAgentID uuid.UUID
	if req.TargetAgentID != "" {
		targetAgentID, err = uuid.Parse(req.TargetAgentID)
		if err != nil {
			errs.Add("target_agent_id", "Invalid Agent ID format")
			return nil, validationError(errs)
		}
	}

	// Verify project exists
	if _, err := s.store.Projects().GetByID(pID); err != nil {
		return nil, notFound("Project not found")
	}

	now := time.Now().UTC()
	review := &model.Review{
		ID:            uuid.New(),
		ProjectID:     pID,
		CommitSHA:     req.CommitSHA,
		TargetAgentID: targetAgentID,
		ReviewerType:  req.ReviewerType,
		Reviewer:      "review_agent_001",
		Status:        model.ReviewInProgress,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.store.Reviews().Create(review); err != nil {
		s.log.Error("failed to create review", zap.Error(err))
		return nil, internalError("Failed to create review")
	}

	// Trigger async processing
	go s.processReview(context.Background(), review)

	return review, nil
}

// processReview runs the quality and security gates for a review.
func (s *ReviewService) processReview(ctx context.Context, review *model.Review) {
	s.log.Info("processing review", zap.String("review_id", review.ID.String()))

	metrics := &model.ReviewMetrics{}
	var allIssues []model.ReviewIssue
	var totalScore float64

	for _, gate := range s.gates {
		res, err := gate.Run(ctx, review.ProjectID, review.CommitSHA)
		if err != nil {
			s.log.Error("gate failed", zap.String("gate", gate.Name()), zap.Error(err))
			continue
		}

		allIssues = append(allIssues, res.Issues...)
		totalScore += res.Score

		// Map metrics from gate results
		switch gate.Name() {
		case "Linter":
			metrics.LintErrors = len(res.Issues)
		case "SAST":
			metrics.SASTFindings = len(res.Issues)
		case "Complexity":
			metrics.Complexity = int(res.Score)
		}
	}

	review.Metrics = metrics
	review.Issues = allIssues
	review.Score = totalScore / float64(len(s.gates))
	review.Status = model.ReviewCompleted
	review.UpdatedAt = time.Now().UTC()

	// Simple result logic for now
	review.Result = model.ReviewApproved
	for _, iss := range allIssues {
		if iss.Severity == "high" || iss.Severity == "critical" {
			review.Result = model.ReviewChangesReq
			break
		}
	}

	if err := s.store.Reviews().Update(review); err != nil {
		s.log.Error("failed to update review status", zap.Error(err))
	}
}

// GetReview returns a review by ID.
func (s *ReviewService) GetReview(id string) (*model.Review, *Error) {
	uID, err := uuid.Parse(id)
	if err != nil {
		return nil, validationError(validation.Errors{"id": "Invalid Review ID format"})
	}
	review, err := s.store.Reviews().GetByID(uID)
	if err != nil {
		return nil, notFound("Review not found")
	}
	return review, nil
}
