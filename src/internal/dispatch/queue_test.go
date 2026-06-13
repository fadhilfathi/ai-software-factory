package dispatch

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/aion"
	"github.com/google/uuid"
)

// ----------------------------------------------------------------------------
// Test helpers
// ----------------------------------------------------------------------------

// validSpec returns a well-formed WorkerSpec for tests. Attempt
// defaults to 1 (initial).
func validSpec() aion.WorkerSpec {
	return aion.WorkerSpec{
		ExecutionID: uuid.New(),
		TaskID:      uuid.New(),
		AgentID:     uuid.New(),
		ProjectID:   uuid.New(),
		Model:          "MiniMax-M3",
		Provider:       "TokenRouter",
		PermissionMode: "YOLO",
		Attempt:        1,
	}
}

// validSpecWithAttempt is validSpec with a specific attempt.
func validSpecWithAttempt(attempt int) aion.WorkerSpec {
	s := validSpec()
	s.Attempt = attempt
	return s
}

// ----------------------------------------------------------------------------
// Enqueue / Dequeue
// ----------------------------------------------------------------------------

func TestEnqueueDequeue_SingleProducerConsumer(t *testing.T) {
	q := NewInMemoryQueue()
	spec := validSpec()
	if err := q.Enqueue(context.Background(), spec); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if got := q.Len(); got != 1 {
		t.Errorf("Len: got %d, want 1", got)
	}
	got, err := q.Dequeue(context.Background())
	if err != nil {
		t.Fatalf("Dequeue: %v", err)
	}
	if got.ExecutionID != spec.ExecutionID {
		t.Errorf("Dequeue: got execID %s, want %s", got.ExecutionID, spec.ExecutionID)
	}
	if got := q.Len(); got != 0 {
		t.Errorf("Len after dequeue: got %d, want 0", got)
	}
	if got := q.InFlightCount(); got != 1 {
		t.Errorf("InFlightCount after dequeue: got %d, want 1", got)
	}
}

func TestEnqueueDequeue_FIFO(t *testing.T) {
	q := NewInMemoryQueue()
	specs := []aion.WorkerSpec{validSpec(), validSpec(), validSpec()}
	for _, s := range specs {
		if err := q.Enqueue(context.Background(), s); err != nil {
			t.Fatalf("Enqueue: %v", err)
		}
	}
	for i, want := range specs {
		got, err := q.Dequeue(context.Background())
		if err != nil {
			t.Fatalf("Dequeue[%d]: %v", i, err)
		}
		if got.ExecutionID != want.ExecutionID {
			t.Errorf("Dequeue[%d]: got %s, want %s (FIFO order broken)", i, got.ExecutionID, want.ExecutionID)
		}
	}
}

func TestEnqueueDequeue_Concurrent(t *testing.T) {
	q := NewInMemoryQueue(WithBufferSize(200))
	const numProducers = 10
	const perProducer = 10
	const total = numProducers * perProducer

	var wgP, wgC sync.WaitGroup
	produced := make(chan aion.WorkerSpec, total)

	// Producers
	for i := 0; i < numProducers; i++ {
		wgP.Add(1)
		go func() {
			defer wgP.Done()
			for j := 0; j < perProducer; j++ {
				s := validSpec()
				if err := q.Enqueue(context.Background(), s); err != nil {
					t.Errorf("Enqueue: %v", err)
					return
				}
				produced <- s
			}
		}()
	}

	// Consumer: dequeues everything, then signals done.
	consumed := make(map[uuid.UUID]bool, total)
	var consumedMu sync.Mutex
	wgC.Add(1)
	go func() {
		defer wgC.Done()
		for i := 0; i < total; i++ {
			spec, err := q.Dequeue(context.Background())
			if err != nil {
				t.Errorf("Dequeue: %v", err)
				return
			}
			consumedMu.Lock()
			consumed[spec.ExecutionID] = true
			consumedMu.Unlock()
		}
	}()

	wgP.Wait()
	close(produced)
	wgC.Wait()

	if got := len(consumed); got != total {
		t.Errorf("consumed: got %d, want %d", got, total)
	}
	// Verify all produced are in consumed (no drops)
	for s := range produced {
		if !consumed[s.ExecutionID] {
			t.Errorf("spec %s was produced but not consumed", s.ExecutionID)
		}
	}
}

