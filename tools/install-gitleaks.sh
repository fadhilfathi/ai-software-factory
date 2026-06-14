#!/usr/bin/env bash
# Install gitleaks locally for pre-commit / pre-push scanning.
# CI uses the official gitleaks-action Docker image (see .github/workflows/secret-scan.yml).
#
# Usage:
#   bash tools/install-gitleaks.sh
#
# Idempotent: re-running replaces the binary in place.

set -euo pipefail

VERSION="8.18.4"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
INSTALL_DIR="$ROOT/tools"
mkdir -p "$INSTALL_DIR"

case "$(uname -s)" in
  Linux*)   PLATFORM="linux_x64" ;;
  Darwin*)  PLATFORM="darwin_x64" ;;
  MINGW*|MSYS*|CYGWIN*) PLATFORM="windows_x64"; EXT=".exe" ;;
  *) echo "Unsupported platform: $(uname -s)" >&2; exit 1 ;;
esac

URL="https://github.com/gitleaks/gitleaks/releases/download/v${VERSION}/gitleaks_${VERSION}_${PLATFORM}.zip"
TMP_ZIP="$INSTALL_DIR/gitleaks.zip"

echo "Downloading gitleaks ${VERSION} for ${PLATFORM}..."
powershell -NoProfile -Command "Invoke-WebRequest -Uri '${URL}' -OutFile '${TMP_ZIP}' -UseBasicParsing" \
  || curl -fL "$URL" -o "$TMP_ZIP" \
  || wget -q "$URL" -O "$TMP_ZIP"

echo "Extracting..."
if command -v powershell >/dev/null 2>&1; then
  powershell -NoProfile -Command "Expand-Archive -Path '${TMP_ZIP}' -DestinationPath '${INSTALL_DIR}' -Force"
else
  unzip -o "$TMP_ZIP" -d "$INSTALL_DIR"
fi

# Strip the release's bundled LICENSE and README so tools/ contains
# only the install script and the binary. The release's LICENSE is the
# same as our repo's third-party-licenses/ entries; no need to track
# a copy in tools/.
rm -f "$TMP_ZIP" "$INSTALL_DIR/LICENSE" "$INSTALL_DIR/README.md"
chmod +x "$INSTALL_DIR/gitleaks${EXT:-}"
echo "Installed: $("$INSTALL_DIR/gitleaks${EXT:-}" version)"
