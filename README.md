# CasaDrop

Self-hosted file sharing for your homelab — share files and entire folders via link or QR, receive uploads from others, with password protection, expiry, and download limits. Multi-user with OIDC/SSO, media streaming, and one-step public access over Tailscale, Cloudflare Tunnel, or any reverse proxy — all in a single lightweight static Go binary.

[![Docker Image](https://img.shields.io/badge/docker-ghcr.io%2Fuser%2Fcasadrop-blue)](https://ghcr.io/user/casadrop)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

---

## Quick Start

```bash
docker run -d -p 8080:8080 -v casadrop_data:/data ghcr.io/user/casadrop:latest
```

Open http://localhost:8080 and start sharing!

---

## Highlights

- **Tiny & fast** — a single ~17 MB static Go binary with embedded SQLite (no external database, no CGO)
- **Files & folders** — share single files or whole directories with a browse UI and on-the-fly ZIP download
- **Receive links** — let others upload to you, with per-link limits, allowed extensions, and webhooks
- **Multi-user + SSO** — Admin/User/Viewer roles, local email+password accounts, and OIDC (Authentik, Keycloak, …)
- **Public access, your way** — share links auto-match how you reach the app: Tailscale, Cloudflare Tunnel, Pangolin, or any reverse proxy
- **Sharing extras** — passwords, expiry, download limits, QR codes, image thumbnails, and media streaming

---

## Features

### File Sharing
- Drag & Drop upload (up to 100 GB configurable)
- Password protection with bcrypt hashing
- Expiration (1 hour to 30 days)
- Download limits
- QR codes for mobile sharing
- Image thumbnails

### Folder Sharing
- Share entire directories
- Browse folders in web UI
- Download as ZIP
- Password protection

### Receive Links
- Let others upload files to you
- Upload limits per link
- File type restrictions
- Auto-share received files
- Webhook notifications

### Multi-User Support
- **Three roles**: Admin, User, Viewer
- **Admin**: Full access, manage users/settings, see all shares
- **User**: Create/manage own shares and receive links
- **Viewer**: Read-only, download shared files only
- User management UI for admins
- Ownership tracking for shares

### OIDC/SSO Authentication
- Single Sign-On with any OIDC provider (Authentik, Keycloak, Google, etc.)
- Auto-provisioning of users on first login
- Configurable default role for new users
- Optional: Disable local password authentication

### Monitoring
- Prometheus metrics at `/api/metrics` (admin-only)
- Health probes: `/healthz` (liveness), `/readyz` (readiness)
- Statistics dashboard
- Upload/download tracking

### Security
- Role-based access control (RBAC)
- Local login: single admin password **or** per-user email + password; plus OIDC/SSO
- Optional **TOTP two-factor (2FA)** for the admin login — authenticator app, enrolled in Settings
- Rate limiting + account lockout (per-IP; configurable trusted proxies)
- Strict Content-Security-Policy (`script-src 'self'`), HSTS, security headers
- Webhook SSRF guard (literal-IP block, redirect refusal, optional DNS pinning)
- Session absolute lifetime on top of rolling idle timeout
- Blocked executable uploads
- SQLite with WAL mode (pure-Go, fully static binary)

---

## Installation

### Docker Compose (Recommended)

```yaml
# docker-compose.yml
services:
  casadrop:
    image: ghcr.io/user/casadrop:latest
    container_name: casadrop
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
      - /mnt/media:/media:ro  # Optional: Share host directories
    environment:
      - TZ=Europe/Berlin
      - ADMIN_PASSWORD=your-secure-password
    restart: unless-stopped
```

```bash
docker compose up -d
```

### With OIDC (e.g., Authentik)

```yaml
services:
  casadrop:
    image: ghcr.io/user/casadrop:latest
    container_name: casadrop
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
    environment:
      - TZ=Europe/Berlin
      - OIDC_ENABLED=true
      - OIDC_ISSUER_URL=https://auth.example.com/application/o/casadrop/
      - OIDC_CLIENT_ID=your-client-id
      - OIDC_CLIENT_SECRET=your-client-secret
      - OIDC_REDIRECT_URL=https://share.example.com/auth/oidc/callback
      - OIDC_DEFAULT_ROLE=user
    restart: unless-stopped
```

### With Traefik

```yaml
services:
  casadrop:
    image: ghcr.io/user/casadrop:latest
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.casadrop.rule=Host(`share.example.com`)"
      - "traefik.http.routers.casadrop.tls.certresolver=letsencrypt"
    volumes:
      - ./data:/data
    networks:
      - traefik
    restart: unless-stopped

networks:
  traefik:
    external: true
```

### ZimaOS / CasaOS

See [docs/zimaos.md](docs/zimaos.md) for ZimaOS-specific setup with auto-detection for EasyTier, Tailscale (Funnel), and Cloudflare Tunnel.

### Build from Source

Requires Go 1.25+. No CGO needed (pure-Go SQLite) — produces a fully static binary:

```bash
CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -trimpath -o casadrop ./cmd/server
./casadrop   # serves on :8080 (override with PORT)
```

Or build the container image: `docker build -t casadrop:latest .`

---

## Configuration

### General Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8080 | Internal port |
| `EXTERNAL_PORT` | 8080 | External port used when building links |
| `DATA_DIR` | /data | Data directory |
| `ADMIN_PASSWORD` | - | Admin password (if set, skips the setup wizard) |
| `TZ` | Europe/Berlin | Timezone |
| `SHARE_ALLOWED_PATHS` | /DATA,/media,/home | Paths for file browser |
| `MAX_FOLDER_ZIP_GB` | 10 | Max uncompressed budget per folder-ZIP download |
| `TRUSTED_PROXY` | - | CIDRs/IPs of trusted proxies; gates `X-Forwarded-For` (anti-spoof) |

### Webhooks

| Variable | Default | Description |
|----------|---------|-------------|
| `WEBHOOK_URL` | - | Webhook notification URL |
| `WEBHOOK_SECRET` | - | HMAC-SHA256 signing secret |
| `STRICT_WEBHOOK_URLS` | true | Reject webhook URLs whose host is a literal private/loopback IP |
| `WEBHOOK_STRICT_SSRF` | false | Resolve + pin the target IP, reject private addresses (defeats DNS rebinding) |

### OIDC/SSO Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `OIDC_ENABLED` | false | Enable OIDC authentication |
| `OIDC_ISSUER_URL` | - | OIDC provider URL |
| `OIDC_CLIENT_ID` | - | OAuth2 client ID |
| `OIDC_CLIENT_SECRET` | - | OAuth2 client secret |
| `OIDC_REDIRECT_URL` | - | Callback URL (https://your-domain/auth/oidc/callback) |
| `OIDC_SCOPES` | openid,profile,email | OAuth2 scopes |
| `OIDC_DEFAULT_ROLE` | viewer | Default role for auto-provisioned users (admin/user/viewer) |
| `OIDC_AUTO_PROVISION` | true | Auto-create users on first OIDC login |
| `OIDC_DISABLE_LOCAL_AUTH` | false | Disable password login (OIDC only) |

### Network / Public Access

Behind a reverse proxy or tunnel (Pangolin/Newt, Traefik, Caddy, Tailscale),
share links automatically use the host you reach the app through (via
`X-Forwarded-Host`/`Host`) — no configuration needed. The variables below pin a
specific public URL, mostly for ZimaOS auto-detection:

| Variable | Description |
|----------|-------------|
| `TUNNEL_URL` | Cloudflare Tunnel URL |
| `TAILSCALE_URL` | Tailscale Funnel URL (auto-detected when the Tailscale CLI/socket is available) |
| `EASYTIER_IP` | EasyTier VPN IP (auto-detected) |
| `CUSTOM_URL` | Custom URL (WireGuard, reverse proxy, etc.) |
| `LOCAL_IP` | Local network IP (auto-detected) |

---

## User Roles

| Role | Upload | View Own | View All | Delete Own | Delete All | Manage Users | Settings |
|------|--------|----------|----------|------------|------------|--------------|----------|
| Admin | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| User | Yes | Yes | No | Yes | No | No | No |
| Viewer | No | Yes | No | No | No | No | No |

---

## API

```bash
# Upload file (requires authentication)
curl -X POST http://localhost:8080/api/upload \
  -H "Cookie: casadrop_session=..." \
  -F "file=@document.pdf" \
  -F "expires_in=24"

# Share folder
curl -X POST http://localhost:8080/api/share-folder \
  -H "Cookie: casadrop_session=..." \
  -H "Content-Type: application/json" \
  -d '{"path":"/media/photos","expires_in":24}'

# Create receive link
curl -X POST http://localhost:8080/api/receive-links \
  -H "Cookie: casadrop_session=..." \
  -H "Content-Type: application/json" \
  -d '{"name":"Project Files","max_uploads":10}'

# Get current user info
curl http://localhost:8080/api/me \
  -H "Cookie: casadrop_session=..."

# List users (admin only)
curl http://localhost:8080/api/users \
  -H "Cookie: casadrop_session=..."

# Prometheus metrics (admin-only)
curl http://localhost:8080/api/metrics -H "Cookie: casadrop_session=..."

# Health probes (no auth)
curl http://localhost:8080/healthz   # liveness
curl http://localhost:8080/readyz    # readiness (storage reachable)
```

See [docs/api.md](docs/api.md) for full API documentation.

---

## Documentation

- [Docker Compose Setup](docs/docker-compose.md)
- [Reverse Proxy (Traefik/Caddy/nginx)](docs/reverse-proxy.md)
- [OIDC Configuration](docs/oidc.md)
- [ZimaOS Setup](docs/zimaos.md)
- [API Reference](docs/api.md)
- [Migration from Pingvin Share](docs/migration-pingvin.md)

---

## Roadmap

- [x] SQLite database
- [x] Prometheus metrics
- [x] Folder sharing
- [x] Receive links
- [x] Image thumbnails
- [x] OIDC/OAuth2 (Authentik, Keycloak)
- [x] Multi-user support with RBAC
- [x] Per-user local authentication (email + password)
- [x] Email notifications (SMTP)
- [ ] Tailscale Taildrop ("send to my device")
- [ ] S3 storage backend
- [ ] Share link customization

---

## Contributing

Contributions are welcome! Please open an issue or PR.

---

## License

MIT

---

## Acknowledgments

- Originally developed as Zima-Share for ZimaOS
