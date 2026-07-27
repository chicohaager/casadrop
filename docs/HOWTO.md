# CasaDrop — The Complete HowTo

A practical, start-to-finish guide: install it, share your first file, open it to
the internet safely, and automate it. Written for homelab users who want
something running in ten minutes and hardened by the end of the afternoon.

If you want a terse feature/flag reference instead, read the
[README](../README.md). This guide is the walkthrough.

---

## Table of contents

1. [What CasaDrop does](#1-what-casadrop-does)
2. [Before you start](#2-before-you-start)
3. [Install](#3-install)
4. [First run: claiming the admin account](#4-first-run-claiming-the-admin-account)
5. [Sharing files](#5-sharing-files)
6. [Sharing what's already on the server](#6-sharing-whats-already-on-the-server)
7. [Receive links: letting other people send *you* files](#7-receive-links-letting-other-people-send-you-files)
8. [Users, roles and quotas](#8-users-roles-and-quotas)
9. [Reaching CasaDrop from outside your LAN](#9-reaching-casadrop-from-outside-your-lan)
10. [Automating with the API](#10-automating-with-the-api)
11. [Hardening checklist](#11-hardening-checklist)
12. [Backups, data layout and upgrades](#12-backups-data-layout-and-upgrades)
13. [Troubleshooting](#13-troubleshooting)

---

## 1. What CasaDrop does

CasaDrop is a single ~17 MB static Go binary with an embedded SQLite database.
No external database, no CGO, no Redis, no Node runtime. It gives you:

- **Share links** for files and whole folders — with optional password, expiry,
  and download limits, plus QR codes and media streaming.
- **Receive links** — a public upload page you hand to someone else so they can
  send files *to you*, with per-link limits.
- **Share from host** — publish a file or directory that already lives on the
  server, without re-uploading it.
- **Multi-user** — Admin / User / Viewer roles, local accounts, and OIDC/SSO.

### Version note (please read)

This guide describes **2.4.0**, the current release. The image is published for
**linux/amd64 and linux/arm64**, so it runs on a normal x86 box as well as on a
Raspberry Pi 4+ or an ARM NAS. Tags:
[hub.docker.com/r/chicohaager/casadrop/tags](https://hub.docker.com/r/chicohaager/casadrop/tags).

> ⚠️ **Upgrading from 2.3.0 or older? Do it before you add users.** Those
> versions could only ever store a *single* local account — creating a second one
> failed with a bare `HTTP 500 Failed to create user`. 2.4.0 fixes this and
> migrates existing databases automatically on first start. OIDC/SSO accounts and
> the shared admin login were never affected.

---

## 2. Before you start

You need:

- **Docker** with the Compose plugin (`docker compose version` should work), or
  **Go 1.25+** if you build from source.
- **A free port.** The examples use `8080`.
- **An x86-64 or ARM64 host.** The image is published for both, so a Raspberry
  Pi 4+ works out of the box — Docker pulls the right architecture for you.
- About **2 minutes** for a first working instance.

CasaDrop stores everything under one data directory: the SQLite database, the
uploaded files, thumbnails, and its config files. Back that one directory up and
you have backed up CasaDrop. See [§12](#12-backups-data-layout-and-upgrades).

---

## 3. Install

### Option A — kick the tyres (one command)

```bash
docker run -d --name casadrop -p 8080:8080 -v casadrop_data:/data chicohaager/casadrop:latest
```

Then open <http://localhost:8080> and jump to
[First run](#4-first-run-claiming-the-admin-account).

### Option B — Docker Compose (recommended)

Create `docker-compose.yml`:

```yaml
services:
  casadrop:
    image: chicohaager/casadrop:latest
    container_name: casadrop
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      # CasaDrop's own data: database, uploads, thumbnails, config.
      - ./data:/data
      # Optional: host directories you want to publish via "Share from host".
      # Read-only is deliberate — CasaDrop never needs to write here.
      - /mnt/media:/media:ro
    environment:
      - TZ=Europe/Berlin
      # Which host paths the file browser may reach. Default: /DATA,/media,/home
      - SHARE_ALLOWED_PATHS=/media
      # Set this ONLY if CasaDrop sits behind a reverse proxy — see §9.
      # - TRUSTED_PROXY=172.18.0.0/16
    # Sensible container hardening. CasaDrop runs as uid 10001 internally.
    security_opt:
      - no-new-privileges:true
    # No healthcheck needed: the image ships one against /healthz that follows
    # $PORT. Add your own only if you want different timings (see
    # docs/docker-compose.md).
```

Start it:

```bash
docker compose up -d
docker compose logs -f casadrop   # you will need this in a moment
```

Note there is **no `ADMIN_PASSWORD` here on purpose** — see the next section for
why that is the safer of the two options.

### Option C — build from source

For hacking on CasaDrop, or for an architecture we don't publish. No CGO, so the
result is a fully static binary:

```bash
git clone https://github.com/chicohaager/casadrop.git
cd casadrop
CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -trimpath -o casadrop ./cmd/server
./casadrop        # serves on :8080
```

Or build the image locally (works on arm64 too):

```bash
docker build -t casadrop:local .
```

### ZimaOS / CasaOS

See [zimaos.md](zimaos.md) and [INSTALL-ZIMAOS-EN.md](INSTALL-ZIMAOS-EN.md) for
the App-Store flow and the network auto-detection.

---

## 4. First run: claiming the admin account

There are two ways to get an admin login, and the difference matters if your
instance is reachable from the internet.

### Path 1 — the setup wizard (recommended)

If you do **not** set `ADMIN_PASSWORD`, the first request redirects to `/setup`.
To stop a random stranger from claiming your instance before you do, the wizard
demands a **one-time setup token that is printed only to the server log**:

```bash
docker compose logs casadrop | grep "SETUP TOKEN"
```

```
2026/07/13 12:03:17 === CasaDrop SETUP TOKEN: ca1d2cd304cb36cd5f2ec60f839114be ===
2026/07/13 12:03:17 Open /setup and enter this token to create the admin account.
```

Open <http://localhost:8080>, paste the token, choose a password (**minimum 8
characters**, hashed with bcrypt), confirm it, and press **Create Admin
Account**. You are logged in immediately and the token is burned.

Lost the token? Restart the container — a new one is printed on every start
until setup is completed.

### Path 2 — `ADMIN_PASSWORD`

Set `ADMIN_PASSWORD=…` in the environment and the wizard is skipped entirely.
Convenient for a throwaway LAN instance, but be aware:

- the password sits in your compose file and in `docker inspect` output, and
- it is compared **in plain text** — it is never hashed at rest.

Prefer the wizard for anything that is exposed.

### Right after you log in

1. Open **Settings** and enable **2FA (TOTP)** — scan the QR code with any
   authenticator app. Note it protects the *shared admin login* only; per-user
   accounts do not get a TOTP prompt.
2. Check the upload size limit (**Settings**, default **10 GB**, max 100 GB).
3. If you are behind a proxy, set `TRUSTED_PROXY` now — see [§9](#9-reaching-casadrop-from-outside-your-lan).

The navigation you will use throughout this guide: **Upload**, **Share from
host**, **Shares**, **Receive**, **Settings**.

---

## 5. Sharing files

Go to **Upload**, drag files onto the drop zone (or click it), and set the
options:

| Option | Meaning |
|---|---|
| **Password** | Visitors must enter it before they see or download the file. Stored bcrypt-hashed. |
| **Expires in** | 1 hour … 30 days, or **Never (unlimited)**. Default is 24 hours. |
| **Max downloads** | Link dies after N downloads. `0` = unlimited. |

You get back a share URL like `http://your-host:8080/s/34861e37`, plus a QR code
at `/qr/{id}` for phones. Images get thumbnails; audio and video can be streamed
rather than downloaded.

Good to know:

- **Files over 100 MB are uploaded in 8 MB chunks automatically** by the browser,
  so a dropped connection doesn't cost you the whole transfer.
- Up to **50 files** per multi-upload.
- Expired shares are cleaned up by a background job every hour.

To revoke a share early, open **Shares** and delete it. You can also edit an
existing share — including setting it to never expire.

---

## 6. Sharing what's already on the server

**Share from host** (admin only) publishes files that already exist on the
server — a movie on your NAS, a folder of photos — **without copying them**.

Two modes, and the difference is about disk:

- **Link instead of copy** (symlink): no extra disk space, no quota usage. The
  share breaks if you move or delete the original.
- **Copy**: duplicates the file into CasaDrop's uploads directory. Survives the
  original being moved; counts against the owner's storage quota (2.4+).

The browser can only reach paths listed in **`SHARE_ALLOWED_PATHS`** (default
`/DATA,/media,/home`). Anything outside is refused — so mount only what you
intend to expose, and mount it read-only.

**Folder shares** get a browsable page with breadcrumbs, per-file downloads, and
an on-the-fly **ZIP** of the whole tree. The ZIP has an uncompressed budget of
`MAX_FOLDER_ZIP_GB` (default 10 GB) to stop one download from eating the box.

---

## 7. Receive links: letting other people send *you* files

This is the feature people install CasaDrop for. Go to **Receive** → create a
link → hand out the URL (`/r/{id}`). The recipient needs no account and sees a
simple upload page.

Per link you can set:

| Setting | Effect |
|---|---|
| **Password** | Uploader must know it. |
| **Max uploads** | Link stops accepting after N files. |
| **Max file size** | Per-file cap for this link. |
| **Allowed extensions** | Whitelist, e.g. `jpg,png,pdf`. Everything else is rejected outright. |
| **Auto-share** | Every received file automatically gets its own share link. |
| **Webhook** | POST a notification when a file lands. |
| **Expiry** | Defaults to *never* for receive links. |

Since `/r/{id}/upload` is the one endpoint where **anonymous strangers write to
your disk**, it has its own defences:

- **Per-IP rate limit — always on.** `RECEIVE_RATE_PER_HOUR`, default **30
  uploads/hour**.
- **ClamAV malware scanning** (2.4+, opt-in via `CLAMAV_ADDR`). It is
  **fail-closed**: an infected file is rejected with `422`, and if the scanner
  errors or clamd is unreachable the upload is rejected with `503` rather than
  waved through.
- **Proof-of-work throttle** (2.4+, opt-in via `RECEIVE_POW_BITS`). The
  browser must burn CPU on a SHA-256 puzzle before it may upload, which makes
  bulk abuse expensive. Requires a **secure context (HTTPS or localhost)** to
  work in the browser.

Turning both on (source build):

```yaml
    environment:
      - CLAMAV_ADDR=clamav:3310     # unset = scanning disabled
      - CLAMAV_TIMEOUT=30           # seconds
      - RECEIVE_POW_BITS=12         # unset/0 = disabled; higher = harder
      - RECEIVE_RATE_PER_HOUR=30

  clamav:
    image: clamav/clamav:latest
    container_name: clamav
    restart: unless-stopped
```

On startup you will see the feature confirm itself:

```
Receive-upload proof-of-work: ENABLED (12 bits)
```

---

## 8. Users, roles and quotas

> Reminder: on 2.3.0 and older, only **one** local account could be created at
> all. Upgrade first — see the [version note](#version-note-please-read).

### The roles

| Role | Upload / create shares | See own | See all | Delete all | Manage users | Settings |
|---|---|---|---|---|---|---|
| **Admin** | yes | yes | yes | yes | yes | yes |
| **User** | yes | yes | no | no | no | no |
| **Viewer** | **no** (403) | yes | no | no | no | no |

Host browsing and "share from host" are **admin-only**, so a User cannot publish
arbitrary server paths.

### Creating users

**Settings → User Management → add a user** (email, name, role, password,
optional quota). Then they log in at `/login` with **their email + password**.

Leaving the email field **blank** on the login form uses the single shared admin
password instead — that is the backwards-compatible path, and the only one that
prompts for 2FA.

### Storage quotas (2.4+)

Set a per-user quota in GB (0 = unlimited). When the user exceeds it, uploads
fail with `413`. What counts:

- ✅ uploaded files, copied host shares, and **files received through that
  user's receive links** (so an anonymous uploader cannot blow past the link
  owner's limit)
- ❌ symlink shares and in-place folder shares — they reference host files and
  consume no managed storage

Usage and limit are visible via `/api/me` and in the user list.

### SSO / OIDC

Works with Authentik, Keycloak, Authelia, Google, Azure AD, Okta. Users can be
auto-provisioned on first login with a default role. Full walkthrough:
[oidc.md](oidc.md).

> Email notifications are configured **in the admin UI (Settings)**, not through
> environment variables. There are no working `SMTP_*` env vars — ignore any
> older doc that lists them.

---

## 9. Reaching CasaDrop from outside your LAN

Share links follow the host you reach the app through (via `Host` /
`X-Forwarded-Host`), so behind a proxy the links are usually right with no
configuration.

### ⚠️ The one setting people get wrong: `TRUSTED_PROXY`

CasaDrop is **fail-closed** about forwarded headers. If `TRUSTED_PROXY` is
**unset**, `X-Forwarded-For` is *ignored* and the socket peer is used as the
client IP. Behind a reverse proxy that peer is *the proxy* — so:

- every visitor shares one rate-limit bucket,
- one person fat-fingering a password can lock out everybody, and
- the host in generated share links is not trusted from the header.

Fix it by naming your proxy:

```yaml
      - TRUSTED_PROXY=172.18.0.0/16      # your proxy's IP or CIDR
```

Only then are `X-Forwarded-For` / `X-Forwarded-Proto` honoured — and only from
that peer, which is what stops a random client from spoofing its IP.

### Reverse proxy (Traefik / Caddy / nginx)

Ready-made configs: [reverse-proxy.md](reverse-proxy.md), plus compose examples
for [Traefik](docker-compose.traefik.yml) and [Caddy](docker-compose.caddy.yml).

### Tailscale Funnel

A stable HTTPS URL, no Cloudflare account, nothing exposed to the open internet
except what you funnel:

```bash
TAILSCALE_AUTHKEY=tskey-xxx docker compose --profile tailscale up -d
docker exec casadrop-tailscale tailscale funnel --bg 8080
```

### Cloudflare Tunnel

```bash
# Quick tunnel — throwaway *.trycloudflare.com URL
docker compose --profile tunnel up -d

# Named tunnel — fixed hostname
CLOUDFLARE_TUNNEL_TOKEN=xxx docker compose --profile tunnel up -d
```

Heads-up on quick tunnels: switching the primary network **to** Cloudflare in
Settings deliberately mints a **fresh** URL — which **kills the previously shared
link**. That is the intended trade-off for "quickly send someone a thing". If you
need links that keep working, use a named tunnel or Tailscale.

---

## 10. Automating with the API

Don't script against session cookies — mint an **API key**. Keys carry a role of
their own, so give your backup script a `user` key, not an admin one.

Every command below was run against a live instance exactly as written.

```bash
BASE=http://localhost:8080

# 1. Log in once as admin (blank email = the shared admin password)
curl -s -c cookies.txt -X POST $BASE/login \
  -H "Content-Type: application/json" \
  -d '{"email":"","password":"YOUR-ADMIN-PASSWORD"}'
# → {"success":true}

# 2. Mint an API key. The secret is shown EXACTLY ONCE — store it now.
curl -s -b cookies.txt -X POST $BASE/api/api-keys \
  -H "Content-Type: application/json" \
  -d '{"name":"backup-script","role":"user"}'
# → {"id":"787fab5b","key":"cdp_d1a0c7d7…","prefix":"cdp_d1a0c7d7...","role":"user"}
```

From here on, authenticate with the `X-API-Key` header — no cookie, no CSRF
token needed:

```bash
KEY=cdp_d1a0c7d7…

# Upload a file: 24h expiry, dies after 5 downloads
curl -s -X POST $BASE/api/upload -H "X-API-Key: $KEY" \
  -F "file=@report.pdf" -F "expires_in=24" -F "max_downloads=5"
# → {"id":"34861e37", … ,"share_url":"http://localhost:8080/s/34861e37"}

# A share that never expires: expires_in=0
curl -s -X POST $BASE/api/upload -H "X-API-Key: $KEY" \
  -F "file=@logo.png" -F "expires_in=0"

# Anyone can now fetch it, no auth:
curl -s $BASE/d/34861e37 -o report.pdf

# Create a receive link (sizes in BYTES, extensions comma-separated)
curl -s -X POST $BASE/api/receive-links -H "X-API-Key: $KEY" \
  -H "Content-Type: application/json" \
  -d '{"name":"Photos from guests","max_uploads":10,
       "max_file_size":10485760,"allowed_extensions":"jpg,png,pdf"}'
# → {"id":"4bf05c8f", … ,"receive_url":"http://localhost:8080/r/4bf05c8f"}

# …and that link accepts anonymous uploads, enforcing the whitelist:
curl -s -X POST $BASE/r/4bf05c8f/upload -F "file=@holiday.pdf"
# → {"success":true,"file_id":"0676ec05", …}
curl -s -X POST $BASE/r/4bf05c8f/upload -F "file=@notes.txt"
# → File type .txt not allowed
```

`expires_in` is **hours**; omit it and you get 24, pass `0` for *never*.

### Monitoring

```bash
curl -s $BASE/healthz     # liveness — 200 while serving
curl -s $BASE/readyz      # readiness — 503 if the database is unreachable
curl -s -b cookies.txt $BASE/api/metrics   # Prometheus, ADMIN-ONLY (401 otherwise)
```

Note `/api/metrics`, not `/metrics` — and it requires an admin session or admin
API key, so scrape it with credentials.

Full endpoint list: [api.md](api.md).

---

## 11. Hardening checklist

Before you point a public DNS name at this thing:

- [ ] **Use the setup wizard**, not `ADMIN_PASSWORD` (which is stored and
      compared in plain text).
- [ ] **Enable 2FA** for the admin login (Settings).
- [ ] **Set `TRUSTED_PROXY`** to your proxy's CIDR, or your rate limiting and
      lockouts are effectively per-proxy rather than per-visitor ([§9](#9-reaching-casadrop-from-outside-your-lan)).
- [ ] **Terminate TLS** at your proxy/tunnel. HSTS is sent automatically over
      HTTPS, and the proof-of-work throttle needs a secure context.
- [ ] **Mount host directories read-only** and keep `SHARE_ALLOWED_PATHS` as
      narrow as possible.
- [ ] **Give out non-admin accounts.** Viewer for people who only download; User
      for people who upload. Host browsing stays admin-only.
- [ ] **Protect the receive path** if the link is public: keep the per-IP rate
      limit, and consider ClamAV + proof-of-work (2.4+).
- [ ] **Leave the webhook SSRF guards on.** `WEBHOOK_STRICT_SSRF` and
      `STRICT_WEBHOOK_URLS` default to `true` and refuse deliveries to private
      or loopback addresses; only set them to `false` for a deliberate LAN
      receiver.
- [ ] **Set a quota** on each user (2.4+) so one account cannot fill the disk.

Already done for you by the app: bcrypt (cost 12) password hashing, CSRF
protection on the login/setup forms, a strict `script-src 'self'` CSP, rate
limiting (5 login attempts/minute) with a 15-minute lockout after 10 failures,
`SameSite=Strict` session cookies with a 24-hour idle and 7-day absolute
lifetime, and blocked executable uploads.

---

## 12. Backups, data layout and upgrades

Everything lives under `DATA_DIR` (`/data` in Docker):

| Path | What it is |
|---|---|
| `shares.db` | **The** SQLite database — shares, users, receive links, API keys |
| `uploads/` | Uploaded and copied files |
| `uploads/received/{link_id}/` | Files people sent you through receive links |
| `thumbnails/` | Generated image previews (regenerable) |
| `admin_config.json` | Admin password hash, TOTP secret — **secret, mode 0600** |
| `sessions.json` | Active sessions (hashed tokens) |
| `tunnel_config.json` | Network/tunnel settings, upload size limit, extension rules |

### Backup

The database runs in WAL mode, so the safest snapshot is a stopped container:

```bash
docker compose stop casadrop
tar czf casadrop-backup-$(date +%F).tar.gz ./data
docker compose start casadrop
```

Restoring is the reverse: stop, replace `./data`, start.

### Upgrade

```bash
docker compose pull
docker compose up -d
```

Schema migrations run automatically on startup and are logged. Take a backup
first anyway — migrations are one-way.

---

## 13. Troubleshooting

**I lost the setup token.**
Restart the container; a fresh token is printed to the log on every start until
setup completes. `docker compose logs casadrop | grep "SETUP TOKEN"`.

**Everyone keeps getting rate-limited / one person locked out the whole house.**
You are behind a reverse proxy without `TRUSTED_PROXY`, so every request appears
to come from the proxy's IP. See [§9](#9-reaching-casadrop-from-outside-your-lan).

**Share links point at the wrong host or port.**
Set `EXTERNAL_PORT` to the port users actually connect to, and pick the right
primary network in Settings. Behind a proxy, make sure it forwards
`X-Forwarded-Host` *and* that `TRUSTED_PROXY` names it.

**"File too large".**
The default cap is 10 GB. Raise it in **Settings** (up to 100 GB). A per-receive-link
`max_file_size` overrides the global limit for that link.

**Receive uploads all fail with 503.**
`CLAMAV_ADDR` is set but clamd is unreachable. This is deliberate: scanning is
fail-closed, so uploads are rejected rather than accepted unscanned. Check the
clamav container.

**The upload page's proof-of-work never completes.**
It needs a secure context. Serve the page over **HTTPS** (or test on
`localhost`); on a plain-HTTP LAN address the browser's WebCrypto is unavailable
and the per-IP rate limit is your protection instead.

**A user gets 403 on upload.**
They have the **Viewer** role, which is read-only by design. Give them **User**.

**Creating a second user fails with `500 Failed to create user`.**
You are on 2.3.0 or older, which can only store one local account. Upgrade to
2.4.0 — existing databases are migrated automatically on first start.

---

## Getting help

- Issues: [github.com/chicohaager/casadrop/issues](https://github.com/chicohaager/casadrop/issues)
- Discussions: [github.com/chicohaager/casadrop/discussions](https://github.com/chicohaager/casadrop/discussions)
- Contributing: [CONTRIBUTING.md](../CONTRIBUTING.md)

Licensed MIT. Have fun, and don't expose it without reading [§11](#11-hardening-checklist).
