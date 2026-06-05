# CasaDrop 2.3.0 — Installation (English)

Secure, self-hosted file sharing for your homelab.
This package installs on **ZimaOS / CasaOS** (one-click) or on **any Linux
server with Docker**.

## Package contents

| File | Purpose |
|------|---------|
| `casadrop-zimaos-app.yaml` | App definition for the ZimaOS/CasaOS App Store **and** usable as a Docker Compose file |
| `casadrop-2.3.0-amd64.tar.gz` | Offline image (only needed when there is no internet / Docker Hub access) |
| `.env.example` | Template for optional settings |
| `INSTALL_DE.md` / `INSTALL_EN.md` | This guide |
| `NOTICE.txt` | Copyright / licensing note |
| `SHA256SUMS` | Checksums for integrity verification |

**Architecture:** amd64 (x86-64). **Default port:** 8080.

---

## Option A — ZimaOS / CasaOS (one-click, recommended)

1. In the ZimaOS dashboard: **App Store → Custom Install / Import**.
2. Upload `casadrop-zimaos-app.yaml` (or paste its content).
3. Optionally adjust the port (`WEBUI_PORT`), then **Install**.
   The image is pulled automatically from Docker Hub (`chicohaager/casadrop:2.3.0`).
4. Open the app tile → the setup wizard appears.

---

## Option B — Any Linux server with Docker

```bash
# 1) (offline only) load the image locally — skip if you have internet:
gunzip -c casadrop-2.3.0-amd64.tar.gz | docker load

# 2) start:
docker compose -f casadrop-zimaos-app.yaml up -d

# 3) custom port (optional):
WEBUI_PORT=8085 docker compose -f casadrop-zimaos-app.yaml up -d
```

Then open `http://<server-ip>:8080` (or your chosen port).

---

## First run: set the admin password

On first open a **setup wizard** creates the admin password. For safety it
requires a **one-time setup token** that is printed only to the container log:

```bash
docker logs casadrop 2>&1 | grep -i "setup token"
```

Or skip the wizard by setting a password before start, in `.env`:

```
ADMIN_PASSWORD=AStrongPassword
```

---

## Optional settings (`.env`)

Copy `.env.example` to `.env` and edit. Key switches:

| Variable | Default | Meaning |
|----------|---------|---------|
| `WEBUI_PORT` | 8080 | Host port of the WebUI |
| `TZ` | Europe/Berlin | Timezone |
| `ADMIN_PASSWORD` | – | Admin password (skips the wizard) |
| `TRUSTED_PROXY` | – | Reverse-proxy IP/CIDR (recovers real client IP) |
| `SHARE_ALLOWED_PATHS` | /DATA,/media | File-browser roots (admin only) |
| `OIDC_ENABLED` | false | Enable SSO/OIDC |

> **Reverse proxy:** behind nginx/Traefik/Caddy, set `TRUSTED_PROXY` to the
> proxy IP — otherwise `X-Forwarded-For` headers are ignored (fail-closed) and
> rate limits only ever see the proxy IP.

---

## Data & backup

All payload data (SQLite DB, uploads, thumbnails) lives in the `/data` volume,
bound to `/DATA/AppData/casadrop/data` on ZimaOS. Back up that directory.

## Update

```bash
docker compose -f casadrop-zimaos-app.yaml pull
docker compose -f casadrop-zimaos-app.yaml up -d
```

## Verify integrity

```bash
sha256sum -c SHA256SUMS
```

## Support

Source & issues: https://github.com/chicohaager/casadrop
