# Infrastructure runbook — TokenRouter MiniMax Factory

**Owner:** Ops
**Date:** 2026-06-14
**Audience:** operators, on-call, Lead
**Pre-reqs:** Docker 24+ with Compose v2, GNU coreutils, `curl`

## 1. First-time local stack

```bash
# From the repo root.
cp .env.example .env
# Edit .env: set JWT_SECRET to a 32+ char random value (the Go config
# panics on shorter values). Optional: override POSTGRES_USER /
# POSTGRES_PASSWORD / DB_NAME if you don't want the defaults.
docker compose up -d --build
```

The stack brings up 4 containers:

| Service  | Port (host→container) | Image / build           | Purpose                |
|----------|-----------------------|-------------------------|------------------------|
| db       | 5432→5432             | postgres:16-alpine      | primary datastore      |
| redis    | 6379→6379             | redis:7-alpine          | cache, rate-limit      |
| api      | 8080→8080             | ./src (Go binary)       | HTTP API               |
| frontend | 3000→3000             | ./frontend (Next.js)    | web UI                 |

All four services declare `healthcheck:` blocks. `db` and `redis` must report healthy before `api` is started (compose `depends_on: condition: service_healthy`).

## 2. Health check

```bash
bash scripts/healthcheck.sh
```

Probes:
- `http://localhost:8080/healthz` (api)
- `http://localhost:3000` (frontend, redirects to `/` are fine)
- `docker compose ps` (all services `healthy` or `running`)

The script exits 0 on full health, non-zero on any failure. Use it in a loop:

```bash
until bash scripts/healthcheck.sh; do sleep 5; done
```

## 3. Common operations

### Tail logs

```bash
docker compose logs -f --tail=100 api
docker compose logs -f --tail=100 db
```

### Restart one service (preserves the others)

```bash
docker compose restart api
```

### Wipe state (DESTRUCTIVE — drops DB and Redis volumes)

```bash
docker compose down -v
docker compose up -d --build
```

### Live-reload after a Go change (rebuild only the api)

```bash
docker compose build api
docker compose up -d api
```

### Inspect the database

```bash
docker compose exec db psql -U postgres -d project
```

### Inspect Redis

```bash
docker compose exec redis redis-cli
```

## 4. Environment variable convention

The compose file uses **inline `environment:` with defaults**, not `env_file:`. This means:

- The project-root `.env` is consumed by `docker compose` at **interpolation time** (replacing `${VAR:-default}`).
- The container does **not** see the entire `.env` file at runtime — only the keys explicitly listed in each service's `environment:` block.
- Adding a new operator-only secret (e.g. `GITHUB_TOKEN`) requires both:
  1. The line in `.env` / `.env.example`, and
  2. The matching entry in the service's `environment:` block.

This is intentional: it keeps the container's environment surface minimal and explicit. Don't switch the services to `env_file: .env` without coordinating with the Lead.

## 5. Validation

```bash
python scripts/validate-infra.py
```

Static check of the compose file, Dockerfiles, env references, and scripts. Run before opening a PR that touches infra. See `docs/reset/infra-validation.md` for the latest run.

## 6. Deploy

```bash
bash scripts/deploy.sh
```

`deploy.sh` is the production deploy entry point. It:
1. Pulls the latest from the remote
2. Rebuilds images
3. Restarts the stack with a brief health-probe loop

**Do not run on prod without coordinating with the Lead.** For local/staging, it's safe to run repeatedly.

## 7. Troubleshooting

### `api` is in a restart loop

```bash
docker compose logs api --tail=200
```

Common causes:
- `JWT_SECRET` shorter than 32 chars (Go config panics on this).
- `db` or `redis` unhealthy. Check `docker compose ps`.

### Port already in use

```bash
lsof -iTCP:8080 -sTCP:LISTEN  # macOS / Linux
netstat -ano | findstr :8080  # Windows
```

Override via `.env`: `API_PORT=8090`.

### Frontend cannot reach API

The `frontend` service connects to `http://api:8080` (compose-internal DNS). If you see CORS errors in the browser, the api is unreachable from the frontend container:

```bash
docker compose exec frontend wget -qO- http://api:8080/healthz
```

If that fails, check `docker compose ps` and the api's logs.

### Postgres won't start (corrupted volume)

```bash
docker compose down
docker volume rm ai-software-factory_pgdata
docker compose up -d --build
```

## 8. CI

The CI workflow `.github/workflows/ci.yml` runs `npm run build` (frontend) and `go build ./...` (backend) on every push. It does **not** run `docker compose up` (no Docker in the runner pool today). The gitleaks workflow (`.github/workflows/secret-scan.yml`) runs on the same triggers.

For the production deploy, see the deploy host's on-call documentation. The `scripts/deploy.sh` is the canonical entry point but is intentionally idempotent — re-running it is safe.
