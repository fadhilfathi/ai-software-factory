package service

import (
	"context"
	"testing"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockStore is a mock implementation of store.Store.
type MockStore struct {
	mock.Mock
}

func (m *MockStore) Users() store.UserStore           { return m.Called().Get(0).(store.UserStore) }
func (m *MockStore) Projects() store.ProjectStore     { return m.Called().Get(0).(store.ProjectStore) }
func (m *MockStore) Agents() store.AgentStore         { return m.Called().Get(0).(store.AgentStore) }
func (m *MockStore) Capabilities() store.CapabilityStore { return m.Called().Get(0).(store.CapabilityStore) }
func (m *MockStore) AgentRuns() store.AgentRunStore   { return m.Called().Get(0).(store.AgentRunStore) }
func (m *MockStore) Executions() store.ExecutionStore { return m.Called().Get(0).(store.ExecutionStore) }
func (m *MockStore) Deliverables() store.DeliverableStore {
	return m.Called().Get(0).(store.DeliverableStore)
}
func (m *MockStore) Tasks() store.TaskStore           { return m.Called().Get(0).(store.TaskStore) }
func (m *MockStore) Code() store.CodeStore            { return m.Called().Get(0).(store.CodeStore) }
func (m *MockStore) Reviews() store.ReviewStore       { return m.Called().Get(0).(store.ReviewStore) }
func (m *MockStore) Deployments() store.DeploymentStore {
	return m.Called().Get(0).(store.DeploymentStore)
}
func (m *MockStore) Webhooks() store.WebhookStore { return m.Called().Get(0).(store.WebhookStore) }
func (m *MockStore) AuditLogs() store.AuditLogStore {
	return m.Called().Get(0).(store.AuditLogStore)
}
func (m *MockStore) Tokens() store.TokenStore { return m.Called().Get(0).(store.TokenStore) }
// TASK-404: append-only assignment_events store. The MockStore
// returns whatever AssignmentEventStore the test wires in via
// .On("AssignmentEvents", ...) — typically a hand-rolled
// in-memory fake for the assignment test file.
func (m *MockStore) AssignmentEvents() store.AssignmentEventStore {
	return m.Called().Get(0).(store.AssignmentEventStore)
}
// TASK-404: current-state assignments store. Same mock pattern.
func (m *MockStore) Assignments() store.AssignmentStore {
	return m.Called().Get(0).(store.AssignmentStore)
}
// TASK-404: WithTx wrapper. The MockStore accepts a closure and
// calls it with a nil Tx (the test uses the real in-memory store
// for end-to-end coverage, not this mock).
func (m *MockStore) WithTx(ctx context.Context, fn func(store.Tx) error) error {
	return m.Called(ctx, fn).Get(0).(error)
}

// MockProjectStore
type MockProjectStore struct {
	mock.Mock
}

func (m *MockProjectStore) Create(p *model.Project) error             { return m.Called(p).Error(0) }
func (m *MockProjectStore) GetByID(id uuid.UUID) (*model.Project, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Project), args.Error(1)
}
func (m *MockProjectStore) List(f store.ProjectFilter) ([]*model.Project, int, error) {
	args := m.Called(f)
	return args.Get(0).([]*model.Project), args.Int(1), args.Error(2)
}
func (m *MockProjectStore) Update(p *model.Project) error { return m.Called(p).Error(0) }
func (m *MockProjectStore) Delete(id uuid.UUID) error    { return m.Called(id).Error(0) }

// MockReviewStore
type MockReviewStore struct {
	mock.Mock
}

func (m *MockReviewStore) Create(r *model.Review) error { return m.Called(r).Error(0) }
func (m *MockReviewStore) GetByID(id uuid.UUID) (*model.Review, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Review), args.Error(1)
}
func (m *MockReviewStore) ListByProject(pID uuid.UUID) ([]*model.Review, error) {
	args := m.Called(pID)
	return args.Get(0).([]*model.Review), args.Error(1)
}
func (m *MockReviewStore) Update(r *model.Review) error { return m.Called(r).Error(0) }
func (m *MockReviewStore) CreateComment(c *model.ReviewComment) error { return m.Called(c).Error(0) }
func (m *MockReviewStore) ListComments(rID uuid.UUID) ([]*model.ReviewComment, error) {
	args := m.Called(rID)
	return args.Get(0).([]*model.ReviewComment), args.Error(1)
}

// MockOrchestrator
type MockOrchestrator struct {
	mock.Mock
}

func (m *MockOrchestrator) StartMonitoring(ctx context.Context)                    { m.Called(ctx) }
func (m *MockOrchestrator) HandleAgentFailure(agentID string) error              { return m.Called(agentID).Error(0) }
func (m *MockOrchestrator) SpawnAgentProcess(ctx context.Context, agent *model.Agent) error {
	return m.Called(ctx, agent).Error(0)
}

func TestReviewService_CreateReview(t *testing.T) {
	log := zap.NewNop()
	pID := uuid.New()
	targetAgentID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		mockStore := new(MockStore)
		mockProjStore := new(MockProjectStore)
		mockRevStore := new(MockReviewStore)
		mockOrch := new(MockOrchestrator)

		mockStore.On("Projects").Return(mockProjStore)
		mockStore.On("Reviews").Return(mockRevStore)

		mockProjStore.On("GetByID", pID).Return(&model.Project{ID: pID}, nil)
		mockRevStore.On("Create", mock.AnythingOfType("*model.Review")).Return(nil)

		svc := NewReviewService(mockStore, mockOrch, log)
		req := CreateReviewRequest{
			ProjectID:     pID.String(),
			CommitSHA:     "sha123",
			ReviewerType:  "agent",
			TargetAgentID: targetAgentID.String(),
		}

		review, err := svc.CreateReview(req)

		assert.Nil(t, err)
		assert.NotNil(t, review)
		assert.Equal(t, pID, review.ProjectID)
		assert.Equal(t, targetAgentID, review.TargetAgentID)
		assert.Equal(t, "sha123", review.CommitSHA)
		assert.Equal(t, model.ReviewInProgress, review.Status)

		// Wait a bit for async processReview to start and potentially call update
		time.Sleep(100 * time.Millisecond)

		mockProjStore.AssertExpectations(t)
		mockRevStore.AssertExpectations(t)
	})

	t.Run("Project Not Found", func(t *testing.T) {
		mockStore := new(MockStore)
		mockProjStore := new(MockProjectStore)
		mockOrch := new(MockOrchestrator)

		mockStore.On("Projects").Return(mockProjStore)
		mockProjStore.On("GetByID", pID).Return(nil, store.ErrNotFound)

		svc := NewReviewService(mockStore, mockOrch, log)
		req := CreateReviewRequest{
			ProjectID:    pID.String(),
			CommitSHA:    "sha123",
			ReviewerType: "agent",
		}

		review, err := svc.CreateReview(req)

		assert.NotNil(t, err)
		assert.Equal(t, 404, err.StatusCode)
		assert.Nil(t, review)
	})

	t.Run("Invalid Project ID", func(t *testing.T) {
		svc := NewReviewService(nil, nil, log)
		req := CreateReviewRequest{
			ProjectID:    "invalid-uuid",
			CommitSHA:    "sha123",
			ReviewerType: "agent",
		}

		review, err := svc.CreateReview(req)

		assert.NotNil(t, err)
		assert.Equal(t, 400, err.StatusCode)
		assert.Nil(t, review)
	})
}
