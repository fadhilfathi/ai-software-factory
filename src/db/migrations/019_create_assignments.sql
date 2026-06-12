-- 019_create_assignments.sql — Sprint 4 Assignments (source of truth for "who is currently assigned")
--  1. CREATE TABLE assignments ... (id, task_id, agent_id, assigned_at, completed_at, status)
--  2. ALTER TABLE ... CHECK constraint on status enum
--  3. CREATE INDEX idx_assignments_task_id ON assignments(task_id)
--  4. CREATE INDEX idx_assignments_agent_id ON assignments(agent_id)
--  5. CREATE UNIQUE INDEX uq_assignments_one_active_per_task ON assignments(task_id) WHERE status = 'active'
--
-- Scope: this migration is the current-state table for task
-- assignments. The TASK-404 endpoint POST /v1/tasks/:id/assign writes
-- to this table inside a transaction (see migration 020 events table
-- and the service.AssignmentService.WithTx wrapper). The previous
-- active row (if any) is flipped to 'superseded' and the new row is
-- inserted with status='active'. The partial unique index enforces
-- "at most one active assignment per task" at the DB layer.
--
-- The append-only history of all assignment actions lives in
-- migration 020 (assignment_events). The two are linked by
-- assignment_events.assignment_id → assignments.id.
--
-- Design notes:
--   * completed_at is NULLABLE: an active row has completed_at = NULL;
--     a superseded/completed/cancelled row has completed_at set to
--     the time the row stopped being active.
--   * status enum is enforced at the DB layer (CHECK constraint)
--     AND in the service (model.IsValidAssignmentStatus). Defence
--     in depth.
--   * The partial unique index on (task_id) WHERE status = 'active'
--     is the canonical enforcement of "one active assignment per
--     task". A race between two concurrent POSTs will fail one of
--     them with a unique-constraint violation; the service maps
--     that to a 409. Documented in service/assignment.go.

BEGIN;

CREATE TABLE IF NOT EXISTS assignments (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id      UUID         NOT NULL REFERENCES tasks(id)  ON DELETE CASCADE,
    agent_id     UUID         NOT NULL REFERENCES agents(id) ON DELETE RESTRICT,
    assigned_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ  NULL,
    status       TEXT         NOT NULL DEFAULT 'active'
                              CHECK (status IN ('active', 'superseded', 'completed', 'cancelled'))
);

CREATE INDEX IF NOT EXISTS idx_assignments_task_id
    ON assignments(task_id);

CREATE INDEX IF NOT EXISTS idx_assignments_agent_id
    ON assignments(agent_id);

-- The "one active assignment per task" invariant. Enforced at the DB
-- layer so a race condition between two concurrent POSTs surfaces as
-- a unique-constraint violation that the service can map to 409.
-- IF NOT EXISTS keeps this re-runnable.
CREATE UNIQUE INDEX IF NOT EXISTS uq_assignments_one_active_per_task
    ON assignments(task_id)
    WHERE status = 'active';

COMMIT;
