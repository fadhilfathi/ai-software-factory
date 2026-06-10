# Monitoring Configuration

> **AI Software Factory** — health checks, metrics, logging, and observability setup.

---

## Table of Contents

- [Health Check Architecture](#health-check-architecture)
- [Layer 1: Docker Container Health Checks](#layer-1-docker-container-health-checks)
- [Layer 2: Application Health Endpoints](#layer-2-application-health-endpoints)
- [Layer 3: Health Check Script](#layer-3-health-check-script)
- [Layer 4: CI/CD Pipeline Monitoring](#layer-4-cicd-pipeline-monitoring)
- [Logging](#logging)
- [Docker Monitoring Commands](#docker-monitoring-commands)
- [Prometheus / Grafana Setup (Future)](#prometheus--grafana-setup-future)
- [Alerting](#alerting)

---

## Health Check Architecture

The system implements **4 layers of health monitoring**:

```
┌──────────────────────────────────────────────────────────────────┐
│  Layer 4: CI/CD Pipeline Monitoring                               │
│  ───────────────────────────────                                  │
│  GitHub Actions deploy.yml validates health post-deploy,          │
│  auto-rollbacks on failure.                                       │
├──────────────────────────────────────────────────────────────────┤
│  Layer 3: Health Check Script                                     │
│  ───────────────────────────────                                  │
│  scripts/healthcheck.sh — CLI tool for interactive or             │
│  cron-based monitoring. Supports JSON output for external tools.  │
├──────────────────────────────────────────────────────────────────┤
│  Layer 2: Application Health Endpoints                            │
│  ───────────────────────────────                                  │
│  API: GET /v1/healthz → 200 OK (also returns DB connectivity)     │
│  Frontend: HTTP 200 on /                                          │
├──────────────────────────────────────────────────────────────────┤
│  Layer 1: Docker Container Health Checks                          │
│  ───────────────────────────────                                  │
│  Built into Dockerfiles & docker-compose.yml.                     │
│  Docker restarts unhealthy containers automatically.              │
└──────────────────────────────────────────────────────────────────┘
```

---

## Layer 1: Docker Container Health Checks

Each service defines a Docker `HEALTHCHECK` instruction in its Dockerfile and/or
`healthcheck` block in `docker-compose.yml`.

### API Health Check

**Dockerfile** (`src/Dockerfile`):
```
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/v1/healthz || exit 1
```

**docker-compose.yml** override:
```
healthcheck:
  test:        ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/v1/healthz"]
  interval:    30s
  timeout:     5s
  retries:     3
  start_period: 10s
```

### Frontend Health Check

**Dockerfile** (`frontend/Dockerfile`):
```
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3000/ || exit 1
```

**docker-compose.yml** override:
```
healthcheck:
  test:        ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:3000/"]
  interval:    30s
  timeout:     5s
  retries:     3
  start_period: 15s
```

### Database Health Check

Defined only in `docker-compose.yml`:
```
healthcheck:
  test:        ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-postgres}"]
  interval:    10s
  timeout:     5s
  retries:     5
  start_period: 15s
```

### Health Check States

| State | Meaning | Docker Action |
|-------|---------|---------------|
| `healthy` | Container is responding | Normal operation |
| `starting` | Within `start_period`; health check hasn't passed yet | Waiting |
| `unhealthy` | Health check failed `retries` times | Container killed by restart policy, then recreated |

The `compose.yml` `depends_on` with `condition: service_healthy` ensures the stack
starts in the correct order: **db → api → frontend**.

---

## Layer 2: Application Health Endpoints

### API: `GET /v1/healthz`

Returns HTTP `200 OK` when the API is running and connected to the database.

```
$ curl -v http://localhost:8080/v1/healthz

* Expected response:
< HTTP/1.1 200 OK
< Content-Type: application/json
<
{"status":"ok","timestamp":"2026-06-10T13:00:00Z","db":"connected"}
```

### Frontend: `GET /`

Returns HTTP `200` when the Next.js server is serving pages.

```
$ curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/
200
```

---

## Layer 3: Health Check Script

The `scripts/healthcheck.sh` script provides a unified CLI interface to all health checks.

### Usage

```bash
# Check all services (default)
./scripts/healthcheck.sh

# Check specific service
./scripts/healthcheck.sh api
./scripts/healthcheck.sh frontend
./scripts/healthcheck.sh docker

# JSON output (for monitoring tools, cron jobs)
./scripts/healthcheck.sh --json

# Continuous monitoring (every 5 seconds)
./scripts/healthcheck.sh --watch

# Custom interval
./scripts/healthcheck.sh --watch 10
```

### JSON Output

```json
{
  "timestamp": "2026-06-10T13:00:00Z",
  "services": {
    "api":      {"url": "http://localhost:8080/v1/healthz", "status": true, "http_code": "200"},
    "frontend": {"url": "http://localhost:3000",           "status": true, "http_code": "200"}
  },
  "healthy": true
}
```

### Cron Job Integration

```bash
# Check every 5 minutes and log failures
*/5 * * * * cd /opt/ai-software-factory && ./scripts/healthcheck.sh --json >> /var/log/healthcheck.json 2>&1

# Alert on failure (requires mail or notification tool)
*/5 * * * * cd /opt/ai-software-factory && ./scripts/healthcheck.sh --json | grep -q '"healthy": false' && curl -s -X POST https://hooks.example.com/alert -d '{"service":"ai-software-factory","status":"unhealthy"}'
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `API_URL` | `http://localhost:8080/v1/healthz` | API health endpoint URL |
| `FRONTEND_URL` | `http://localhost:3000` | Frontend health check URL |

Override these when checking from outside the Docker network:

```bash
API_URL=https://api.example.com/v1/healthz FRONTEND_URL=https://app.example.com ./scripts/healthcheck.sh
```

---

## Layer 4: CI/CD Pipeline Monitoring

### Deploy Validation

The `deploy.yml` workflow runs a health check after every deployment:

```yaml
- name: Health check after deploy
  run: |
    for i in $(seq 1 30); do
      # Check each service's Docker health status
      for svc in api frontend; do
        status=$(docker inspect --format='{{.State.Health.Status}}' "ai-software-factory-$svc-1")
        # All must be "healthy"
      done
    done
```

- **Timeout:** 60 seconds (30 retries × 2s interval)
- **On failure:** Automatic rollback (see [Rollback](#rollback-procedures) in deployment-guide.md)
- **On success:** Service table printed to workflow log

### CI E2E Smoke Test

The `ci.yml` workflow runs a full-stack smoke test on every push/PR:

```yaml
- name: Smoke test endpoints
  run: |
    curl -sf http://localhost:8080/v1/healthz
    curl -sf -o /dev/null -w "HTTP %{http_code}" http://localhost:3000/
```

Logs from failed smoke tests are dumped automatically via the `dump logs on failure` step.

### GitHub Actions Monitoring

| What to Monitor | How |
|----------------|-----|
| Workflow status | GitHub → Actions tab → workflow run history |
| Failure rate | GitHub → Insights → Workflow / CI → Add `status: failure` filter |
| Deploy frequency | GitHub → Actions → Deploy workflow → run list |
| Rollback events | Check deploy workflow runs where the health check step failed |

---

## Logging

### Docker Container Logs

```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f api
docker compose logs -f frontend
docker compose logs -f db

# Last N lines
docker compose logs --tail=100 api

# Timestamps
docker compose logs -t api
```

### Log Levels

The API supports configurable log levels via the `LOG_LEVEL` environment variable:

| Level | Use Case |
|-------|----------|
| `debug` | Development, tracing issues |
| `info` | Normal operation (default) |
| `warn` | Production, reduce noise |
| `error` | Errors only |

```bash
# Temporarily increase verbosity without restarting
LOG_LEVEL=debug docker compose up -d api
```

### Log File Locations (when running natively)

```
src/           — Go API logs → stdout/stderr (capture via docker logs or pipe to file)
frontend/      — Next.js logs → stdout/stderr
db/            — PostgreSQL logs → container stdout
```

For persistent log storage, configure Docker's logging driver:

```bash
# In docker-compose.yml, add to each service:
logging:
  driver: "json-file"
  options:
    max-size: "10m"
    max-file: "3"
```

---

## Docker Monitoring Commands

### Container Status

```bash
# Quick overview
docker compose ps

# Detailed container info (IP, ports, mounts)
docker inspect ai-software-factory-api-1

# Resource usage
docker stats --no-stream

# Running containers with health status
docker ps --format 'table {{.Names}}\t{{.Status}}'
```

### Resource Usage

```bash
# Real-time stats (hit Ctrl+C to stop)
docker stats

# One-shot
docker stats --no-stream

# Specific container
docker stats ai-software-factory-api-1
```

### Disk Usage

```bash
# Docker disk usage
docker system df

# Volume inspection
docker volume ls
docker volume inspect ai-software-factory_pgdata
```

### Network

```bash
# List networks
docker network ls

# Inspect compose network
docker network inspect ai-software-factory_default
```

---

## Prometheus / Grafana Setup (Future)

This section outlines the planned observability stack for production deployments.

### Recommended Stack

| Component | Purpose | How |
|-----------|---------|-----|
| **Prometheus** | Metrics collection | Scrape `/metrics` endpoints from API and node exporter |
| **Grafana** | Visualization | Dashboards for request rate, latency, error rate, saturation |
| **Node Exporter** | Host-level metrics | CPU, memory, disk, network per VM |
| **cAdvisor** | Container-level metrics | Per-container resource usage |
| **Loki** | Log aggregation | Centralized log search from all containers |

### Adding Metrics Endpoint to API

The Go API should expose a `/metrics` endpoint (Prometheus format):

```go
import "github.com/prometheus/client_golang/prometheus/promhttp"

// In main.go or router setup
router.Handle("GET /metrics", promhttp.Handler())
```

### Docker Compose Monitoring Stack

Add to `docker-compose.yml` (separate profile for production monitoring):

```yaml
services:
  prometheus:
    image: prom/prometheus:latest
    profiles: ["monitoring"]
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"

  grafana:
    image: grafana/grafana:latest
    profiles: ["monitoring"]
    ports:
      - "3001:3000"
    depends_on:
      - prometheus

  node-exporter:
    image: prom/node-exporter:latest
    profiles: ["monitoring"]
    network_mode: "host"

  cadvisor:
    image: gcr.io/cadvisor/cadvisor:latest
    profiles: ["monitoring"]
    volumes:
      - /:/rootfs:ro
      - /var/run:/var/run:ro
      - /sys:/sys:ro
      - /var/lib/docker/:/var/lib/docker:ro
    ports:
      - "8081:8080"
```

### Prometheus Scrape Config

```yaml
# monitoring/prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'api'
    static_configs:
      - targets: ['api:8080']

  - job_name: 'node'
    static_configs:
      - targets: ['localhost:9100']

  - job_name: 'cadvisor'
    static_configs:
      - targets: ['localhost:8081']
```

---

## Alerting

### Current Capabilities

| Alert Method | What It Covers | How |
|-------------|---------------|-----|
| **Docker restart policy** | Container crash | `restart: unless-stopped` in compose.yml |
| **CI/CD health check** | Post-deploy failure | Auto-rollback in deploy.yml |
| **GitHub notifications** | Workflow failure | GitHub → Settings → Notifications → Actions |
| **Manual monitoring** | Ad-hoc health checks | `./scripts/healthcheck.sh --watch` |

### Future Alerting (with monitoring stack)

| Alert | Severity | Condition | Channel |
|-------|----------|-----------|---------|
| Service down | Critical | Health check fails for 60s | Email, Slack |
| High latency | Warning | p95 > 500ms for 5 min | Slack |
| Disk space low | Warning | Usage > 80% | Email |
| Memory pressure | Warning | RSS > 90% for 5 min | Slack |
| Deploy failure | Critical | Deploy workflow fails | GitHub + Slack |
| Certificate expiry | Warning | < 30 days | Email |
