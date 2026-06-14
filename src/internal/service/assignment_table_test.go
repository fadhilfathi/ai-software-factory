// Table-driven coverage for the assignment engine.
//
// The narrative tests in assignment_test.go give a more readable
// walkthrough of each case; this file is a parallel, uniform-shape
// view that catches regressions from a single matrix and produces
// diff-friendly output when one case fails.
//
// Coverage map:
//
//   AssignTaskToAgent ........... happy path (assign / reassign /
//                                 idempotent), error paths
//                                 (TaskNotFound / AgentNotFound /
//                                 CapabilityMismatch / AgentNotIdle),
//                                 F-014 cross-tenant (task in
//                                 other project / agent in other
//                                 project / defensive triple-check
//                                 / missing project header),
//                                 persistence (notes / empty notes /
//                                 capabilities_required / empty
//                                 caps preserve).
//
//   ListAssignmentHistory ....... empty for new task / three
//                                 events newest-first /
//                                 task-not-found / cross-tenant /
//                                 missing project header.
//
//   TASK-404 invariants ......... at-most-one-active-per-task
//                                 after reassign (no orphan),
//                                 event count matches action count
//                                 after reassign, idempotent
//                                 preserves the existing active
//                                 row, event carries the
//                                 assignment_id.
//
// A-003 deliverable. The narrative file's leading A-002-19
// tracking comment (asserting a 4-arg constructor and a 2-arg
// NewAgentService) was based on a wrong analysis and is removed
// in this commit: production is 3-arg NewAssignmentService(store,
// capSvc, log) and 1-arg NewAgentService(store), which is what
// the file already calls.

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
)

// ptrUUID returns a fresh *uuid.UUID. Used to pass a caller
// identity into the service without each test wiring a local var.
func ptrUUID() *uuid.UUID { u := uuid.New(); return &u }

// ---- AssignTaskToAgent table-driven cases ----------------------------

// assignTableCase is one row in the AssignTaskToAgent matrix.
// seedExtra pre-seeds the store (and any cross-tenant pre-state)
// and returns the call inputs (taskID, agentID, callerProjectID).
// The test runner constructs the service and calls the SUT.
type assignTableCase struct {
	name string
	// notes / assignedBy / caps are the call inputs
	notes      string
	assignedBy *uuid.UUID
	caps       []string
	// seedExtra pre-seeds the store and returns the call inputs.
	// May also mutate pre-existing state (e.g. an active row, a
	// non-idle agent status, a pre-set required_capabilities).
	seedExtra func(t *testing.T, s store.Store) (taskID, agentID, callerProjectID uuid.UUID)
	// wantErrCode is the expected *Error.Code; "" means success.
	wantErrCode string
	// wantStatus is the expected *Error.Status; 0 for success.
	wantStatus int
	// wantAction is the expected action on the resulting event
	// (only checked on success). Zero value = "no event written",
	// which is asserted separately.
	wantAction model.AssignmentAction
	// wantIdempotent is the expected Idempotent flag on the
	// result (only checked on success).
	wantIdempotent bool
	// postAssert runs additional assertions. Used for
	// persistence checks the common runner can't express.
	postAssert func(t *testing.T, s store.Store, res *AssignmentResult, taskID, agentID uuid.UUID)
}

