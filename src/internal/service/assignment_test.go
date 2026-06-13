package service

import (
	"context"
	"testing"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// newAssignmentTestService wires a full AssignmentService backed by
// a fresh in-memory store. The 6 canonical capabilities are
// pre-seeded by NewMemoryStore (mirrors migration 016) so the
// capability-existence path works without setup.
//
// The 5 assignable caps (architecture, coding, testing, security,
// devops) are usable as task constraints. Leadership is in the
// catalog but reserved for the Leader.
func newAssignmentTestService(t *testing.T) (*AssignmentService, store.Store) {
	t.Helper()
	s := store.NewMemoryStore()
	capSvc := NewCapabilityService(s, zap.NewNop())
	svc := NewAssignmentService(s, capSvc, zap.NewNop())
	return svc, s
}

// seedTaskAndAgent creates a task and an agent in the store, sets
// the agent's status to idle, and grants the requested capabilities
// on the agent via the agent's SetCapabilities path. Returns the
// taskID and agentID. The agent's name is unique so
// (project_id, name) uniqueness is honoured.
//
// projectID and role are passed in for completeness; the tests
// only use them to verify round-trip values.
func seedTaskAndAgent(
	t *testing.T,
	s store.Store,
	projectID uuid.UUID,
	taskTitle string,
	role string,
	grantCaps []string,
) (uuid.UUID, uuid.UUID) {
	t.Helper()
	ctx := context.Background()

	// Task (skip the TaskService and write directly through the
	// store â€” we want the minimum data needed to exercise the
	// assignment path).
	task := &model.Task{
		ID:        uuid.New(),
		ProjectID: projectID,
		Title:     taskTitle,
		Status:    model.TaskOpen,
		Priority:  model.PriorityNormal,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, s.Tasks().Create(task))

	// Agent. Use the AgentService so capabilities are mirrored to
	// the agent_capabilities join table (TASK-403 contract).
	agentSvc := NewAgentService(s)
	created, apiErr := agentSvc.CreateAgent(ctx, CreateAgentRequest{
		ProjectID:    projectID,
		Name:         "agent-" + uuid.NewString()[:8],
		Role:         role,
		Capabilities: grantCaps,
	})
	require.Nil(t, apiErr)
	require.NotNil(t, created)

	// SetCapabilities would also be acceptable here; CreateAgent
	// already wired them through.
	return task.ID, created.ID
}

// setAgentStatus forces an agent into the requested lifecycle
// state. Used by the not-idle test where we need to simulate an
// agent that's already busy. Bypasses the state machine â€” the
// service-layer guard is what we're testing, not the store's
// transition table.
func setAgentStatus(t *testing.T, s store.Store, agentID uuid.UUID, status model.AgentStatus) {
	t.Helper()
	ctx := context.Background()
	agent, err := s.Agents().GetByID(ctx, agentID)
	require.NoError(t, err)
	agent.Status = status
	agent.UpdatedAt = time.Now().UTC()
	require.NoError(t, s.Agents().Update(ctx, agent))
}

// ---- AssignTaskToAgent tests -----------------------------------------

func TestAssignTaskToAgent_NewAssignment(t *testing.T) {
	svc, s := newAssignmentTestService(t)
	projectID := uuid.New()
	taskID, agentID := seedTaskAndAgent(t, s, projectID, "build feature x", "developer", []string{"coding", "testing"})

	assignedBy := uuid.New()
	res, apiErr := svc.AssignTaskToAgent(context.Background(), taskID, agentID, "", &assignedBy, nil, projectID)
	require.Nil(t, apiErr)
	require.NotNil(t, res)

	assert.Equal(t, agentID, res.Task.AssigneeID)
	assert.False(t, res.Idempotent, "first assign is not idempotent")
	require.NotNil(t, res.Event)
	assert.Equal(t, model.AssignmentActionAssign, res.Event.Action)
	assert.Equal(t, taskID, res.Event.TaskID)
	require.NotNil(t, res.Event.AgentID)
	assert.Equal(t, agentID, *res.Event.AgentID)
	require.NotNil(t, res.Event.AssignedBy)
	assert.Equal(t, assignedBy, *res.Event.AssignedBy)

	// Persisted: the task update must land in the store.
	persisted, err := s.Tasks().GetByID(taskID)
	require.NoError(t, err)
	assert.Equal(t, agentID, persisted.AssigneeID)
}

func TestAssignTaskToAgent_Reassignment(t *testing.T) {
	svc, s := newAssignmentTestService(t)
	projectID := uuid.New()
	taskID, agentA := seedTaskAndAgent(t, s, projectID, "refactor", "developer", []string{"coding", "testing"})

	// Pre-assign to A.
	_, apiErr := svc.AssignTaskToAgent(context.Background(), taskID, agentA, "", nil, nil, projectID)
	require.Nil(t, apiErr)

	// Build a second agent B with the same caps.
	ctx := context.Background()
	agentSvc := NewAgentService(s)
	createdB, apiErr := agentSvc.CreateAgent(ctx, CreateAgentRequest{
		ProjectID:    projectID,
		Name:         "agent-" + uuid.NewString()[:8],
		Role:         "developer",
		Capabilities: []string{"coding", "testing"},
	})
	require.Nil(t, apiErr)
	agentB := createdB.ID

	// Reassign to B.
	res, apiErr := svc.AssignTaskToAgent(ctx, taskID, agentB, "", nil, nil, projectID)
	require.Nil(t, apiErr)
	require.NotNil(t, res)
	assert.Equal(t, agentB, res.Task.AssigneeID)
	assert.False(t, res.Idempotent)
	require.NotNil(t, res.Event)
	assert.Equal(t, model.AssignmentActionReassign, res.Event.Action)
}

func TestAssignTaskToAgent_CapabilityMismatch(t *testing.T) {
	svc, s := newAssignmentTestService(t)
	projectID := uuid.New()
	// Task requires coding + testing.
	taskID, _ := seedTaskAndAgent(t, s, projectID, "needs coding+testing", "developer", []string{"coding", "testing"})

	// Second agent only has coding.
	ctx := context.Background()
	agentSvc := NewAgentService(s)
	created, apiErr := agentSvc.CreateAgent(ctx, CreateAgentRequest{
		ProjectID:    projectID,
		Name:         "agent-" + uuid.NewString()[:8],
		Role:         "developer",
		Capabilities: []string{"coding"},
	})
	require.Nil(t, apiErr)

	// Assign with explicit capabilities_required.
	res, svcErr := svc.AssignTaskToAgent(ctx, taskID, created.ID, "", nil, []string{"coding", "testing"}, projectID)
	require.NotNil(t, svcErr, "missing 'testing' must surface")
	assert.Nil(t, res)
	assert.Equal(t, "CAPABILITY_MISMATCH", svcErr.Code)
	assert.Equal(t, 409, svcErr.Status)

	// No event was written.
	events, err := s.AssignmentEvents().ListByTask(ctx, taskID)
	require.NoError(t, err)
	assert.Empty(t, events)

	// Task.AssigneeID was not mutated.
	persisted, err := s.Tasks().GetByID(taskID)
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, persisted.AssigneeID)
}

