# API Reference

> Field names and response shapes below were captured from a running
> instance. `tests/verify-api-md.py` replays them against a live server and
> fails if a documented key is no longer sent — run it after changing any
> handler's response.

CasaDrop provides a REST API for all operations. Authentication is required for most endpoints.

## Authentication

When `ADMIN_PASSWORD` is set or setup is completed, requests to protected
endpoints must carry one of:

| Credential | Header / cookie |
|------------|-----------------|
| Session | `Cookie: casadrop_session=<token>` |
| Session as bearer | `Authorization: Bearer <token>` |
| API key | `X-API-Key: <key>` |

Endpoints marked **admin only** additionally require the Admin role; a `User` or
`Viewer` gets 403. Mutating `/api/*` calls made with a **cookie** also pass a
cross-site guard, so send them same-origin or use a bearer token / API key.

### Check Auth Status

```bash
GET /api/auth/status
```

Response:
```json
{
  "authenticated": true,
  "setupRequired": false,
  "envPassword": true,
  "sessionExpiry": "2026-09-12T19:10:55+02:00"
}
```

`envPassword` is true when the admin password comes from `ADMIN_PASSWORD`
(the setup wizard is then skipped). `sessionExpiry` is only present while
authenticated.

### Login

Form-encoded and JSON are both accepted:

```bash
POST /login
Content-Type: application/x-www-form-urlencoded

email=user@example.com     # optional — omit for the shared admin password
password=your-password
totp=123456                # only when admin 2FA is enabled
```

```bash
POST /login
Content-Type: application/json

{"password": "your-password"}
```

Returns `{"success": true}` and sets the `casadrop_session` cookie.

## File Sharing

### Upload Single File

```bash
POST /api/upload
Content-Type: multipart/form-data

file: <binary>
password: optional-password
expires_in: 24            # HOURS as an integer, not "24h"
max_downloads: 10         # 0 = unlimited
```

**Expiry is an integer number of hours**, and the omitted case differs from the
zero case:

| `expires_in` | Result |
|--------------|--------|
| omitted or empty | default 24 hours |
| `0` (sent explicitly) | never expires (`9999-12-31T23:59:59Z`) |
| `168` | 7 days |
| above 876000 | clamped to 100 years |

Response:
```json
{
  "id": "18ed942f",
  "file_name": "document.pdf",
  "file_size": 1048576,
  "mime_type": "application/pdf",
  "has_password": false,
  "expires_at": "2026-09-12T19:11:03.086834588+02:00",
  "created_at": "2026-09-11T19:11:03.086834688+02:00",
  "downloads": 0,
  "max_downloads": 10,
  "share_url": "https://share.example.com/s/18ed942f"
}
```

`file_name` carries the *original* upload name; the stored name on disk is an
internal detail and is not returned.

### Upload Multiple Files

```bash
POST /api/upload/multi
Content-Type: multipart/form-data

files: <binary>           # repeat the field; "files[]" is REJECTED with 400
files: <binary>
password: optional-password
expires_in: 24
max_downloads: 0
```

Response:
```json
{
  "shares": [ { "id": "b64c611e", "file_name": "document.pdf", "...": "as above" } ],
  "success": 2,
  "failed": 0
}
```

`errors` is present only when `failed > 0`.

### Share From Server Path

Share existing files without copying:

Admin only. The path must resolve inside `SHARE_ALLOWED_PATHS`.

```bash
POST /api/share-from-path
Content-Type: application/json

{
  "path": "/media/movies/movie.mp4",
  "password": "",
  "expires_in": 168,
  "max_downloads": 0,
  "use_symlink": true
}
```

