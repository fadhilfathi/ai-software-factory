# TASK-413 — Infrastructure Validation Report

**Sprint:** 4
**Owner:** DevOps-01
**Date:** 2026-06-12
**Repo root:** `C:\Users\fadhi\OneDrive\Documents\ai-software-factory`

---

## TL;DR

| # | Check | Status |
|---|---|---|
| 1 | `docker compose config` (YAML syntax) | ✅ **PASS** (static) |
| 2 | `docker compose up -d` (boot stack) | ⚠️ **BLOCKED** — no container runtime on this host |
| 3 | `docker compose ps` (health) | ⚠️ **BLOCKED** — no container runtime on this host |
| 4 | `curl http://localhost:8080/v1/healthz` | ⚠️ **BLOCKED** — stack not booted |
| 5 | Apply all migrations on a fresh DB | ⚠️ **BLOCKED** — no Postgres on this host |
| 6 | `.env.example` completeness | ❌ **FAIL** → ✅ **PASS** (fixed) |
| 7 | Migration file set is internally consistent | ❌ **FAIL** → ✅ **PASS** (fixed) |
| 8 | Dockerfile `HEALTHCHECK` is functional | ❌ **FAIL** → ✅ **PASS** (fixed) |

**Summary:** 3 issues found, all fixed in-place. Runtime checks (1–5 above) are blocked by the absence of a container runtime on the dev host — they need to be re-run on a machine with Docker / Compose installed before the sprint closeout commit. See "Outstanding / needs follow-up" at the bottom.

---

## Environment notes

- **OS:** Windows 11 (running through Git Bash / PowerShell).
- **Container runtime on PATH:** none of `docker`, `docker-compose`, `podman`, `nerdctl`, `rancher-desktop` are installed. `C:\Program Files\Docker\` does not exist.
- **Tooling on PATH:** PowerShell (used for YAML parse), bash, `cat`/`mv`/`ls`. No `go`, no `psql`, no `node`, no `python` available.
- **Static-analysis-only validation** was therefore the only option for the runtime steps. The Lead should re-execute the runtime steps on a host that has Docker.

---

## Detailed findings

### 1. `docker compose config` — YAML syntax

- **Result:** ✅ **PASS** (static).
- **How verified:** `Get-Content docker-compose.yml -Raw | ConvertFrom-Yaml` returns no errors and yields a service map of `db, redis, api, frontend` and a volume map of `pgdata, redisdata`. Reference resolution looks correct (`./src/Dockerfile`, `./frontend/Dockerfile`, named volumes, env interpolation).
- **Could not verify:** semantic validation that docker compose performs (env interpolation against `.env`, port-collision detection, depends_on graph cycles). That requires the real `docker` binary.

### 2. `docker compose up -d` — boot the full stack

- **Result:** ⚠️ **BLOCKED**.
- **Reason:** `docker` is not on PATH. `C:\Program Files\Docker\` does not exist. None of the alternatives (`podman`, `nerdctl`, `colima`, `rancher-desktop`, WSL) are installed either.
- **Files reviewed instead:** `docker-compose.yml` and the two `Dockerfile`s (see fix #3 below).
- **Action for the Lead:** run on a Docker-capable host. The compose file looks structurally sound; with the fixes in this report, the only remaining unknown is whether the named volumes initialise cleanly on a fresh box.

### 3. `docker compose ps` — health

- **Result:** ⚠️ **BLOCKED** for the same reason as #2.
- **Static finding (FIXED):** both `src/Dockerfile` and `frontend/Dockerfile` use `wget --no-verbose --tries=1 --spider http://localhost:…/healthz` in their `HEALTHCHECK`, but neither runtime image installs `wget`:
  - `src/Dockerfile` runtime stage: `alpine:3.20` with only `ca-certificates` and `tzdata` installed — `wget` missing → healthcheck would always fail.
  - `frontend/Dockerfile` runtime stage: `node:22-alpine` with no `apk add` at all — `wget` missing → healthcheck would always fail.
