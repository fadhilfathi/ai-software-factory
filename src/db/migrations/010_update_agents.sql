-- KEEP IN SYNC: `role`, `provider`, and `capabilities` types/defaults in this
-- file must match those in 015_update_agents_table.sql. Both migrations touch
-- the same columns; IF NOT EXISTS makes the second a no-op, but the column
-- type/default must agree so apply order is irrelevant.
ALTER TABLE agents
    ADD COLUMN IF NOT EXISTS role VARCHAR(255),
    ADD COLUMN IF NOT EXISTS provider VARCHAR(100),
    ADD COLUMN IF NOT EXISTS capabilities JSONB DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_agents_role ON agents(role);
