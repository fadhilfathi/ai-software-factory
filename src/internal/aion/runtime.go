// Package aion provides the integration boundary between the AI Software
// Factory's task-execution service and the Aion Agent Runtime (the worker
// engine that actually executes assigned tasks).
//
// Two implementations of the Runtime interface ship in Sprint 5:
//
//   - MockRuntime (aion/mock.go) — in-process, FakeScript-driven, no
//     subprocess. Honors the JSON-over-stdio protocol envelope so the
//     dispatch / state-machine logic is identical to the subprocess path.
//     Default for `go test` and dev.
//
//   - ProcessRuntime (aion/process.go) — `os/exec` against the `aion`
//     CLI as a child process. Used when the production binary runs
//     against a real Aion installation (gated by RuntimeMode="process"
//     in the agent config, or AION_E2E=1 for E2E tests).
//
// The Runtime contract is intentionally minimal: Spawn, Wait, Cancel.
// Higher-level concerns (queueing, retry, recovery) live in package
// dispatch (TASK-502) and the state machine in service.ExecutionService
// (TASK-501/503). Sprint 5 ships the in-memory store; a postgres-backed
// dispatch queue is a Sprint 6 follow-up.
//
// Cross-tenant semantics: WorkerSpec carries the callerProjectID, but the
// runtime itself does not enforce the boundary — that's the service's
// job. The runtime treats the spawn request as already-validated.
package aion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ----------------------------------------------------------------------------
// Errors
// ----------------------------------------------------------------------------

// ErrWorkerNotFound is returned by Wait / Cancel when the supplied
// handle does not correspond to a worker the runtime knows about. In
// practice this is fatal — the service treats it as 500.
var ErrWorkerNotFound = errors.New("aion: worker not found")

// ErrWorkerTimeout is returned by Wait when the worker's deadline
// elapsed before it reached a terminal state. The worker itself is
// still alive (or already finished) — the caller is responsible for
// either reaping it or letting it run to completion.
var ErrWorkerTimeout = errors.New("aion: worker wait timed out")

// ErrRuntimeClosed is returned by Spawn / Wait / Cancel when the
// runtime has been Closed. The execution service stops spawning new
// workers once the runtime is closed (graceful shutdown).
var ErrRuntimeClosed = errors.New("aion: runtime closed")

// ErrInvalidSpec is returned by Spawn when the WorkerSpec fails
// validation (empty ExecutionID, malformed capabilities, etc.). The
// service treats this as 400.
var ErrInvalidSpec = errors.New("aion: invalid worker spec")

// ----------------------------------------------------------------------------
// Status
// ----------------------------------------------------------------------------

// WorkerStatus is the lifecycle of an Aion worker. Mirrors the
// model.ExecutionStatus values used by the cross-tenant TASK-422
// implementation, plus "cancelled" for the runtime's Cancel path.
type WorkerStatus string

const (
	WorkerPending   WorkerStatus = "pending"   // spawn requested, not yet started
	WorkerRunning   WorkerStatus = "running"   // worker process is alive
	WorkerCompleted WorkerStatus = "completed" // worker exited 0
	WorkerFailed    WorkerStatus = "failed"    // worker exited non-zero OR emitted an error frame
	WorkerCancelled WorkerStatus = "cancelled" // runtime.Cancel was called
)

// IsTerminal returns true for completed / failed / cancelled. The
// execution service treats a terminal status as the worker being
// done; further Wait / Cancel calls are no-ops.
func (s WorkerStatus) IsTerminal() bool {
	switch s {
	case WorkerCompleted, WorkerFailed, WorkerCancelled:
		return true
	default:
		return false
	}
}

// ----------------------------------------------------------------------------
// WorkerSpec / WorkerHandle / WorkerResult
// ----------------------------------------------------------------------------

// WorkerSpec is the input to Runtime.Spawn. It carries everything a
// worker process needs to start — the execution / task / agent ids,
// the project (for cross-tenant tagging), the Aion configuration
// (model / provider / permission mode), and an optional input payload
// (e.g. the task body for the worker to act on).
//
// JSON tags reflect the JSON-over-stdio protocol envelope (TASK-504):
// the subprocess path serialises the spec to stdout (or argv, for
// short specs) and the worker echoes it back in the result frame.
type WorkerSpec struct {
	// Identity
	ExecutionID uuid.UUID `json:"execution_id"`
	TaskID      uuid.UUID `json:"task_id"`
	AgentID     uuid.UUID `json:"agent_id"`
	ProjectID   uuid.UUID `json:"project_id"`

	// Aion config
	Model          string `json:"model"`
	Provider       string `json:"provider"`
	PermissionMode string `json:"permission_mode"`

	// Optional input the worker operates on (e.g. the task body).
	// nil for now; populated by Sprint 5 dispatch / TASK-504.
	Input json.RawMessage `json:"input,omitempty"`

	// Attempt is 1-based; bumped by the dispatch queue on Nack.
	Attempt int `json:"attempt"`
}

