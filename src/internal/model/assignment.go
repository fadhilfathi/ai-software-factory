package model

import (
	"time"

	"github.com/google/uuid"
)

// MaxAssignmentNotesBytes caps the size of the `notes` field on
// POST /v1/tasks/:id/assign per api-spec.md §3.1. The cap matches
// the precedent set by MaxDeliverableContentBytes: an explicit
// application-level ceiling (well under any HTTP-body size limit)
// so the caller gets a structured PAYLOAD_TOO_LARGE error rather
// than a 413 from the http server body reader.
//
// The assignment_events.notes column itself is TEXT (no DB-level
// limit); this constant is enforced in the service and the
// handler. 1 KiB = 1024 bytes (binary KiB per the spec).
const MaxAssignmentNotesBytes int64 = 1 << 10

// AssignmentStatus is the lifecycle state of a single assignment row
// in the `assignments` table (migration 019). Persisted as TEXT with
// a CHECK constraint. Only one row per task may have status='active'
// at any time (enforced by the partial unique index
// uq_assignments_one_active_per_task in migration 019).
type AssignmentStatus string

const (
	// AssignmentStatusActive — the row represents the current
	// "who is assigned to this task right now". At most one such
	// row per task.
	AssignmentStatusActive AssignmentStatus = "active"
	// AssignmentStatusSuperseded — the row was the active one but
	// has been replaced by a newer assignment. completed_at is
	// set to the time of the replacement.
	AssignmentStatusSuperseded AssignmentStatus = "superseded"
	// AssignmentStatusCompleted — the row finished its lifecycle
	// because the assigned task was completed (TASK-405 will
	// drive this transition). completed_at is set.
	AssignmentStatusCompleted AssignmentStatus = "completed"
	// AssignmentStatusCancelled — the row was explicitly cancelled
	// (e.g. an admin override or a Sprint 5+ DELETE
	// /v1/tasks/:id/assign). completed_at is set.
	AssignmentStatusCancelled AssignmentStatus = "cancelled"
)

// AllAssignmentStatuses returns every valid status value. Used by
// service-level validation that mirrors the DB CHECK constraint.
func AllAssignmentStatuses() []AssignmentStatus {
	return []AssignmentStatus{
		AssignmentStatusActive,
		AssignmentStatusSuperseded,
		AssignmentStatusCompleted,
		AssignmentStatusCancelled,
	}
}

// IsValidAssignmentStatus reports whether the status is one of the
// four known enum values.
func IsValidAssignmentStatus(s AssignmentStatus) bool {
	for _, v := range AllAssignmentStatuses() {
		if v == s {
			return true
		}
	}
	return false
}

// Assignment is one row of the assignments table (migration 019).
// It is the "current state" projection: at most one Assignment with
// status=AssignmentStatusActive exists per task. The full history
// of actions lives in assignment_events (migration 020) and is
// linked back to assignments via AssignmentEvent.AssignmentID.
type Assignment struct {
	ID          uuid.UUID        `json:"id"`
	TaskID      uuid.UUID        `json:"task_id"`
	AgentID     uuid.UUID        `json:"agent_id"`
	AssignedAt  time.Time        `json:"assigned_at"`
	CompletedAt *time.Time       `json:"completed_at,omitempty"`
	Status      AssignmentStatus `json:"status"`
}

// AssignmentAction is the verb recorded in an AssignmentEvent row.
// Persisted as TEXT with a CHECK constraint (migration 019) so the
// service can rely on the DB to reject bogus values.
type AssignmentAction string

const (
	// AssignmentActionAssign — first-time assignment. task.AssigneeID
	// was unset before this event.
	AssignmentActionAssign AssignmentAction = "assign"
	// AssignmentActionReassign — task.AssigneeID was set to a
	// different agent before this event.
	AssignmentActionReassign AssignmentAction = "reassign"
	// AssignmentActionUnassign — task.AssigneeID was set before this
	// event and is unset after. The TASK-404 endpoint does not yet
	// emit this — a future Sprint 5 DELETE /v1/tasks/:id/assign will.
	// The enum is reserved so history rows from that endpoint don't
	// need a schema migration.
	AssignmentActionUnassign AssignmentAction = "unassign"
)

// AllAssignmentActions returns the set of valid actions. Used by
// service-level validation that mirrors the DB CHECK constraint.
func AllAssignmentActions() []AssignmentAction {
	return []AssignmentAction{
		AssignmentActionAssign,
		AssignmentActionReassign,
		AssignmentActionUnassign,
	}
}

// IsValidAssignmentAction reports whether the action is one of the
// three known enum values.
func IsValidAssignmentAction(a AssignmentAction) bool {
	for _, v := range AllAssignmentActions() {
		if v == a {
			return true
		}
	}
	return false
}

// AssignmentEvent is one row of the assignment_events history table
// (migration 020). It is the response shape of
// GET /v1/tasks/:id/history (api-spec.md §3.1) and the embedded
// "event" object in the POST /v1/tasks/:id/assign response.
//
// Notes:
//   - AssignmentID points back to the row in the `assignments`
//     table (migration 019) that caused this event. Always set in
//     TASK-404 (every event has a backing assignment).
//   - AgentID is *uuid.UUID because unassign events have no agent.
//   - AssignedBy is *uuid.UUID because system-initiated assignments
//     (Sprint 5+ autobalancer) have no human user. For TASK-404 the
//     handler always sets it from c.Get("user_id").
//   - Notes is the human-readable audit reason. Optional.
type AssignmentEvent struct {
	ID           uuid.UUID        `json:"id"`
	AssignmentID uuid.UUID        `json:"assignment_id"`
	TaskID       uuid.UUID        `json:"task_id"`
	AgentID      *uuid.UUID       `json:"agent_id,omitempty"`
	AssignedBy   *uuid.UUID       `json:"assigned_by,omitempty"`
	AssignedAt   time.Time        `json:"assigned_at"`
	Action       AssignmentAction `json:"action"`
	Notes        string           `json:"notes"`
}
