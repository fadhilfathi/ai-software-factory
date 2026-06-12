package postgres

import (
	"context"
	"fmt"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresDeliverableStore struct {
	s *postgresStore
}

func (s *postgresDeliverableStore) pool() *pgxpool.Pool {
	return s.s.pool
}

func (s *postgresDeliverableStore) Create(d *model.Deliverable) error {
	query := `INSERT INTO deliverables (task_id, agent_id, title, content, version, created_at)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	err := s.pool().QueryRow(context.Background(), query,
		d.TaskID, d.AgentID, d.Title, d.Content, d.Version, d.CreatedAt,
	).Scan(&d.ID)
	if err != nil {
		return fmt.Errorf("create deliverable: %w", err)
	}
	return nil
}

func (s *postgresDeliverableStore) GetByID(id uuid.UUID) (*model.Deliverable, error) {
	query := `SELECT id, task_id, agent_id, title, content, version, created_at
		FROM deliverables WHERE id = $1`
	d := &model.Deliverable{}
	err := s.pool().QueryRow(context.Background(), query, id).Scan(
		&d.ID, &d.TaskID, &d.AgentID, &d.Title, &d.Content, &d.Version, &d.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get deliverable: %w", err)
	}
	return d, nil
}

func (s *postgresDeliverableStore) Update(d *model.Deliverable) error {
	query := `UPDATE deliverables SET title=$1, content=$2, version=$3 WHERE id=$4`
	tag, err := s.pool().Exec(context.Background(), query,
		d.Title, d.Content, d.Version, d.ID)
	if err != nil {
		return fmt.Errorf("update deliverable: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *postgresDeliverableStore) ListByTask(taskID uuid.UUID) ([]*model.Deliverable, error) {
	query := `SELECT id, task_id, agent_id, title, content, version, created_at
		FROM deliverables WHERE task_id = $1 ORDER BY created_at DESC`
	rows, err := s.pool().Query(context.Background(), query, taskID)
	if err != nil {
		return nil, fmt.Errorf("list deliverables by task: %w", err)
	}
	defer rows.Close()

	var deliverables []*model.Deliverable
	for rows.Next() {
		d := &model.Deliverable{}
		if err := rows.Scan(&d.ID, &d.TaskID, &d.AgentID, &d.Title, &d.Content, &d.Version, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan deliverable: %w", err)
		}
		deliverables = append(deliverables, d)
	}
	return deliverables, nil
}

func (s *postgresDeliverableStore) ListByAgent(agentID uuid.UUID) ([]*model.Deliverable, error) {
	query := `SELECT id, task_id, agent_id, title, content, version, created_at
		FROM deliverables WHERE agent_id = $1 ORDER BY created_at DESC`
	rows, err := s.pool().Query(context.Background(), query, agentID)
	if err != nil {
		return nil, fmt.Errorf("list deliverables by agent: %w", err)
	}
	defer rows.Close()

	var deliverables []*model.Deliverable
	for rows.Next() {
		d := &model.Deliverable{}
		if err := rows.Scan(&d.ID, &d.TaskID, &d.AgentID, &d.Title, &d.Content, &d.Version, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan deliverable: %w", err)
		}
		deliverables = append(deliverables, d)
	}
	return deliverables, nil
}
