// Integration tests for the postgres-backed assignment stores.
// Gated behind the `integration` build tag so a default
// `go test ./...` doesn't try to open a postgres connection
// (which is unavailable in CI containers and on this dev host).
//
// Run with:
//
//   go test -tags=integration ./internal/store/postgres/...
//
// Required env: DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME.
// Migrations are run on connect; the test reuses the schema and
// TRUNCATEs the two assignment tables between subtests so
// consecutive runs don't poison each other.
//
// A-002-17 chore: the deferred store-layer coverage for the
// A-002-04 retraction. These tests assert the postgres-backed
// AssignmentStore and AssignmentEventStore match the in-memory
// store's behavioural surface (Create / GetByID / Update /
// GetActiveByTask / ListByTask with DESC ordering, plus the
// per-error cases). When the integration.NewPostgresStore stub
// is filled in (Sprint 5 plan), the same test runs unchanged.

//go:build integration

package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/db"
	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// requirePool returns a connected, migrated pgx pool. It skips
// the test if DB_HOST is not set (the canonical "no integration
// env" signal). The caller is responsible for pool.Close().
func requirePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if os.Getenv("DB_HOST") == "" {
		t.Skip("DB_HOST not set; integration tests skipped (run with DB_HOST=... go test -tags=integration)")
	}
	cfg := db.DefaultConfig()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := db.Connect(ctx, cfg)
	require.NoError(t, err, "connect to postgres at %s:%s/%s", cfg.Host, cfg.Port, cfg.DBName)
	// Run migrations so the schema is current. RunMigrations is
	// idempotent: it tracks applied versions in schema_migrations.
	migrationsDir := findMigrationsDir(t)
	require.NoError(t, db.RunMigrations(ctx, pool, migrationsDir),
		"run migrations from %s", migrationsDir)
	return pool
}

