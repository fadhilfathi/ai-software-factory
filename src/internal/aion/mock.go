// MockRuntime is the in-process, FakeScript-driven implementation of
// the Runtime interface (Mode A in docs/sprint5/integration-test-plan.md
// §2). Used by:
//
//   - Unit tests in src/internal/service/execution_test.go
//   - Dev / local integration tests (no subprocess)
//   - The CI gate by default (Mode B is gated by AION_E2E=1)
//
// Behaviour is controlled by a per-spec FakeScript, supplied either
// at Spawn time (MockSpawn) or pre-registered (RegisterScript). The
// default script (zero value) completes the worker immediately with
// an empty result and no error — matches the existing mock goroutine's
// "happy path" semantics so existing tests continue to pass without
// changes.
package aion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// FakeScript is the MockRuntime's per-worker behaviour recipe. Zero
// value is the "happy path": complete immediately with empty result.
type FakeScript struct {
	// Delay before transitioning to the terminal outcome. 0 means
	// "complete as soon as Spawn returns".
	Delay time.Duration

	// Outcome is the terminal status the worker transitions to.
	// Defaults to WorkerCompleted when zero.
	Outcome WorkerStatus

	// Result is the body for completed workers. Ignored for other
	// outcomes. Optional.
	Result json.RawMessage

	// ErrorMessage is the failure reason for failed workers.
	// Ignored for other outcomes.
	ErrorMessage string
}

// mockWorker is the MockRuntime's per-worker state.
type mockWorker struct {
	spec     WorkerSpec
	script   FakeScript
	resultCh chan WorkerResult
	cancelCh chan struct{}
	done     atomic.Bool
}

// resultChSize is the buffer for the result channel. 1 is enough —
// Wait consumes exactly one result.
const resultChSize = 1

// MockRuntime is the in-process Runtime. Thread-safe.
type MockRuntime struct {
	mu      sync.Mutex
	workers map[WorkerHandle]*mockWorker
	closed  atomic.Bool
	closeCh chan struct{}

	// perSpecScripts lets a test pre-register a script for a
	// specific spec key (e.g. by ExecutionID) so the Spawn call
	// doesn't need to carry it. Nil map means "use the default
	// script" for every spawn.
	perSpecScripts map[string]FakeScript

	// defaultScript is the fallback FakeScript applied when no
	// per-spec script matches. Zero value is "complete immediately
	// with empty result". Tests that don't know the execution ID
	// in advance can set this once to apply to every spawn.
	defaultScript FakeScript

	// counter is the spawn counter, used to make handles unique
	// even if ExecutionID is the same across spawns (rare, but
	// allowed — the dispatch queue enforces uniqueness at the
	// item level, not at the ExecutionID level).
	counter atomic.Uint64
}

// NewMockRuntime constructs an empty MockRuntime. Callers can
// optionally pre-register per-spec scripts via RegisterScript or
// per-spawn scripts via MockSpawn.
func NewMockRuntime() *MockRuntime {
	return &MockRuntime{
		workers:        make(map[WorkerHandle]*mockWorker),
		closeCh:        make(chan struct{}),
		perSpecScripts: nil, // lazy-initialised on first RegisterScript
	}
}

// RegisterScript associates a FakeScript with the given spec key.
// The key is usually the ExecutionID's string form, but anything
// unique per worker works. Calling RegisterScript for the same key
// overwrites the previous entry.
//
// Tests that want every worker to behave the same way can pass the
// empty string "" as the key.
func (m *MockRuntime) RegisterScript(key string, script FakeScript) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.perSpecScripts == nil {
		m.perSpecScripts = make(map[string]FakeScript)
	}
	m.perSpecScripts[key] = script
}

// scriptFor returns the registered script for a spec, or the
// zero-value script (immediate complete) if none is registered.
func (m *MockRuntime) scriptFor(spec WorkerSpec) FakeScript {
	m.mu.Lock()
	defer m.mu.Unlock()
	if script, ok := m.perSpecScripts[spec.ExecutionID.String()]; ok {
		return script
	}
	return m.defaultScript
}

