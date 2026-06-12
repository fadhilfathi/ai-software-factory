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

type postgresTaskStore struct {
	s *postgresStore
}

func (s *postgresTaskStore) pool() *pgxpool.Pool {
	return s.s.pool
}

func (s *postgresTaskStore) Create(t *model.Task) error {
	query := `INSERT INTO tasks (id, project_id, title, description, status, priority, assignee_id, position, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := s.pool().Exec(context.Background(), query,
		t.ID, t.ProjectID, t.Title, t.Description, t.Status, t.Priority, t.AssigneeID, t.Position, t.CreatedAt, t.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	return nil
}

func (s *postgresTaskStore) GetByID(id uuid.UUID) (*model.Task, error) {
	query := `SELECT id, project_id, title, description, status, priority, assignee_id, position, created_at, updated_at
		FROM tasks WHERE id = $1`
	t := &model.Task{}
	err := s.pool().QueryRow(context.Background(), query, id).Scan(
		&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.AssigneeID, &t.Position, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	return t, nil
}

func (s *postgresTaskStore) List(filter store.TaskFilter) ([]*model.Task, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.ProjectID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("project_id = $%d", argIdx))
		args = append(args, filter.ProjectID)
		argIdx++
	}
	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}
	if filter.AssigneeID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("assignee_id = $%d", argIdx))
		args = append(args, filter.AssigneeID)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tasks %s", whereClause)
	var total int
	err := s.pool().QueryRow(context.Background(), countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count tasks: %w", err)
	}

	page, limit := filter.Page, filter.Limit
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	offset := (page - 1) * limit
	dataQuery := fmt.Sprintf(`SELECT id, project_id, title, description, status, priority, assignee_id, position, created_at, updated_at
		FROM tasks %s ORDER BY position ASC, created_at DESC LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := s.pool().Query(context.Background(), dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*model.Task
	for rows.Next() {
		t := &model.Task{}
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.AssigneeID, &t.Position, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, t)
	}

	return tasks, total, nil
}

func (s *postgresTaskStore) Update(t *model.Task) error {
	query := `UPDATE tasks SET title=$1, description=$2, status=$3, priority=$4, assignee_id=$5, position=$6, updated_at=NOW()
		WHERE id=$7`
	tag, err := s.pool().Exec(context.Background(), query,
		t.Title, t.Description, t.Status, t.Priority, t.AssigneeID, t.Position, t.ID)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *postgresTaskStore) Delete(id uuid.UUID) error {
	tag, err := s.pool().Exec(context.Background(), "DELETE FROM tasks WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}
