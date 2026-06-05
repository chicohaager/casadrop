#!/bin/bash
# CasaDrop mit Tailscale Funnel starten
# Verwendung: ./start-with-tailscale.sh

set -e

# Konfiguration
TAILSCALE_AUTHKEY="${TAILSCALE_AUTHKEY:-}"
TAILSCALE_HOSTNAME="${TAILSCALE_HOSTNAME:-casadrop}"
WEBUI_PORT="${WEBUI_PORT:-8085}"

echo "=== CasaDrop mit Tailscale Funnel ==="
echo ""

if [ -z "$TAILSCALE_AUTHKEY" ]; then
    echo "ERROR: TAILSCALE_AUTHKEY is not set."
    echo "Create one at https://login.tailscale.com/admin/settings/keys and run:"
    echo "  TAILSCALE_AUTHKEY=tskey-... ./start-with-tailscale.sh"
    exit 1
fi

# Erstelle AppData-Verzeichnisse
echo "Erstelle Datenverzeichnisse..."
sudo mkdir -p /DATA/AppData/casadrop/data
sudo mkdir -p /DATA/AppData/casadrop/tailscale
# The app container runs as uid 10001 (see Dockerfile), so the data dir must be
# owned by 10001 — otherwise the scratch image (no su-exec re-chown) can't write.
sudo chown -R 10001:10001 /DATA/AppData/casadrop

# Migriere bestehende Daten falls vorhanden
if docker volume inspect casadrop-data >/dev/null 2>&1; then
    echo "Migriere bestehende Daten..."
    docker run --rm -v casadrop-data:/source -v /DATA/AppData/casadrop/data:/dest alpine sh -c "cp -a /source/. /dest/ 2>/dev/null || true"
fi

# Alte Container stoppen
echo "Stoppe alte Container..."
docker stop big-bear-tailscale zima-share casadrop casadrop-tailscale 2>/dev/null || true
docker rm big-bear-tailscale zima-share casadrop casadrop-tailscale 2>/dev/null || true

# Alte Volumes entfernen (Daten sind jetzt in AppData)
docker volume rm casadrop-data casadrop-tailscale-state 2>/dev/null || true

# Wechsle ins Projektverzeichnis
cd "$(dirname "$0")"

# Container starten
echo "Starte CasaDrop und Tailscale..."
TAILSCALE_AUTHKEY="$TAILSCALE_AUTHKEY" \
TAILSCALE_HOSTNAME="$TAILSCALE_HOSTNAME" \
WEBUI_PORT="$WEBUI_PORT" \
docker compose --profile tailscale up -d casadrop tailscale

# Warte auf Tailscale-Verbindung
echo "Warte auf Tailscale-Verbindung..."
sleep 5

# Prüfe Tailscale Status
echo "Tailscale Status:"
docker exec casadrop-tailscale tailscale status || true

# Aktiviere Funnel
echo ""
echo "Aktiviere Tailscale Funnel..."
docker exec casadrop-tailscale tailscale funnel --bg 8080

# Zeige URLs
echo ""
echo "=== CasaDrop ist gestartet ==="
echo ""
echo "Lokal:     http://localhost:$WEBUI_PORT"
echo "Tailscale: $(docker exec casadrop-tailscale tailscale funnel status 2>/dev/null | grep -oP 'https://[^\s]+' | head -1 || echo 'Funnel wird konfiguriert...')"
echo ""
echo "Logs anzeigen: docker compose logs -f"
