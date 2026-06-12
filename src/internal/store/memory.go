package store

import (
	"bytes"
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/google/uuid"
)

// memoryStore implements Store with in-memory maps protected by a mutex.
type memoryStore struct {
	mu          sync.RWMutex
	users       map[string]*model.User
	usersEmail  map[string]*model.User
	projects    map[string]*model.Project
	agents      map[string]*model.Agent
	agentRuns   map[string]*model.AgentRun
	executions   map[string]*model.Execution
	deliverables map[string]*model.Deliverable
	// deliverableVersions is the TASK-406 append-only history
	// table mirror. Key: "deliverableID:version" for O(1)
	// uniqueness check; we ALSO maintain a per-deliverable
	// version-id slice for ListVersions ordering.
	deliverableVersions         map[string]*model.DeliverableVersion
	deliverableVersionByDeliverable map[string][]string // deliverable_id → []version_id (in insertion order)
	tasks        map[string]*model.Task
	codeGens    map[string]*model.CodeGenRequest
	files       map[string]*model.ProjectFile // key: "projectID:path"
	commits     map[string]*model.Commit      // key: "projectID:sha"
	reviews        map[string]*model.Review
	reviewComments map[string]*model.ReviewComment
	deployments    map[string]*model.Deployment
	webhooks    map[string]*model.Webhook
	auditLogs   map[string]*model.AuditLog
	tokens      map[string]uuid.UUID
	capabilities map[string]*model.CapabilityRow // key: capability.name
	// assignmentEvents is the TASK-404 append-only history. Keyed by
	// event ID for O(1) Append; a secondary index by task_id powers
	// the ListByTask scan.
	assignmentEvents map[string]*model.AssignmentEvent
	// assignmentByTask is a denormalised index for ListByTask so the
	// handler doesn't have to walk every event in the store. Built
	// lazily on Append; rebuilt in tests that mutate directly.
	assignmentByTask map[string][]string // task_id → []event_id (in insertion order)
	// assignments is the TASK-404 current-state table (migration 019).
	assignments map[string]*model.Assignment
	// activeAssignmentByTask is a denormalised index that maps a
	// task_id to the assignment_id of its currently-active row.
	// Powers GetActiveByTask in O(1). The index is updated under the
	// store's mutex inside Create/Update.
	activeAssignmentByTask map[string]string
}

// NewMemoryStore creates a new in-memory store ready for use.
func NewMemoryStore() Store {
	m := &memoryStore{
		users:       make(map[string]*model.User),
		usersEmail:  make(map[string]*model.User),
		projects:    make(map[string]*model.Project),
		agents:      make(map[string]*model.Agent),
		agentRuns:   make(map[string]*model.AgentRun),
		executions:   make(map[string]*model.Execution),
		deliverables: make(map[string]*model.Deliverable),
		deliverableVersions:         make(map[string]*model.DeliverableVersion),
		deliverableVersionByDeliverable: make(map[string][]string),
		tasks:        make(map[string]*model.Task),
		codeGens:    make(map[string]*model.CodeGenRequest),
		files:       make(map[string]*model.ProjectFile),
		commits:        make(map[string]*model.Commit),
		reviews:        make(map[string]*model.Review),
		reviewComments: make(map[string]*model.ReviewComment),
		deployments:    make(map[string]*model.Deployment),
		webhooks:    make(map[string]*model.Webhook),
		auditLogs:   make(map[string]*model.AuditLog),
		tokens:      make(map[string]uuid.UUID),
		capabilities: make(map[string]*model.CapabilityRow),
		// TASK-404: assignment_events append-only history.
		assignmentEvents: make(map[string]*model.AssignmentEvent),
		assignmentByTask: make(map[string][]string),
		// TASK-404: assignments current-state table.
		assignments:             make(map[string]*model.Assignment),
		activeAssignmentByTask: make(map[string]string),
	}
	// Seed the 6 canonical capabilities from data-model.md §2 so the
	// in-memory store behaves the same as a fresh Postgres install
	// after 016_agent_registry.sql has run.
	for _, c := range canonicalCapabilitySeed() {
		m.capabilities[c.Name] = &c
	}
	return m
}

// canonicalCapabilitySeed returns the 6 capability rows from
// 016_agent_registry.sql. The Postgres impl does not use this —
// the SQL seed is the source of truth in production — but the
// memory store needs a copy to behave equivalently.
func canonicalCapabilitySeed() []model.CapabilityRow {
	return []model.CapabilityRow{
		{Name: "architecture", DisplayName: "Architecture", Category: "architecture", Description: "System design, ADRs, dependency choices."},
		{Name: "coding", DisplayName: "Coding", Category: "coding", Description: "Source code and unit tests in the source tree."},
		{Name: "testing", DisplayName: "Testing", Category: "testing", Description: "Test execution, coverage, bug reports."},
		{Name: "security", DisplayName: "Security", Category: "security", Description: "Threat modeling, code audit, secret scanning."},
		{Name: "devops", DisplayName: "Devops", Category: "devops", Description: "Build, deploy, infra-as-code, monitoring."},
		{Name: "leadership", DisplayName: "Leader", Category: "leadership", Description: "Planning, decomposition, dispatch, conflict resolution."},
	}
}

func (m *memoryStore) Users() UserStore             { return &memoryUserStore{m} }
func (m *memoryStore) Projects() ProjectStore        { return &memoryProjectStore{m} }
func (m *memoryStore) Agents() AgentStore             { return &memoryAgentStore{m} }
func (m *memoryStore) Capabilities() CapabilityStore  { return &memoryCapabilityStore{m} }
func (m *memoryStore) AgentRuns() AgentRunStore       { return &memoryAgentRunStore{m} }
func (m *memoryStore) Executions() ExecutionStore     { return &memoryExecutionStore{m} }
func (m *memoryStore) Deliverables() DeliverableStore { return &memoryDeliverableStore{m} }
func (m *memoryStore) DeliverableVersions() DeliverableVersionStore {
	return &memoryDeliverableVersionStore{m}
}
func (m *memoryStore) Tasks() TaskStore             { return &memoryTaskStore{m} }
func (m *memoryStore) Code() CodeStore              { return &memoryCodeStore{m} }
func (m *memoryStore) Reviews() ReviewStore         { return &memoryReviewStore{m} }
func (m *memoryStore) Deployments() DeploymentStore { return &memoryDeploymentStore{m} }
func (m *memoryStore) Webhooks() WebhookStore       { return &memoryWebhookStore{m} }
func (m *memoryStore) AuditLogs() AuditLogStore     { return &memoryAuditLogStore{m} }
func (m *memoryStore) Tokens() TokenStore           { return &memoryTokenStore{m} }
// TASK-404: in-memory append-only assignment history. Mirrors
// postgresAssignmentEventStore behaviour.
func (m *memoryStore) AssignmentEvents() AssignmentEventStore {
	return &memoryAssignmentEventStore{m}
}
// TASK-404: in-memory current-state assignments table. Mirrors
// postgresAssignmentStore behaviour.
func (m *memoryStore) Assignments() AssignmentStore {
	return &memoryAssignmentStore{m}
}

