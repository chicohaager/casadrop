# CasaDrop - ZimaOS Installation Guide

## Overview

This guide covers installing CasaDrop on ZimaOS, including:
- Removing an existing installation
- Fresh installation via Docker Compose
- Configuration options
- Integration with Pangolin/Newt, Tailscale, EasyTier

---

## 1. Remove Old Installation

### Option A: Via ZimaOS UI

1. Open **ZimaOS Dashboard** → **Apps**
2. Find **CasaDrop** (or **Zima-Share**)
3. Click **⋮** (Three dots) → **Remove**
4. Select **"Delete app data"** if desired

### Option B: Via SSH/Terminal

```bash
# Stop and remove container
docker stop casadrop
docker rm casadrop

# Alternative: Container with old name
docker stop zima-share
docker rm zima-share

# Optional: Remove old image
docker rmi chicohaager/casadrop:latest

# Optional: Delete data (WARNING: Deletes all shares!)
# sudo rm -rf /DATA/AppData/casadrop

# Alternative: Keep config, delete uploads only
# sudo rm -rf /DATA/AppData/casadrop/uploads/*
```

### Option C: Clean Everything (Complete Reset)

```bash
# Find and stop all CasaDrop containers
docker ps -a | grep -E "casadrop|zima-share" | awk '{print $1}' | xargs -r docker stop
docker ps -a | grep -E "casadrop|zima-share" | awk '{print $1}' | xargs -r docker rm

# Remove all images
docker images | grep -E "casadrop|zima-share" | awk '{print $3}' | xargs -r docker rmi

# Prune unused volumes (optional)
docker volume prune -f

# Delete AppData (CAUTION!)
sudo rm -rf /DATA/AppData/casadrop
sudo rm -rf /DATA/AppData/zima-share
```

---

## 2. Fresh Installation

### Create Directories

```bash
# Create configuration directory
mkdir -p /DATA/AppData/casadrop
cd /DATA/AppData/casadrop
```

### Create docker-compose.yml

```bash
cat > docker-compose.yml << 'EOF'
version: "3.8"

services:
  casadrop:
    image: chicohaager/casadrop:latest
    container_name: casadrop
    restart: unless-stopped
    ports:
      - "8787:8080"
    volumes:
      # Data directory (database, config, uploads)
      - ./data:/data
      
      # Media directories for file browser (read-only recommended)
      - /DATA:/DATA:ro
      - /media:/media:ro
      
    environment:
      # ============= GENERAL =============
      - TZ=Europe/Berlin
      - PORT=8080
      - EXTERNAL_PORT=8787
      
      # ============= SECURITY =============
      # Admin password (CHANGE THIS!)
      - ADMIN_PASSWORD=your-secure-password
      
      # Allowed paths for file browser
      - SHARE_ALLOWED_PATHS=/DATA,/media
      
      # ============= NETWORK (Optional) =============
      # Set manually if auto-detection doesn't work
      # - LOCAL_IP=192.168.1.100
      # - EASYTIER_IP=10.147.20.50
      # - TAILSCALE_URL=https://zima.tail1234.ts.net
      # Pangolin/Newt needs no variable — share links automatically
      # follow the X-Forwarded-Host/Host header.
      
    # For EasyTier/Tailscale auto-detection:
    network_mode: host
    # OR with port mapping (then set IPs manually):
    # networks:
    #   - default

EOF
```

### Start Installation

```bash
cd /DATA/AppData/casadrop
docker compose pull
docker compose up -d
```

### Verify Installation

```bash
# Show logs
docker logs casadrop

# Check status
docker ps | grep casadrop

# Open Web UI
echo "CasaDrop running at: http://$(hostname -I | awk '{print $1}'):8787"
```

---

## 3. Configuration Variants

### Variant A: Minimal (Local Network)

```yaml
services:
  casadrop:
    image: chicohaager/casadrop:latest
    container_name: casadrop
    restart: unless-stopped
    ports:
      - "8787:8080"
    volumes:
      - ./data:/data
    environment:
      - TZ=Europe/Berlin
      - ADMIN_PASSWORD=my-password
```

### Variant B: With Pangolin/Newt

Pangolin/Newt needs **no dedicated variable**: CasaDrop builds share links from the
`X-Forwarded-Host`/`Host` header sent by the reverse proxy, so the public domain is
used automatically.

```yaml
services:
  casadrop:
    image: chicohaager/casadrop:latest
    container_name: casadrop
    restart: unless-stopped
    ports:
      - "8787:8080"
    volumes:
      - ./data:/data
      - /DATA:/DATA:ro
    environment:
      - TZ=Europe/Berlin
      - ADMIN_PASSWORD=my-password
      - EXTERNAL_PORT=8787
      - SHARE_ALLOWED_PATHS=/DATA,/media
```

### Variant C: With OIDC (Authentik/Keycloak)

```yaml
services:
  casadrop:
    image: chicohaager/casadrop:latest
    container_name: casadrop
    restart: unless-stopped
    ports:
      - "8787:8080"
    volumes:
      - ./data:/data
    environment:
      - TZ=Europe/Berlin
      
      # OIDC Configuration
      - OIDC_ENABLED=true
      - OIDC_ISSUER_URL=https://auth.example.com/application/o/casadrop/
      - OIDC_CLIENT_ID=casadrop
      - OIDC_CLIENT_SECRET=secret-key
      - OIDC_REDIRECT_URL=https://share.example.com/auth/oidc/callback
      - OIDC_DEFAULT_ROLE=user
      
      # Optional: Disable local authentication
      # - OIDC_DISABLE_LOCAL_AUTH=true
```

### Variant D: With Cloudflare Tunnel

