package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestReviewStatusConstants(t *testing.T) {
	assert.Equal(t, ReviewStatus("in_progress"), ReviewInProgress)
	assert.Equal(t, ReviewStatus("completed"), ReviewCompleted)
	assert.Equal(t, ReviewStatus("failed"), ReviewFailed)
}

func TestReviewResultConstants(t *testing.T) {
	assert.Equal(t, ReviewResult("approved"), ReviewApproved)
	assert.Equal(t, ReviewResult("changes_requested"), ReviewChangesReq)
	assert.Equal(t, ReviewResult("rejected"), ReviewRejected)
}

func TestReviewIssueStruct(t *testing.T) {
	issue := ReviewIssue{
		Severity:   "high",
		File:       "handler.go",
		Line:       42,
		Message:    "SQL injection vulnerability",
		Suggestion: "Use parameterized queries",
	}

	assert.Equal(t, "high", issue.Severity)
	assert.Equal(t, "handler.go", issue.File)
	assert.Equal(t, 42, issue.Line)
	assert.Equal(t, "SQL injection vulnerability", issue.Message)
	assert.Equal(t, "Use parameterized queries", issue.Suggestion)
}

func TestReviewIssueWithoutSuggestion(t *testing.T) {
	issue := ReviewIssue{
		Severity: "medium",
		File:     "service.go",
		Line:     10,
		Message:  "Missing error handling",
	}
	assert.Empty(t, issue.Suggestion)
}

func TestReviewMetricsStruct(t *testing.T) {
	metrics := ReviewMetrics{
		Complexity:   "moderate",
		TestCoverage: 85.5,
		Duplications: 3,
	}
	assert.Equal(t, "moderate", metrics.Complexity)
	assert.Equal(t, 85.5, metrics.TestCoverage)
	assert.Equal(t, 3, metrics.Duplications)
}

func TestReviewStruct(t *testing.T) {
	now := time.Now().UTC()
	review := Review{
		ID:           uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		ProjectID:    uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		CommitSHA:    "abc123def",
		ReviewerType: "ai",
		Reviewer:     "reviewer-agent-1",
		Status:       ReviewCompleted,
		Result:       ReviewApproved,
		Score:        92.5,
		Issues: []ReviewIssue{
			{Severity: "low", File: "main.go", Line: 5, Message: "Unused import"},
			{Severity: "medium", File: "handler.go", Line: 20, Message: "Missing validation"},
		},
		Metrics: &ReviewMetrics{
			Complexity:   "low",
			TestCoverage: 90.0,
			Duplications: 0,
		},
		CreatedAt: now.Add(-time.Hour),
		UpdatedAt: now,
	}

	assert.Equal(t, uuid.MustParse("11111111-1111-1111-1111-111111111111"), review.ID)
	assert.Equal(t, uuid.MustParse("22222222-2222-2222-2222-222222222222"), review.ProjectID)
	assert.Equal(t, "abc123def", review.CommitSHA)
	assert.Equal(t, "ai", review.ReviewerType)
	assert.Equal(t, "reviewer-agent-1", review.Reviewer)
	assert.Equal(t, ReviewCompleted, review.Status)
	assert.Equal(t, ReviewApproved, review.Result)
	assert.Equal(t, 92.5, review.Score)
	assert.Len(t, review.Issues, 2)
	assert.Equal(t, "low", review.Issues[0].Severity)
	assert.NotNil(t, review.Metrics)
	assert.Equal(t, "low", review.Metrics.Complexity)
	assert.Equal(t, 90.0, review.Metrics.TestCoverage)
	assert.Equal(t, now.Add(-time.Hour), review.CreatedAt)
	assert.Equal(t, now, review.UpdatedAt)
}

func TestReviewWithNilMetrics(t *testing.T) {
	review := Review{
		ID:        uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		ProjectID: uuid.MustParse("44444444-4444-4444-4444-444444444444"),
		CommitSHA: "sha1",
		Status:    ReviewInProgress,
	}
	assert.Nil(t, review.Metrics)
	assert.Nil(t, review.Issues)
	assert.Zero(t, review.Score)
}