// WithTx is a no-op for the in-memory store. The mutex already
// serialises every write, so a closure executed sequentially under
// the mutex has the same atomicity guarantees as a real SQL
// transaction. We still run the closure with a Tx view so the
// service code is portable between backends.
func (m *memoryStore) WithTx(ctx context.Context, fn func(Tx) error) error {
	return fn(&memoryTx{m: m})
}

// --- User Store ---

type memoryUserStore struct{ m *memoryStore }

func (s *memoryUserStore) Create(u *model.User) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, exists := s.m.users[u.ID.String()]; exists {
		return ErrAlreadyExists
	}
	if _, exists := s.m.usersEmail[u.Email]; exists {
		return ErrAlreadyExists
	}
	s.m.users[u.ID.String()] = u
	s.m.usersEmail[u.Email] = u
	return nil
}

func (s *memoryUserStore) GetByID(id uuid.UUID) (*model.User, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	u, ok := s.m.users[id.String()]
	if !ok {
		return nil, ErrNotFound
	}
	return u, nil
}

func (s *memoryUserStore) GetByEmail(email string) (*model.User, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	u, ok := s.m.usersEmail[email]
	if !ok {
		return nil, ErrNotFound
	}
	return u, nil
}

func (s *memoryUserStore) List() ([]*model.User, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	result := make([]*model.User, 0, len(s.m.users))
	for _, u := range s.m.users {
		result = append(result, u)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}

func (s *memoryUserStore) Update(u *model.User) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, ok := s.m.users[u.ID.String()]; !ok {
		return ErrNotFound
	}
	// Update email index if changed
	old, _ := s.m.users[u.ID.String()]
	if old.Email != u.Email {
		delete(s.m.usersEmail, old.Email)
		s.m.usersEmail[u.Email] = u
	}
	s.m.users[u.ID.String()] = u
	return nil
}

// CheckProjectAccess returns true if the user has access to the project
func (s *memoryUserStore) CheckProjectAccess(userID, projectID uuid.UUID) bool {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	user, ok := s.m.users[userID.String()]
	if !ok {
		return false
	}
	// Admin users have access to all projects
	if user.Role == model.RoleAdmin {
		return true
	}
	// Check if user is a member of the project
	for _, pid := range user.Projects {
		if pid == projectID.String() {
			return true
		}
	}
	return false
}

// --- Project Store ---

type memoryProjectStore struct{ m *memoryStore }

func (s *memoryProjectStore) Create(p *model.Project) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, exists := s.m.projects[p.ID.String()]; exists {
		return ErrAlreadyExists
	}
	s.m.projects[p.ID.String()] = p
	return nil
}

func (s *memoryProjectStore) GetByID(id uuid.UUID) (*model.Project, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	p, ok := s.m.projects[id.String()]
	if !ok {
		return nil, ErrNotFound
	}
	return p, nil
}

func (s *memoryProjectStore) List(filter ProjectFilter) ([]*model.Project, int, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()

	var filtered []*model.Project
	for _, p := range s.m.projects {
		if filter.Status != "" && p.Status != filter.Status {
			continue
		}
		if filter.OwnerID != uuid.Nil && p.OwnerID != filter.OwnerID {
			continue
		}
		if filter.Search != "" && !strings.Contains(strings.ToLower(p.Name), strings.ToLower(filter.Search)) && !strings.Contains(strings.ToLower(p.Description), strings.ToLower(filter.Search)) {
			continue
		}

		// Calculate active agents for this project
		activeCount := 0
		for _, a := range s.m.agents {
			if a.ProjectID == p.ID && a.Status == model.AgentBusy {
				activeCount++
			}
		}
		p.ActiveAgents = activeCount

		filtered = append(filtered, p)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
	})

	total := len(filtered)

	page, limit := filter.Page, filter.Limit
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	start := (page - 1) * limit
	if start >= len(filtered) {
		return []*model.Project{}, total, nil
	}
	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], total, nil
}

func (s *memoryProjectStore) Update(p *model.Project) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, ok := s.m.projects[p.ID.String()]; !ok {
		return ErrNotFound
	}
	s.m.projects[p.ID.String()] = p
	return nil
}

func (s *memoryProjectStore) Delete(id uuid.UUID) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	key := id.String()
	if _, ok := s.m.projects[key]; !ok {
		return ErrNotFound
	}
	delete(s.m.projects, key)
	return nil
}

// --- Agent Store ---

// memoryAgentStore is the in-memory AgentStore implementation.
//
// Concurrency: all reads go through s.m.mu (a single RWMutex on
// memoryStore) for simplicity. The data set is small (the agent
// table is bounded by the number of projects * agents-per-project).
type memoryAgentStore struct{ m *memoryStore }

// create is the package-private helper called by Service / store
// internals to set default fields. It is intentionally not on the
// AgentStore interface; AgentStore.Create takes a fully-formed
// model.Agent and expects the service to set defaults before calling.
func (s *memoryAgentStore) Create(ctx context.Context, a *model.Agent) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	for _, existing := range s.m.agents {
		if existing.ProjectID == a.ProjectID && existing.Name == a.Name && existing.RetiredAt == nil {
			return ErrAlreadyExists
		}
	}
	// Defensive copy so the caller cannot mutate the stored entry.
	out := *a
	if a.Capabilities != nil {
		out.Capabilities = append([]string(nil), a.Capabilities...)
	}
	if len(a.Metadata) > 0 {
		buf := make([]byte, len(a.Metadata))
		copy(buf, a.Metadata)
		out.Metadata = buf
	}
	s.m.agents[a.ID.String()] = &out
	return nil
}

