-- 022_add_deliverable_versioning.sql — TASK-406 (Deliverable Storage).
--
-- Sprint 4 update to deliverables. The `deliverables` table was
-- originally created in 009_create_deliverables.sql (Sprint 1/2)
-- with: id, task_id, agent_id, title VARCHAR(500), content TEXT,
-- version INT NOT NULL DEFAULT 1, created_at TIMESTAMPTZ, plus
-- indexes (task_id), (agent_id), (task_id, agent_id).
--
-- This Sprint 4 migration is the canonical Sprint 4 schema for
-- deliverables per data-model.md §6. It does NOT recreate the
-- table (009 already did); it layers on the Sprint 4 additions
-- the brief requires:
--
--   * updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW() — set on
--     every UPDATE (the service writes a fresh value on every
--     append-only version-create; the default is a safety net)
--   * (task_id, created_at DESC) — primary list-by-task index
--   * (agent_id, created_at DESC) — primary list-by-agent index
--   * (task_id, version DESC) — "current version per task" lookup
--     (plain DESC composite; the partial sub-select shape from
--     data-model.md is overkill for Sprint 4 — a plain index is
--     good enough when the deliverables table is small relative
--     to a per-task version count)
--
-- The existing 009 indexes are kept for backward compatibility
-- with any code path that may reference them; the new DESC-ordered
-- composite indexes are the canonical Sprint 4 list paths.
--
-- Sprint 4 migration block: 016 (agent_registry) + 017 (agent_capabilities)
-- + 018 (task_required_capabilities) + 019 (assignments) + 020
-- (assignment_events) + 022 (this file, TASK-406 deliverables) + 023
-- (deliverable_versions, see file) + 024 (executions, TASK-405).

-- ----------------------------------------------------------------------------
-- Additive columns
-- ----------------------------------------------------------------------------

-- updated_at: idempotent add. The service writes a fresh value
-- on every append-only version-create, so the default only fires
-- for the first INSERT path.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'deliverables' AND column_name = 'updated_at'
    ) THEN
        ALTER TABLE deliverables
            ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
    END IF;
END $$;

COMMENT ON COLUMN deliverables.updated_at IS
    'Set to NOW() on every UPDATE. The service writes a fresh '
    'value on every append-only version-create (i.e. on every '
    'PUT /v1/deliverables/:id that produces a new version).';

-- ----------------------------------------------------------------------------
-- Sprint 4 indexes
--
-- IF NOT EXISTS guards make this migration idempotent. The
-- existing 009 indexes (idx_deliverables_task_id,
-- idx_deliverables_agent_id, idx_deliverables_task_agent) are
-- kept for backward compatibility.
-- ----------------------------------------------------------------------------

CREATE INDEX IF NOT EXISTS ix_deliverables_task_id_created_at
    ON deliverables (task_id, created_at DESC);

CREATE INDEX IF NOT EXISTS ix_deliverables_agent_id_created_at
    ON deliverables (agent_id, created_at DESC);

-- "Current version per task" index. A plain (task_id, version DESC)
-- composite is sufficient for Sprint 4 — the planner can use it
-- for "WHERE task_id = $1 ORDER BY version DESC LIMIT 1" without
-- a sub-select. The brief accepts either this shape or a
-- sub-select partial index; we go with the simpler plain composite.
CREATE INDEX IF NOT EXISTS ix_deliverables_task_id_version
    ON deliverables (task_id, version DESC);