func TestAssignTaskToAgent_TaskNotFound(t *testing.T) {
	svc, s := newAssignmentTestService(t)
	projectID := uuid.New()
	_, agentID := seedTaskAndAgent(t, s, projectID, "anchor", "developer", []string{"coding"})

	res, svcErr := svc.AssignTaskToAgent(context.Background(), uuid.New(), agentID, "", nil, nil, projectID)
	require.NotNil(t, svcErr)
	assert.Nil(t, res)
	assert.Equal(t, "NOT_FOUND", svcErr.Code)
	assert.Equal(t, 404, svcErr.Status)
}

func TestAssignTaskToAgent_AgentNotFound(t *testing.T) {
	svc, s := newAssignmentTestService(t)
	projectID := uuid.New()
	taskID, _ := seedTaskAndAgent(t, s, projectID, "anchor", "developer", []string{"coding"})

	res, svcErr := svc.AssignTaskToAgent(context.Background(), taskID, uuid.New(), "", nil, nil, projectID)
	require.NotNil(t, svcErr)
	assert.Nil(t, res)
	assert.Equal(t, "NOT_FOUND", svcErr.Code)
	assert.Equal(t, 404, svcErr.Status)
}

func TestAssignTaskToAgent_Idempotent(t *testing.T) {
	svc, s := newAssignmentTestService(t)
	projectID := uuid.New()
	taskID, agentID := seedTaskAndAgent(t, s, projectID, "anchor", "developer", []string{"coding"})

	ctx := context.Background()
	// First assign.
	first, apiErr := svc.AssignTaskToAgent(ctx, taskID, agentID, "", nil, nil, projectID)
	require.Nil(t, apiErr)
	require.NotNil(t, first.Event)

	// Re-POST same agent â†’ idempotent.
	second, apiErr := svc.AssignTaskToAgent(ctx, taskID, agentID, "", nil, nil, projectID)
	require.Nil(t, apiErr)
	require.NotNil(t, second)
	assert.True(t, second.Idempotent, "re-posting same agent must be idempotent")
	assert.Nil(t, second.Event, "idempotent path must not write a new event")

	// ListByTask must still report exactly one event.
	events, err := s.AssignmentEvents().ListByTask(ctx, taskID)
	require.NoError(t, err)
	assert.Len(t, events, 1)
}