func (s *memoryAgentStore) GetByID(_ context.Context, id uuid.UUID) (*model.Agent, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	a, ok := s.m.agents[id.String()]
	if !ok {
		return nil, ErrNotFound
	}
	out := *a
	if a.Capabilities != nil {
		out.Capabilities = append([]string(nil), a.Capabilities...)
	}
	if len(a.Metadata) > 0 {
		buf := make([]byte, len(a.Metadata))
		copy(buf, a.Metadata)
		out.Metadata = buf
	}
	return &out, nil
}

// List implements cursor pagination over the in-memory map. The cursor
// is the agent ID of the last item on the previous page; pages are
// ordered by (created_at, id) for stability. Limit defaults to 50 and
// is capped at 200 (api-spec.md §1.2).
func (s *memoryAgentStore) List(_ context.Context, filter model.AgentFilter) (*model.AgentListResult, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	// ProjectID is required (api-spec.md §1.2). A zero UUID is a
	// service-layer bug; we surface it as "no results" rather than
	// scanning the whole table.
	if filter.ProjectID == uuid.Nil {
		return &model.AgentListResult{Data: []*model.Agent{}, HasMore: false}, nil
	}

	matches := make([]*model.Agent, 0, len(s.m.agents))
	for _, a := range s.m.agents {
		if a.ProjectID != filter.ProjectID {
			continue
		}
		if !filter.IncludeRetired && a.RetiredAt != nil {
			continue
		}
		if filter.Status != "" && a.Status != filter.Status {
			continue
		}
		if filter.Capability != "" && !capabilityMatches(a.Capabilities, filter.Capability) {
			continue
		}
		matches = append(matches, a)
	}

	// Stable order: (created_at, id) — the same key the Postgres
	// impl uses, so service-level behaviour matches across stores.
	sort.Slice(matches, func(i, j int) bool {
		if !matches[i].CreatedAt.Equal(matches[j].CreatedAt) {
			return matches[i].CreatedAt.Before(matches[j].CreatedAt)
		}
		return matches[i].ID.String() < matches[j].ID.String()
	})

	// Apply the cursor (the previous page's last id).
	startIdx := 0
	if filter.Cursor != "" {
		for i, a := range matches {
			if a.ID.String() == filter.Cursor {
				startIdx = i + 1
				break
			}
		}
	}

	endIdx := startIdx + limit
	hasMore := endIdx < len(matches)
	if endIdx > len(matches) {
		endIdx = len(matches)
	}
	page := matches[startIdx:endIdx]

	out := make([]*model.Agent, 0, len(page))
	for _, a := range page {
		c := *a
		if a.Capabilities != nil {
			c.Capabilities = append([]string(nil), a.Capabilities...)
		}
		if len(a.Metadata) > 0 {
			buf := make([]byte, len(a.Metadata))
			copy(buf, a.Metadata)
			out.Metadata = buf
		}
		out = append(out, &c)
	}

	var nextCursor string
	if hasMore && len(out) > 0 {
		nextCursor = out[len(out)-1].ID.String()
	}

	return &model.AgentListResult{
		Data:       out,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (s *memoryAgentStore) Update(_ context.Context, a *model.Agent) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	existing, ok := s.m.agents[a.ID.String()]
	if !ok {
		return ErrNotFound
	}
	if existing.Version != a.Version {
		return ErrConflict
	}
	a.Version++
	if a.CreatedAt.IsZero() {
		a.CreatedAt = existing.CreatedAt
	}
	if a.Capabilities != nil {
		cp := append([]string(nil), a.Capabilities...)
		a.Capabilities = cp
	}
	s.m.agents[a.ID.String()] = a
	return nil
}

func (s *memoryAgentStore) SoftDelete(_ context.Context, id uuid.UUID) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	a, ok := s.m.agents[id.String()]
	if !ok {
		return ErrNotFound
	}
	now := time.Now()
	a.Status = model.AgentRetired
	a.RetiredAt = &now
	a.Version++
	return nil
}

func (s *memoryAgentStore) SetCapabilities(_ context.Context, agentID uuid.UUID, names []string) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	a, ok := s.m.agents[agentID.String()]
	if !ok {
		return ErrNotFound
	}
	a.Capabilities = append([]string(nil), names...)
	a.Version++
	return nil
}

func (s *memoryAgentStore) ListCapabilitiesByAgent(_ context.Context, agentID uuid.UUID) ([]*model.AgentCapability, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	a, ok := s.m.agents[agentID.String()]
	if !ok {
		return nil, ErrNotFound
	}
	out := make([]*model.AgentCapability, 0, len(a.Capabilities))
	for _, name := range a.Capabilities {
		// The memory store does not carry proficiency / granted_at
		// separately from the cache; default to 0 / now. The
		// Postgres impl will join agent_capabilities and surface
		// the real values.
		zero := 0
		out = append(out, &model.AgentCapability{
			Name:        name,
			DisplayName: name,
			Category:    "",
			Proficiency: &zero,
			GrantedAt:   a.CreatedAt,
		})
	}
	return out, nil
}

func capabilityMatches(agentCaps []string, want string) bool {
	for _, c := range agentCaps {
		if c == want {
			return true
		}
	}
	return false
}

// --- Capability Store ---

// memoryCapabilityStore is the in-memory CapabilityStore impl. The
// catalog is read-only; the service layer should not be writing
// capabilities. The seed data is loaded by NewMemoryStore (see the
// capabilities map on memoryStore).
type memoryCapabilityStore struct{ m *memoryStore }

func (s *memoryCapabilityStore) GetByName(_ context.Context, name string) (*model.CapabilityRow, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	c, ok := s.m.capabilities[name]
	if !ok {
		return nil, ErrNotFound
	}
	out := *c
	return &out, nil
}

func (s *memoryCapabilityStore) Exists(_ context.Context, name string) (bool, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	_, ok := s.m.capabilities[name]
	return ok, nil
}

