package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
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

	if review.ID == uuid.Nil {
		review.ID = uuid.New()
	}

	query := `
		INSERT INTO reviews (id, project_id, target_agent_id, reviewer_id, agent_id, commit_sha, status, result, score, metrics, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	metricsJSON, _ := json.Marshal(review.Metrics)
	_, err = tx.Exec(ctx, query,
		review.ID, review.ProjectID, review.TargetAgentID, review.ReviewerID, review.AgentID, review.CommitSHA,
		review.Status, review.Result, review.Score, metricsJSON, review.CreatedAt, review.UpdatedAt,
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
	ctx := context.Background()
	row := st.s.pool.QueryRow(ctx, `
		SELECT id, project_id, target_agent_id, reviewer_id, agent_id, commit_sha, status, result, score, metrics, created_at, updated_at
		FROM reviews WHERE id = $1`, id)
	return scanReview(row)
}

func scanReview(row pgx.Row) (*model.Review, error) {
	var r model.Review
	var metricsJSON []byte
	err := row.Scan(&r.ID, &r.ProjectID, &r.TargetAgentID, &r.ReviewerID, &r.AgentID, &r.CommitSHA, &r.Status, &r.Result, &r.Score, &metricsJSON, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if len(metricsJSON) > 0 {
		_ = json.Unmarshal(metricsJSON, &r.Metrics)
	}
	return &r, nil
}

func (st *postgresReviewStore) ListByProject(projectID uuid.UUID) ([]*model.Review, error) {
	ctx := context.Background()
	rows, err := st.s.pool.Query(ctx, `
		SELECT id, project_id, target_agent_id, reviewer_id, agent_id, commit_sha, status, result, score, metrics, created_at, updated_at
		FROM reviews WHERE project_id = $1 ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []*model.Review
	for rows.Next() {
		var r model.Review
		var metricsJSON []byte
		err := rows.Scan(&r.ID, &r.ProjectID, &r.TargetAgentID, &r.ReviewerID, &r.AgentID, &r.CommitSHA, &r.Status, &r.Result, &r.Score, &metricsJSON, &r.CreatedAt, &r.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if len(metricsJSON) > 0 {
			_ = json.Unmarshal(metricsJSON, &r.Metrics)
		}
		reviews = append(reviews, &r)
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

func (st *postgresReviewStore) CreateComment(c *model.ReviewComment) error {
	ctx := context.Background()
	tx, err := st.s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO review_comments (id, review_id, file, line, author_id, content, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err = tx.Exec(ctx, query, c.ID, c.ReviewID, c.File, c.Line, c.AuthorID, c.Content, c.CreatedAt)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
