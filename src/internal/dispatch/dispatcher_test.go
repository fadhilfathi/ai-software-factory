package dispatch

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/aion"
	"go.uber.org/zap"
)

// ----------------------------------------------------------------------------
// Dispatcher unit tests
// ----------------------------------------------------------------------------

func TestDispatcher_HappyPath_Ack(t *testing.T) {
	q := NewInMemoryQueue(WithMaxAttempts(3))
	rt := aion.NewMockRuntime() // default script = WorkerCompleted, no delay
	d := NewDispatcher(q, rt, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := d.Start(ctx, 1); err != nil {
		t.Fatalf("Start: %v", err)
	}

	spec := validSpec()
	if err := q.Enqueue(ctx, spec); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	// Wait for the dispatcher to process.
	if err := waitFor(func() bool {
		stats := d.Stats()
		return stats.Spawned == 1 && stats.Completed == 1
	}, time.Second); err != nil {
		t.Errorf("Dispatcher did not complete: %v (stats: %+v)", err, d.Stats())
	}

	if err := d.Stop(ctx); err != nil {
		t.Errorf("Stop: %v", err)
	}
	if q.InFlightCount() != 0 {
		t.Errorf("InFlightCount after stop: got %d, want 0", q.InFlightCount())
	}
	if q.Len() != 0 {
		t.Errorf("Len after stop: got %d, want 0", q.Len())
	}
}

func TestDispatcher_Failure_Nack_Retry(t *testing.T) {
	q := NewInMemoryQueue(WithMaxAttempts(3))
	rt := aion.NewMockRuntime()
	// Set every spec to fail.
	rt.SetDefaultScript(aion.FakeScript{
		Outcome:      aion.WorkerFailed,
		ErrorMessage: "test failure",
	})
	d := NewDispatcher(q, rt, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := d.Start(ctx, 1); err != nil {
		t.Fatalf("Start: %v", err)
	}

	spec := validSpec()
	if err := q.Enqueue(ctx, spec); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	// Wait for 3 spawns (initial + 2 retries).
	if err := waitFor(func() bool {
		return d.Stats().Spawned == 3 && q.DroppedCount() == 1
	}, 2*time.Second); err != nil {
		t.Errorf("Dispatcher did not exhaust retries: %v (stats: %+v, dropped: %d)",
			err, d.Stats(), q.DroppedCount())
	}

	if err := d.Stop(ctx); err != nil {
		t.Errorf("Stop: %v", err)
	}
	if d.Stats().Failed != 3 {
		t.Errorf("Failed count: got %d, want 3", d.Stats().Failed)
	}
	if d.Stats().Dropped != 1 {
		t.Errorf("Dropped count: got %d, want 1 (queue reports it)", d.Stats().Dropped)
	}
}

func TestDispatcher_Concurrent(t *testing.T) {
	q := NewInMemoryQueue(WithBufferSize(100), WithMaxAttempts(1)) // no retries for speed
	rt := aion.NewMockRuntime()                                    // default = happy path
	d := NewDispatcher(q, rt, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := d.Start(ctx, 4); err != nil {
		t.Fatalf("Start: %v", err)
	}

	const N = 50
	for i := 0; i < N; i++ {
		if err := q.Enqueue(ctx, validSpec()); err != nil {
			t.Fatalf("Enqueue[%d]: %v", i, err)
		}
	}

	// Wait for all N to complete.
	if err := waitFor(func() bool {
		return d.Stats().Completed == N
	}, 5*time.Second); err != nil {
		t.Errorf("Dispatcher did not complete all %d: %v (stats: %+v)", N, err, d.Stats())
	}

	if err := d.Stop(ctx); err != nil {
		t.Errorf("Stop: %v", err)
	}
	if d.Stats().Failed != 0 {
		t.Errorf("Failed count: got %d, want 0", d.Stats().Failed)
	}
	if q.InFlightCount() != 0 {
		t.Errorf("InFlightCount after stop: got %d, want 0", q.InFlightCount())
	}
}

func TestDispatcher_MixedOutcomes(t *testing.T) {
	q := NewInMemoryQueue(WithMaxAttempts(1))
	rt := aion.NewMockRuntime()
	d := NewDispatcher(q, rt, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := d.Start(ctx, 4); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// 3 happy, 2 failed, 1 cancelled.
	specs := []aion.WorkerSpec{validSpec(), validSpec(), validSpec(), validSpec(), validSpec(), validSpec()}
	rt.RegisterScript(specs[0].ExecutionID.String(), aion.FakeScript{Outcome: aion.WorkerCompleted})
	rt.RegisterScript(specs[1].ExecutionID.String(), aion.FakeScript{Outcome: aion.WorkerCompleted})
	rt.RegisterScript(specs[2].ExecutionID.String(), aion.FakeScript{Outcome: aion.WorkerCompleted})
	rt.RegisterScript(specs[3].ExecutionID.String(), aion.FakeScript{Outcome: aion.WorkerFailed, ErrorMessage: "fail"})
	rt.RegisterScript(specs[4].ExecutionID.String(), aion.FakeScript{Outcome: aion.WorkerFailed, ErrorMessage: "fail"})
	// specs[5] uses default (completed). But we want a cancel. Instead,
	// we'll cancel ctx AFTER the 5 above finish. Specs[5] will be
	// enqueued but its worker will be cancelled by ctx done.

	for _, s := range specs[:5] {
		if err := q.Enqueue(ctx, s); err != nil {
			t.Fatalf("Enqueue: %v", err)
		}
	}

	// Wait for the 5 to complete (3 ack, 2 nack-as-failed).
	if err := waitFor(func() bool {
		return d.Stats().Completed == 3 && d.Stats().Failed == 2
	}, 2*time.Second); err != nil {
		t.Errorf("Dispatcher did not handle mixed outcomes: %v (stats: %+v)", err, d.Stats())
	}

	// Enqueue one more, then cancel ctx. The dispatcher should
	// record the cancellation.
	specC := validSpec()
	rt.RegisterScript(specC.ExecutionID.String(), aion.FakeScript{Delay: 100 * time.Millisecond, Outcome: aion.WorkerCompleted})
	if err := q.Enqueue(ctx, specC); err != nil {
		t.Fatalf("Enqueue cancel-spec: %v", err)
	}
	// Give the dispatcher a moment to spawn the worker.
	time.Sleep(20 * time.Millisecond)
	cancel()

	if err := d.Stop(context.Background()); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Logf("Stop (expected timeout or success): %v", err)
	}
	stats := d.Stats()
	if stats.Cancelled < 1 {
		t.Errorf("Cancelled count: got %d, want >= 1 (stats: %+v)", stats.Cancelled, stats)
	}
}

func TestDispatcher_Start_Idempotent(t *testing.T) {
	q := NewInMemoryQueue()
	rt := aion.NewMockRuntime()
	d := NewDispatcher(q, rt, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := d.Start(ctx, 2); err != nil {
		t.Fatalf("Start[0]: %v", err)
	}
	if err := d.Start(ctx, 4); err != nil { // should be no-op
		t.Errorf("Start[1] (idempotent): %v", err)
	}
	if d.Workers() != 2 {
		t.Errorf("Workers: got %d, want 2 (Start[1] should not change worker count)", d.Workers())
	}
	if err := d.Stop(ctx); err != nil {
		t.Errorf("Stop: %v", err)
	}
}

func TestDispatcher_Start_InvalidWorkerCount(t *testing.T) {
	q := NewInMemoryQueue()
	rt := aion.NewMockRuntime()
	d := NewDispatcher(q, rt, zap.NewNop())

	err := d.Start(context.Background(), 0)
	if err == nil {
		t.Errorf("Start(workers=0): got nil, want error")
	}
}

func TestDispatcher_Stop_NotRunning(t *testing.T) {
	q := NewInMemoryQueue()
	rt := aion.NewMockRuntime()
	d := NewDispatcher(q, rt, zap.NewNop())

	if err := d.Stop(context.Background()); err != nil {
		t.Errorf("Stop without Start: got %v, want nil", err)
	}
}

// ----------------------------------------------------------------------------
// Test helpers
// ----------------------------------------------------------------------------

// waitFor polls fn every 10ms until it returns true or timeout
// elapses. Returns nil on success, an error on timeout.
func waitFor(fn func() bool, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	if fn() {
		return nil
	}
	return errors.New("timeout")
}
