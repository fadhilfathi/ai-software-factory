package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// postgresAssignmentEventStore is the postgres-backed implementation
// of store.AssignmentEventStore for the append-only assignment_events
// table (migration 020).
//
// The store takes a DBTX so it can run inside a transaction (set
// by AssignmentService.AssignTaskToAgent via WithTx). When called
// outside a transaction, the DBTX is the *pgxpool.Pool.
type postgresAssignmentEventStore struct {
	s  *postgresStore
	db DBTX
}

// Append writes a new assignment_event row. The store respects
// server-side defaults for ID and assigned_at but uses the caller's
// values if they are set, so the service can be deterministic in
// tests.
//
// Behaviour:
//   - ID zero → server generates a UUID via gen_random_uuid() default.
//   - AssignedAt zero → server defaults to NOW().
//   - Action must be one of {assign, reassign, unassign} — enforced
//     by the DB CHECK constraint; a bad value surfaces as a
//     pgx error wrapped in fmt.Errorf (the service layer validates
//     first so this is a defence-in-depth catch).
//   - AssignmentID must reference an existing assignments row
//     (FK constraint added in migration 020; the service ensures
//     the assignment row was inserted in the same transaction).
func (s *postgresAssignmentEventStore) Append(ctx context.Context, ev *model.AssignmentEvent) (*model.AssignmentEvent, error) {
	if ev == nil {
		return nil, fmt.Errorf("assignment event is nil")
	}
	if ev.AssignmentID == uuid.Nil {
		return nil, fmt.Errorf("assignment event must have assignment_id")
	}
	if ev.ID == uuid.Nil {
		ev.ID = uuid.New()
	}
	if ev.AssignedAt.IsZero() {
		ev.AssignedAt = time.Now().UTC()
	}
	// Defensive: validate action enum before hitting the DB so
	// callers get a clean error instead of a driver-level message.
	if !model.IsValidAssignmentAction(ev.Action) {
		return nil, fmt.Errorf("invalid assignment action %q (want assign|reassign|unassign)", ev.Action)
	}
	if ev.TaskID == uuid.Nil {
		return nil, fmt.Errorf("assignment event must have a task_id")
	}

	const query = `INSERT INTO assignment_events
		(id, assignment_id, task_id, agent_id, assigned_by, assigned_at, action, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, assignment_id, task_id, agent_id, assigned_by, assigned_at, action, notes`

	row := s.db.QueryRow(ctx, query,
		ev.ID, ev.AssignmentID, ev.TaskID, ev.AgentID, ev.AssignedBy, ev.AssignedAt, string(ev.Action), ev.Notes)
	out := &model.AssignmentEvent{}
	var agentID, assignedBy *uuid.UUID
	var assignedAt time.Time
	if err := row.Scan(&out.ID, &out.AssignmentID, &out.TaskID, &agentID, &assignedBy, &assignedAt, &out.Action, &out.Notes); err != nil {
		return nil, fmt.Errorf("insert assignment event: %w", err)
	}
	out.AgentID = agentID
	out.AssignedBy = assignedBy
	out.AssignedAt = assignedAt
	return out, nil
}

// ListByTask returns all events for a task, newest first. The DB
// uses the (task_id, assigned_at DESC) index from migration 020.
// Returns an empty slice (not nil) when the task has no events.
// The store does NOT 404 when the task itself doesn't exist —
// that's the service layer's responsibility.
//
// Reads always go through the pool, not a tx, so this method
// requires the store to be constructed with a pool DBTX (which is
// the case when called via the Store interface — only the
// transactional closure in AssignmentService uses a tx DBTX).
func (s *postgresAssignmentEventStore) ListByTask(ctx context.Context, taskID uuid.UUID) ([]*model.AssignmentEvent, error) {
	const query = `SELECT id, assignment_id, task_id, agent_id, assigned_by, assigned_at, action, notes
		FROM assignment_events
		WHERE task_id = $1
		ORDER BY assigned_at DESC, id DESC`
	rows, err := s.db.Query(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("list assignment events: %w", err)
	}
	defer rows.Close()

	out := []*model.AssignmentEvent{}
	for rows.Next() {
		ev := &model.AssignmentEvent{}
		var agentID, assignedBy *uuid.UUID
		if err := rows.Scan(&ev.ID, &ev.AssignmentID, &ev.TaskID, &agentID, &assignedBy, &ev.AssignedAt, &ev.Action, &ev.Notes); err != nil {
			return nil, fmt.Errorf("scan assignment event: %w", err)
		}
		ev.AgentID = agentID
		ev.AssignedBy = assignedBy
		out = append(out, ev)
	}
	if err := rows.Err(); err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("iterate assignment events: %w", err)
	}
	return out, nil
}

// Compile-time interface assertion.
var _ store.AssignmentEventStore = (*postgresAssignmentEventStore)(nil)