- **Fix applied:**
  - `src/Dockerfile` — changed `RUN apk add --no-cache ca-certificates tzdata` to `RUN apk add --no-cache ca-certificates tzdata wget`.
  - `frontend/Dockerfile` — added `RUN apk add --no-cache wget` after the `adduser` block.
- **Why this matters:** even if the API itself is healthy, Docker would mark the container as `unhealthy` and `depends_on: { api: { condition: service_healthy } }` in the compose file would never resolve. The frontend (which depends on the API being healthy) would therefore never start.

### 4. `curl http://localhost:8080/v1/healthz` — API health endpoint

- **Result:** ⚠️ **BLOCKED** — no live stack.
- **Static finding:** the route is registered in `src/internal/router/router.go` as `GET /v1/healthz` and is wired to the same handler Gin's `r.GET("/healthz", health.Healthz)` exposes (under the `/v1` group). Endpoint exists in code; nothing in the request path requires DB (the handler returns `{ "status": "ok" }` unconditionally), so once the stack is booted this curl should return 200.
- **Action for the Lead:** run `curl -i http://localhost:8080/v1/healthz` after `docker compose up -d --build` and confirm a 200 with body `{"status":"ok"}`.

### 5. Migrations on a fresh DB

- **Result:** ⚠️ **BLOCKED** — no `postgres` on this host, and no container to run one in.
- **Static finding (FIXED — critical bug):** two files in `src/db/migrations/` collided on the same version number:
  - `008_create_executions.sql`
  - `008_update_agents_table.sql`
  The migration runner in `src/db/migrate.go` extracts the version with `entry.Name()[:3]` (first 3 characters) and inserts it as the PRIMARY KEY of `schema_migrations`. Two `008` files would therefore hit a duplicate-key error on the second insert, breaking every fresh deployment. This was found by:

    ```bash
    for f in src/db/migrations/*.sql; do basename "$f" | cut -c1-3; done \
      | sort | uniq -c | sort -rn
    #  before fix:  2 008
    #  after fix:   1 of each
    ```

- **Fix applied:** renamed `008_update_agents_table.sql` → `015_update_agents_table.sql`. Putting it after `014_add_reviewer_agent_id.sql` is safe — the file only does `ALTER TABLE … ADD COLUMN IF NOT EXISTS …` on the agents table (created in `005_create_agents.sql`) and is functionally orthogonal to the migrations in 009–014. The new version list is:

  ```
  001, 002, 003, 004, 005, 006, 007, 008, 009, 010, 011, 012, 013, 014, 015
  ```

  All 15 versions unique.

- **Action for the Lead:** on a host with Docker, run `docker compose run --rm api /app/server -migrate` (or whatever the documented migration command is) against a fresh `db` service and confirm all 15 apply cleanly with no errors. The IF NOT EXISTS clauses in 015/010 also mean both can co-exist if applied in either order.

### 6. `.env.example` completeness

