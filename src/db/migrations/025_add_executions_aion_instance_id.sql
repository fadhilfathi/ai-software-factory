-- 025_add_executions_aion_instance_id.sql — TASK-501 (Aion Agent Runtime).
--
-- Sprint 5 follow-on to migration 024 (TASK-405 Execution Tracking).
-- Adds the optional aion_agent_instance_id UUID column to the
-- executions table. This column records the Aion agent-process
-- instance identifier populated by the aion.Runtime.Spawn path.
--
-- The column is NULLABLE because the legacy mock-goroutine path
-- (Sprint 4 default) does not produce an Aion instance ID. Executions
-- spawned by aion.Runtime in Sprint 5+ will have a non-NULL value;
-- exec-by-exec.
--
-- We deliberately keep the column UUID-typed (not TEXT) so foreign-key
-- joins to a future aion_agents table (Sprint 6+) work without a
-- type cast. We do NOT add a FK constraint at this stage because the
-- aion_agents table is not yet defined; a Sprint 6 migration will
-- add the FK when the parent table is introduced.
--
-- Migrations 021-023 are reserved for future Sprint 4 follow-ups
-- (e.g. deliverable-versioning which already landed as 022/023) and
-- 024 is TASK-405. 025 is the next free slot per the migration
-- numbering plan and is appropriate for the first Sprint 5 migration.

ALTER TABLE executions
    ADD COLUMN IF NOT EXISTS aion_agent_instance_id UUID NULL;

COMMENT ON COLUMN executions.aion_agent_instance_id IS
    'Optional Aion agent-process instance identifier populated by '
    'TASK-501 aion.Runtime.Spawn. NULL for executions created via the '
    'legacy mock-goroutine path (Sprint 4 default).';
