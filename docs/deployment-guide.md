# Deployment Guide

> **AI Software Factory** — deployment strategies, CI/CD pipeline reference, rollback procedures,
> and environment-specific instructions.

---

## Table of Contents

- [Deployment Overview](#deployment-overview)
- [Architecture](#architecture)
- [Option 1: Local Docker Compose Deploy](#option-1-local-docker-compose-deploy)
- [Option 2: CI/CD Pipeline (GitHub Actions → GHCR)](#option-2-cicd-pipeline-github-actions--ghcr)
- [Option 3: SSH + Manual Pull Deploy](#option-3-ssh--manual-pull-deploy)
- [Rollback Procedures](#rollback-procedures)
- [Docker Configuration Reference](#docker-configuration-reference)
- [CI/CD Workflows Reference](#cicd-workflows-reference)
- [Troubleshooting Deployments](#troubleshooting-deployments)

---

## Deployment Overview

The project supports three deployment paths, ordered by complexity:

| Method | Use Case | Infrastructure Required |
|--------|----------|------------------------|
| **Docker Compose** | Local dev, staging, single-VM prod | Docker, 1 VM or local machine |
| **CI/CD Pipeline** | Automated deploys via GitHub Actions | GitHub repo, GHCR access, target host |
| **Manual Pull** | Air-gapped or manual-control environments | Docker, SSH access to target host |

**Image registry:** Production images are pushed to **GitHub Container Registry (GHCR)**
under the repository namespace. Each push to `main` produces two tags per service:

```
ghcr.io/fadhilfathi/ai-software-factory/api:latest
ghcr.io/fadhilfathi/ai-software-factory/api:<sha>  # e.g. a1b2c3d
ghcr.io/fadhilfathi/ai-software-factory/frontend:latest
ghcr.io/fadhilfathi/ai-software-factory/frontend:<sha>
```

---

## Architecture

```
┌─────────────────────────────────────────────────┐
│                  Internet                        │
└──────────────────────┬──────────────────────────┘
                       │
                  ┌────┴────┐
                  │  Host   │  (VM / local machine)
                  │  :80/443│
                  └────┬────┘
                       │
          ┌────────────┼────────────┐
          │            │            │
     ┌────┴────┐ ┌────┴────┐ ┌────┴────┐
     │  API    │ │Frontend │ │   DB    │
     │ :8080   │ │ :3000   │ │ :5432   │
     └────┬────┘ └────┬────┘ └────┬────┘
          │            │            │
          └────────────┴────────────┘
               Docker Compose Network
```

All three services run inside a single Docker Compose network on one host. The API and
Frontend both have health checks; the Frontend depends on the API being healthy, and
the API depends on the database.

---

## Option 1: Local Docker Compose Deploy

Best for: local development, staging, demo environments, or single-VM production.

### Quick Start

```bash
# 1. Configure environment
cp .env.example .env
# Edit .env with your settings (especially passwords for production)

# 2. Build and start
docker compose up -d --build

# 3. Verify health
./scripts/healthcheck.sh
```

### Using the Deploy Script

The `scripts/deploy.sh` script wraps common Compose operations:

```bash
# Start the stack (default, builds if needed)
./scripts/deploy.sh

# Force rebuild images and start
./scripts/deploy.sh --build

# Pull latest images from registry and restart
./scripts/deploy.sh --pull

# Stop and remove containers + volumes
./scripts/deploy.sh down

# Tail logs for all services
./scripts/deploy.sh logs

# Tail logs for a specific service
./scripts/deploy.sh logs api

# Restart a single service
./scripts/deploy.sh restart frontend

# Restart all services
./scripts/deploy.sh restart
```

### Build Commands Reference

```bash
# Build all (Go binary + frontend + Docker images)
./scripts/build.sh

# Build only the Go API binary
./scripts/build.sh api

# Build only the frontend
./scripts/build.sh frontend

# Build only Docker images (uses cached binaries)
./scripts/build.sh docker
```

### Service Management

```bash
# Check running containers
docker compose ps

# View service logs
docker compose logs -f       # all services
docker compose logs -f api   # API only

# Restart a service
docker compose restart frontend

# Scale a service (not useful in single-host but possible)
docker compose up -d --scale api=2
```

---

## Option 2: CI/CD Pipeline (GitHub Actions → GHCR)

Best for: automated deploys from `main` branch with zero-touch rollback.

### Pipeline Flow

```
Push to main
    │
    ▼
┌──────────────────────┐
│   CI (ci.yml)        │
│   ┌──────────────┐   │
│   │ Lint (Go +   │   │
│   │ Frontend)    │   │
│   └──────┬───────┘   │
│   ┌──────┴───────┐   │
│   │ Test (Go)    │   │
│   └──────┬───────┘   │
│   ┌──────┴───────┐   │
│   │ Build (Go +  │   │
│   │ Frontend +   │   │
│   │ Docker)      │   │
│   └──────┬───────┘   │
│   ┌──────┴───────┐   │
│   │ E2E Smoke    │   │
│   │ (Compose up) │   │
│   └──────────────┘   │
└──────────┬───────────┘
           │ CI passes
           ▼
┌──────────────────────┐
│ Deploy (deploy.yml)  │
│ ┌──────────────────┐ │
│ │ Build & Push     │ │
│ │ → GHCR (api)     │ │
│ │ → GHCR (frontend)│ │
│ └────────┬─────────┘ │
│ ┌────────┴─────────┐ │
│ │ Deploy Stack     │ │
│ │ → docker compose  │ │
│ │   pull & up -d   │ │
│ └────────┬─────────┘ │
│ ┌────────┴─────────┐ │
│ │ Health Check     │ │
│ │ → 30s timeout    │ │
│ └────────┬─────────┘ │
│    ┌─────┴─────┐     │
│   ✅ Pass    ❌ Fail  │
│   (done)      │      │
│          ┌────┴────┐ │
│          │ Rollback │ │
│          └─────────┘ │
└──────────────────────┘
```

### Workflow Files

- **`.github/workflows/ci.yml`** — Triggered on push/PR to `main`. Runs lint → test → build → e2e smoke. Failing-fast: lint failures cancel dependent jobs.
- **`.github/workflows/deploy.yml`** — Triggered on push to `main`. Must be protected by branch protection (CI must pass first). Builds Docker images, pushes to GHCR, pulls on the target host, deploys, and validates health.

### GitHub Secrets Required

| Secret | Description |
|--------|-------------|
| `GITHUB_TOKEN` | Auto-provided. Needs `packages: write` permission for GHCR push. |

No other secrets are required for the default pipeline. If deploying to a remote host
(instead of the runner), add SSH or cloud credentials as needed.

### Setting Up GHCR Access

The deploy workflow logs into GHCR using the built-in `GITHUB_TOKEN`. To pull images
from another machine, create a Personal Access Token (classic) with `read:packages` scope:

```bash
echo <PAT> | docker login ghcr.io -u <username> --password-stdin
```

---

## Option 3: SSH + Manual Pull Deploy

Best for: environments without GitHub Actions runners, or manual promotion workflows.

```bash
# On the target host
cd /opt/ai-software-factory

# Pull latest images
docker compose pull

# Recreate containers with new images
docker compose up -d --remove-orphans

# Verify health
./scripts/healthcheck.sh
```

For automation, wrap this in a cron job or webhook:

```bash
# Example: /etc/cron.d/ai-software-factory-deploy
# Run every hour — compare running image with latest and redeploy if different
0 * * * * root cd /opt/ai-software-factory && docker compose pull && docker compose up -d --remove-orphans
```

---

## Rollback Procedures

### Rollback via CI/CD (Automatic)

The deploy workflow includes an **automatic rollback** in the health check step:

1. New containers start via `docker compose up -d`
2. Pipeline waits up to 60s for all services to report `healthy`
3. If timeout expires, the step fails and the rollback block runs:
   - Calls `docker compose up -d` again with the **previous images** (still cached locally)
   - Logs a warning but does not re-throw the error
4. **Result:** The stack reverts to the previous deployment automatically

### Manual Rollback (via tags)

```bash
# Find the previous working SHA
# Option A: Check GitHub Actions deploy run history
# Option B: List tags in GHCR
docker pull ghcr.io/fadhilfathi/ai-software-factory/api:latest

# Check the "org.opencontainers.image.revision" label for the SHA
docker inspect ghcr.io/fadhilfathi/ai-software-factory/api:latest \
  --format '{{index .Config.Labels "org.opencontainers.image.revision"}}'

# Roll back to a specific SHA
export ROLLBACK_SHA="a1b2c3d"

# Manually edit docker-compose.yml to pin image tags, or use docker compose directly:
docker compose -f docker-compose.yml up -d \
  --pull=missing
# Then pull and tag the old images:
docker pull ghcr.io/fadhilfathi/ai-software-factory/api:$ROLLBACK_SHA
docker pull ghcr.io/fadhilfathi/ai-software-factory/frontend:$ROLLBACK_SHA
```

### Full Reset (destructive)

```bash
# Stop everything, delete volumes (database data included)
./scripts/deploy.sh down

# Pull the desired version
docker compose pull

# Start fresh
./scripts/deploy.sh
```

---

## Docker Configuration Reference

### Images

| Service | Dockerfile | Base Image | Runtime User |
|---------|-----------|------------|--------------|
| API | `src/Dockerfile` | `golang:1.22-alpine` → `alpine:3.20` | `appuser` (non-root) |
| Frontend | `frontend/Dockerfile` | `node:22-alpine` → `node:22-alpine` | `appuser` (non-root) |

### Compose Dependencies

```
db (postgres:16-alpine) ──→ api ──→ frontend
     └── health check: pg_isready
                       api ──→ health check: /v1/healthz
                                    frontend ──→ health check: HTTP 200 on /
```

### Port Map (Host → Container)

| Service | Host Port | Container Port | Configurable Via |
|---------|-----------|----------------|-----------------|
| API | `${API_PORT:-8080}` | `8080` | `.env` |
| Frontend | `${FRONTEND_PORT:-3000}` | `3000` | `.env` |
| DB | `${POSTGRES_PORT:-5432}` | `5432` | `.env` |

---

## CI/CD Workflows Reference

### `ci.yml` — CI Pipeline

```
Triggers:  push to main, PR targeting main
Concurrency: cancel-in-progress per branch
Jobs (parallelized where possible):
  1. lint-go        → golangci-lint + go fmt check
  2. lint-frontend  → ESLint
  3. test-go        → needs lint-go: go vet + go test -race -shuffle
  4. build-go       → needs lint-go: verify Go compile + Docker build
  5. build-frontend → needs lint-frontend: npm run build + Docker build
  6. e2e-smoke       → needs build-go + build-frontend:
                      docker compose up → wait healthy → curl endpoints
Total time: ~3-5 minutes
```

### `deploy.yml` — Deploy Pipeline

```
Triggers:  push to main
Permissions: contents: read, packages: write
Jobs:
  1. docker   → Build & Push images to GHCR (api + frontend)
               Tags: :latest and :<sha>
               Cache: GitHub Actions cache (type=gha)
  2. deploy   → needs docker:
               GHCR login → pull fresh images → docker compose up -d
               → health check (30 retries, 2s interval)
               → auto-rollback on health failure
```

---

## Troubleshooting Deployments

| Symptom | Cause | Fix |
|---------|-------|-----|
| `docker compose up` fails with "no configuration file" | Wrong working directory | Run from project root (where `docker-compose.yml` lives) |
| GHCR push fails with 403 | Token lacks `write:packages` | Check repo → Settings → Actions → General → Workflow permissions → "Read and write permissions" |
| `docker compose pull` fails | Not authenticated to GHCR | Run `docker login ghcr.io -u <user> --password-stdin` with a PAT |
| Services restart in a loop | Health check failing | Run `docker compose logs <svc>` to see startup errors |
| `Image not found` on deploy | Wrong tag or registry URL | Verify `REGISTRY` and `IMAGE_TAG` in deploy.yml match available images |
| Rollback ran but stack is still down | Rollback images also broken | Manually deploy a known-good SHA (see [Manual Rollback](#manual-rollback-via-tags)) |
| E2E smoke test fails in CI but works locally | Docker layer caching mismatch | Add `--pull` to the compose build step, or clear GitHub Actions cache |
| `CGO_ENABLED=0` build fails | C code dependency | Ensure no CGO dependencies; use pure Go or static musl builds |