func (s *memoryCapabilityStore) List(_ context.Context, filter model.CapabilityFilter) (*model.CapabilityListResult, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	matches := make([]model.CapabilityRow, 0, len(s.m.capabilities))
	for _, c := range s.m.capabilities {
		if filter.Category != "" && c.Category != filter.Category {
			continue
		}
		matches = append(matches, *c)
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Name < matches[j].Name
	})

	startIdx := 0
	if filter.Cursor != "" {
		for i, c := range matches {
			if c.Name == filter.Cursor {
				startIdx = i + 1
				break
			}
		}
	}
	endIdx := startIdx + limit
	hasMore := endIdx < len(matches)
	if endIdx > len(matches) {
		endIdx = len(matches)
	}
	page := matches[startIdx:endIdx]

	var nextCursor string
	if hasMore && len(page) > 0 {
		nextCursor = page[len(page)-1].Name
	}

	return &model.CapabilityListResult{
		Data:       page,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

// --- Agent Run Store ---

type memoryAgentRunStore struct{ m *memoryStore }

func (s *memoryAgentRunStore) Create(r *model.AgentRun) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, exists := s.m.agentRuns[r.ID.String()]; exists {
		return ErrAlreadyExists
	}
	s.m.agentRuns[r.ID.String()] = r
	return nil
}

func (s *memoryAgentRunStore) GetByID(id uuid.UUID) (*model.AgentRun, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	r, ok := s.m.agentRuns[id.String()]
	if !ok {
		return nil, ErrNotFound
	}
	return r, nil
}

func (s *memoryAgentRunStore) List(filter AgentRunFilter) ([]*model.AgentRun, int, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	var filtered []*model.AgentRun
	for _, r := range s.m.agentRuns {
		if filter.AgentID != uuid.Nil && r.AgentID != filter.AgentID {
			continue
		}
		if filter.TaskID != uuid.Nil {
			if r.TaskID == nil || *r.TaskID != filter.TaskID {
				continue
			}
		}
		if filter.Status != "" && string(r.Status) != filter.Status {
			continue
		}
		filtered = append(filtered, r)
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
	})
	total := len(filtered)
	page, limit := filter.Page, filter.Limit
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	start := (page - 1) * limit
	if start >= len(filtered) {
		return []*model.AgentRun{}, total, nil
	}
	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[start:end], total, nil
}

func (s *memoryAgentRunStore) Update(r *model.AgentRun) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, ok := s.m.agentRuns[r.ID.String()]; !ok {
		return ErrNotFound
	}
	s.m.agentRuns[r.ID.String()] = r
	return nil
}

// --- Execution Store ---
//
// TASK-405 update: the execution store moved from page/limit
// pagination to keyset (cursor) pagination, and the `Update` method
// was narrowed to `UpdateStatus(ctx, id, newStatus, errorMessage)`.
// The store does NOT enforce status transitions — the service layer
// is the source of truth for the state machine.

type memoryExecutionStore struct{ m *memoryStore }

func (s *memoryExecutionStore) Create(_ context.Context, e *model.Execution) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, exists := s.m.executions[e.ExecutionID.String()]; exists {
		return ErrAlreadyExists
	}
	s.m.executions[e.ExecutionID.String()] = e
	return nil
}

func (s *memoryExecutionStore) GetByID(_ context.Context, id uuid.UUID) (*model.Execution, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	e, ok := s.m.executions[id.String()]
	if !ok {
		return nil, ErrNotFound
	}
	return e, nil
}

// List returns a keyset-paginated page of executions matching the
// filter. Sort order is (started_at DESC, execution_id DESC) — the
// execution_id tie-breaker keeps pagination stable when two rows
// share the same started_at timestamp.
//
// Cursor semantics: when filter.Cursor is non-zero, we return rows
// whose (started_at, execution_id) is strictly less than the cursor
// pair. The cursor is the execution_id of the last row in the
// previous page; we look it up to get its started_at. UUIDs sort
// lexicographically as raw 16-byte strings, so execution_id alone
// would be a stable cursor too, but pairing with started_at matches
// the postgres ORDER BY clause and avoids surprises when rows have
// the same execution_id across runs of the test suite.
func (s *memoryExecutionStore) List(_ context.Context, filter model.ExecutionFilter) (*model.ExecutionListResult, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()

	// Resolve the cursor's started_at if a cursor was supplied.
	// A cursor that doesn't resolve to an existing row is treated
	// as "no cursor" — the caller is asking for a page beyond the
	// last one. We don't return an error because that would force
	// every caller to handle ErrNotFound on every List call.
	var cursorStartedAt time.Time
	if filter.Cursor != uuid.Nil {
		if prev, ok := s.m.executions[filter.Cursor.String()]; ok {
			cursorStartedAt = prev.StartedAt
		}
	}

	// First pass: filter (cheap, in memory).
	filtered := make([]*model.Execution, 0, len(s.m.executions))
	for _, e := range s.m.executions {
		if filter.AgentID != uuid.Nil && e.AgentID != filter.AgentID {
			continue
		}
		if filter.TaskID != uuid.Nil && e.TaskID != filter.TaskID {
			continue
		}
		if filter.Status != "" && e.Status != filter.Status {
			continue
		}
		filtered = append(filtered, e)
	}

	// Second pass: apply cursor cutoff (keyset predicate).
	if filter.Cursor != uuid.Nil && !cursorStartedAt.IsZero() {
		cut := filtered[:0]
		for _, e := range filtered {
			// Strictly less than the cursor row.
			if e.StartedAt.Before(cursorStartedAt) {
				cut = append(cut, e)
				continue
			}
			if e.StartedAt.Equal(cursorStartedAt) && bytes.Compare(e.ExecutionID[:], filter.Cursor[:]) < 0 {
				cut = append(cut, e)
				continue
			}
		}
		filtered = cut
	}

	// Sort: started_at DESC, then execution_id DESC for stability.
	sort.Slice(filtered, func(i, j int) bool {
		if !filtered[i].StartedAt.Equal(filtered[j].StartedAt) {
			return filtered[i].StartedAt.After(filtered[j].StartedAt)
		}
		return bytes.Compare(filtered[i].ExecutionID[:], filtered[j].ExecutionID[:]) > 0
	})

	// Page size: clamp to [1, MaxExecutionLimit]; default to
	// DefaultExecutionLimit when the caller didn't ask.
	limit := filter.Limit
	if limit <= 0 {
		limit = model.DefaultExecutionLimit
	}
	if limit > model.MaxExecutionLimit {
		limit = model.MaxExecutionLimit
	}

	// Page slice.
	end := limit
	if end > len(filtered) {
		end = len(filtered)
	}
	page := make([]*model.Execution, end)
	copy(page, filtered[:end])

	// NextCursor: empty when this page exhausted the result set.
	var nextCursor uuid.UUID
	if end < len(filtered) && end > 0 {
		nextCursor = page[end-1].ExecutionID
	}

	return &model.ExecutionListResult{Items: page, NextCursor: nextCursor}, nil
}

