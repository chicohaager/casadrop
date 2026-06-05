# ZimaOS / CasaOS Setup

CasaDrop works seamlessly with ZimaOS and CasaOS. This guide covers the optimized setup for these platforms.

## Installation via App Store

CasaDrop will be available in the CasaOS App Store. Until then, use manual installation.

## Manual Installation

### 1. Create docker-compose.yml

```yaml
name: casadrop

services:
  casadrop:
    image: ghcr.io/user/casadrop:latest
    container_name: casadrop
    hostname: casadrop
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - type: volume
        source: casadrop-data
        target: /data
      # ZimaOS standard paths (read-only for security)
      - /DATA:/DATA:ro
      - /media:/media:ro
    environment:
      - PORT=8080
      - DATA_DIR=/data
      - TZ=Europe/Berlin
      - SHARE_ALLOWED_PATHS=/DATA,/media
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/api/auth/status"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 10s

volumes:
  casadrop-data:
    name: casadrop-data

x-casaos:
  architectures:
    - amd64
    - arm64
    - arm
  main: casadrop
  author: CasaDrop
  category: Utilities
  description:
    en_us: Simple and secure file sharing for your homelab. Share files via link with optional password protection and expiration.
  developer: CasaDrop
  icon: https://raw.githubusercontent.com/user/casadrop/main/web/static/img/icon.png
  tagline:
    en_us: Secure File Sharing
  title:
    custom: CasaDrop
  index: /
  port_map: "8080"
  scheme: http
  store_app_id: casadrop
```

### 2. Deploy

```bash
cd /DATA/AppData
mkdir casadrop && cd casadrop

# Save docker-compose.yml here

docker compose up -d
```

### 3. Access

Open `http://your-zima-ip:8080` or access via CasaOS dashboard.

## With Cloudflare Tunnel

For public HTTPS access:

```yaml
services:
  casadrop:
    # ... same as above ...

  tunnel:
    image: cloudflare/cloudflared:latest
    container_name: casadrop-tunnel
    restart: unless-stopped
    command: tunnel --no-autoupdate run
    environment:
      - TUNNEL_TOKEN=${CLOUDFLARE_TUNNEL_TOKEN}
    depends_on:
      - casadrop
```

Set up a Cloudflare Tunnel at [dash.cloudflare.com](https://dash.cloudflare.com) and add your token.

## ZimaOS Specific Paths

| Path | Description |
|------|-------------|
| `/DATA` | Main data storage |
| `/media` | Media files |
| `/DATA/AppData` | Application data |
| `/DATA/Downloads` | Downloads folder |

Mount the paths you want to share in the volumes section.

## Integration with ZimaOS Apps

### Jellyfin/Plex Media

Share media files directly:

```yaml
volumes:
  - /DATA/Media:/media:ro
```

Then use the file browser to share `/media/Movies/movie.mp4`.

### Immich Photos

Share photos from Immich library:

```yaml
volumes:
  - /DATA/AppData/immich/library:/photos:ro
```

## Troubleshooting

### Permission Denied

If you get permission errors with mounted volumes:

```bash
# Check ownership
ls -la /DATA

# CasaDrop runs as UID 1000 by default
# Ensure files are readable
chmod -R o+r /DATA/shared-folder
```

### Port Conflict

If port 8080 is used by another app:

```yaml
ports:
  - "8085:8080"  # Use port 8085 instead
environment:
  - EXTERNAL_PORT=8085
```

### Memory Issues on Zimaboard

For Zimaboard with limited RAM:

```yaml
deploy:
  resources:
    limits:
      memory: 256M
```

## Backup

CasaDrop stores data in `/data` inside the container:

```bash
# Backup
docker run --rm -v casadrop-data:/data -v /DATA/Backups:/backup \
  alpine tar czf /backup/casadrop-backup.tar.gz /data

# Restore
docker run --rm -v casadrop-data:/data -v /DATA/Backups:/backup \
  alpine tar xzf /backup/casadrop-backup.tar.gz -C /
```
