package events

import (
	"errors"
	"testing"
)

// TestStateMachine_AllValidTransitions asserts that the canonical
// happy-path edges (per brief §6.3) all pass Validate and
// StateManager.Valid. It is a table-driven sweep over the
// transition table so a future refactor that drops or adds an edge
// fails here loudly.
//
// The cases are the minimal viable set for Sprint 5; Sprint 6 will
// extend with Assigned → Failed and Queued → Failed (see
// state.go's transition-table comment).
func TestStateMachine_AllValidTransitions(t *testing.T) {
	sm := NewStateManager()

	cases := []struct {
		name string
		from Status
		to   Status
	}{
		// Happy path
		{"Queued→Assigned", StatusQueued, StatusAssigned},
		{"Assigned→Running", StatusAssigned, StatusRunning},
		{"Running→Review", StatusRunning, StatusReview},
		{"Review→Completed", StatusReview, StatusCompleted},

		// Failure paths
		{"Running→Failed", StatusRunning, StatusFailed},
		{"Review→Failed", StatusReview, StatusFailed},

		// Idempotent "already there" — used by the UI to no-op
		// re-renders without surfacing an error.
		{"Queued→Queued", StatusQueued, StatusQueued},
		{"Assigned→Assigned", StatusAssigned, StatusAssigned},
		{"Running→Running", StatusRunning, StatusRunning},
		{"Review→Review", StatusReview, StatusReview},
		{"Completed→Completed", StatusCompleted, StatusCompleted},
		{"Failed→Failed", StatusFailed, StatusFailed},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !sm.Valid(tc.from, tc.to) {
				t.Errorf("StateManager.Valid(%q, %q) = false, want true", tc.from, tc.to)
			}
			if err := Validate(tc.from, tc.to); err != nil {
				t.Errorf("Validate(%q, %q) = %v, want nil", tc.from, tc.to, err)
			}
		})
	}
}

// TestStateMachine_InvalidTransition asserts that every other edge
// is REJECTED with an error that wraps ErrInvalidTransition. This
// is the second half of the state-machine contract: if a caller
// proposes a forbidden edge (e.g., Queued → Completed, skipping
// review), the system surfaces a typed error so the API layer can
// map it to 409 Conflict.
//
// Sprint 6 will extend this table when more edges are added; until
// then, any edge NOT in the AllValidTransitions table above MUST be
// rejected.
func TestStateMachine_InvalidTransition(t *testing.T) {
	sm := NewStateManager()

	// Build a closed set of every from→to pair from AllStatuses
	// and assert: rejected ⇔ NOT in the AllValidTransitions table.
	validSet := map[struct{ from, to Status }]bool{
		{StatusQueued, StatusAssigned}:                  true,
		{StatusAssigned, StatusRunning}:                 true,
		{StatusRunning, StatusReview}:                   true,
		{StatusReview, StatusCompleted}:                 true,
		{StatusRunning, StatusFailed}:                   true,
		{StatusReview, StatusFailed}:                    true,
		// idempotent
		{StatusQueued, StatusQueued}:     true,
		{StatusAssigned, StatusAssigned}: true,
		{StatusRunning, StatusRunning}:   true,
		{StatusReview, StatusReview}:     true,
		{StatusCompleted, StatusCompleted}: true,
		{StatusFailed, StatusFailed}:     true,
	}

	for _, from := range AllStatuses {
		for _, to := range AllStatuses {
			t.Run(string(from)+"→"+string(to), func(t *testing.T) {
				shouldBeValid := validSet[struct{ from, to Status }{from, to}]

				gotValid := sm.Valid(from, to)
				if gotValid != shouldBeValid {
					t.Errorf("StateManager.Valid(%q, %q) = %v, want %v", from, to, gotValid, shouldBeValid)
				}

				err := Validate(from, to)
				if shouldBeValid {
					if err != nil {
						t.Errorf("Validate(%q, %q) = %v, want nil", from, to, err)
					}
					return
				}
				if err == nil {
					t.Fatalf("Validate(%q, %q) = nil, want error", from, to)
				}
				if !errors.Is(err, ErrInvalidTransition) {
					t.Errorf("Validate(%q, %q) = %v, want it to wrap ErrInvalidTransition", from, to, err)
				}
			})
		}
	}
}