// UpdateStatus transitions an execution to newStatus. Sets
// CompletedAt = now() when newStatus is terminal
// (completed/failed) and ErrorMessage when supplied. Returns
// ErrNotFound if the id is unknown. Does NOT validate state
// transitions — the service layer is the source of truth for
// the state machine.
func (s *memoryExecutionStore) UpdateStatus(_ context.Context, id uuid.UUID, newStatus model.ExecutionStatus, errorMessage *string) (*model.Execution, error) {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	e, ok := s.m.executions[id.String()]
	if !ok {
		return nil, ErrNotFound
	}
	e.Status = newStatus
	if newStatus == model.ExecutionStatusCompleted || newStatus == model.ExecutionStatusFailed {
		now := time.Now().UTC()
		e.CompletedAt = &now
	}
	if errorMessage != nil {
		e.ErrorMessage = errorMessage
	} else if newStatus != model.ExecutionStatusFailed {
		// Clear the error message on a non-failure transition
		// (e.g. operator manually flipped failed → running).
		e.ErrorMessage = nil
	}
	e.UpdatedAt = time.Now().UTC()
	return e, nil
}

// --- Deliverable Store ---
//
// TASK-406 update: the deliverable store moved from
// ListByTask/ListByAgent (two single-filter methods) to a single
// List method that takes a DeliverableFilter (combined
// task/agent filter, cursor pagination). The Update method is
// IN-PLACE on the main row; the service layer coordinates with
// memoryDeliverableVersionStore.Insert via WithTx to enforce
// the append-only history invariant (one new deliverable_versions
// row per PUT).

type memoryDeliverableStore struct{ m *memoryStore }

func (s *memoryDeliverableStore) Create(_ context.Context, d *model.Deliverable) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, exists := s.m.deliverables[d.ID.String()]; exists {
		return ErrAlreadyExists
	}
	s.m.deliverables[d.ID.String()] = d
	return nil
}

func (s *memoryDeliverableStore) GetByID(_ context.Context, id uuid.UUID) (*model.Deliverable, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	d, ok := s.m.deliverables[id.String()]
	if !ok {
		return nil, ErrNotFound
	}
	return d, nil
}

// List returns a keyset-paginated page of deliverables matching
// the filter. Sort order is (created_at DESC, id DESC) — the
// id tie-breaker keeps pagination stable when two rows share
// the same created_at timestamp.
//
// Cursor semantics: when filter.Cursor is non-zero, we return
// rows whose (created_at, id) is strictly less than the cursor
// pair. A cursor that doesn't resolve to an existing row is
// treated as "no cursor" (return the first page).
//
// We do NOT enforce "at least one of TaskID/AgentID must be
// set" at the store layer — the service does that. The store
// returns an unfiltered list when both IDs are uuid.Nil.
func (s *memoryDeliverableStore) List(_ context.Context, filter model.DeliverableFilter) (*model.DeliverableListResult, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()

	// Resolve the cursor's created_at if a cursor was supplied.
	var cursorCreatedAt time.Time
	if filter.Cursor != uuid.Nil {
		if prev, ok := s.m.deliverables[filter.Cursor.String()]; ok {
			cursorCreatedAt = prev.CreatedAt
		}
	}

	// First pass: filter.
	filtered := make([]*model.Deliverable, 0, len(s.m.deliverables))
	for _, d := range s.m.deliverables {
		if filter.TaskID != uuid.Nil && d.TaskID != filter.TaskID {
			continue
		}
		if filter.AgentID != uuid.Nil && d.AgentID != filter.AgentID {
			continue
		}
		filtered = append(filtered, d)
	}

	// Second pass: apply cursor cutoff (keyset predicate).
	if filter.Cursor != uuid.Nil && !cursorCreatedAt.IsZero() {
		cut := filtered[:0]
		for _, d := range filtered {
			if d.CreatedAt.Before(cursorCreatedAt) {
				cut = append(cut, d)
				continue
			}
			if d.CreatedAt.Equal(cursorCreatedAt) && bytes.Compare(d.ID[:], filter.Cursor[:]) < 0 {
				cut = append(cut, d)
				continue
			}
		}
		filtered = cut
	}

	// Sort: created_at DESC, then id DESC for stability.
	sort.Slice(filtered, func(i, j int) bool {
		if !filtered[i].CreatedAt.Equal(filtered[j].CreatedAt) {
			return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
		}
		return bytes.Compare(filtered[i].ID[:], filtered[j].ID[:]) > 0
	})

	// Page size: clamp to [1, MaxDeliverableLimit]; default
	// to DefaultDeliverableLimit when the caller didn't ask.
	limit := filter.Limit
	if limit <= 0 {
		limit = model.DefaultDeliverableLimit
	}
	if limit > model.MaxDeliverableLimit {
		limit = model.MaxDeliverableLimit
	}

	end := limit
	if end > len(filtered) {
		end = len(filtered)
	}
	page := make([]*model.Deliverable, end)
	copy(page, filtered[:end])

	// NextCursor: empty when this page exhausted the result set.
	var nextCursor uuid.UUID
	if end < len(filtered) && end > 0 {
		nextCursor = page[end-1].ID
	}

	return &model.DeliverableListResult{Items: page, NextCursor: nextCursor}, nil
}

