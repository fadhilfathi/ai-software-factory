// Package events: state.go holds the Sprint 5 (TASK-503) state
// machine that the ExecutionService will publish through MemoryBus.
//
// Scope (per Lead, 2026-06-14, Option C — minimal):
//
//   - 6 state constants matching brief §6.3:
//     QUEUED → ASSIGNED → RUNNING → REVIEW → COMPLETED/FAILED.
//   - A transition table (the only source of truth for valid edges).
//   - A StateManager interface so future producers and consumers
//     (TASK-501 runtime, TASK-505 deliverable capture, TASK-506 SSE)
//     can call into the state machine without depending on the
//     package-private helpers.
//
// Out of scope (deferred to Sprint 6 per Lead's dispatch):
//
//   - Wiring publishTransition into ExecutionService. The bus is
//     instantiated in main.go and stored on Services.Bus; the actual
//     "publish on transition" call lands when TASK-501's runtime
//     layer absorbs this in Sprint 6.
//   - Persistent state events (Postgres-backed bus).
//
// The 6 state values here are INTENTIONALLY a new type (Status), not
// a redefinition of model.ExecutionStatus. Under B-001, the model and
// migration 024/028 now also use the 6-value shape (queued/assigned/
// running/review/completed/failed). The two types are bridged in one
// place (a mapper in service.StateMapper or similar) so consumers can
// keep their Sprint 5 contracts.
package events

import (
	"errors"
	"fmt"
)

// Status is the 6-value enum for the Sprint 5 execution state
// machine. It is the value carried in Event.From and Event.To once
// Sprint 6 wires the publisher. (Event currently uses
// model.ExecutionStatus because Event was authored before this enum
// landed; that bridge is a 3-line fix in Sprint 6.)
type Status string

const (
	// StatusQueued is the initial state, before an agent has been
	// selected. A task may sit in Queued if the dispatcher has
	// accepted it but the agent picker has not run yet.
	StatusQueued Status = "QUEUED"

	// StatusAssigned means an agent has been picked and the
	// dispatch is en route. The worker process may not have
	// started yet.
	StatusAssigned Status = "ASSIGNED"

	// StatusRunning means the agent process is alive and
	// producing output. This is the longest-lived state in the
	// happy path.
	StatusRunning Status = "RUNNING"

	// StatusReview means the agent finished and the system is
	// waiting for human or automated sign-off before flipping to
	// Completed. (Auto-approval is Sprint 6; Sprint 5 always
	// pauses for explicit review.)
	StatusReview Status = "REVIEW"

	// StatusCompleted is the terminal happy state. The
	// DeliverableService (TASK-505) will pick up the artifact
	// from the running worker's stdout and create a deliverable
	// version.
	StatusCompleted Status = "COMPLETED"

	// StatusFailed is the terminal sad state. ErrorMessage on
	// the Event carries the failure detail (mock error in
	// tests, real runtime error in production).
	StatusFailed Status = "FAILED"
)

// AllStatuses is the canonical iteration order. Tests and the future
// UI dashboard use this so they don't drift from the const block.
var AllStatuses = []Status{
	StatusQueued,
	StatusAssigned,
	StatusRunning,
	StatusReview,
	StatusCompleted,
	StatusFailed,
}

// ErrInvalidTransition is returned by StateManager.Validate when the
// caller proposes a from→to edge that the transition table forbids.
// The caller should treat this as a 409 Conflict at the API layer
// (TASK-506 will surface it).
var ErrInvalidTransition = errors.New("events: invalid state transition")

// validTransitions is the SINGLE SOURCE OF TRUTH for the state
// machine. The shape is from→{to,...}; absence of an edge means it
// is not allowed.
//
// Rationale (Sprint 5, brief §6.3):
//   - Queued → Assigned (dispatcher picked an agent)
//   - Assigned → Running (worker process started)
//   - Running → Review (worker finished, awaiting approval)
//   - Running → Failed (worker crashed, timeout, or bad output)
//   - Review → Completed (human or auto-approver signed off)
//   - Review → Failed (rejected at review)
//   - Completed, Failed are TERMINAL — no edges out.
//
// Sprint 6 may add: Assigned → Failed (worker never started),
// Queued → Failed (dispatcher could not pick an agent).
var validTransitions = map[Status]map[Status]struct{}{
	StatusQueued: {
		StatusAssigned: {},
	},
	StatusAssigned: {
		StatusRunning: {},
	},
	StatusRunning: {
		StatusReview: {},
		StatusFailed:  {},
	},
	StatusReview: {
		StatusCompleted: {},
		StatusFailed:    {},
	},
	// StatusCompleted and StatusFailed have no outgoing edges.
	// Their keys are present as empty maps so a defensive
	// Validate(StatusCompleted, X) returns ErrInvalidTransition
	// (not a nil-map panic).
	StatusCompleted: {},
	StatusFailed:    {},
}

// StateManager is the consumer-facing interface to the state
// machine. It is small on purpose: one read-only check (Valid) and
// one enumeration helper (Allowed). Sprint 6 may add a
// Transition(from, to) method that mutates persistent state and
// publishes a Bus event; that lands alongside the ExecutionService
// refactor.
type StateManager interface {
	// Valid reports whether from→to is a permitted edge in the
	// Sprint 5 state machine. It returns true for any pair of
	// equal statuses (idempotent "transition" — used by the UI
	// to render an "already there" state without erroring).
	Valid(from, to Status) bool

	// Allowed returns the set of statuses reachable from from in
	// one step. The returned slice is in the canonical order from
	// AllStatuses so the UI can render a stable dropdown.
	Allowed(from Status) []Status
}

// NewStateManager returns a StateManager backed by the package-level
// transition table. The implementation is stateless and safe for
// concurrent use; callers may share a single instance.
func NewStateManager() StateManager {
	return &stateManager{}
}

type stateManager struct{}

// Valid implements StateManager. See the interface comment for the
// idempotent-equal rule.
func (s *stateManager) Valid(from, to Status) bool {
	if from == to {
		return true
	}
	outs, ok := validTransitions[from]
	if !ok {
		return false
	}
	_, allowed := outs[to]
	return allowed
}

// Allowed implements StateManager. The returned slice is in the
// canonical order from AllStatuses.
func (s *stateManager) Allowed(from Status) []Status {
	outs := validTransitions[from]
	if len(outs) == 0 {
		return nil
	}
	allowed := make([]Status, 0, len(outs))
	for _, status := range AllStatuses {
		if _, ok := outs[status]; ok {
			allowed = append(allowed, status)
		}
	}
	return allowed
}

// Validate is a package-level convenience for callers that want the
// error form. It is the same as StateManager.Valid(from, to) but
// returns ErrInvalidTransition instead of false.
//
// Use Validate when you want a single error path; use Valid when
// you want a bool (e.g., for UI gating).
func Validate(from, to Status) error {
	if NewStateManager().Valid(from, to) {
		return nil
	}
	return fmt.Errorf("%w: %s → %s", ErrInvalidTransition, from, to)
}
