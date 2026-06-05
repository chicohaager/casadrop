#!/bin/bash
# CasaDrop Start Script with Auto-Detection
# Detects Tailscale, EasyTier, and local IP automatically

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== CasaDrop Auto-Detection ===${NC}"

# Tailscale IP detection
detect_tailscale() {
    if command -v tailscale &> /dev/null; then
        TS_IP=$(tailscale ip -4 2>/dev/null | head -1)
        if [ -n "$TS_IP" ]; then
            echo "$TS_IP"
            return
        fi
    fi

    # Interface scan (tailscale*)
    TS_IFACE=$(ip -o link show | grep -oE 'tailscale[0-9]*' | head -1)
    if [ -n "$TS_IFACE" ]; then
        TS_IP=$(ip -4 addr show "$TS_IFACE" 2>/dev/null | grep -oE '([0-9]{1,3}\.){3}[0-9]{1,3}' | head -1)
        if [ -n "$TS_IP" ]; then
            echo "$TS_IP"
            return
        fi
    fi
}

# Tailscale Funnel URL detection
detect_tailscale_url() {
    if command -v tailscale &> /dev/null; then
        TS_STATUS=$(tailscale status --json 2>/dev/null)
        if [ -n "$TS_STATUS" ]; then
            # Tolerate both compact and pretty-printed JSON (`status --json` has a space after the colon)
            TS_DNS=$(echo "$TS_STATUS" | grep -oE '"DNSName"[[:space:]]*:[[:space:]]*"[^"]+"' | head -1 | sed -E 's/.*:[[:space:]]*"([^"]+)".*/\1/' | sed 's/\.$//')
            if [ -n "$TS_DNS" ]; then
                echo "https://$TS_DNS"
                return
            fi
        fi
    fi
}

# EasyTier IP detection
detect_easytier() {
    # Method 1: easytier-cli
    if command -v easytier-cli &> /dev/null; then
        ET_IP=$(easytier-cli peer list 2>/dev/null | grep -E '(local|self)' | grep -oE '([0-9]{1,3}\.){3}[0-9]{1,3}' | head -1)
        if [ -n "$ET_IP" ]; then
            echo "$ET_IP"
            return
        fi
    fi

    # Method 2: Scan tun/easytier interfaces
    for IFACE in $(ip -o link show 2>/dev/null | grep -oE '(tun|easytier)[a-z0-9]*' | head -3); do
        ET_IP=$(ip -4 addr show "$IFACE" 2>/dev/null | grep -oE '([0-9]{1,3}\.){3}[0-9]{1,3}' | grep -v '127\.' | head -1)
        if [ -n "$ET_IP" ]; then
            echo "$ET_IP"
            return
        fi
    done
}

# Local IP detection (not Docker, not loopback)
detect_local_ip() {
    # Prefer eth0, then other interfaces
    for IFACE in eth0 enp0s3 ens3 eno1 wlan0 end0; do
        LOCAL_IP=$(ip -4 addr show "$IFACE" 2>/dev/null | grep -oE '([0-9]{1,3}\.){3}[0-9]{1,3}' | grep -v '127\.' | head -1)
        if [ -n "$LOCAL_IP" ]; then
            echo "$LOCAL_IP"
            return
        fi
    done

    # Fallback: First non-Docker, non-loopback IP
    LOCAL_IP=$(ip -4 addr show | grep -oE '([0-9]{1,3}\.){3}[0-9]{1,3}' | grep -v '127\.' | grep -v '172\.' | grep -v '169\.254\.' | head -1)
    echo "$LOCAL_IP"
}

# Run detection
TAILSCALE_IP=$(detect_tailscale)
TAILSCALE_URL=$(detect_tailscale_url)
EASYTIER_IP=$(detect_easytier)
LOCAL_IP=$(detect_local_ip)

# Display results
echo ""
if [ -n "$TAILSCALE_URL" ]; then
    echo -e "${GREEN}✓ Tailscale URL:${NC} $TAILSCALE_URL"
elif [ -n "$TAILSCALE_IP" ]; then
    echo -e "${GREEN}✓ Tailscale IP:${NC} $TAILSCALE_IP"
else
    echo -e "${YELLOW}○ Tailscale:${NC} not detected"
fi

if [ -n "$EASYTIER_IP" ]; then
    echo -e "${GREEN}✓ EasyTier IP:${NC} $EASYTIER_IP"
else
    echo -e "${YELLOW}○ EasyTier:${NC} not detected"
fi

if [ -n "$LOCAL_IP" ]; then
    echo -e "${GREEN}✓ Local IP:${NC} $LOCAL_IP"
else
    echo -e "${YELLOW}○ Local IP:${NC} not detected"
fi

echo ""
echo -e "${GREEN}=== Starting CasaDrop ===${NC}"
echo ""

# Export detected variables
export LOCAL_IP
export TAILSCALE_URL
export EASYTIER_IP

# Start with Docker Compose
exec docker compose up "$@"
