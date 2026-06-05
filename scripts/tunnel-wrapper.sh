#!/bin/sh
# tunnel-wrapper.sh — Cloudflare Tunnel entrypoint for the optional `tunnel`
# compose service. Two modes, picked automatically:
#
#   1. Quick Tunnel (default, free, no Cloudflare account): runs
#      `cloudflared tunnel --url $TARGET_URL`, which mints a random
#      https://<words>.trycloudflare.com URL. We scrape that URL from
#      cloudflared's output and persist it to /data/tunnel_url.txt, which the
#      app reads back via internal/handlers/network.go (detectCloudflareTunnel,
#      Method 1) and scripts/entrypoint.sh. This is the "direct sharing" path.
#
#   2. Token Tunnel (CLOUDFLARE_TUNNEL_TOKEN set): runs a named tunnel whose
#      public hostname is configured in the Cloudflare dashboard. The hostname
#      isn't discoverable here, so we write the sentinel "token" (which
#      network.go explicitly ignores) and let the user pin the fixed domain via
#      TUNNEL_URL / the settings UI.
#
# The data volume is shared with the app container (casadrop-data:/data), so the
# file written here is the file the app reads.
set -eu

TARGET_URL="${TARGET_URL:-http://casadrop:8080}"
DATA_DIR="${DATA_DIR:-/data}"
URL_FILE="${DATA_DIR}/tunnel_url.txt"

mkdir -p "$DATA_DIR"

# ---- Token Tunnel ---------------------------------------------------------
if [ -n "${CLOUDFLARE_TUNNEL_TOKEN:-}" ]; then
    # Sentinel: tells the app a tunnel exists but the URL is managed elsewhere.
    printf 'token\n' > "$URL_FILE"
    echo "[tunnel] starting named tunnel via token"
    exec cloudflared tunnel --no-autoupdate run --token "$CLOUDFLARE_TUNNEL_TOKEN"
fi

# ---- Quick Tunnel (free, no account) --------------------------------------
echo "[tunnel] starting quick tunnel -> ${TARGET_URL}"

# Rotation channel: the app drops this sentinel into the shared data dir when
# the user (re-)selects Cloudflare as the primary network. We watch for it and
# mint a fresh trycloudflare.com URL on demand — see requestCloudflareRotate in
# internal/handlers/network.go. The app only WRITES the request (the data dir
# is owned by the app user); this wrapper, running as root, owns tunnel_url.txt
# and is the only side that clears/rewrites it.
REQ_FILE="${DATA_DIR}/tunnel_rotate.request"

# cloudflared appends to a single log; `tail -f` below streams it and we read the
# NEWEST minted URL with `tail -n1`. The supervisor loop caps this file's size in
# place (see below) so a long-lived tunnel can't fill the container's writable
# layer.
LOG="$(mktemp)"
CF_PID=""
PREV_URL=""

# Clear any stale rotate request and dead URL from a previous container run.
rm -f "$REQ_FILE"
: > "$URL_FILE"

# start_cloudflared launches cloudflared and waits (up to ~60s) for a freshly
# minted URL that differs from the previous one, persisting it to URL_FILE.
start_cloudflared() {
    cloudflared tunnel --no-autoupdate --url "$TARGET_URL" >> "$LOG" 2>&1 &
    CF_PID="$!"

    i=0
    while [ "$i" -lt 120 ]; do
        if ! kill -0 "$CF_PID" 2>/dev/null; then
            echo "[tunnel] cloudflared exited before a URL was assigned"
            return 1
        fi
        URL="$(grep -oE 'https://[a-zA-Z0-9.-]+\.trycloudflare\.com' "$LOG" | tail -n1 || true)"
        if [ -n "$URL" ] && [ "$URL" != "$PREV_URL" ]; then
            printf '%s\n' "$URL" > "$URL_FILE"
            PREV_URL="$URL"
            echo "[tunnel] quick tunnel ready: ${URL}"
            return 0
        fi
        i=$((i + 1))
        sleep 1
    done
    echo "[tunnel] WARNING: no fresh trycloudflare.com URL detected after 60s"
    return 0
}

# stop_cloudflared terminates the current cloudflared and reaps it.
stop_cloudflared() {
    [ -n "$CF_PID" ] || return 0
    kill "$CF_PID" 2>/dev/null || true
    wait "$CF_PID" 2>/dev/null || true
    CF_PID=""
}

# Clean container stop: kill cloudflared and drop the request sentinel.
trap 'stop_cloudflared; rm -f "$REQ_FILE"; exit 0' TERM INT

# Surface cloudflared's ongoing logs on the container's stdout.
tail -f "$LOG" &

start_cloudflared

# Supervisor loop: rotate the URL on request, or restart cloudflared if it dies.
while true; do
    if [ -f "$REQ_FILE" ]; then
        rm -f "$REQ_FILE"
        echo "[tunnel] rotate requested -> minting a fresh URL"
        # Blank the URL first so the UI shows a transient "minting" state
        # instead of the link that's about to die.
        : > "$URL_FILE"
        stop_cloudflared
        start_cloudflared
    elif ! kill -0 "$CF_PID" 2>/dev/null; then
        echo "[tunnel] cloudflared died -> restarting"
        : > "$URL_FILE"
        start_cloudflared
    fi
    # Cap the log so a long-lived tunnel can't fill the disk. Truncate in place
    # (preserves the inode the `tail -f` above follows), keeping recent lines.
    if [ "$(wc -c < "$LOG" 2>/dev/null || echo 0)" -gt 1048576 ]; then
        tail -n 200 "$LOG" > "${LOG}.trim" 2>/dev/null || true
        cat "${LOG}.trim" > "$LOG" 2>/dev/null || true
        rm -f "${LOG}.trim"
    fi
    sleep 1
done
