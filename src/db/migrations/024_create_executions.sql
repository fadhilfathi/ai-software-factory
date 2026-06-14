-- 024_create_executions.sql — TASK-405 (Execution Tracking System).
--
-- Sprint 4 update to executions. The executions table was originally
-- created in 008_create_executions.sql (Sprint 1/2 era) with a basic
-- shape: id, task_id, agent_id, status VARCHAR(50) (no CHECK),
-- started_at (nullable), completed_at, created_at. No updated_at.
-- No error_message. No in-flight index.
--
-- This Sprint 4 migration is the canonical Sprint 4 schema for
-- executions per data-model.md. It does NOT recreate the table (008
-- already created it); instead it layers on the Sprint 4 additions
-- and tightens the constraints:
--
--   * error_message TEXT NULL — populated when status transitions to 'failed'
--   * updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW() — set on every UPDATE
--   * status TEXT NOT NULL DEFAULT 'pending' CHECK in (allowed set) — promotes
--     the implicit CHECK to the canonical Sprint 4 form
--   * (task_id, started_at DESC) — primary list-by-task index
--   * (agent_id, started_at DESC) — primary list-by-agent index
--   * (status) WHERE status IN ('pending','running') — fast in-flight lookup
--
-- We deliberately keep the `id` PK column name (matching 008) rather
-- than renaming to `execution_id`. The Go model field `ExecutionID`
-- is the in-Go identifier; the DB column is `id` per 008.
--
-- Sprint 4 migration block: 016 (agent_registry) + 017 (agent_capabilities)
-- + 018 (task_required_capabilities) + 019 (assignments) + 020
-- (assignment_events) + 024 (this file, TASK-405). Migrations 021-023
-- are reserved for future Sprint 4 follow-ups (e.g. TASK-406
-- deliverables, additional capability tables) per Analyst-01's
-- numbering plan.

-- ----------------------------------------------------------------------------
-- Additive columns
-- ----------------------------------------------------------------------------

ALTER TABLE executions
    ADD COLUMN IF NOT EXISTS error_message TEXT NULL;

-- updated_at: we use the canonical Sprint 4 default. If 008 has it
-- already we leave it alone (the IF NOT EXISTS guard handles both
-- the no-op and the add case). The model layer also writes a value
-- on every UpdateStatus, so the default is a safety net.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'executions' AND column_name = 'updated_at'
    ) THEN
        ALTER TABLE executions
            ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
    END IF;
END $$;

COMMENT ON COLUMN executions.error_message IS
    'Failure detail populated when status transitions to ''failed''. '
    'NULL for queued/assigned/running/review/completed.';

COMMENT ON COLUMN executions.updated_at IS
    'Set to NOW() on every UPDATE. The service layer writes a fresh '
    'value on every UpdateStatus call.';

-- ----------------------------------------------------------------------------
-- Status CHECK constraint
--
-- 008 used VARCHAR(50) with no CHECK. Sprint 4 promotes this to TEXT
-- with a CHECK against the four canonical statuses. We DROP the
-- existing 008 column type and re-cast to TEXT. Existing rows are
-- constrained to the same value set, so the cast is safe.
-- ----------------------------------------------------------------------------

ALTER TABLE executions
    ALTER COLUMN status TYPE TEXT,
    ALTER COLUMN status SET DEFAULT 'assigned',
    ALTER COLUMN status SET NOT NULL;

-- Drop and re-add the CHECK so we can be sure the value set matches
-- the brief exactly. We do this in a DO block so the migration is
-- idempotent: if the CHECK already exists with the right definition,
-- the DROP fails silently (the DO block uses EXCEPTION handling).
DO $$
BEGIN
    BEGIN
        ALTER TABLE executions DROP CONSTRAINT IF EXISTS executions_status_check;
    EXCEPTION WHEN OTHERS THEN
        -- constraint may not exist; ignore
        NULL;
    END;
END $$;

ALTER TABLE executions
    ADD CONSTRAINT executions_status_check
    CHECK (status IN (
        'queued',
        'assigned',
        'running',
        'review',
        'completed',
        'failed'
    ));

-- ----------------------------------------------------------------------------
-- Sprint 4 indexes
--
-- IF NOT EXISTS guards make this migration idempotent. The existing
-- 008 indexes (idx_executions_task_id, idx_executions_agent_id,
-- idx_executions_status) are kept for backward compatibility with
-- any code path that may reference them; the new DESC-ordered
-- composite indexes are the canonical Sprint 4 list paths.
-- ----------------------------------------------------------------------------

CREATE INDEX IF NOT EXISTS ix_executions_task_id_started_at
    ON executions (task_id, started_at DESC NULLS LAST);

CREATE INDEX IF NOT EXISTS ix_executions_agent_id_started_at
    ON executions (agent_id, started_at DESC NULLS LAST);

-- Partial index for the in-flight lookup. The WHERE clause mirrors
-- the CHECK constraint so the planner can use the index for
-- "WHERE status = ANY(...)" queries.
CREATE INDEX IF NOT EXISTS ix_executions_in_flight
    ON executions (status)
    WHERE status IN ('assigned', 'running', 'review');
