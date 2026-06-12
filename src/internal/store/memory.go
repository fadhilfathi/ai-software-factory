package store

import (
	"sort"
	"strings"
	"sync"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
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
	tasks        map[string]*model.Task
	codeGens    map[string]*model.CodeGenRequest
	files       map[string]*model.ProjectFile // key: "projectID:path"
	commits     map[string]*model.Commit      // key: "projectID:sha"
	reviews     map[string]*model.Review
	deployments map[string]*model.Deployment
	webhooks    map[string]*model.Webhook
	auditLogs   map[string]*model.AuditLog
}

// NewMemoryStore creates a new in-memory store ready for use.
func NewMemoryStore() Store {
	return &memoryStore{
		users:       make(map[string]*model.User),
		usersEmail:  make(map[string]*model.User),
		projects:    make(map[string]*model.Project),
		agents:      make(map[string]*model.Agent),
		agentRuns:   make(map[string]*model.AgentRun),
		executions:   make(map[string]*model.Execution),
		deliverables: make(map[string]*model.Deliverable),
		tasks:        make(map[string]*model.Task),
		codeGens:    make(map[string]*model.CodeGenRequest),
		files:       make(map[string]*model.ProjectFile),
		commits:     make(map[string]*model.Commit),
		reviews:     make(map[string]*model.Review),
		deployments: make(map[string]*model.Deployment),
		webhooks:    make(map[string]*model.Webhook),
		auditLogs:   make(map[string]*model.AuditLog),
	}
}

func (m *memoryStore) Users() UserStore             { return &memoryUserStore{m} }
func (m *memoryStore) Projects() ProjectStore        { return &memoryProjectStore{m} }
func (m *memoryStore) Agents() AgentStore             { return &memoryAgentStore{m} }
func (m *memoryStore) AgentRuns() AgentRunStore       { return &memoryAgentRunStore{m} }
func (m *memoryStore) Executions() ExecutionStore     { return &memoryExecutionStore{m} }
func (m *memoryStore) Deliverables() DeliverableStore { return &memoryDeliverableStore{m} }
func (m *memoryStore) Tasks() TaskStore             { return &memoryTaskStore{m} }
func (m *memoryStore) Code() CodeStore              { return &memoryCodeStore{m} }
func (m *memoryStore) Reviews() ReviewStore         { return &memoryReviewStore{m} }
func (m *memoryStore) Deployments() DeploymentStore { return &memoryDeploymentStore{m} }
func (m *memoryStore) Webhooks() WebhookStore       { return &memoryWebhookStore{m} }
func (m *memoryStore) AuditLogs() AuditLogStore     { return &memoryAuditLogStore{m} }

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
			if a.ProjectID == p.ID && a.Status == model.AgentWorking {
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

type memoryAgentStore struct{ m *memoryStore }

func (s *memoryAgentStore) Create(a *model.Agent) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, exists := s.m.agents[a.ID.String()]; exists {
		return ErrAlreadyExists
	}
	s.m.agents[a.ID.String()] = a
	return nil
}

func (s *memoryAgentStore) GetByID(id uuid.UUID) (*model.Agent, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	a, ok := s.m.agents[id.String()]
	if !ok {
		return nil, ErrNotFound
	}
	return a, nil
}

func (s *memoryAgentStore) List(filter AgentFilter) ([]*model.Agent, int, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()

	var filtered []*model.Agent
	for _, a := range s.m.agents {
		if filter.ProjectID != uuid.Nil && a.ProjectID != filter.ProjectID.String() {
			continue
		}
		if filter.Status != "" && a.Status != filter.Status {
			continue
		}
		if filter.Type != "" && a.Type != filter.Type {
			continue
		}
		if filter.Role != "" && a.Role != filter.Role {
			continue
		}
		filtered = append(filtered, a)
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
		return []*model.Agent{}, total, nil
	}
	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], total, nil
}

func (s *memoryAgentStore) Update(a *model.Agent) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, ok := s.m.agents[a.ID.String()]; !ok {
		return ErrNotFound
	}
	s.m.agents[a.ID.String()] = a
	return nil
}

func (s *memoryAgentStore) Delete(id uuid.UUID) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	key := id.String()
	if _, ok := s.m.agents[key]; !ok {
		return ErrNotFound
	}
	delete(s.m.agents, key)
	return nil
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

type memoryExecutionStore struct{ m *memoryStore }

func (s *memoryExecutionStore) Create(e *model.Execution) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, exists := s.m.executions[e.ExecutionID.String()]; exists {
		return ErrAlreadyExists
	}
	s.m.executions[e.ExecutionID.String()] = e
	return nil
}

func (s *memoryExecutionStore) GetByID(id uuid.UUID) (*model.Execution, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	e, ok := s.m.executions[id.String()]
	if !ok {
		return nil, ErrNotFound
	}
	return e, nil
}

