# Data Model — Sprint 4 (Canonical)

> **Status:** Canonical Postgres schema, owned by Analyst-01 (TASK-401).
> **Audience:** Developer-01 (TASK-402/403/404/405/406) for store/handler work, Tester-01 (TASK-411), Security-01 (TASK-412), DevOps-01 (TASK-413/414).
> **Database:** PostgreSQL 16 (primary). Migrations live under `src/db/migrations/`.

This doc is the **source of truth** for table shapes in Sprint 4. Existing tables from Sprints 1–3 are reproduced here for context; **Sprint 4 only adds** the `capabilities`, `agent_capabilities`, `assignments`, `assignment_events`, and `deliverable_versions` tables, plus a small additive migration to `agents` and `deliverables` (and a new `tasks.required_capabilities` column on the pre-existing `tasks` table).

Notation:

- `(*)` = new in Sprint 4.
- `[+]` = column added in Sprint 4 to a pre-existing table.
- All ids are `UUID` with `DEFAULT gen_random_uuid()` (requires `pgcrypto` or Postgres 13+).
- All `*_at` columns are `TIMESTAMPTZ` server-set in UTC.

---

## 1. `agents`  *(existing, extended)*

Pre-existing from Sprint 1. Sprint 4 adds the `capabilities` denormalized cache column (already in `010_update_agents.sql`) and ensures the `status` enum includes the full state set.

```sql
CREATE TABLE agents (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID         NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name            VARCHAR(80)  NOT NULL,
    role            VARCHAR(255) NOT NULL,                       -- aligned with 010/015 (TASK-416)
    status          VARCHAR(20)  NOT NULL DEFAULT 'initializing',
    capabilities    JSONB        NOT NULL DEFAULT '[]'::jsonb,   -- denormalized cache; see agent_capabilities
    last_active_at  TIMESTAMPTZ,
    metadata        JSONB        NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    version         INT          NOT NULL DEFAULT 1,            -- Optimistic concurrency: bumped on every mutation; PUT /v1/agents/:id requires the caller's version to match, otherwise 409 VERSION_CONFLICT (api-spec §1.4).
    retired_at      TIMESTAMPTZ,                                -- [+] Sprint 4: soft-delete timestamp
    CONSTRAINT agents_status_chk CHECK (status IN
        ('initializing','idle','busy','paused','error','retired')),
    CONSTRAINT agents_name_unique_per_project UNIQUE (project_id, name)
);
```

**Indexes**

```sql
CREATE INDEX idx_agents_project_status    ON agents (project_id, status) WHERE retired_at IS NULL;
CREATE INDEX idx_agents_capabilities_gin  ON agents USING GIN (capabilities);
CREATE INDEX idx_agents_metadata_gin      ON agents USING GIN (metadata);
```

**Notes**

- `status` values are validated at the application layer against the catalog in `agent-orchestration-design.md` §2; the DB CHECK is a backstop.
- `capabilities` is **redundant** with `agent_capabilities` (next table). Writes go to both inside a single transaction. The JSONB column is for cheap `WHERE capabilities @> '["coding"]'::jsonb` queries (GIN-indexed); the join table is for FKs and per-capability metadata.
- `retired_at` is the new "soft delete" flag. `DELETE /v1/agents/:id` sets `status='retired'` and `retired_at=NOW()` rather than removing the row.
- `version` is the optimistic-concurrency token. Every successful `UPDATE agents SET ...` bumps `version = version + 1` in the same transaction. `PUT /v1/agents/:id` requires the caller to send the current `version` they read; a mismatch returns `409 VERSION_CONFLICT` (see api-spec §1.4). Default is 1 on insert. The token is **not** exposed for human comparison — it's a pure write-side guard, not a logical revision counter.

---

## 2. `capabilities`  *(new)* (*)

The global, project-independent catalog of named capabilities.

```sql
CREATE TABLE capabilities (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(48)  NOT NULL,                       -- e.g. 'coding', 'leadership'
    display_name  VARCHAR(80)  NOT NULL,                       -- e.g. 'Coding'
    category      VARCHAR(20)  NOT NULL,                       -- see constraint
    description   VARCHAR(500),
    version       INT          NOT NULL DEFAULT 1,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT capabilities_name_unique  UNIQUE (name),
    CONSTRAINT capabilities_category_chk CHECK (category IN
        ('architecture','coding','testing','security','devops','leadership'))
);
```

