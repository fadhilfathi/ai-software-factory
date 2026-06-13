-- 017_create_agent_capabilities.sql — Sprint 4 Agent Capability Join Table
--  1. CREATE TABLE agent_capabilities ... (join table agents <-> capabilities)
--  2. CREATE INDEX idx_agent_capabilities_agent_id ON agent_capabilities(agent_id)
--  3. CREATE INDEX idx_agent_capabilities_capability_id ON agent_capabilities(capability_id)
--  4. CREATE UNIQUE INDEX uq_agent_capabilities_agent_capability ON agent_capabilities(agent_id, capability_id)
--
-- Scope: this migration is the JOIN TABLE only. The capabilities catalog
-- (rows + seed) lives in 016_agent_registry.sql, consolidated by TASK-402
-- (see deviation note in docs/sprint4/infra-validation.md entry #6).
--
-- Relationship: an agent may hold zero, one, or many capabilities.
-- Each (agent, capability) row carries the proficiency (1-5, default 1)
-- and a granted_at timestamp with optional granted_by user audit field.
-- The primary key is the composite (agent_id, capability_id) which also
-- gives us a free UNIQUE index for the validation join.

BEGIN;

CREATE TABLE IF NOT EXISTS agent_capabilities (
    agent_id      UUID         NOT NULL REFERENCES agents(id)     ON DELETE CASCADE,
    capability_id UUID         NOT NULL REFERENCES capabilities(id) ON DELETE CASCADE,
    proficiency   INT          NOT NULL DEFAULT 1 CHECK (proficiency BETWEEN 1 AND 5),
    granted_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    granted_by    UUID         NULL REFERENCES users(id) ON DELETE SET NULL,
    PRIMARY KEY (agent_id, capability_id)
);

-- Forward lookups: "what capabilities does agent X have?" — the hot path
-- for the validation seam in TASK-403 and the assignment engine in TASK-404.
CREATE INDEX IF NOT EXISTS idx_agent_capabilities_agent_id
    ON agent_capabilities(agent_id);

-- Reverse lookups: "which agents hold capability Y?" — useful for
-- capability-filtered agent discovery and reporting.
CREATE INDEX IF NOT EXISTS idx_agent_capabilities_capability_id
    ON agent_capabilities(capability_id);

-- The PK already provides (agent_id, capability_id) uniqueness, but we
-- add an explicit named UNIQUE index for clarity in query plans and to
-- document the contract. IF NOT EXISTS keeps this re-runnable.
CREATE UNIQUE INDEX IF NOT EXISTS uq_agent_capabilities_agent_capability
    ON agent_capabilities(agent_id, capability_id);

COMMIT;