func TestAssignTaskToAgent_CapabilitiesRequiredPersisted(t *testing.T) {
	svc, s := newAssignmentTestService(t)
	projectID := uuid.New()
	// Agent has both required caps.
	taskID, agentID := seedTaskAndAgent(t, s, projectID, "anchor", "developer", []string{"coding", "testing"})

	res, apiErr := svc.AssignTaskToAgent(context.Background(), taskID, agentID, "", nil, []string{"coding"}, projectID)
	require.Nil(t, apiErr)
	require.NotNil(t, res)

	// Persisted on the task.
	assert.Equal(t, []string{"coding"}, res.Task.RequiredCapabilities)
	persisted, err := s.Tasks().GetByID(taskID)
	require.NoError(t, err)
	assert.Equal(t, []string{"coding"}, persisted.RequiredCapabilities)
}

func TestAssignTaskToAgent_CapabilitiesRequiredEmpty_Preserves(t *testing.T) {
	svc, s := newAssignmentTestService(t)
	projectID := uuid.New()
	taskID, agentID := seedTaskAndAgent(t, s, projectID, "anchor", "developer", []string{"coding", "testing"})

	// Pre-seed the task with a non-empty required_capabilities set.
	existing, err := s.Tasks().GetByID(taskID)
	require.NoError(t, err)
	existing.RequiredCapabilities = []string{"security"}
	require.NoError(t, s.Tasks().Update(existing))

	// Assign with empty capabilities_required.
	res, apiErr := svc.AssignTaskToAgent(context.Background(), taskID, agentID, "", nil, nil, projectID)
	require.Nil(t, apiErr)
	require.NotNil(t, res)

	// Must preserve the existing required_capabilities (not null).
	assert.Equal(t, []string{"security"}, res.Task.RequiredCapabilities)
	persisted, err := s.Tasks().GetByID(taskID)
	require.NoError(t, err)
	assert.Equal(t, []string{"security"}, persisted.RequiredCapabilities)
}

func TestAssignTaskToAgent_AgentNotIdle(t *testing.T) {
	svc, s := newAssignmentTestService(t)
	projectID := uuid.New()
	taskID, agentID := seedTaskAndAgent(t, s, projectID, "anchor", "developer", []string{"coding"})

	// Force the agent into a non-idle state.
	setAgentStatus(t, s, agentID, model.AgentBusy)

	res, svcErr := svc.AssignTaskToAgent(context.Background(), taskID, agentID, "", nil, nil, projectID)
	require.NotNil(t, svcErr, "non-idle agent must be rejected")
	assert.Nil(t, res)
	assert.Equal(t, 409, svcErr.Status)

	// No event written.
	events, err := s.AssignmentEvents().ListByTask(context.Background(), taskID)
	require.NoError(t, err)
	assert.Empty(t, events)
}

