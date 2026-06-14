-- 027_tighten_agent_role_length.sql
-- A-001-followup (2026-06-14). The api-spec.md §Agents says role is
-- 1-80 chars; the agents.role column is VARCHAR(255) (set in 010 and
-- preserved in 015). Tighten the column to VARCHAR(80) to match the
-- spec. The application-side validateAgentRole in
-- internal/service/agent.go is tightened in the same PR.
--
-- Safety: this migration runs a guard SELECT first. If any existing
-- row has role > 80 chars, the migration aborts with a clear error
-- rather than silently truncating. Operators must manually decide
-- what to do (truncate, reject, etc.) before re-running. No data is
-- lost on failure; the transaction rolls back.
--
-- The column type tightening is the only safe direction (5 -> 80
-- would be additive; 255 -> 80 is contractive). VARCHAR(N) is a
-- length limit, not a storage class, so the on-disk format is
-- unchanged for in-range rows. No re-write is needed.

BEGIN;

-- 1) Pre-flight: refuse to run if any row would lose data.
DO $$
DECLARE
    too_long_count BIGINT;
    sample_name    TEXT;
    sample_role    TEXT;
BEGIN
    SELECT COUNT(*), MIN(name), MIN(role)
      INTO too_long_count, sample_name, sample_role
      FROM agents
     WHERE length(role) > 80;

    IF too_long_count > 0 THEN
        RAISE EXCEPTION
            'Cannot tighten agents.role to VARCHAR(80): % row(s) exceed 80 chars. '
            'First offender: name=%, role=%. Inspect and either update or retire '
            'these rows before re-running this migration.',
            too_long_count, sample_name, sample_role;
    END IF;
END$$;

-- 2) Tighten the column. Both 010_update_agents.sql and
--    015_update_agents_table.sql added the role column; the constraint
--    is the same on both, so a single ALTER covers it. We add a CHECK
--    in addition to the type cap so the constraint is self-documenting
--    in pg_constraint.
ALTER TABLE agents
    ALTER COLUMN role TYPE VARCHAR(80);

-- 3) Self-documenting CHECK (defensive — VARCHAR already enforces this,
--    but the constraint gives a name we can reference in error messages).
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'agents_role_length_chk'
    ) THEN
        ALTER TABLE agents
            ADD CONSTRAINT agents_role_length_chk CHECK (length(role) <= 80);
    END IF;
END$$;

COMMIT;
