// Package service_test contains the table-driven 6-state execution
// lifecycle test suite for B-001 (commit 4). It complements the
// HTTP-level tests in handler/execution_test.go and the narrative
// state-machine tests in execution_test.go with a focused matrix
// that exhaustively covers the valid edges and the most common
// invalid edges of the 6-state machine.
//
// State machine (recap, see service/execution.go for the canonical
// definition):
//
//   queued    -> assigned, failed
//   assigned  -> running, failed, queued   (operator/recovery return)
//   running   -> review, failed
//   review    -> completed, failed        (the only path into completed)
//   completed -> (terminal)
//   failed    -> (terminal)
//
// Direct running -> completed is intentionally NOT a valid edge.
// The reviewer action (ReviewExecution) is the only path into
// completed; this test suite verifies the edge table above holds.
package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stateEdgeCase is one row of the state-machine matrix. `from` and
// `to` use the model.ExecutionStatus enum; `wantOK` is true iff the
// transition should succeed; `wantErr` is the sentinel error we
// expect on failure. `reason` and `via` are optional inputs:
//
//   - reason: error_message stored on the row when `to` is failed.
//   - via:    the entry-point the test should use. "Update" means
//             UpdateExecutionStatus; "Review" means ReviewExecution
//             (only valid when from=review); "Cancel" means
//             CancelExecution (only valid for non-terminal rows).
type stateEdgeCase struct {
	name    string
	from    model.ExecutionStatus
	to      model.ExecutionStatus
	via     string // "Update" | "Review" | "Cancel"
	reason  string
	wantOK  bool
	wantErr error
}

