package dispatch

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/fadhilfathi/AI-Software-Factory/internal/aion"
	"go.uber.org/zap"
)

// ----------------------------------------------------------------------------
// Dispatcher
// ----------------------------------------------------------------------------

// Dispatcher is a fixed-size pool of worker goroutines that pull
// WorkerSpecs from a DispatchQueue and drive aion.Runtime.Spawn +
// Wait. On terminal status, the dispatcher calls Ack (success) or
// Nack (failure) on the queue, which handles retry-vs-DLQ decisions.
//
// The dispatcher is intentionally tenant-agnostic — it just routes
// specs to the runtime. Cross-tenant validation is the service's
// job (it builds the spec, not the queue).
//
// Lifecycle:
//
//	d := NewDispatcher(q, runtime, log)
//	d.Start(ctx, 4)              // 4 workers
//	... enqueue specs from the service layer ...
//	err := d.Stop(ctx)           // graceful shutdown: wait for in-flight to finish
//
// Concurrency: Start is idempotent (multiple calls are a no-op).
// Stop is idempotent (multiple calls block on the same wait group).
type Dispatcher struct {
	queue   DispatchQueue
	runtime aion.Runtime
	log     *zap.Logger

	mu      sync.Mutex
	workers int
	wg      sync.WaitGroup
	cancel  context.CancelFunc
	running bool

	// stats for observability. Guarded by mu.
	stats DispatcherStats
}

// DispatcherStats is a snapshot of dispatch counters. Read with
// Dispatcher.Stats(). All counters are monotonic for the lifetime
// of the dispatcher (Start to Stop).
type DispatcherStats struct {
	Spawned   int64 // total runtime.Spawn calls
	Completed int64 // total Ack'd (worker returned WorkerCompleted)
	Failed    int64 // total Nack'd with reason (worker failed or runtime errored)
	Cancelled int64 // total Nack'd due to ctx cancellation
	Retries   int64 // total Nack's that triggered a retry (Attempt < max)
	Dropped   int64 // total Nack's that hit the DLQ (Attempt >= max)
}

// NewDispatcher constructs a Dispatcher. queue and runtime are
// required; log is used for structured logging at INFO/WARN/ERROR.
func NewDispatcher(queue DispatchQueue, runtime aion.Runtime, log *zap.Logger) *Dispatcher {
	if log == nil {
		log = zap.NewNop()
	}
	return &Dispatcher{queue: queue, runtime: runtime, log: log}
}

// Start launches `workers` worker goroutines. The dispatcher's
// lifecycle is tied to ctx: when ctx is cancelled, the workers
// exit cleanly. Subsequent calls to Start are no-ops (the
// dispatcher can only be started once).
func (d *Dispatcher) Start(ctx context.Context, workers int) error {
	if workers < 1 {
		return fmt.Errorf("dispatcher: workers must be >= 1, got %d", workers)
	}
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return nil
	}
	d.running = true
	d.workers = workers
	workerCtx, cancel := context.WithCancel(ctx)
	d.cancel = cancel
	d.mu.Unlock()

	for i := 0; i < workers; i++ {
		d.wg.Add(1)
		go d.workerLoop(workerCtx, i)
	}
	d.log.Info("dispatcher started", zap.Int("workers", workers))
	return nil
}

// workerLoop is the per-goroutine worker. It dequeues specs and
// drives them through the runtime. On terminal status (or runtime
// error), it acks or nacks the queue.
func (d *Dispatcher) workerLoop(ctx context.Context, id int) {
	defer d.wg.Done()
	log := d.log.With(zap.Int("worker", id))
	log.Debug("worker loop started")
	for {
		select {
		case <-ctx.Done():
			log.Debug("worker loop exiting (ctx done)")
			return
		default:
		}
		spec, err := d.queue.Dequeue(ctx)
		if err != nil {
			if errors.Is(err, ErrQueueClosed) {
				log.Debug("worker loop exiting (queue closed)")
				return
			}
			if errors.Is(err, context.Canceled) {
				log.Debug("worker loop exiting (ctx cancelled)")
				return
			}
			// Any other error (context.DeadlineExceeded, etc.):
			// log and continue. A ctx.DeadlineExceeded typically
			// means the dispatcher is using a per-Dequeue timeout,
			// which we don't currently do, but a future change
			// might.
			log.Warn("dequeue error", zap.Error(err))
			continue
		}
		d.handleSpec(ctx, log, spec)
	}
}

