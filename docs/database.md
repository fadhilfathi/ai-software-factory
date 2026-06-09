# AI Software Factory — Database Design

## Entity-Relationship Diagram

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│    users     │     │    teams     │     │ team_members │
├──────────────┤     ├──────────────┤     ├──────────────┤
│ id (PK)      │◀───┐│ id (PK)      │◀───┐│ id (PK)      │
│ email        │    ││ name         │    ││ team_id (FK) │──▶│
│ name         │    ││ owner_id(FK) │──▶││ user_id (FK) │──▶│
│ password_hash│    ││ created_at   │    ││ role         │
│ role         │    │└──────────────┘    ││ created_at   │
│ avatar_url   │    │                    │└──────────────┘
│ created_at   │    │                    │
│ updated_at   │    │                    │
└──────┬───────┘    │                    │
       │            │                    │
       │            │                    │
       ▼            │                    │
┌──────────────┐    │                    │
│   projects   │    │                    │
├──────────────┤    │                    │
│ id (PK)      │    │                    │
│ name         │    │                    │
│ description  │    │                    │
│ status       │    │                    │
│ owner_id(FK) │──▶│                    │
│ template     │    │                    │
│ config (JSON)│    │                    │
│ created_at   │    │                    │
│ updated_at   │    │                    │
└──────┬───────┘    │                    │
       │            │                    │
       │            │                    │
       ▼            │                    │
┌──────────────┐    │                    │
│    agents    │    │                    │
├──────────────┤    │                    │
│ id (PK)      │    │                    │
│ project_id   │◀──┘                    │
│ type         │    │                    │
│ status       │    │                    │
│ config (JSON)│    │                    │
│ session_id   │    │                    │
│ started_at   │    │                    │
│ ended_at     │    │                    │
└──────┬───────┘    │                    │
       │            │                    │
       │            │                    │
       ▼            │                    │
┌──────────────┐    │                    │
│    tasks     │    │                    │
├──────────────┤    │                    │
│ id (PK)      │    │                    │
│ project_id   │◀──┘                    │
│ title        │    │                    │
│ description  │    │                    │
│ type         │    │                    │
│ status       │    │                    │
│ priority     │    │                    │
│ assignee_id  │    │                    │
│ agent_id (FK)│──▶│                    │
│ parent_id    │    │                    │
│ acceptance_  │    │                    │
│   criteria   │    │                    │
│ estimated_   │    │                    │
│   hours      │    │                    │
│ created_at   │    │                    │
│ updated_at   │    │                    │
└──────┬───────┘    │                    │
       │            │                    │
       ├────────────┼────────────────────┘
       │            │
       ▼            ▼
┌──────────────┐  ┌──────────────┐
│code_artifacts│  │   reviews    │
├──────────────┤  ├──────────────┤
│ id (PK)      │  │ id (PK)      │
│ task_id (FK) │  │ task_id (FK) │
│ project_id   │  │ project_id   │
│ file_path    │  │ commit_sha   │
│ content      │  │ reviewer_id  │
│ language     │  │ status       │
│ version      │  │ score        │
│ size_bytes   │  │ issues (JSON)│
│ created_at   │  │ metrics(JSON)│
│ updated_at   │  │ created_at   │
└──────────────┘  └──────────────┘
       │
       ▼
┌──────────────┐
│ deployments  │
├──────────────┤
│ id (PK)      │
│ project_id   │
│ environment  │
│ status       │
│ version      │
│ branch       │
│ commit_sha   │
│ url          │
│ started_at   │
│ completed_at │
│ rollback_id  │
└──────────────┘
       │
       ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│notifications │  │ audit_logs   │  │webhook_config│
├──────────────┤  ├──────────────┤  ├──────────────┤
│ id (PK)      │  │ id (PK)      │  │ id (PK)      │
│ user_id (FK) │  │ user_id      │  │ project_id   │
│ type         │  │ action       │  │ url          │
│ title        │  │ resource     │  │ events(JSON) │
│ message      │  │ resource_id  │  │ secret       │
│ read         │  │ details(JSON)│  │ active       │
│ channel      │  │ ip_address   │  │ last_used_at │
│ created_at   │  │ created_at   │  │ created_at   │
└──────────────┘  └──────────────┘  └──────────────┘
```

## Core Tables

### users
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255),
    role VARCHAR(50) DEFAULT 'user',
    avatar_url TEXT,
    oauth_provider VARCHAR(50),
    oauth_id VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_oauth ON users(oauth_provider, oauth_id);
```

### teams
```sql
CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    owner_id UUID REFERENCES users(id) ON DELETE SET NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_teams_owner ON teams(owner_id);
```

### projects
```sql
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) DEFAULT 'initializing',
    owner_id UUID REFERENCES users(id) ON DELETE SET NULL,
    team_id UUID REFERENCES teams(id) ON DELETE SET NULL,
    template VARCHAR(100),
    config JSONB DEFAULT '{}',
    progress INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_projects_owner ON projects(owner_id);
CREATE INDEX idx_projects_team ON projects(team_id);
CREATE INDEX idx_projects_status ON projects(status);
```

