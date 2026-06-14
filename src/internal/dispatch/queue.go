// Package dispatch provides the dispatch queue + dispatcher for the
// AI Software Factory's task-execution pipeline.
//
// Sprint 5 ships the in-memory implementation. A Postgres-backed
// implementation is a Sprint 6 follow-up (TASK-502 follow-up + the
// dispatch_queue Postgres table sketched in §6.2 of the brief).
//
// The package boundary is:
//
//   - DispatchQueue: the queue itself (Enqueue/Dequeue/Ack/Nack/Close).
//     InMemoryQueue is the Sprint 5 impl. The interface is intentionally
//     minimal so a future Postgres implementation can swap in without
//     touching the dispatcher or the service layer.
//
//   - Dispatcher: a fixed-size pool of worker goroutines that pull
//     specs from the queue and drive aion.Runtime.Spawn + Wait. On
//     terminal status, the dispatcher calls Ack (success) or Nack
//     (failure) on the queue, which handles retry-vs-DLQ decisions.
//
// Cross-tenant: WorkerSpec carries the callerProjectID. The queue
// preserves it on retry. The dispatcher is tenant-agnostic — it just
// routes specs to the runtime. Cross-tenant validation is the
// service's job (it builds the spec, not the queue).
package dispatch

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/fadhilfathi/AI-Software-Factory/internal/aion"
	"github.com/google/uuid"
)

// ----------------------------------------------------------------------------
// Errors
// ----------------------------------------------------------------------------

// ErrQueueFull is returned by Enqueue when the queue is bounded and at
// capacity. Sprint 5's InMemoryQueue uses a buffered channel of
// configurable size; if Enqueue is called when the buffer is full and
// no consumer is reading, Enqueue blocks or returns ErrQueueFull
// depending on the Enqueue variant.
var ErrQueueFull = errors.New("dispatch: queue full")

// ErrQueueClosed is returned by Enqueue / Dequeue after the queue has
// been closed. The error is permanent: a closed queue cannot be
// re-opened. Callers should construct a new queue.
var ErrQueueClosed = errors.New("dispatch: queue closed")

// ErrUnknownSpec is returned by Ack / Nack when the spec is not in
// the in-flight set. This can happen if the spec was dequeued, the
// queue was closed, the spec was acked/nacked by another worker, or
// the spec is genuinely unknown. In all cases, the operation is a
// no-op from the dispatcher's perspective — the dispatcher will move
// on to the next spec.
var ErrUnknownSpec = errors.New("dispatch: unknown spec (not in-flight)")

// ErrSpecInvalid is returned by Enqueue when the spec fails validation
// (WorkerSpec.Validate). The queue does not accept malformed specs.
var ErrSpecInvalid = errors.New("dispatch: spec invalid")

// ----------------------------------------------------------------------------
// DispatchQueue interface
// ----------------------------------------------------------------------------

