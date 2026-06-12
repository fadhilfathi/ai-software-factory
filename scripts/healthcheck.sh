#!/usr/bin/env bash
# =============================================================================
# healthcheck.sh — Check health of all services
# =============================================================================
# Usage:
#   ./scripts/healthcheck.sh          # all services
#   ./scripts/healthcheck.sh api      # specific service
#   ./scripts/healthcheck.sh --json   # JSON output for monitoring tools
#   ./scripts/healthcheck.sh --watch  # continuous monitoring (every 5s)
# =============================================================================
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

info()  { echo -e "\033[0;32m$*\033[0m"; }
warn()  { echo -e "\033[0;33m$*\033[0m"; }
err()   { echo -e "\033[0;31m$*\033[0m" >&2; }

check_api() {
  local url="${API_URL:-http://localhost:8080/v1/healthz}"
  local status_code
  status_code=$(curl -sf -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000")
  if [ "$status_code" = "200" ]; then
    info "  ✅ API ($url) — HTTP $status_code"
    return 0
  else
    err "  ❌ API ($url) — HTTP $status_code"
    return 1
  fi
}

check_frontend() {
  local url="${FRONTEND_URL:-http://localhost:3000}"
  local status_code
  status_code=$(curl -sf -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000")
  if [ "$status_code" = "200" ]; then
    info "  ✅ Frontend ($url) — HTTP $status_code"
    return 0
  else
    err "  ❌ Frontend ($url) — HTTP $status_code"
    return 1
  fi
}

check_docker() {
  if ! command -v docker &>/dev/null; then
    warn "  ⚠️  Docker not available"
    return 0
  fi

  for svc in api frontend redis db; do
    local id
    id=$(docker compose ps -q "$svc" 2>/dev/null || true)
    if [ -z "$id" ]; then
      warn "  ⚠️  $svc — not running"
      continue
    fi
    local status
    status=$(docker inspect --format='{{.State.Health.Status}}' "$id" 2>/dev/null || echo "unknown")
    case "$status" in
      healthy)   info  "  ✅ $svc — $status" ;;
      starting)  warn  "  ⏳ $svc — $status" ;;
      *)         err   "  ❌ $svc — $status" ;;
    esac
  done
}

output_json() {
  local api_status frontend_status
  api_status=$(curl -sf -o /dev/null -w "%{http_code}" http://localhost:8080/v1/healthz 2>/dev/null || echo "unreachable")
  frontend_status=$(curl -sf -o /dev/null -w "%{http_code}" http://localhost:3000 2>/dev/null || echo "unreachable")

  cat <<JSON
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "services": {
    "api":      {"url": "http://localhost:8080/v1/healthz", "status": $([ "$api_status" = "200" ] && echo true || echo false), "http_code": "$api_status"},
    "frontend": {"url": "http://localhost:3000",           "status": $([ "$frontend_status" = "200" ] && echo true || echo false), "http_code": "$frontend_status"}
  },
  "healthy": $([ "$api_status" = "200" ] && [ "$frontend_status" = "200" ] && echo true || echo false)
}
JSON
}

# --- Main ---
case "${1:-all}" in
  api)
    check_api
    ;;
  frontend)
    check_frontend
    ;;
  docker)
    check_docker
    ;;
  all)
    echo "=== Health Check: $(date -u +%Y-%m-%dT%H:%M:%SZ) ==="
    check_api
    check_frontend
    check_docker
    echo ""
    ;;
  --json)
    output_json
    ;;
  --watch)
    shift
    interval="${1:-5}"
    echo "Continuous monitoring (every ${interval}s) — Ctrl+C to stop"
    while true; do
      clear 2>/dev/null || true
      echo "=== Health Check: $(date -u +%Y-%m-%dT%H:%M:%SZ) ==="
      check_api
      check_frontend
      check_docker
      sleep "$interval"
    done
    ;;
  *)
    echo "Usage: $0 [all|api|frontend|docker|--json|--watch]"
    echo ""
    echo "  all         Check all services (default)"
    echo "  api         Check API health endpoint only"
    echo "  frontend    Check frontend only"
    echo "  docker      Check Docker container health only"
    echo "  --json      JSON output for monitoring tools"
    echo "  --watch [N] Continuous check every N seconds (default 5)"
    exit 1
    ;;
esac