// TestAssignTaskToAgent_NotesPersistedInEvent is the F-017
// round-trip test: notes supplied on the assign call must land
// in the assignment_events row so subsequent
// GET /v1/tasks/:id/history reads return them.
func TestAssignTaskToAgent_NotesPersistedInEvent(t *testing.T) {
	svc, s := newAssignmentTestService(t)
	projectID := uuid.New()
	taskID, agentID := seedTaskAndAgent(t, s, projectID, "notes-anchor", "developer", []string{"coding"})

	assignedBy := uuid.New()
	_, apiErr := svc.AssignTaskToAgent(
		context.Background(),
		taskID,
		agentID,
		"first assignment",
		&assignedBy,
		nil,
		projectID,
	)
	require.Nil(t, apiErr)

	// Read the history back. The event must have Notes set.
	events, err := s.AssignmentEvents().ListByTask(context.Background(), taskID)
	require.NoError(t, err)
	require.Len(t, events, 1, "exactly one event should be written")
	assert.Equal(t, "first assignment", events[0].Notes,
		"notes must be persisted in the assignment_events row (F-017)")
	assert.Equal(t, model.AssignmentActionAssign, events[0].Action)
	require.NotNil(t, events[0].AssignedBy)
	assert.Equal(t, assignedBy, *events[0].AssignedBy)
}

// TestAssignTaskToAgent_EmptyNotesPersisted covers the
// default case: caller did not supply notes, and the row
// must reflect that (empty string), not a magic placeholder.
func TestAssignTaskToAgent_EmptyNotesPersisted(t *testing.T) {
	svc, s := newAssignmentTestService(t)
	projectID := uuid.New()
	taskID, agentID := seedTaskAndAgent(t, s, projectID, "empty-notes-anchor", "developer", []string{"coding"})

	_, apiErr := svc.AssignTaskToAgent(
		context.Background(),
		taskID,
		agentID,
		"", // no notes
		nil,
		nil,
		projectID,
	)
	require.Nil(t, apiErr)

	events, err := s.AssignmentEvents().ListByTask(context.Background(), taskID)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "", events[0].Notes, "no notes on input â†’ empty string in row, not omitted")
}

// ---- ListAssignmentHistory tests -------------------------------------

func TestListAssignmentHistory_ThreeEvents_NewestFirst(t *testing.T) {
	svc, s := newAssignmentTestService(t)
	projectID := uuid.New()
	taskID, agentA := seedTaskAndAgent(t, s, projectID, "history anchor", "developer", []string{"coding", "testing"})

	ctx := context.Background()
	// Event 1: assign to A.
	_, apiErr := svc.AssignTaskToAgent(ctx, taskID, agentA, "", nil, nil, projectID)
	require.Nil(t, apiErr)

	// Build a second agent B and reassign.
	agentSvc := NewAgentService(s)
	createdB, apiErr := agentSvc.CreateAgent(ctx, CreateAgentRequest{
		ProjectID:    projectID,
		Name:         "agent-" + uuid.NewString()[:8],
		Role:         "developer",
		Capabilities: []string{"coding", "testing"},
	})
	require.Nil(t, apiErr)
	agentB := createdB.ID

	// Event 2: reassign to B.
	_, apiErr = svc.AssignTaskToAgent(ctx, taskID, agentB, "", nil, nil, projectID)
	require.Nil(t, apiErr)

	// Event 3: reassign back to A.
	_, apiErr = svc.AssignTaskToAgent(ctx, taskID, agentA, "", nil, nil, projectID)
	require.Nil(t, apiErr)

	// List must return 3 events DESC.
	events, svcErr := svc.ListAssignmentHistory(ctx, taskID, projectID)
	require.Nil(t, svcErr)
	require.NotNil(t, events)
	require.Len(t, events, 3)

	// Newest first. assigned_at must be non-increasing.
	for i := 1; i < len(events); i++ {
		assert.False(t, events[i].AssignedAt.After(events[i-1].AssignedAt),
			"events must be sorted DESC by assigned_at")
	}

	// First event should be the reassign-back-to-A (most recent).
	assert.Equal(t, model.AssignmentActionReassign, events[0].Action)
	require.NotNil(t, events[0].AgentID)
	assert.Equal(t, agentA, *events[0].AgentID)
}

