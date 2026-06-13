-- 016_agent_registry.sql
-- Sprint 4 (TASK-402). Adds the agents-table extensions the new
-- /v1/agents API needs (api-spec.md §1) and creates the capabilities
-- catalog (data-model.md §2). This file consolidates the two concerns
-- per the Lead's kickoff brief: "New migration:
-- src/db/migrations/016_agent_registry.sql" and "Capability catalog
-- (6 rows, seeded by 016)".
--
-- Per data-model.md §11, all changes are additive and forward-compatible
-- with the existing 005/010/015 agents-table migrations.

-- =====================================================================
-- Part A: capabilities catalog (data-model.md §2)
-- =====================================================================

CREATE TABLE IF NOT EXISTS capabilities (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(48)  NOT NULL,
    display_name  VARCHAR(80)  NOT NULL,
    category      VARCHAR(20)  NOT NULL,
    description   VARCHAR(500),
    version       INT          NOT NULL DEFAULT 1,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT capabilities_name_unique  UNIQUE (name),
    CONSTRAINT capabilities_category_chk CHECK (category IN
        ('architecture','coding','testing','security','devops','leadership'))
);

CREATE INDEX IF NOT EXISTS idx_capabilities_category ON capabilities (category);

-- Seed data: 6 rows per data-model.md §2 (the canonical catalog).
-- ON CONFLICT (name) DO NOTHING makes the seed idempotent against
-- repeated runs; the description is the only mutable field and a
-- future migration can update it with a targeted UPDATE.
INSERT INTO capabilities (name, display_name, category, description) VALUES
    ('architecture', 'Architecture', 'architecture', 'System design, ADRs, dependency choices.'),
    ('coding',       'Coding',       'coding',       'Source code and unit tests in the source tree.'),
    ('testing',      'Testing',      'testing',      'Test execution, coverage, bug reports.'),
    ('security',     'Security',     'security',     'Threat modeling, code audit, secret scanning.'),
    ('devops',       'DevOps',       'devops',       'Build, deploy, infra-as-code, monitoring.'),
    ('leadership',   'Leader',       'leadership',   'Planning, decomposition, dispatch, conflict resolution.')
ON CONFLICT (name) DO NOTHING;

-- =====================================================================
-- Part B: agents table extensions (data-model.md §1 + api-spec.md §1)
-- =====================================================================
--
-- The agents table was originally created in 005_create_agents.sql and
-- incrementally extended in 010 and 015. The current shape does not
-- match the canonical spec; this migration brings it in line.
--
-- The 005/010/015 shape still wins for backwards compatibility — this
-- migration is purely additive. Columns are added with sensible
-- defaults so existing rows remain valid.

-- Status CHECK constraint. 005/010/015 do not enforce the lifecycle
-- state set; the canonical spec requires:
--   ('initializing','idle','busy','paused','error','retired')
-- We add it as a named constraint so it can be referenced in error
-- messages and dropped in a future migration if the state set grows.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'agents_status_chk'
    ) THEN
        ALTER TABLE agents
            ADD CONSTRAINT agents_status_chk CHECK (status IN
                ('initializing','idle','busy','paused','error','retired'));
    END IF;
END$$;

-- [+] Sprint 4: soft-delete timestamp. Soft-delete is implemented as
-- status='retired' + retired_at=NOW(); the partial index below
-- keys on retired_at IS NULL so the active-agent hot path skips
-- retired rows.
ALTER TABLE agents
    ADD COLUMN IF NOT EXISTS retired_at TIMESTAMPTZ;

-- [+] Sprint 4: denormalised metadata column. 005/010/015 store
-- this inside the `config` JSONB blob; the new API surfaces metadata
-- as a top-level field. Add as a separate column for clarity. The
-- GIN index supports `WHERE metadata @> '{...}'` queries from the
-- Agent Activity Dashboard (TASK-410).
ALTER TABLE agents
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

-- [+] Sprint 4: last_active_at. 005 had `last_heartbeat` which was
-- used as a health-check signal; the new field is a more general
-- "most recent activity" timestamp that the capability engine (TASK-403)
-- and assignment engine (TASK-404) use for the least_recently_active
-- strategy. Add as a separate column; the old last_heartbeat can stay
-- (the migration is additive).
ALTER TABLE agents
    ADD COLUMN IF NOT EXISTS last_active_at TIMESTAMPTZ;

-- [+] Sprint 4: optimistic-concurrency version. Not in data-model.md
-- §1 (a deviation flagged in the TASK-402 status report) but the
-- api-spec.md §1.4 PUT contract requires it. Default 1 for existing
-- rows; bumped on every successful Update / SetCapabilities.
ALTER TABLE agents
    ADD COLUMN IF NOT EXISTS version INT NOT NULL DEFAULT 1;

-- [+] Sprint 4: unique name per project. data-model.md §1 calls for
-- UNIQUE (project_id, name). 005 did not have project_id; 010 added
-- it but without a unique constraint. We add the constraint now,
-- guarded by IF NOT EXISTS semantics on the index (Postgres has no
-- "ADD CONSTRAINT IF NOT EXISTS" pre-9.6, so use a DO block).
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'agents_name_unique_per_project'
    ) THEN
        ALTER TABLE agents
            ADD CONSTRAINT agents_name_unique_per_project UNIQUE (project_id, name);
    END IF;
END$$;

-- Indexes per data-model.md §1.

-- Partial index for the active-agent hot path
-- (`WHERE retired_at IS NULL`). Skips retired rows; complements the
-- full idx_agents_project_id already in 005.
CREATE INDEX IF NOT EXISTS idx_agents_project_status
    ON agents (project_id, status) WHERE retired_at IS NULL;

-- GIN on the denormalised capabilities cache supports
-- `WHERE capabilities @> '["coding"]'::jsonb` from the list endpoint's
-- `?capability=` filter (api-spec.md §1.2).
CREATE INDEX IF NOT EXISTS idx_agents_capabilities_gin
    ON agents USING GIN (capabilities);

-- GIN on metadata supports the activity dashboard's metadata filter
-- (TASK-410) and any `WHERE metadata @> '{...}'` query.
CREATE INDEX IF NOT EXISTS idx_agents_metadata_gin
    ON agents USING GIN (metadata);
