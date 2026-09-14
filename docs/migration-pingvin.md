# Migration from Pingvin Share

Pingvin Share was archived in June 2025. CasaDrop provides a lightweight alternative with similar features.

## Feature Comparison

| Feature | Pingvin Share | CasaDrop |
|---------|---------------|----------|
| File sharing via link | Yes | Yes |
| Folder sharing | Limited — multi-file shares with ZIP download, no folder upload or tree | Yes — upload a folder, browse it, ZIP on the fly |
| Share files already on the server | No | Yes ("Share from host") |
| Receive links (reverse shares) | Yes | Yes — per-link limits, allowed extensions, webhook |
| Password protection | Yes | Yes |
| Expiration | Yes | Yes (1 hour to 30 days, or never) |
| Download limits | Yes | Yes |
| QR code for a share link | No (QR only for 2FA enrolment) | Yes |
| Media streaming | Preview in the browser | Yes, seekable (Range requests) |
| Image thumbnails | No | Yes |
| Multi-user | Yes | Yes (Admin / User / Viewer roles) |
| OIDC/SSO | Yes | Yes (Authentik, Keycloak, any OIDC provider) |
| LDAP | Yes | No |
| Two-factor (TOTP) | Yes | Yes (admin login) |
| Email | Yes | Yes (SMTP) |
| Webhooks | No | Yes (HMAC-signed) |
| ClamAV scanning | Yes | Yes (receive-link uploads, fail-closed) |
| Per-user storage quota | No (one global maximum share size) | Yes |
| Storage backends | Local, S3 | Local (S3 on the roadmap) |
| Public access helpers | No | Tailscale, Cloudflare Tunnel, Pangolin, Taildrop |
| Prometheus metrics | No | Yes |
| Custom branding (logo, name) | Yes | Planned |
| Languages | 29 | 14 |
| API documentation | OpenAPI/Swagger | Markdown ([api.md](api.md)) |
| Database | SQLite | SQLite |
| Docker image (amd64, compressed) | ~186 MB | ~17 MB |
| Stack | Node.js (NestJS + Next.js) | Go, single static binary |

## Migration Steps

### 1. Export Shares (Manual)

Pingvin Share doesn't provide an export function. Document your active shares.
Pingvin keeps its metadata in an SQLite file inside the `./data` folder you
mounted into the container:

```bash
sqlite3 ./data/pingvin-share.db "SELECT id, name, expiration FROM Share;"
```

### 2. Deploy CasaDrop

```bash
mkdir casadrop && cd casadrop

cat > docker-compose.yml << 'EOF'
services:
  casadrop:
    image: chicohaager/casadrop:latest
    container_name: casadrop
    ports:
      - "3000:8080"  # Same port as Pingvin
    volumes:
      - ./data:/data
      - /path/to/files:/files:ro
    environment:
      - TZ=Europe/Berlin
    restart: unless-stopped
EOF

docker compose up -d
```

### 3. Configure Admin Password

Visit `http://localhost:3000/setup` to set your admin password.

### 4. Recreate Shares

Use the CasaDrop web interface to recreate your important shares:

1. Open CasaDrop at `http://localhost:3000`
2. Login with admin password
3. For uploaded files: Upload them again
4. For server files: Use "Share from host"

### 5. Update Reverse Proxy

If using a reverse proxy, update the backend:

```nginx
# Before (Pingvin)
proxy_pass http://pingvin-share:3000;

# After (CasaDrop)
proxy_pass http://casadrop:8080;
```

### 6. Stop Pingvin Share

```bash
# Backup Pingvin data first — it is the ./data bind mount from its compose file
cp -r ./data ./pingvin-backup

# Stop the container
docker compose -f pingvin-docker-compose.yml down
```

## URL Compatibility

CasaDrop uses different URL patterns:

| Type | Pingvin Share | CasaDrop |
|------|---------------|----------|
| Share Page | `/s/{id}` | `/s/{id}` (same!) |
| Download | `/api/shares/{id}/files/{fileId}` | `/d/{id}` |
| Reverse Share | `/upload/{id}` | `/r/{id}` |

The share page URL is compatible, so existing links with `/s/` prefix will work.

## Feature Differences

### Email Notifications

Both have built-in email. In CasaDrop the SMTP server is configured by the
admin in the settings (or through `GET`/`POST /api/smtp`, with
`POST /api/smtp/test` to check the connection); it is used to send a share by
e-mail and to notify you of downloads.

On top of that, CasaDrop can call a webhook for n8n, Home Assistant and the
like. Each event is its own switch — there is no `events` array:

```bash
POST /api/webhook
{
  "enabled": true,
  "url": "https://n8n.example.com/webhook/share-notification",
  "on_download": true,
  "on_limit_reached": true,
  "on_expire": false,
  "secret": "hmac-secret"
}
```

Deliveries are signed with HMAC-SHA256 when a secret is set. See
[api.md](api.md#webhooks) for the details.

### Multi-user

CasaDrop has user accounts with three roles — Admin, User and Viewer — with
local e-mail + password login, and OIDC/SSO against Authentik, Keycloak or any
other OIDC provider (`OIDC_ENABLED=true` plus issuer, client ID and secret; see
the README for the compose snippet).

### Theming

CasaDrop has built-in dark/light mode (auto-detects system preference).

Custom branding (your own logo and name) is planned, not yet available.

## Getting Help

- GitHub Issues: [github.com/chicohaager/casadrop/issues](https://github.com/chicohaager/casadrop/issues)
- Discussions: [github.com/chicohaager/casadrop/discussions](https://github.com/chicohaager/casadrop/discussions)

## Why CasaDrop?

After Pingvin Share was archived, the homelab community needed an alternative:

- **Active Development**: CasaDrop is actively maintained
- **Lightweight**: a single 17 MB Go binary, no Node.js runtime — the Docker image is about a tenth the size
- **Self-contained**: SQLite database, no external dependencies
- **Fast**: Low memory footprint, quick startup
- **Homelab-focused**: Built for Docker, supports common integrations
