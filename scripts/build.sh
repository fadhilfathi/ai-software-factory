#!/usr/bin/env bash
# =============================================================================
# build.sh — Build Go API + Next.js frontend locally
# =============================================================================
# Usage:  ./scripts/build.sh [api|frontend|all]
#         ./scripts/build.sh            # default: all
#
# Output:
#   api/       — compiled Go binary
#   frontend/.next/ — stand alone Next.js build
# =============================================================================
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

build_api() {
  echo "==> Building Go API..."
  cd "$ROOT/src"

  if ! command -v go &>/dev/null; then
    echo "  ⚠️  Go not found — skipping API build"
    echo "  Install Go from https://go.dev/dl/"
    return 1
  fi

  CGO_ENABLED=0 go build \
    -ldflags="-s -w -extldflags=-static" \
    -o "$ROOT/build/api" ./cmd/main.go

  echo "  ✅ API binary: build/api ($(ls -lh "$ROOT/build/api" | awk '{print $5}'))"
}

build_frontend() {
  echo "==> Building Frontend..."
  cd "$ROOT/frontend"

  if ! command -v node &>/dev/null; then
    echo "  ⚠️  Node.js not found — skipping frontend build"
    return 1
  fi

  if [ ! -d node_modules ]; then
    echo "  → npm ci..."
    npm ci --loglevel=warn
  fi

  npm run build
  echo "  ✅ Frontend built: frontend/.next/"
}

build_docker() {
  echo "==> Building Docker images..."
  if ! command -v docker &>/dev/null; then
    echo "  ⚠️  Docker not found — skipping Docker build"
    return 1
  fi

  docker compose build --pull
  echo "  ✅ Docker images built"
}

# --- Main ---
TARGET="${1:-all}"

case "$TARGET" in
  api)
    build_api
    ;;
  frontend)
    build_frontend
    ;;
  docker)
    build_docker
    ;;
  all)
    build_api
    build_frontend
    build_docker
    ;;
  *)
    echo "Usage: $0 [api|frontend|docker|all]"
    exit 1
    ;;
esac

echo ""
echo "✅ Build complete: $TARGET"