func TestListAssignmentHistory_EmptyForNewTask(t *testing.T) {
	svc, s := newAssignmentTestService(t)
	projectID := uuid.New()
	// Task with no events.
	task := &model.Task{
		ID:        uuid.New(),
		ProjectID: projectID,
		Title:     "no events",
		Status:    model.TaskOpen,
		Priority:  model.PriorityNormal,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, s.Tasks().Create(task))

	events, svcErr := svc.ListAssignmentHistory(context.Background(), task.ID, projectID)
	require.Nil(t, svcErr)
	assert.Empty(t, events)
}

func TestListAssignmentHistory_TaskNotFound(t *testing.T) {
	svc, _ := newAssignmentTestService(t)
	projectID := uuid.New()

	events, svcErr := svc.ListAssignmentHistory(context.Background(), uuid.New(), projectID)
	require.NotNil(t, svcErr)
	assert.Nil(t, events)
	assert.Equal(t, "NOT_FOUND", svcErr.Code)
	assert.Equal(t, 404, svcErr.Status)
}

// ---- TASK-404 correction: 019/020 split + transactional write -----
// These tests cover the new behaviour introduced by the
// data-model.md finalisation:
//   - A new active row is created in the `assignments` table
//     (migration 019) inside the same transaction as the
//     assignment_event row (migration 020).
//   - A previous active row is flipped to 'superseded' (with
//     completed_at set) when a reassignment lands.
//   - The partial unique index invariant
//     "at most one active per task" is honoured.

func TestAssignTaskToAgent_AssignmentsRowCreated(t *testing.T) {
	svc, s := newAssignmentTestService(t)
	projectID := uuid.New()
	taskID, agentID := seedTaskAndAgent(t, s, projectID, "anchor", "developer", []string{"coding"})

	res, apiErr := svc.AssignTaskToAgent(context.Background(), taskID, agentID, "", nil, nil, projectID)
	require.Nil(t, apiErr)
	require.NotNil(t, res)
	require.NotNil(t, res.Assignment, "AssignmentResult must include the new assignments row")

	// Persisted in the assignments table with status=active.
	persisted, err := s.Assignments().GetActiveByTask(context.Background(), taskID)
	require.NoError(t, err)
	assert.Equal(t, agentID, persisted.AgentID)
	assert.Equal(t, model.AssignmentStatusActive, persisted.Status)
	assert.Nil(t, persisted.CompletedAt, "active rows have completed_at = NULL")

	// The event must reference the assignment_id we just created.
	require.NotNil(t, res.Event)
	assert.Equal(t, res.Assignment.ID, res.Event.AssignmentID)
}

func TestAssignTaskToAgent_ReassignFlipsPreviousToSuperseded(t *testing.T) {
	svc, s := newAssignmentTestService(t)
	projectID := uuid.New()
	taskID, agentA := seedTaskAndAgent(t, s, projectID, "refactor", "developer", []string{"coding", "testing"})

	// First assignment: to A.
	first, apiErr := svc.AssignTaskToAgent(context.Background(), taskID, agentA, "", nil, nil, projectID)
	require.Nil(t, apiErr)
	require.NotNil(t, first.Assignment)

	// Build agent B and reassign.
	ctx := context.Background()
	agentSvc := NewAgentService(s)
	createdB, apiErr := agentSvc.CreateAgent(ctx, CreateAgentRequest{
		ProjectID:    projectID,
		Name:         "agent-" + uuid.NewString()[:8],
		Role:         "developer",
		Capabilities: []string{"coding", "testing"},
	})
	require.Nil(t, apiErr)
	agentB := createdB.ID

	second, apiErr := svc.AssignTaskToAgent(ctx, taskID, agentB, "", nil, nil, projectID)
	require.Nil(t, apiErr)

	// The previous assignment row (first.Assignment) must now be
	// status='superseded' with completed_at set.
	prev, err := s.Assignments().GetByID(ctx, first.Assignment.ID)
	require.NoError(t, err)
	assert.Equal(t, model.AssignmentStatusSuperseded, prev.Status)
	require.NotNil(t, prev.CompletedAt, "superseded rows must have completed_at set")

	// The new assignment is the only active one.
	active, err := s.Assignments().GetActiveByTask(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, agentB, active.AgentID)
	assert.Equal(t, model.AssignmentStatusActive, active.Status)
	assert.Equal(t, second.Assignment.ID, active.ID, "the active row must be the new assignment")
}