**Seed data** (Sprint 4 migration `016_create_capabilities.sql`):

```sql
INSERT INTO capabilities (name, display_name, category, description) VALUES
    ('architecture', 'Architecture', 'architecture', 'System design, ADRs, dependency choices.'),
    ('coding',       'Coding',       'coding',       'Source code and unit tests in the source tree.'),
    ('testing',      'Testing',      'testing',      'Test execution, coverage, bug reports.'),
    ('security',     'Security',     'security',     'Threat modeling, code audit, secret scanning.'),
    ('devops',       'DevOps',       'devops',       'Build, deploy, infra-as-code, monitoring.'),
    ('leadership',   'Leader',       'leadership',   'Planning, decomposition, dispatch, conflict resolution.');
```

**Assignability**

Of the six seeded capabilities, **five are task-assignable** via the standard assignment engine:

- `architecture`, `coding`, `testing`, `security`, `devops` — these can be requested by any task via `task.required_capabilities` and matched against agent capability sets.

The sixth, **`leadership`**, is **role-only**: it is declared only by the Leader agent and is not surfaced in the standard `required_capabilities` list. A "leadership task" (planning, decomposition, conflict resolution) is dispatched directly by the Leader, not routed through `POST /v1/tasks/:id/assign`'s auto-mode. This keeps the assignment engine's contract simple (any task with N capabilities goes to any agent with ≥ 1 matching capability) while still letting the Leader do its planning work. See `agent-orchestration-design.md` §4 for the routing rules.

**Indexes**

```sql
CREATE INDEX idx_capabilities_category ON capabilities (category);
```

---

## 3. `agent_capabilities`  *(new)* (*)

Many-to-many join. Lets us FK an assignment's capability resolution back to a real capability row, and store per-capability metadata (e.g. proficiency) without a future schema change.

```sql
CREATE TABLE agent_capabilities (
    agent_id       UUID         NOT NULL REFERENCES agents(id)       ON DELETE CASCADE,
    capability_id  UUID         NOT NULL REFERENCES capabilities(id) ON DELETE RESTRICT,
    proficiency    INT,                                             -- 1..5, nullable; forward-compat (TASK-403)
    granted_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    granted_by     UUID         REFERENCES agents(id),              -- who granted (system / operator)
    PRIMARY KEY (agent_id, capability_id),
    CONSTRAINT agent_capabilities_proficiency_chk
        CHECK (proficiency IS NULL OR proficiency BETWEEN 1 AND 5)
);
```

**Indexes**

```sql
CREATE INDEX idx_agent_capabilities_capability ON agent_capabilities (capability_id);
```

**Invariant**

- Whenever a row is added/removed in this table, the corresponding entry in `agents.capabilities[]` is updated in the same transaction. The store layer is responsible for keeping the two in sync (`agent.Store.SetCapabilities(agentID, names []string)` is the only allowed write path).

---

## 4. `assignments`  *(new)* (*)

Records the act of the assignment engine binding a Task to an Agent. There is **at most one active row per task**, enforced by a partial unique index on `status = 'active'`. When a task is re-assigned, the prior active row is moved to `status = 'superseded'` (with `completed_at` set) and a new active row is inserted. The `assignment_events` table (§5) records the audit trail of every transition.

```sql
CREATE TABLE assignments (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id       UUID         NOT NULL REFERENCES tasks(id)  ON DELETE CASCADE,
    agent_id      UUID         NOT NULL REFERENCES agents(id) ON DELETE RESTRICT,
    assigned_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),         -- Go model `Assignment.AssignedAt` (time.Time)
    completed_at  TIMESTAMPTZ,                                 -- nullable; set when the row leaves 'active'; Go model `Assignment.CompletedAt` (*time.Time)
    status        TEXT         NOT NULL DEFAULT 'active',     -- 'active' | 'superseded' | 'completed' | 'cancelled'
    CONSTRAINT assignments_status_chk CHECK (status IN
        ('active','superseded','completed','cancelled'))
);
```

**Indexes**

```sql
CREATE INDEX idx_assignments_task_id
    ON assignments (task_id);
CREATE INDEX idx_assignments_agent_id
    ON assignments (agent_id);
CREATE UNIQUE INDEX uq_assignments_one_active_per_task
    ON assignments (task_id) WHERE status = 'active';
```