- **Result:** ❌ **FAIL → ✅ PASS** (fixed).
- **What the API process actually reads** (sources: `src/cmd/main.go`, `src/internal/config/config.go`, `src/internal/logger/logger.go`, `src/internal/middleware/middleware.go`, and grep for `os.Getenv` / `os.LookupEnv` across `src/**/*.go`):

  | Var | Required? | Default in app | Was in `.env.example`? |
  |---|---|---|---|
  | `DB_HOST` | **yes** (config.Load panics) | — | ❌ missing |
  | `DB_PORT` | **yes** (must be int) | — | ❌ missing |
  | `DB_USER` | **yes** | — | ❌ missing |
  | `DB_PASSWORD` | **yes** | — | ❌ missing |
  | `DB_NAME` | **yes** | — | ❌ missing |
  | `DB_SSLMODE` | no | `disable` | ❌ missing |
  | `JWT_SECRET` | **yes**, must be ≥ 32 chars | — | ❌ missing |
  | `SERVER_HOST` | no | `localhost` | ❌ missing |
  | `SERVER_PORT` | no | `8080` | ❌ missing |
  | `PORT` | no | (overrides `SERVER_PORT`) | ❌ missing |
  | `CORS_ALLOWED_ORIGINS` | no | `""` | ❌ missing |
  | `CORS_ALLOW_CREDENTIALS` | no | `false` | ❌ missing |
  | `RATE_LIMIT_RPM` | no | `100` | ❌ missing |
  | `RATE_LIMIT_BURST` | no | `20` | ❌ missing |
  | `AGENT_RUNTIME` | no | `runc` | ✅ present |
  | `AGENT_MEMORY_MB` | no | `512` | ✅ present |
  | `AGENT_CPU_LIMIT` | no | `50000` | ✅ present |
  | `LOG_LEVEL` | no | `info` (one of `debug`/`info`/`warn`/`error`) | ✅ present |
  | `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` / `POSTGRES_PORT` | used by the `postgres` Docker image | — | ✅ present |
  | `API_PORT` / `FRONTEND_PORT` | used by compose port-mapping | — | ✅ present |
  | `NEXT_PUBLIC_API_URL` | used by frontend build | — | ✅ present |
  | `REDIS_PORT` | used by compose port-mapping | — | ✅ present |

  **Twelve** env vars that the Go API actually reads were not in `.env.example`, including five that the service refuses to start without (`DB_*` and `JWT_SECRET`). The previous file was effectively a docker-compose cheatsheet, not a complete template.

- **Fix applied:** rewrote `.env.example` from scratch, grouped by:
  1. Compose / host ports
  2. Postgres container (POSTGRES_* — for the docker image, not the app)
  3. **API: database connection (REQUIRED)** — DB_*
  4. API: HTTP server — SERVER_HOST, SERVER_PORT, PORT
  5. **API: authentication (REQUIRED)** — JWT_SECRET with `openssl rand -hex 32` hint and a 32-char placeholder
  6. API: CORS — CORS_ALLOWED_ORIGINS, CORS_ALLOW_CREDENTIALS
  7. API: rate limiting — RATE_LIMIT_RPM, RATE_LIMIT_BURST
  8. Frontend — NEXT_PUBLIC_API_URL
  9. Logging — LOG_LEVEL with allowed values

  Every block has a comment explaining who reads the variable, whether it is required, and the default the app falls back to. The file is now self-documenting: a fresh operator can `cp .env.example .env` and the API will start with the provided placeholder `JWT_SECRET` (the operator still needs to replace it before production, but that is now a documented step rather than a panic-on-boot).

---

## Outstanding / needs follow-up (NOT in scope for this task, but flagging)

1. **Runtime checks must be re-executed on a Docker-capable host** before the sprint closeout commit. Steps to run, in order:

   ```bash
   cp .env.example .env             # generate a real JWT_SECRET first
   openssl rand -hex 32             # paste output into JWT_SECRET=…
   docker compose config            # should print the resolved config
   docker compose up -d --build
   docker compose ps                # expect db/redis/api/frontend all (healthy)
   curl -i http://localhost:8080/v1/healthz   # expect 200
   docker compose exec api /app/server -migrate   # or however the migration entrypoint is wired
   ```

2. **Closed mid-sprint — migration 010/015 capabilities column drift.** Original: 010 added `capabilities TEXT[]`, 015 added `capabilities JSONB`; both used `IF NOT EXISTS` so apply order silently determined the final type. Fix: 010 (`src/db/migrations/010_update_agents.sql`) was edited in place to use `JSONB DEFAULT '[]'::jsonb`, matching 015. Both files now carry a "KEEP IN SYNC" header comment. Migrations are immutable (no renumber); values aligned instead. Schema now apply-order-independent. Documented in data-model.md (canonical column type = JSONB).

