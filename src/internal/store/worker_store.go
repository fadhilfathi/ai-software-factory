// WorkerStore is the persistence boundary for Aion workers. The
// in-memory store (memory.go) ships first; a postgres impl is a
// Sprint 6 follow-up.
//
// All methods are sync (no callback / no streaming). The dispatch
// layer (TASK-502) wraps the Store with its own in-flight
// tracking — the WorkerStore here is just the durable record.
package store

import (
	"context"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/google/uuid"
)

// ErrNotFound is re-exported here for documentation continuity; the
// canonical sentinel is declared in store.go. WorkerStore methods
// return the canonical store.ErrNotFound on miss.
var _ = ErrNotFound

// WorkerStore persists Worker rows.
type WorkerStore interface {
	// Create inserts a new worker. ID is assigned by the store
	// (uuid.New). Returns the persisted row (with ID set).
	Create(ctx context.Context, w *model.Worker) (*model.Worker, error)

	// GetByID returns the worker with the given id, or
	// store.ErrNotFound.
	GetByID(ctx context.Context, id uuid.UUID) (*model.Worker, error)

	// ListByExecution returns all workers (across attempts)
	// for the given execution, ordered by Attempt ASC. The
	// first row is the original spawn; later rows are retries
	// (TASK-508).
	ListByExecution(ctx context.Context, executionID uuid.UUID) ([]*model.Worker, error)

	// ListByAgent returns all workers for the given agent,
	// ordered by StartedAt DESC (most-recent first). Used by
	// the activity dashboard (TASK-506).
	ListByAgent(ctx context.Context, agentID uuid.UUID) ([]*model.Worker, error)

	// Update mutates a worker in place. The ID must match an
	// existing row; Status, StartedAt, CompletedAt, Result,
	// ErrorMessage, PID, and Handle are all overwritten. The
	// store does NOT validate the transition — that's the
	// service layer's job.
	Update(ctx context.Context, w *model.Worker) error

	// Delete removes a worker. Mostly used by tests +
	// retention/cleanup; not on the hot path.
	Delete(ctx context.Context, id uuid.UUID) error
}