**Why a partial unique index:** a Task can be re-assigned over its lifetime; we keep the full history of past bindings, but only one row per task can be `active` at a time. The partial unique index `uq_assignments_one_active_per_task` (on `(task_id) WHERE status = 'active'`) enforces this at the DB level. The Go service layer (`AssignmentService`) also enforces it as a defense-in-depth check.

**Status transitions**

- `active` → `superseded`: a new active row is inserted for the same `task_id` (re-assignment); the previous row's `completed_at` is set to the swap timestamp.
- `active` → `completed`: the task finished (driven by the execution system, TASK-405); `completed_at` is set.
- `active` → `cancelled`: the assignment was explicitly cancelled (e.g. operator override) before the task finished; `completed_at` is set.
- Terminal states (`superseded`, `completed`, `cancelled`) never transition again.

**Note on audit fields:** *who* performed an assignment is **not** stored on this row — it lives in `assignment_events.assigned_by` (§5) as part of the append-only history. The "who is responsible for the current active assignment?" question is answered by querying the most recent event for the active row. The same is true for the *action verb* (`assign` / `reassign` / `unassign`), which lives in `assignment_events.action`. See §4.1 for Sprint 5 additive columns that augment the audit metadata directly on this table.

## 4.1. `assignments` — Sprint 5 additive columns (deferred)

The Sprint 4 design intentionally keeps `assignments` minimal: identity columns, two FKs, lifecycle timestamps (`assigned_at`, `completed_at`), and a 4-value status flag. Four columns are **deferred to Sprint 5** because they support features (project-scoped queries, assignment-strategy observability, candidate-snapshot debugging) that are out of scope for Sprint 4 but will land in a future additive `ALTER TABLE`. The four deferred columns are:

| Column | Type | Nullable | Purpose |
|---|---|---|---|
| `project_id` | `UUID` REFERENCES `projects(id)` | NOT NULL (backfill required) | Project-scoped query shortcut. Backfill on deploy: `UPDATE assignments a SET project_id = t.project_id FROM tasks t WHERE a.task_id = t.id`. A `(project_id, status)` index will be added in the same migration. |
| `strategy` | `TEXT` | NULL | Which selection strategy produced this assignment: `manual`, `rule_based`, `least_recently_active`, `fallback`. Captures *how* the assignment was made (status captures *what state* it is in). |
| `selected_reason` | `TEXT` | NULL | Free-text justification — e.g. `rule_pass:capabilities+availability`, `manual_override:<operator_uuid>`, `fallback:no_rule_match`. |
| `candidates` | `JSONB` | NULL | Snapshot of the candidate-agent list at selection time (a JSON array of `{agent_id, score, reason}` objects). Useful for "why was X chosen over Y?" debug queries. |

**Plan:** these will land in a future Sprint 5 `ALTER TABLE` migration `025_extend_assignments.sql` (or similar). They support the future TASK-410 (Agent Activity Dashboard) and the project-scoped reporting for Sprint 5+. **Out of scope for Sprint 4.**

---

## 5. `assignment_events`  *(new)* (*)

Append-only event log for the assignment workflow. Used by the Agent Activity Dashboard (TASK-410) and by debugging. The assignment service writes one row per `assign`, `reassign`, `unassign` action.

**Note on naming vs §4:** the `assignments` table (§4) has a 4-value `status` column that describes the **current state** of a row (`active` / `superseded` / `completed` / `cancelled`). The `assignment_events` table has a 3-value `action` column that describes the **verb that was performed** at a point in time (`assign` / `reassign` / `unassign`). They are complementary, not duplicative: `status` answers "what state is this row in now?" while `action` answers "what just happened?". The same `assignments` row will have a single `status` value but many `action` events over its lifetime.

```sql
CREATE TABLE assignment_events (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id  UUID         NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
    task_id        UUID         NOT NULL REFERENCES tasks(id)        ON DELETE CASCADE,  -- denormalized for direct task-history queries
    agent_id       UUID         REFERENCES agents(id) ON DELETE SET NULL,                  -- the assignee at the time of the event (nullable; denormalized)
    assigned_by    UUID         REFERENCES users(id)  ON DELETE SET NULL,                  -- who triggered the action (operator UUID; nullable; FK to `users`, not `agents`)
    assigned_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),                                   -- when the action happened
    action         TEXT         NOT NULL,                                                 -- 'assign' | 'reassign' | 'unassign'
    notes          TEXT,                                                                  -- free-text justification (rule results, operator note, etc.); nullable
    CONSTRAINT assignment_events_action_chk CHECK (action IN
        ('assign','reassign','unassign'))
);
```