func (s *memoryExecutionStore) List(filter ExecutionFilter) ([]*model.Execution, int, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()

	var filtered []*model.Execution
	for _, e := range s.m.executions {
		if filter.AgentID != uuid.Nil && e.AgentID != filter.AgentID {
			continue
		}
		if filter.TaskID != uuid.Nil && e.TaskID != filter.TaskID {
			continue
		}
		if filter.Status != "" && string(e.Status) != filter.Status {
			continue
		}
		filtered = append(filtered, e)
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
		return []*model.Execution{}, total, nil
	}
	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], total, nil
}

func (s *memoryExecutionStore) Update(e *model.Execution) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, ok := s.m.executions[e.ExecutionID.String()]; !ok {
		return ErrNotFound
	}
	s.m.executions[e.ExecutionID.String()] = e
	return nil
}

// --- Deliverable Store ---

type memoryDeliverableStore struct{ m *memoryStore }

func (s *memoryDeliverableStore) Create(d *model.Deliverable) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, exists := s.m.deliverables[d.ID.String()]; exists {
		return ErrAlreadyExists
	}
	s.m.deliverables[d.ID.String()] = d
	return nil
}

func (s *memoryDeliverableStore) GetByID(id uuid.UUID) (*model.Deliverable, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	d, ok := s.m.deliverables[id.String()]
	if !ok {
		return nil, ErrNotFound
	}
	return d, nil
}

func (s *memoryDeliverableStore) ListByTask(taskID uuid.UUID) ([]*model.Deliverable, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	var result []*model.Deliverable
	for _, d := range s.m.deliverables {
		if d.TaskID == taskID {
			result = append(result, d)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}

func (s *memoryDeliverableStore) Update(d *model.Deliverable) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, ok := s.m.deliverables[d.ID.String()]; !ok {
		return ErrNotFound
	}
	s.m.deliverables[d.ID.String()] = d
	return nil
}

func (s *memoryDeliverableStore) ListByAgent(agentID uuid.UUID) ([]*model.Deliverable, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	var result []*model.Deliverable
	for _, d := range s.m.deliverables {
		if d.AgentID == agentID {
			result = append(result, d)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
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
	if _, exists := s.m.codeGens[r.ID]; exists {
		return ErrAlreadyExists
	}
	s.m.codeGens[r.ID] = r
	return nil
}

func (s *memoryCodeStore) GetCodeGenByID(id string) (*model.CodeGenRequest, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	r, ok := s.m.codeGens[id]
	if !ok {
		return nil, ErrNotFound
	}
	return r, nil
}

func (s *memoryCodeStore) ListCodeGenByProject(projectID string) ([]*model.CodeGenRequest, error) {
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
	if _, ok := s.m.codeGens[r.ID]; !ok {
		return ErrNotFound
	}
	s.m.codeGens[r.ID] = r
	return nil
}

func (s *memoryCodeStore) SaveFile(f *model.ProjectFile) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	key := f.ProjectID + ":" + f.Path // store under project:path key
	s.m.files[key] = f
	return nil
}

func (s *memoryCodeStore) GetFile(projectID, path string) (*model.ProjectFile, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	key := projectID + ":" + path
	f, ok := s.m.files[key]
	if !ok {
		return nil, ErrNotFound
	}
	return f, nil
}

func (s *memoryCodeStore) ListFiles(projectID string) ([]*model.ProjectFile, error) {
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
	key := c.ProjectID + ":" + c.SHA
	if _, exists := s.m.commits[key]; exists {
		return ErrAlreadyExists
	}
	s.m.commits[key] = c
	return nil
}

func (s *memoryCodeStore) GetCommit(projectID, sha string) (*model.Commit, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	key := projectID + ":" + sha
	c, ok := s.m.commits[key]
	if !ok {
		return nil, ErrNotFound
	}
	return c, nil
}

func (s *memoryCodeStore) ListCommits(projectID string) ([]*model.Commit, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	var result []*model.Commit
	for key, c := range s.m.commits {
		if strings.HasPrefix(key, projectID+":") {
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
	if _, exists := s.m.reviews[r.ID]; exists {
		return ErrAlreadyExists
	}
	s.m.reviews[r.ID] = r
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

func (s *memoryReviewStore) ListByProject(projectID string) ([]*model.Review, error) {
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
	if _, ok := s.m.reviews[r.ID]; !ok {
		return ErrNotFound
	}
	s.m.reviews[r.ID] = r
	return nil
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

func (s *memoryAuditLogStore) List(filter AuditLogFilter) ([]*model.AuditLog, int, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()

	var filtered []*model.AuditLog
	for _, l := range s.m.auditLogs {
		if filter.EntityType != "" && l.EntityType != filter.EntityType {
			continue
		}
		if filter.EntityID != uuid.Nil && l.EntityID != filter.EntityID {
			continue
		}
		if filter.UserID != uuid.Nil {
			if l.UserID == nil || *l.UserID != filter.UserID {
				continue
			}
		}
		filtered = append(filtered, l)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
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
		return []*model.AuditLog{}, total, nil
	}
	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], total, nil
}