// Validate ensures the spec is well-formed before we hand it to the
// runtime. Empty UUIDs and missing Aion config are caught here.
func (s WorkerSpec) Validate() error {
	if s.ExecutionID == uuid.Nil {
		return fmt.Errorf("%w: execution_id required", ErrInvalidSpec)
	}
	if s.TaskID == uuid.Nil {
		return fmt.Errorf("%w: task_id required", ErrInvalidSpec)
	}
	if s.AgentID == uuid.Nil {
		return fmt.Errorf("%w: agent_id required", ErrInvalidSpec)
	}
	if s.ProjectID == uuid.Nil {
		return fmt.Errorf("%w: project_id required", ErrInvalidSpec)
	}
	if s.Model == "" {
		return fmt.Errorf("%w: model required", ErrInvalidSpec)
	}
	if s.Provider == "" {
		return fmt.Errorf("%w: provider required", ErrInvalidSpec)
	}
	if s.PermissionMode == "" {
		return fmt.Errorf("%w: permission_mode required", ErrInvalidSpec)
	}
	if s.Attempt < 1 {
		return fmt.Errorf("%w: attempt must be >= 1", ErrInvalidSpec)
	}
	return nil
}

// WorkerHandle is the opaque, runtime-issued identifier for a
// spawned worker. The service persists this (along with the PID
// if the runtime exposes one) in the WorkerStore. Wait and Cancel
// take a handle; they do not take a UUID.
//
// The string type lets MockRuntime use "fake-<uuid>" and
// ProcessRuntime use "<pid>-<uuid>" without leaking implementation
// details. The service treats it as opaque.
type WorkerHandle string

// WorkerResult is the output of Runtime.Wait. Always reflects a
// terminal status (completed / failed / cancelled). The service uses
// Status + ErrorMessage to drive the execution status transition
// (UpdateExecutionStatus); Result is opaque and persisted on the
// worker row for debugging.
type WorkerResult struct {
	Handle       WorkerHandle
	ExecutionID  uuid.UUID
	Status       WorkerStatus
	Result       json.RawMessage // opaque worker output (deliverable body, etc.)
	ErrorMessage string          // populated when Status == WorkerFailed
	StartedAt    time.Time
	CompletedAt  time.Time
}

// ----------------------------------------------------------------------------
// Runtime interface
// ----------------------------------------------------------------------------

// Runtime is the integration boundary. The execution service holds
// a Runtime and uses Spawn/Wait/Cancel to drive task execution
// through the Aion agent engine.
type Runtime interface {
	// Spawn requests the runtime to start a new worker for the
	// given spec. The returned handle is later used for Wait and
	// Cancel. Spawn does NOT block on the worker completing —
	// that's what Wait is for.
	//
	// Implementations may return immediately after forking (e.g.
	// ProcessRuntime), or after the worker reports "started"
	// (e.g. a future SDK-backed impl). For Sprint 5 the Mock
	// returns a fake handle synchronously and the Process impl
	// returns after the child is forked (no handshake).
	Spawn(ctx context.Context, spec WorkerSpec) (WorkerHandle, error)

	// Wait blocks until the worker reaches a terminal status
	// (completed / failed / cancelled) or the context is
	// cancelled. Always returns a WorkerResult reflecting the
	// terminal status (or an error if the runtime lost track of
	// the worker).
	//
	// For Sprint 5, the MockRuntime honours a FakeScript; the
	// ProcessRuntime reaps the child process via Wait().
	Wait(ctx context.Context, handle WorkerHandle) (WorkerResult, error)

	// Cancel requests the runtime to stop the worker
	// (best-effort; the process may not exit immediately).
	// Idempotent. Returns ErrWorkerNotFound if the handle is
	// unknown.
	Cancel(ctx context.Context, handle WorkerHandle) error

	// Close stops accepting new Spawns, waits for in-flight
	// workers to finish (or be cancelled), and releases
	// resources. After Close, Spawn / Wait / Cancel return
	// ErrRuntimeClosed.
	Close() error
}

// ----------------------------------------------------------------------------
// Message envelope (JSON-over-stdio protocol, TASK-504)
// ----------------------------------------------------------------------------

// Message is the wire format that the worker process speaks to the
// runtime over stdio. Subprocess runtime emits and receives
// Message frames; the in-process MockRuntime parses and produces
// the same frames so the handler logic is shared.
//
// The Type field discriminates:
//
//	"started"    — worker has booted and is ready for input
//	"progress"   — partial output (optional, Sprint 6+)
//	"result"     — terminal: worker succeeded
//	"error"      — terminal: worker failed
//	"cancelled"  — terminal: runtime sent SIGTERM
//
// For Sprint 5 only "started", "result", "error", "cancelled" are
// used. "progress" is reserved for TASK-506 (Execution Monitoring
// Dashboard).
type Message struct {
	Type        string          `json:"type"`
	ExecutionID uuid.UUID       `json:"execution_id"`
	Body        json.RawMessage `json:"body,omitempty"`
	Error       string          `json:"error,omitempty"`
	At          time.Time       `json:"at"`
}

// NewMessage is a small constructor for tests + the in-process
// MockRuntime. Production code paths (ProcessRuntime) use json.Marshal
// directly.
func NewMessage(msgType string, execID uuid.UUID, body json.RawMessage, errMsg string) Message {
	return Message{
		Type:        msgType,
		ExecutionID: execID,
		Body:        body,
		Error:       errMsg,
		At:          time.Now().UTC(),
	}
}