// SetDefaultScript registers a fallback script applied when no
// per-spec script is found. Useful for tests that don't know the
// execution ID in advance (it's generated inside CreateExecution)
// and want every spawned worker to use the same outcome. Calling
// with a zero-value FakeScript clears the fallback. The method is
// safe to call from multiple goroutines.
func (m *MockRuntime) SetDefaultScript(script FakeScript) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultScript = script
}

// ----------------------------------------------------------------------------
// Runtime interface
// ----------------------------------------------------------------------------

// Spawn registers a new mock worker and returns a handle. The
// worker starts a goroutine that runs the script (delay → terminal
// status) and pushes the result on the result channel for Wait to
// consume.
func (m *MockRuntime) Spawn(ctx context.Context, spec WorkerSpec) (WorkerHandle, error) {
	if m.closed.Load() {
		return "", ErrRuntimeClosed
	}
	if err := spec.Validate(); err != nil {
		return "", err
	}

	handle := WorkerHandle(fmt.Sprintf("mock-%d-%s", m.counter.Add(1), spec.ExecutionID.String()))
	script := m.scriptFor(spec)

	worker := &mockWorker{
		spec:     spec,
		script:   script,
		resultCh: make(chan WorkerResult, resultChSize),
		cancelCh: make(chan struct{}),
	}

	m.mu.Lock()
	m.workers[handle] = worker
	m.mu.Unlock()

	go m.run(worker, handle, script)

	return handle, nil
}

// run is the per-worker goroutine: sleeps for the script's delay
// (or until cancel) and then produces the result.
func (m *MockRuntime) run(worker *mockWorker, handle WorkerHandle, script FakeScript) {
	startedAt := time.Now().UTC()

	// Fast path: zero delay, no cancel race.
	if script.Delay <= 0 {
		m.terminate(worker, handle, startedAt, script)
		return
	}

	timer := time.NewTimer(script.Delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		m.terminate(worker, handle, startedAt, script)
	case <-worker.cancelCh:
		// Cancelled before the timer fired.
		completedAt := time.Now().UTC()
		_ = completedAt
		m.pushResult(worker, WorkerResult{
			Handle:       handle,
			ExecutionID:  worker.spec.ExecutionID,
			Status:       WorkerCancelled,
			StartedAt:    startedAt,
			CompletedAt:  completedAt,
			ErrorMessage: "cancelled by runtime",
		})
	case <-m.closeCh:
		// Runtime was closed. Treat as a normal cancel.
		completedAt := time.Now().UTC()
		m.pushResult(worker, WorkerResult{
			Handle:       handle,
			ExecutionID:  worker.spec.ExecutionID,
			Status:       WorkerCancelled,
			StartedAt:    startedAt,
			CompletedAt:  completedAt,
			ErrorMessage: "runtime closed",
		})
	}
}

// terminate computes the terminal WorkerResult for a non-cancelled
// worker and pushes it.
func (m *MockRuntime) terminate(worker *mockWorker, handle WorkerHandle, startedAt time.Time, script FakeScript) {
	completedAt := time.Now().UTC()

	outcome := script.Outcome
	if outcome == "" {
		outcome = WorkerCompleted
	}

	result := WorkerResult{
		Handle:       handle,
		ExecutionID:  worker.spec.ExecutionID,
		Status:       outcome,
		StartedAt:    startedAt,
		CompletedAt:  completedAt,
		Result:       script.Result,
		ErrorMessage: script.ErrorMessage,
	}

	if !outcome.IsTerminal() {
		// Defensive: if the script specified a non-terminal
		// outcome (e.g. "running"), coerce to "failed" with
		// a clear message. Validate() can't catch this
		// because it only enforces a known set of status
		// strings for the worker, not for the script.
		result.Status = WorkerFailed
		result.ErrorMessage = fmt.Sprintf("mock: script specified non-terminal outcome %q; coerced to failed", outcome)
	}

	m.pushResult(worker, result)
}