// DispatchQueue is the contract between the service layer (which
// produces specs) and the dispatcher (which consumes them). The
// interface is intentionally minimal — the queue is a thin
// pass-through with retry semantics, not a workflow engine.
type DispatchQueue interface {
	// Enqueue adds a spec to the queue. Returns:
	//   - ErrSpecInvalid if spec.Validate() fails
	//   - ErrQueueClosed if the queue has been closed
	//   - ErrQueueFull if the underlying buffer is full and the
	//     queue is configured as non-blocking (see
	//     InMemoryQueueOption.NonBlocking). For the default
	//     InMemoryQueue, Enqueue blocks until the buffer has room.
	Enqueue(ctx context.Context, spec aion.WorkerSpec) error

	// Dequeue blocks until a spec is available, the queue is closed,
	// or ctx is cancelled. Returns:
	//   - the spec on success
	//   - ErrQueueClosed if the queue was closed (no more items will
	//     be produced, and the buffer has been drained)
	//   - ctx.Err() if ctx was cancelled
	Dequeue(ctx context.Context) (aion.WorkerSpec, error)

	// Ack marks the spec as successfully completed. The implementation
	// is responsible for cleaning up the in-flight set. Idempotent:
	// calling Ack twice on the same spec is a no-op (the second call
	// returns ErrUnknownSpec because the spec is no longer in-flight).
	Ack(ctx context.Context, spec aion.WorkerSpec, result aion.WorkerResult) error

	// Nack marks the spec as failed. The implementation decides
	// whether to retry (re-queue with Attempt+1) or move to a
	// dead-letter state. The contract is "do the right thing":
	//   - if spec.Attempt < queue.MaxAttempts(): re-queue
	//   - otherwise: drop (or move to a DLQ, for a future
	//     Postgres-backed impl)
	// The reason is recorded for observability.
	//
	// The returned NackResult tells the dispatcher what the queue
	// did with the spec, so the dispatcher can update its stats
	// (Retries / Dropped). On ErrUnknownSpec the result is the
	// zero value (neither dropped nor retried).
	Nack(ctx context.Context, spec aion.WorkerSpec, result aion.WorkerResult, reason error) (NackResult, error)

	// Close marks the queue as closed. After Close:
	//   - Enqueue returns ErrQueueClosed
	//   - Dequeue drains remaining items then returns ErrQueueClosed
	//   - Ack / Nack continue to work for in-flight items
	// Close does not block on in-flight completion; the caller is
	// responsible for stopping the dispatcher first.
	Close() error

	// Len returns the number of pending items (not including
	// in-flight). For observability / metrics only; not a
	// synchronization point.
	Len() int
}

// ----------------------------------------------------------------------------
// NackResult
// ----------------------------------------------------------------------------

// NackResult is the outcome of a DispatchQueue.Nack call. It tells the
// dispatcher whether the spec was retried, dropped, or neither (e.g.
// ErrUnknownSpec). Exactly one of Dropped / Retried is set on a successful
// Nack; the zero value means no work was done (the spec was unknown).
//
// This was added as part of A-002-05 (dispatcher stats) so the
// dispatcher can correctly count Dropped (DLQ) and Retries outcomes
// without having to inspect the queue's internal counters (which would
// race under concurrent workers).
type NackResult struct {
	Dropped bool // spec hit the DLQ (attempts exhausted, queue closed, or ctx cancelled)
	Retried bool // spec was re-queued (attempts remaining)
}

// ----------------------------------------------------------------------------
// InMemoryQueue
// ----------------------------------------------------------------------------

// InMemoryQueueOption configures the InMemoryQueue at construction.
type InMemoryQueueOption func(*InMemoryQueue)

// WithBufferSize overrides the default pending-spec buffer size
// (default: 1024). A larger buffer trades memory for
// back-pressure-on-enqueue latency. For Sprint 5 the default is
// sufficient; tune for production if the dispatcher is slower than
// the enqueue rate.
func WithBufferSize(n int) InMemoryQueueOption {
	return func(q *InMemoryQueue) { q.bufferSize = n }
}

// WithMaxAttempts overrides the default retry cap (default: 3 =
// initial + 2 retries). A spec that fails more than MaxAttempts
// times is moved to a dead-letter state (dropped, for Sprint 5).
// TASK-508 (Recovery) will define the dead-letter semantics in
// more detail; for now we drop and log.
func WithMaxAttempts(n int) InMemoryQueueOption {
	return func(q *InMemoryQueue) { q.maxAttempts = n }
}

// WithNonBlocking makes Enqueue return ErrQueueFull instead of
// blocking when the buffer is full. Useful for tests + production
// paths that prefer explicit backpressure over latency.
func WithNonBlocking() InMemoryQueueOption {
	return func(q *InMemoryQueue) { q.nonBlocking = true }
}

