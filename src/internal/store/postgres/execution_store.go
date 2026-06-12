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

type postgresExecutionStore struct {
	s *postgresStore
}

func (s *postgresExecutionStore) pool() *pgxpool.Pool {
	return s.s.pool
}

func (s *postgresExecutionStore) Create(e *model.Execution) error {
	query := `INSERT INTO executions (task_id, agent_id, status, started_at, completed_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	err := s.pool().QueryRow(context.Background(), query,
		e.TaskID, e.AgentID, e.Status, e.StartedAt, e.CompletedAt, e.CreatedAt,
	).Scan(&e.ExecutionID)
	if err != nil {
		return fmt.Errorf("create execution: %w", err)
	}
	return nil
}

func (s *postgresExecutionStore) GetByID(id uuid.UUID) (*model.Execution, error) {
	query := `SELECT id, task_id, agent_id, status, started_at, completed_at, created_at
		FROM executions WHERE id = $1`
	e := &model.Execution{}
	err := s.pool().QueryRow(context.Background(), query, id).Scan(
		&e.ExecutionID, &e.TaskID, &e.AgentID, &e.Status, &e.StartedAt, &e.CompletedAt, &e.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get execution: %w", err)
	}
	return e, nil
}

func (s *postgresExecutionStore) List(filter store.ExecutionFilter) ([]*model.Execution, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.TaskID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("task_id = $%d", argIdx))
		args = append(args, filter.TaskID)
		argIdx++
	}
	if filter.AgentID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("agent_id = $%d", argIdx))
		args = append(args, filter.AgentID)
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

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM executions %s", whereClause)
	var total int
	err := s.pool().QueryRow(context.Background(), countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count executions: %w", err)
	}

	page, limit := filter.Page, filter.Limit
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	offset := (page - 1) * limit
	dataQuery := fmt.Sprintf(`SELECT id, task_id, agent_id, status, started_at, completed_at, created_at
		FROM executions %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := s.pool().Query(context.Background(), dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list executions: %w", err)
	}
	defer rows.Close()

	var executions []*model.Execution
	for rows.Next() {
		e := &model.Execution{}
		if err := rows.Scan(&e.ExecutionID, &e.TaskID, &e.AgentID, &e.Status, &e.StartedAt, &e.CompletedAt, &e.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan execution: %w", err)
		}
		executions = append(executions, e)
	}

	return executions, total, nil
}

func (s *postgresExecutionStore) Update(e *model.Execution) error {
	query := `UPDATE executions SET status=$1, started_at=$2, completed_at=$3 WHERE id=$4`
	tag, err := s.pool().Exec(context.Background(), query,
		e.Status, e.StartedAt, e.CompletedAt, e.ExecutionID)
	if err != nil {
		return fmt.Errorf("update execution: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}