// pushResult sends a result to Wait exactly once. Uses done
// (atomic.Bool) to guard against double-push from cancel + close
// racing.
func (m *MockRuntime) pushResult(worker *mockWorker, result WorkerResult) {
	if !worker.done.CompareAndSwap(false, true) {
		return
	}
	worker.resultCh <- result
}

// Wait blocks until the worker reaches a terminal status or the
// context is cancelled.
func (m *MockRuntime) Wait(ctx context.Context, handle WorkerHandle) (WorkerResult, error) {
	if m.closed.Load() {
		return WorkerResult{}, ErrRuntimeClosed
	}

	m.mu.Lock()
	worker, ok := m.workers[handle]
	m.mu.Unlock()
	if !ok {
		return WorkerResult{}, ErrWorkerNotFound
	}

	select {
	case result := <-worker.resultCh:
		return result, nil
	case <-ctx.Done():
		// Context cancellation: the worker may still be
		// running. We DON'T cancel it here — that's the
		// caller's job (Cancel). We just return
		// ErrWorkerTimeout so the service can decide what
		// to do (typically: cancel + retry).
		return WorkerResult{}, fmt.Errorf("%w: %v", ErrWorkerTimeout, ctx.Err())
	case <-m.closeCh:
		// Runtime closed: drain any pending result, else
		// return ErrRuntimeClosed.
		select {
		case result := <-worker.resultCh:
			return result, nil
		default:
			return WorkerResult{}, ErrRuntimeClosed
		}
	}
}

// Cancel signals the worker to stop. Idempotent. Returns
// ErrWorkerNotFound for unknown handles.
func (m *MockRuntime) Cancel(ctx context.Context, handle WorkerHandle) error {
	m.mu.Lock()
	worker, ok := m.workers[handle]
	m.mu.Unlock()
	if !ok {
		return ErrWorkerNotFound
	}

	// Non-blocking close; the run() goroutine handles it.
	select {
	case <-worker.cancelCh:
		// already closed
	default:
		close(worker.cancelCh)
	}

	// Ensure Wait returns even if the run goroutine never gets
	// to its select (e.g. if it had a 0-delay fast path and
	// already terminated). The done flag in pushResult is the
	// guard against double-push.
	m.pushResult(worker, WorkerResult{
		Handle:       handle,
		ExecutionID:  worker.spec.ExecutionID,
		Status:       WorkerCancelled,
		StartedAt:    time.Now().UTC(),
		CompletedAt:  time.Now().UTC(),
		ErrorMessage: "cancelled by runtime",
	})
	return nil
}

// Close stops accepting new spawns and cancels all in-flight
// workers. Wait calls return ErrRuntimeClosed.
func (m *MockRuntime) Close() error {
	if !m.closed.CompareAndSwap(false, true) {
		return nil // already closed
	}
	close(m.closeCh)

	// Best-effort: cancel all workers.
	m.mu.Lock()
	workers := make([]*mockWorker, 0, len(m.workers))
	for _, w := range m.workers {
		workers = append(workers, w)
	}
	m.mu.Unlock()

	for _, w := range workers {
		select {
		case <-w.cancelCh:
		default:
			close(w.cancelCh)
		}
	}
	return nil
}

// ActiveWorkers returns the count of workers that are still in
// the map (terminal or not). Useful for tests that want to assert
// "all workers cleaned up" after Close.
func (m *MockRuntime) ActiveWorkers() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.workers)
}

// Forget removes a worker from the map. Tests use this to assert
// that the cleanup path actually drops the reference (otherwise the
// runtime holds the worker struct indefinitely).
func (m *MockRuntime) Forget(handle WorkerHandle) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.workers, handle)
}

// ----------------------------------------------------------------------------
// Compile-time interface check
// ----------------------------------------------------------------------------

var _ Runtime = (*MockRuntime)(nil)

// Silence the "imported and not used" linter in tests that import
// the package for type-only reasons.
var _ = errors.New
var _ = uuid.Nil