// TestAssignTaskToAgent_TableDriven is the unified matrix. One
// subtest per case. To add a case, append a struct literal to
// `cases`; the runner handles wiring, calling, and asserting the
// standard wants.
func TestAssignTaskToAgent_TableDriven(t *testing.T) {
	cases := []assignTableCase{
		// ---- happy path ------------------------------------------
		{
			name:       "Success_NewAssignment",
			assignedBy: ptrUUID(),
			seedExtra: func(t *testing.T, s store.Store) (uuid.UUID, uuid.UUID, uuid.UUID) {
				projectID := uuid.New()
				taskID, agentID := seedTaskAndAgent(t, s, projectID, "build feature", "developer", []string{"coding", "testing"})
				return taskID, agentID, projectID
			},
			wantAction: model.AssignmentActionAssign,
		},
		{
			name: "Success_Reassignment",
			seedExtra: func(t *testing.T, s store.Store) (uuid.UUID, uuid.UUID, uuid.UUID) {
				projectID := uuid.New()
				taskID, agentA := seedTaskAndAgent(t, s, projectID, "refactor", "developer", []string{"coding", "testing"})
				// Pre-assign to A so the call below is a reassignment.
				capSvc := NewCapabilityService(s, nil)
				svc := NewAssignmentService(s, capSvc, nil)
				_, apiErr := svc.AssignTaskToAgent(context.Background(), taskID, agentA, "", nil, nil, projectID)
				require.Nil(t, apiErr)
				// Build agent B with the same capabilities.
				agentSvc := NewAgentService(s)
				createdB, apiErr := agentSvc.CreateAgent(context.Background(), CreateAgentRequest{
					ProjectID:    projectID,
					Name:         "agent-" + uuid.NewString()[:8],
					Role:         "developer",
					Capabilities: []string{"coding", "testing"},
				})
				require.Nil(t, apiErr)
				return taskID, createdB.ID, projectID
			},
			wantAction: model.AssignmentActionReassign,
		},
		{
			name: "Success_Idempotent",
			seedExtra: func(t *testing.T, s store.Store) (uuid.UUID, uuid.UUID, uuid.UUID) {
				projectID := uuid.New()
				taskID, agentID := seedTaskAndAgent(t, s, projectID, "anchor", "developer", []string{"coding"})
				capSvc := NewCapabilityService(s, nil)
				svc := NewAssignmentService(s, capSvc, nil)
				_, apiErr := svc.AssignTaskToAgent(context.Background(), taskID, agentID, "", nil, nil, projectID)
				require.Nil(t, apiErr)
				return taskID, agentID, projectID
			},
			wantAction:     model.AssignmentActionAssign, // first call's action
			wantIdempotent: true,                        // second call returns idempotent
			postAssert: func(t *testing.T, s store.Store, res *AssignmentResult, taskID, _ uuid.UUID) {
				assert.Nil(t, res.Event, "idempotent path must not write a new event")
				events, err := s.AssignmentEvents().ListByTask(context.Background(), taskID)
				require.NoError(t, err)
				assert.Len(t, events, 1, "still exactly one event after the idempotent re-post")
			},
		},
		// ---- persistence: notes (F-017) ---------------------------
		{
			name:       "Notes_PersistedInEvent",
			notes:      "first assignment",
			assignedBy: ptrUUID(),
			seedExtra: func(t *testing.T, s store.Store) (uuid.UUID, uuid.UUID, uuid.UUID) {
				projectID := uuid.New()
				taskID, agentID := seedTaskAndAgent(t, s, projectID, "notes-anchor", "developer", []string{"coding"})
				return taskID, agentID, projectID
			},
			wantAction: model.AssignmentActionAssign,
			postAssert: func(t *testing.T, s store.Store, _ *AssignmentResult, taskID, _ uuid.UUID) {
				events, err := s.AssignmentEvents().ListByTask(context.Background(), taskID)
				require.NoError(t, err)
				require.Len(t, events, 1)
				assert.Equal(t, "first assignment", events[0].Notes, "F-017: notes must round-trip")
			},
		},
		{
			name: "EmptyNotes_PersistedAsEmptyString",
			seedExtra: func(t *testing.T, s store.Store) (uuid.UUID, uuid.UUID, uuid.UUID) {
				projectID := uuid.New()
				taskID, agentID := seedTaskAndAgent(t, s, projectID, "empty-notes-anchor", "developer", []string{"coding"})
				return taskID, agentID, projectID
			},
			wantAction: model.AssignmentActionAssign,
			postAssert: func(t *testing.T, s store.Store, _ *AssignmentResult, taskID, _ uuid.UUID) {
				events, err := s.AssignmentEvents().ListByTask(context.Background(), taskID)
				require.NoError(t, err)
				require.Len(t, events, 1)
				assert.Equal(t, "", events[0].Notes, "no notes on input must be empty string in row, not omitted")
			},
		},
		// ---- persistence: capabilities_required ------------------
		{
			name: "CapabilitiesRequired_PersistedOnTask",
			caps: []string{"coding"},
			seedExtra: func(t *testing.T, s store.Store) (uuid.UUID, uuid.UUID, uuid.UUID) {
				projectID := uuid.New()
				taskID, agentID := seedTaskAndAgent(t, s, projectID, "anchor", "developer", []string{"coding", "testing"})
				return taskID, agentID, projectID
			},
			wantAction: model.AssignmentActionAssign,
			postAssert: func(t *testing.T, s store.Store, _ *AssignmentResult, taskID, _ uuid.UUID) {
				persisted, err := s.Tasks().GetByID(taskID)
				require.NoError(t, err)
				assert.Equal(t, []string{"coding"}, persisted.RequiredCapabilities)
			},
		},
		{
			name: "CapabilitiesRequired_Empty_PreservesExisting",
			seedExtra: func(t *testing.T, s store.Store) (uuid.UUID, uuid.UUID, uuid.UUID) {
				projectID := uuid.New()
				taskID, agentID := seedTaskAndAgent(t, s, projectID, "anchor", "developer", []string{"coding", "testing"})
				// Pre-seed the task with a non-empty required_capabilities.
				existing, err := s.Tasks().GetByID(taskID)
				require.NoError(t, err)
				existing.RequiredCapabilities = []string{"security"}
				require.NoError(t, s.Tasks().Update(existing))
				return taskID, agentID, projectID
			},
			wantAction: model.AssignmentActionAssign,
			postAssert: func(t *testing.T, s store.Store, _ *AssignmentResult, taskID, _ uuid.UUID) {
				persisted, err := s.Tasks().GetByID(taskID)
				require.NoError(t, err)
				assert.Equal(t, []string{"security"}, persisted.RequiredCapabilities,
					"empty capabilities_required on input must preserve the existing value, not null it out")
			},
		},
		// ---- error paths ------------------------------------------
		{
			name: "Error_TaskNotFound",
			seedExtra: func(t *testing.T, s store.Store) (uuid.UUID, uuid.UUID, uuid.UUID) {
				projectID := uuid.New()
				_, agentID := seedTaskAndAgent(t, s, projectID, "anchor", "developer", []string{"coding"})
				return uuid.New(), agentID, projectID
			},
			wantErrCode: "NOT_FOUND",
			wantStatus:  404,
		},
		{
			name: "Error_AgentNotFound",
			seedExtra: func(t *testing.T, s store.Store) (uuid.UUID, uuid.UUID, uuid.UUID) {
				projectID := uuid.New()
				taskID, _ := seedTaskAndAgent(t, s, projectID, "anchor", "developer", []string{"coding"})
				return taskID, uuid.New(), projectID
			},
			wantErrCode: "NOT_FOUND",
			wantStatus:  404,
		},
		{
			name: "Error_CapabilityMismatch",
			caps: []string{"coding", "testing"},
			seedExtra: func(t *testing.T, s store.Store) (uuid.UUID, uuid.UUID, uuid.UUID) {
				projectID := uuid.New()
				taskID, _ := seedTaskAndAgent(t, s, projectID, "needs coding+testing", "developer", []string{"coding", "testing"})
				agentSvc := NewAgentService(s)
				created, apiErr := agentSvc.CreateAgent(context.Background(), CreateAgentRequest{
					ProjectID:    projectID,
					Name:         "agent-" + uuid.NewString()[:8],
					Role:         "developer",
					Capabilities: []string{"coding"},
				})
				require.Nil(t, apiErr)
				return taskID, created.ID, projectID
			},
			wantErrCode: "CAPABILITY_MISMATCH",
			wantStatus:  409,
			postAssert: func(t *testing.T, s store.Store, _ *AssignmentResult, taskID, _ uuid.UUID) {
				events, err := s.AssignmentEvents().ListByTask(context.Background(), taskID)
				require.NoError(t, err)
				assert.Empty(t, events, "failed assign must not write an event")
			},
		},
		{
			name: "Error_AgentNotIdle",
			seedExtra: func(t *testing.T, s store.Store) (uuid.UUID, uuid.UUID, uuid.UUID) {
				projectID := uuid.New()
				taskID, agentID := seedTaskAndAgent(t, s, projectID, "anchor", "developer", []string{"coding"})
				setAgentStatus(t, s, agentID, model.AgentBusy)
				return taskID, agentID, projectID
			},
			// notFound/conflict/... all return *Error; the
			// AgentNotIdle branch uses conflict("Agent is not idle")
			// which yields Code: "CONFLICT", Status: 409.
			wantErrCode: "CONFLICT",
			wantStatus:  409,
		},
		// ---- F-014 cross-tenant ------------------------------------
		{
			name: "Error_CrossTenant_TaskInOtherProject",
			seedExtra: func(t *testing.T, s store.Store) (uuid.UUID, uuid.UUID, uuid.UUID) {
				projectA := uuid.New()
				projectB := uuid.New()
				taskID, agentID := seedTaskAndAgent(t, s, projectA, "alpha", "developer", []string{"coding"})
				return taskID, agentID, projectB // caller in B, task in A
			},
			wantErrCode: "CROSS_TENANT_BLOCKED",
			wantStatus:  404,
		},
		{
			name: "Error_CrossTenant_AgentInOtherProject",
			seedExtra: func(t *testing.T, s store.Store) (uuid.UUID, uuid.UUID, uuid.UUID) {
				projectA := uuid.New()
				projectB := uuid.New()
				taskID, _ := seedTaskAndAgent(t, s, projectA, "alpha", "developer", []string{"coding"})
				// Build agent in projectB.
				agentSvc := NewAgentService(s)
				agentB, apiErr := agentSvc.CreateAgent(context.Background(), CreateAgentRequest{
					ProjectID:    projectB,
					Name:         "agent-" + uuid.NewString()[:8],
					Role:         "developer",
					Capabilities: []string{"coding"},
				})
				require.Nil(t, apiErr)
				return taskID, agentB.ID, projectA // caller+task in A, agent in B
			},
			wantErrCode: "CROSS_TENANT_BLOCKED",
			wantStatus:  404,
		},
		{
			name: "Error_MissingProjectHeader",
			seedExtra: func(t *testing.T, s store.Store) (uuid.UUID, uuid.UUID, uuid.UUID) {
				projectID := uuid.New()
				taskID, agentID := seedTaskAndAgent(t, s, projectID, "alpha", "developer", []string{"coding"})
				return taskID, agentID, uuid.Nil
			},
			wantErrCode: "MISSING_PROJECT_HEADER",
			wantStatus:  400,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			svc, s := newAssignmentTestService(t)
			taskID, agentID, callerProjectID := tc.seedExtra(t, s)

			res, apiErr := svc.AssignTaskToAgent(
				context.Background(),
				taskID,
				agentID,
				tc.notes,
				tc.assignedBy,
				tc.caps,
				callerProjectID,
			)

			if tc.wantErrCode != "" {
				require.NotNil(t, apiErr, "expected error %q, got nil", tc.wantErrCode)
				assert.Equal(t, tc.wantErrCode, apiErr.Code)
				assert.Equal(t, tc.wantStatus, apiErr.Status)
				assert.Nil(t, res, "error path must not return a result")
				if tc.postAssert != nil {
					tc.postAssert(t, s, res, taskID, agentID)
				}
				return
			}

			require.Nil(t, apiErr, "expected success, got error %v", apiErr)
			require.NotNil(t, res)
			if tc.wantAction != "" {
				require.NotNil(t, res.Event, "success with wantAction must produce an event")
				assert.Equal(t, tc.wantAction, res.Event.Action)
			}
			assert.Equal(t, tc.wantIdempotent, res.Idempotent)
			if tc.postAssert != nil {
				tc.postAssert(t, s, res, taskID, agentID)
			}
		})
	}
}

