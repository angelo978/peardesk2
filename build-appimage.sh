#!/usr/bin/env bash
# Run this on any Ubuntu/Debian machine or inside Docker:
#   docker run --rm -v "$(pwd)":/src -w /src ubuntu:22.04 bash build-appimage.sh
set -euo pipefail

echo "=== PearDesk AppImage builder ==="

# ── 1. System dependencies ────────────────────────────────────────────────────
apt-get update -qq
apt-get install -y --no-install-recommends \
  libx11-dev libxrandr-dev libxcursor-dev libxi-dev \
  libxinerama-dev libgl1-mesa-dev libgles2-mesa-dev \
  libfontconfig1-dev libfreetype6-dev libxtst-dev \
  pkg-config gcc wget curl ca-certificates

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

# ── 3. Build binary ───────────────────────────────────────────────────────────
echo "Building PearDesk..."
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
  go build -ldflags="-s -w" -o /tmp/peardesk-bin ./cmd/peardesk
echo "Binary built: $(du -sh /tmp/peardesk-bin)"

# ── 4. Download appimagetool ──────────────────────────────────────────────────
echo "Downloading appimagetool..."
wget -q https://github.com/AppImage/AppImageKit/releases/download/continuous/appimagetool-x86_64.AppImage \
  -O /tmp/appimagetool
chmod +x /tmp/appimagetool

# ── 5. Assemble AppDir ────────────────────────────────────────────────────────
rm -rf /tmp/AppDir
mkdir -p /tmp/AppDir/usr/bin
mkdir -p /tmp/AppDir/usr/share/icons/hicolor/256x256/apps

cp /tmp/peardesk-bin /tmp/AppDir/usr/bin/peardesk
cp assets/icon.png   /tmp/AppDir/peardesk.png
cp assets/icon.png   /tmp/AppDir/usr/share/icons/hicolor/256x256/apps/peardesk.png

cat > /tmp/AppDir/peardesk.desktop << 'EOF'
[Desktop Entry]
Name=PearDesk
Exec=peardesk
Icon=peardesk
Type=Application
Categories=Network;RemoteAccess;
EOF

cat > /tmp/AppDir/AppRun << 'EOF'
#!/bin/sh
exec "${APPDIR}/usr/bin/peardesk" "$@"
EOF
chmod +x /tmp/AppDir/AppRun

# ── 6. Package AppImage ───────────────────────────────────────────────────────
mkdir -p dist
ARCH=x86_64 APPIMAGE_EXTRACT_AND_RUN=1 \
  /tmp/appimagetool /tmp/AppDir dist/PearDesk-linux-x86_64.AppImage

echo ""
echo "✓ Done: dist/PearDesk-linux-x86_64.AppImage"
ls -lh dist/PearDesk-linux-x86_64.AppImage