**Indexes**

```sql
CREATE INDEX idx_assignment_events_task_at
    ON assignment_events (task_id, assigned_at DESC);
CREATE INDEX idx_assignment_events_agent_at
    ON assignment_events (agent_id, assigned_at DESC)
    WHERE agent_id IS NOT NULL;
CREATE INDEX idx_assignment_events_assignment_id
    ON assignment_events (assignment_id);
```

---

## 6. `agent_state_events`  *(new)* (*)

Companion to §1/§2 of the design. Every transition (e.g. `idle → busy`) is logged here. Powers the Agent Activity Dashboard timeline.

```sql
CREATE TABLE agent_state_events (
    id           BIGSERIAL    PRIMARY KEY,
    agent_id     UUID         NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    from_status  VARCHAR(20),
    to_status    VARCHAR(20)  NOT NULL,
    reason       TEXT,                                            -- 'task_assigned', 'execution_failed', 'operator_pause', ...
    actor_id     UUID         REFERENCES agents(id),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agent_state_events_agent ON agent_state_events (agent_id, created_at DESC);
```

---

## 7. `tasks`  *(existing, extended)* (*)

The pre-existing `tasks` table. Sprint 4 adds `required_capabilities` (TASK-403) so a task can declare *which* capabilities it needs, not just the single `required_capability` it had before.

```sql
ALTER TABLE tasks
    ADD COLUMN required_capabilities JSONB NOT NULL DEFAULT '[]'::jsonb;
```

**Indexes**

```sql
CREATE INDEX idx_tasks_required_capabilities_gin ON tasks USING GIN (required_capabilities);
```

> The pre-existing `required_capability VARCHAR` column is kept for backward compatibility and for single-capability tasks. New code should write to **both**: `required_capabilities` (JSONB array, used by the assignment engine) and the first element of that array into `required_capability` (used by older queries and the existing API responses in `api-spec.md` §3.1).

**Notes**

- `required_capabilities` is the **authoritative** capability contract for assignment. The assignment engine evaluates whether the candidate agent's declared capabilities intersect the task's required set. Any-of matching is the default (`agent_caps ∩ task_caps ≠ ∅`); all-of is a future per-task flag.
- The `[]` default makes a "no requirements" task trivially assignable to any idle agent. Real tasks should always declare at least one capability.
- The GIN index supports containment queries like `WHERE required_capabilities @> '["coding"]'::jsonb` (find tasks that need coding) and inverse lookups (find agents whose capabilities are a subset of the required set).

---

## 8. `executions`  *(existing, Sprint 4 additive delta in 024)*

Originally created in `008_create_executions.sql` (Sprint 1/2) as a basic shape: `id`, `task_id`, `agent_id`, `status VARCHAR(50)` (no CHECK), `started_at`, `completed_at`, `created_at`. Sprint 4 migration `024_create_executions.sql` is an **additive ALTER** that layers on `error_message`, `updated_at`, a status CHECK constraint (promoted from `VARCHAR(50)` to `TEXT`), and 3 new indexes — see §12 row 024 for the full additive delta. The schema below reflects the **post-024** state (008 base + 024 additions).

**Note on lifecycle naming vs §4:** the `assignments` table (§4) uses `assigned_at` / `completed_at` for the assignment lifecycle. The `executions` table uses `started_at` / `completed_at` for the execution lifecycle. Both `completed_at`s share the semantic "lifecycle end time", but the start times differ (`assigned_at` vs `started_at`) because assignment and execution are different lifecycles — an assignment is "this agent is bound to this task", an execution is "this agent is actually running the task right now". The naming is intentionally distinct; do not try to unify them.

**Note on Go field naming:** the Go model field is `ExecutionID` (not plain `ID`) — preserved from the original Sprint 1/2 model to avoid a wider refactor. The DB column is `id` (UUID PK). Callers should treat `ExecutionID` as the primary key; the field name discrepancy is a known wart (see `src/internal/model/execution.go` lines 60-62).

