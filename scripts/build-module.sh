#!/bin/bash
# Build CasaDrop ZimaOS native module (.raw sysext package)
#
# Usage: ./scripts/build-module.sh
#
# Requirements: go 1.25+, mksquashfs (pure-Go build — no CGO/gcc needed since v2.2)
#
# Output: casadrop.raw (squashfs image for ZimaOS module installation)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BUILD_DIR="$(mktemp -d)"
MODULE_NAME="casadrop"
MODULE_PORT=8085

cleanup() {
    rm -rf "$BUILD_DIR"
}
trap cleanup EXIT

echo "=== Building CasaDrop ZimaOS Module ==="
echo "Project: $PROJECT_DIR"
echo "Build dir: $BUILD_DIR"

# Check dependencies
for cmd in go mksquashfs; do
    if ! command -v "$cmd" &>/dev/null; then
        echo "ERROR: $cmd is required but not found"
        exit 1
    fi
done

# Step 1: Build a fully static, pure-Go binary (no CGO — modernc.org/sqlite).
echo ""
echo "--- Building binary ---"
cd "$PROJECT_DIR"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -trimpath -o "$BUILD_DIR/casadrop" ./cmd/server
echo "Binary: $(file "$BUILD_DIR/casadrop")"

# Step 2: Create sysext directory structure
echo ""
echo "--- Creating module structure ---"
SYSEXT="$BUILD_DIR/sysext"

# Binary
install -Dm755 "$BUILD_DIR/casadrop" "$SYSEXT/usr/bin/casadrop"

# Extension release (sysext identity)
mkdir -p "$SYSEXT/usr/lib/extension-release.d"
echo "ID=_any" > "$SYSEXT/usr/lib/extension-release.d/extension-release.casadrop"

# Systemd service
mkdir -p "$SYSEXT/usr/lib/systemd/system"
cat > "$SYSEXT/usr/lib/systemd/system/casadrop.service" <<'EOF'
[Unit]
Description=CasaDrop - Self-hosted File Sharing
After=casaos-gateway.service
After=casaos-message-bus.service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStartPre=/usr/bin/mkdir -p /DATA/AppData/casadrop/data /DATA/AppData/casadrop/uploads /DATA/AppData/casadrop/thumbnails
ExecStart=/usr/bin/casadrop
Environment=PORT=8085
Environment=DATA_DIR=/DATA/AppData/casadrop
Environment=TEMPLATES_DIR=/usr/share/casaos/www/modules/casadrop/templates
Environment=STATIC_DIR=/usr/share/casaos/www/modules/casadrop/static
Environment=SHARE_ALLOWED_PATHS=/DATA,/media,/home
Environment=TZ=Europe/Berlin
StandardOutput=journal
StandardError=journal
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# CasaOS module config
mkdir -p "$SYSEXT/usr/share/casaos/modules"
cat > "$SYSEXT/usr/share/casaos/modules/casadrop.json" <<'EOF'
{
  "name": "casadrop",
  "ui": {
    "name": "casadrop",
    "title": { "en_us": "CasaDrop", "de_de": "CasaDrop" },
    "prefetch": true,
    "show": true,
    "entry": "/modules/casadrop/index.html",
    "icon": "/modules/casadrop/logo.png",
    "description": "Self-hosted File Sharing",
    "formality": {
      "type": "newtab",
      "props": { "width": "100vh", "height": "100vh", "hasModalCard": true, "animation": "zoom-in" }
    }
  },
  "services": [ { "name": "casadrop" } ]
}
EOF

# Web UI module entry point
MODULE_WWW="$SYSEXT/usr/share/casaos/www/modules/casadrop"
mkdir -p "$MODULE_WWW/static/css" "$MODULE_WWW/static/js" "$MODULE_WWW/templates"

cat > "$MODULE_WWW/index.html" <<EOF
<!DOCTYPE html>
<html>
<head><meta http-equiv="refresh" content="0;url=http://localhost:${MODULE_PORT}"></head>
<body><p>Redirecting to <a href="http://localhost:${MODULE_PORT}">CasaDrop</a>...</p></body>
</html>
EOF

# Copy web assets
cp "$PROJECT_DIR/web/static/css/style.css" "$MODULE_WWW/static/css/"
if [ -f "$PROJECT_DIR/web/static/css/auth.css" ]; then
    cp "$PROJECT_DIR/web/static/css/auth.css" "$MODULE_WWW/static/css/"
fi
cp "$PROJECT_DIR/web/static/js/app.js" "$MODULE_WWW/static/js/"
cp "$PROJECT_DIR/web/templates/"*.html "$MODULE_WWW/templates/"

# Copy logo
if [ -f "$PROJECT_DIR/logo.png" ]; then
    cp "$PROJECT_DIR/logo.png" "$MODULE_WWW/logo.png"
    cp "$PROJECT_DIR/logo.png" "$MODULE_WWW/static/logo.png"
fi

# Step 3: Build squashfs image
echo ""
echo "--- Building squashfs image ---"
OUTPUT="$PROJECT_DIR/casadrop.raw"
mksquashfs "$SYSEXT" "$OUTPUT" -noappend -all-root -comp gzip
echo ""
echo "=== Module built successfully ==="
echo "Output: $OUTPUT ($(du -h "$OUTPUT" | cut -f1))"
echo ""
echo "Install on ZimaOS:"
echo "  1. Copy casadrop.raw to /DATA/Downloads/ on your ZimaOS device"
echo "  2. Install via ZimaOS UI or CLI"
echo "  3. Access at http://<zimaos-ip>:${MODULE_PORT}"
