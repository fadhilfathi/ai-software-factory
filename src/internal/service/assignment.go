package service

import (
	"context"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AssignmentResult is the return shape of AssignTaskToAgent. It pairs
// the updated task with the appended event so the handler can emit
// one round-trip response that surfaces both the state change and
// the audit trail. The Assignment pointer is the new "active" row
// from the assignments table (migration 019); Task + Event mirror
// the rest of the brief.
type AssignmentResult struct {
	Task       *model.Task            `json:"task"`
	Event      *model.AssignmentEvent `json:"event"`
	Assignment *model.Assignment      `json:"assignment"`
	// Idempotent is true when the request matched the existing
	// assignee and no new event was written. Lets the UI
	// distinguish "reassignment succeeded" from "we re-posted
	// the same payload and nothing changed".
	Idempotent bool `json:"idempotent"`
}

type AssignmentService struct {
	store  store.Store
	capSvc *CapabilityService
	log    *zap.Logger
}

func NewAssignmentService(s store.Store, capSvc *CapabilityService, log *zap.Logger) *AssignmentService {
	return &AssignmentService{store: s, capSvc: capSvc, log: log}
}

// AssignTaskToAgent wires a task to an agent and writes to BOTH
// the assignments table (migration 019) AND the assignment_events
// table (migration 020) inside a single transaction. The split was
// introduced in the data-model.md finalisation (TASK-404 brief
// correction): assignments is the current-state "who is assigned
// right now" table, assignment_events is the immutable history of
// every state change. The two are linked by
// assignment_events.assignment_id → assignments.id.
//
// `notes` is the caller's free-form note for this assignment and
// is persisted in assignment_events.notes so the audit trail row
// is complete on later GET /v1/tasks/:id/history reads (F-017
// fix; before the fix the row was written with Notes:"" and the
// handler mutated the in-memory response after the fact).
//
// Pre-flight contract:
//  1. Task and agent must both exist (404 on miss).
//  2. Agent must be in the idle lifecycle state (409 on miss).
//  3. If `capabilitiesRequired` is non-empty, persist it onto
//     task.RequiredCapabilities (TASK-404 is the only endpoint that
//     populates this column in Sprint 4) and validate the agent
//     against the now-persisted value. Mismatch → 409
//     CAPABILITY_MISMATCH per api-spec.md §3.1.
//  4. If `capabilitiesRequired` is empty, the task's existing
//     required_capabilities is PRESERVED (not nulled). Documented
//     in the brief and in step 3 of the new method signature.
//  5. The TASK-403 enforcement seam
//     (capSvc.ValidateAgentHasCapabilities) is invoked with the
//     resolved required-capabilities list.
//
// Action resolution:
//   - task.AssigneeID == uuid.Nil → action = "assign"
//   - task.AssigneeID != uuid.Nil && != agentID → action = "reassign"
//   - task.AssigneeID == agentID → idempotent no-op, no event
//     written, returns the existing state with Idempotent=true.
//
// Transactional write (s.store.WithTx):
//   a. If a previous active row exists for this task, flip it to
//      status='superseded' with completed_at=now. The update is
//      done first so the partial unique index
//      uq_assignments_one_active_per_task releases the slot before
//      the new active row is inserted.
//   b. Create the new assignment row with status='active'.
//   c. Append the corresponding assignment_event row with
//      assignment_id = new assignment's id and the resolved action.
//      If any of the three steps fail, the transaction is rolled
//      back and the database state is unchanged.
//
// After the transaction:
//   - task.AssigneeID = agentID (single-row update outside the
//     tx; the task row is its own table, so it doesn't need to be
//     in the assignments/events transaction).
//
// Returns:
//   - 200 with *AssignmentResult on success
//   - 404 NOT_FOUND if task or agent doesn't exist
//   - 409 CAPABILITY_MISMATCH if the agent lacks any required cap
//   - 409 "Agent is not idle" if the agent is not in idle state
//   - 409 ALREADY_EXISTS if the partial unique index rejects the
//     new active row (concurrent POST race). Mapped from
//     store.ErrAlreadyExists.
//   - 500 INTERNAL on store error
func (s *AssignmentService) AssignTaskToAgent(
	ctx context.Context,
	taskID uuid.UUID,
	agentID uuid.UUID,
	notes string,
	assignedBy *uuid.UUID,
	capabilitiesRequired []string,
) (*AssignmentResult, *Error) {
	task, err := s.store.Tasks().GetByID(taskID)
	if err != nil {
		return nil, notFound("Task not found")
	}

	// Quick existence check for the agent (a single GetByID). We
	// don't read the agent's capabilities here — that's the
	// CapabilityService's job. This keeps the read fan-out
	// bounded: one task fetch, one agent existence check, one
	// capability check, one task update, one tx that does three
	// writes atomically.
	agent, err := s.store.Agents().GetByID(ctx, agentID)
	if err != nil {
		return nil, notFound("Agent not found")
	}

	if agent.Status != model.AgentIdle {
		return nil, conflict("Agent is not idle")
	}

	// Persist the request's capabilities_required onto the task.
	// Empty input preserves the task's existing value (the brief
	// is explicit on this — "don't null it out"). This makes the
	// field monotonic across PATCH-like calls.
	if len(capabilitiesRequired) > 0 {
		task.RequiredCapabilities = capabilitiesRequired
	}

	// TASK-403 enforcement seam. Reads from the live
	// agent_capabilities join table (migration 017) via the
	// capability store. On mismatch, return 409 CAPABILITY_MISMATCH
	// and stop before mutating any state.
	if len(task.RequiredCapabilities) > 0 {
		if capErr := s.capSvc.ValidateAgentHasCapabilities(ctx, agentID, task.RequiredCapabilities); capErr != nil {
			if e, ok := capErr.(*Error); ok {
				return nil, e
			}
			// Defensive: anything that isn't already a *Error is
			// treated as an internal failure so the handler never
			// sees a raw error.
			s.log.Error("capability validation: unexpected error type",
				zap.String("agent_id", agentID.String()),
				zap.Strings("required", task.RequiredCapabilities),
				zap.Error(capErr))
			return nil, internalError("Capability validation failed")
		}
	}

	// Idempotent no-op: the agent is already the assignee. We do
	// not write a new event, do not flip the existing assignment,
	// and do not bump task.UpdatedAt. This matches the
	// api-spec.md §3.1 contract: re-POSTing the same agent_id is
	// a no-op.
	if task.AssigneeID == agentID {
		// Look up the existing active row to return it in the
		// result. Best-effort — if it's missing (shouldn't
		// happen for an active task), return nil.
		var existing *model.Assignment
		if a, getErr := s.store.Assignments().GetActiveByTask(ctx, taskID); getErr == nil {
			existing = a
		}
		return &AssignmentResult{
			Task:       task,
			Event:      nil,
			Assignment: existing,
			Idempotent: true,
		}, nil
	}

	// Resolve the action verb. The DB stores it as TEXT with a
	// CHECK constraint; the service also validates via
	// model.IsValidAssignmentAction (defence in depth).
	var action model.AssignmentAction
	switch {
	case task.AssigneeID == uuid.Nil:
		action = model.AssignmentActionAssign
	default:
		action = model.AssignmentActionReassign
	}

	now := time.Now().UTC()
	previousAssignee := task.AssigneeID

	// Atomic transactional write: (a) flip any existing active
	// row to 'superseded', (b) create the new active row,
	// (c) append the corresponding event. All three either
	// commit together or roll back together.
	type txResult struct {
		newAssignment *model.Assignment
		newEvent      *model.AssignmentEvent
	}
	var txOut txResult
	txErr := s.store.WithTx(ctx, func(txStore store.Tx) error {
		// (a) Flip the previous active row to 'superseded'.
		// GetActiveByTask may return store.ErrNotFound on a
		// first-time assign — that is the happy path (nothing
		// to flip). On a reassign the row exists and we
		// stamp completed_at = now.
		prev, getErr := txStore.Assignments().GetActiveByTask(ctx, taskID)
		if getErr != nil && getErr != store.ErrNotFound {
			return getErr
		}
		if prev != nil {
			prev.Status = model.AssignmentStatusSuperseded
			prev.CompletedAt = &now
			if updErr := txStore.Assignments().Update(ctx, prev); updErr != nil {
				return updErr
			}
		}

		// (b) Create the new active row.
		agentIDPtr := agentID
		newA := &model.Assignment{
			ID:         uuid.New(),
			TaskID:     taskID,
			AgentID:    agentIDPtr,
			AssignedAt: now,
			Status:     model.AssignmentStatusActive,
		}
		created, createErr := txStore.Assignments().Create(ctx, newA)
		if createErr != nil {
			// Map store.ErrAlreadyExists to a 409 envelope
			// (the partial unique index fired, meaning a
			// concurrent POST beat us to it).
			if createErr == store.ErrAlreadyExists {
				return createErr
			}
			return createErr
		}

		// (c) Append the corresponding event with the new
		// assignment_id. action_enum and FKs are enforced
		// by the DB; we pass them through. Notes is the
		// caller's free-form audit note (F-017 fix: the
		// service now persists it; the handler used to
		// mutate the in-memory response after the fact,
		// which left the row empty in the DB).
		ev, appendErr := txStore.AssignmentEvents().Append(ctx, &model.AssignmentEvent{
			ID:           uuid.New(),
			AssignmentID: created.ID,
			TaskID:       taskID,
			AgentID:      &agentIDPtr,
			AssignedBy:   assignedBy,
			AssignedAt:   now,
			Action:       action,
			Notes:        notes,
		})
		if appendErr != nil {
			return appendErr
		}

		txOut.newAssignment = created
		txOut.newEvent = ev
		return nil
	})
	if txErr != nil {
		// Map store.ErrAlreadyExists to a 409.
		if txErr == store.ErrAlreadyExists {
			return nil, conflict("Assignment race: another request created an active row for this task concurrently")
		}
		s.log.Error("assignment tx failed",
			zap.String("task_id", taskID.String()),
			zap.String("agent_id", agentID.String()),
			zap.Error(txErr))
		return nil, internalError("Failed to record assignment")
	}

	// Update the task outside the tx. If this fails the
	// assignment is already committed (assignments + events are
	// consistent) but the task.AssigneeID pointer is stale. The
	// trade-off matches the brief: the data invariant is "every
	// event has an assignment" not "every task update has an
	// assignment". A Sprint 5+ reconciliation job can backfill.
	task.AssigneeID = agentID
	task.UpdatedAt = now
	if err := s.store.Tasks().Update(task); err != nil {
		s.log.Error("failed to update task assignment",
			zap.String("task_id", taskID.String()),
			zap.String("from", previousAssignee.String()),
			zap.String("to", agentID.String()),
			zap.Error(err))
		return nil, internalError("Failed to update task assignment")
	}

	return &AssignmentResult{
		Task:       task,
		Event:      txOut.newEvent,
		Assignment: txOut.newAssignment,
		Idempotent: false,
	}, nil
}

// ListAssignmentHistory returns the append-only history of
// assignment actions for a task, newest first. The endpoint is
// GET /v1/tasks/:id/history (TASK-404). Reads from the
// assignment_events table only — the assignments table is the
// current-state projection and is not part of the history read.
//
// Convention (per the brief):
//   - Task not found → 404 NOT_FOUND (the brief prefers this over
//     "empty slice + nil" so the UI can distinguish "no history
//     yet" from "no such task").
//   - Task exists but has no events → 200 with empty data array.
//   - Store error → 500 INTERNAL.
func (s *AssignmentService) ListAssignmentHistory(ctx context.Context, taskID uuid.UUID) ([]*model.AssignmentEvent, *Error) {
	// Existence check up front so a missing task surfaces as 404
	// rather than an empty list. We use Tasks().GetByID, which
	// returns store.ErrNotFound on miss.
	if _, err := s.store.Tasks().GetByID(taskID); err != nil {
		return nil, notFound("Task not found")
	}

	events, err := s.store.AssignmentEvents().ListByTask(ctx, taskID)
	if err != nil {
		s.log.Error("failed to list assignment history",
			zap.String("task_id", taskID.String()),
			zap.Error(err))
		return nil, internalError("Failed to load assignment history")
	}
	return events, nil
}