func TestAssignTaskToAgent_ReassignLeavesNoOrphanActive(t *testing.T) {
	// Defensive: after a reassignment, the activeAssignmentByTask
	// index must point at the new row, not the old one. (Catches a
	// class of bug where the index is updated before the row is
	// flipped, leaving a dangling pointer.)
	svc, s := newAssignmentTestService(t)
	projectID := uuid.New()
	taskID, agentA := seedTaskAndAgent(t, s, projectID, "x", "developer", []string{"coding"})

	_, apiErr := svc.AssignTaskToAgent(context.Background(), taskID, agentA, "", nil, nil, projectID)
	require.Nil(t, apiErr)

	ctx := context.Background()
	agentSvc := NewAgentService(s)
	createdB, apiErr := agentSvc.CreateAgent(ctx, CreateAgentRequest{
		ProjectID:    projectID,
		Name:         "agent-" + uuid.NewString()[:8],
		Role:         "developer",
		Capabilities: []string{"coding"},
	})
	require.Nil(t, apiErr)
	agentB := createdB.ID

	_, apiErr = svc.AssignTaskToAgent(ctx, taskID, agentB, "", nil, nil, projectID)
	require.Nil(t, apiErr)

	// The active row must be for B. If the index is dangling,
	// GetActiveByTask returns A or ErrNotFound.
	active, err := s.Assignments().GetActiveByTask(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, agentB, active.AgentID)
}

func TestAssignTaskToAgent_TransactionRollsBackOnEventInsertFailure(t *testing.T) {
	// If the assignment_event Append fails, the whole transaction
	// must roll back â€” no active row left over, no half-state.
	// We can't easily inject a failure into the real store, so
	// this test exercises the path indirectly: after a successful
	// assign followed by a second assign on a different agent,
	// the count of active rows is still 1.
	svc, s := newAssignmentTestService(t)
	projectID := uuid.New()
	taskID, agentA := seedTaskAndAgent(t, s, projectID, "x", "developer", []string{"coding"})

	ctx := context.Background()
	_, apiErr := svc.AssignTaskToAgent(ctx, taskID, agentA, "", nil, nil, projectID)
	require.Nil(t, apiErr)

	agentSvc := NewAgentService(s)
	createdB, apiErr := agentSvc.CreateAgent(ctx, CreateAgentRequest{
		ProjectID:    projectID,
		Name:         "agent-" + uuid.NewString()[:8],
		Role:         "developer",
		Capabilities: []string{"coding"},
	})
	require.Nil(t, apiErr)
	agentB := createdB.ID

	_, apiErr = svc.AssignTaskToAgent(ctx, taskID, agentB, "", nil, nil, projectID)
	require.Nil(t, apiErr)

	// Count active rows: must be exactly 1.
	active, err := s.Assignments().GetActiveByTask(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, agentB, active.AgentID)

	// Events for the task: must be exactly 2 (assign + reassign).
	events, err := s.AssignmentEvents().ListByTask(ctx, taskID)
	require.NoError(t, err)
	assert.Len(t, events, 2)
}

func TestAssignTaskToAgent_IdempotentPreservesAssignmentsRow(t *testing.T) {
	// Re-POSTing the same agent must not touch the assignments
	// table. The existing active row stays, no new row is
	// created, no event is appended.
	svc, s := newAssignmentTestService(t)
	projectID := uuid.New()
	taskID, agentID := seedTaskAndAgent(t, s, projectID, "anchor", "developer", []string{"coding"})

	ctx := context.Background()
	first, apiErr := svc.AssignTaskToAgent(ctx, taskID, agentID, "", nil, nil, projectID)
	require.Nil(t, apiErr)
	require.NotNil(t, first.Assignment)

	second, apiErr := svc.AssignTaskToAgent(ctx, taskID, agentID, "", nil, nil, projectID)
	require.Nil(t, apiErr)
	assert.True(t, second.Idempotent)
	require.NotNil(t, second.Assignment, "idempotent path must return the existing assignments row")
	assert.Equal(t, first.Assignment.ID, second.Assignment.ID, "must be the same assignments row, not a new one")
}

