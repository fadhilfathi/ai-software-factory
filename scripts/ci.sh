#!/usr/bin/env bash
# =============================================================================
# ci.sh — Run full CI pipeline locally
# =============================================================================
# Mirrors what GitHub Actions CI does, so you can reproduce failures
# before pushing.
#
# Usage:  ./scripts/ci.sh
# =============================================================================
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║       Local CI Pipeline — AI Software Factory               ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

mkdir -p "$ROOT/build"

# ---- Phase 1: Lint ----
echo "────────────────────────────────────────────────────────────────"
echo " Phase 1: Lint"
echo "────────────────────────────────────────────────────────────────"

if command -v go &>/dev/null; then
  echo "→ Go vet..."
  (cd "$ROOT/src" && go vet ./...)
  echo "→ Go fmt check..."
  UNFORMATTED=$(cd "$ROOT/src" && go fmt ./...)
  if [ -n "$UNFORMATTED" ]; then
    echo "⚠️  Unformatted files:"
    echo "$UNFORMATTED"
  fi
else
  echo "⚠️  Go not installed — skipping Go lint"
fi

if command -v node &>/dev/null; then
  echo "→ Frontend lint..."
  (cd "$ROOT/frontend" && npm run lint 2>/dev/null || echo "⚠️  Lint found issues")
else
  echo "⚠️  Node.js not installed — skipping frontend lint"
fi

# ---- Phase 2: Test ----
echo ""
echo "────────────────────────────────────────────────────────────────"
echo " Phase 2: Test"
echo "────────────────────────────────────────────────────────────────"

bash "$ROOT/scripts/test.sh"

# ---- Phase 3: Build ----
echo ""
echo "────────────────────────────────────────────────────────────────"
echo " Phase 3: Build"
echo "────────────────────────────────────────────────────────────────"

bash "$ROOT/scripts/build.sh"

# ---- Phase 4: Docker (if available) ----
echo ""
echo "────────────────────────────────────────────────────────────────"
echo " Phase 4: Docker compose validation"
echo "────────────────────────────────────────────────────────────────"

if command -v docker &>/dev/null; then
  echo "→ Validating docker-compose.yml..."
  docker compose config --quiet && echo "✅ docker-compose.yml is valid"
else
  echo "⚠️  Docker not installed — skipping compose validation"
fi

echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║       ✅ CI pipeline complete                                ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""
echo "Build artifacts: $ROOT/build/"
ls -lh "$ROOT/build/" 2>/dev/null || echo "(none)"
