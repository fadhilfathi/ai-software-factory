package postgres

import (
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
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
func (s *postgresStore) AgentRuns() store.AgentRunStore       { return &postgresAgentRunStore{s} }
func (s *postgresStore) Executions() store.ExecutionStore     { return &postgresExecutionStore{s} }
func (s *postgresStore) Deliverables() store.DeliverableStore { return &postgresDeliverableStore{s} }
func (s *postgresStore) Tasks() store.TaskStore               { return &postgresTaskStore{s} }
func (s *postgresStore) Code() store.CodeStore                { return s.fallback.Code() }
func (s *postgresStore) Reviews() store.ReviewStore           { return s.fallback.Reviews() }
func (s *postgresStore) Deployments() store.DeploymentStore   { return s.fallback.Deployments() }
func (s *postgresStore) Webhooks() store.WebhookStore         { return s.fallback.Webhooks() }
func (s *postgresStore) AuditLogs() store.AuditLogStore       { return &postgresAuditLogStore{s} }