// TestExecutionStateMachine_TableDriven is the B-001 c4 matrix test.
// It builds a seed execution in each `from` state (via direct
// UpdateExecutionStatus calls), then attempts the transition under
// test and asserts the outcome.
func TestExecutionStateMachine_TableDriven(t *testing.T) {
	t.Parallel()

	// 6-state valid edges. Run a separate sub-test for each so the
	// matrix output is greppable.
	validEdges := []stateEdgeCase{
		// queued -> ...
		{name: "Queued_ToAssigned", from: model.ExecutionStatusQueued, to: model.ExecutionStatusAssigned, via: "Update", wantOK: true},
		{name: "Queued_ToFailed", from: model.ExecutionStatusQueued, to: model.ExecutionStatusFailed, via: "Update", reason: "no agent available", wantOK: true},

		// assigned -> ...
		{name: "Assigned_ToRunning", from: model.ExecutionStatusAssigned, to: model.ExecutionStatusRunning, via: "Update", wantOK: true},
		{name: "Assigned_ToFailed", from: model.ExecutionStatusAssigned, to: model.ExecutionStatusFailed, via: "Update", reason: "agent pre-empted", wantOK: true},
		{name: "Assigned_ToQueued_OperatorReturn", from: model.ExecutionStatusAssigned, to: model.ExecutionStatusQueued, via: "Update", wantOK: true},

		// running -> ...  (running -> completed is INVALID; the
		// reviewer path is the only way in)
		{name: "Running_ToReview", from: model.ExecutionStatusRunning, to: model.ExecutionStatusReview, via: "Update", wantOK: true},
		{name: "Running_ToFailed", from: model.ExecutionStatusRunning, to: model.ExecutionStatusFailed, via: "Update", reason: "worker panic", wantOK: true},

		// review -> ...
		{name: "Review_ToCompleted_Accept", from: model.ExecutionStatusReview, to: model.ExecutionStatusCompleted, via: "Review", wantOK: true},
		{name: "Review_ToFailed_Reject", from: model.ExecutionStatusReview, to: model.ExecutionStatusFailed, via: "Review", reason: "output not as spec'd", wantOK: true},

		// idempotent no-op on assigned (carried over from Sprint 5)
		{name: "Assigned_ToAssigned_Idempotent", from: model.ExecutionStatusAssigned, to: model.ExecutionStatusAssigned, via: "Update", wantOK: true},
	}

	invalidEdges := []stateEdgeCase{
		// Running -> Completed is the headline INVALID edge of B-001.
		// The reviewer action is the only path into completed.
		{name: "Running_ToCompleted_BLOCKED", from: model.ExecutionStatusRunning, to: model.ExecutionStatusCompleted, via: "Update", wantOK: false, wantErr: ErrInvalidStateTransition},

		// Queued cannot jump to running (must go through assigned first)
		{name: "Queued_ToRunning_BLOCKED", from: model.ExecutionStatusQueued, to: model.ExecutionStatusRunning, via: "Update", wantOK: false, wantErr: ErrInvalidStateTransition},
		{name: "Queued_ToReview_BLOCKED", from: model.ExecutionStatusQueued, to: model.ExecutionStatusReview, via: "Update", wantOK: false, wantErr: ErrInvalidStateTransition},
		{name: "Queued_ToCompleted_BLOCKED", from: model.ExecutionStatusQueued, to: model.ExecutionStatusCompleted, via: "Update", wantOK: false, wantErr: ErrInvalidStateTransition},

		// Assigned cannot go directly to review or completed
		{name: "Assigned_ToReview_BLOCKED", from: model.ExecutionStatusAssigned, to: model.ExecutionStatusReview, via: "Update", wantOK: false, wantErr: ErrInvalidStateTransition},
		{name: "Assigned_ToCompleted_BLOCKED", from: model.ExecutionStatusAssigned, to: model.ExecutionStatusCompleted, via: "Update", wantOK: false, wantErr: ErrInvalidStateTransition},

		// Running cannot go back to assigned or queued
		{name: "Running_ToAssigned_BLOCKED", from: model.ExecutionStatusRunning, to: model.ExecutionStatusAssigned, via: "Update", wantOK: false, wantErr: ErrInvalidStateTransition},
		{name: "Running_ToQueued_BLOCKED", from: model.ExecutionStatusRunning, to: model.ExecutionStatusQueued, via: "Update", wantOK: false, wantErr: ErrInvalidStateTransition},

		// Review cannot go back to running or assigned
		{name: "Review_ToRunning_BLOCKED", from: model.ExecutionStatusReview, to: model.ExecutionStatusRunning, via: "Update", wantOK: false, wantErr: ErrInvalidStateTransition},
		{name: "Review_ToAssigned_BLOCKED", from: model.ExecutionStatusReview, to: model.ExecutionStatusAssigned, via: "Update", wantOK: false, wantErr: ErrInvalidStateTransition},
		{name: "Review_ToQueued_BLOCKED", from: model.ExecutionStatusReview, to: model.ExecutionStatusQueued, via: "Update", wantOK: false, wantErr: ErrInvalidStateTransition},

		// Terminal states are sticky
		{name: "Completed_ToAnything_BLOCKED", from: model.ExecutionStatusCompleted, to: model.ExecutionStatusFailed, via: "Update", wantOK: false, wantErr: ErrInvalidStateTransition},
		{name: "Failed_ToAnything_BLOCKED", from: model.ExecutionStatusFailed, to: model.ExecutionStatusCompleted, via: "Update", wantOK: false, wantErr: ErrInvalidStateTransition},
		{name: "Completed_ToCancel_BLOCKED", from: model.ExecutionStatusCompleted, to: model.ExecutionStatusFailed, via: "Cancel", wantOK: false, wantErr: ErrInvalidStateTransition},
		{name: "Failed_ToCancel_BLOCKED", from: model.ExecutionStatusFailed, to: model.ExecutionStatusFailed, via: "Cancel", wantOK: false, wantErr: ErrInvalidStateTransition},
	}

	allCases := append(validEdges, invalidEdges...)

	for _, tc := range allCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, s := newExecutionTestService(t)
			ctx := context.Background()

			// Seed: create a row and drive it into the `from` state.
			taskID, agentID, projectID := seedExecutionTaskAndAgent(t, s)
			exec, err := svc.CreateExecution(ctx, taskID, agentID, projectID)
			require.NoError(t, err)
			require.NoError(t, driveToState(t, svc, exec.ExecutionID, projectID, tc.from))

			// Attempt the transition under test.
			var (
				gotErr error
				after  *model.Execution
			)
			switch tc.via {
			case "Review":
				accepted := tc.to == model.ExecutionStatusCompleted
				after, gotErr = svc.ReviewExecution(ctx, exec.ExecutionID, accepted, tc.reason, projectID)
				if gotErr == nil {
					assert.Equal(t, tc.to, after.Status, "ReviewAction.To mismatch")
				}
			case "Cancel":
				gotErr = svc.CancelExecution(ctx, exec.ExecutionID, projectID)
				if gotErr == nil {
					final, ferr := svc.GetExecution(ctx, exec.ExecutionID, projectID)
					require.NoError(t, ferr)
					assert.Equal(t, model.ExecutionStatusFailed, final.Status)
					require.NotNil(t, final.ErrorMessage)
					assert.Equal(t, "cancelled by operator", *final.ErrorMessage)
				}
			default: // "Update"
				var msg *string
				if tc.reason != "" {
					r := tc.reason
					msg = &r
				}
				after, gotErr = svc.UpdateExecutionStatus(ctx, exec.ExecutionID, tc.to, msg, projectID)
				if gotErr == nil {
					assert.Equal(t, tc.to, after.Status)
				}
			}

			// Assert outcome.
			if tc.wantOK {
				require.NoError(t, gotErr, "expected transition %s -> %s to succeed", tc.from, tc.to)
				// If we transitioned to failed with a reason, verify the reason persisted.
				if tc.to == model.ExecutionStatusFailed && tc.reason != "" && tc.via == "Update" {
					final, _ := svc.GetExecution(ctx, exec.ExecutionID, projectID)
					require.NotNil(t, final.ErrorMessage)
					assert.Equal(t, tc.reason, *final.ErrorMessage)
				}
				// If we transitioned to completed, verify CompletedAt was set.
				if tc.to == model.ExecutionStatusCompleted {
					final, _ := svc.GetExecution(ctx, exec.ExecutionID, projectID)
					assert.NotNil(t, final.CompletedAt, "CompletedAt must be set on terminal Completed")
				}
			} else {
				require.Error(t, gotErr, "expected transition %s -> %s to fail", tc.from, tc.to)
				if tc.wantErr != nil {
					assert.True(t, errors.Is(gotErr, tc.wantErr),
						"expected error chain to contain %v, got %v", tc.wantErr, gotErr)
				}
			}
		})
	}
}

