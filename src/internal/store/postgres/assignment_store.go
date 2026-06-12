package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX is the small interface shared by *pgxpool.Pool and pgx.Tx.
// Both expose Exec/Query/QueryRow with the same shape, so each
// postgres sub-store takes a DBTX and is unaware of whether it is
// running inside a transaction or not. This is the standard pgx
// pattern for tx-aware sub-stores.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Compile-time assertions that both pgxpool.Pool and pgx.Tx satisfy
// DBTX. If pgx changes its API surface these will fail to build,
// which is the desired loud-failure signal.
var (
	_ DBTX = (*pgxpool.Pool)(nil)
	_ DBTX = (pgx.Tx)(nil)
)

// postgresAssignmentStore is the postgres-backed implementation of
// store.AssignmentStore for the assignments table (migration 019).
//
// The store takes a DBTX so it can run inside a transaction (set
// by AssignmentService.AssignTaskToAgent via WithTx). When called
// outside a transaction, the DBTX is the *pgxpool.Pool.
type postgresAssignmentStore struct {
	s  *postgresStore
	db DBTX
}

func (s *postgresAssignmentStore) Create(ctx context.Context, a *model.Assignment) (*model.Assignment, error) {
	if a == nil {
		return nil, fmt.Errorf("assignment is nil")
	}
	if a.TaskID == uuid.Nil || a.AgentID == uuid.Nil {
		return nil, fmt.Errorf("assignment must have task_id and agent_id")
	}
	if a.Status == "" {
		a.Status = model.AssignmentStatusActive
	}
	if a.Status != model.AssignmentStatusActive {
		return nil, fmt.Errorf("Create expects status=active, got %q", a.Status)
	}
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	if a.AssignedAt.IsZero() {
		a.AssignedAt = time.Now().UTC()
	}

	const query = `INSERT INTO assignments (id, task_id, agent_id, assigned_at, completed_at, status)
		VALUES ($1, $2, $3, $4, NULL, $5)
		RETURNING id, task_id, agent_id, assigned_at, completed_at, status`
	row := s.db.QueryRow(ctx, query, a.ID, a.TaskID, a.AgentID, a.AssignedAt, string(a.Status))
	out := &model.Assignment{}
	var completedAt *time.Time
	var status string
	if err := row.Scan(&out.ID, &out.TaskID, &out.AgentID, &out.AssignedAt, &completedAt, &status); err != nil {
		// Map the partial unique index violation to a typed
		// sentinel so the service can return 409.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, store.ErrAlreadyExists
		}
		return nil, fmt.Errorf("insert assignment: %w", err)
	}
	out.CompletedAt = completedAt
	out.Status = model.AssignmentStatus(status)
	return out, nil
}

// Update mutates an existing row. Used by the service to flip a
// previous active row to 'superseded' (with completed_at = now).
func (s *postgresAssignmentStore) Update(ctx context.Context, a *model.Assignment) error {
	if a == nil {
		return fmt.Errorf("assignment is nil")
	}
	if a.ID == uuid.Nil {
		return fmt.Errorf("assignment must have id")
	}
	if !model.IsValidAssignmentStatus(a.Status) {
		return fmt.Errorf("invalid assignment status %q", a.Status)
	}
	const query = `UPDATE assignments SET status=$1, completed_at=$2 WHERE id=$3`
	tag, err := s.db.Exec(ctx, query, string(a.Status), a.CompletedAt, a.ID)
	if err != nil {
		// Map the partial unique index violation to ErrAlreadyExists.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return store.ErrAlreadyExists
		}
		return fmt.Errorf("update assignment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *postgresAssignmentStore) GetByID(ctx context.Context, id uuid.UUID) (*model.Assignment, error) {
	const query = `SELECT id, task_id, agent_id, assigned_at, completed_at, status
		FROM assignments WHERE id = $1`
	a := &model.Assignment{}
	var completedAt *time.Time
	var status string
	err := s.db.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.TaskID, &a.AgentID, &a.AssignedAt, &completedAt, &status,
	)
	if err == pgx.ErrNoRows {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get assignment: %w", err)
	}
	a.CompletedAt = completedAt
	a.Status = model.AssignmentStatus(status)
	return a, nil
}

// GetActiveByTask returns the active row for the task using the
// partial unique index uq_assignments_one_active_per_task. The
// query is O(1) in practice because there is at most one matching
// row.
func (s *postgresAssignmentStore) GetActiveByTask(ctx context.Context, taskID uuid.UUID) (*model.Assignment, error) {
	const query = `SELECT id, task_id, agent_id, assigned_at, completed_at, status
		FROM assignments
		WHERE task_id = $1 AND status = 'active'
		LIMIT 1`
	a := &model.Assignment{}
	var completedAt *time.Time
	var status string
	err := s.db.QueryRow(ctx, query, taskID).Scan(
		&a.ID, &a.TaskID, &a.AgentID, &a.AssignedAt, &completedAt, &status,
	)
	if err == pgx.ErrNoRows {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get active assignment: %w", err)
	}
	a.CompletedAt = completedAt
	a.Status = model.AssignmentStatus(status)
	return a, nil
}

// Compile-time interface assertion.
var _ store.AssignmentStore = (*postgresAssignmentStore)(nil)