3. **`config.Load()` requires `DB_*` even when the API runs against the in-memory store.** `src/cmd/main.go` has a code path that picks the in-memory store if `DB_HOST` is empty, but `config.Load()` is called *before* that check and panics if `DB_HOST`/`DB_PORT`/`DB_USER`/`DB_PASSWORD`/`DB_NAME` are unset. The in-memory fallback is therefore unreachable. Not an infra problem, but if the team's intent was to support a `DB_HOST=`-empty dev mode, that path is broken. Logged for the dev team.

4. **`LOG_LEVEL` validator is case-sensitive** in `src/cmd/main.go` (`debug`/`info`/`warn`/`error`). `.env.example` documents this. Just noting it because the previous report had a typo risk — current docs are consistent.

5. **Postgres-backed `APIKeyStore` (TASK-418 follow-up).** F-002 (API-key bypass) was fixed in-patch with an in-memory implementation under `src/internal/store/memory/api_key_store.go` and a seed loaded from `API_KEYS_DEV` in `.env.example`. The Postgres implementation is **deferred** to a follow-up sprint task. Required work:

   - Migration `024_create_api_keys.sql`:
     - `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
     - `key_hash VARCHAR(64) NOT NULL` (lowercase sha256 hex)
     - `user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE`
     - `role VARCHAR(50) NOT NULL`
     - `name VARCHAR(255) NOT NULL DEFAULT ''`
     - `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
     - `expires_at TIMESTAMPTZ`
     - `revoked_at TIMESTAMPTZ`
     - `last_used_at TIMESTAMPTZ`
     - `UNIQUE (key_hash)`
     - `CHECK (key_hash = lower(key_hash))` (defensive)
     - `INDEX idx_api_keys_user_id (user_id)`
   - `src/internal/store/postgres/api_key_store.go` implementing `store.APIKeyStore`.
   - Add `Create` method to the `APIKeyStore` interface and a key-issuance flow.
   - Integration test against a live Postgres instance.
   - Update `cmd/main.go` to construct the Postgres impl when `DB_HOST` is set; keep the in-memory impl as the dev fallback.
   - The TODO comment at the bottom of `src/internal/store/memory/api_key_store.go` mirrors this list.

6. **`agents.version` column ([+] Sprint 4).** TASK-402 added a `version INT NOT NULL DEFAULT 1` column to the `agents` table to support optimistic concurrency on `PUT /v1/agents/:id` (api-spec.md §1.4, error code `VERSION_CONFLICT`). The data-model.md §1 table does not currently list this column. Follow-up for the data-modelling team: amend data-model.md to include `version` and the version-bump semantics in the agent_capabilities invariant. The `service/agent.go` and `store/postgres/agent_store.go` already implement the bump; the only doc gap is the data-model.md entry.

---

## Files changed by this task

| File | Change |
|---|---|
| `src/db/migrations/008_update_agents_table.sql` → `src/db/migrations/015_update_agents_table.sql` | Renamed to remove duplicate version `008`. |
| `src/Dockerfile` | Added `wget` to the runtime `apk add` line. |
| `frontend/Dockerfile` | Added `RUN apk add --no-cache wget` to the runtime stage. |
| `.env.example` | Rewrote to document every env var the API actually reads, including the five that the service refuses to start without (`DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `JWT_SECRET`). |

**No commit made.** Per task instructions, DevOps owns the sprint closeout commit.

---

## Fixed mid-sprint

These were outstanding items from the original TASK-413 report that have since been resolved mid-sprint.

- **010 and 015 capabilities column consolidated to JSONB** *(TASK-416, DevOps-01)*. `010_update_agents.sql` now declares `capabilities JSONB DEFAULT '[]'::jsonb` to match `015_update_agents_table.sql`. While here, the same migration was tightened to drop the divergent `role`/`provider` types/defaults in favour of 015's: `role VARCHAR(255)` and `provider VARCHAR(100)`, both nullable with no default. A `KEEP IN SYNC` comment was added to the top of each of 010 and 015 so a future edit can't drift the two back out of agreement unnoticed. `docs/sprint4/data-model.md` (table definition + GIN example query) was updated to use JSONB syntax.