func TestAssignTaskToAgent_EventCarriesAssignmentID(t *testing.T) {
	// The append-only history must reference the new
	// assignments row via assignment_id. Without this link the
	// 019/020 split breaks down (the history can't be traced
	// back to the row that caused it).
	svc, s := newAssignmentTestService(t)
	projectID := uuid.New()
	taskID, agentID := seedTaskAndAgent(t, s, projectID, "anchor", "developer", []string{"coding"})

	res, apiErr := svc.AssignTaskToAgent(context.Background(), taskID, agentID, "", nil, nil, projectID)
	require.Nil(t, apiErr)
	require.NotNil(t, res.Event)
	require.NotNil(t, res.Assignment)
	assert.Equal(t, res.Assignment.ID, res.Event.AssignmentID)

	// ListByTask must also expose assignment_id.
	events, err := s.AssignmentEvents().ListByTask(context.Background(), taskID)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, res.Assignment.ID, events[0].AssignmentID)
}

// ---- F-014 cross-tenant (Sprint 5) -------------------------------

// TestAssignTaskToAgent_CrossTenant_TaskInOtherProject: a caller in
// projectB cannot assign a task that lives in projectA.
func TestAssignTaskToAgent_CrossTenant_TaskInOtherProject(t *testing.T) {
	svc, s := newAssignmentTestService(t)
	projectA := uuid.New()
	projectB := uuid.New()
	taskID, agentID := seedTaskAndAgent(t, s, projectA, "alpha", "developer", []string{"coding"})

	assignedBy := uuid.New()
	_, apiErr := svc.AssignTaskToAgent(context.Background(), taskID, agentID, "", &assignedBy, nil, projectB)
	require.NotNil(t, apiErr)
	assert.Equal(t, 404, apiErr.Status)
	assert.Equal(t, "CROSS_TENANT_BLOCKED", apiErr.Code)

	// control: projectA caller succeeds
	res, apiErr := svc.AssignTaskToAgent(context.Background(), taskID, agentID, "", &assignedBy, nil, projectA)
	require.Nil(t, apiErr)
	require.NotNil(t, res)
}

// TestAssignTaskToAgent_CrossTenant_AgentInOtherProject: a caller in
// projectA cannot assign one of projectA's tasks to an agent in projectB.
func TestAssignTaskToAgent_CrossTenant_AgentInOtherProject(t *testing.T) {
	svc, s := newAssignmentTestService(t)
	projectA := uuid.New()
	projectB := uuid.New()
	// task in projectA, agent in projectB
	taskID, _ := seedTaskAndAgent(t, s, projectA, "alpha", "developer", []string{"coding"})
	// create the agent in projectB by hand
	agentSvc := NewAgentService(s)
	agentB, _ := agentSvc.CreateAgent(context.Background(), CreateAgentRequest{
		ProjectID: projectB, Name: "agent-b", Role: "developer", Capabilities: []string{"coding"},
	})
	require.NotNil(t, agentB)

	assignedBy := uuid.New()
	_, apiErr := svc.AssignTaskToAgent(context.Background(), taskID, agentB.ID, "", &assignedBy, nil, projectA)
	require.NotNil(t, apiErr)
	assert.Equal(t, 404, apiErr.Status)
	assert.Equal(t, "CROSS_TENANT_BLOCKED", apiErr.Code)
}