func TestEnqueue_InvalidSpec(t *testing.T) {
	q := NewInMemoryQueue()
	bad := validSpec()
	bad.ExecutionID = uuid.Nil // invalid
	err := q.Enqueue(context.Background(), bad)
	if !errors.Is(err, ErrSpecInvalid) {
		t.Errorf("Enqueue invalid: got %v, want ErrSpecInvalid", err)
	}
}

func TestEnqueue_QueueFull_NonBlocking(t *testing.T) {
	q := NewInMemoryQueue(WithBufferSize(1), WithNonBlocking())
	if err := q.Enqueue(context.Background(), validSpec()); err != nil {
		t.Fatalf("Enqueue[0]: %v", err)
	}
	// Buffer is now full (1/1). Second Enqueue should return ErrQueueFull.
	err := q.Enqueue(context.Background(), validSpec())
	if !errors.Is(err, ErrQueueFull) {
		t.Errorf("Enqueue when full: got %v, want ErrQueueFull", err)
	}
}

func TestEnqueue_QueueFull_Blocking(t *testing.T) {
	q := NewInMemoryQueue(WithBufferSize(1)) // blocking by default
	if err := q.Enqueue(context.Background(), validSpec()); err != nil {
		t.Fatalf("Enqueue[0]: %v", err)
	}
	// Second Enqueue should block. Use a goroutine + select.
	done := make(chan error, 1)
	go func() {
		done <- q.Enqueue(context.Background(), validSpec())
	}()
	select {
	case err := <-done:
		t.Errorf("Enqueue[1] did not block: got %v", err)
	case <-time.After(50 * time.Millisecond):
		// Good, it's blocking.
	}
	// Drain the queue so the goroutine can complete.
	if _, err := q.Dequeue(context.Background()); err != nil {
		t.Fatalf("Dequeue: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Enqueue[1] after drain: %v", err)
		}
	case <-time.After(time.Second):
		t.Errorf("Enqueue[1] did not unblock after drain")
	}
}

// ----------------------------------------------------------------------------
// Ack
// ----------------------------------------------------------------------------

func TestAck_RemovesFromInFlight(t *testing.T) {
	q := NewInMemoryQueue()
	spec := validSpec()
	_ = q.Enqueue(context.Background(), spec)
	got, _ := q.Dequeue(context.Background())
	if q.InFlightCount() != 1 {
		t.Fatalf("InFlightCount after dequeue: got %d, want 1", q.InFlightCount())
	}
	if err := q.Ack(context.Background(), got, aion.WorkerResult{Status: aion.WorkerCompleted}); err != nil {
		t.Fatalf("Ack: %v", err)
	}
	if q.InFlightCount() != 0 {
		t.Errorf("InFlightCount after ack: got %d, want 0", q.InFlightCount())
	}
}

func TestAck_Idempotent(t *testing.T) {
	q := NewInMemoryQueue()
	spec := validSpec()
	_ = q.Enqueue(context.Background(), spec)
	got, _ := q.Dequeue(context.Background())
	_ = q.Ack(context.Background(), got, aion.WorkerResult{Status: aion.WorkerCompleted})
	err := q.Ack(context.Background(), got, aion.WorkerResult{Status: aion.WorkerCompleted})
	if !errors.Is(err, ErrUnknownSpec) {
		t.Errorf("Ack idempotent: got %v, want ErrUnknownSpec", err)
	}
}

// ----------------------------------------------------------------------------
// Nack (retry vs. DLQ)
// ----------------------------------------------------------------------------

func TestNack_RetryUnderMax(t *testing.T) {
	q := NewInMemoryQueue(WithMaxAttempts(3))
	spec := validSpecWithAttempt(1) // attempt 1
	_ = q.Enqueue(context.Background(), spec)
	got, _ := q.Dequeue(context.Background())
	// Nack with attempt 1 < max 3 → re-queue with attempt 2.
	if err := q.Nack(context.Background(), got, aion.WorkerResult{Status: aion.WorkerFailed}, errors.New("test")); err != nil {
		t.Fatalf("Nack: %v", err)
	}
	// The retry should be on the queue.
	if q.Len() != 1 {
		t.Errorf("Len after nack (retry): got %d, want 1", q.Len())
	}
	if q.DroppedCount() != 0 {
		t.Errorf("DroppedCount: got %d, want 0", q.DroppedCount())
	}
	retry, _ := q.Dequeue(context.Background())
	if retry.Attempt != 2 {
		t.Errorf("Retry attempt: got %d, want 2", retry.Attempt)
	}
}

