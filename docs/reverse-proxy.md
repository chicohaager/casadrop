# Reverse Proxy Configuration

CasaDrop works behind any reverse proxy. Here are configurations for popular options.

## Traefik

### Docker Labels

```yaml
services:
  casadrop:
    image: chicohaager/casadrop:latest
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.casadrop.rule=Host(`share.example.com`)"
      - "traefik.http.routers.casadrop.entrypoints=websecure"
      - "traefik.http.routers.casadrop.tls.certresolver=letsencrypt"
      - "traefik.http.services.casadrop.loadbalancer.server.port=8080"
    networks:
      - traefik

networks:
  traefik:
    external: true
```

### File Provider

```yaml
# traefik/dynamic/casadrop.yml
http:
  routers:
    casadrop:
      rule: "Host(`share.example.com`)"
      entryPoints:
        - websecure
      service: casadrop
      tls:
        certResolver: letsencrypt

  services:
    casadrop:
      loadBalancer:
        servers:
          - url: "http://casadrop:8080"
```

## Caddy

### Caddyfile

```
share.example.com {
    reverse_proxy casadrop:8080
}
```

### With Upload Size Limit

```
share.example.com {
    request_body {
        max_size 10GB
    }
    reverse_proxy casadrop:8080
}
```

## Nginx

### Basic Configuration

```nginx
server {
    listen 443 ssl http2;
    server_name share.example.com;

    ssl_certificate /etc/letsencrypt/live/share.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/share.example.com/privkey.pem;

    client_max_body_size 10G;

    location / {
        proxy_pass http://casadrop:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket support (for future features)
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # Timeout for large uploads
        proxy_connect_timeout 300;
        proxy_send_timeout 300;
        proxy_read_timeout 300;
    }
}

server {
    listen 80;
    server_name share.example.com;
    return 301 https://$server_name$request_uri;
}
```

### Nginx Proxy Manager

1. Add Proxy Host
2. Domain: `share.example.com`
3. Forward Hostname: `casadrop`
4. Forward Port: `8080`
5. Enable SSL with Let's Encrypt
6. Advanced tab - Custom Nginx Configuration:
   ```nginx
   client_max_body_size 10G;
   proxy_connect_timeout 300;
   proxy_send_timeout 300;
   proxy_read_timeout 300;
   ```

## HAProxy

```haproxy
frontend http_front
    bind *:443 ssl crt /etc/haproxy/certs/share.example.com.pem
    acl host_casadrop hdr(host) -i share.example.com
    use_backend casadrop_back if host_casadrop

backend casadrop_back
    server casadrop casadrop:8080 check
    timeout server 300s
    timeout tunnel 300s
```

## Important Headers

CasaDrop reads these headers for proper URL generation:

| Header | Purpose |
|--------|---------|
| `X-Forwarded-Proto` | Detect HTTPS for share links |
| `X-Forwarded-Host` | Original hostname |
| `X-Real-IP` | Client IP for rate limiting |
| `X-Forwarded-For` | Client IP chain |

## Upload Size

For large file uploads, ensure your reverse proxy allows sufficient body size:

| Proxy | Setting |
|-------|---------|
| Nginx | `client_max_body_size 10G;` |
| Traefik | No default limit |
| Caddy | `request_body { max_size 10GB }` |
| HAProxy | No default limit |

## Timeouts

Large file uploads require extended timeouts:

| Proxy | Settings |
|-------|----------|
| Nginx | `proxy_read_timeout 300;` |
| Traefik | `respondingTimeouts.readTimeout: 300s` |
| Caddy | Automatic |