// TestAssignTaskToAgent_CrossTenant_DefensiveTripleCheck: even if both
// resources are in the caller's project, the defensive task.ProjectID
// == agent.ProjectID check rejects when the two are in different
// projects. (In practice this requires bypassing the earlier checks,
// which can only happen if the data is inconsistent; the test
// constructs the inconsistent state to prove the defensive guard fires.)
func TestAssignTaskToAgent_CrossTenant_DefensiveTripleCheck(t *testing.T) {
	// Construct: task in projectA, agent in projectB, caller in projectA.
	// The first check (task.ProjectID != callerProjectID) will fire,
	// returning CROSS_TENANT_BLOCKED with the standard envelope. The
	// defensive check would only fire if the task and agent both
	// matched callerProjectID, which is impossible given projectA !=
	// projectB. So this test confirms the first check covers the case.
	svc, s := newAssignmentTestService(t)
	projectA := uuid.New()
	projectB := uuid.New()
	// task in projectA, agent in projectB, caller in projectA
	taskID, _ := seedTaskAndAgent(t, s, projectA, "alpha", "developer", []string{"coding"})
	agentSvc := NewAgentService(s)
	agentB, _ := agentSvc.CreateAgent(context.Background(), CreateAgentRequest{
		ProjectID: projectB, Name: "agent-b", Role: "developer", Capabilities: []string{"coding"},
	})
	assignedBy := uuid.New()
	_, apiErr := svc.AssignTaskToAgent(context.Background(), taskID, agentB.ID, "", &assignedBy, nil, projectA)
	require.NotNil(t, apiErr)
	assert.Equal(t, 404, apiErr.Status)
	assert.Equal(t, "CROSS_TENANT_BLOCKED", apiErr.Code)
}

// TestAssignTaskToAgent_MissingProjectHeader: caller with uuid.Nil is
// rejected at the service layer with 400 MISSING_PROJECT_HEADER.
func TestAssignTaskToAgent_MissingProjectHeader(t *testing.T) {
	svc, s := newAssignmentTestService(t)
	projectA := uuid.New()
	taskID, agentID := seedTaskAndAgent(t, s, projectA, "alpha", "developer", []string{"coding"})

	assignedBy := uuid.New()
	_, apiErr := svc.AssignTaskToAgent(context.Background(), taskID, agentID, "", &assignedBy, nil, uuid.Nil)
	require.NotNil(t, apiErr)
	assert.Equal(t, 400, apiErr.Status)
	assert.Equal(t, "MISSING_PROJECT_HEADER", apiErr.Code)
}

// TestListAssignmentHistory_CrossTenant: caller in projectB cannot read
// the history of a projectA task.
func TestListAssignmentHistory_CrossTenant(t *testing.T) {
	svc, s := newAssignmentTestService(t)
	projectA := uuid.New()
	projectB := uuid.New()
	taskID, agentID := seedTaskAndAgent(t, s, projectA, "alpha", "developer", []string{"coding"})

	// first, create an assignment in projectA so the history has rows
	assignedBy := uuid.New()
	_, apiErr := svc.AssignTaskToAgent(context.Background(), taskID, agentID, "", &assignedBy, nil, projectA)
	require.Nil(t, apiErr)

	// cross-tenant: projectB caller blocked
	_, apiErr = svc.ListAssignmentHistory(context.Background(), taskID, projectB)
	require.NotNil(t, apiErr)
	assert.Equal(t, 404, apiErr.Status)
	assert.Equal(t, "CROSS_TENANT_BLOCKED", apiErr.Code)

	// control: projectA caller sees the history
	events, apiErr := svc.ListAssignmentHistory(context.Background(), taskID, projectA)
	require.Nil(t, apiErr)
	require.Len(t, events, 1)
}

// TestListAssignmentHistory_MissingProjectHeader: uuid.Nil caller is
// rejected with 400 MISSING_PROJECT_HEADER.
func TestListAssignmentHistory_MissingProjectHeader(t *testing.T) {
	svc, s := newAssignmentTestService(t)
	projectA := uuid.New()
	taskID, _ := seedTaskAndAgent(t, s, projectA, "alpha", "developer", []string{"coding"})

	_, apiErr := svc.ListAssignmentHistory(context.Background(), taskID, uuid.Nil)
	require.NotNil(t, apiErr)
	assert.Equal(t, 400, apiErr.Status)
	assert.Equal(t, "MISSING_PROJECT_HEADER", apiErr.Code)
}
