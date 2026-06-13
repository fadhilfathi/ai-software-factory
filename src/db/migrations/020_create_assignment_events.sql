-- 020_create_assignment_events.sql — Sprint 4 Assignment History (append-only)
--  1. CREATE TABLE assignment_events ... (id, assignment_id, task_id, agent_id, assigned_by, assigned_at, action, notes)
--  2. ALTER TABLE ... CHECK constraint on action enum
--  3. CREATE INDEX idx_assignment_events_task_at (task_id, assigned_at DESC)
--  4. CREATE INDEX idx_assignment_events_agent_at (agent_id, assigned_at DESC)
--  5. CREATE INDEX idx_assignment_events_assignment_id (assignment_id)
--
-- Scope: this migration is the immutable history of task-assignment
-- actions. The TASK-404 endpoint POST /v1/tasks/:id/assign writes to
-- this table inside a transaction with the assignments table
-- (migration 019). Rows are append-only — there is no UPDATE/DELETE
-- in the service code (see src/internal/service/assignment.go).
--
-- Relationship to the assignments table (migration 019):
--   - Each assignment_events row references the assignments row that
--     caused it via assignment_id.
--   - The 020/019 split was introduced in the data-model.md finalisation
--     (TASK-404 brief correction): assignments is the current-state
--     "who is assigned right now" table, assignment_events is the
--     immutable history of every state change. The two are written in
--     a single transaction by AssignmentService.AssignTaskToAgent.
--
-- Design notes:
--   * assignment_id is NOT NULL with ON DELETE CASCADE: deleting an
--     assignment row should remove the corresponding event(s) to keep
--     referential integrity. In practice the service never deletes
--     from assignments (rows are flipped to 'superseded' / 'completed'
--     / 'cancelled' instead) so this is a safety net.
--   * agent_id is NULLABLE: the action enum supports 'unassign' for a
--     future Sprint 5 endpoint, and unassigning a task has no agent.
--   * assigned_by is NULLABLE: the JWT middleware sets user_id for
--     human/api-key callers but system-initiated assignments (e.g. an
--     autobalancer in Sprint 5+) may not have a user. For Sprint 4
--     this is always populated from c.Get("user_id") when the call is
--     made through the HTTP layer.
--   * action text CHECK enforces the 3-value enum at the DB layer
--     (defence in depth — the service also validates).

BEGIN;

CREATE TABLE IF NOT EXISTS assignment_events (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id UUID         NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
    task_id       UUID         NOT NULL REFERENCES tasks(id)     ON DELETE CASCADE,
    agent_id      UUID         NULL     REFERENCES agents(id)    ON DELETE SET NULL,
    assigned_by   UUID         NULL     REFERENCES users(id)     ON DELETE SET NULL,
    assigned_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    action        TEXT         NOT NULL CHECK (action IN ('assign', 'reassign', 'unassign')),
    notes         TEXT         NULL
);

-- Hot path: "show me the history of task X, newest first" — used by
-- GET /v1/tasks/:id/history (TASK-404) and the future TASK-408 UI.
CREATE INDEX IF NOT EXISTS idx_assignment_events_task_at
    ON assignment_events(task_id, assigned_at DESC);

-- Reverse lookup: "what is this agent working on?" — used by the
-- agent detail page (TASK-407) and the activity dashboard (TASK-410).
CREATE INDEX IF NOT EXISTS idx_assignment_events_agent_at
    ON assignment_events(agent_id, assigned_at DESC)
    WHERE agent_id IS NOT NULL;

-- Direct lookup: "all events for assignment X" — used by future
-- audit/inspection endpoints and the migration sanity check.
CREATE INDEX IF NOT EXISTS idx_assignment_events_assignment_id
    ON assignment_events(assignment_id);

COMMIT;