// InMemoryQueue is the Sprint 5 DispatchQueue implementation.
//
// Concurrency model:
//
//   - Enqueue: writes to a buffered channel. Thread-safe (channels
//     are). May block if the buffer is full (unless WithNonBlocking).
//   - Dequeue: reads from the same channel. Thread-safe. May block
//     until a spec is available or the queue is closed.
//   - Ack / Nack: mutate a sync.Mutex-protected in-flight set.
//     Thread-safe.
//
// The queue does NOT persist across restarts. For Sprint 5 the
// service restart flushes all pending + in-flight specs. A Sprint
// 6 follow-up will add a Postgres-backed impl that persists both.
type InMemoryQueue struct {
	// pending is the channel of specs waiting to be dequeued.
	// Buffered; size is bufferSize.
	pending chan aion.WorkerSpec

	// mu guards inFlight, dlq, closed, and dropped.
	mu sync.Mutex

	// inFlight tracks specs that have been dequeued but not yet
	// acked/nacked. Map key is spec.ExecutionID. The value is the
	// spec, for diagnostic purposes.
	inFlight map[uuid.UUID]aion.WorkerSpec

	// dlq tracks specs that have been dropped (attempt >= max).
	// In Sprint 5 the dispatcher just logs these; a Sprint 6
	// Postgres impl will persist them.
	dlq []aion.WorkerSpec

	// closed is true after Close has been called. Guarded by mu.
	closed bool

	// dropped counts the total number of nacks that hit max
	// attempts. For observability / tests.
	dropped int

	// bufferSize is the size of the pending channel.
	bufferSize int

	// maxAttempts is the retry cap. Default 3.
	maxAttempts int

	// nonBlocking is true if Enqueue should return ErrQueueFull
	// instead of blocking when the buffer is full.
	nonBlocking bool

	// closeCh is closed when the queue is closed. Dequeue watches
	// this to break out of the read.
	closeCh chan struct{}
}

// NewInMemoryQueue constructs an InMemoryQueue with the given options.
// The queue is ready to use immediately; no Start() call is needed.
func NewInMemoryQueue(opts ...InMemoryQueueOption) *InMemoryQueue {
	q := &InMemoryQueue{
		bufferSize:  1024,
		maxAttempts: 3,
		closeCh:     make(chan struct{}),
	}
	for _, opt := range opts {
		opt(q)
	}
	q.pending = make(chan aion.WorkerSpec, q.bufferSize)
	q.inFlight = make(map[uuid.UUID]aion.WorkerSpec, q.bufferSize)
	return q
}

// Enqueue adds a spec to the queue. See DispatchQueue for error
// semantics. The spec is validated before being added; validation
// failures return ErrSpecInvalid wrapping the underlying
// aion.ErrInvalidSpec.
func (q *InMemoryQueue) Enqueue(ctx context.Context, spec aion.WorkerSpec) error {
	if err := spec.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrSpecInvalid, err)
	}
	q.mu.Lock()
	closed := q.closed
	q.mu.Unlock()
	if closed {
		return ErrQueueClosed
	}
	if q.nonBlocking {
		select {
		case q.pending <- spec:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		default:
			return ErrQueueFull
		}
	}
	select {
	case q.pending <- spec:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-q.closeCh:
		return ErrQueueClosed
	}
}

// Dequeue blocks until a spec is available, the queue is closed, or
// ctx is cancelled. See DispatchQueue for error semantics.
func (q *InMemoryQueue) Dequeue(ctx context.Context) (aion.WorkerSpec, error) {
	// First check if there's a pending spec we can grab immediately
	// (non-blocking path). If not, block on the channel.
	for {
		select {
		case spec, ok := <-q.pending:
			if !ok {
				return aion.WorkerSpec{}, ErrQueueClosed
			}
			q.mu.Lock()
			q.inFlight[spec.ExecutionID] = spec
			q.mu.Unlock()
			return spec, nil
		case <-ctx.Done():
			return aion.WorkerSpec{}, ctx.Err()
		case <-q.closeCh:
			// Drain remaining items, then return ErrQueueClosed.
			// This is so a "closed but not empty" queue can be
			// drained by a graceful-shutdown dispatcher.
			select {
			case spec, ok := <-q.pending:
				if !ok {
					return aion.WorkerSpec{}, ErrQueueClosed
				}
				q.mu.Lock()
				q.inFlight[spec.ExecutionID] = spec
				q.mu.Unlock()
				return spec, nil
			default:
				return aion.WorkerSpec{}, ErrQueueClosed
			}
		}
	}
}

