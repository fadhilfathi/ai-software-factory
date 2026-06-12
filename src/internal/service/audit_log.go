package service

import (
	"context"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type AuditLogService struct {
	store store.Store
	log   *zap.Logger
}

func NewAuditLogService(s store.Store, log *zap.Logger) *AuditLogService {
	return &AuditLogService{store: s, log: log}
}

func (s *AuditLogService) LogAction(ctx context.Context, entityType string, entityID uuid.UUID, action string, userID *uuid.UUID, changes interface{}) {
	// We use a fire-and-forget or best-effort logging approach for system actions
	// In a real system, this might be sent to a queue.
	
	now := time.Now().UTC()
	log := &model.AuditLog{
		ID:         uuid.New(),
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		UserID:     userID,
		CreatedAt:  now,
	}
	
	if changes != nil {
		// Convert changes to RawMessage if needed, or assume it's already serializable
		// Simplified for this implementation
	}

	if err := s.store.AuditLogs().Create(log); err != nil {
		s.log.Error("failed to create audit log", zap.Error(err))
	}
}

func (s *AuditLogService) ListLogs(ctx context.Context, filter store.AuditLogFilter) ([]*model.AuditLog, *store.Pagination, *Error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 20
	}

	logs, total, err := s.store.AuditLogs().List(filter)
	if err != nil {
		s.log.Error("failed to list audit logs", zap.Error(err))
		return nil, nil, internalError("Failed to list audit logs")
	}

	pages := (total + filter.Limit - 1) / filter.Limit
	pagination := &store.Pagination{
		Page:  filter.Page,
		Limit: filter.Limit,
		Total: total,
		Pages: pages,
	}

	return logs, pagination, nil
}
