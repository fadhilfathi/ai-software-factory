# Migration Guide

> **Applies to:** AI Software Factory v1.0.0  
> **Last updated:** 2026-06-10

This guide covers upgrading the AI Software Factory platform between releases.
Each section targets a specific **from → to** path and includes configuration
changes, data migration steps, and rollback procedures.

---

## Table of Contents

- [Migration Overview](#migration-overview)
- [Pre-release → v1.0.0](#pre-release--v100)
- [Fresh Installation (no prior version)](#fresh-installation-no-prior-version)
- [Configuration Migration](#configuration-migration)
- [Data Migration](#data-migration)
- [Rollback Procedures](#rollback-procedures)
- [Verification Checklist](#verification-checklist)
- [Troubleshooting](#troubleshooting)
- [Appendix: Version Compatibility Matrix](#appendix-version-compatibility-matrix)

---

## Migration Overview

| From | To | Downtime Required | Migration Time (est.) |
|------|----|-------------------|----------------------|
| Pre-release / prototype | v1.0.0 | Yes (~15 min) | 30–60 min |
| Fresh install (none) | v1.0.0 | No | 5 min |

**Key principles:**

- Migrations are **additive** — existing data is never destroyed without
  explicit confirmation.
- Always take a **database snapshot** before starting a migration.
- Each release's migration steps are **independently runnable** — you can skip
  versions but must run the steps for each skipped version in order.
- Rollback instructions are provided for every migration path.

---

## Pre-release → v1.0.0

If you ran a prototype, development build, or early-access version of the AI
Software Factory before the v1.0.0 release, use this section.

### 1. Prerequisites

Before starting:

```bash
# Verify current versions
docker --version                # Must be 24+
docker compose version          # Must be 2.24+
git --version                   # Must be 2.40+

# Check PostgreSQL version (if running natively)
psql --version                  # Must be 16

# Check Go version (if developing natively)
go version                      # Must be 1.22+
```

### 2. Backup Existing Data

```bash
# 1. Database dump
docker compose exec db pg_dump -U postgres project > backup_$(date +%Y%m%d_%H%M%S).sql

# 2. Environment configuration
cp .env .env.backup_$(date +%Y%m%d_%H%M%S)

# 3. Note your current image tags (for rollback)
docker images --filter=reference='ghcr.io/fadhilfathi/ai-software-factory/*' --format 'table {{.Repository}}:{{.Tag}}'
```

### 3. Stop the Running Stack

```bash
docker compose down
```

### 4. Update the Repository

```bash
# Fetch the latest release tag
git fetch --tags origin
git checkout v1.0.0

# Or pull the latest main branch
git pull origin main
```

### 5. Review Configuration Changes

Compare your existing `.env` with the new `.env.example`:

```bash
diff .env .env.example
```

**New variables in v1.0.0:**

| Variable | Default | Required | Purpose |
|----------|---------|----------|---------|
| `API_PORT` | `8080` | Yes | API server listen port |
| `LOG_LEVEL` | `info` | No | Log verbosity (debug/info/warn/error) |
| `FRONTEND_PORT` | `3000` | Yes | Frontend dev server port |
| `NEXT_PUBLIC_API_URL` | `http://api:8080/v1` | Yes | API base URL for frontend |
| `POSTGRES_USER` | `postgres` | Yes | Database user |
| `POSTGRES_PASSWORD` | `***` | Yes | Database password |
| `POSTGRES_DB` | `project` | Yes | Database name |
| `POSTGRES_PORT` | `5432` | Yes | Database port |

**Removed variables:** None.

**Changed defaults:** None.

### 6. Apply Database Migrations

Database migrations run automatically on container startup. To run them
manually:

```bash
# Rebuild and start
docker compose up -d --build

# Verify migration status via health endpoint
curl http://localhost:8080/v1/healthz
# Expected: {"status":"ok"}
```

If you need to run migrations against an existing database without starting the
full stack:

```bash
docker compose run --rm api ./api -migrate-only
```

### 7. Verify Data Integrity

```bash
# Check project count
curl -H "Authorization: Bearer <token>" http://localhost:8080/v1/projects?limit=1

# Verify agent records exist
curl -H "Authorization: Bearer <token>" http://localhost:8080/v1/agents?limit=1

# Run health check
./scripts/healthcheck.sh
```

### 8. Smoke Test Key Workflows

1. Authenticate: `POST /v1/auth/login` with existing credentials
2. List projects: `GET /v1/projects` with pagination
3. Create a task: `POST /v1/tasks` with required fields
4. Verify realtime events: Check dashboard SSE connection

---

## Fresh Installation (no prior version)

For new deployments, no migration is needed. Follow the quick start:

```bash
git clone https://github.com/fadhilfathi/AI-Software-Factory.git
cd AI-Software-Factory
cp .env.example .env
# Edit .env to set POSTGRES_PASSWORD
docker compose up -d --build
```

That's it. The database is created and migrations run automatically on first
startup. No data migration steps are required.

---

## Configuration Migration

When moving between environments (e.g., dev → staging → production), use
environment-specific `.env` files:

```bash
# Copy environment-specific configs
cp .env.production .env

# Or use Docker Compose profiles for different environments
docker compose --profile prod up -d
```

**Key environment-specific settings:**

| Setting | Development | Staging | Production |
|---------|-------------|---------|------------|
| `LOG_LEVEL` | `debug` | `debug` | `info` |
| API JWT expiry | 24h | 24h | 1h |
| Rate limit (per user) | 1000/h | 1000/h | 100/h |
| CORS origins | `*` | `https://staging.example.com` | `https://app.example.com` |

---

## Data Migration

### Schema Migrations

Database schema changes are managed via the application's migration system.
Migrations are:

- **Idempotent** — running the same migration twice is safe.
- **Ordered** — applied in sequence; the system tracks which have run.
- **Versioned** — each migration is tied to its release.

To check migration status:

```bash
docker compose exec api ./api -migrate-status
```

To manually trigger pending migrations:

```bash
docker compose exec api ./api -migrate
```

### Data Export / Import

To transfer data between environments:

```bash
# Export
docker compose exec db pg_dump -U postgres --data-only project > data_export.sql

# Import (target database must be empty or migrated)
docker compose exec -T db psql -U postgres project < data_export.sql
```

> **Note:** API keys and password hashes are included in the export. Treat the
> dump file as sensitive — store it securely and delete after use.

---

## Rollback Procedures

### Rollback to Previous Release

```bash
# 1. Stop the current stack
docker compose down

# 2. Checkout the previous release
git checkout <previous-tag>

# 3. Restore the previous database snapshot
docker compose up -d db
docker compose exec -T db psql -U postgres project < backup_<date>.sql

# 4. Start the previous stack
docker compose up -d --build

# 5. Verify health
./scripts/healthcheck.sh
```

### Rollback a Failed Migration

If a migration fails mid-way:

1. The stack logs the error and stops.
2. Check the logs: `docker compose logs api | grep migrate`
3. The failing migration is NOT applied — the previous schema is intact.
4. Fix the issue (configuration, permissions, disk space) and re-run:
   `docker compose up -d`

If a migration partially applied (rare — migrations are transactional):

```bash
# Restore the pre-migration database snapshot
docker compose exec -T db psql -U postgres project < backup_<date>.sql

# Roll back the release
git checkout <previous-tag>
docker compose up -d --build
```

---

## Verification Checklist

Use this checklist after every migration:

- [ ] All services start without errors (`docker compose ps` → all `Up`)
- [ ] Health endpoint returns `{"status":"ok"}` (`GET /v1/healthz`)
- [ ] Database migrations applied successfully (check logs for `migration applied`)
- [ ] Existing projects load correctly (`GET /v1/projects`)
- [ ] Agent orchestration starts and completes (`POST /v1/agents`)
- [ ] Authentication works for existing users (`POST /v1/auth/login`)
- [ ] API keys continue to work (`GET /v1/projects` with API key)
- [ ] Webhook deliveries are not failing (`GET /v1/webhooks/events`)
- [ ] Frontend loads without console errors
- [ ] Realtime connections established (SSE endpoint active)
- [ ] Rollback plan documented and backup available

---

## Troubleshooting

### Services fail to start after upgrade

```bash
# Check service logs
docker compose logs api
docker compose logs frontend
docker compose logs db

# Common causes:
#   - Missing .env variables (compare with .env.example)
#   - Port conflicts (check if :8080 or :3000 is already in use)
#   - Database connection refused (PostgreSQL not fully started yet)
```

### Database connection errors

```bash
# Verify database is accepting connections
docker compose exec db pg_isready -U postgres

# Check the POSTGRES_PASSWORD in .env matches the actual database password
# If password was rotated, update .env and restart
```

### Migration version mismatch

```bash
# If the migration tracker says "already applied" for a version you need:
docker compose exec api ./api -migrate-force <version>

# Or reset the tracker (last resort — only if you've verified schema is correct):
docker compose exec api ./api -migrate-reset
```

### Slow agent startup after migration

Agents connect to the configured LLM provider on first use. If you changed
provider credentials or endpoints, verify the API configuration:

```bash
# Check API logs for provider connection errors
docker compose logs api | grep -i "llm\|provider\|model"
```

---

## Appendix: Version Compatibility Matrix

| App Version | API Version | DB Schema | Docker Compose | Min Go | Min Node.js | Min PostgreSQL |
|------------|-------------|-----------|----------------|--------|-------------|----------------|
| v1.0.0     | v1          | v1        | 2.24+          | 1.22+  | 20 (LTS)    | 16             |

---

## Need Help?

- **GitHub Issues:** Open a bug or question at
  [github.com/fadhilfathi/AI-Software-Factory/issues](https://github.com/fadhilfathi/AI-Software-Factory/issues)
- **Documentation:** [README](./README.md) · [User Guide](./docs/user-guide.md) ·
  [Developer Guide](./docs/developer-guide.md) · [Deployment Guide](./docs/deployment-guide.md)
- **Changelog:** [CHANGELOG.md](./CHANGELOG.md)
- **Release Notes:** [RELEASE_NOTES.md](./RELEASE_NOTES.md)
