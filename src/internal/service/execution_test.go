package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/aion"
	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// newExecutionTestService wires a fresh in-memory-backed
// ExecutionService with a fast, deterministic mock goroutine.
//
// The cfg here is the "tests run fast" config: zero sleep (the
// timer fires immediately on the next select tick) and zero
// failure rate. Tests that need a different shape override the
// returned service's cfg directly.
func newExecutionTestService(t *testing.T) (*ExecutionService, store.Store) {
	t.Helper()
	s := store.NewMemoryStore()
	cfg := &ExecutionServiceConfig{
		MockSleep:       func() time.Duration { return 0 },
		MockFailureRate: 0.0,
	}
	svc := NewExecutionService(s, zap.NewNop(), cfg, aion.NewMockRuntime())
	t.Cleanup(func() {
		// Drain in-flight goroutines so they don't leak across
		// tests. We give them a short window; if a test has
		// configured a long sleep, it should call Shutdown
		// explicitly with its own ctx.
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = svc.Shutdown(ctx)
	})
	return svc, s
}

// seedExecutionTaskAndAgent creates a minimal task and agent in
// the in-memory store so CreateExecution's validation passes.
// The agent is created via the AgentService so the capability
// join-table mirror works (TASK-403 contract); the task is
// written directly to keep this helper focused.
// TASK-422: returns (taskID, agentID, projectID) so callers can
// pass task.ProjectID as the callerProjectID on every service call
// (the service enforces the cross-tenant boundary).
func seedExecutionTaskAndAgent(t *testing.T, s store.Store) (uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	ctx := context.Background()

	projectID := uuid.New()
	task := &model.Task{
		ID:        uuid.New(),
		ProjectID: projectID,
		Title:     "exec-test-" + uuid.NewString()[:8],
		Status:    model.TaskOpen,
		Priority:  model.PriorityNormal,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, s.Tasks().Create(task))

	agentSvc := NewAgentService(s)
	created, apiErr := agentSvc.CreateAgent(ctx, CreateAgentRequest{
		ProjectID:    task.ProjectID,
		Name:         "agent-" + uuid.NewString()[:8],
		Role:         "developer",
		Capabilities: []string{"coding"},
	})
	require.Nil(t, apiErr)
	return task.ID, created.ID, projectID
}

// waitForStatus polls GetByID until the execution reaches one of
// the target statuses, or fails the test on timeout. The polling
// interval is 5ms (the in-memory store updates are synchronous;
// the mock goroutine has near-zero sleep in test config).
// TASK-422: projectID is required so the helper can pass the
// cross-tenant boundary on every GetExecution call.
func waitForStatus(t *testing.T, svc *ExecutionService, id, projectID uuid.UUID, targets ...model.ExecutionStatus) *model.Execution {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		e, err := svc.GetExecution(context.Background(), id, projectID)
		if err == nil {
			for _, want := range targets {
				if e.Status == want {
					return e
				}
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	e, _ := svc.GetExecution(context.Background(), id, projectID)
	t.Fatalf("execution %s did not reach any of %v within deadline; last seen status=%v", id, targets, e)
	return nil
}

// ----------------------------------------------------------------------------
// CreateExecution — happy path + validation
// ----------------------------------------------------------------------------

func TestCreateExecution_Success(t *testing.T) {
	svc, s := newExecutionTestService(t)
	taskID, agentID, projectID := seedExecutionTaskAndAgent(t, s)

	exec, err := svc.CreateExecution(context.Background(), taskID, agentID, projectID)
	require.NoError(t, err)
	require.NotNil(t, exec)
	assert.NotEqual(t, uuid.Nil, exec.ExecutionID)
	assert.Equal(t, taskID, exec.TaskID)
	assert.Equal(t, agentID, exec.AgentID)
	assert.Equal(t, model.ExecutionStatusPending, exec.Status)
	assert.Nil(t, exec.CompletedAt)
}

func TestCreateExecution_TaskNotFound(t *testing.T) {
	svc, s := newExecutionTestService(t)
	_, agentID, projectID := seedExecutionTaskAndAgent(t, s)

	exec, err := svc.CreateExecution(context.Background(), uuid.New(), agentID, projectID)
	assert.Nil(t, exec)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrTaskNotFound), "expected ErrTaskNotFound, got %v", err)
}

