package postgres

// Postgres-backed implementation of the AgentStore and
// CapabilityStore interfaces. Mirrors the layered pattern in
// project_store.go (single struct, pool() helper, store.ErrNotFound
// on no-rows).
//
// Migration reference: src/db/migrations/016_agent_registry.sql
// (Sprint 4 TASK-402). All column names below match the
// post-migration schema exactly.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ============================================================================
// Agents
// ============================================================================

type postgresAgentStore struct {
	s *postgresStore
}

func (s *postgresAgentStore) pool() *pgxpool.Pool { return s.s.pool }

// agentColumns is the canonical SELECT list for the agents table.
// Centralised here so all queries project the same shape and any
// schema change touches exactly one spot.
const agentColumns = `id, project_id, name, role, status, capabilities,
	last_active_at, metadata, runtime, version, retired_at, created_at, updated_at`

// scanAgent scans a single row of the agentColumns projection into a
// model.Agent. The metadata column is read as []byte and assigned to
// json.RawMessage; the capabilities column is read as []string.
func scanAgent(row pgx.Row) (*model.Agent, error) {
	a := &model.Agent{}
	var meta []byte
	var runtime []byte
	err := row.Scan(
		&a.ID, &a.ProjectID, &a.Name, &a.Role, &a.Status, &a.Capabilities,
		&a.LastActiveAt, &meta, &runtime, &a.Version, &a.RetiredAt, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(meta) > 0 {
		a.Metadata = append(json.RawMessage(nil), meta...)
	}
	if len(runtime) > 0 {
		a.Runtime = append(json.RawMessage(nil), runtime...)
	}
	return a, nil
}

func (s *postgresAgentStore) Create(ctx context.Context, a *model.Agent) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	if a.Status == "" {
		a.Status = model.AgentInitializing
	}
	if a.Version == 0 {
		a.Version = 1
	}
	if len(a.Metadata) == 0 {
		a.Metadata = json.RawMessage(`{}`)
	}
	if a.Capabilities == nil {
		a.Capabilities = []string{}
	}
	query := `INSERT INTO agents
		(id, project_id, name, role, status, capabilities, last_active_at, metadata, runtime, version, retired_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
	_, err := s.pool().Exec(ctx, query,
		a.ID, a.ProjectID, a.Name, a.Role, a.Status, a.Capabilities,
		a.LastActiveAt, []byte(a.Metadata), []byte(a.Runtime), a.Version, a.RetiredAt, a.CreatedAt, a.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// unique_violation on agents_name_unique_per_project
			return store.ErrAlreadyExists
		}
		return fmt.Errorf("create agent: %w", err)
	}
	return nil
}

func (s *postgresAgentStore) GetByID(ctx context.Context, id uuid.UUID) (*model.Agent, error) {
	query := `SELECT ` + agentColumns + ` FROM agents WHERE id = $1`
	a, err := scanAgent(s.pool().QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get agent: %w", err)
	}
	return a, nil
}

// List implements cursor pagination. The cursor is the agent ID of
// the last item on the previous page; the query orders by
// (created_at, id) for stable ordering. Limit defaults to 50 and is
// capped at 200 (api-spec.md §1.2). The partial index
// `idx_agents_project_status ... WHERE retired_at IS NULL` is used
// for the active-agent path; include_retired=true skips the partial
// index and uses idx_agents_project_id instead.
func (s *postgresAgentStore) List(ctx context.Context, f model.AgentFilter) (*model.AgentListResult, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	// Build the WHERE clause. ProjectID is required; the service
	// layer guarantees this (api-spec.md §1.2).
	conds := []string{"project_id = $1"}
	args := []interface{}{f.ProjectID}
	argIdx := 2

	if !f.IncludeRetired {
		conds = append(conds, "retired_at IS NULL")
	}
	if f.Status != "" {
		conds = append(conds, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, f.Status)
		argIdx++
	}
	if f.Capability != "" {
		// GIN index on capabilities (idx_agents_capabilities_gin)
		// makes this a single index hit. The @> operator
		// requires the JSONB form `["name"]`.
		conds = append(conds, fmt.Sprintf("capabilities @> $%d::jsonb", argIdx))
		args = append(args, fmt.Sprintf(`["%s"]`, escapeJSON(f.Capability)))
		argIdx++
	}
	if f.Cursor != "" {
		// Cursor = previous page's last id. Use the (created_at,
		// id) tuple comparison for stable ordering across pages.
		cursorID, err := uuid.Parse(f.Cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		conds = append(conds, fmt.Sprintf("(created_at, id) > ($%d, $%d)", argIdx, argIdx+1))
		args = append(args, lookupCreatedAt(ctx, s.pool(), cursorID), cursorID)
		argIdx += 2
	}

	// Fetch limit+1 to detect has_more without a separate count.
	args = append(args, limit+1)
	limitArg := argIdx
	argIdx++

	query := fmt.Sprintf(`SELECT %s FROM agents WHERE %s
		ORDER BY created_at, id LIMIT $%d`, agentColumns, strings.Join(conds, " AND "), limitArg)
	rows, err := s.pool().Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()

	out := make([]*model.Agent, 0, limit)
	for rows.Next() {
		a, err := scanAgent(rows)
		if err != nil {
			return nil, fmt.Errorf("scan agent: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list agents rows: %w", err)
	}

	hasMore := len(out) > limit
	if hasMore {
		out = out[:limit]
	}
	var nextCursor string
	if hasMore && len(out) > 0 {
		nextCursor = out[len(out)-1].ID.String()
	}
	return &model.AgentListResult{Data: out, NextCursor: nextCursor, HasMore: hasMore}, nil
}

// Update applies a partial update. The caller passes the row's
// current Version; the WHERE clause includes version = $X so a
// concurrent writer's update is detected and the row count is 0
// (mapped to ErrConflict). On success, version is bumped and
// updated_at is set to NOW().
func (s *postgresAgentStore) Update(ctx context.Context, a *model.Agent) error {
	currentVersion := a.Version
	a.Version++
	query := `UPDATE agents SET
		name = $2, role = $3, status = $4, capabilities = $5,
		last_active_at = $6, metadata = $7, runtime = $8, updated_at = NOW()
		WHERE id = $1 AND version = $9 AND retired_at IS NULL`
	tag, err := s.pool().Exec(ctx, query,
		a.ID, a.Name, a.Role, a.Status, a.Capabilities,
		a.LastActiveAt, []byte(a.Metadata), []byte(a.Runtime), currentVersion)
	if err != nil {
		return fmt.Errorf("update agent: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Either the id does not exist, the version was stale, or
		// the row is retired. The service layer distinguishes
		// these via a follow-up GetByID if needed.
		return store.ErrConflict
	}
	return nil
}

// SoftDelete transitions the row to status=retired with
// retired_at=NOW() and bumps the version. We do not use
// `UPDATE ... SET status='retired'` in a single statement because
// we need the version bump to be visible to any concurrent
// optimistic-concurrency reader.
func (s *postgresAgentStore) SoftDelete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE agents SET
		status = 'retired', retired_at = NOW(), version = version + 1, updated_at = NOW()
		WHERE id = $1 AND retired_at IS NULL`
	tag, err := s.pool().Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("soft-delete agent: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

// SetCapabilities is the canonical write path for the agent's
// capability list (data-model.md §3). It (1) replaces the
// agent_capabilities join rows, (2) updates the agents.capabilities
// JSONB cache, (3) bumps the version, all in a single transaction.
//
// The service layer is responsible for validating that every name
// in `names` exists in the capabilities catalog before calling this
// method. We do not re-validate here to keep the seam tight.
func (s *postgresAgentStore) SetCapabilities(ctx context.Context, agentID uuid.UUID, names []string) error {
	tx, err := s.pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("set-capabilities begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Replace the join rows. We delete-and-insert rather than diff
	// to keep the SQL simple and atomic; for small lists (the
	// api-spec caps at a handful of names) this is the right
	// trade-off.
	if _, err := tx.Exec(ctx, `DELETE FROM agent_capabilities WHERE agent_id = $1`, agentID); err != nil {
		return fmt.Errorf("set-capabilities delete join: %w", err)
	}
	for _, name := range names {
		// Resolve capability id from name. We assume the catalog
		// is small and the lookup is cheap. The service layer's
		// existence check has already validated every name; this
		// is a defensive belt-and-braces.
		var capID uuid.UUID
		err := tx.QueryRow(ctx, `SELECT id FROM capabilities WHERE name = $1`, name).Scan(&capID)
		if errors.Is(err, pgx.ErrNoRows) {
			// Treat as a no-op for unknown capabilities so a
			// concurrent capability-deletion does not fail the
			// whole transaction. The service layer's earlier
			// validation is the real gate.
			continue
		}
		if err != nil {
			return fmt.Errorf("set-capabilities lookup cap: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO agent_capabilities (agent_id, capability_id, granted_at) VALUES ($1, $2, NOW())`,
			agentID, capID); err != nil {
			return fmt.Errorf("set-capabilities insert join: %w", err)
		}
	}

	// Update the denormalised cache and bump the version.
	tag, err := tx.Exec(ctx, `UPDATE agents SET capabilities = $2, version = version + 1, updated_at = NOW() WHERE id = $1`,
		agentID, names)
	if err != nil {
		return fmt.Errorf("set-capabilities update cache: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("set-capabilities commit: %w", err)
	}
	return nil
}

// ListCapabilitiesByAgent returns the agent's granted capabilities
// with proficiency / granted_at / display_name / category from the
// joined rows. The api-spec.md §1.6 response shape.
func (s *postgresAgentStore) ListCapabilitiesByAgent(ctx context.Context, agentID uuid.UUID) ([]*model.AgentCapabilityView, error) {
	query := `SELECT c.name, c.display_name, c.category, ac.proficiency, ac.granted_at
		FROM agent_capabilities ac
		JOIN capabilities c ON c.id = ac.capability_id
		WHERE ac.agent_id = $1
		ORDER BY ac.granted_at, c.name`
	rows, err := s.pool().Query(ctx, query, agentID)
	if err != nil {
		return nil, fmt.Errorf("list agent capabilities: %w", err)
	}
	defer rows.Close()

	out := make([]*model.AgentCapabilityView, 0, 8)
	for rows.Next() {
		var ac model.AgentCapabilityView
		if err := rows.Scan(&ac.Name, &ac.DisplayName, &ac.Category, &ac.Proficiency, &ac.GrantedAt); err != nil {
			return nil, fmt.Errorf("scan agent capability: %w", err)
		}
		out = append(out, &ac)
	}
	return out, rows.Err()
}

// lookupCreatedAt is a small helper used by the cursor pagination:
// it fetches the created_at timestamp of the cursor's agent id, so
// the (created_at, id) tuple comparison in the main query can find
// the right slice.
func lookupCreatedAt(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) interface{} {
	var t interface{}
	_ = pool.QueryRow(ctx, `SELECT created_at FROM agents WHERE id = $1`, id).Scan(&t)
	return t
}

// escapeJSON escapes a string for safe interpolation into a JSON
// array literal. Capability names are short identifier-like
// strings but we still sanitise against accidental quote injection.
func escapeJSON(s string) string {
	// capability names are constrained to [a-z] (data-model.md §2
	// seed), so this is a defensive guard only.
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return r.Replace(s)
}

// ============================================================================
// Capabilities catalog
// ============================================================================

type postgresCapabilityStore struct {
	s *postgresStore
}

func (s *postgresCapabilityStore) pool() *pgxpool.Pool { return s.s.pool }

func (s *postgresCapabilityStore) GetByName(ctx context.Context, name string) (*model.CapabilityRow, error) {
	query := `SELECT id, name, display_name, category, COALESCE(description, ''), version
		FROM capabilities WHERE name = $1`
	c := &model.CapabilityRow{}
	err := s.pool().QueryRow(ctx, query, name).Scan(
		&c.ID, &c.Name, &c.DisplayName, &c.Category, &c.Description, &c.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get capability: %w", err)
	}
	return c, nil
}

func (s *postgresCapabilityStore) Exists(ctx context.Context, name string) (bool, error) {
	var exists bool
	err := s.pool().QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM capabilities WHERE name = $1)`, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("capability exists: %w", err)
	}
	return exists, nil
}

func (s *postgresCapabilityStore) List(ctx context.Context, f model.CapabilityFilter) (*model.CapabilityListResult, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	conds := []string{"1 = 1"}
	args := []interface{}{}
	argIdx := 1

	if f.Category != "" {
		conds = append(conds, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, f.Category)
		argIdx++
	}
	if f.Cursor != "" {
		conds = append(conds, fmt.Sprintf("name > $%d", argIdx))
		args = append(args, f.Cursor)
		argIdx++
	}
	args = append(args, limit+1)
	limitArg := argIdx

	query := fmt.Sprintf(`SELECT id, name, display_name, category, COALESCE(description, ''), version
		FROM capabilities WHERE %s ORDER BY name LIMIT $%d`,
		strings.Join(conds, " AND "), limitArg)
	rows, err := s.pool().Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list capabilities: %w", err)
	}
	defer rows.Close()

	out := make([]model.CapabilityRow, 0, limit)
	for rows.Next() {
		var c model.CapabilityRow
		if err := rows.Scan(&c.ID, &c.Name, &c.DisplayName, &c.Category, &c.Description, &c.Version); err != nil {
			return nil, fmt.Errorf("scan capability: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list capabilities rows: %w", err)
	}

	hasMore := len(out) > limit
	if hasMore {
		out = out[:limit]
	}
	var nextCursor string
	if hasMore && len(out) > 0 {
		nextCursor = out[len(out)-1].Name
	}
	return &model.CapabilityListResult{Data: out, NextCursor: nextCursor, HasMore: hasMore}, nil
}
