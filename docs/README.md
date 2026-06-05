# CasaDrop Documentation

Self-hosted file sharing for your homelab.

## Quick Links

| Topic | Description |
|-------|-------------|
| [Docker Compose Setup](docker-compose.md) | Standard deployment guide |
| [Reverse Proxy](reverse-proxy.md) | Nginx, Traefik, Caddy configurations |
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
| `PANGOLIN_URL` | Pangolin public URL |
| `ZEROTIER_IP` | ZeroTier network IP |
| `CUSTOM_URL` | Custom/WireGuard URL |
| `LOCAL_IP` | Local network IP |

### Optional Features

| Variable | Default | Description |
|----------|---------|-------------|
| `SHARE_ALLOWED_PATHS` | `/DATA,/media,/home,/mnt` | Paths for file browser |
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
    └── /data/casadrop.db (SQLite)
```

## Support

- GitHub Issues: [github.com/user/casadrop/issues](https://github.com/user/casadrop/issues)
- Discussions: [github.com/user/casadrop/discussions](https://github.com/user/casadrop/discussions)