func TestCreateExecution_AgentNotFound(t *testing.T) {
	svc, s := newExecutionTestService(t)
	taskID, _, projectID := seedExecutionTaskAndAgent(t, s)

	exec, err := svc.CreateExecution(context.Background(), taskID, uuid.New(), projectID)
	assert.Nil(t, exec)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAgentNotFound), "expected ErrAgentNotFound, got %v", err)
}

// ----------------------------------------------------------------------------
// Mock goroutine
// ----------------------------------------------------------------------------

func TestCreateExecution_MockGoroutine_CompletesSuccessfully(t *testing.T) {
	svc, s := newExecutionTestService(t)
	taskID, agentID, projectID := seedExecutionTaskAndAgent(t, s)

	exec, err := svc.CreateExecution(context.Background(), taskID, agentID, projectID)
	require.NoError(t, err)

	// cfg has MockSleep=0 and MockFailureRate=0, so the goroutine
	// should transition to 'completed' within a few ms.
	final := waitForStatus(t, svc, exec.ExecutionID, projectID, model.ExecutionStatusCompleted)
	assert.Equal(t, model.ExecutionStatusCompleted, final.Status)
	assert.NotNil(t, final.CompletedAt)
	assert.Nil(t, final.ErrorMessage)
}

func TestCreateExecution_MockGoroutine_FailsWhenRateIs100(t *testing.T) {
	svc, s := newExecutionTestService(t)
	// TASK-501: the legacy cfg.MockFailureRate knob is now
	// expressed via aion.MockRuntime.SetDefaultScript. We pull
	// the runtime out of the service via the exported (test-only)
	// accessor pattern below. The default-script fallback means
	// the per-spawn ExecutionID doesn't need to be known up-front.
	svc.cfg.MockFailureRate = 1.0 // kept for documentation; no longer read by the runtime path
	mockRT := svc.runtime.(*aion.MockRuntime)
	mockRT.SetDefaultScript(aion.FakeScript{
		Outcome:      aion.WorkerFailed,
		ErrorMessage: "mock failure",
	})
	taskID, agentID, projectID := seedExecutionTaskAndAgent(t, s)

	exec, err := svc.CreateExecution(context.Background(), taskID, agentID, projectID)
	require.NoError(t, err)

	final := waitForStatus(t, svc, exec.ExecutionID, projectID, model.ExecutionStatusFailed)
	assert.Equal(t, model.ExecutionStatusFailed, final.Status)
	assert.NotNil(t, final.CompletedAt)
	require.NotNil(t, final.ErrorMessage)
	assert.Contains(t, *final.ErrorMessage, "mock failure")
}

// ----------------------------------------------------------------------------
// GetExecution
// ----------------------------------------------------------------------------

func TestGetExecution_NotFound(t *testing.T) {
	svc, _ := newExecutionTestService(t)

	exec, err := svc.GetExecution(context.Background(), uuid.New(), uuid.New())
	assert.Nil(t, exec)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrExecutionNotFound), "expected ErrExecutionNotFound, got %v", err)
}

// ----------------------------------------------------------------------------
// ListExecutions — pagination + filter
// ----------------------------------------------------------------------------

func TestListExecutions_Pagination(t *testing.T) {
	svc, s := newExecutionTestService(t)
	taskID, agentID, projectID := seedExecutionTaskAndAgent(t, s)

	// Seed 7 executions.
	for i := 0; i < 7; i++ {
		_, err := svc.CreateExecution(context.Background(), taskID, agentID, projectID)
		require.NoError(t, err)
	}

	// First page: limit=3, no cursor.
	page1, err := svc.ListExecutions(context.Background(), model.ExecutionFilter{Limit: 3}, projectID)
	require.NoError(t, err)
	assert.Len(t, page1.Items, 3)
	assert.NotEqual(t, uuid.Nil, page1.NextCursor, "expected a next cursor after first page")

	// Second page: limit=3, cursor=NextCursor.
	page2, err := svc.ListExecutions(context.Background(), model.ExecutionFilter{Limit: 3, Cursor: page1.NextCursor}, projectID)
	require.NoError(t, err)
	assert.Len(t, page2.Items, 3)
	assert.NotEqual(t, uuid.Nil, page2.NextCursor)

	// Third page: limit=3, cursor=page2.NextCursor → 1 row left, NextCursor=nil.
	page3, err := svc.ListExecutions(context.Background(), model.ExecutionFilter{Limit: 3, Cursor: page2.NextCursor}, projectID)
	require.NoError(t, err)
	assert.Len(t, page3.Items, 1)
	assert.Equal(t, uuid.Nil, page3.NextCursor, "expected empty NextCursor on last page")

	// No duplicate IDs across pages.
	seen := map[uuid.UUID]bool{}
	for _, e := range append(append(page1.Items, page2.Items...), page3.Items...) {
		assert.False(t, seen[e.ExecutionID], "duplicate id %s across pages", e.ExecutionID)
		seen[e.ExecutionID] = true
	}
}

