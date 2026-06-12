package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresAgentRunStore struct {
	s *postgresStore
}

func (s *postgresAgentRunStore) pool() *pgxpool.Pool {
	return s.s.pool
}

func (s *postgresAgentRunStore) Create(r *model.AgentRun) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	query := `INSERT INTO agent_runs (id, agent_id, task_id, status, input, output, started_at, completed_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := s.pool().Exec(context.Background(), query,
		r.ID, r.AgentID, r.TaskID, r.Status, r.Input, r.Output, r.StartedAt, r.CompletedAt, r.CreatedAt)
	if err != nil {
		return fmt.Errorf("create agent run: %w", err)
	}
	return nil
}

func (s *postgresAgentRunStore) GetByID(id uuid.UUID) (*model.AgentRun, error) {
	query := `SELECT id, agent_id, task_id, status, input, output, started_at, completed_at, created_at
		FROM agent_runs WHERE id = $1`
	r := &model.AgentRun{}
	err := s.pool().QueryRow(context.Background(), query, id).Scan(
		&r.ID, &r.AgentID, &r.TaskID, &r.Status, &r.Input, &r.Output, &r.StartedAt, &r.CompletedAt, &r.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get agent run: %w", err)
	}
	return r, nil
}

func (s *postgresAgentRunStore) List(filter store.AgentRunFilter) ([]*model.AgentRun, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.AgentID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("agent_id = $%d", argIdx))
		args = append(args, filter.AgentID)
		argIdx++
	}
	if filter.TaskID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("task_id = $%d", argIdx))
		args = append(args, filter.TaskID)
		argIdx++
	}
	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM agent_runs %s", whereClause)
	var total int
	err := s.pool().QueryRow(context.Background(), countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count agent runs: %w", err)
	}

	page, limit := filter.Page, filter.Limit
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	offset := (page - 1) * limit
	dataQuery := fmt.Sprintf(`SELECT id, agent_id, task_id, status, input, output, started_at, completed_at, created_at
		FROM agent_runs %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := s.pool().Query(context.Background(), dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list agent runs: %w", err)
	}
	defer rows.Close()

	var runs []*model.AgentRun
	for rows.Next() {
		r := &model.AgentRun{}
		if err := rows.Scan(&r.ID, &r.AgentID, &r.TaskID, &r.Status, &r.Input, &r.Output, &r.StartedAt, &r.CompletedAt, &r.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan agent run: %w", err)
		}
		runs = append(runs, r)
	}

	return runs, total, nil
}

func (s *postgresAgentRunStore) Update(r *model.AgentRun) error {
	query := `UPDATE agent_runs SET status=$1, input=$2, output=$3, started_at=$4, completed_at=$5
		WHERE id=$6`
	tag, err := s.pool().Exec(context.Background(), query,
		r.Status, r.Input, r.Output, r.StartedAt, r.CompletedAt, r.ID)
	if err != nil {
		return fmt.Errorf("update agent run: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}
