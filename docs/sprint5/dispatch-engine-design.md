# Task Dispatch Engine (TASK-502) — Design

> Sprint 5 · Developer-01 · branch `feat/sprint-5-task-502-task-dispatch`
> Status: prep complete on branch, NOT committed, awaiting Lead review.
> Brief: docs/sprint5/brief.md §6.2 (Lead's broadcast, 2026-06-14)

## 0. Why this exists

TASK-501 shipped the Aion Runtime abstraction (aion.Runtime
interface, MockRuntime + ProcessRuntime impls) and wired it into
the ExecutionService. But the ExecutionService still invokes the
runtime synchronously on its own goroutine per execution.

For Sprint 5+ the runtime invocations need to be:
- **Concurrent** — multiple executions in flight at once
- **Buffered** — the service layer can enqueue work faster than
  the runtime can consume
- **Retried** — failed executions get retried with backoff
- **Decoupled** — the service layer doesn't directly own the
  runtime; it just enqueues and forgets

The Dispatch Engine is the layer that provides these properties. It
sits between the service layer (which enqueues `aion.WorkerSpec`
items) and the runtime (which consumes them).

## 1. Architecture

```
┌──────────────────┐
│ ExecutionService │ (calls q.Enqueue)
│ (service/        │
│  execution.go)   │
└────────┬─────────┘
         │ aion.WorkerSpec
         ▼
┌──────────────────┐
│  DispatchQueue   │ (interface; InMemoryQueue impl)
│  (dispatch/      │
│   queue.go)      │
└────────┬─────────┘
         │ Dequeue (blocking)
         ▼
┌──────────────────┐
│  Dispatcher      │ (N worker goroutines)
│  (dispatch/      │
│   dispatcher.go) │
└────────┬─────────┘
         │ aion.WorkerSpec
         ▼
┌──────────────────┐
│  aion.Runtime    │ (Mock or Process)
└──────────────────┘
```

Key design choices:
- **Channel + mutex** for concurrency in the queue. The
  channel is the producer-consumer buffer; the mutex protects
  the in-flight + DLQ state.
- **Fixed-size worker pool** in the dispatcher. Sprint 5 ships
  with a small default (4 workers); the caller can override.
- **Ack/Nack for retry** — the queue owns the retry-vs-DLQ
  decision (not the dispatcher). This makes the policy easy
  to swap (Sprint 6 Postgres-backed queue can have a
  different policy).
- **Bounded backpressure** — InMemoryQueue uses a buffered
  channel (default 1024) so Enqueue doesn't block forever
  under sustained pressure. A `WithNonBlocking` option makes
  Enqueue return `ErrQueueFull` immediately instead of
  blocking.

## 2. DispatchQueue interface

```go
type DispatchQueue interface {
    Enqueue(ctx context.Context, spec aion.WorkerSpec) error
    Dequeue(ctx context.Context) (aion.WorkerSpec, error)
    Ack(ctx context.Context, spec aion.WorkerSpec, result aion.WorkerResult) error
    Nack(ctx context.Context, spec aion.WorkerSpec, result aion.WorkerResult, reason error) error
    Close() error
    Len() int
}
```

### Enqueue error semantics

- `ErrSpecInvalid` if `spec.Validate()` fails. The queue does
  not accept malformed specs. Validation runs BEFORE the closed
  check so a malformed spec returns the more useful error.
- `ErrQueueClosed` if the queue has been closed.
- `ErrQueueFull` if the buffer is full and the queue is in
  non-blocking mode (configured via `WithNonBlocking()`).
- `ctx.Err()` if the caller's context is cancelled.

### Dequeue error semantics

- Returns the spec on success.
- `ErrQueueClosed` if the queue was closed and the buffer has
  been drained. (Closing the queue does NOT discard pending
  items — they're drained by the dispatcher first.)
- `ctx.Err()` if the caller's context is cancelled.

### Ack/Nack error semantics

- `ErrUnknownSpec` if the spec is not in-flight. This is
  idempotent-safe: a second Ack on the same spec returns this
  error and the dispatcher treats it as a no-op (already
  handled).
- `nil` on success.
- The reason argument to Nack is recorded for observability
  but not persisted in Sprint 5.

### Close semantics

- Idempotent: a second Close is a no-op.
- After Close: Enqueue returns `ErrQueueClosed`, Dequeue drains
  remaining items then returns `ErrQueueClosed`, Ack/Nack
  continue to work for in-flight items.
- Close does NOT wait for in-flight items to complete. The
  caller is responsible for stopping the dispatcher first.

## 3. InMemoryQueue implementation

### Data structures

```go
type InMemoryQueue struct {
    pending  chan aion.WorkerSpec           // buffered, size = bufferSize
    mu       sync.Mutex                     // protects the rest
    inFlight map[uuid.UUID]aion.WorkerSpec  // execID → spec
    dlq      []aion.WorkerSpec              // dropped (attempt >= max)
    closed   bool                           // guarded by mu
    dropped  int                            // count of DLQ'd
    closeCh  chan struct{}                  // closed by Close()
    bufferSize  int                         // configurable
    maxAttempts int                          // default 3
    nonBlocking bool                         // configurable
}
```

### Concurrency model

- **Enqueue**: writes to `pending` (channel, atomic). Mutex
  only used to read the `closed` flag. Mutex is not held
  during the channel send, so Enqueue never blocks on the
  mutex.
- **Dequeue**: reads from `pending` (channel, atomic). Adds
  the spec to `inFlight` under the mutex.
- **Ack**: removes from `inFlight` under the mutex.
- **Nack**: reads `inFlight` + `closed` under the mutex,
  releases the mutex, then re-queues via the channel (or
  drops into `dlq` if at max attempts). The mutex is not
  held during the channel send, so Nack never blocks on the
  mutex.

### Retry policy (Nack)

If `spec.Attempt < maxAttempts`:
- Remove from `inFlight` (mutex)
- Re-queue with `Attempt+1` (channel send)

If `spec.Attempt >= maxAttempts`:
- Remove from `inFlight` (mutex)
- Append to `dlq` (mutex)
- Increment `dropped` counter (mutex)

The Sprint 5 dispatcher increments `Stats.Dropped` when it
sees the `dlq` change. A Sprint 6 follow-up will persist the
DLQ in Postgres (TASK-502 follow-up + recovery table from
§6.2 of the brief).

## 4. Dispatcher

### Lifecycle

```go
d := NewDispatcher(q, rt, log)
d.Start(ctx, 4)            // 4 workers
... enqueue specs ...
err := d.Stop(ctx)         // graceful shutdown
```

`Start` is idempotent (a second call is a no-op). `Stop` is
idempotent (a second call returns nil).

### Per-worker loop

```go
for {
    spec, err := queue.Dequeue(ctx)
    if err != nil {
        if ErrQueueClosed || ctx.Canceled { return }
        // log and continue
    }
    handle, err := runtime.Spawn(ctx, spec)
    if err != nil {
        queue.Nack(ctx, spec, WorkerResult{Status: WorkerFailed}, err)
        continue
    }
    result, err := runtime.Wait(ctx, handle)
    if err != nil {
        // either runtime error or ctx cancellation
        queue.Nack(ctx, spec, result, err)
        continue
    }
    if result.Status == WorkerCompleted {
        queue.Ack(ctx, spec, result)
    } else {
        queue.Nack(ctx, spec, result, fmt.Errorf("terminal: %s", result.Status))
    }
}
```

### Stats

The dispatcher maintains monotonic counters for observability:

```go
type DispatcherStats struct {
    Spawned   int64  // total runtime.Spawn calls
    Completed int64  // total Ack'd
    Failed    int64  // total Nack'd with reason (worker failed or runtime errored)
    Cancelled int64  // total Nack'd due to ctx cancellation
    Retries   int64  // not yet populated (the queue tracks these)
    Dropped   int64  // not yet populated (the queue tracks these)
}
```

Future work: expose queue-internal retry/DLQ counters via
`queue.Stats()` (Sprint 6 follow-up).

## 5. Cross-tenant invariants

`aion.WorkerSpec` carries `ProjectID`. The queue preserves it
on retry. The dispatcher is tenant-agnostic — it just routes
specs to the runtime.

The service layer is responsible for tenant validation BEFORE
enqueuing (the existing F-016 cross-tenant checks in
`service/execution.go` already do this). The dispatch package
does not re-validate.

## 6. Sprint 5 acceptance

- [x] `src/internal/dispatch/queue.go` — DispatchQueue interface + InMemoryQueue
- [x] `src/internal/dispatch/dispatcher.go` — Dispatcher with worker pool
- [x] `src/internal/dispatch/queue_test.go` — 13 unit tests
- [x] `src/internal/dispatch/dispatcher_test.go` — 7 unit tests
- [ ] `src/internal/integration/dispatch_test.go` — end-to-end test (deferred; will be added on Lead approval)
- [ ] `go build ./...` passes (waiting for no-Go-on-Windows host verification)
- [ ] `go test ./internal/dispatch/...` passes (waiting for same)
- [ ] NO COMMIT (per Lead's "no commit" rule for prep work)

## 7. Out of scope (Sprint 6+)

- **Postgres-backed DispatchQueue** — persist pending + in-flight
  + DLQ across restarts. The `dispatch_queue` Postgres table
  sketch is in §6.2 of the brief.
- **Recovery (TASK-508)** — alternative-agent dispatch on
  ErrCapabilityMismatch, jittered backoff, RECOVERY_TOTAL_BUDGET
  env var. The dispatch package's Nack hook is the right
  insertion point; TASK-508 will wire it.
- **Heartbeat / progress check** (Obs 1 from brief round 2) —
  the dispatcher's Wait() can be extended to check
  `last_progress_at` and Nack if the worker is alive but
  stalled. Sprint 6 follow-up.
- **Stale-claim reaper** (Obs 2) — a background goroutine
  that Nacks specs whose workers have died without
  reporting. Sprint 6 follow-up.
- **/v1/tasks/:id/execute endpoint** — the brief asks for
  Enqueue on `POST /v1/tasks/:id/execute` (new endpoint)
  or auto-enqueue on assignment. The dispatch package is
  handler-agnostic; the routing decision lives in
  `service/execution.go` and `handler/execution.go`.
  Adding the endpoint is a separate task (TASK-509 or
  TASK-510 follow-up).

## 8. Open questions

None for the design itself. The brief is clear. The integration
test should be added once Lead approves the prep and I can run
`go test` to verify.

## 9. References

- TASK-501 design: `docs/sprint5/aion-runtime-integration.md`
  (Q3 §0 design decisions, especially §0.2 Sprint 5 reality
  check that explains why MockRuntime is canonical for Sprint 5)
- Brief §6.2: Task Dispatch Engine
- Lead's TASK-502 GO broadcast:
  `docs/sprint5/lead-updates/2026-06-14-dev01-pr7-merged-task-502-go.md`
- Cluster 5/6 (TestProjectScopedRoutes) carryover from
  Sprint 4 SQG — out of scope for TASK-502 but flagged for
  the integration test when it lands.