`use_symlink: true` references the host file in place (no copy, and it does not
count against the owner's storage quota); `false` copies it into the uploads
directory. Response is the same share object as `/api/upload`.

### List Shares

```bash
GET /api/shares
```

Returns a **bare JSON array**, not an object with a `shares` key. Admins see
every share; other roles see only their own.

```json
[
  {
    "id": "bf74d1a2",
    "file_name": "document.pdf",
    "file_size": 1048576,
    "mime_type": "application/pdf",
    "has_password": false,
    "expires_at": "2026-09-12T17:11:03Z",
    "created_at": "2026-09-11T17:11:03Z",
    "downloads": 5,
    "max_downloads": 10,
    "share_url": "https://share.example.com/s/bf74d1a2"
  }
]
```

Folder shares carry `"is_directory": true`; it is omitted for file shares.

### Get Share Info

```bash
GET /api/shares/{id}
```

### Delete Share

```bash
DELETE /api/shares/{id}
```

## Download

### Download File (Public)

```bash
GET /d/{id}
GET /d/{id}?password=xxx
```

### Stream File (Public)

For media streaming with range support:

```bash
GET /stream/{id}
GET /stream/{id}?password=xxx
```

### Share Page (Public)

```bash
GET /s/{id}
```

## Folder Sharing

### Share Folder

Admin only, same `SHARE_ALLOWED_PATHS` restriction as above.

```bash
POST /api/share-folder
Content-Type: application/json

{
  "path": "/media/photos/vacation",
  "password": "",
  "expires_in": 168,
  "max_downloads": 0
}
```

Response is a share object with `"is_directory": true` and
`"mime_type": "application/x-directory"`.

### Get Folder Contents (Public)

```bash
GET /folder/{id}/contents?path=/subdir
GET /folder/{id}/contents?password=xxx
```

### Download Folder File (Public)

```bash
GET /folder/{id}/download?path=/photo.jpg
```

### Download Folder as ZIP (Public)

```bash
GET /folder/{id}/zip
GET /folder/{id}/zip?path=/subdir
```

## Receive Links

Receive links allow others to upload files to you.

### Create Receive Link

```bash
POST /api/receive-links
Content-Type: application/json

{
  "name": "Project Files",
  "password": "",
  "expires_in": 168,
  "max_uploads": 10,
  "max_file_size": 104857600,
  "allowed_extensions": ".pdf,.docx",
  "auto_share": true,
  "webhook_url": "https://hooks.example.com/received"
}
```

`expires_in` is in hours (`0` = never). `max_uploads` and `max_file_size`
(bytes) use `0` for unlimited. `auto_share` turns each arriving file into a
share automatically. `webhook_url` is validated against the SSRF guard at
creation time — a private or loopback target is refused with 400.

Response:
```json
{
  "id": "e7a00ee4",
  "name": "Project Files",
  "has_password": false,
  "max_uploads": 10,
  "auto_share": true,
  "created_at": "2026-09-11T19:11:32.958166912+02:00",
  "expires_at": "2026-09-18T19:11:32.958166672+02:00",
  "receive_url": "https://share.example.com/r/e7a00ee4"
}
```

### List Receive Links

```bash
GET /api/receive-links
```

Returns a **bare JSON array**. List entries carry two fields the create response
does not: `current_uploads` and `files_count`, plus `total_size`.

```json
[
  {
    "id": "e7a00ee4",
    "name": "Project Files",
    "has_password": false,
    "max_uploads": 10,
    "current_uploads": 0,
    "files_count": 0,
    "total_size": 0,
    "auto_share": true,
    "created_at": "2026-09-11T17:11:32Z",
    "expires_at": "2026-09-18T17:11:32Z",
    "receive_url": "https://share.example.com/r/e7a00ee4"
  }
]
```

### Get Receive Link Details

```bash
GET /api/receive-links/{id}
```

### Delete Receive Link

```bash
DELETE /api/receive-links/{id}
```

### Get Received Files

```bash
GET /api/receive-links/{id}/files
```

Bare JSON array:

```json
[
  {
    "id": "22815181",
    "receive_link_id": "e7a00ee4",
    "file_name": "22815181.pdf",
    "original_name": "document.pdf",
    "file_size": 4,
    "mime_type": "application/pdf",
    "uploader_ip": "203.0.113.44",
    "uploader_agent": "Mozilla/5.0 ...",
    "created_at": "2026-09-11T17:11:33Z",
    "share_id": "6c39cffd"
  }
]
```

`share_id` is present only when the link has `auto_share` enabled.
`uploader_ip` is the socket peer unless `TRUSTED_PROXY` is configured.

### Download Received File

```bash
GET /api/receive-links/{id}/files/{fileId}
```

### Upload to Receive Link (Public)

```bash
POST /r/{id}/upload
Content-Type: multipart/form-data

file: <binary>
password: optional-password
```

Response:
```json
{
  "success": true,
  "file_id": "22815181",
  "file_name": "document.pdf",
  "file_size": 4,
  "share_id": "6c39cffd",
  "share_url": "https://share.example.com/s/6c39cffd"
}
```

`share_id` / `share_url` appear only when the link has `auto_share` enabled.
This is the only endpoint anonymous strangers can write to, so it carries extra
controls: a per-IP rate limit (`RECEIVE_RATE_PER_HOUR`, default 30), optional
ClamAV scanning (`CLAMAV_ADDR`) and an optional proof-of-work
(`RECEIVE_POW_BITS`, challenge at `GET /r/{id}/challenge`).

## File Browser

Both endpoints below are **admin-only** — they read the host filesystem under
`SHARE_ALLOWED_PATHS`.

### Browse Server Files

```bash
GET /api/browse?path=/media
```

Response (field names as sent by the server — snake_case, no `modTime`):
```json
{
  "path": "/media",
  "parent": "/",
  "entries": [
    {
      "name": "movies",
      "path": "/media/movies",
      "is_dir": true,
      "size": 4096,
      "is_image": false
    },
    {
      "name": "photo.jpg",
      "path": "/media/photo.jpg",
      "is_dir": false,
      "size": 2048576,
      "is_image": true
    }
  ]
}
```

`is_image` marks the entries the preview endpoint below can render. It is
decided from the extension (`.jpg`, `.jpeg`, `.png`, `.gif`, `.bmp`, `.tif`,
`.tiff`) so that listing a directory never opens a file.

### Preview a Server File

```bash
GET /api/browse/thumbnail?path=/media/photo.jpg
```

Returns a JPEG thumbnail (max 300×300, cached under `data/thumbnails/`) for a
file that is already on the host — the browse dialog uses it to show previews
instead of a plain file list. `/thumbnail/{id}` only covers files that exist as
a share.

| Status | Meaning |
|--------|---------|
| 200 | `image/jpeg` thumbnail, `Cache-Control: private, max-age=60` |
| 400 | Path invalid, not absolute, or not a regular file |
| 403 | Path outside `SHARE_ALLOWED_PATHS` (checked after symlink resolution) |
| 415 | Extension not supported, or the file does not decode (e.g. `.webp` — no pure-Go decoder available) |

## Network & Configuration

### Get Network Info

```bash
GET /api/network
```

Readable by any signed-in user. Field names are lowerCamelCase with a
**lowercase `p`/`i` tail** (`localIp`, `tunnelUrl`) — not `localIP`/`tunnelURL`.
There is no `zerotierIP` (EasyTier replaced ZeroTier: the field is `easytierIp`)
and no `pangolinURL` (Pangolin needs no field — links follow the request host).

```json
{
  "localIp": "192.168.10.100",
  "tunnelUrl": "https://xxx.trycloudflare.com",
  "tailscaleUrl": "https://casadrop.tailnet-example.ts.net",
  "easytierIp": "10.147.99.50",
  "customUrl": "",
  "port": "8080",
  "primaryNetwork": "local",
  "primaryUrl": "http://192.168.10.100:8080",
  "maxFileSizeGB": 10,
  "networks": {
    "local":      { "enabled": true, "url": "192.168.10.100", "detected": "192.168.10.100" },
    "cloudflare": { "enabled": true, "url": "", "detected": "" },
    "tailscale":  { "enabled": true, "url": "", "detected": "" },
    "easytier":   { "enabled": true, "url": "", "detected": "" },
    "custom":     { "enabled": true, "url": "", "detected": "" }
  }
}
```

The five flat fields are legacy and are **emitted only for networks that are
enabled**; `networks` is the authoritative per-network view (`url` = configured
value or detected fallback, `detected` = what auto-detection found).
`primaryNetwork` is one of `local`, `cloudflare`, `tailscale`, `easytier`,
`custom` — never `tunnel`. The size limit is `maxFileSizeGB` (gigabytes), not
`maxFileSize` in bytes.

### Get/Set Tunnel Config

Admin only. **`GET` wraps the configuration in a `config` key; `POST` takes the
inner object unwrapped.**

```bash
GET /api/tunnel
```
```json
{
  "config": {
    "enabled": false,
    "url": "",
    "cloudflareUrl": "",
    "tailscaleUrl": "",
    "easytierIp": "",
    "customUrl": "",
    "localIp": "",
    "primaryNetwork": "",
    "maxFileSizeGB": 0,
    "allowedExtensions": "",
    "blockedExtensions": ""
  },
  "isExternal": false,
  "url": ""
}
```

```bash
POST /api/tunnel
Content-Type: application/json

{
  "primaryNetwork": "cloudflare",
  "cloudflareUrl": "https://example.trycloudflare.com",
  "cloudflareEnabled": true,
  "maxFileSizeGB": 10
}
```

Per-network on/off flags are `localEnabled`, `cloudflareEnabled`,
`tailscaleEnabled`, `easytierEnabled`, `customEnabled` (omitted = enabled).
Cloudflare and Tailscale URLs must be `https://`; `easytierIp` must be a bare IP
address. Switching `primaryNetwork` **to** `cloudflare` mints a fresh
quick-tunnel URL and the response then carries `"cloudflareRotating": true`.

## Webhooks

### Get/Set Webhook Config

Admin only. There is no `events` array — each event is its own boolean, and the
names are `download` / `limit_reached` / `expire`, not `share.*`.

```bash
GET /api/webhook
```
```json
{
  "enabled": true,
  "url": "https://hooks.example.com/casadrop",
  "on_download": true,
  "on_expire": false,
  "on_limit_reached": true,
  "secret_set": true
}
```

The stored secret is **never** returned; `secret_set` only says whether one
exists.

```bash
POST /api/webhook
Content-Type: application/json

{
  "enabled": true,
  "url": "https://hooks.example.com/casadrop",
  "on_download": true,
  "on_limit_reached": true,
  "on_expire": true,
  "secret": "hmac-secret"
}
```

The write **merges**: any field you omit keeps its stored value. An omitted
`secret` therefore keeps the stored secret (important, because `GET` never
returns it); send `"secret": ""` to remove one. Enabling with an empty `url` is
refused with 400.

Deliveries carry `X-Webhook-Event` and, when a secret is set,
`X-Webhook-Signature: sha256=<hex>` — HMAC-SHA256 over the **raw request body**.
Per-receive-link webhooks (`webhook_url` above) send the `file_received` event
and are **not** signed.

### Test Webhook

```bash
POST /api/webhook/test
```

Queues a synthetic `download` delivery. Returns 400 `Webhook not configured`
when disabled or without a URL. A 200 means the notification was **queued**, not
delivered — delivery is fire-and-forget with no retries; confirm at the receiver
or in `docker logs casadrop`.

## Statistics

### Get Stats

```bash
GET /api/stats
```

Response (snake_case; there is no `activeShares` — the fourth and fifth fields
are `protected_shares` and `expiring_soon`):

```json
{
  "total_shares": 42,
  "total_downloads": 1337,
  "total_size": 10737418240,
  "protected_shares": 7,
  "expiring_soon": 3
}
```

`expiring_soon` counts shares expiring within 24 hours. Admins get totals over
all shares; other roles only over their own.

## Utilities

### Generate QR Code

```bash
GET /qr/{id}
```

Returns PNG image.

### Get Thumbnail

```bash
GET /thumbnail/{id}
```

Returns a JPEG thumbnail for a share whose `mime_type` is `image/*` (SVG
excluded). **Video is not supported** — a non-image share gets 400. Decoders
registered: JPEG, PNG, GIF, BMP, TIFF. `.webp` has no pure-Go decoder in the
dependency set and returns 415.

## Not covered here

These endpoint groups exist and are wired in `internal/routes/routes.go`, but
this reference does not document them yet — see `CLAUDE.md` for the route list:
user management (`/api/users`, `/api/me`), admin 2FA (`/api/admin/2fa*`),
API keys (`/api/api-keys`), Taildrop (`/api/taildrop/*`), OIDC
(`/auth/oidc/*`, `/api/auth/oidc/*`), SMTP (`/api/smtp*`), chunked upload
(`/api/upload/chunk/*`) and the health probes (`/healthz`, `/readyz`).

## Prometheus Metrics

Admin-only — requires an admin session cookie or an admin API key.

```bash
GET /api/metrics
```

Metric names use the **`zima_` prefix**, a leftover from the rename to CasaDrop;
no `casadrop_*` metric exists. Alongside these, the endpoint serves the standard
Go runtime and process collectors.

| Metric | Type |
|--------|------|
| `zima_uploads_total` | counter (labelled) |
| `zima_downloads_total` | counter |
| `zima_upload_bytes_total` | counter |
| `zima_download_bytes_total` | counter |
| `zima_shares_created_total` | counter |
| `zima_shares_deleted_total` | counter |
| `zima_shares_expired_total` | counter |
| `zima_active_shares` | gauge |
| `zima_storage_used_bytes` | gauge |
| `zima_http_requests_total` | counter (labelled) |
| `zima_http_request_duration_seconds` | histogram (labelled) |
| `zima_auth_failures_total` | counter (labelled) |
| `zima_rate_limit_hits_total` | counter (labelled) |

`zima_uploads_total`, `zima_auth_failures_total` and `zima_rate_limit_hits_total`
are labelled vectors with no zero-initialised series, so they are **absent from
the output until the first matching event**. Do not treat a missing series as a
broken exporter.