// Ack marks the spec as successfully completed. Removes it from
// the in-flight set. Idempotent: a second call returns
// ErrUnknownSpec.
func (q *InMemoryQueue) Ack(ctx context.Context, spec aion.WorkerSpec, result aion.WorkerResult) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if _, ok := q.inFlight[spec.ExecutionID]; !ok {
		return ErrUnknownSpec
	}
	delete(q.inFlight, spec.ExecutionID)
	return nil
}

// Nack marks the spec as failed. If spec.Attempt < q.maxAttempts,
// the spec is re-queued with Attempt+1. Otherwise, it is dropped
// (added to the dead-letter list). The reason is recorded for
// observability but not persisted in Sprint 5.
//
// The returned NackResult tells the dispatcher what happened to the
// spec so it can update its stats (Retries / Dropped). See NackResult.
func (q *InMemoryQueue) Nack(ctx context.Context, spec aion.WorkerSpec, result aion.WorkerResult, reason error) (NackResult, error) {
	q.mu.Lock()
	_, inFlight := q.inFlight[spec.ExecutionID]
	closed := q.closed
	q.mu.Unlock()
	if !inFlight {
		return NackResult{}, ErrUnknownSpec
	}
	if closed {
		// If the queue is closed, we can't re-queue. Drop.
		q.mu.Lock()
		delete(q.inFlight, spec.ExecutionID)
		q.dropped++
		q.dlq = append(q.dlq, spec)
		q.mu.Unlock()
		return NackResult{Dropped: true}, nil
	}
	if spec.Attempt < q.maxAttempts {
		// Re-queue with Attempt+1.
		retry := spec
		retry.Attempt = spec.Attempt + 1
		// We remove from in-flight BEFORE re-queueing, so a
		// concurrent Ack on the same spec won't see it. The
		// dispatcher's worker is the only one calling Nack for a
		// given spec, so this is safe in practice.
		q.mu.Lock()
		delete(q.inFlight, spec.ExecutionID)
		q.mu.Unlock()
		// Re-queue. We use a non-blocking select with a fallback
		// to blocking-on-close to avoid losing the retry if the
		// queue is being closed concurrently.
		select {
		case q.pending <- retry:
			return NackResult{Retried: true}, nil
		case <-ctx.Done():
			// Re-queue failed (ctx cancelled). Put the spec back
			// in-flight so a future Ack/Nack can find it. Note:
			// this is a corner case; the dispatcher typically
			// doesn't cancel ctx mid-flight.
			q.mu.Lock()
			q.inFlight[retry.ExecutionID] = retry
			q.mu.Unlock()
			return NackResult{}, ctx.Err()
		}
	}
	// Attempt >= max → dead-letter (drop).
	q.mu.Lock()
	delete(q.inFlight, spec.ExecutionID)
	q.dropped++
	q.dlq = append(q.dlq, spec)
	q.mu.Unlock()
	return NackResult{Dropped: true}, nil
}

// Close marks the queue as closed. Enqueue will return ErrQueueClosed;
// Dequeue will drain remaining items then return ErrQueueClosed. Ack
// and Nack continue to work for in-flight items. Close is idempotent.
func (q *InMemoryQueue) Close() error {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return nil
	}
	q.closed = true
	q.mu.Unlock()
	close(q.closeCh)
	// We don't close the pending channel directly — Dequeue handles
	// the closeCh path. If we close `pending`, a blocked Enqueue
	// would panic ("send on closed channel"). The closeCh lets
	// blocked Enqueue calls return ErrQueueClosed cleanly.
	return nil
}

// Len returns the number of pending items (not including in-flight).
// O(1). Lock-free: reads `len(q.pending)` which is a channel built-in
// and atomic.
func (q *InMemoryQueue) Len() int {
	return len(q.pending)
}

// DroppedCount returns the total number of specs that have been
// nacked to the dead-letter list. For observability / tests.
func (q *InMemoryQueue) DroppedCount() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.dropped
}

// InFlightCount returns the number of currently in-flight specs.
// For observability / tests.
func (q *InMemoryQueue) InFlightCount() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.inFlight)
}