// findMigrationsDir locates db/migrations relative to this test
// file. We try a few candidates because the test may be run from
// the package dir, the src/ root, or the repo root.
func findMigrationsDir(t *testing.T) string {
	t.Helper()
	candidates := []string{
		"db/migrations",
		"../../db/migrations",
		"../../../db/migrations",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	t.Fatalf("could not find db/migrations; tried %v", candidates)
	return ""
}

// truncateAssignments wipes the two assignment tables between
// subtests. Uses TRUNCATE ... RESTART IDENTITY CASCADE so the
// next subtest starts from a known state.
func truncateAssignments(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := pool.Exec(ctx, `TRUNCATE assignment_events, assignments RESTART IDENTITY CASCADE`)
	require.NoError(t, err)
}

// seedTaskAndAgent inserts a bare task and agent into postgres
// and returns their IDs. We use the public store API (not raw
// SQL) so the seed goes through the same code path as production
// writes.
func seedTaskAndAgent(t *testing.T, st store.Store, projectID uuid.UUID) (taskID, agentID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	task := &model.Task{
		ID:              uuid.New(),
		ProjectID:       projectID,
		Title:           "anchor",
		Status:          model.TaskOpen,
		Priority:        model.PriorityNormal,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
		RequiredCapabilities: []string{"coding"},
	}
	require.NoError(t, st.Tasks().Create(task))
	agent := &model.Agent{
		ID:        uuid.New(),
		ProjectID: projectID,
		Name:      "agent-" + uuid.NewString()[:8],
		Role:      "developer",
		Status:    model.AgentIdle,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, st.Agents().Create(agent))
	return task.ID, agent.ID
}

// ---- AssignmentStore -------------------------------------------------

func TestAssignmentStore_Integration(t *testing.T) {
	pool := requirePool(t)
	defer pool.Close()
	st := NewStore(pool)

	cases := []struct {
		name      string
		fn        func(t *testing.T, s store.Store)
	}{
		{
			name: "Create_AndGetByID",
			fn: func(t *testing.T, s store.Store) {
				truncateAssignments(t, pool)
				projectID := uuid.New()
				taskID, agentID := seedTaskAndAgent(t, s, projectID)

				assignment := &model.Assignment{
					ID:        uuid.New(),
					TaskID:    taskID,
					AgentID:   agentID,
					AssignedAt: time.Now().UTC(),
					Status:    model.AssignmentStatusActive,
				}
				require.NoError(t, s.Assignments().Create(context.Background(), assignment))

				got, err := s.Assignments().GetByID(context.Background(), assignment.ID)
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, assignment.ID, got.ID)
				assert.Equal(t, taskID, got.TaskID)
				assert.Equal(t, agentID, got.AgentID)
				assert.Equal(t, model.AssignmentStatusActive, got.Status)
				assert.Nil(t, got.CompletedAt, "fresh assignment has nil completed_at")
			},
		},
		{
			name: "GetByID_NotFound_ReturnsErrNotFound",
			fn: func(t *testing.T, s store.Store) {
				truncateAssignments(t, pool)
				_, err := s.Assignments().GetByID(context.Background(), uuid.New())
				assert.ErrorIs(t, err, store.ErrNotFound,
					"unknown assignment id must surface store.ErrNotFound so the service maps it to 404 NOT_FOUND")
			},
		},
		{
			name: "Update_FlipsStatusToSuperseded_AndSetsCompletedAt",
			fn: func(t *testing.T, s store.Store) {
				truncateAssignments(t, pool)
				projectID := uuid.New()
				taskID, agentA := seedTaskAndAgent(t, s, projectID)

				a := &model.Assignment{
					ID:         uuid.New(),
					TaskID:     taskID,
					AgentID:    agentA,
					AssignedAt: time.Now().UTC(),
					Status:     model.AssignmentStatusActive,
				}
				require.NoError(t, s.Assignments().Create(context.Background(), a))

				now := time.Now().UTC()
				a.Status = model.AssignmentStatusSuperseded
				a.CompletedAt = &now
				require.NoError(t, s.Assignments().Update(context.Background(), a))

				got, err := s.Assignments().GetByID(context.Background(), a.ID)
				require.NoError(t, err)
				assert.Equal(t, model.AssignmentStatusSuperseded, got.Status)
				require.NotNil(t, got.CompletedAt)
				assert.WithinDuration(t, now, *got.CompletedAt, time.Second)
			},
		},
		{
			name: "GetActiveByTask_ReturnsTheActiveRow",
			fn: func(t *testing.T, s store.Store) {
				truncateAssignments(t, pool)
				projectID := uuid.New()
				taskID, agentA := seedTaskAndAgent(t, s, projectID)

				active := &model.Assignment{
					ID:         uuid.New(),
					TaskID:     taskID,
					AgentID:    agentA,
					AssignedAt: time.Now().UTC(),
					Status:     model.AssignmentStatusActive,
				}
				require.NoError(t, s.Assignments().Create(context.Background(), active))

				// Pre-seed a superseded row for the same task. The
				// store must filter by status='active' (the
				// partial unique index in 019_create_assignments.sql
				// is the DB-layer guard; the store SELECT must
				// respect it).
				earlier := time.Now().Add(-time.Hour).UTC()
				now := time.Now().UTC()
				superseded := &model.Assignment{
					ID:         uuid.New(),
					TaskID:     taskID,
					AgentID:    uuid.New(),
					AssignedAt: earlier,
					CompletedAt: &now,
					Status:     model.AssignmentStatusSuperseded,
				}
				require.NoError(t, s.Assignments().Create(context.Background(), superseded))

				got, err := s.Assignments().GetActiveByTask(context.Background(), taskID)
				require.NoError(t, err)
				require.NotNil(t, got, "GetActiveByTask must skip superseded rows")
				assert.Equal(t, active.ID, got.ID)
			},
		},
		{
			name: "GetActiveByTask_NoRows_ReturnsNilNoError",
			fn: func(t *testing.T, s store.Store) {
				truncateAssignments(t, pool)
				projectID := uuid.New()
				taskID, _ := seedTaskAndAgent(t, s, projectID)
				got, err := s.Assignments().GetActiveByTask(context.Background(), taskID)
				assert.NoError(t, err)
				assert.Nil(t, got, "no assignments for this task must return (nil, nil)")
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.fn(t, st)
		})
	}
}

// ---- AssignmentEventStore -------------------------------------------

