# Docker Compose Setup

## Quick Start

```bash
# Create directory
mkdir casadrop && cd casadrop

# Download docker-compose.yml
curl -O https://raw.githubusercontent.com/chicohaager/casadrop/main/docker-compose.yml

# Start
docker compose up -d

# Access at http://localhost:8080
```

## Basic Configuration

```yaml
# docker-compose.yml
services:
  casadrop:
    image: chicohaager/casadrop:latest
    container_name: casadrop
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
      # Optional: Mount host directories for file browser
      - /mnt/media:/media:ro
      - /home:/home:ro
    environment:
      - TZ=Europe/Berlin
      - SHARE_ALLOWED_PATHS=/media,/home
    restart: unless-stopped
```

## With Persistent Volume

```yaml
services:
  casadrop:
    image: chicohaager/casadrop:latest
    container_name: casadrop
    ports:
      - "8080:8080"
    volumes:
      - casadrop-data:/data
    environment:
      - TZ=Europe/Berlin
    restart: unless-stopped

volumes:
  casadrop-data:
```

## With Admin Password

```yaml
services:
  casadrop:
    image: chicohaager/casadrop:latest
    container_name: casadrop
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
    environment:
      - TZ=Europe/Berlin
      - ADMIN_PASSWORD=your-secure-password
    restart: unless-stopped
```

Or use the web-based setup wizard at `/setup` on first launch.

## With Cloudflare Tunnel

For public HTTPS access without port forwarding:

```yaml
services:
  casadrop:
    image: chicohaager/casadrop:latest
    container_name: casadrop
    volumes:
      - casadrop-data:/data
    environment:
      - TZ=Europe/Berlin
    restart: unless-stopped

  tunnel:
    image: cloudflare/cloudflared:latest
    container_name: casadrop-tunnel
    command: tunnel --no-autoupdate run --token ${CLOUDFLARE_TUNNEL_TOKEN}
    environment:
      - CLOUDFLARE_TUNNEL_TOKEN=${CLOUDFLARE_TUNNEL_TOKEN}
    depends_on:
      - casadrop
    restart: unless-stopped

volumes:
  casadrop-data:
```

Run with:
```bash
CLOUDFLARE_TUNNEL_TOKEN=your-token docker compose up -d
```

## With Tailscale Funnel

Alternative to Cloudflare with stable URLs:

```yaml
services:
  casadrop:
    image: chicohaager/casadrop:latest
    container_name: casadrop
    volumes:
      - casadrop-data:/data
    environment:
      - TZ=Europe/Berlin
    restart: unless-stopped

  tailscale:
    image: tailscale/tailscale:latest
    container_name: casadrop-tailscale
    cap_add:
      - NET_ADMIN
      - SYS_MODULE
    volumes:
      - tailscale-state:/var/lib/tailscale
      - /dev/net/tun:/dev/net/tun
    environment:
      - TS_AUTHKEY=${TAILSCALE_AUTHKEY}
      - TS_EXTRA_ARGS=--advertise-tags=tag:container
      - TS_SERVE_CONFIG=/config/serve.json
      - TS_STATE_DIR=/var/lib/tailscale
    network_mode: service:casadrop
    restart: unless-stopped

volumes:
  casadrop-data:
  tailscale-state:
```

## Health Check

CasaDrop includes a built-in health check endpoint:

```yaml
healthcheck:
  test: ["CMD", "wget", "-qO-", "http://localhost:8080/api/auth/status"]
  interval: 30s
  timeout: 3s
  retries: 3
  start_period: 10s
```

## Resource Limits

```yaml
services:
  casadrop:
    image: chicohaager/casadrop:latest
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
        reservations:
          cpus: '0.25'
          memory: 128M
```

## Upgrading

```bash
# Pull latest image
docker compose pull

# Recreate container
docker compose up -d

# Check logs
docker compose logs -f casadrop
```

Data is preserved in the mounted volume.
