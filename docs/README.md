# CasaDrop Documentation

Self-hosted file sharing for your homelab.

## Quick Links

| Topic | Description |
|-------|-------------|
| **[Complete HowTo](HOWTO.md)** | **Start here** — install, share, expose publicly, automate, harden |
| [User Guide (HTML)](CasaDrop-HowTo-Users-2026-08-28.html) | For the people you hand a login to — signing in, sending files, managing shares, receiving files, roles & quota, 2FA. Self-contained, prints cleanly |
| [Network & Webhooks (HTML)](CasaDrop-HowTo-Network-Webhooks-2026-09-07.html) | For operators — which address ends up in a share link and why, reverse proxy / `TRUSTED_PROXY`, tunnels, webhook setup and the SSRF guard |
| [Docker Compose Setup](docker-compose.md) | Standard deployment guide |
| [Reverse Proxy](reverse-proxy.md) | Nginx, Traefik, Caddy configurations |
| [Tailscale](tailscale.md) | Host Tailscale or bundled sidecar, serve/Funnel, Taildrop |
| [OIDC/SSO](oidc.md) | Authentik, Keycloak integration |
| [API Reference](api.md) | REST API documentation |
| [ZimaOS/CasaOS](zimaos.md) | ZimaOS-specific setup |
| [Kubernetes](kubernetes/README.md) | K8s deployment |
| [Migration from Pingvin Share](migration-pingvin.md) | Migration guide |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Internal server port |
| `EXTERNAL_PORT` | `8080` | External port (for URL generation) |
| `DATA_DIR` | `/data` | Data directory for uploads and database |
| `ADMIN_PASSWORD` | - | Admin password (or use /setup) |
| `TZ` | `Europe/Berlin` | Timezone |

### Network/Tunnel Variables

| Variable | Description |
|----------|-------------|
| `TUNNEL_URL` | Cloudflare Tunnel URL |
| `TAILSCALE_URL` | Tailscale Funnel URL |
| `EASYTIER_IP` | EasyTier network IP |
| `CUSTOM_URL` | Custom/WireGuard URL |
| `LOCAL_IP` | Local network IP |

Pangolin/Newt needs no variable: share links follow the `X-Forwarded-Host`/`Host`
header, so they are generated with the public hostname automatically.

### Optional Features

| Variable | Default | Description |
|----------|---------|-------------|
| `SHARE_ALLOWED_PATHS` | `/DATA,/media,/home` | Paths for file browser |
| `WEBHOOK_URL` | - | Webhook notification URL |
| `WEBHOOK_SECRET` | - | HMAC secret for webhooks |

## Architecture

```
CasaDrop
├── Go Backend (single binary)
│   ├── Gorilla Mux (routing)
│   ├── SQLite (metadata, sessions)
│   └── Prometheus metrics
├── Web Frontend
│   ├── Vanilla JS (no framework)
│   ├── i18n (EN/DE)
│   └── Dark/Light theme
└── Storage
    ├── /data/uploads/ (files)
    ├── /data/thumbnails/ (image previews)
    └── /data/shares.db (SQLite)
```

## Support

- GitHub Issues: [github.com/chicohaager/casadrop/issues](https://github.com/chicohaager/casadrop/issues)
- Discussions: [github.com/chicohaager/casadrop/discussions](https://github.com/chicohaager/casadrop/discussions)
