package postgres

import (
	"context"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type postgresReviewStore struct {
	s *postgresStore
}

func (st *postgresReviewStore) Create(review *model.Review) error {
	ctx := context.Background()
	tx, err := st.s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	queryReview := `
		INSERT INTO reviews (
			id, project_id, commit_sha, target_agent_id, reviewer_type, reviewer_id, agent_id, status, result, score, metrics, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err = tx.Exec(ctx, queryReview,
		review.ID, review.ProjectID, review.CommitSHA, review.TargetAgentID, review.ReviewerType, review.ReviewerID, review.AgentID, review.Status, review.Result, review.Score, review.Metrics, review.CreatedAt, review.UpdatedAt,
	)
	if err != nil {
		return err
	}

	queryIssue := `
		INSERT INTO review_issues (id, review_id, severity, file, line, message, suggestion, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`

	for _, issue := range review.Issues {
		if issue.ID == uuid.Nil {
			issue.ID = uuid.New()
		}
		_, err = tx.Exec(ctx, queryIssue, issue.ID, review.ID, issue.Severity, issue.File, issue.Line, issue.Message, issue.Suggestion)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (st *postgresReviewStore) GetByID(id uuid.UUID) (*model.Review, error) {
	query := `
		SELECT id, project_id, commit_sha, target_agent_id, reviewer_type, reviewer_id, agent_id, status, result, score, metrics, created_at, updated_at
		FROM reviews
		WHERE id = $1`

	var review model.Review
	err := st.s.pool.QueryRow(context.Background(), query, id).Scan(
		&review.ID, &review.ProjectID, &review.CommitSHA, &review.TargetAgentID, &review.ReviewerType, &review.ReviewerID, &review.AgentID, &review.Status, &review.Result, &review.Score, &review.Metrics, &review.CreatedAt, &review.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// Fetch issues
	issuesQuery := `SELECT id, review_id, severity, file, line, message, suggestion FROM review_issues WHERE review_id = $1`
	rows, err := st.s.pool.Query(context.Background(), issuesQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var issue model.ReviewIssue
		if err := rows.Scan(&issue.ID, &issue.ReviewID, &issue.Severity, &issue.File, &issue.Line, &issue.Message, &issue.Suggestion); err != nil {
			return nil, err
		}
		review.Issues = append(review.Issues, issue)
	}

	return &review, nil
}

func (st *postgresReviewStore) ListByProject(projectID uuid.UUID) ([]*model.Review, error) {
	query := `
		SELECT id, project_id, commit_sha, target_agent_id, reviewer_type, reviewer_id, agent_id, status, result, score, metrics, created_at, updated_at
		FROM reviews
		WHERE project_id = $1
		ORDER BY created_at DESC`

	rows, err := st.s.pool.Query(context.Background(), query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []*model.Review
	for rows.Next() {
		var review model.Review
		err := rows.Scan(
			&review.ID, &review.ProjectID, &review.CommitSHA, &review.TargetAgentID, &review.ReviewerType, &review.ReviewerID, &review.AgentID, &review.Status, &review.Result, &review.Score, &review.Metrics, &review.CreatedAt, &review.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		reviews = append(reviews, &review)
	}
	return reviews, nil
}

func (st *postgresReviewStore) Update(review *model.Review) error {
	ctx := context.Background()
	tx, err := st.s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	queryReview := `
		UPDATE reviews
		SET status = $2, result = $3, score = $4, metrics = $5, updated_at = $6, target_agent_id = $7, reviewer_id = $8, agent_id = $9
		WHERE id = $1`

	_, err = tx.Exec(ctx, queryReview,
		review.ID, review.Status, review.Result, review.Score, review.Metrics, review.UpdatedAt, review.TargetAgentID, review.ReviewerID, review.AgentID,
	)
	if err != nil {
		return err
	}

	// For simplicity, we delete and re-insert issues on update if they changed.
	_, err = tx.Exec(ctx, "DELETE FROM review_issues WHERE review_id = $1", review.ID)
	if err != nil {
		return err
	}

	queryIssue := `
		INSERT INTO review_issues (id, review_id, severity, file, line, message, suggestion, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`

	for _, issue := range review.Issues {
		if issue.ID == uuid.Nil {
			issue.ID = uuid.New()
		}
		_, err = tx.Exec(ctx, queryIssue, issue.ID, review.ID, issue.Severity, issue.File, issue.Line, issue.Message, issue.Suggestion)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
