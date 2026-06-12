-- Migration: 014_add_reviewer_agent_id
-- Description: Add reviewer_agent_id and target_agent_id clarity to reviews table

ALTER TABLE reviews 
ADD COLUMN IF NOT EXISTS target_agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS reviewer_agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
ALTER COLUMN score TYPE DECIMAL(5,2);

-- Note: We keep agent_id for now to avoid breaking existing code, but will transition to target_agent_id.
-- For this sprint, we'll map model.TargetAgentID to target_agent_id column.
