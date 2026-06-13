// Worker is the in-memory (and future postgres-backed) record of a
// spawned Aion runtime worker. Sprint 5 stores these in the
// in-memory store; the postgres migration is a Sprint 6 follow-up
// (additive — no schema shape change).
//
// One worker row per spawn. The runtime handle is opaque (its
// format is runtime-impl-defined — "mock-N-<uuid>" for the
// in-process MockRuntime, "proc-<pid>-<uuid>" for the subprocess
// ProcessRuntime). PID is also stored for the Process case so
// operators can `kill` the process manually if the runtime loses
// track of it.
package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// WorkerStatus mirrors aion.WorkerStatus at the persistence layer.
// We re-declare it (rather than import aion from model) to keep
// the model package free of runtime-specific imports — the
// service layer is the boundary that maps between the two.
type WorkerStatus string

const (
	WorkerPending   WorkerStatus = "pending"
	WorkerRunning   WorkerStatus = "running"
	WorkerCompleted WorkerStatus = "completed"
	WorkerFailed    WorkerStatus = "failed"
	WorkerCancelled WorkerStatus = "cancelled"
)

// IsTerminal mirrors aion.WorkerStatus.IsTerminal — used by the
// recovery / monitoring layers (TASK-508, TASK-506) to filter for
// rows that no longer need polling.
func (s WorkerStatus) IsTerminal() bool {
	switch s {
	case WorkerCompleted, WorkerFailed, WorkerCancelled:
		return true
	default:
		return false
	}
}

// IsValidWorkerStatus returns true for the five documented status
// values; false for empty, lower-case-variants, or unknown values.
// Used by the store layer to reject bogus writes before they hit
// the in-memory map (and, in Sprint 6+, the postgres CHECK constraint).
func IsValidWorkerStatus(s WorkerStatus) bool {
	switch s {
	case WorkerPending, WorkerRunning, WorkerCompleted, WorkerFailed, WorkerCancelled:
		return true
	default:
		return false
	}
}

// Worker is the persistent record of a single spawned Aion worker.
type Worker struct {
	ID          uuid.UUID `json:"id"`
	ExecutionID uuid.UUID `json:"execution_id"`
	AgentID     uuid.UUID `json:"agent_id"`
	ProjectID   uuid.UUID `json:"project_id"`

	// Runtime-specific handle and PID. PID is nil for in-process
	// runtimes (no OS process involved); set for the subprocess
	// ProcessRuntime so operators can `kill` the process manually
	// if the runtime loses track of it. Handle is an opaque string
	// from the runtime impl (e.g. "mock-N-<uuid>" for MockRuntime,
	// "proc-<pid>-<uuid>" for ProcessRuntime).
	Handle string `json:"handle"`
	PID    *int   `json:"pid,omitempty"`

	Status WorkerStatus `json:"status"`

	// Attempt is 1-based; bumped by the dispatch queue on Nack
	// (Sprint 6+) or by the recovery layer on retry (TASK-508).
	Attempt int `json:"attempt"`

	StartedAt    *time.Time      `json:"started_at,omitempty"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
	Result       json.RawMessage `json:"result,omitempty"`
	ErrorMessage string          `json:"error_message,omitempty"`

	// AionInstanceID is the same value written to
	// model.Execution.AionAgentInstanceID at CreateExecution time
	// (TASK-501). We duplicate it on the worker row for two reasons:
	//  (1) O(1) worker lookup by instance ID for the upcoming
	//      TASK-506 monitoring dashboard's "find workers spawned by
	//      this Aion instance" view.
	//  (2) When the worker is the only surviving record (e.g. the
	//      execution row was rolled back), the instance ID is
	//      still recoverable.
	AionInstanceID *uuid.UUID `json:"aion_instance_id,omitempty"`
}
