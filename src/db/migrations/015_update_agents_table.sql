-- KEEP IN SYNC: `role`, `provider`, and `capabilities` types/defaults in this
-- file must match those in 010_update_agents.sql. Both migrations touch the
-- same columns; IF NOT EXISTS makes the second a no-op, but the column
-- type/default must agree so apply order is irrelevant.
-- Add missing fields to agents table for persistent registry
ALTER TABLE agents ADD COLUMN IF NOT EXISTS role VARCHAR(255);
ALTER TABLE agents ADD COLUMN IF NOT EXISTS provider VARCHAR(100);
ALTER TABLE agents ADD COLUMN IF NOT EXISTS capabilities JSONB DEFAULT '[]'::jsonb;
ALTER TABLE agents ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES projects(id) ON DELETE CASCADE;
ALTER TABLE agents ADD COLUMN IF NOT EXISTS current_task_id UUID REFERENCES tasks(id) ON DELETE SET NULL;
ALTER TABLE agents ADD COLUMN IF NOT EXISTS tasks_done INTEGER DEFAULT 0;
ALTER TABLE agents ADD COLUMN IF NOT EXISTS uptime INTEGER DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_agents_role ON agents(role);
CREATE INDEX IF NOT EXISTS idx_agents_project_id ON agents(project_id);