func TestListExecutions_FilterByTaskAgentStatus(t *testing.T) {
	svc, s := newExecutionTestService(t)
	taskA, agentA, projectA := seedExecutionTaskAndAgent(t, s)
	taskB, agentB, _ := seedExecutionTaskAndAgent(t, s)

	// Re-link taskB/agentB to projectA so all 6 executions
	// land in the same project (and the original filter counts
	// hold). Cross-project semantics are covered below.
	require.NoError(t, s.Tasks().Update(&model.Task{ID: taskB, ProjectID: projectA, Title: "relinked", Status: model.TaskOpen, Priority: model.PriorityNormal, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}))
	require.NoError(t, s.Agents().Update(context.Background(), &model.Agent{ID: agentB, ProjectID: projectA, Name: "relinked-b", Role: "developer", Status: model.AgentIdle, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}))

	// 3 on (taskA, agentA), 2 on (taskA, agentB), 1 on (taskB, agentA).
	for i := 0; i < 3; i++ {
		_, err := svc.CreateExecution(context.Background(), taskA, agentA, projectA)
		require.NoError(t, err)
	}
	for i := 0; i < 2; i++ {
		_, err := svc.CreateExecution(context.Background(), taskA, agentB, projectA)
		require.NoError(t, err)
	}
	_, err := svc.CreateExecution(context.Background(), taskB, agentA, projectA)
	require.NoError(t, err)

	// By task
	res, err := svc.ListExecutions(context.Background(), model.ExecutionFilter{TaskID: taskA, Limit: 100}, projectA)
	require.NoError(t, err)
	assert.Len(t, res.Items, 5)

	// By agent
	res, err = svc.ListExecutions(context.Background(), model.ExecutionFilter{AgentID: agentA, Limit: 100}, projectA)
	require.NoError(t, err)
	assert.Len(t, res.Items, 4)

	// By status (mock goroutine fires immediately; all 6 should be 'completed' shortly)
	// We poll once to let the goroutines drain.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		r, _ := svc.ListExecutions(context.Background(), model.ExecutionFilter{Status: model.ExecutionStatusCompleted, Limit: 100}, projectA)
		if len(r.Items) == 6 {
			res = r
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	assert.Len(t, res.Items, 6, "expected all 6 to be completed")

	// Combined: taskA + agentA
	res, err = svc.ListExecutions(context.Background(), model.ExecutionFilter{TaskID: taskA, AgentID: agentA, Limit: 100}, projectA)
	require.NoError(t, err)
	assert.Len(t, res.Items, 3)
}

// ----------------------------------------------------------------------------
// UpdateExecutionStatus — transitions
// ----------------------------------------------------------------------------

func TestUpdateExecutionStatus_ValidTransitions(t *testing.T) {
	svc, s := newExecutionTestService(t)
	taskID, agentID, projectID := seedExecutionTaskAndAgent(t, s)

	exec, err := svc.CreateExecution(context.Background(), taskID, agentID, projectID)
	require.NoError(t, err)

	// pending → running
	updated, err := svc.UpdateExecutionStatus(context.Background(), exec.ExecutionID, model.ExecutionStatusRunning, nil, projectID)
	require.NoError(t, err)
	assert.Equal(t, model.ExecutionStatusRunning, updated.Status)
	assert.Nil(t, updated.CompletedAt)

	// running → completed
	updated, err = svc.UpdateExecutionStatus(context.Background(), exec.ExecutionID, model.ExecutionStatusCompleted, nil, projectID)
	require.NoError(t, err)
	assert.Equal(t, model.ExecutionStatusCompleted, updated.Status)
	require.NotNil(t, updated.CompletedAt)

	// Same-status no-op: completed → completed returns the same row, no error.
	updated2, err := svc.UpdateExecutionStatus(context.Background(), exec.ExecutionID, model.ExecutionStatusCompleted, nil, projectID)
	require.NoError(t, err)
	assert.Equal(t, model.ExecutionStatusCompleted, updated2.Status)
}

func TestUpdateExecutionStatus_InvalidTransition_409(t *testing.T) {
	svc, s := newExecutionTestService(t)
	taskID, agentID, projectID := seedExecutionTaskAndAgent(t, s)

	exec, err := svc.CreateExecution(context.Background(), taskID, agentID, projectID)
	require.NoError(t, err)

	// pending → pending is allowed (idempotent no-op).
	// pending → running is allowed.
	// running → pending is NOT allowed.
	_, err = svc.UpdateExecutionStatus(context.Background(), exec.ExecutionID, model.ExecutionStatusRunning, nil, projectID)
	require.NoError(t, err)
	_, err = svc.UpdateExecutionStatus(context.Background(), exec.ExecutionID, model.ExecutionStatusPending, nil, projectID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidStateTransition), "expected ErrInvalidStateTransition, got %v", err)

	// Terminal → anything is not allowed.
	_, err = svc.UpdateExecutionStatus(context.Background(), exec.ExecutionID, model.ExecutionStatusCompleted, nil, projectID)
	require.NoError(t, err)
	_, err = svc.UpdateExecutionStatus(context.Background(), exec.ExecutionID, model.ExecutionStatusRunning, nil, projectID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidStateTransition), "expected ErrInvalidStateTransition, got %v", err)
}

