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

// postgresDeliverableStore is the Sprint 4 (TASK-406) implementation
// of store.DeliverableStore. It is built on the DBTX pattern (see
// assignment_store.go) so the same code path can be reused inside
// a postgresTx (see WithTx in store.go).
//
// The store is intentionally narrow: the Update method is
// in-place on the main row. The append-only history invariant
// is enforced by the service layer, which coordinates with
// postgresDeliverableVersionStore.Insert via WithTx.
type postgresDeliverableStore struct {
	db DBTX
}

func (s *postgresDeliverableStore) Create(ctx context.Context, d *model.Deliverable) error {
	// Note: 022 added `updated_at` to the deliverables table.
	// We rely on the column's DEFAULT NOW() for first-INSERT.
	const query = `INSERT INTO deliverables
		(task_id, agent_id, title, content, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		RETURNING id, updated_at`
	now := time.Now().UTC()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	err := s.db.QueryRow(ctx, query,
		d.TaskID, d.AgentID, d.Title, d.Content, d.Version, d.CreatedAt,
	).Scan(&d.ID, &d.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return store.ErrAlreadyExists
		}
		return fmt.Errorf("create deliverable: %w", err)
	}
	return nil
}

func (s *postgresDeliverableStore) GetByID(ctx context.Context, id uuid.UUID) (*model.Deliverable, error) {
	const query = `SELECT id, task_id, agent_id, title, content, version, created_at, updated_at
		FROM deliverables WHERE id = $1`
	row := s.db.QueryRow(ctx, query, id)
	d := &model.Deliverable{}
	err := row.Scan(&d.ID, &d.TaskID, &d.AgentID, &d.Title, &d.Content, &d.Version, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("get deliverable: %w", err)
	}
	return d, nil
}

// List returns a keyset-paginated page of deliverables matching
// the filter. The cursor is the ID of the last row in the
// previous page; we look up its created_at to construct the
// (created_at, id) tuple that defines the keyset boundary.
// ORDER BY is (created_at DESC, id DESC) — the same shape the
// in-memory store uses.
//
// A cursor that doesn't resolve to an existing row is treated
// as "no cursor" (we return the first page). This matches the
// in-memory store's behaviour and avoids forcing callers to
// handle a separate "stale cursor" error path.
func (s *postgresDeliverableStore) List(ctx context.Context, filter model.DeliverableFilter) (*model.DeliverableListResult, error) {
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

	// Keyset cursor: look up the cursor row's created_at.
	if filter.Cursor != uuid.Nil {
		var cursorCreatedAt *time.Time
		err := s.db.QueryRow(ctx, `SELECT created_at FROM deliverables WHERE id = $1`, filter.Cursor).Scan(&cursorCreatedAt)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("resolve cursor: %w", err)
		}
		if err == nil && cursorCreatedAt != nil {
			conditions = append(conditions, fmt.Sprintf(
				"(created_at, id) < ($%d, $%d)", argIdx, argIdx+1,
			))
			args = append(args, *cursorCreatedAt, filter.Cursor)
			argIdx += 2
		}
		// Stale cursor → skip the cursor predicate entirely.
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Page size: clamp to [1, MaxDeliverableLimit]; default
	// to DefaultDeliverableLimit when the caller didn't ask.
	limit := filter.Limit
	if limit <= 0 {
		limit = model.DefaultDeliverableLimit
	}
	if limit > model.MaxDeliverableLimit {
		limit = model.MaxDeliverableLimit
	}
	args = append(args, limit+1) // +1 to detect next page

	dataQuery := fmt.Sprintf(`SELECT id, task_id, agent_id, title, content, version, created_at, updated_at
		FROM deliverables %s
		ORDER BY created_at DESC, id DESC
		LIMIT $%d`, whereClause, argIdx)

	rows, err := s.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("list deliverables: %w", err)
	}
	defer rows.Close()

	items := make([]*model.Deliverable, 0, limit)
	for rows.Next() {
		d := &model.Deliverable{}
		if err := rows.Scan(&d.ID, &d.TaskID, &d.AgentID, &d.Title, &d.Content, &d.Version, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan deliverable: %w", err)
		}
		items = append(items, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate deliverables: %w", err)
	}

	var nextCursor uuid.UUID
	if len(items) > limit {
		items = items[:limit]
		nextCursor = items[limit-1].ID
	}

	return &model.DeliverableListResult{Items: items, NextCursor: nextCursor}, nil
}

// Update applies a new state to an existing deliverable in-place.
// The service coordinates with postgresDeliverableVersionStore.Insert
// via WithTx to maintain the append-only history invariant.
func (s *postgresDeliverableStore) Update(ctx context.Context, d *model.Deliverable) error {
	const query = `UPDATE deliverables SET
		title = $1,
		content = $2,
		version = $3,
		updated_at = NOW()
		WHERE id = $4
		RETURNING updated_at`
	err := s.db.QueryRow(ctx, query,
		d.Title, d.Content, d.Version, d.ID,
	).Scan(&d.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return store.ErrNotFound
		}
		return fmt.Errorf("update deliverable: %w", err)
	}
	return nil
}

// postgresDeliverableVersionStore is the Sprint 4 (TASK-406)
// implementation of store.DeliverableVersionStore (the
// append-only history of deliverable title/content changes).
type postgresDeliverableVersionStore struct {
	db DBTX
}

func (s *postgresDeliverableVersionStore) Insert(ctx context.Context, v *model.DeliverableVersion) error {
	const query = `INSERT INTO deliverable_versions
		(deliverable_id, version, title, content, created_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`
	now := time.Now().UTC()
	if v.CreatedAt.IsZero() {
		v.CreatedAt = now
	}
	err := s.db.QueryRow(ctx, query,
		v.DeliverableID, v.Version, v.Title, v.Content, v.CreatedAt, v.CreatedBy,
	).Scan(&v.ID)
	if err != nil {
		// 23505 = unique_violation on (deliverable_id, version).
		// The service also pre-computes the next version from
		// the current row, so this is a defence-in-depth check
		// against race conditions (two concurrent PUTs trying
		// to write the same version).
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return store.ErrAlreadyExists
		}
		return fmt.Errorf("insert deliverable version: %w", err)
	}
	return nil
}

func (s *postgresDeliverableVersionStore) ListVersions(ctx context.Context, deliverableID uuid.UUID) ([]*model.DeliverableVersion, error) {
	// The brief says "ORDER BY version DESC". The existing 023
	// UNIQUE(deliverable_id, version) btree index is read
	// backwards for this query plan, so no extra index is
	// needed. The CREATE INDEX IF NOT EXISTS in 023 is for
	// created_by (used by future activity dashboards).
	const query = `SELECT id, deliverable_id, version, title, content, created_at, created_by
		FROM deliverable_versions
		WHERE deliverable_id = $1
		ORDER BY version DESC`
	rows, err := s.db.Query(ctx, query, deliverableID)
	if err != nil {
		return nil, fmt.Errorf("list deliverable versions: %w", err)
	}
	defer rows.Close()

	out := make([]*model.DeliverableVersion, 0, 8)
	for rows.Next() {
		v := &model.DeliverableVersion{}
		if err := rows.Scan(&v.ID, &v.DeliverableID, &v.Version, &v.Title, &v.Content, &v.CreatedAt, &v.CreatedBy); err != nil {
			return nil, fmt.Errorf("scan deliverable version: %w", err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate deliverable versions: %w", err)
	}
	return out, nil
}
