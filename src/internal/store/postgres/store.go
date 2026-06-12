package postgres

import (
	"context"
	"fmt"

	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresStore struct {
	pool     *pgxpool.Pool
	fallback store.Store
}

func NewStore(pool *pgxpool.Pool) store.Store {
	return &postgresStore{
		pool:     pool,
		fallback: store.NewMemoryStore(),
	}
}

func (s *postgresStore) Users() store.UserStore             { return &postgresUserStore{s} }
func (s *postgresStore) Projects() store.ProjectStore        { return &postgresProjectStore{s} }
func (s *postgresStore) Agents() store.AgentStore             { return &postgresAgentStore{s} }
func (s *postgresStore) Capabilities() store.CapabilityStore  { return &postgresCapabilityStore{s} }
func (s *postgresStore) AgentRuns() store.AgentRunStore       { return &postgresAgentRunStore{s} }
// TASK-405: ExecutionStore. Returns a DBTX-backed sub-store so the
// same code path can be reused inside a postgresTx (see WithTx).
// No postgresTx.Executions() method today — execution writes are
// single-statement and don't need transactional grouping — but the
// type is ready to extend when TASK-405-followup work needs it.
func (s *postgresStore) Executions() store.ExecutionStore { return &postgresExecutionStore{db: s.pool} }

// Deliverables is TASK-406. Returns a DBTX-backed sub-store so
// the same code path can be reused inside a postgresTx (see
// WithTx). The Update method is in-place; the service layer
// coordinates with DeliverableVersions().Insert to maintain
// the append-only history invariant.
func (s *postgresStore) Deliverables() store.DeliverableStore { return &postgresDeliverableStore{db: s.pool} }

// DeliverableVersions is the TASK-406 append-only history
// store. Both Deliverables and DeliverableVersions are
// re-exposed by postgresTx so the service can coordinate an
// in-place main-row update + a history-row insert in a single
// SQL transaction.
func (s *postgresStore) DeliverableVersions() store.DeliverableVersionStore {
	return &postgresDeliverableVersionStore{db: s.pool}
}
func (s *postgresStore) Tasks() store.TaskStore               { return &postgresTaskStore{s} }
func (s *postgresStore) Code() store.CodeStore                { return &postgresCodeStore{s} }
func (s *postgresStore) Reviews() store.ReviewStore           { return &postgresReviewStore{s} }
func (s *postgresStore) Deployments() store.DeploymentStore   { return s.fallback.Deployments() }
func (s *postgresStore) Webhooks() store.WebhookStore         { return s.fallback.Webhooks() }
func (s *postgresStore) AuditLogs() store.AuditLogStore       { return &postgresAuditLogStore{s} }
func (s *postgresStore) Tokens() store.TokenStore           { return s.fallback.Tokens() }
// TASK-404: append-only history store backed by assignment_events (migration 020).
func (s *postgresStore) AssignmentEvents() store.AssignmentEventStore {
	return &postgresAssignmentEventStore{s: s, db: s.pool}
}
// TASK-404: current-state assignments table backed by the
// `assignments` table (migration 019). Returned with the pool as
// the DBTX so reads happen against the connection pool. Writes
// from inside a transaction go through postgresTx (see WithTx).
func (s *postgresStore) Assignments() store.AssignmentStore {
	return &postgresAssignmentStore{s: s, db: s.pool}
}

// WithTx opens a SQL transaction, runs the closure with a tx-scoped
// view of the store, and commits if the closure returns nil. Any
// non-nil return from the closure triggers a rollback. The closure
// receives the same sub-store methods (Assignments, AssignmentEvents)
// but they execute against the same underlying pgx.Tx, so a Create
// in assignments followed by an Append in assignment_events is
// atomic.
func (s *postgresStore) WithTx(ctx context.Context, fn func(store.Tx) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		// Rollback is a no-op after Commit, so this is safe to
		// always defer. The pgx docs explicitly call this out.
		_ = tx.Rollback(ctx)
	}()

	t := &postgresTx{s: s, tx: tx}
	if err := fn(t); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// postgresTx is the transactional view handed to WithTx closures.
// Both Assignments() and AssignmentEvents() return sub-stores
// bound to the same pgx.Tx.
type postgresTx struct {
	s  *postgresStore
	tx pgx.Tx
}

func (t *postgresTx) Assignments() store.AssignmentStore {
	return &postgresAssignmentStore{s: t.s, db: t.tx}
}

func (t *postgresTx) AssignmentEvents() store.AssignmentEventStore {
	return &postgresAssignmentEventStore{s: t.s, db: t.tx}
}

func (t *postgresTx) Deliverables() store.DeliverableStore {
	return &postgresDeliverableStore{db: t.tx}
}
func (t *postgresTx) DeliverableVersions() store.DeliverableVersionStore {
	return &postgresDeliverableVersionStore{db: t.tx}
}