```sql
CREATE TABLE executions (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id       UUID         NOT NULL REFERENCES tasks(id)  ON DELETE CASCADE,
    agent_id      UUID         NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    status        TEXT         NOT NULL DEFAULT 'pending',     -- 'pending' | 'running' | 'completed' | 'failed' (4 values; promoted from VARCHAR(50) to TEXT in 024)
    started_at    TIMESTAMPTZ,                                 -- nullable; Go model `Execution.StartedAt` (time.Time)
    completed_at  TIMESTAMPTZ,                                 -- nullable; Go model `Execution.CompletedAt` (*time.Time)
    error_message TEXT,                                        -- Sprint 4 (024): populated when status → 'failed'; Go model `Execution.ErrorMessage` (*string)
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),         -- Sprint 4 (024): set on every UPDATE; Go model `Execution.UpdatedAt` (time.Time)
    CONSTRAINT executions_status_chk CHECK (status IN
        ('pending','running','completed','failed'))
);
```

**Indexes**

The 008 base provides 4 indexes; 024 layers 3 more. Both sets are kept (024 does not drop any 008 indexes).

```sql
-- 008 base (kept after 024)
CREATE INDEX idx_executions_task_id     ON executions (task_id);
CREATE INDEX idx_executions_agent_id    ON executions (agent_id);
CREATE INDEX idx_executions_status      ON executions (status);
CREATE INDEX idx_executions_task_agent  ON executions (task_id, agent_id);

-- 024 additions
CREATE INDEX ix_executions_task_id_started_at
    ON executions (task_id, started_at DESC NULLS LAST);
CREATE INDEX ix_executions_agent_id_started_at
    ON executions (agent_id, started_at DESC NULLS LAST);
CREATE INDEX ix_executions_in_flight
    ON executions (status) WHERE status IN ('pending','running');
```

**Status transitions** (per `ExecutionStatus` constants in `src/internal/model/execution.go`):

- `pending` → `running`, `completed`, `failed`
- `running` → `completed`, `failed`
- `completed` → terminal
- `failed` → terminal

---

## 9. `deliverables`  *(existing, Sprint 4 additive delta in 022)*

Originally created in `009_create_deliverables.sql` (Sprint 1/2) as a basic shape: `id`, `task_id`, `agent_id`, `title VARCHAR(500) NOT NULL`, `content TEXT NOT NULL DEFAULT ''`, `version INT NOT NULL DEFAULT 1`, `created_at`. Sprint 4 migration `022_add_deliverable_versioning.sql` is an **additive ALTER** that layers on `updated_at` and 3 new list/index paths — see §12 row 022 for the full additive delta. The schema below reflects the **post-022** state (009 base + 022 additions).

```sql
CREATE TABLE deliverables (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id    UUID         NOT NULL REFERENCES tasks(id)  ON DELETE CASCADE,
    agent_id   UUID         NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    title      VARCHAR(500) NOT NULL,
    content    TEXT         NOT NULL DEFAULT '',
    version    INT          NOT NULL DEFAULT 1,             -- monotonically incremented on every append-only version-create; mirrors MAX(version) in `deliverable_versions` (§10)
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()          -- Sprint 4 (022): set on every UPDATE; Go model `Deliverable.UpdatedAt` (time.Time)
);
```

**Indexes**

The 009 base provides 3 indexes; 022 layers 3 more. Both sets are kept (022 does not drop any 009 indexes).

```sql
-- 009 base (kept after 022)
CREATE INDEX idx_deliverables_task_id     ON deliverables (task_id);
CREATE INDEX idx_deliverables_agent_id    ON deliverables (agent_id);
CREATE INDEX idx_deliverables_task_agent  ON deliverables (task_id, agent_id);

-- 022 additions
CREATE INDEX ix_deliverables_task_id_created_at
    ON deliverables (task_id, created_at DESC);
CREATE INDEX ix_deliverables_agent_id_created_at
    ON deliverables (agent_id, created_at DESC);
CREATE INDEX ix_deliverables_task_id_version
    ON deliverables (task_id, version DESC);               -- "current version per task" lookup (plain composite, per the 022 SQL comment)
```

**Note on `version` vs `deliverable_versions`:** the `version` column on this row is a denormalized pointer that mirrors the max version in `deliverable_versions` (§10) for the same `id`. The append-only invariant (no two rows for the same deliverable sharing a version) is enforced by the `UNIQUE(deliverable_id, version)` constraint on `deliverable_versions` (§10); the `deliverables.version` value is updated transactionally with the new `deliverable_versions` insert. See §9.1 for Sprint 5 additive columns that augment the deliverable metadata.

## 9.1. `deliverables` — Sprint 5 additive columns (deferred)

The Sprint 4 design intentionally keeps `deliverables` minimal: identity, two FKs, three payload fields, two lifecycle timestamps, and a denormalized version pointer. Four columns are **deferred to Sprint 5** because they support features (project-scoped queries, deliverable-type classification, free-form metadata) that are out of scope for Sprint 4 but will land in a future additive `ALTER TABLE`. The four deferred columns are:

| Column | Type | Nullable | Purpose |
|---|---|---|---|
| `project_id` | `UUID` REFERENCES `projects(id)` | NOT NULL (backfill required) | Project-scoped query shortcut. Backfill on deploy: `UPDATE deliverables d SET project_id = t.project_id FROM tasks t WHERE d.task_id = t.id`. A `(project_id, created_at)` index will be added in the same migration. |
| `kind` | `VARCHAR(40)` | NOT NULL | Deliverable type — `code` \| `doc` \| `design` \| `test_report` \| `config` \| `other`. CHECK constraint at the DB level. |
| `description` | `TEXT` | NULL | Optional human-readable summary; shown in list views and dashboards. |
| `metadata` | `JSONB` | NOT NULL DEFAULT `'{}'::jsonb` | Arbitrary structured metadata (e.g. S3 keys for binary content, build artifacts, test coverage). GIN index on `metadata` in the same migration. |

**Plan:** these will land in a future Sprint 5 `ALTER TABLE` migration `025b_extend_deliverables.sql` (or similar). They support the future TASK-410 (Agent Activity Dashboard) and the project-scoped reporting for Sprint 5+. **Out of scope for Sprint 4.**

**Considered but dropped from this list:** `latest_version` (a denormalized `INT` pointer to the max version in `deliverable_versions`) was considered but **excluded** from this list — the existing `version` column on `deliverables` is already the current version number (incremented transactionally with each new `deliverable_versions` insert), so a separate `latest_version` cache would be redundant and not worth the duplication cost. Recorded here so future rounds don't re-flag it.

---

## 10. `deliverable_versions`  *(new — Sprint 4, TASK-406)*

Immutable, append-only history of every version of a deliverable. Created fresh in `023_create_deliverable_versions.sql` — the table did NOT exist in 009 (the original Sprint 1/2 deliverables table had `version` as a simple int column with no history). Powering `GET /v1/deliverables/:id/versions`. New versions are created via `PUT /v1/deliverables/:id` (the `:id` refers to the **deliverable**, not a version). Each PUT writes a new row to this table and updates the parent `deliverables` row's `content` + `version` + `updated_at` in a single transaction.

```sql
CREATE TABLE deliverable_versions (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    deliverable_id  UUID         NOT NULL REFERENCES deliverables(id) ON DELETE CASCADE,
    version         INT          NOT NULL,                       -- monotonically increasing per deliverable; service-computed
    title           TEXT         NOT NULL,
    content         TEXT         NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by      UUID,                                         -- the user (from JWT) who triggered the version-create; NULL for system-driven; NO DB-level FK (validated at the app layer — see Go model `DeliverableVersion.CreatedBy *uuid.UUID`)
    CONSTRAINT uq_deliverable_versions_deliverable_id_version UNIQUE (deliverable_id, version),
    CONSTRAINT ck_deliverable_versions_version_positive       CHECK (version > 0)
);
```

**Indexes**

The `UNIQUE(deliverable_id, version)` constraint above creates an implicit btree index that backs the primary list-versions path (`ORDER BY version DESC` for a given `deliverable_id` — Postgres uses the UNIQUE index backwards). One additional partial index supports the "what did user X change?" report path (TASK-410):

```sql
CREATE INDEX ix_deliverable_versions_created_by
    ON deliverable_versions (created_by) WHERE created_by IS NOT NULL;
```

**Append-only invariant:** the UNIQUE constraint on `(deliverable_id, version)` is the DB-level enforcement. A PUT that tries to write a duplicate `(deliverable_id, version)` fails with 23505 unique_violation, which the store maps to ErrAlreadyExists → 409 in the handler. The service layer also computes the next version from the current `deliverables.version`, so this constraint is a defense-in-depth check (it catches any caller that tries to write a duplicate version directly).

**Versioning rules**

- On `PUT /v1/deliverables/:id`, the service:
  1. `INSERT INTO deliverable_versions (deliverable_id, version, title, content, created_by) VALUES (..., parent.version+1, ...)`
  2. `UPDATE deliverables SET content=..., version=version+1, updated_at=NOW() WHERE id=:id`
- All inside one transaction.
- `GET /v1/deliverables/:id/versions` returns rows from this table, newest first (the `UNIQUE(deliverable_id, version)` btree is scanned backwards).