// ----------------------------------------------------------------------------
// Concurrency
// ----------------------------------------------------------------------------

func TestCreateExecution_ConcurrentCreatesDontRace(t *testing.T) {
	svc, s := newExecutionTestService(t)
	taskID, agentID, projectID := seedExecutionTaskAndAgent(t, s)

	const N = 20
	var wg sync.WaitGroup
	wg.Add(N)
	errs := make(chan error, N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			_, err := svc.CreateExecution(context.Background(), taskID, agentID, projectID)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		assert.NoError(t, err)
	}

	// All N should be readable.
	res, err := svc.ListExecutions(context.Background(), model.ExecutionFilter{TaskID: taskID, Limit: 100}, projectID)
	require.NoError(t, err)
	assert.Len(t, res.Items, N)
}

// ----------------------------------------------------------------------------
// Cancellation
// ----------------------------------------------------------------------------

func TestCreateExecution_MockGoroutine_RespectsShutdown(t *testing.T) {
	// Build a service with a long sleep so the goroutine is
	// guaranteed to be parked on the timer when we call
	// Shutdown.
	s := store.NewMemoryStore()
	cfg := &ExecutionServiceConfig{
		MockSleep:       func() time.Duration { return 10 * time.Second },
		MockFailureRate: 0.0,
	}
	svc := NewExecutionService(s, zap.NewNop(), cfg, aion.NewMockRuntime())

	// Seed task + agent
	task := &model.Task{
		ID: uuid.New(), ProjectID: uuid.New(), Title: "cancel-test",
		Status: model.TaskOpen, Priority: model.PriorityNormal,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, s.Tasks().Create(task))
	agentSvc := NewAgentService(s)
	created, apiErr := agentSvc.CreateAgent(context.Background(), CreateAgentRequest{
		ProjectID: task.ProjectID, Name: "cancel-agent-" + uuid.NewString()[:8],
		Role: "developer", Capabilities: []string{"coding"},
	})
	require.Nil(t, apiErr)

	exec, err := svc.CreateExecution(context.Background(), task.ID, created.ID, task.ProjectID)
	require.NoError(t, err)

	// Give the goroutine a moment to enter the select.
	time.Sleep(50 * time.Millisecond)

	// Shutdown with a generous deadline. The mock goroutine is
	// parked on timer.C; s.stop.Done() will unblock it.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	shutdownErr := svc.Shutdown(ctx)
	require.NoError(t, shutdownErr, "Shutdown should drain the parked goroutine")

	// The row should still be 'pending' — the goroutine never
	// got to write.
	final, err := svc.GetExecution(context.Background(), exec.ExecutionID, task.ProjectID)
	require.NoError(t, err)
	assert.Equal(t, model.ExecutionStatusPending, final.Status, "expected pending (cancelled before transition)")
}

