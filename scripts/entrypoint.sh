#!/bin/sh
set -e

echo "=== CasaDrop Starting ==="
echo "Port: ${PORT:-8080}"
echo "Data: ${DATA_DIR:-/data}"

# Ensure data directories exist
mkdir -p "${DATA_DIR:-/data}/uploads" "${DATA_DIR:-/data}/thumbnails"

# Auto-detect local IP if not set
if [ -z "$LOCAL_IP" ]; then
    LOCAL_IP=$(ip -4 route get 1.1.1.1 2>/dev/null | grep -oP 'src \K\S+' || hostname -i 2>/dev/null | awk '{print $1}' || echo "")
    if [ -n "$LOCAL_IP" ]; then
        export LOCAL_IP
        echo "Local IP: $LOCAL_IP"
    fi
fi

# Auto-detect Tailscale if available
if command -v tailscale >/dev/null 2>&1; then
    TS_IP=$(tailscale ip -4 2>/dev/null || true)
    if [ -n "$TS_IP" ]; then
        echo "Tailscale IP: $TS_IP"
    fi
    TS_STATUS=$(tailscale status --json 2>/dev/null || true)
    if [ -n "$TS_STATUS" ]; then
        # Tolerate both compact ("DNSName":"x") and pretty-printed ("DNSName": "x")
        # JSON — `tailscale status --json` emits the latter. First match is Self.
        TS_DNS=$(echo "$TS_STATUS" | grep -oE '"DNSName"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed -E 's/.*:[[:space:]]*"([^"]*)".*/\1/' | sed 's/\.$//')
        if [ -n "$TS_DNS" ] && [ -z "$TAILSCALE_URL" ]; then
            export TAILSCALE_URL="https://$TS_DNS"
            echo "Tailscale URL: $TAILSCALE_URL"
        fi
    fi
fi

# Auto-detect EasyTier if available
if command -v easytier-cli >/dev/null 2>&1; then
    ET_IP=$(easytier-cli peer list 2>/dev/null | grep -oE '([0-9]{1,3}\.){3}[0-9]{1,3}' | head -1 || true)
    if [ -n "$ET_IP" ] && [ -z "$EASYTIER_IP" ]; then
        export EASYTIER_IP="$ET_IP"
        echo "EasyTier IP: $EASYTIER_IP"
    fi
fi

# Auto-detect Cloudflare Tunnel URL
if [ -z "$TUNNEL_URL" ] && [ -f "${DATA_DIR:-/data}/tunnel_url.txt" ]; then
    TUNNEL_URL=$(cat "${DATA_DIR:-/data}/tunnel_url.txt" | tr -d '[:space:]')
    if [ -n "$TUNNEL_URL" ]; then
        export TUNNEL_URL
        echo "Tunnel URL: $TUNNEL_URL"
    fi
fi

echo "=== Starting CasaDrop Server ==="

# Run as casadrop user if possible, otherwise root
if [ "$(id -u)" = "0" ] && command -v su-exec >/dev/null 2>&1; then
    # Ensure data directory is writable
    chown -R casadrop:casadrop "${DATA_DIR:-/data}" 2>/dev/null || true
    exec su-exec casadrop /app/casadrop
else
    exec /app/casadrop
fi