---

## 11. Cross-entity FK map

```
projects ──┬──< agents ──┬──< agent_capabilities >── capabilities
           │             ├──< agent_state_events
           │             ├──< assignments ──< assignment_events
           │             ├──< executions
           │             └──< deliverables ──< deliverable_versions
           │
           └──< tasks ──┬──< assignments       (via task_id)
                        ├──< assignment_events  (via task_id, denormalized)
                        ├──< executions         (via task_id)
                        └──< deliverables       (via task_id)
```

- `assignments` (TASK-404, §4) is the canonical "current assignment" table. `assignments.task_id → tasks.id` and `assignments.agent_id → agents.id`. The partial unique index `uq_assignments_one_active_per_task` (on `(task_id) WHERE status = 'active'`) enforces the "one active per task" invariant at the DB level; the Go service layer also enforces it as a defense-in-depth check.
- `tasks.AssigneeID` (existing column on the pre-existing `tasks` table) is a denormalized cache of the current `assignments.agent_id WHERE status = 'active'`. The convenient pointer for reads, but `assignments` is the source of truth. Updated in the same transaction as the assignment write.
- `assignment_events` (TASK-404, §5) records the history. Three FKs: `assignment_id → assignments.id` (the binding the event describes), `task_id → tasks.id` (denormalized for direct task-history queries), and `agent_id → agents.id` (the assignee at the time of the event, nullable for events that happened before the agent was bound).
- `executions.task_id → tasks.id` and `executions.agent_id → agents.id` (TASK-405, see §8). The `status`, `started_at`, `completed_at`, and `error_message` columns on `executions` are the per-execution detail.
- `agent_state_events` (deferred — see §14) is the **agent lifecycle history** (idle → busy → error transitions). Owned by the same `agents` row; not currently populated in Sprint 4.

---

## 12. Migration plan

| Migration                                          | Topic                          | Adds / changes                                                                                                                                                                                                                          | TASK    |
|----------------------------------------------------|--------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------|
| `016_agent_registry.sql`                           | Consolidated agent registry    | `capabilities` + seed + `agents` extensions (`retired_at`, `metadata`, `version`, `last_active_at`, `status` CHECK, `UNIQUE (project_id, name)`, GIN indexes, partial `idx_agents_project_status`)                                       | 402     |
| `017_create_agent_capabilities.sql`                | Capability join                | `agent_capabilities` with `(agent_id, capability_id)` PK, `proficiency INT CHECK 1-5`, `granted_at TIMESTAMPTZ`, `granted_by UUID` (audit, nullable); 3 indexes (PK, agent_id, capability_id)                                                | 403     |
| `018_add_task_required_capabilities.sql`           | Task requirements              | `tasks.required_capabilities JSONB NOT NULL DEFAULT '[]'::jsonb` + GIN index `idx_tasks_required_capabilities_gin`                                                                                                                         | 403     |
| `019_create_assignments.sql`                       | Assignment records             | `assignments` (`id uuid pk`, `task_id` → `tasks.id`, `agent_id` → `agents.id`, `assigned_at` NOT NULL, `completed_at` nullable, `status TEXT` CHECK (`'active','superseded','completed','cancelled'`)) + 3 indexes — `idx_assignments_task_id` (plain), `idx_assignments_agent_id` (plain), `uq_assignments_one_active_per_task` UNIQUE partial on `(task_id) WHERE status = 'active'` | 404     |
| `020_create_assignment_events.sql`                 | Assignment history             | `assignment_events` (`id uuid pk`, `assignment_id` → `assignments.id` CASCADE, `task_id` → `tasks.id` CASCADE denormalized, `agent_id` → `agents.id` SET NULL denormalized, `assigned_by` → `users.id` SET NULL nullable, `assigned_at`, `action TEXT` 3-value CHECK (`'assign','reassign','unassign'`), `notes TEXT` NULL) + 3 indexes — `idx_assignment_events_task_at` on `(task_id, assigned_at DESC)`, `idx_assignment_events_agent_at` on `(agent_id, assigned_at DESC) WHERE agent_id IS NOT NULL`, `idx_assignment_events_assignment_id` on `(assignment_id)`            | 404     |
| `021_create_agent_state_events.sql`                | Agent state history            | `agent_state_events` + `idx_agent_state_events_agent`                                                                                                                                                                                     | —       |
| `022_add_deliverable_versioning.sql`               | Deliverable versioning prep    | Additive ALTER on the 009 `deliverables` table: `ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()` (idempotent via DO block, set on every UPDATE) + 3 indexes — `ix_deliverables_task_id_created_at` on `(task_id, created_at DESC)`, `ix_deliverables_agent_id_created_at` on `(agent_id, created_at DESC)`, `ix_deliverables_task_id_version` on `(task_id, version DESC)` (plain composite, "current version per task" lookup). The 009 base indexes (`idx_deliverables_task_id`, `idx_deliverables_agent_id`, `idx_deliverables_task_agent`) are kept. | 406     |
| `023_create_deliverable_versions.sql`              | Deliverable version history    | Fresh CREATE of `deliverable_versions` (table did NOT exist in 009): `id uuid pk`, `deliverable_id uuid fk → deliverables(id) CASCADE`, `version int NOT NULL`, `title text NOT NULL`, `content text NOT NULL`, `created_at timestamptz NOT NULL DEFAULT NOW()`, `created_by uuid NULL` (no DB-level FK, validated at app layer); `UNIQUE(deliverable_id, version)`, `CHECK (version > 0)`; + 1 partial index `ix_deliverable_versions_created_by ON (created_by) WHERE created_by IS NOT NULL` (the UNIQUE creates an implicit btree that backs the list-versions path). | 406     |
| `024_create_executions.sql`                        | Execution records              | `executions` (`id uuid pk`, `task_id uuid fk → tasks(id)`, `agent_id uuid fk → agents(id)`, `status CHECK (pending, running, completed, failed)`, `started_at`, `completed_at` nullable, `error_message` nullable, `created_at`, `updated_at`); 3 indexes — `ix_executions_task_id_started_at` on `(task_id, started_at DESC NULLS LAST)`, `ix_executions_agent_id_started_at` on `(agent_id, started_at DESC NULLS LAST)`, `ix_executions_in_flight` partial on `(status) WHERE status IN ('pending','running')` | 405     |

