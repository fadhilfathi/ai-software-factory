#!/usr/bin/env bash
# =============================================================================
# deploy.sh — Deploy the full stack via docker-compose
# =============================================================================
# Usage:
#   ./scripts/deploy.sh              # build + up (production tag)
#   ./scripts/deploy.sh --build      # force rebuild before up
#   ./scripts/deploy.sh --pull       # pull latest images
#   ./scripts/deploy.sh down         # tear down the stack
#   ./scripts/deploy.sh logs [svc]   # tail logs for a service
#   ./scripts/deploy.sh restart [svc] # restart a service
#
# Prerequisites:
#   - Docker + docker-compose-plugin installed
#   - .env file at project root (copy from .env.example)
# =============================================================================
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# Colour helpers
info()  { echo -e "\033[0;32m==>\033[0m $*"; }
warn()  { echo -e "\033[0;33m==>\033[0m $*"; }
err()   { echo -e "\033[0;31m==>\033[0m $*" >&2; }

# --- Preflight ---
check_prereqs() {
  if ! command -v docker &>/dev/null; then
    err "Docker not found — install from https://docs.docker.com/get-docker/"
    exit 1
  fi
  docker compose version &>/dev/null || {
    err "docker-compose-plugin not installed"
    err "Install: https://docs.docker.com/compose/install/"
    exit 1
  }
  if [ ! -f .env ]; then
    warn ".env file not found — copying from .env.example"
    cp .env.example .env
    warn "⚠️  Edit .env with your secrets before deploying to production"
  fi
}

# --- Health check ---
wait_healthy() {
  local timeout="${1:-60}"
  local interval=3

  info "Waiting up to ${timeout}s for all services to become healthy..."
  local elapsed=0
  while [ $elapsed -lt $timeout ]; do
    local all_healthy=true
    for svc in api frontend redis db; do
      local id
      id=$(docker compose ps -q "$svc" 2>/dev/null || true)
      if [ -z "$id" ]; then
        all_healthy=false
        break
      fi
      local status
      status=$(docker inspect --format='{{.State.Health.Status}}' "$id" 2>/dev/null || echo "starting")
      if [ "$status" != "healthy" ]; then
        all_healthy=false
        break
      fi
    done
    if $all_healthy; then
      info "✅ All services healthy"
      docker ps --format 'table {{.Names}}\t{{.Status}}'
      return 0
    fi
    sleep $interval
    elapsed=$((elapsed + interval))
  done

  err "⏱ Timeout — not all services became healthy within ${timeout}s"
  docker ps --format 'table {{.Names}}\t{{.Status}}'
  return 1
}

# --- Commands ---
cmd_up() {
  local build_flag=""
  if [ "${1:-}" = "--build" ]; then
    build_flag="--build"
    shift
  fi

  check_prereqs
  info "Starting stack..."$'\n'
  docker compose up -d $build_flag --remove-orphans
  wait_healthy
}

cmd_down() {
  info "Tearing down stack..."
  docker compose down -v 2>/dev/null || docker compose down
  info "✅ Stack stopped"
}

cmd_logs() {
  local svc="${1:-}"
  if [ -n "$svc" ]; then
    docker compose logs --tail=100 -f "$svc"
  else
    docker compose logs --tail=50 -f
  fi
}

cmd_restart() {
  local svc="${1:-}"
  if [ -n "$svc" ]; then
    info "Restarting $svc..."
    docker compose restart "$svc"
    wait_healthy 30
  else
    info "Restarting all services..."
    docker compose restart
    wait_healthy 60
  fi
}

cmd_pull() {
  check_prereqs
  info "Pulling latest images..."
  docker compose pull
  info "✅ Images up to date"
}

# --- Main ---
case "${1:-up}" in
  up)
    shift 2>/dev/null || true
    cmd_up "$@"
    ;;
  down)
    cmd_down
    ;;
  logs)
    shift; cmd_logs "$@"
    ;;
  restart)
    shift; cmd_restart "$@"
    ;;
  pull)
    cmd_pull
    ;;
  --build)
    cmd_up --build
    ;;
  --pull)
    cmd_pull
    cmd_up
    ;;
  *)
    echo "Usage: $0 [up|down|logs|restart|pull|--build]"
    echo ""
    echo "  up              Start stack (default)"
    echo "  up --build      Force rebuild then start"
    echo "  down            Tear down stack"
    echo "  logs [svc]      Tail logs"
    echo "  restart [svc]   Restart service(s)"
    echo "  pull            Pull latest images"
    echo "  --build         Alias for 'up --build'"
    exit 1
    ;;
esac
