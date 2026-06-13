-- 018_add_task_required_capabilities.sql — Sprint 4 Assignment Required Capabilities
--  1. ALTER TABLE tasks ADD COLUMN required_capabilities JSONB NOT NULL DEFAULT '[]'::jsonb
--  2. CREATE INDEX idx_tasks_required_capabilities_gin ON tasks USING GIN (required_capabilities)
--
-- Scope: this migration adds the persistent storage for a task's required
-- capability set. It is read by the AssignmentService seam added in TASK-403
-- (service.ValidateAgentHasCapabilities) and populated by the
-- /v1/tasks/:id/assign endpoint built in TASK-404.
--
-- Design notes:
--   * JSONB (not TEXT[]) so we can grow the shape later (e.g. {name, min_proficiency})
--     without another column-add migration. Matches the 016 agents.capabilities column type.
--   * NOT NULL DEFAULT '[]' keeps the contract "every task has a (possibly empty) requirement set"
--     so we never have to NULL-check in the service layer.
--   * GIN index supports containment queries ('required_capabilities @> $1')
--     that the capability-filtered task list endpoint will use in Sprint 5+.

BEGIN;

ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS required_capabilities JSONB NOT NULL DEFAULT '[]'::jsonb;

CREATE INDEX IF NOT EXISTS idx_tasks_required_capabilities_gin
    ON tasks USING GIN (required_capabilities);

COMMIT;