All migrations are **additive** and **forward-compatible**: no existing data is modified, no existing column types change.

> **Note on migration count:** Total Sprint 4 migrations: 9 (016-024). Original plan: 8. TASK-402 consolidated 016 from 2 files into 1. TASK-403 added 017 (agent_capabilities join) and 018 (tasks.required_capabilities). TASK-404 added 019 (create_assignments, with partial unique index for one-active-per-task) and 020 (create_assignment_events, FK to 019). TASK-405 adds 024 (create_executions). 021 (agent_state_events) is reserved for a future sprint; 022-023 are for TASK-406 (deliverable_versioning).

---

## 13. Auth (Sprint 4 dev affordance)

API-key validation in Sprint 4 is performed against an **in-memory** `APIKeyStore` (TASK-418), not a Postgres table. This is a deliberate dev affordance to keep auth wiring testable without committing to a key-management schema in Sprint 4. The store is populated at process start from a static seed and does not survive restarts.

A future sprint will replace this with a persistent `api_keys` table (with rotation, scopes, and audit). When that lands, this section will be replaced by a real schema entry.

---

## 14. Open questions / forward work

- **Binary content:** the `content TEXT` column is fine for source code / docs. Binary artifacts (e.g. compiled binaries, screenshots) will need an S3 key in `metadata` and a separate `artifact` table — deferred to Sprint 6+ along with the artifact viewer.
- **Search:** the `metadata_gin` indexes hint at `jsonb_path_ops` for full-text search; we are not adding `tsvector` columns in Sprint 4. Trigram search on `name`/`title` is a follow-up.
- **Multi-tenancy:** every domain table is scoped to `project_id`. A `tenant_id` column is **not** introduced in Sprint 4 (single-tenant per deployment).
- **`agent_capabilities.proficiency` default:** what should the default be — `1`, `3`, or `NULL`? TASK-403 leaves the column nullable (per the `agent_capabilities_proficiency_chk` constraint), so the v1.0 contract needs to pick a default semantics for the UI. Recommended: `NULL` semantically means "untrained / rating not yet provided" (no proficiency signal); the UI renders an empty state. Anything else forces a magic number on day-zero grants.
- **`021 create_agent_state_events` timing:** when does this migration get implemented? Likely Sprint 5 alongside the lifecycle-state event sourcing work (the Agent Activity Dashboard in TASK-410 needs the state-transition timeline). The TASK column is `—` for now; the migration number is reserved to keep the Sprint 4 numbering stable.