```yaml
services:
  casadrop:
    image: chicohaager/casadrop:latest
    container_name: casadrop
    restart: unless-stopped
    ports:
      - "8787:8080"
    volumes:
      - ./data:/data
    environment:
      - TZ=Europe/Berlin
      - ADMIN_PASSWORD=my-password
      - TUNNEL_URL=https://share.my-domain.com

  # Cloudflare Tunnel Container
  tunnel:
    image: cloudflare/cloudflared:latest
    container_name: casadrop-tunnel
    restart: unless-stopped
    command: tunnel run
    environment:
      - TUNNEL_TOKEN=your-tunnel-token
    depends_on:
      - casadrop
```

### Variant E: Full with Host Network

```yaml
services:
  casadrop:
    image: chicohaager/casadrop:latest
    container_name: casadrop
    restart: unless-stopped
    network_mode: host
    volumes:
      - ./data:/data
      - /DATA:/DATA:ro
      - /media:/media:ro
    environment:
      - TZ=Europe/Berlin
      - PORT=8787
      - EXTERNAL_PORT=8787
      - ADMIN_PASSWORD=my-password
      - SHARE_ALLOWED_PATHS=/DATA,/media,/home
```

---

## 4. Environment Variables Reference

### General

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8080 | Internal port |
| `EXTERNAL_PORT` | 8080 | External port (for share URLs) |
| `DATA_DIR` | /data | Data directory |
| `ADMIN_PASSWORD` | - | Admin password |
| `TZ` | UTC | Timezone |

### File Browser

| Variable | Default | Description |
|----------|---------|-------------|
| `SHARE_ALLOWED_PATHS` | /DATA,/media,/home | Allowed paths (comma-separated) |

### Network

| Variable | Default | Description |
|----------|---------|-------------|
| `LOCAL_IP` | auto | Local IP address |
| `EASYTIER_IP` | auto | EasyTier IP |
| `TAILSCALE_URL` | auto | Tailscale Funnel URL |
| `TUNNEL_URL` | - | Cloudflare Tunnel URL |
| `CUSTOM_URL` | - | Custom URL (WireGuard, etc.) |

There is no variable for Pangolin/Newt — share links automatically follow the
`X-Forwarded-Host`/`Host` header.

### OIDC/SSO

| Variable | Default | Description |
|----------|---------|-------------|
| `OIDC_ENABLED` | false | Enable OIDC |
| `OIDC_ISSUER_URL` | - | OIDC provider URL |
| `OIDC_CLIENT_ID` | - | Client ID |
| `OIDC_CLIENT_SECRET` | - | Client secret |
| `OIDC_REDIRECT_URL` | - | Callback URL |
| `OIDC_DEFAULT_ROLE` | user | Default role (admin/user/viewer) |

### Email

Email/SMTP is **not** configured via environment variables. It is configured only
in the admin UI under **Settings** (persisted through the `/api/smtp` endpoints).

---

## 5. After Installation

### First Login

1. Open `http://[ZIMA-IP]:8787`
2. Login with `admin` / `[ADMIN_PASSWORD]`
3. Go to **Settings** to configure network settings

### Configure Network

1. Open **Settings** → **Network**
2. Choose **Primary Network** (for share URLs):
   - `Cloudflare Tunnel` - For Cloudflare Tunnel
   - `Tailscale Funnel` - For Tailscale Funnel
   - `EasyTier` - For EasyTier
   - `Custom Domain / URL` - Own reverse proxy / domain (e.g. Pangolin/Newt)
   - `Local Network` - For local network
3. Disable unused networks

### Set Up Receive Links

1. Go to **Receive Links**
2. Click **Create Receive Link**
3. Optional: Set target directory (e.g., `/DATA/Uploads`)
4. Share the link - others can upload files to you

---

## 6. Backup & Restore

### Create Backup

```bash
cd /DATA/AppData/casadrop

# Stop container
docker compose stop

# Create backup
tar -czvf casadrop-backup-$(date +%Y%m%d).tar.gz data/

# Start container
docker compose start
```

### Restore Backup

```bash
cd /DATA/AppData/casadrop

# Stop container
docker compose stop

# Backup current data directory
mv data data.old

# Extract backup
tar -xzvf casadrop-backup-YYYYMMDD.tar.gz

# Start container
docker compose start
```

---

## 7. Update

```bash
cd /DATA/AppData/casadrop

# Pull new image
docker compose pull

# Restart container
docker compose up -d

# Clean up old images
docker image prune -f
```

---

## 8. Troubleshooting

### Container Won't Start

```bash
# Check logs
docker logs casadrop

# Check permissions
ls -la /DATA/AppData/casadrop/data/
```

### Web UI Not Accessible

```bash
# Check port
netstat -tlnp | grep 8787

# Check firewall
sudo ufw status
```

### Networks Not Detected

With `network_mode: host`, auto-detection should work.
Otherwise set manually:

```yaml
environment:
  - LOCAL_IP=192.168.1.100
  - EASYTIER_IP=10.147.20.50
```

### Database Errors

```bash
# Repair SQLite
docker exec casadrop sqlite3 /data/shares.db "PRAGMA integrity_check;"

# Or reset completely (DELETES ALL DATA!)
docker compose stop
rm /DATA/AppData/casadrop/data/shares.db
docker compose start
```

---

## 9. Uninstallation

### Remove Container Only (Keep Data)

```bash
cd /DATA/AppData/casadrop
docker compose down
```

### Complete Uninstallation

```bash
cd /DATA/AppData/casadrop
docker compose down -v --rmi all
cd ..
rm -rf casadrop
```

---

## Support

- **GitHub Issues**: [github.com/chicohaager/casadrop/issues](https://github.com/chicohaager/casadrop/issues)
- **IceWhale Community**: [community.zimaspace.com](https://community.zimaspace.com)
