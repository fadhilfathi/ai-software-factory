# Setup Guide

> How to set up and run the AI Software Factory for local development.

---

## Prerequisites

| Tool | Version | Required For |
|------|---------|--------------|
| Docker | 24+ | Full stack (recommended) |
| Go | 1.22+ | Backend native development |
| Node.js | 22+ | Frontend native development |
| Git | 2.40+ | Version control |

---

## Quick Start (Docker Compose)

### 1. Clone and configure

```bash
git clone https://github.com/fadhilfathi/AI-Software-Factory.git
cd AI-Software-Factory
cp .env.example .env
```

### 2. Start all services

```bash
docker compose up -d --build
```

This starts three containers:
- **api** — Go/Gin backend on port 8080
- **frontend** — Next.js on port 3000
- **db** — PostgreSQL 16 on port 5432

### 3. Verify

```bash
./scripts/healthcheck.sh
```

Expected output:
```
✔ API (http://localhost:8080/v1/healthz) — HTTP 200
✔ Frontend (http://localhost:3000) — HTTP 200
✔ api — healthy
✔ frontend — healthy
```

Open http://localhost:3000 in your browser.

---

## Environment Variables

All variables are defined in `.env.example` at the project root.

| Variable | Default | Description |
|----------|---------|-------------|
| `API_PORT` | `8080` | Port the Go API listens on |
| `LOG_LEVEL` | `info` | Log verbosity: `debug`, `info`, `warn`, `error` |
| `FRONTEND_PORT` | `3000` | Port the Next.js frontend listens on |
| `NEXT_PUBLIC_API_URL` | `http://api:8080/v1` | API base URL used by frontend browser code |
| `DB_HOST` | *(unset)* | PostgreSQL hostname. When unset, API uses in-memory store |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `postgres` | PostgreSQL user |
| `DB_PASSWORD` | `postgres` | PostgreSQL password |
| `DB_NAME` | `project` | PostgreSQL database name |

> **Note on `NEXT_PUBLIC_API_URL`:** Inside Docker Compose, `api` resolves to the API container. When running frontend outside Docker, set this to `http://localhost:8080/v1`.

---

## Running Locally with In-Memory Store

No database required. Omit `DB_HOST` and the API uses an in-memory store.

### Terminal 1 — Backend

```bash
cd src
go run ./cmd/main.go
# Server starts on :8080 using in-memory store
```

### Terminal 2 — Frontend

```bash
cd frontend
npm ci
NEXT_PUBLIC_API_URL=http://localhost:8080/v1 npm run dev
# Dev server starts on :3000
```

### Verify

```bash
curl http://localhost:8080/v1/healthz
# {"status":"ok"}
```

All project and task data is ephemeral — data resets on server restart.

---

## Running with PostgreSQL

Set `DB_HOST` to enable the PostgreSQL-backed store:

```bash
# Start only the database
docker compose up -d db

# Run backend with DB connection
cd src
DB_HOST=localhost DB_PORT=5432 DB_USER=postgres DB_PASSWORD=postgres DB_NAME=project go run ./cmd/main.go
```

When `DB_HOST` is set, the backend automatically:
1. Connects to PostgreSQL using `pgx/v5`
2. Runs schema migrations from `src/db/migrations/`
3. Wraps the PostgreSQL store with an in-memory fallback for stores not yet migrated

---

## Project Structure

```
AI-Software-Factory/
├── frontend/              # Next.js application
│   └── src/
│       ├── app/           # App Router pages
│       │   └── projects/  # Project management UI
│       ├── components/    # React components
│       │   ├── kanban/    # Kanban board (drag-and-drop)
│       │   ├── layout/    # Layout components
│       │   ├── ui/        # Primitive UI components
│       │   └── shared/    # Shared components
│       ├── hooks/         # Custom React hooks
│       └── lib/           # API client, React Query hooks, types
├── src/                   # Go backend
│   ├── cmd/               # Entry point
│   ├── internal/
│   │   ├── handler/       # HTTP handlers
│   │   ├── model/         # Domain models
│   │   ├── router/        # Route definitions
│   │   ├── service/       # Business logic
│   │   ├── store/         # Data store (memory + postgres)
│   │   └── middleware/    # Auth, CORS, rate limiting
│   └── db/                # Database migrations
├── docker-compose.yml     # Docker Compose configuration
└── docs/                  # Documentation
```

---

## Common Tasks

### Create a project

```bash
curl -X POST http://localhost:8080/v1/projects \
  -H "Content-Type: application/json" \
  -d '{"name": "My Project", "description": "A test project"}'
```

### List projects

```bash
curl http://localhost:8080/v1/projects
```

### Create a task

```bash
curl -X POST http://localhost:8080/v1/projects/{projectId}/tasks \
  -H "Content-Type: application/json" \
  -d '{"title": "Implement login", "priority": "high"}'
```

### Move a task on the Kanban board

```bash
curl -X PATCH http://localhost:8080/v1/tasks/{taskId}/status \
  -H "Content-Type: application/json" \
  -d '{"status": "in_progress"}'
```

---

## Troubleshooting

| Symptom | Likely Cause | Fix |
|---------|-------------|-----|
| Container exits immediately | Missing `.env` | Run `cp .env.example .env` |
| `port is already allocated` | Port conflict | Change port in `.env` |
| API health fails | API not started | Check `docker compose ps` |
| Tasks not persisting | No `DB_HOST` set | Using in-memory store; data resets on restart |
| Kanban drag not working | JS error in browser | Check console; ensure `@dnd-kit` is installed |

---

## Windows Notes

- Use Git Bash for shell scripts
- Enable WSL 2 backend in Docker Desktop
- Share `C:\Users` drive in Docker Desktop settings
