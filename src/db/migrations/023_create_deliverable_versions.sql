-- 023_create_deliverable_versions.sql — TASK-406 (Deliverable Storage).
--
-- Sprint 4 fresh CREATE of `deliverable_versions` for the
-- append-only version history of each deliverable. The base
-- `deliverables` table holds the *current* state (current
-- version, current title, current content); `deliverable_versions`
-- is the immutable history (every title/content that was ever
-- "current", keyed by (deliverable_id, version)).
--
-- The append-only invariant is enforced by the
-- UNIQUE(deliverable_id, version) constraint: a PUT that tries
-- to write version N when (deliverable_id, N) already exists
-- fails with a 23505 unique_violation, which the store maps
-- to ErrAlreadyExists → 409 in the handler.
--
-- This table did NOT exist in 009 (the original Sprint 1/2
-- deliverables table had version as a simple int column with
-- no history). Sprint 4 introduces the history table for the
-- first time.
--
-- Sprint 4 migration block: 016-020 (capabilities + TASK-404) +
-- 022 (deliverables additive, see file) + 023 (this file) +
-- 024 (executions, TASK-405).

-- ----------------------------------------------------------------------------
-- deliverable_versions
-- ----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS deliverable_versions (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    deliverable_id  UUID         NOT NULL
        REFERENCES deliverables(id) ON DELETE CASCADE,
    version         INT          NOT NULL,
    title           TEXT         NOT NULL,
    content         TEXT         NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by      UUID         NULL,

    -- Append-only invariant: no two rows for the same deliverable
    -- can share a version. The service layer also computes the
    -- next version from the current deliverables row, so this
    -- constraint is a defence-in-depth check (it catches any
    -- caller that tries to write a duplicate version directly).
    CONSTRAINT uq_deliverable_versions_deliverable_id_version
        UNIQUE (deliverable_id, version),

    -- Sanity: version must be positive. The service starts at 1
    -- and monotonically increments; we reject 0 and negative
    -- values at the DB level as a sanity check.
    CONSTRAINT ck_deliverable_versions_version_positive
        CHECK (version > 0)
);

COMMENT ON TABLE deliverable_versions IS
    'Append-only history of deliverable title/content changes. '
    'One row per (deliverable_id, version). The current state '
    'is mirrored in the parent deliverables row.';

COMMENT ON COLUMN deliverable_versions.created_by IS
    'The user (from JWT) who triggered the version-create. '
    'NULL for system-driven version-creates (e.g. an automated '
    're-run of an agent).';

-- ----------------------------------------------------------------------------
-- Indexes
-- ----------------------------------------------------------------------------

-- Primary list-versions path: ORDER BY version DESC for a
-- given deliverable. Postgres uses the UNIQUE index (created
-- implicitly above) backwards for this query plan; no extra
-- index needed.

-- Lookup by created_by for "what did user X change?" reports
-- (used by future activity dashboards — TASK-410).
CREATE INDEX IF NOT EXISTS ix_deliverable_versions_created_by
    ON deliverable_versions (created_by)
    WHERE created_by IS NOT NULL;
