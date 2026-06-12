package model

import (
	"time"

	"github.com/google/uuid"
)

// Deliverable represents the current state of an artifact produced
// by an agent for a task. Sprint 4 (TASK-406) adds:
//   - UpdatedAt: set on every append-only version-create
//   - Version: monotonically incremented by the service (1, 2, 3, ...)
//   - The full history of every (title, content) the deliverable
//     ever had lives in `deliverable_versions` (one row per
//     version). The current row in `deliverables` mirrors the
//     latest version's title and content.
//
// The brief uses `version int NOT NULL DEFAULT 1`; the model
// represents it as a plain int. Sprint 4 has no
// `IsValidDeliverableVersion` validator because versions are
// server-assigned (the service computes the next version from
// the current row), not caller-supplied.
type Deliverable struct {
	ID        uuid.UUID `json:"id"`
	TaskID    uuid.UUID `json:"task_id"`
	AgentID   uuid.UUID `json:"agent_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DeliverableVersion is one immutable historical snapshot of a
// deliverable. The (deliverable_id, version) pair is unique by
// construction (UNIQUE constraint at the DB level); a PUT that
// tries to write a duplicate version fails with 23505
// unique_violation → 409 in the handler.
//
// The service writes a new row on every PUT. There is no UPDATE
// path for `deliverable_versions` rows — once written, a version
// is permanent. This is the append-only invariant.
type DeliverableVersion struct {
	ID           uuid.UUID  `json:"id"`
	DeliverableID uuid.UUID `json:"deliverable_id"`
	Version      int        `json:"version"`
	Title        string     `json:"title"`
	Content      string     `json:"content"`
	CreatedAt    time.Time  `json:"created_at"`
	CreatedBy    *uuid.UUID `json:"created_by,omitempty"`
}

// DeliverableFilter is the input to ListDeliverables. The zero
// value is a "first page, no filter" request: returns the most
// recent deliverables across all tasks/agents, default page size
// 50, max 200.
//
// Cursor-based keyset pagination: when a previous response returns
// a NextCursor, the caller passes that cursor back as Cursor to
// fetch the next page. The cursor is the ID of the last row in
// the previous page; the next page contains rows whose ID <
// cursor, ordered by (created_at DESC, id DESC) for stability.
//
// At least one of TaskID or AgentID is expected to be set; the
// service validates this and 400s otherwise. (The brief allows
// the service to require at least one filter — see
// ListDeliverables in service/deliverable.go.)
type DeliverableFilter struct {
	TaskID  uuid.UUID
	AgentID uuid.UUID
	Cursor  uuid.UUID // empty = no cursor (first page)
	Limit   int       // 0 = use default (50); clamped to [1, 200]
}

// DeliverableListResult is the output of ListDeliverables.
// NextCursor is empty when there are no more pages; the caller
// can use it as a signal to stop paginating.
type DeliverableListResult struct {
	Items      []*Deliverable
	NextCursor uuid.UUID
}

// DefaultDeliverableLimit and MaxDeliverableLimit bound the
// per-page size, matching the pattern from TASK-405
// (DefaultExecutionLimit, MaxExecutionLimit).
const (
	DefaultDeliverableLimit = 50
	MaxDeliverableLimit     = 200
)

// MaxDeliverableContentBytes caps the size of a single
// deliverable's `content` field (markdown body) at 1 MiB. This
// is the application-layer DoS hardening added by TASK-424 / F-023
// — the DB still permits up to ~1 GB, but the service rejects
// anything larger than 1 MiB with a 413 PAYLOAD_TOO_LARGE error
// before any write is attempted. The handler also wraps the
// request body in http.MaxBytesReader with a small headroom over
// this value (handler maxDeliverableRequestBytes) so the JSON
// envelope itself is bounded.
const MaxDeliverableContentBytes int64 = 1 << 20 // 1 MiB = 1048576 bytes