// driveToState moves an execution from the post-create default
// (assigned) to the requested state using only the valid edges.
// `from = assigned` is the trivial case (no-op).
//
// Uses only the state machine's valid forward edges so a test that
// asks for `from = review` always traverses assigned -> running ->
// review, never any INVALID edge.
func driveToState(t *testing.T, svc *ExecutionService, execID, projectID uuid.UUID, target model.ExecutionStatus) error {
	t.Helper()
	ctx := context.Background()

	if target == model.ExecutionStatusAssigned {
		return nil
	}

	switch target {
	case model.ExecutionStatusQueued:
		_, err := svc.UpdateExecutionStatus(ctx, execID, model.ExecutionStatusQueued, nil, projectID)
		return err
	case model.ExecutionStatusRunning:
		_, err := svc.UpdateExecutionStatus(ctx, execID, model.ExecutionStatusRunning, nil, projectID)
		return err
	case model.ExecutionStatusReview:
		if _, err := svc.UpdateExecutionStatus(ctx, execID, model.ExecutionStatusRunning, nil, projectID); err != nil {
			return err
		}
		_, err := svc.UpdateExecutionStatus(ctx, execID, model.ExecutionStatusReview, nil, projectID)
		return err
	case model.ExecutionStatusCompleted:
		if _, err := svc.UpdateExecutionStatus(ctx, execID, model.ExecutionStatusRunning, nil, projectID); err != nil {
			return err
		}
		if _, err := svc.UpdateExecutionStatus(ctx, execID, model.ExecutionStatusReview, nil, projectID); err != nil {
			return err
		}
		_, err := svc.ReviewExecution(ctx, execID, true, "", projectID)
		return err
	case model.ExecutionStatusFailed:
		// Go straight from assigned to failed (valid edge).
		reason := "seed for state-machine test"
		_, err := svc.UpdateExecutionStatus(ctx, execID, model.ExecutionStatusFailed, &reason, projectID)
		return err
	}
	return nil
}