func TestAssignmentEventStore_Integration(t *testing.T) {
	pool := requirePool(t)
	defer pool.Close()
	st := NewStore(pool)

	cases := []struct {
		name string
		fn   func(t *testing.T, s store.Store)
	}{
		{
			name: "Create_AndListByTask_NewestFirst",
			fn: func(t *testing.T, s store.Store) {
				truncateAssignments(t, pool)
				projectID := uuid.New()
				taskID, agentID := seedTaskAndAgent(t, s, projectID)

				ctx := context.Background()
				// Event 1: oldest (assigned_at = 1h ago).
				t1 := time.Now().Add(-time.Hour).UTC()
				e1 := &model.AssignmentEvent{
					ID:           uuid.New(),
					AssignmentID: uuid.New(),
					TaskID:       taskID,
					AgentID:      &agentID,
					Action:       model.AssignmentActionAssign,
					Notes:        "first",
					AssignedAt:   t1,
				}
				require.NoError(t, s.AssignmentEvents().Create(ctx, e1))
				// Event 2: newer (assigned_at = 1m ago).
				t2 := time.Now().Add(-time.Minute).UTC()
				e2 := &model.AssignmentEvent{
					ID:           uuid.New(),
					AssignmentID: e1.AssignmentID,
					TaskID:       taskID,
					AgentID:      &agentID,
					Action:       model.AssignmentActionReassign,
					Notes:        "second",
					AssignedAt:   t2,
				}
				require.NoError(t, s.AssignmentEvents().Create(ctx, e2))
				// Event 3: newest (assigned_at = now).
				t3 := time.Now().UTC()
				e3 := &model.AssignmentEvent{
					ID:           uuid.New(),
					AssignmentID: e1.AssignmentID,
					TaskID:       taskID,
					AgentID:      &agentID,
					Action:       model.AssignmentActionReassign,
					Notes:        "third",
					AssignedAt:   t3,
				}
				require.NoError(t, s.AssignmentEvents().Create(ctx, e3))

				events, err := s.AssignmentEvents().ListByTask(ctx, taskID)
				require.NoError(t, err)
				require.Len(t, events, 3, "ListByTask must return all three events")

				// DESC ordering invariant: assigned_at must be
				// monotonically non-increasing.
				for i := 1; i < len(events); i++ {
					assert.False(t, events[i].AssignedAt.After(events[i-1].AssignedAt),
						"events[%d] (%v) must not be after events[%d] (%v)",
						i, events[i].AssignedAt, i-1, events[i-1].AssignedAt)
				}

				// The newest event is the third write; the
				// oldest is the first write. ListByTask should
				// return [e3, e2, e1] (newest first).
				assert.Equal(t, e3.ID, events[0].ID, "newest event must be first")
				assert.Equal(t, e2.ID, events[1].ID)
				assert.Equal(t, e1.ID, events[2].ID, "oldest event must be last")
				assert.Equal(t, "third", events[0].Notes, "F-017: notes must round-trip on the read path")
			},
		},
		{
			name: "ListByTask_NoEvents_ReturnsEmptySlice",
			fn: func(t *testing.T, s store.Store) {
				truncateAssignments(t, pool)
				projectID := uuid.New()
				taskID, _ := seedTaskAndAgent(t, s, projectID)
				events, err := s.AssignmentEvents().ListByTask(context.Background(), taskID)
				assert.NoError(t, err)
				assert.Empty(t, events, "no events for a task must return an empty slice, not nil and not an error")
			},
		},
		{
			name: "Create_RespectsNullableAgentID_ForUnassign",
			fn: func(t *testing.T, s store.Store) {
				// Forward-compat: the unassign action verb is
				// reserved for Sprint 5+; this test asserts the
				// store can persist an event with nil agent_id
				// (the migration declares the column NULLABLE
				// for this reason).
				truncateAssignments(t, pool)
				projectID := uuid.New()
				taskID, _ := seedTaskAndAgent(t, s, projectID)
				e := &model.AssignmentEvent{
					ID:           uuid.New(),
					AssignmentID: uuid.New(),
					TaskID:       taskID,
					AgentID:      nil, // unassign: no agent
					Action:       model.AssignmentActionUnassign,
					Notes:        "Sprint 5+ future unassign smoke test",
					AssignedAt:   time.Now().UTC(),
				}
				require.NoError(t, s.AssignmentEvents().Create(context.Background(), e))

				events, err := s.AssignmentEvents().ListByTask(context.Background(), taskID)
				require.NoError(t, err)
				require.Len(t, events, 1)
				assert.Nil(t, events[0].AgentID, "unassign event must persist nil agent_id (nullable column)")
				assert.Equal(t, model.AssignmentActionUnassign, events[0].Action)
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.fn(t, st)
		})
	}
}
