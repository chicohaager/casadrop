# Migration from Pingvin Share

Pingvin Share was archived in June 2025. CasaDrop provides a lightweight alternative with similar features.

## Feature Comparison

| Feature | Pingvin Share | CasaDrop |
|---------|---------------|----------|
| File Sharing | Yes | Yes |
| Password Protection | Yes | Yes |
| Expiration | Yes | Yes |
| Download Limits | Yes | Yes |
| Reverse Shares | Yes | Yes (Receive Links) |
| Folder Sharing | Limited | Yes |
| Multi-user | Yes | Planned |
| OIDC/SSO | Yes | Planned |
| Email Notifications | Yes | Webhook-based |
| Database | PostgreSQL | SQLite |
| Size | ~300 MB | ~15 MB |
| Stack | Node.js/Next.js | Go |

## Migration Steps

### 1. Export Shares (Manual)

Pingvin Share doesn't provide an export function. Document your active shares:

```bash
# From Pingvin Share database
docker exec pingvin-share-db psql -U pingvin -d pingvin -c \
  "SELECT id, name, expiration FROM share WHERE expiration > NOW();"
```

### 2. Deploy CasaDrop

```bash
mkdir casadrop && cd casadrop

cat > docker-compose.yml << 'EOF'
services:
  casadrop:
    image: ghcr.io/user/casadrop:latest
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
4. For server files: Use "Share from Path" feature

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
# Backup Pingvin data first
docker cp pingvin-share-backend:/app/data ./pingvin-backup

# Stop containers
docker compose -f pingvin-docker-compose.yml down

# Optional: Remove volumes
docker volume rm pingvin-share_data pingvin-share_db
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

Pingvin Share had built-in email. CasaDrop uses webhooks:

```bash
# Configure webhook for n8n/Home Assistant
POST /api/webhook
{
  "url": "https://n8n.example.com/webhook/share-notification",
  "events": ["share.created", "share.downloaded"]
}
```

### Multi-user

Currently CasaDrop is single-admin. Multi-user support is planned.

For now, use OIDC with Authentik/Keycloak (coming soon) or share the admin password among trusted users.

### Theming

CasaDrop has built-in dark/light mode (auto-detects system preference).

Custom branding is not yet supported but planned.

## Getting Help

- GitHub Issues: [github.com/user/casadrop/issues](https://github.com/user/casadrop/issues)
- Discussions: [github.com/user/casadrop/discussions](https://github.com/user/casadrop/discussions)

## Why CasaDrop?

After Pingvin Share was archived, the homelab community needed an alternative:

- **Active Development**: CasaDrop is actively maintained
- **Lightweight**: Single Go binary, no Node.js/PostgreSQL required
- **Self-contained**: SQLite database, no external dependencies
- **Fast**: Low memory footprint, quick startup
- **Homelab-focused**: Built for Docker, supports common integrations
