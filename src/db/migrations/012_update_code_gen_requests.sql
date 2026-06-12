-- Migration: 012_update_code_gen_requests
-- Description: Add execution_id and output to code_generation_requests

ALTER TABLE code_generation_requests 
ADD COLUMN IF NOT EXISTS execution_id UUID,
ADD COLUMN IF NOT EXISTS output TEXT;

CREATE INDEX IF NOT EXISTS idx_code_gen_execution_id ON code_generation_requests(execution_id);
