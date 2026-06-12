package model

import (
	"time"

	"github.com/google/uuid"
)

// ReviewStatus represents the lifecycle state of a code review.
type ReviewStatus string

const (
	ReviewInProgress ReviewStatus = "in_progress"
	ReviewCompleted  ReviewStatus = "completed"
	ReviewFailed     ReviewStatus = "failed"
)

// ReviewResult represents the outcome of a completed review.
type ReviewResult string

const (
	ReviewApproved   ReviewResult = "approved"
	ReviewChangesReq ReviewResult = "changes_requested"
	ReviewRejected   ReviewResult = "rejected"
)

// ReviewIssue represents a single finding from a code review.
type ReviewIssue struct {
	ID         uuid.UUID `json:"id"`
	ReviewID   uuid.UUID `json:"review_id"`
	Severity   string    `json:"severity"`
	File       string    `json:"file"`
	Line       int       `json:"line"`
	Message    string    `json:"message"`
	Suggestion string    `json:"suggestion,omitempty"`
}

// ReviewMetrics holds aggregate quality metrics from a review.
type ReviewMetrics struct {
	Complexity    int     `json:"complexity"` // Average complexity
	MaxComplexity int     `json:"max_complexity"`
	TestCoverage  float64 `json:"test_coverage"`
	Duplications  int     `json:"duplications"`
	LintErrors    int     `json:"lint_errors"`
	SASTFindings  int     `json:"sast_findings"`
}

// Review represents a code review request and its results.
type Review struct {
	ID            uuid.UUID      `json:"id"`
	ProjectID     uuid.UUID      `json:"project_id"`
	CommitSHA     string         `json:"commit_sha"`
	TargetAgentID uuid.UUID      `json:"target_agent_id,omitempty"`
	ReviewerType  string         `json:"reviewer_type"`
	ReviewerID    *uuid.UUID     `json:"reviewer_id,omitempty"`
	AgentID       *uuid.UUID     `json:"agent_id,omitempty"`
	Reviewer      string         `json:"reviewer,omitempty"` // Legacy field for display name
	Status        ReviewStatus   `json:"status"`
	Result        ReviewResult   `json:"result,omitempty"`
	Score         float64        `json:"score,omitempty"`
	Issues        []ReviewIssue  `json:"issues,omitempty"`
	Metrics       *ReviewMetrics `json:"metrics,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// ReviewComment represents a single comment on a code review.
type ReviewComment struct {
	ID        uuid.UUID `json:"id"`
	ReviewID  uuid.UUID `json:"review_id"`
	File      string    `json:"file"`
	Line      int       `json:"line"`
	AuthorID  uuid.UUID `json:"author_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
