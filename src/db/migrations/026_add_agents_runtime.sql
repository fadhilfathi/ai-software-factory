-- 026_add_agents_runtime.sql — TASK-501 (Aion Agent Runtime).
--
-- Sprint 5 follow-on to the agents table (originally created in
-- migration 005, modified by 017 for capabilities, 018 for the
-- RequiredCapabilities column, 022 for sprint 4 closeout polish).
-- Adds the optional runtime JSONB column to the agents table.
-- This column records the TASK-501 Aion runtime configuration
-- for this agent (model, provider, permission mode, etc.).
--
-- The column is NULLABLE because not every agent needs Aion
-- runtime config (the legacy mock-goroutine path doesn't use it,
-- and agents that haven't been touched by TASK-507 won't have a
-- value). We deliberately keep it as a separate column from
-- `metadata` so a user-set metadata entry like "model": "..." is
-- not picked up as the Aion model identifier.
--
-- We do NOT add a CHECK constraint on the JSONB shape; the
-- service layer is responsible for validation. Future migrations
-- (Sprint 6+) may add a constraint if the shape stabilises.

ALTER TABLE agents
    ADD COLUMN IF NOT EXISTS runtime JSONB NULL;

COMMENT ON COLUMN agents.runtime IS
    'Optional TASK-501 Aion runtime configuration. Distinct from '
    '`metadata` so user-set metadata entries do not leak into the '
    'Aion spec. NULL for agents that have not been configured for '
    'the Aion runtime (legacy mock-goroutine path).';
