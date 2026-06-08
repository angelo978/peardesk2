#!/usr/bin/env bash
# Cross-compila PearDesk.exe da Linux/macOS con mingw-w64.
# Su Ubuntu/Debian:  sudo apt-get install gcc-mingw-w64-x86-64
# Su macOS:          brew install mingw-w64
# Con Docker:
#   docker run --rm -v "$(pwd)":/src -w /src ubuntu:22.04 bash build-windows.sh
set -euo pipefail

echo "=== PearDesk Windows EXE builder ==="

# ── 1. System dependencies ────────────────────────────────────────────────────
if command -v apt-get &>/dev/null; then
  apt-get update -qq
  apt-get install -y --no-install-recommends \
    gcc-mingw-w64-x86-64 pkg-config wget ca-certificates
fi

# ── 2. Install Go ─────────────────────────────────────────────────────────────
GO_VERSION="1.22.5"
if ! command -v go &>/dev/null || [[ "$(go version)" != *"go${GO_VERSION}"* ]]; then
  echo "Installing Go ${GO_VERSION}..."
  wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -O /tmp/go.tar.gz
  rm -rf /usr/local/go
  tar -C /usr/local -xzf /tmp/go.tar.gz
  rm /tmp/go.tar.gz
fi
export PATH="/usr/local/go/bin:$PATH"
go version

# ── 3. Build ──────────────────────────────────────────────────────────────────
echo "Building PearDesk.exe..."
mkdir -p dist
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
  CC=x86_64-w64-mingw32-gcc \
  go build -ldflags="-s -w -H=windowsgui" \
  -o dist/PearDesk.exe ./cmd/peardesk

echo ""
echo "✓ Done: dist/PearDesk.exe"
ls -lh dist/PearDesk.exe