// Update applies a new state to an existing deliverable in-place.
// The service coordinates with memoryDeliverableVersionStore.Insert
// via WithTx to maintain the append-only history invariant.
func (s *memoryDeliverableStore) Update(_ context.Context, d *model.Deliverable) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, ok := s.m.deliverables[d.ID.String()]; !ok {
		return ErrNotFound
	}
	s.m.deliverables[d.ID.String()] = d
	return nil
}

// --- Deliverable Version Store ---
//
// TASK-406 append-only history. The Insert method enforces the
// (deliverable_id, version) uniqueness invariant (a duplicate
// returns ErrAlreadyExists). ListVersions returns all versions
// for a deliverable, ordered by version DESC — no cursor
// pagination needed (version counts per deliverable are
// expected to be small, typically < 100 in Sprint 4).

type memoryDeliverableVersionStore struct{ m *memoryStore }

func (s *memoryDeliverableVersionStore) Insert(_ context.Context, v *model.DeliverableVersion) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	key := v.DeliverableID.String() + ":" + strconv.Itoa(v.Version)
	if _, exists := s.m.deliverableVersions[key]; exists {
		return ErrAlreadyExists
	}
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	if v.CreatedAt.IsZero() {
		v.CreatedAt = time.Now().UTC()
	}
	s.m.deliverableVersions[key] = v
	s.m.deliverableVersionByDeliverable[v.DeliverableID.String()] = append(
		s.m.deliverableVersionByDeliverable[v.DeliverableID.String()], v.ID.String(),
	)
	return nil
}

func (s *memoryDeliverableVersionStore) ListVersions(_ context.Context, deliverableID uuid.UUID) ([]*model.DeliverableVersion, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	ids := s.m.deliverableVersionByDeliverable[deliverableID.String()]
	if len(ids) == 0 {
		return []*model.DeliverableVersion{}, nil
	}
	out := make([]*model.DeliverableVersion, 0, len(ids))
	for _, id := range ids {
		// Linear scan over deliverableVersions for each id;
		// version counts per deliverable are small so this
		// is fine. If we ever need to optimise, build a
		// second index (id → key).
		for _, v := range s.m.deliverableVersions {
			if v.ID.String() == id {
				out = append(out, v)
				break
			}
		}
	}
	// Sort by version DESC.
	sort.Slice(out, func(i, j int) bool {
		return out[i].Version > out[j].Version
	})
	return out, nil
}

// --- Task Store ---

type memoryTaskStore struct{ m *memoryStore }

func (s *memoryTaskStore) Create(t *model.Task) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, exists := s.m.tasks[t.ID.String()]; exists {
		return ErrAlreadyExists
	}
	s.m.tasks[t.ID.String()] = t
	return nil
}

func (s *memoryTaskStore) GetByID(id uuid.UUID) (*model.Task, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	t, ok := s.m.tasks[id.String()]
	if !ok {
		return nil, ErrNotFound
	}
	return t, nil
}

func (s *memoryTaskStore) List(filter TaskFilter) ([]*model.Task, int, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()

	var filtered []*model.Task
	for _, t := range s.m.tasks {
		if filter.ProjectID != uuid.Nil && t.ProjectID != filter.ProjectID {
			continue
		}
		if filter.Status != "" && t.Status != filter.Status {
			continue
		}
		if filter.AssigneeID != uuid.Nil && t.AssigneeID != filter.AssigneeID {
			continue
		}
		filtered = append(filtered, t)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
	})

	total := len(filtered)
	page, limit := filter.Page, filter.Limit
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	start := (page - 1) * limit
	if start >= len(filtered) {
		return []*model.Task{}, total, nil
	}
	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], total, nil
}

func (s *memoryTaskStore) Update(t *model.Task) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, ok := s.m.tasks[t.ID.String()]; !ok {
		return ErrNotFound
	}
	s.m.tasks[t.ID.String()] = t
	return nil
}

func (s *memoryTaskStore) Delete(id uuid.UUID) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	key := id.String()
	if _, ok := s.m.tasks[key]; !ok {
		return ErrNotFound
	}
	delete(s.m.tasks, key)
	return nil
}

// --- Code Store ---

type memoryCodeStore struct{ m *memoryStore }

func (s *memoryCodeStore) CreateCodeGen(r *model.CodeGenRequest) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, exists := s.m.codeGens[r.ID.String()]; exists {
		return ErrAlreadyExists
	}
	s.m.codeGens[r.ID.String()] = r
	return nil
}

func (s *memoryCodeStore) GetCodeGenByID(id uuid.UUID) (*model.CodeGenRequest, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	r, ok := s.m.codeGens[id.String()]
	if !ok {
		return nil, ErrNotFound
	}
	return r, nil
}

func (s *memoryCodeStore) ListCodeGenByProject(projectID uuid.UUID) ([]*model.CodeGenRequest, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	var result []*model.CodeGenRequest
	for _, r := range s.m.codeGens {
		if r.ProjectID == projectID {
			result = append(result, r)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}

func (s *memoryCodeStore) UpdateCodeGen(r *model.CodeGenRequest) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, ok := s.m.codeGens[r.ID.String()]; !ok {
		return ErrNotFound
	}
	s.m.codeGens[r.ID.String()] = r
	return nil
}

func (s *memoryCodeStore) SaveFile(f *model.ProjectFile) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	key := f.ProjectID.String() + ":" + f.Path // store under project:path key
	s.m.files[key] = f
	return nil
}

func (s *memoryCodeStore) GetFile(projectID uuid.UUID, path string) (*model.ProjectFile, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	key := projectID.String() + ":" + path
	f, ok := s.m.files[key]
	if !ok {
		return nil, ErrNotFound
	}
	return f, nil
}

func (s *memoryCodeStore) ListFiles(projectID uuid.UUID) ([]*model.ProjectFile, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	var result []*model.ProjectFile
	for _, f := range s.m.files {
		if f.ProjectID == projectID {
			result = append(result, f)
		}
	}
	return result, nil
}

func (s *memoryCodeStore) CreateCommit(c *model.Commit) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	key := c.ProjectID.String() + ":" + c.SHA
	if _, exists := s.m.commits[key]; exists {
		return ErrAlreadyExists
	}
	s.m.commits[key] = c
	return nil
}

func (s *memoryCodeStore) GetCommit(projectID uuid.UUID, sha string) (*model.Commit, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	key := projectID.String() + ":" + sha
	c, ok := s.m.commits[key]
	if !ok {
		return nil, ErrNotFound
	}
	return c, nil
}

