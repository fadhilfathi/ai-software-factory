package store

import (
	"sort"
	"strings"
	"sync"

	"github.com/example/project/internal/model"
)

// memoryStore implements Store with in-memory maps protected by a mutex.
type memoryStore struct {
	mu         sync.RWMutex
	users      map[string]*model.User
	usersEmail map[string]*model.User
	projects   map[string]*model.Project
	agents     map[string]*model.Agent
	tasks      map[string]*model.Task
	codeGens   map[string]*model.CodeGenRequest
	files      map[string]*model.ProjectFile // key: "projectID:path"
	commits    map[string]*model.Commit      // key: "projectID:sha"
	reviews    map[string]*model.Review
	deployments map[string]*model.Deployment
	webhooks   map[string]*model.Webhook
}

// NewMemoryStore creates a new in-memory store ready for use.
func NewMemoryStore() Store {
	return &memoryStore{
		users:       make(map[string]*model.User),
		usersEmail:  make(map[string]*model.User),
		projects:    make(map[string]*model.Project),
		agents:      make(map[string]*model.Agent),
		tasks:       make(map[string]*model.Task),
		codeGens:    make(map[string]*model.CodeGenRequest),
		files:       make(map[string]*model.ProjectFile),
		commits:     make(map[string]*model.Commit),
		reviews:     make(map[string]*model.Review),
		deployments: make(map[string]*model.Deployment),
		webhooks:    make(map[string]*model.Webhook),
	}
}

func (m *memoryStore) Users() UserStore           { return &memoryUserStore{m} }
func (m *memoryStore) Projects() ProjectStore      { return &memoryProjectStore{m} }
func (m *memoryStore) Agents() AgentStore           { return &memoryAgentStore{m} }
func (m *memoryStore) Tasks() TaskStore             { return &memoryTaskStore{m} }
func (m *memoryStore) Code() CodeStore              { return &memoryCodeStore{m} }
func (m *memoryStore) Reviews() ReviewStore         { return &memoryReviewStore{m} }
func (m *memoryStore) Deployments() DeploymentStore { return &memoryDeploymentStore{m} }
func (m *memoryStore) Webhooks() WebhookStore       { return &memoryWebhookStore{m} }

// --- User Store ---

type memoryUserStore struct{ m *memoryStore }

func (s *memoryUserStore) Create(u *model.User) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, exists := s.m.users[u.ID]; exists {
		return ErrAlreadyExists
	}
	if _, exists := s.m.usersEmail[u.Email]; exists {
		return ErrAlreadyExists
	}
	s.m.users[u.ID] = u
	s.m.usersEmail[u.Email] = u
	return nil
}

func (s *memoryUserStore) GetByID(id string) (*model.User, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	u, ok := s.m.users[id]
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
	if _, ok := s.m.users[u.ID]; !ok {
		return ErrNotFound
	}
	// Update email index if changed
	old, _ := s.m.users[u.ID]
	if old.Email != u.Email {
		delete(s.m.usersEmail, old.Email)
		s.m.usersEmail[u.Email] = u
	}
	s.m.users[u.ID] = u
	return nil
}

// --- Project Store ---

type memoryProjectStore struct{ m *memoryStore }

func (s *memoryProjectStore) Create(p *model.Project) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, exists := s.m.projects[p.ID]; exists {
		return ErrAlreadyExists
	}
	s.m.projects[p.ID] = p
	return nil
}

func (s *memoryProjectStore) GetByID(id string) (*model.Project, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	p, ok := s.m.projects[id]
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
	if _, ok := s.m.projects[p.ID]; !ok {
		return ErrNotFound
	}
	s.m.projects[p.ID] = p
	return nil
}

func (s *memoryProjectStore) Delete(id string) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, ok := s.m.projects[id]; !ok {
		return ErrNotFound
	}
	delete(s.m.projects, id)
	return nil
}

// --- Agent Store ---

type memoryAgentStore struct{ m *memoryStore }

func (s *memoryAgentStore) Create(a *model.Agent) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, exists := s.m.agents[a.ID]; exists {
		return ErrAlreadyExists
	}
	s.m.agents[a.ID] = a
	return nil
}

func (s *memoryAgentStore) GetByID(id string) (*model.Agent, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	a, ok := s.m.agents[id]
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
		if filter.ProjectID != "" && a.ProjectID != filter.ProjectID {
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
	if _, ok := s.m.agents[a.ID]; !ok {
		return ErrNotFound
	}
	s.m.agents[a.ID] = a
	return nil
}

func (s *memoryAgentStore) Delete(id string) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, ok := s.m.agents[id]; !ok {
		return ErrNotFound
	}
	delete(s.m.agents, id)
	return nil
}

// --- Task Store ---

type memoryTaskStore struct{ m *memoryStore }

func (s *memoryTaskStore) Create(t *model.Task) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, exists := s.m.tasks[t.ID]; exists {
		return ErrAlreadyExists
	}
	s.m.tasks[t.ID] = t
	return nil
}

func (s *memoryTaskStore) GetByID(id string) (*model.Task, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	t, ok := s.m.tasks[id]
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
		if filter.ProjectID != "" && t.ProjectID != filter.ProjectID {
			continue
		}
		if filter.Status != "" && t.Status != filter.Status {
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
	if _, ok := s.m.tasks[t.ID]; !ok {
		return ErrNotFound
	}
	s.m.tasks[t.ID] = t
	return nil
}

func (s *memoryTaskStore) Delete(id string) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	if _, ok := s.m.tasks[id]; !ok {
		return ErrNotFound
	}
	delete(s.m.tasks, id)
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
	key := f.ModifiedBy + ":" + f.Path // store under generated key, use project:path
	s.m.files[key] = f
	return nil
}

func (s *memoryCodeStore) GetFile(projectID, path string) (*model.ProjectFile, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	// linear scan — fine for in-memory
	for _, f := range s.m.files {
		if strings.Contains(f.ModifiedBy, projectID) && f.Path == path {
			return f, nil
		}
	}
	return nil, ErrNotFound
}

func (s *memoryCodeStore) ListFiles(projectID string) ([]*model.ProjectFile, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	var result []*model.ProjectFile
	for _, f := range s.m.files {
		if strings.Contains(f.ModifiedBy, projectID) {
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

func (s *memoryReviewStore) GetByID(id string) (*model.Review, error) {
	s.m.mu.RLock()
	defer s.m.mu.RUnlock()
	r, ok := s.m.reviews[id]
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