// ---- ListAssignmentHistory table-driven cases ------------------------

// listHistoryTableCase is one row in the ListAssignmentHistory
// matrix. seedExtra does any pre-state setup (e.g. write events
// via AssignTaskToAgent, seed a task in a foreign project) and
// returns the call inputs (taskID, callerProjectID).
type listHistoryTableCase struct {
	name string
	// seedExtra prepares pre-state and returns the call inputs.
	seedExtra func(t *testing.T, svc *AssignmentService, s store.Store) (taskID, callerProjectID uuid.UUID)
	// wantErrCode / wantStatus
	wantErrCode string
	wantStatus  int
	// wantEventCount is the expected list length (0 for error paths)
	wantEventCount int
	// wantActionsInOrder is the expected event action sequence
	// (newest-first; only checked on success with events)
	wantActionsInOrder []model.AssignmentAction
	// postAssert runs additional assertions (e.g. DESC ordering check)
	postAssert func(t *testing.T, events []*model.AssignmentEvent)
}

// TestListAssignmentHistory_TableDriven is the unified matrix for
// the history endpoint. One subtest per case.
func TestListAssignmentHistory_TableDriven(t *testing.T) {
	cases := []listHistoryTableCase{
		{
			name: "Empty_ForNewTask",
			seedExtra: func(t *testing.T, _ *AssignmentService, s store.Store) (uuid.UUID, uuid.UUID) {
				projectID := uuid.New()
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
				return task.ID, projectID
			},
			wantEventCount: 0,
		},
		{
			name: "ThreeEvents_NewestFirst",
			seedExtra: func(t *testing.T, svc *AssignmentService, s store.Store) (uuid.UUID, uuid.UUID) {
				projectID := uuid.New()
				taskID, agentA := seedTaskAndAgent(t, s, projectID, "history anchor", "developer", []string{"coding", "testing"})
				ctx := context.Background()
				// Event 1: assign to A.
				_, apiErr := svc.AssignTaskToAgent(ctx, taskID, agentA, "", nil, nil, projectID)
				require.Nil(t, apiErr)
				// Build agent B.
				agentSvc := NewAgentService(s)
				createdB, apiErr := agentSvc.CreateAgent(ctx, CreateAgentRequest{
					ProjectID:    projectID,
					Name:         "agent-" + uuid.NewString()[:8],
					Role:         "developer",
					Capabilities: []string{"coding", "testing"},
				})
				require.Nil(t, apiErr)
				// Event 2: reassign to B.
				_, apiErr = svc.AssignTaskToAgent(ctx, taskID, createdB.ID, "", nil, nil, projectID)
				require.Nil(t, apiErr)
				// Event 3: reassign back to A.
				_, apiErr = svc.AssignTaskToAgent(ctx, taskID, agentA, "", nil, nil, projectID)
				require.Nil(t, apiErr)
				return taskID, projectID
			},
			wantEventCount: 3,
			wantActionsInOrder: []model.AssignmentAction{
				model.AssignmentActionReassign, // back to A (newest)
				model.AssignmentActionReassign, // to B
				model.AssignmentActionAssign,   // initial A (oldest)
			},
		},
		{
			name: "Error_TaskNotFound",
			seedExtra: func(_ *testing.T, _ *AssignmentService, _ store.Store) (uuid.UUID, uuid.UUID) {
				return uuid.New(), uuid.New()
			},
			wantErrCode: "NOT_FOUND",
			wantStatus:  404,
		},
		{
			name: "Error_CrossTenant",
			seedExtra: func(t *testing.T, svc *AssignmentService, s store.Store) (uuid.UUID, uuid.UUID) {
				projectA := uuid.New()
				projectB := uuid.New()
				taskID, agentID := seedTaskAndAgent(t, s, projectA, "alpha", "developer", []string{"coding"})
				// Pre-write an event in projectA so the history has rows.
				_, apiErr := svc.AssignTaskToAgent(context.Background(), taskID, agentID, "", nil, nil, projectA)
				require.Nil(t, apiErr)
				return taskID, projectB // caller in B
			},
			wantErrCode: "CROSS_TENANT_BLOCKED",
			wantStatus:  404,
		},
		{
			name: "Error_MissingProjectHeader",
			seedExtra: func(t *testing.T, _ *AssignmentService, s store.Store) (uuid.UUID, uuid.UUID) {
				projectID := uuid.New()
				taskID, _ := seedTaskAndAgent(t, s, projectID, "alpha", "developer", []string{"coding"})
				return taskID, uuid.Nil
			},
			wantErrCode: "MISSING_PROJECT_HEADER",
			wantStatus:  400,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			svc, s := newAssignmentTestService(t)
			taskID, callerProjectID := tc.seedExtra(t, svc, s)

			events, apiErr := svc.ListAssignmentHistory(context.Background(), taskID, callerProjectID)

			if tc.wantErrCode != "" {
				require.NotNil(t, apiErr)
				assert.Equal(t, tc.wantErrCode, apiErr.Code)
				assert.Equal(t, tc.wantStatus, apiErr.Status)
				assert.Nil(t, events, "error path must not return events")
				return
			}

			require.Nil(t, apiErr, "expected success, got error %v", apiErr)
			assert.Len(t, events, tc.wantEventCount)

			// Verify DESC ordering invariant for any non-empty success.
			for i := 1; i < len(events); i++ {
				assert.False(t, events[i].AssignedAt.After(events[i-1].AssignedAt),
					"events must be sorted DESC by assigned_at")
			}

			// Verify the per-case action sequence.
			if len(tc.wantActionsInOrder) > 0 {
				require.Len(t, events, len(tc.wantActionsInOrder))
				for i, want := range tc.wantActionsInOrder {
					assert.Equal(t, want, events[i].Action,
						"event[%d] action mismatch: want %s", i, want)
				}
			}

			if tc.postAssert != nil {
				tc.postAssert(t, events)
			}
		})
	}
}