// TestExecutionStateMachine_CrossTenant verifies the F-014 cross-tenant
// guard on the two new entry points. A row is owned by projectA;
// projectB calls ReviewExecution and CancelExecution; both must
// return ErrCrossTenantBlocked.
func TestExecutionStateMachine_CrossTenant(t *testing.T) {
	t.Parallel()

	svc, s := newExecutionTestService(t)
	ctx := context.Background()

	taskID, agentID, projectA := seedExecutionTaskAndAgent(t, s)
	exec, err := svc.CreateExecution(ctx, taskID, agentID, projectA)
	require.NoError(t, err)
	require.NoError(t, driveToState(t, svc, exec.ExecutionID, projectA, model.ExecutionStatusReview))

	projectB := uuid.New()

	t.Run("Review_CrossTenant", func(t *testing.T) {
		_, err := svc.ReviewExecution(ctx, exec.ExecutionID, true, "", projectB)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrCrossTenantBlocked), "expected ErrCrossTenantBlocked, got %v", err)
	})

	t.Run("Cancel_CrossTenant", func(t *testing.T) {
		err := svc.CancelExecution(ctx, exec.ExecutionID, projectB)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrCrossTenantBlocked), "expected ErrCrossTenantBlocked, got %v", err)
	})
}

// TestExecutionStateMachine_ReviewAction_Direct is a focused test
// for the ReviewExecution entry point that verifies the ReviewAction
// payload shape (id, from, to, at) returned to the handler.
func TestExecutionStateMachine_ReviewAction_Direct(t *testing.T) {
	t.Parallel()

	svc, s := newExecutionTestService(t)
	ctx := context.Background()

	taskID, agentID, projectID := seedExecutionTaskAndAgent(t, s)
	exec, err := svc.CreateExecution(ctx, taskID, agentID, projectID)
	require.NoError(t, err)
	require.NoError(t, driveToState(t, svc, exec.ExecutionID, projectID, model.ExecutionStatusReview))

	before := time.Now().UTC().Add(-1 * time.Second)
	action, err := svc.ReviewExecution(ctx, exec.ExecutionID, true, "", projectID)
	require.NoError(t, err)
	after := time.Now().UTC().Add(1 * time.Second)

	require.NotNil(t, action)
	assert.Equal(t, exec.ExecutionID, action.ExecutionID, "Action.ExecutionID must echo input")
	assert.Equal(t, model.ExecutionStatusReview, action.From, "Action.From must be 'review' (the prior state)")
	assert.Equal(t, model.ExecutionStatusCompleted, action.To, "Action.To must be 'completed' (the new state)")
	assert.True(t, action.At.After(before) && action.At.Before(after),
		"Action.At should be ~now (between %v and %v), got %v", before, after, action.At)
}

// TestExecutionStateMachine_Cancel_FromEachNonTerminal verifies the
// cancel action from each of the 4 non-terminal states.
func TestExecutionStateMachine_Cancel_FromEachNonTerminal(t *testing.T) {
	t.Parallel()

	nonTerminal := []model.ExecutionStatus{
		model.ExecutionStatusQueued,
		model.ExecutionStatusAssigned,
		model.ExecutionStatusRunning,
		model.ExecutionStatusReview,
	}

	for _, from := range nonTerminal {
		from := from
		t.Run("Cancel_From_"+string(from), func(t *testing.T) {
			t.Parallel()

			svc, s := newExecutionTestService(t)
			ctx := context.Background()

			taskID, agentID, projectID := seedExecutionTaskAndAgent(t, s)
			exec, err := svc.CreateExecution(ctx, taskID, agentID, projectID)
			require.NoError(t, err)
			require.NoError(t, driveToState(t, svc, exec.ExecutionID, projectID, from))

			require.NoError(t, svc.CancelExecution(ctx, exec.ExecutionID, projectID))

			final, err := svc.GetExecution(ctx, exec.ExecutionID, projectID)
			require.NoError(t, err)
			assert.Equal(t, model.ExecutionStatusFailed, final.Status)
			require.NotNil(t, final.ErrorMessage)
			assert.Equal(t, "cancelled by operator", *final.ErrorMessage)
		})
	}
}