func (s *memoryCodeStore) ListCommits(projectID uuid.UUID) ([]*model.Commit, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	var result []*model.Commit
	for key, c := range s.m.commits {
		if strings.HasPrefix(key, projectID.String()+":") {
			result = append(result, c)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}

// --- Review Store ---

type memoryReviewStore struct{ m *memoryStore }

func (s *memoryReviewStore) Create(r *model.Review) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, exists := s.m.reviews[r.ID.String()]; exists {
		return ErrAlreadyExists
	}
	s.m.reviews[r.ID.String()] = r
	return nil
}

func (s *memoryReviewStore) GetByID(id uuid.UUID) (*model.Review, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	r, ok := s.m.reviews[id.String()]
	if !ok {
		return nil, ErrNotFound
	}
	return r, nil
}

func (s *memoryReviewStore) ListByProject(projectID uuid.UUID) ([]*model.Review, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	var result []*model.Review
	for _, r := range s.m.reviews {
		if r.ProjectID == projectID {
			result = append(result, r)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}

func (s *memoryReviewStore) Update(r *model.Review) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, ok := s.m.reviews[r.ID.String()]; !ok {
		return ErrNotFound
	}
	s.m.reviews[r.ID.String()] = r
	return nil
}

func (s *memoryReviewStore) CreateComment(c *model.ReviewComment) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, exists := s.m.reviewComments[c.ID.String()]; exists {
		return ErrAlreadyExists
	}
	s.m.reviewComments[c.ID.String()] = c
	return nil
}

func (s *memoryReviewStore) ListComments(reviewID uuid.UUID) ([]*model.ReviewComment, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	var result []*model.ReviewComment
	for _, c := range s.m.reviewComments {
		if c.ReviewID == reviewID {
			result = append(result, c)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}

// --- Deployment Store ---

type memoryDeploymentStore struct{ m *memoryStore }

func (s *memoryDeploymentStore) Create(d *model.Deployment) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, exists := s.m.deployments[d.ID]; exists {
		return ErrAlreadyExists
	}
	s.m.deployments[d.ID] = d
	return nil
}

func (s *memoryDeploymentStore) GetByID(id string) (*model.Deployment, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	d, ok := s.m.deployments[id]
	if !ok {
		return nil, ErrNotFound
	}
	return d, nil
}

func (s *memoryDeploymentStore) ListByProject(projectID string) ([]*model.Deployment, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	var result []*model.Deployment
	for _, d := range s.m.deployments {
		if d.ProjectID == projectID {
			result = append(result, d)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}

func (s *memoryDeploymentStore) Update(d *model.Deployment) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, ok := s.m.deployments[d.ID]; !ok {
		return ErrNotFound
	}
	s.m.deployments[d.ID] = d
	return nil
}

// --- Webhook Store ---

type memoryWebhookStore struct{ m *memoryStore }

func (s *memoryWebhookStore) Create(w *model.Webhook) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, exists := s.m.webhooks[w.ID]; exists {
		return ErrAlreadyExists
	}
	s.m.webhooks[w.ID] = w
	return nil
}

func (s *memoryWebhookStore) GetByID(id string) (*model.Webhook, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	w, ok := s.m.webhooks[id]
	if !ok {
		return nil, ErrNotFound
	}
	return w, nil
}

func (s *memoryWebhookStore) List() ([]*model.Webhook, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	result := make([]*model.Webhook, 0, len(s.m.webhooks))
	for _, w := range s.m.webhooks {
		result = append(result, w)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}

func (s *memoryWebhookStore) Update(w *model.Webhook) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, ok := s.m.webhooks[w.ID]; !ok {
		return ErrNotFound
	}
	s.m.webhooks[w.ID] = w
	return nil
}

func (s *memoryWebhookStore) Delete(id string) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, ok := s.m.webhooks[id]; !ok {
		return ErrNotFound
	}
	delete(s.m.webhooks, id)
	return nil
}

// --- Audit Log Store ---

type memoryAuditLogStore struct{ m *memoryStore }

func (s *memoryAuditLogStore) Create(l *model.AuditLog) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	s.m.auditLogs[l.ID.String()] = l
	return nil
}

	return filtered[start:end], total, nil
}

// --- Assignment Event Store (TASK-404) -------------------------------

// memoryAssignmentEventStore is the in-memory backing for the
// append-only assignment_events table (migration 019). The store is
// intentionally not exposed to callers as a mutable surface — there
// is no Update or Delete. The interface itself has no Update or
// Delete methods; the contract is enforced at compile time.
type memoryAssignmentEventStore struct{ m *memoryStore }

// Append writes a new event row. Mirrors postgresAssignmentEventStore:
//   - ID zero → server generates a UUID.
//   - AssignedAt zero → server defaults to time.Now().UTC().
//   - Action must be in the {assign, reassign, unassign} enum.
//   - TaskID is required.
func (s *memoryAssignmentEventStore) Append(ctx context.Context, ev *model.AssignmentEvent) (*model.AssignmentEvent, error) {
	if ev == nil {
		return nil, ErrInvalidInput
	}
	if ev.TaskID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	if !model.IsValidAssignmentAction(ev.Action) {
		return nil, ErrInvalidInput
	}
	if ev.ID == uuid.Nil {
		ev.ID = uuid.New()
	}
	if ev.AssignedAt.IsZero() {
		ev.AssignedAt = time.Now().UTC()
	}
	// Defensive copy so callers can't mutate the stored event by
	// mutating the input pointer.
	stored := *ev

	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	s.m.assignmentEvents[stored.ID.String()] = &stored
	s.m.assignmentByTask[stored.TaskID.String()] = append(
		s.m.assignmentByTask[stored.TaskID.String()],
		stored.ID.String(),
	)
	return &stored, nil
}