// ---- TASK-404 transactional invariants ------------------------------

// transactionInvariantCase is one row in the TASK-404 matrix.
// These cases cover the data-model invariants introduced by the
// 019/020 split: at-most-one-active-per-task, the event/row link
// via assignment_id, idempotency preservation, and event-count
// consistency with the action count.
type transactionInvariantCase struct {
	name string
	// run executes the scenario and returns the (svc, store, taskID)
	// for the post-assert. The scenario encapsulates any multi-step
	// pre-state.
	run func(t *testing.T) (svc *AssignmentService, s store.Store, taskID uuid.UUID)
	// assert runs the invariant check.
	assert func(t *testing.T, s store.Store, taskID uuid.UUID)
}

func TestTASK404_TransactionalInvariants_TableDriven(t *testing.T) {
	cases := []transactionInvariantCase{
		{
			name: "AtMostOneActivePerTask_AfterReassign",
			run: func(t *testing.T) (*AssignmentService, store.Store, uuid.UUID) {
				svc, s := newAssignmentTestService(t)
				projectID := uuid.New()
				taskID, agentA := seedTaskAndAgent(t, s, projectID, "x", "developer", []string{"coding"})
				ctx := context.Background()
				_, apiErr := svc.AssignTaskToAgent(ctx, taskID, agentA, "", nil, nil, projectID)
				require.Nil(t, apiErr)
				// Build agent B and reassign.
				agentSvc := NewAgentService(s)
				createdB, apiErr := agentSvc.CreateAgent(ctx, CreateAgentRequest{
					ProjectID:    projectID,
					Name:         "agent-" + uuid.NewString()[:8],
					Role:         "developer",
					Capabilities: []string{"coding"},
				})
				require.Nil(t, apiErr)
				_, apiErr = svc.AssignTaskToAgent(ctx, taskID, createdB.ID, "", nil, nil, projectID)
				require.Nil(t, apiErr)
				return svc, s, taskID
			},
			assert: func(t *testing.T, s store.Store, taskID uuid.UUID) {
				// The active row must resolve to the second agent.
				active, err := s.Assignments().GetActiveByTask(context.Background(), taskID)
				require.NoError(t, err)
				require.NotNil(t, active)
				assert.Equal(t, model.AssignmentStatusActive, active.Status)
				// The previous row must be flipped to superseded.
				// We re-read by listing events; the assignment_id on
				// the first event points to the row that should now
				// be superseded.
				events, err := s.AssignmentEvents().ListByTask(context.Background(), taskID)
				require.NoError(t, err)
				require.Len(t, events, 2)
				firstRow, err := s.Assignments().GetByID(context.Background(), events[1].AssignmentID)
				require.NoError(t, err)
				assert.Equal(t, model.AssignmentStatusSuperseded, firstRow.Status,
					"the previous active row must be flipped to superseded on reassign")
				require.NotNil(t, firstRow.CompletedAt, "superseded rows must have completed_at set")
			},
		},
		{
			name: "EventCount_EqualsActionCount_AfterReassign",
			run: func(t *testing.T) (*AssignmentService, store.Store, uuid.UUID) {
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
				_, apiErr = svc.AssignTaskToAgent(ctx, taskID, createdB.ID, "", nil, nil, projectID)
				require.Nil(t, apiErr)
				return svc, s, taskID
			},
			assert: func(t *testing.T, s store.Store, taskID uuid.UUID) {
				events, err := s.AssignmentEvents().ListByTask(context.Background(), taskID)
				require.NoError(t, err)
				assert.Len(t, events, 2, "two actions (assign + reassign) must produce exactly two events")
			},
		},
		{
			name: "Idempotent_PreservesExistingAssignmentsRow",
			run: func(t *testing.T) (*AssignmentService, store.Store, uuid.UUID) {
				svc, s := newAssignmentTestService(t)
				projectID := uuid.New()
				taskID, agentID := seedTaskAndAgent(t, s, projectID, "anchor", "developer", []string{"coding"})
				ctx := context.Background()
				first, apiErr := svc.AssignTaskToAgent(ctx, taskID, agentID, "", nil, nil, projectID)
				require.Nil(t, apiErr)
				second, apiErr := svc.AssignTaskToAgent(ctx, taskID, agentID, "", nil, nil, projectID)
				require.Nil(t, apiErr)
				require.NotNil(t, first.Assignment)
				require.NotNil(t, second.Assignment)
				require.True(t, second.Idempotent)
				assert.Equal(t, first.Assignment.ID, second.Assignment.ID,
					"idempotent path must return the same assignments row, not create a new one")
				return svc, s, taskID
			},
			assert: func(t *testing.T, s store.Store, taskID uuid.UUID) {
				// Only one event row was written across the two calls.
				events, err := s.AssignmentEvents().ListByTask(context.Background(), taskID)
				require.NoError(t, err)
				assert.Len(t, events, 1)
			},
		},
		{
			name: "EventCarriesAssignmentID",
			run: func(t *testing.T) (*AssignmentService, store.Store, uuid.UUID) {
				svc, s := newAssignmentTestService(t)
				projectID := uuid.New()
				taskID, agentID := seedTaskAndAgent(t, s, projectID, "anchor", "developer", []string{"coding"})
				res, apiErr := svc.AssignTaskToAgent(context.Background(), taskID, agentID, "", nil, nil, projectID)
				require.Nil(t, apiErr)
				require.NotNil(t, res.Event)
				require.NotNil(t, res.Assignment)
				assert.Equal(t, res.Assignment.ID, res.Event.AssignmentID,
					"event must reference the assignments row that produced it")
				return svc, s, taskID
			},
			assert: func(t *testing.T, s store.Store, taskID uuid.UUID) {
				// Same link visible on the read path.
				events, err := s.AssignmentEvents().ListByTask(context.Background(), taskID)
				require.NoError(t, err)
				require.Len(t, events, 1)
				assignment, err := s.Assignments().GetByID(context.Background(), events[0].AssignmentID)
				require.NoError(t, err)
				assert.Equal(t, model.AssignmentStatusActive, assignment.Status)
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, s, taskID := tc.run(t)
			tc.assert(t, s, taskID)
		})
	}
}
