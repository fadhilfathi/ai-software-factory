package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// postgresExecutionStore is the Sprint 4 (TASK-405) implementation of
// store.ExecutionStore. It is built on the DBTX pattern (see
// assignment_store.go): the underlying Exec/Query/QueryRow work for
// both *pgxpool.Pool and pgx.Tx, so a future transactional refactor
// can wrap multiple calls in a single tx without rewriting this file.
//
// The store is intentionally narrow: it does NOT validate status
// transitions. The service layer (see service/execution.go) is the
// single source of truth for the state machine.
type postgresExecutionStore struct {
	db DBTX
}

func (s *postgresExecutionStore) Create(ctx context.Context, e *model.Execution) error {
	const query = `INSERT INTO executions
		(task_id, agent_id, status, started_at, completed_at, error_message,
		 aion_agent_instance_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	now := time.Now().UTC()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	if e.UpdatedAt.IsZero() {
		e.UpdatedAt = now
	}

	err := s.db.QueryRow(ctx, query,
		e.TaskID, e.AgentID, e.Status, e.StartedAt, e.CompletedAt, e.ErrorMessage,
		e.AionAgentInstanceID, e.CreatedAt, e.UpdatedAt,
	).Scan(&e.ExecutionID)
	if err != nil {
		// 23505 = unique_violation. The PK on `id` is the only
		// unique index on executions today, so any 23505 here is
		// a duplicate id (caller-supplied or generated race).
		// We surface it as ErrAlreadyExists so the service can
		// 409 the request.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return store.ErrAlreadyExists
		}
		return fmt.Errorf("create execution: %w", err)
	}
	return nil
}

func (s *postgresExecutionStore) GetByID(ctx context.Context, id uuid.UUID) (*model.Execution, error) {
	const query = `SELECT id, task_id, agent_id, status, started_at, completed_at,
		error_message, aion_agent_instance_id, created_at, updated_at
		FROM executions WHERE id = $1`
	return s.scanOne(ctx, query, id)
}

// List returns a keyset-paginated page of executions. The cursor is
// the ExecutionID of the last row in the previous page; we look up
// its started_at to construct the (started_at, id) tuple that
// defines the keyset boundary. The ORDER BY is (started_at DESC
// NULLS LAST, id DESC) — the same shape the in-memory store uses.
//
// A cursor that doesn't resolve to an existing row is treated as
// "no cursor" (we return the first page). This matches the
// in-memory store's behaviour and avoids forcing callers to handle
// a separate "stale cursor" error path.
func (s *postgresExecutionStore) List(ctx context.Context, filter model.ExecutionFilter) (*model.ExecutionListResult, error) {
	var conditions []string
	var args []any
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
		args = append(args, string(filter.Status))
		argIdx++
	}

	// Keyset cursor: look up the cursor row's started_at.
	// We do this BEFORE building the page query so the cursor
	// is bound at argIdx and we don't have to renumber the
	// filter args. A stale cursor (row was deleted) is treated
	// as "no cursor" — we return the first page. This matches
	// the in-memory store's behaviour and avoids forcing
	// callers to handle a separate "stale cursor" error path.
	if filter.Cursor != uuid.Nil {
		var cursorStartedAt *time.Time
		err := s.db.QueryRow(ctx, `SELECT started_at FROM executions WHERE id = $1`, filter.Cursor).Scan(&cursorStartedAt)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("resolve cursor: %w", err)
		}
		if err == nil && cursorStartedAt != nil {
			conditions = append(conditions, fmt.Sprintf(
				"(started_at, id) < ($%d, $%d)", argIdx, argIdx+1,
			))
			args = append(args, *cursorStartedAt, filter.Cursor)
			argIdx += 2
		}
		// err != nil (cursor row was deleted) or cursorStartedAt == nil:
		// skip the cursor predicate entirely.
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Page size: clamp to [1, MaxExecutionLimit]; default to
	// DefaultExecutionLimit when the caller didn't ask.
	limit := filter.Limit
	if limit <= 0 {
		limit = model.DefaultExecutionLimit
	}
	if limit > model.MaxExecutionLimit {
		limit = model.MaxExecutionLimit
	}
	args = append(args, limit+1) // +1 to detect "is there a next page?"

	dataQuery := fmt.Sprintf(`SELECT id, task_id, agent_id, status, started_at, completed_at,
		error_message, aion_agent_instance_id, created_at, updated_at
		FROM executions %s
		ORDER BY started_at DESC NULLS LAST, id DESC
		LIMIT $%d`, whereClause, argIdx)

	rows, err := s.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("list executions: %w", err)
	}
	defer rows.Close()

	items := make([]*model.Execution, 0, limit)
	for rows.Next() {
		e, err := scanExecutionRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan execution row: %w", err)
		}
		items = append(items, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate executions: %w", err)
	}

	// We fetched limit+1 to detect a next page without a
	// separate COUNT query. If we got back more than `limit`,
	// trim and set the cursor to the last item in the page.
	var nextCursor uuid.UUID
	if len(items) > limit {
		items = items[:limit]
		nextCursor = items[limit-1].ExecutionID
	}

	return &model.ExecutionListResult{Items: items, NextCursor: nextCursor}, nil
}

// UpdateStatus transitions an execution to newStatus. The store
// does NOT validate the transition (no SQL trigger, no CHECK
// beyond the value set); the service layer is the state machine.
//
// Terminal transitions (completed/failed) set completed_at = NOW()
// and optionally set error_message. Non-terminal transitions
// clear error_message so a row that bounced through failed and
// back to running doesn't carry stale error text.
func (s *postgresExecutionStore) UpdateStatus(ctx context.Context, id uuid.UUID, newStatus model.ExecutionStatus, errorMessage *string) (*model.Execution, error) {
	const query = `UPDATE executions SET
		status = $1,
		completed_at = CASE WHEN $1 IN ('completed', 'failed') THEN NOW() ELSE completed_at END,
		error_message = CASE
			WHEN $2::text IS NOT NULL THEN $2
			WHEN $1 <> 'failed' THEN NULL
			ELSE error_message
		END,
		updated_at = NOW()
		WHERE id = $3
		RETURNING id, task_id, agent_id, status, started_at, completed_at,
			error_message, aion_agent_instance_id, created_at, updated_at`

	var errMsgArg *string
	if errorMessage != nil {
		errMsgArg = errorMessage
	}
	row := s.db.QueryRow(ctx, query, string(newStatus), errMsgArg, id)
	e, err := s.scanOneRow(row)
	if err != nil {
		return nil, err
	}
	return e, nil
}

// --- scan helpers -----------------------------------------------------------
//
// scanOne is the single-row helper used by GetByID. It centralises
// the pgx.ErrNoRows → ErrNotFound translation and keeps the public
// methods short. Multi-row callers (List) use scanExecutionRow
// directly on a pgx.Rows.
func (s *postgresExecutionStore) scanOne(ctx context.Context, query string, args ...any) (*model.Execution, error) {
	row := s.db.QueryRow(ctx, query, args...)
	e, err := s.scanOneRow(row)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (s *postgresExecutionStore) scanOneRow(row pgx.Row) (*model.Execution, error) {
	e := &model.Execution{}
	err := row.Scan(
		&e.ExecutionID, &e.TaskID, &e.AgentID, &e.Status, &e.StartedAt, &e.CompletedAt,
		&e.ErrorMessage, &e.AionAgentInstanceID, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("scan execution: %w", err)
	}
	return e, nil
}

// scanExecutionRow scans a single row from a multi-row query. The
// column order must match the SELECT in List and any other
// multi-row query that uses it.
func scanExecutionRow(rows pgx.Rows) (*model.Execution, error) {
	e := &model.Execution{}
	err := rows.Scan(
		&e.ExecutionID, &e.TaskID, &e.AgentID, &e.Status, &e.StartedAt, &e.CompletedAt,
		&e.ErrorMessage, &e.AionAgentInstanceID, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return e, nil
}