// ListByTask returns all events for a task, newest first. Uses the
// denormalised assignmentByTask index for O(n) scan where n is the
// number of events on that task (not the global event count).
// Returns an empty slice (not nil) when the task has no events.
func (s *memoryAssignmentEventStore) ListByTask(ctx context.Context, taskID uuid.UUID) ([]*model.AssignmentEvent, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()

	ids := s.m.assignmentByTask[taskID.String()]
	if len(ids) == 0 {
		return []*model.AssignmentEvent{}, nil
	}
	events := make([]*model.AssignmentEvent, 0, len(ids))
	for _, id := range ids {
		if ev, ok := s.m.assignmentEvents[id]; ok {
			events = append(events, ev)
		}
	}
	// Newest first: assigned_at DESC, id DESC (matches postgres).
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].AssignedAt.Equal(events[j].AssignedAt) {
			return events[i].ID.String() > events[j].ID.String()
		}
		return events[i].AssignedAt.After(events[j].AssignedAt)
	})
	return events, nil
}

// --- Assignment Store (TASK-404) -------------------------------------

// memoryAssignmentStore is the in-memory backing for the assignments
// table (migration 019). At most one row per task may be active
// (the DB has a partial unique index; the in-memory store enforces
// the same invariant inside the mutex). The store is also
// responsible for maintaining the activeAssignmentByTask index so
// GetActiveByTask is O(1).
type memoryAssignmentStore struct{ m *memoryStore }

// Create inserts a new assignment row. The store auto-fills ID and
// AssignedAt if they are zero. For Sprint 4 the caller is expected
// to set status='active'; any other status is rejected because
// superseded/completed/cancelled are reached via Update, not Create.
func (s *memoryAssignmentStore) Create(ctx context.Context, a *model.Assignment) (*model.Assignment, error) {
	if a == nil {
		return nil, ErrInvalidInput
	}
	if a.TaskID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	if a.AgentID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	if a.Status == "" {
		a.Status = model.AssignmentStatusActive
	}
	if a.Status != model.AssignmentStatusActive {
		return nil, ErrInvalidInput
	}
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	if a.AssignedAt.IsZero() {
		a.AssignedAt = time.Now().UTC()
	}

	s.m.mu.Lock()
	defer s.m.mu.Unlock()

	// Enforce "one active per task" invariant. Mirrors the DB
	// partial unique index uq_assignments_one_active_per_task.
	if existing, ok := s.m.activeAssignmentByTask[a.TaskID.String()]; ok {
		return nil, ErrAlreadyExists
	}

	stored := *a
	s.m.assignments[stored.ID.String()] = &stored
	s.m.activeAssignmentByTask[stored.TaskID.String()] = stored.ID.String()
	return &stored, nil
}

// Update mutates an existing row. Used by the service to flip a
// previous active row to 'superseded' (and set completed_at = now).
// Reactivating a superseded row is allowed only if no other active
// row exists for the task — the invariant is checked atomically
// under the mutex.
func (s *memoryAssignmentStore) Update(ctx context.Context, a *model.Assignment) error {
	if a == nil {
		return ErrInvalidInput
	}
	if a.ID == uuid.Nil {
		return ErrInvalidInput
	}
	if !model.IsValidAssignmentStatus(a.Status) {
		return ErrInvalidInput
	}

	s.m.mu.Lock()
	defer s.m.mu.Unlock()

	existing, ok := s.m.assignments[a.ID.String()]
	if !ok {
		return ErrNotFound
	}

	// If we're flipping to 'active' and there's already an active
	// row for the same task, reject (mirrors the partial unique
	// index invariant).
	if a.Status == model.AssignmentStatusActive {
		if currentActive, hasActive := s.m.activeAssignmentByTask[a.TaskID.String()]; hasActive && currentActive != a.ID.String() {
			return ErrAlreadyExists
		}
	}

	*existing = *a
	// Maintain the activeAssignmentByTask index.
	if existing.Status == model.AssignmentStatusActive {
		s.m.activeAssignmentByTask[existing.TaskID.String()] = existing.ID.String()
	} else {
		if current, ok := s.m.activeAssignmentByTask[existing.TaskID.String()]; ok && current == existing.ID.String() {
			delete(s.m.activeAssignmentByTask, existing.TaskID.String())
		}
	}
	return nil
}

// GetByID returns the assignment by primary key, or ErrNotFound.
func (s *memoryAssignmentStore) GetByID(ctx context.Context, id uuid.UUID) (*model.Assignment, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	a, ok := s.m.assignments[id.String()]
	if !ok {
		return nil, ErrNotFound
	}
	// Defensive copy.
	out := *a
	return &out, nil
}

// GetActiveByTask returns the active row for the task, or
// ErrNotFound. Uses the activeAssignmentByTask index for O(1).
func (s *memoryAssignmentStore) GetActiveByTask(ctx context.Context, taskID uuid.UUID) (*model.Assignment, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	id, ok := s.m.activeAssignmentByTask[taskID.String()]
	if !ok {
		return nil, ErrNotFound
	}
	a, ok := s.m.assignments[id]
	if !ok {
		return nil, ErrNotFound
	}
	out := *a
	return &out, nil
}

// --- memoryTx (TASK-404 WithTx) -------------------------------------

// memoryTx is the in-memory implementation of Tx. The closure
// receives the same shared maps as the parent store, so mutations
// made via the tx-scoped sub-stores are visible to the parent
// store. The mutex already serialises every write, so the closure
// has SQL-transaction-like atomicity without explicit begin/commit.
type memoryTx struct{ m *memoryStore }

func (t *memoryTx) Assignments() AssignmentStore { return &memoryAssignmentStore{t.m} }
func (t *memoryTx) AssignmentEvents() AssignmentEventStore {
	return &memoryAssignmentEventStore{t.m}
}
func (t *memoryTx) Deliverables() DeliverableStore { return &memoryDeliverableStore{t.m} }
func (t *memoryTx) DeliverableVersions() DeliverableVersionStore {
	return &memoryDeliverableVersionStore{t.m}
}

// --- Token Store ---

type memoryTokenStore struct{ m *memoryStore }

func (s *memoryTokenStore) Set(key string, userID uuid.UUID, ttl int) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	s.m.tokens[key] = userID
	return nil
}

func (s *memoryTokenStore) Get(key string) (uuid.UUID, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	userID, ok := s.m.tokens[key]
	if !ok {
		return uuid.Nil, ErrNotFound
	}
	return userID, nil
}

func (s *memoryTokenStore) Delete(key string) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	delete(s.m.tokens, key)
	return nil
}