func TestNack_RetryExhaustion(t *testing.T) {
	q := NewInMemoryQueue(WithMaxAttempts(3))
	spec := validSpecWithAttempt(3) // attempt 3 == max
	_ = q.Enqueue(context.Background(), spec)
	got, _ := q.Dequeue(context.Background())
	// Nack with attempt 3 == max 3 → DLQ, not retry.
	if err := q.Nack(context.Background(), got, aion.WorkerResult{Status: aion.WorkerFailed}, errors.New("test")); err != nil {
		t.Fatalf("Nack: %v", err)
	}
	if q.Len() != 0 {
		t.Errorf("Len after nack (DLQ): got %d, want 0", q.Len())
	}
	if q.DroppedCount() != 1 {
		t.Errorf("DroppedCount: got %d, want 1", q.DroppedCount())
	}
}

func TestNack_Idempotent(t *testing.T) {
	q := NewInMemoryQueue()
	spec := validSpecWithAttempt(1) // attempt 1
	_ = q.Enqueue(context.Background(), spec)
	got, _ := q.Dequeue(context.Background())
	_ = q.Nack(context.Background(), got, aion.WorkerResult{Status: aion.WorkerFailed}, errors.New("test"))
	// After nack (which re-queued), the original spec is no longer in-flight.
	err := q.Nack(context.Background(), got, aion.WorkerResult{Status: aion.WorkerFailed}, errors.New("test"))
	if !errors.Is(err, ErrUnknownSpec) {
		t.Errorf("Nack idempotent: got %v, want ErrUnknownSpec", err)
	}
}

// ----------------------------------------------------------------------------
// Close
// ----------------------------------------------------------------------------

func TestClose_EnqueueAfterClose(t *testing.T) {
	q := NewInMemoryQueue()
	if err := q.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	err := q.Enqueue(context.Background(), validSpec())
	if !errors.Is(err, ErrQueueClosed) {
		t.Errorf("Enqueue after close: got %v, want ErrQueueClosed", err)
	}
}

func TestClose_DequeueAfterClose_Empty(t *testing.T) {
	q := NewInMemoryQueue()
	if err := q.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	_, err := q.Dequeue(context.Background())
	if !errors.Is(err, ErrQueueClosed) {
		t.Errorf("Dequeue after close (empty): got %v, want ErrQueueClosed", err)
	}
}

func TestClose_DequeueDrainsFirst(t *testing.T) {
	q := NewInMemoryQueue()
	spec := validSpec()
	_ = q.Enqueue(context.Background(), spec)
	_ = q.Close()
	// Dequeue should drain the remaining spec, then return ErrQueueClosed.
	got, err := q.Dequeue(context.Background())
	if err != nil {
		t.Errorf("Dequeue after close (drain): got %v, want nil (drain first)", err)
	}
	if got.ExecutionID != spec.ExecutionID {
		t.Errorf("Dequeue drain: got %s, want %s", got.ExecutionID, spec.ExecutionID)
	}
	_, err = q.Dequeue(context.Background())
	if !errors.Is(err, ErrQueueClosed) {
		t.Errorf("Dequeue after drain: got %v, want ErrQueueClosed", err)
	}
}

func TestClose_Idempotent(t *testing.T) {
	q := NewInMemoryQueue()
	if err := q.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := q.Close(); err != nil {
		t.Errorf("Close idempotent: got %v, want nil", err)
	}
}

// ----------------------------------------------------------------------------
// Context cancellation
// ----------------------------------------------------------------------------

func TestDequeue_ContextCancellation(t *testing.T) {
	q := NewInMemoryQueue()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	_, err := q.Dequeue(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Dequeue with cancelled ctx: got %v, want context.Canceled", err)
	}
}

func TestEnqueue_ContextCancellation(t *testing.T) {
	q := NewInMemoryQueue(WithBufferSize(1)) // blocking
	// Fill the buffer so the next Enqueue blocks.
	_ = q.Enqueue(context.Background(), validSpec())
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	err := q.Enqueue(ctx, validSpec())
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Enqueue with cancelled ctx: got %v, want context.Canceled", err)
	}
}
