package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresAuditLogStore struct {
	s *postgresStore
}

func (s *postgresAuditLogStore) pool() *pgxpool.Pool {
	return s.s.pool
}

func (s *postgresAuditLogStore) Create(l *model.AuditLog) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	query := `INSERT INTO audit_log (id, entity_type, entity_id, action, user_id, changes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := s.pool().Exec(context.Background(), query,
		l.ID, l.EntityType, l.EntityID, l.Action, l.UserID, l.Changes, l.CreatedAt)
	if err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}
	return nil
}

func (s *postgresAuditLogStore) List(filter store.AuditLogFilter) ([]*model.AuditLog, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.EntityType != "" {
		conditions = append(conditions, fmt.Sprintf("entity_type = $%d", argIdx))
		args = append(args, filter.EntityType)
		argIdx++
	}
	if filter.EntityID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("entity_id = $%d", argIdx))
		args = append(args, filter.EntityID)
		argIdx++
	}
	if filter.UserID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIdx))
		args = append(args, filter.UserID)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_log %s", whereClause)
	var total int
	err := s.pool().QueryRow(context.Background(), countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	page, limit := filter.Page, filter.Limit
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	offset := (page - 1) * limit
	dataQuery := fmt.Sprintf(`SELECT id, entity_type, entity_id, action, user_id, changes, created_at
		FROM audit_log %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := s.pool().Query(context.Background(), dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*model.AuditLog
	for rows.Next() {
		l := &model.AuditLog{}
		if err := rows.Scan(&l.ID, &l.EntityType, &l.EntityID, &l.Action, &l.UserID, &l.Changes, &l.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan audit log: %w", err)
		}
		logs = append(logs, l)
	}

	return logs, total, nil
}