// handleSpec drives one spec through the runtime: Spawn, then Wait,
// then Ack/Nack. Failures along the way are Nack'd with the
// corresponding reason. The retry-vs-DLQ decision lives in the
// queue (Nack), not in the dispatcher.
func (d *Dispatcher) handleSpec(ctx context.Context, log *zap.Logger, spec aion.WorkerSpec) {
	log = log.With(
		zap.String("execution_id", spec.ExecutionID.String()),
		zap.String("task_id", spec.TaskID.String()),
		zap.String("agent_id", spec.AgentID.String()),
		zap.String("project_id", spec.ProjectID.String()),
		zap.Int("attempt", spec.Attempt),
	)

	// Spawn. If Spawn returns an error, the worker is not
	// running — nack and let the queue decide retry.
	d.mu.Lock()
	d.stats.Spawned++
	d.mu.Unlock()
	handle, err := d.runtime.Spawn(ctx, spec)
	if err != nil {
		log.Warn("spawn failed", zap.Error(err))
		d.mu.Lock()
		d.stats.Failed++
		d.mu.Unlock()
		if nackErr := d.queue.Nack(ctx, spec, aion.WorkerResult{ExecutionID: spec.ExecutionID, Status: aion.WorkerFailed}, err); nackErr != nil && !errors.Is(nackErr, ErrUnknownSpec) {
			log.Warn("nack after spawn error failed", zap.Error(nackErr))
		}
		return
	}
	log = log.With(zap.String("handle", string(handle)))

	// Wait for the worker to reach a terminal status. Wait blocks
	// until the worker is done or ctx is cancelled.
	result, err := d.runtime.Wait(ctx, handle)
	if err != nil {
		// Wait can fail for two reasons:
		//   1. The runtime errored (e.g., child process died
		//      unexpectedly). We don't have a clean status;
		//      Nack with WorkerFailed and the error.
		//   2. ctx was cancelled. We mark the result as
		//      WorkerCancelled and Nack.
		status := aion.WorkerFailed
		if errors.Is(err, context.Canceled) {
			status = aion.WorkerCancelled
		}
		result = aion.WorkerResult{
			Handle:       handle,
			ExecutionID:  spec.ExecutionID,
			Status:       status,
			ErrorMessage: err.Error(),
		}
		log.Warn("wait error", zap.String("status", string(status)), zap.Error(err))
		d.mu.Lock()
		if status == aion.WorkerCancelled {
			d.stats.Cancelled++
		} else {
			d.stats.Failed++
		}
		d.mu.Unlock()
		if nackErr := d.queue.Nack(ctx, spec, result, err); nackErr != nil && !errors.Is(nackErr, ErrUnknownSpec) {
			log.Warn("nack after wait error failed", zap.Error(nackErr))
		}
		return
	}

	// Worker reached a terminal status. Ack if completed, Nack
	// otherwise (failed or cancelled).
	log.Info("worker terminal",
		zap.String("status", string(result.Status)),
		zap.Time("started_at", result.StartedAt),
		zap.Time("completed_at", result.CompletedAt),
	)
	switch result.Status {
	case aion.WorkerCompleted:
		d.mu.Lock()
		d.stats.Completed++
		d.mu.Unlock()
		if ackErr := d.queue.Ack(ctx, spec, result); ackErr != nil && !errors.Is(ackErr, ErrUnknownSpec) {
			log.Warn("ack after completion failed", zap.Error(ackErr))
		}
	default:
		// WorkerFailed or WorkerCancelled — Nack with the result
		// as the reason. The queue decides retry.
		d.mu.Lock()
		if result.Status == aion.WorkerCancelled {
			d.stats.Cancelled++
		} else {
			d.stats.Failed++
		}
		d.mu.Unlock()
		reason := fmt.Errorf("worker terminal status: %s (error: %s)", result.Status, result.ErrorMessage)
		if nackErr := d.queue.Nack(ctx, spec, result, reason); nackErr != nil && !errors.Is(nackErr, ErrUnknownSpec) {
			log.Warn("nack after terminal status failed", zap.Error(nackErr))
		}
	}
}

// Stop gracefully shuts down the dispatcher. It:
//  1. Cancels the worker context, signalling workers to exit
//     after their current spec.
//  2. Waits for all workers to finish (with the supplied ctx's
//     deadline). If ctx is cancelled before workers finish, Stop
//     returns ctx.Err() and the workers are leaked (they will
//     exit when the underlying runtime closes, or the process
//     exits).
//
// Stop is idempotent.
func (d *Dispatcher) Stop(ctx context.Context) error {
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return nil
	}
	d.running = false
	cancel := d.cancel
	d.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		d.log.Info("dispatcher stopped",
			zap.Int64("spawned", d.stats.Spawned),
			zap.Int64("completed", d.stats.Completed),
			zap.Int64("failed", d.stats.Failed),
			zap.Int64("cancelled", d.stats.Cancelled),
		)
		return nil
	case <-ctx.Done():
		d.log.Warn("dispatcher stop timed out", zap.Error(ctx.Err()))
		return ctx.Err()
	}
}

// Stats returns a snapshot of the dispatcher's counters.
func (d *Dispatcher) Stats() DispatcherStats {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.stats
}

// Workers returns the configured worker count (0 if not started).
func (d *Dispatcher) Workers() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.workers
}

// ----------------------------------------------------------------------------
// Convenience constructor for tests
// ----------------------------------------------------------------------------

// NewTestDispatcher returns a Dispatcher with a noop logger. Used
// by tests that don't care about logging.
func NewTestDispatcher(queue DispatchQueue, runtime aion.Runtime) *Dispatcher {
	return NewDispatcher(queue, runtime, zap.NewNop())
}
