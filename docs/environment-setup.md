# Environment Setup Guide

> **AI Software Factory** — local development environment prerequisites, configuration, and verification.

---

## Table of Contents

- [Prerequisites](#prerequisites)
- [Repository Setup](#repository-setup)
- [Environment Variables](#environment-variables)
- [First-Time Setup](#first-time-setup)
- [Verification](#verification)
- [Windows Notes](#windows-notes)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

| Tool | Version | Required For | Install |
|------|---------|--------------|---------|
| Docker | 24+ | Running the full stack (recommended) | [docs.docker.com/get-docker](https://docs.docker.com/get-docker/) |
| Go | 1.25+ | Backend development / debugging | [go.dev/dl](https://go.dev/dl/) |
| Node.js | 22+ | Frontend development / debugging | [nodejs.org](https://nodejs.org/) |
| Git | 2.40+ | Version control | [git-scm.com](https://git-scm.com/) |

**Minimum recommendation:** Docker only. The full stack runs inside containers. Go and Node.js are only needed if you want to run binaries natively for Go backend or frontend debugging.

### Docker Compose Plugin

Docker Desktop includes the compose plugin by default. On Linux, install it separately:

```bash
sudo apt-get install docker-compose-plugin   # Debian/Ubuntu
sudo dnf install docker-compose-plugin       # Fedora
```

Verify:

```bash
docker compose version
# Expected: Docker Compose version v2.x.x
```

---

## Repository Setup

```bash
# Clone
git clone https://github.com/fadhilfathi/AI-Software-Factory.git
cd AI-Software-Factory

# Configure environment
cp .env.example .env
```

> **Security warning:** The default `.env` values are for local development only.
> Never use default credentials (`postgres`/`postgres`, etc.) in a production or
> internet-facing environment. Generate strong random passwords instead.

---

## Environment Variables

All variables are documented in `.env.example` at the project root.

| Variable | Default | Description |
|----------|---------|-------------|
| `API_PORT` | `8080` | Port the Go API listens on (container and host) |
| `LOG_LEVEL` | `info` | Log verbosity: `debug`, `info`, `warn`, `error` |
| `FRONTEND_PORT` | `3000` | Port the Next.js frontend listens on (container and host) |
| `NEXT_PUBLIC_API_URL` | `http://api:8080/v1` | API base URL used by the frontend browser code |
| `POSTGRES_USER` | `postgres` | PostgreSQL user |
| `POSTGRES_PASSWORD` | `postgres` | PostgreSQL password |
| `POSTGRES_DB` | `project` | PostgreSQL database name |
| `POSTGRES_PORT` | `5432` | PostgreSQL host port mapping |

> **Note on `NEXT_PUBLIC_API_URL`:** Inside Docker Compose, `api` resolves to the
> API container. When running frontend outside Docker, set this to `http://localhost:8080/v1`.

---

## First-Time Setup

### Option A: Docker Compose (Recommended)

The fastest path to a running environment — no local toolchain required beyond Docker.

```bash
# Build images and start all services
docker compose up -d --build

# Verify all services are healthy
./scripts/healthcheck.sh

# Tail logs
docker compose logs -f
```

### Option B: Native Development

Run components directly on your machine for faster iteration.

**API (Terminal 1):**

```bash
cd src
go run ./cmd/main.go
# Server starts on :8080
```

**Frontend (Terminal 2):**

```bash
cd frontend
npm ci
npm run dev
# Dev server starts on :3000
```

**Database (Terminal 3 or Docker):**

```bash
# Start only the database via Docker
docker compose up -d db
```

### Option C: Hybrid

Run the database in Docker while developing API and frontend natively:

```bash
docker compose up -d db
cd src && go run ./cmd/main.go     # Terminal 1
cd frontend && npm run dev         # Terminal 2
```

---

## Verification

Run the health check to confirm everything is working:

```bash
# Check all services via Docker
./scripts/healthcheck.sh

# Or, check individual services
./scripts/healthcheck.sh api
./scripts/healthcheck.sh frontend

# JSON output (for scripting / monitoring tools)
./scripts/healthcheck.sh --json
```

**Expected output (healthy stack):**

```
=== Health Check: 2026-06-10T13:00:00Z ===
  ✅ API (http://localhost:8080/v1/healthz) — HTTP 200
  ✅ Frontend (http://localhost:3000) — HTTP 200
  ✅ api — healthy
  ✅ frontend — healthy
```

---

## Windows Notes

- **Git Bash (recommended):** All shell scripts (`scripts/*.sh`) work correctly
  under Git Bash / MSYS2. The project is tested on Windows 10 with Git Bash.
- **Docker Desktop:** Use the WSL 2 backend for best performance. Ensure WSL 2
  is enabled in Docker Desktop → Settings → General → "Use WSL 2 based engine".
- **Line endings:** Git is configured (via `.gitattributes` or `.gitignore` rules)
  to handle CRLF/LF automatically. Scripts must stay LF (Unix line endings).
- **File sharing:** The project lives under `C:\Users\<user>\OneDrive\Documents\`.
  Docker Desktop needs this drive shared. Add it in Docker Desktop →
  Settings → Resources → File Sharing.
- **Port conflicts:** If ports 8080, 3000, or 5432 are taken, change them in
  `.env` (see [Environment Variables](#environment-variables)).

---

## Troubleshooting

| Symptom | Likely Cause | Fix |
|---------|-------------|-----|
| `docker: command not found` | Docker not installed | Install Docker Desktop or Docker Engine |
| `docker compose: command not found` | Compose plugin missing | Install `docker-compose-plugin` or use `docker-compose` (legacy) |
| Container exits immediately | Missing `.env` or bad config | Run `cp .env.example .env` and check values |
| `port is already allocated` | Port conflict | Change port in `.env` (e.g. `API_PORT=8081`) |
| `permission denied` on scripts | Windows line endings | Run `sed -i 's/\r$//' scripts/*.sh` in Git Bash |
| API health fails with `000` | API not started or wrong port | Check `docker compose ps` for container status |
| `next dev` EADDRINUSE | Port 3000 in use | Kill the process or run `npm run dev -- -p 3001` |