### agents
```sql
CREATE TABLE agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(50) DEFAULT 'spawning',
    config JSONB DEFAULT '{}',
    session_id VARCHAR(255),
    model VARCHAR(100),
    tokens_used INTEGER DEFAULT 0,
    started_at TIMESTAMPTZ DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_agents_project ON agents(project_id);
CREATE INDEX idx_agents_status ON agents(status);
```

### tasks
```sql
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(50) DEFAULT 'backlog',
    priority VARCHAR(20) DEFAULT 'should_have',
    assignee_id UUID REFERENCES users(id) ON DELETE SET NULL,
    agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    parent_id UUID REFERENCES tasks(id) ON DELETE SET NULL,
    acceptance_criteria JSONB DEFAULT '[]',
    estimated_hours DECIMAL(6,2),
    actual_hours DECIMAL(6,2),
    dependencies UUID[] DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_tasks_project ON tasks(project_id);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_agent ON tasks(agent_id);
CREATE INDEX idx_tasks_parent ON tasks(parent_id);
```

### code_artifacts
```sql
CREATE TABLE code_artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    content TEXT NOT NULL,
    language VARCHAR(50),
    version INTEGER DEFAULT 1,
    size_bytes INTEGER,
    checksum VARCHAR(64),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_code_artifacts_task ON code_artifacts(task_id);
CREATE INDEX idx_code_artifacts_project ON code_artifacts(project_id);
CREATE INDEX idx_code_artifacts_path ON code_artifacts(file_path);
```

### reviews
```sql
CREATE TABLE reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    commit_sha VARCHAR(40),
    reviewer_id UUID REFERENCES users(id) ON DELETE SET NULL,
    agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    status VARCHAR(50) DEFAULT 'pending',
    result VARCHAR(50),
    score DECIMAL(3,1),
    issues JSONB DEFAULT '[]',
    metrics JSONB DEFAULT '{}',
    comments TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_reviews_task ON reviews(task_id);
CREATE INDEX idx_reviews_project ON reviews(project_id);
CREATE INDEX idx_reviews_status ON reviews(status);
```

### deployments
```sql
CREATE TABLE deployments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    environment VARCHAR(50) NOT NULL,
    status VARCHAR(50) DEFAULT 'queued',
    version VARCHAR(100),
    branch VARCHAR(100),
    commit_sha VARCHAR(40),
    url TEXT,
    config JSONB DEFAULT '{}',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    rollback_id UUID REFERENCES deployments(id),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_deployments_project ON deployments(project_id);
CREATE INDEX idx_deployments_env ON deployments(environment);
CREATE INDEX idx_deployments_status ON deployments(status);
```

### notifications
```sql
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    title VARCHAR(255),
    message TEXT,
    data JSONB DEFAULT '{}',
    read BOOLEAN DEFAULT FALSE,
    channel VARCHAR(50) DEFAULT 'in_app',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_notifications_user ON notifications(user_id);
CREATE INDEX idx_notifications_read ON notifications(read);
CREATE INDEX idx_notifications_created ON notifications(created_at);
```

### audit_logs
```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    resource VARCHAR(100) NOT NULL,
    resource_id UUID,
    details JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource, resource_id);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at);
```

## Indexing Strategy

### Primary Indexes
- All UUID primary keys (B-tree)
- Foreign key columns for JOIN performance

### Secondary Indexes
- Status columns for filtered queries
- Timestamp columns for time-range queries
- Composite indexes for common query patterns:
  - `tasks(project_id, status)` — task board queries
  - `reviews(project_id, status)` — review queue
  - `deployments(project_id, environment)` — deployment history

### Full-Text Search
- Elasticsearch for audit log search
- PostgreSQL `pg_trgm` for fuzzy project name search

## Partitioning

### audit_logs
- Partition by month: `audit_logs_2026_06`
- Automatic partition creation via pg_partman
- Old partitions archived to cold storage

### notifications
- Partition by month for similar reasons
- Users can purge old notifications

## Migration Strategy

1. **Tool:** Knex.js or Prisma Migrate
2. **Pattern:** Forward-only migrations (no down migrations in production)
3. **Versioning:** Sequential migration numbers
4. **Review:** All migrations reviewed before merge
5. **Testing:** Migrations run against test database before production

## Data Retention Policy

| Table | Hot Data | Cold Data | Deletion |
|-------|----------|-----------|----------|
| audit_logs | 90 days | 1 year (S3) | After 1 year |
| notifications | 30 days | None | After 30 days |
| code_artifacts | Current version | All versions (Git) | Never (Git handles) |
| reviews | Active | Archived after 90 days | Never |
| deployments | Last 50 per project | Archived | After 180 days |

## Backup and Recovery

### Automated Backups
- **Frequency:** Daily at 2:00 AM UTC
- **Retention:** 30 days
- **Storage:** Cross-region S3 bucket
- **Format:** PostgreSQL custom format (`pg_dump -Fc`)

### Point-in-Time Recovery
- WAL archiving enabled
- Recovery window: 7 days
- Recovery time: < 1 hour for most scenarios

### Recovery Procedures
1. **Single table restore:** `pg_restore --table=users backup.dump`
2. **Full database restore:** `pg_restore -d ai_factory backup.dump`
3. **Point-in-time:** Replay WAL to specific timestamp

### Backup Testing
- Monthly restoration to test environment
- Verify data integrity checksums
- Document recovery time metrics
