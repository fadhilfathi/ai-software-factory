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

type postgresProjectStore struct {
	s *postgresStore
}

func (s *postgresProjectStore) pool() *pgxpool.Pool {
	return s.s.pool
}

func (s *postgresProjectStore) Create(p *model.Project) error {
	query := `INSERT INTO projects (id, name, description, owner_id, status, template, progress, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := s.pool().Exec(context.Background(), query,
		p.ID, p.Name, p.Description, p.OwnerID, p.Status, p.Template, p.Progress, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	return nil
}

func (s *postgresProjectStore) GetByID(id uuid.UUID) (*model.Project, error) {
	query := `SELECT id, name, description, owner_id, status, template, progress, created_at, updated_at
		FROM projects WHERE id = $1`
	p := &model.Project{}
	err := s.pool().QueryRow(context.Background(), query, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.OwnerID, &p.Status, &p.Template, &p.Progress, &p.CreatedAt, &p.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	return p, nil
}

func (s *postgresProjectStore) List(filter store.ProjectFilter) ([]*model.Project, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}
	if filter.OwnerID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("owner_id = $%d", argIdx))
		args = append(args, filter.OwnerID)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM projects %s", whereClause)
	var total int
	err := s.pool().QueryRow(context.Background(), countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count projects: %w", err)
	}

	page, limit := filter.Page, filter.Limit
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	offset := (page - 1) * limit
	dataQuery := fmt.Sprintf(`SELECT id, name, description, owner_id, status, template, progress, created_at, updated_at
		FROM projects %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := s.pool().Query(context.Background(), dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var projects []*model.Project
	for rows.Next() {
		p := &model.Project{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.OwnerID, &p.Status, &p.Template, &p.Progress, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan project: %w", err)
		}
		projects = append(projects, p)
	}

	return projects, total, nil
}

func (s *postgresProjectStore) Update(p *model.Project) error {
	query := `UPDATE projects SET name=$1, description=$2, owner_id=$3, status=$4, template=$5, progress=$6, updated_at=NOW()
		WHERE id=$7`
	tag, err := s.pool().Exec(context.Background(), query,
		p.Name, p.Description, p.OwnerID, p.Status, p.Template, p.Progress, p.ID)
	if err != nil {
		return fmt.Errorf("update project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *postgresProjectStore) Delete(id uuid.UUID) error {
	tag, err := s.pool().Exec(context.Background(), "DELETE FROM projects WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}
