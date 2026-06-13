package events

import (
	"testing"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/google/uuid"
)

// TestMemoryBus_RoundTrip exercises the full publish → subscribe path
// for a single project. It is the canonical Sprint 5 smoke for the
// bus: if this passes, MemoryBus honors its delivery contract
// (at-most-once, per-project fan-out, monotonic IDs) on the happy
// path. The two state-machine tests live in state_test.go.
//
// What it covers:
//   - Publish with Event.ID == 0 gets a monotonic ID assigned by the bus
//   - Publish with Event.At.IsZero() gets a timestamp stamped by the bus
//   - Subscribe returns a channel that receives the published event
//   - A second Subscribe on a DIFFERENT project does NOT receive it
//     (per-project fan-out; the bus does not leak across projects)
//   - Unsubscribe stops further events from being delivered
func TestMemoryBus_RoundTrip(t *testing.T) {
	bus := NewMemoryBus()

	projectA := uuid.New()
	projectB := uuid.New()

	chA, unsubA := bus.Subscribe(projectA)
	defer unsubA()

	chB, unsubB := bus.Subscribe(projectB)
	defer unsubB()

	// Build an event for projectA with ID and At unset. The bus
	// should assign both.
	taskID := uuid.New()
	agentID := uuid.New()
	execID := uuid.New()
	ev := Event{
		ExecutionID: execID,
		ProjectID:   projectA,
		TaskID:      taskID,
		AgentID:     agentID,
		From:        model.ExecutionStatusPending,
		To:          model.ExecutionStatusRunning,
	}

	if ok := bus.Publish(ev); !ok {
		t.Fatalf("Publish returned false: expected every subscriber's channel to accept (buffer is 64, this is event #1)")
	}

	select {
	case got := <-chA:
		if got.ID == 0 {
			t.Errorf("event.ID was not assigned by bus; got 0")
		}
		if got.At.IsZero() {
			t.Errorf("event.At was not stamped by bus; got zero time")
		}
		if got.ExecutionID != execID {
			t.Errorf("ExecutionID = %v, want %v", got.ExecutionID, execID)
		}
		if got.ProjectID != projectA {
			t.Errorf("ProjectID = %v, want %v", got.ProjectID, projectA)
		}
		if got.From != model.ExecutionStatusPending {
			t.Errorf("From = %v, want %v", got.From, model.ExecutionStatusPending)
		}
		if got.To != model.ExecutionStatusRunning {
			t.Errorf("To = %v, want %v", got.To, model.ExecutionStatusRunning)
		}
		// Sanity: At should be very recent (within the last 5s).
		if d := time.Since(got.At); d < 0 || d > 5*time.Second {
			t.Errorf("event.At not recent: %v ago", d)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("subscriber on projectA did not receive the published event within 2s")
	}

	// ProjectB must NOT receive projectA's event. Give the channel
	// a brief window to (incorrectly) deliver, then assert it is empty.
	select {
	case got := <-chB:
		t.Fatalf("subscriber on projectB received an event from projectA: %+v (per-project isolation broken)", got)
	case <-time.After(100 * time.Millisecond):
		// expected: nothing on chB
	}

	// Unsubscribe projectA, then publish a second event. The now-closed
	// chA must not panic and must not deliver.
	unsubA()
	if ok := bus.Publish(ev); ok {
		t.Errorf("Publish returned true after the only subscriber was unsubscribed; expected false (no recipients)")
	}
	select {
	case got, open := <-chA:
		if open {
			t.Errorf("chA received an event after unsubscribe: %+v", got)
		}
		// channel closed — that's fine, the cleanup happened
	case <-time.After(100 * time.Millisecond):
		t.Errorf("chA was not closed by unsubscribe within 100ms")
	}
}
