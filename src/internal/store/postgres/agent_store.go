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

type postgresAgentStore struct {
	s *postgresStore
}

func (s *postgresAgentStore) pool() *pgxpool.Pool {
	return s.s.pool
}

func (s *postgresAgentStore) Create(a *model.Agent) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	query := `INSERT INTO agents (id, project_id, name, agent_type, role, model, provider, capabilities, status, config, current_task_id, tasks_done, uptime, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`
	_, err := s.pool().Exec(context.Background(), query,
		a.ID, a.ProjectID, a.Name, a.Type, a.Role, a.Model, a.Provider, a.Capabilities, a.Status, a.Config, a.CurrentTaskID, a.TasksDone, a.Uptime, a.CreatedAt, a.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create agent: %w", err)
	}
	return nil
}

func (s *postgresAgentStore) GetByID(id uuid.UUID) (*model.Agent, error) {
	query := `SELECT id, project_id, name, agent_type, role, model, provider, capabilities, status, config, current_task_id, tasks_done, uptime, created_at, updated_at
		FROM agents WHERE id = $1`
	a := &model.Agent{}
	err := s.pool().QueryRow(context.Background(), query, id).Scan(
		&a.ID, &a.ProjectID, &a.Name, &a.Type, &a.Role, &a.Model, &a.Provider, &a.Capabilities, &a.Status, &a.Config, &a.CurrentTaskID, &a.TasksDone, &a.Uptime, &a.CreatedAt, &a.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get agent: %w", err)
	}
	return a, nil
}

func (s *postgresAgentStore) List(filter store.AgentFilter) ([]*model.Agent, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.ProjectID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("project_id = $%d", argIdx))
		args = append(args, filter.ProjectID)
		argIdx++
	}
	if filter.Role != "" {
		conditions = append(conditions, fmt.Sprintf("role = $%d", argIdx))
		args = append(args, filter.Role)
		argIdx++
	}
	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}
	if filter.Type != "" {
		conditions = append(conditions, fmt.Sprintf("agent_type = $%d", argIdx))
		args = append(args, filter.Type)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM agents %s", whereClause)
	var total int
	err := s.pool().QueryRow(context.Background(), countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count agents: %w", err)
	}

	page, limit := filter.Page, filter.Limit
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	offset := (page - 1) * limit
	dataQuery := fmt.Sprintf(`SELECT id, project_id, name, agent_type, role, model, provider, capabilities, status, config, current_task_id, tasks_done, uptime, created_at, updated_at
		FROM agents %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := s.pool().Query(context.Background(), dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()

	var agents []*model.Agent
	for rows.Next() {
		a := &model.Agent{}
		if err := rows.Scan(&a.ID, &a.ProjectID, &a.Name, &a.Type, &a.Role, &a.Model, &a.Provider, &a.Capabilities, &a.Status, &a.Config, &a.CurrentTaskID, &a.TasksDone, &a.Uptime, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan agent: %w", err)
		}
		agents = append(agents, a)
	}

	return agents, total, nil
}

func (s *postgresAgentStore) Update(a *model.Agent) error {
	query := `UPDATE agents SET project_id=$1, name=$2, agent_type=$3, role=$4, model=$5, provider=$6, capabilities=$7, status=$8, config=$9, current_task_id=$10, tasks_done=$11, uptime=$12, updated_at=NOW()
		WHERE id=$13`
	tag, err := s.pool().Exec(context.Background(), query,
		a.ProjectID, a.Name, a.Type, a.Role, a.Model, a.Provider, a.Capabilities, a.Status, a.Config, a.CurrentTaskID, a.TasksDone, a.Uptime, a.ID)
	if err != nil {
		return fmt.Errorf("update agent: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *postgresAgentStore) Delete(id uuid.UUID) error {
	tag, err := s.pool().Exec(context.Background(), "DELETE FROM agents WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete agent: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}
