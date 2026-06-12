#!/usr/bin/env bash
# =============================================================================
# Sprint Quality Gate — local runner
# =============================================================================
# Mirrors .github/workflows/sprint-quality-gate.yml for local pre-push
# verification. Runs the same 14 steps in the same order.
#
# Usage:  ./scripts/quality-gate.sh
# Exit:   0 if all 14 steps pass, 1 otherwise.
#
# Note: the GitHub Actions workflow is the actual merge gate for the sprint;
# this script is a dev convenience for catching failures before pushing.
# =============================================================================

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SRC_DIR="$ROOT/src"
COMPOSE_FILE="$ROOT/docker-compose.yml"
HEALTH_URL="http://localhost:8080/v1/healthz"
GO_VERSION="1.25"
NODE_VERSION="20"

# ---- helpers ----------------------------------------------------------------
step() { printf "\n\033[1;34m▶ %s\033[0m\n" "$*"; }
fail() { printf "\n\033[1;31m::error::%s\033[0m\n" "$*"; exit 1; }
need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    fail "required tool '$1' not on PATH"
  fi
}

# ---- prerequisites ----------------------------------------------------------
step "Prerequisites"
need go
need node
need docker
need curl
need git

# ---- 1. Checkout ------------------------------------------------------------
step "1/14 — Checkout"
git -C "$ROOT" rev-parse HEAD >/dev/null || fail "not inside a git repo"
echo "  HEAD: $(git -C "$ROOT" rev-parse --short HEAD)"

# ---- 2. Set up Go -----------------------------------------------------------
step "2/14 — Set up Go $GO_VERSION"
go version

# ---- 3. Set up Node ---------------------------------------------------------
step "3/14 — Set up Node $NODE_VERSION"
node --version

# ---- 4. Cache Go modules + build cache --------------------------------------
step "4/14 — Cache Go modules + build cache (keyed on go.sum)"
# No-op locally. Document the cache key so devs know what GitHub Actions
# will key on, and so the developer can clear their local cache with the
# same shape if needed.
if command -v sha256sum >/dev/null 2>&1; then
  HASH="$(sha256sum "$SRC_DIR/go.sum" 2>/dev/null | cut -c1-12)"
else
  HASH="nohash"
fi
echo "  (local cache key would be: local-go-$HASH)"

# ---- 5. Go mod download -----------------------------------------------------
step "5/14 — Go mod download"
( cd "$SRC_DIR" && go mod download ) || fail "go mod download failed"

# ---- 6. Go vet --------------------------------------------------------------
step "6/14 — Go vet"
( cd "$SRC_DIR" && go vet ./... ) || fail "go vet failed"

# ---- 7. Go build ------------------------------------------------------------
step "7/14 — Go build"
( cd "$SRC_DIR" && go build ./... ) || fail "go build failed"

# ---- 8. Go unit tests -------------------------------------------------------
step "8/14 — Go unit tests (./internal/...)"
( cd "$SRC_DIR" && go test -count=1 -timeout 5m ./internal/... ) || fail "go test (unit) failed"

# ---- 9. docker compose config ----------------------------------------------
step "9/14 — docker compose config"
( cd "$ROOT" && docker compose -f "$COMPOSE_FILE" config >/dev/null ) || fail "docker compose config failed"

# ---- 10. docker compose up -d ----------------------------------------------
step "10/14 — docker compose up -d"
( cd "$ROOT" && docker compose -f "$COMPOSE_FILE" up -d ) || fail "docker compose up failed"

# ---- 11. Wait for /v1/healthz ----------------------------------------------
step "11/14 — Wait for /v1/healthz (up to 120s)"
HEALTHY=0
for i in $(seq 1 60); do
  if curl -fsS "$HEALTH_URL" >/dev/null 2>&1; then
    echo "  Stack is healthy after $i attempt(s)"
    HEALTHY=1
    break
  fi
  echo "  attempt $i/60: not ready, sleeping 2s..."
  sleep 2
done
if [ "$HEALTHY" -ne 1 ]; then
  echo "::error::Stack did not become healthy within 120s"
  echo "=== docker compose ps ==="
  ( cd "$ROOT" && docker compose -f "$COMPOSE_FILE" ps ) || true
  echo "=== docker compose logs (tail 100) ==="
  ( cd "$ROOT" && docker compose -f "$COMPOSE_FILE" logs --tail=100 ) || true
  fail "stack not healthy"
fi

# ---- 12. curl /v1/healthz ---------------------------------------------------
step "12/14 — curl /v1/healthz"
curl -fsS "$HEALTH_URL" || fail "curl /v1/healthz failed"

# ---- 13. Go integration tests -----------------------------------------------
step "13/14 — Go integration tests (./internal/integration/...)"
if [ -d "$SRC_DIR/internal/integration" ]; then
  ( cd "$SRC_DIR" && go test -count=1 -timeout 10m ./internal/integration/... ) || fail "go test (integration) failed"
else
  echo "  ::notice::No internal/integration directory yet — skipping (will be populated by TASK-411)"
fi

# ---- 14. docker compose down -v --------------------------------------------
step "14/14 — docker compose down -v"
( cd "$ROOT" && docker compose -f "$COMPOSE_FILE" down -v )

printf "\n\033[1;32m✅ All 14 gate steps passed.\033[0m\n"