// ----------------------------------------------------------------------------
// Cross-tenant (F-016, TASK-422)
// ----------------------------------------------------------------------------

// mustCreateTaskInProject seeds a minimal task in the given project.
func mustCreateTaskInProject(t *testing.T, s store.Store, projectID uuid.UUID) uuid.UUID {
	t.Helper()
	task := &model.Task{
		ID:        uuid.New(),
		ProjectID: projectID,
		Title:     "xt-task-" + uuid.NewString()[:8],
		Status:    model.TaskOpen,
		Priority:  model.PriorityNormal,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, s.Tasks().Create(task))
	return task.ID
}

// mustCreateAgentInProject seeds an agent in the given project.
func mustCreateAgentInProject(t *testing.T, s store.Store, projectID uuid.UUID) uuid.UUID {
	t.Helper()
	agentSvc := NewAgentService(s)
	created, apiErr := agentSvc.CreateAgent(context.Background(), CreateAgentRequest{
		ProjectID:    projectID,
		Name:         "xt-agent-" + uuid.NewString()[:8],
		Role:         "developer",
		Capabilities: []string{"coding"},
	})
	require.Nil(t, apiErr)
	return created.ID
}

func TestCreateExecution_CrossTenant_TaskInOtherProject_Blocks(t *testing.T) {
	svc, s := newExecutionTestService(t)
	_, _, ownerProject := seedExecutionTaskAndAgent(t, s)
	otherProject := uuid.New()
	otherTask := mustCreateTaskInProject(t, s, otherProject)
	agentInOwner := mustCreateAgentInProject(t, s, ownerProject)

	exec, err := svc.CreateExecution(context.Background(), otherTask, agentInOwner, ownerProject)
	assert.Nil(t, exec)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCrossTenantBlocked), "expected ErrCrossTenantBlocked, got %v", err)
}

func TestCreateExecution_CrossTenant_AgentInOtherProject_Blocks(t *testing.T) {
	svc, s := newExecutionTestService(t)
	_, _, ownerProject := seedExecutionTaskAndAgent(t, s)
	otherProject := uuid.New()
	taskInOwner := mustCreateTaskInProject(t, s, ownerProject)
	agentInOther := mustCreateAgentInProject(t, s, otherProject)

	exec, err := svc.CreateExecution(context.Background(), taskInOwner, agentInOther, ownerProject)
	assert.Nil(t, exec)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCrossTenantBlocked), "expected ErrCrossTenantBlocked, got %v", err)
}

func TestGetExecution_CrossTenant_Blocks(t *testing.T) {
	svc, s := newExecutionTestService(t)
	taskID, agentID, ownerProject := seedExecutionTaskAndAgent(t, s)
	exec, err := svc.CreateExecution(context.Background(), taskID, agentID, ownerProject)
	require.NoError(t, err)

	otherProject := uuid.New()
	got, err := svc.GetExecution(context.Background(), exec.ExecutionID, otherProject)
	assert.Nil(t, got)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCrossTenantBlocked), "expected ErrCrossTenantBlocked, got %v", err)
}

func TestListExecutions_CrossTenant_TaskFilterBlocks(t *testing.T) {
	svc, s := newExecutionTestService(t)
	taskID, agentID, ownerProject := seedExecutionTaskAndAgent(t, s)
	_, err := svc.CreateExecution(context.Background(), taskID, agentID, ownerProject)
	require.NoError(t, err)

	otherProject := uuid.New()
	res, err := svc.ListExecutions(context.Background(), model.ExecutionFilter{TaskID: taskID, Limit: 100}, otherProject)
	assert.Nil(t, res)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCrossTenantBlocked), "expected ErrCrossTenantBlocked, got %v", err)
}

func TestUpdateExecutionStatus_CrossTenant_Blocks(t *testing.T) {
	svc, s := newExecutionTestService(t)
	taskID, agentID, ownerProject := seedExecutionTaskAndAgent(t, s)
	exec, err := svc.CreateExecution(context.Background(), taskID, agentID, ownerProject)
	require.NoError(t, err)

	otherProject := uuid.New()
	updated, err := svc.UpdateExecutionStatus(context.Background(), exec.ExecutionID, model.ExecutionStatusRunning, nil, otherProject)
	assert.Nil(t, updated)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCrossTenantBlocked), "expected ErrCrossTenantBlocked, got %v", err)
}
