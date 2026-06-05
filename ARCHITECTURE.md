# CasaDrop — Architecture

*Last updated: 2026-04-13 (post three-pass security review)*

This document describes how CasaDrop is put together: package boundaries,
request lifecycle, data model, background processes, and deployment
topologies. It is the map you'd want if you had to debug a production
incident at 3am, or onboard a new contributor in under an hour.

For the feature list see `README.md`. For security posture see `SECURITY.md`.
For the change history see `CHANGELOG.md`. For an overview of the latest
review see `SESSION_NOTES.md`.

---

## 1. Design goals

CasaDrop is a **single-binary, self-hosted file sharing service for
homelabs**. Those three words drive almost every architectural decision:

1. **Single binary.** One `casadrop` executable. No language runtime on
   the host. No sidecars. No external service dependencies at runtime
   — not even an SMTP daemon (outbound only, optional).
2. **Self-hosted.** Operators run the binary on boxes they own,
   often behind one or more tunnels (Cloudflare, Tailscale Funnel,
   Pangolin/Newt, ZeroTier, plain LAN). The app must render consistent
   share URLs regardless of which ingress a request arrived on.
3. **Homelab.** Single admin is the common case. Multi-user exists
   (`admin`/`user`/`viewer` roles) but is opt-in. Threat model assumes
   mostly-trusted users on a mostly-trusted network with occasional
   exposure to the public internet through a tunnel.

Consequences:
- **SQLite, not Postgres.** Zero-ops, single file, WAL mode for
  concurrent reads. Pure-Go driver (`modernc.org/sqlite`) since v2.2 so
  the binary doesn't need glibc/CGO.
- **Defaults favour operator ergonomics, not paranoia.** e.g. webhooks
  to private IPs are allowed unless you opt in to `STRICT_WEBHOOK_URLS`.
- **SameSite=Strict + server-side sessions, not JWT.** Sessions are
  stored in-process with a JSON persistence file so they survive
  restarts. Token-based CSRF is redundant given the cookie flags.

---

## 2. Package map

```
zima-share/                          ← module name (legacy; binary is "casadrop")
├── cmd/server/                      ← Thin main() — wires deps, starts srv, drains workers
│   └── main.go
├── internal/
│   ├── auth/                        ← OIDC provider, OIDC handlers, user provisioning
│   │   ├── auth.go                  ← HashPassword/CheckPassword, BcryptCost constant
│   │   ├── oidc.go                  ← Provider (state store + cleanup loop + Stop())
│   │   ├── oidc_handlers.go         ← LoginHandler/CallbackHandler/LogoutHandler
│   │   └── user_service.go          ← FindOrCreateOIDCUser (with per-issuer rate limit)
│   ├── config/                      ← Tunnel/app config file I/O
│   ├── email/                       ← Outbound SMTP service (optional)
│   ├── handlers/                    ← HTTP handler implementations (no routing logic here)
│   │   ├── handlers.go              ← Upload/share/misc
│   │   ├── download.go              ← /s/{id}, /d/{id}, /stream/{id}
│   │   ├── folder.go                ← Folder share + zip streaming
│   │   ├── receive.go               ← Receive-link upload flow
│   │   ├── users.go                 ← /api/users CRUD
│   │   ├── upload.go                ← Chunked + single-shot uploads
│   │   ├── email.go                 ← /api/smtp + expiry notifier (Stop())
│   │   └── misc.go                  ← Webhook config, stats, browse, QR
│   ├── metrics/                     ← Prometheus middleware + counters
│   ├── middleware/                  ← Session auth, rate limit, security headers, CSRF
│   │   ├── auth.go                  ← AdminAuth (sessions, lockout, CSRF, Stop())
│   │   ├── ratelimit.go             ← Generic IP-bucket rate limiter (Stop())
│   │   ├── security_headers.go
│   │   └── maxbody.go
│   ├── models/                      ← Data models — pure structs, no behaviour
│   ├── preview/                     ← Image/video thumbnail generation
│   ├── routes/                      ← HTTP router — single source of truth for URL mapping
│   │   ├── routes.go                ← routes.New(Deps) → *mux.Router
│   │   └── routes_test.go           ← TestAuthFlow integration test
│   ├── storage/                     ← Persistence layer
│   │   ├── interface.go             ← StorageBackend interface
│   │   ├── sqlite.go                ← SQLiteStorage impl (modernc.org/sqlite)
│   │   ├── storage.go               ← Public Storage wrapper
│   │   └── migrate_users.go         ← v2.1 schema migration
│   ├── utils/                       ← Cross-cutting helpers (URL/IP, filename sanitise)
│   └── webhook/                     ← Outbound webhook dispatcher + HMAC signing
├── web/
│   ├── static/                      ← CSS, JS, i18n assets
│   └── templates/                   ← html/template files served to browsers
├── scripts/
│   └── entrypoint.sh                ← Auto-detection wrapper for the alpine image
├── Dockerfile                       ← Default image (alpine runtime, retains entrypoint)
├── Dockerfile.scratch               ← Minimal scratch image (no auto-detect)
└── docker-compose*.yaml
```

### Dependency direction

```
      ┌──────────────────────────┐
      │       cmd/server         │
      └────────────┬─────────────┘
                   │
        ┌──────────▼──────────┐
        │   internal/routes   │
        └──────────┬──────────┘
                   │ depends on
        ┌──────────▼─────────────────────┐
        │ internal/handlers   middleware │
        │                     auth       │
        └──────────┬─────────────────────┘
                   │ depends on
        ┌──────────▼──────────┐
        │ internal/storage    │
        │ internal/models     │
        │ internal/utils      │
        │ internal/webhook    │
        └─────────────────────┘
```

- `internal/middleware/auth.go` deliberately does **not** import
  `internal/auth` to avoid an import cycle. It duplicates the bcrypt
  cost constant (12) inline and documents why.
- `internal/routes` is the **only** package allowed to know the full
  URL map. Tests live next to it so router regressions are caught
  immediately.

---

## 3. Request lifecycle

All HTTP traffic enters through `http.Server{Handler: router}` where
`router` is the `*mux.Router` returned by `routes.New()`.

### 3.1 Middleware chain

Every request passes through:

```
┌─────────────────────────────────────────────┐
│ metrics.Middleware                          │  Prometheus counters
├─────────────────────────────────────────────┤
│ middleware.SecurityHeaders                  │  CSP, X-Frame, HSTS, …
├─────────────────────────────────────────────┤
│ adminAuth.Middleware  (protected subtree)   │  Session cookie → context
├─────────────────────────────────────────────┤
│ middleware.MaxBodySizeSkipPaths (api/*)     │  1 MB cap on JSON APIs
├─────────────────────────────────────────────┤
│ RequireCanCreateShares / RequireAdmin       │  Per-route role guards
└─────────────────────────────────────────────┘
                     │
                     ▼
           handler-specific logic
```

Upload endpoints are exempt from the 1 MB body cap via
`MaxBodySizeSkipPaths(1<<20, "/api/upload", "/api/upload/multi", "/api/upload/chunk")`.
They enforce their own caps via `http.MaxBytesReader(w, r.Body, configuredMaxSize)`
plus an 8 MB multipart memory budget before spilling to disk.

### 3.2 Request flow — file download

```
GET /d/{id}
    │
    ▼
rateLimitDownload (10 req/min/IP)
    │
    ▼
h.DownloadFile
    ├─ storage.Get(id)                 ← share metadata
    ├─ optional password check         ← bcrypt, with per-share-per-IP lockout
    ├─ storage.IncrementDownloads(id)  ← atomic UPDATE with limit check
    ├─ webhook.NotifyDownload(...)     ← async via go h.webhook.Send(...)
    └─ http.ServeFile(w, r, filePath)  ← supports Range, gives Accept-Ranges
```

The download-counter is authoritative via a single atomic SQL statement:
`UPDATE shares SET downloads = downloads + 1 WHERE id = ? AND (max_downloads = 0 OR downloads < max_downloads)`.
`rows.RowsAffected()` tells us whether the limit was reached.

### 3.3 Request flow — chunked upload

Three-phase protocol:

1. `POST /api/upload/chunk/init` — create server-side `ChunkUpload`
   record in an in-memory map guarded by `chunkUploadsMu`. Returns
   `uploadId`.
2. `POST /api/upload/chunk/{uploadId}?index=N` — stream chunk N into
   `$TEMP/chunk_N`. Updates `ChunksReceived[N]` under the mutex.
3. `POST /api/upload/chunk/{uploadId}/finalize` — snapshot received
   count under the mutex, concatenate chunks into the destination
   file, create share, delete temp directory.

A package-level goroutine (`startChunkCleanupWorker`) removes upload
records older than 24 hours. It is started with `sync.Once`, so even
tests won't spawn extras, and it has its own `chunkStopCh` for
graceful shutdown via `StopChunkCleanupWorker()`.

### 3.4 Request flow — OIDC login

```
GET /auth/oidc/login
    ├─ generateRandomString → state (cryptographically random)
    ├─ provider.states[state] = OIDCState{nonce, returnTo, expiresAt}
    └─ 302 → IdP authorize URL

GET /auth/oidc/callback?code=…&state=…
    ├─ verify state exists + not expired
    ├─ exchange code → id_token
    ├─ verifier.Verify → claims
    ├─ UserService.FindOrCreateOIDCUser(...)
    │     ├─ lookup by (oidc_subject, oidc_issuer)
    │     ├─ fallback lookup by email
    │     └─ auto-provision (rate-limited: 20/hour/issuer)
    ├─ AdminAuth.CreateSessionForUser(...)
    └─ 302 → / (or returnTo from state)
```

The `provider.states` map is cleaned by `cleanupLoop()` every
`CleanupInterval` (5 min). `Stop()` exits the loop cleanly.

---

## 4. Data model

See `CLAUDE.md` for the full SQL schema. Summary of the important
relationships:

```
users ──┬──< shares >──┬── folder_contents
        │              │
        │              └── email_transfers
        │
        └──< receive_links >── received_files
```

- **`users`** — local + OIDC-provisioned users. `UNIQUE(oidc_subject,
  oidc_issuer)` ties external identities.
- **`shares`** — every user-visible share (file, folder, or symlink).
  `user_id` is nullable so pre-v2.1 rows remain addressable.
- **`folder_contents`** — flattened folder listing for browse UI.
  `FOREIGN KEY shares(id) ON DELETE CASCADE`.
- **`receive_links`** — upload buckets. `auto_share=1` auto-publishes
  received files as shares.
- **`received_files`** — child rows of a `receive_links` entry.
  Optional `share_id` FK if auto-share triggered.
- **`oidc_config`** — exactly one row (`CHECK (id = 1)`), stores
  persisted OIDC provider settings that override environment.
- **`api_keys`** — optional Bearer-token auth for CLI/scripts.

SQLite is opened in WAL mode with `foreign_keys=ON`. The DSN literal
lives in `internal/storage/sqlite.go`:

```go
dsn := "file:" + dbPath +
    "?_pragma=journal_mode(WAL)" +
    "&_pragma=busy_timeout(5000)" +
    "&_pragma=synchronous(NORMAL)" +
    "&_pragma=foreign_keys(ON)"
db, err := sql.Open("sqlite", dsn)
```

Note the syntax change from `mattn/go-sqlite3`: modernc uses
`_pragma=<name>(<value>)`, not `_journal_mode=WAL`. `time.Time`
bindings also differ — see the explicit string format in
`GetExpiringSoon`.

---

## 5. Background workers

Every long-lived goroutine exposes a `Stop()` method so `main.go` can
drain it after `http.Server.Shutdown()` returns. Drain order matters:
we stop the HTTP server **first** so that in-flight requests can still
acquire rate-limiter tokens and touch sessions during their final
milliseconds, and then we stop the background workers so the process
can exit cleanly.

| Worker | File | Period | Purpose |
|---|---|---|---|
| `AdminAuth` cleanup | `middleware/auth.go` | 15 min | Expire sessions, CSRF tokens, failed-attempt records |
| `RateLimiter` cleanup | `middleware/ratelimit.go` | configurable window | Expire IP buckets |
| `Provider` OIDC cleanup | `auth/oidc.go` | 5 min | Expire pending OIDC states |
| `EmailHandler` expiry notifier | `handlers/email.go` | 1 hour | Send "share expires in 24h" emails |
| `chunkCleanupWorker` | `handlers/upload.go` | 15 min | Remove chunked-upload temp dirs older than 24h |
| Storage cleanup of expired shares | `storage/sqlite.go` | 1 hour | Delete expired share rows + files |

The expired-share cleanup is the only worker started **inside**
`storage` rather than `main`. It reads the ctx off the storage struct
for shutdown; the pattern differs because storage existed before the
graceful-shutdown refactor. Fair game for a future pass.

---

## 6. Security boundaries

### Authentication

- **Local password** — bcrypt cost 12 (`auth.BcryptCost`). Optionally
  overridden by `ADMIN_PASSWORD` env (plain-compared in constant time).
  `ValidatePassword` always runs a bcrypt compare to avoid leaking
  which mode is active via response time.
- **OIDC** — state + nonce + exp (10 min), verified via `oidc.Verifier`.
  Auto-provisioning rate-limited to 20 accounts/hour/issuer.
- **API keys** — optional, validated via `AdminAuth.SetAPIKeyValidator`.
- **Share password** — bcrypt, optional, rate-limited per share + IP.

### Sessions

Server-side map, persisted to `sessions.json` with 0600 perms, 24h TTL,
auto-extended on activity. Cookies: `HttpOnly`, `Secure` (under HTTPS or
`X-Forwarded-Proto: https`), `SameSite=Strict`. The strict SameSite
makes token-based CSRF unnecessary.

### Authorisation

Three roles (`admin`/`user`/`viewer`) with per-role middleware:

```
aa.RequireAdmin()             → admin only
aa.RequireCanCreateShares()   → admin + user
aa.Middleware                 → any authenticated role
(no middleware)               → public
```

Ownership is enforced in-handler: non-admins only see their own shares
and receive-links.

### Secrets at rest

- Admin password: bcrypt hash in `admin_config.json` (0600)
- User passwords: bcrypt hash in `users` table
- Share passwords: bcrypt hash in `shares.password_hash`
- OIDC client secret: `oidc_config.client_secret` (plaintext — marked
  for future envelope encryption)
- Webhook HMAC secret: `webhook_config.secret` (plaintext — deferred,
  same reason)

### Attack-surface reduction (v2.2)

- Pure-Go binary, no CGO, runs in `scratch` container
- Non-root `uid=10001` in all provided images
- Multipart form budget: 8 MB per request
- Folder-ZIP uncompressed byte budget: configurable via `MAX_FOLDER_ZIP_GB`
- Opt-in SSRF guard on user-supplied webhook URLs (`STRICT_WEBHOOK_URLS`)
- ZIP-slip defence-in-depth on folder ZIPs

---

## 7. Deployment topologies

### 7.1 Single-box homelab (default)

```
    internet / LAN client
           │
           ▼
    ┌──────────────┐
    │ reverse prox │ (optional; nginx/Traefik/Caddy)
    └──────┬───────┘
           │
           ▼
    ┌──────────────┐
    │ casadrop     │ alpine image, uid=10001
    │   :8080      │
    └──────┬───────┘
           │
           ▼
      data/ volume
       ├─ shares.db
       ├─ uploads/
       └─ thumbnails/
```

### 7.2 Multi-network homelab

The `scripts/entrypoint.sh` auto-detects LAN IP, Tailscale DNS name,
EasyTier IP, and Cloudflare tunnel URL, and exports them as env vars
before running the binary. The `GET /api/network` endpoint then
returns all known ingress URLs so the frontend can render the correct
share link based on the admin-selected "primary network".

### 7.3 Scratch deployment (experimental)

`Dockerfile.scratch` produces a ~3 MB image with no shell, no package
manager, no auto-detection. Operators must pass ingress URLs as env
vars manually. Use this when attack-surface reduction matters more
than plug-and-play detection (e.g. Kubernetes clusters with a
reverse proxy that handles ingress).

### 7.4 Reverse proxy contract

CasaDrop honours these headers:

| Header | Used for |
|---|---|
| `X-Forwarded-For` | client IP (first hop only, see `utils.GetClientIP`) |
| `X-Real-IP` | client IP fallback |
| `X-Forwarded-Proto` | `Secure` cookie flag |
| `Host` / `r.Host` | post-logout redirect construction |

**`X-Forwarded-Host` is deliberately ignored** after the v2.2 review.
Proxies that rewrite `Host` to the public-facing hostname (nginx,
Traefik, Caddy all do this by default) will Just Work. Proxies that
don't will need their config adjusted.

---

## 8. Testing strategy

| Level | Location | What it covers |
|---|---|---|
| Unit | `internal/storage/storage_test.go` | SQLite backend — CRUD, expiry, migration, counters |
| Unit | `internal/auth/oidc_test.go` | OIDC parseScopes, generateRandomString, config handling |
| Unit | `internal/middleware/*_test.go` | Rate limiter, security headers, max body middleware |
| Unit | `internal/handlers/handlers_test.go` | Individual handler shape (webhook config, receive link CRUD) |
| Integration | `internal/routes/routes_test.go::TestAuthFlow` | Full router: status → login → protected → logout → denied |

The integration test is the **regression gate** for middleware-layer
changes. If it goes red, something in the auth chain, session
management, or router wiring is wrong.

```bash
CGO_ENABLED=0 go test ./...
```

all packages must stay green. CGO is explicitly off to match the
production build.

---

## 9. Extension points

Places new features plug in cleanly:

| I want to... | Where |
|---|---|
| Add an HTTP route | `internal/routes/routes.go` (single source of truth) |
| Add a handler | `internal/handlers/` — function on `*Handler`, register in routes |
| Add a storage method | `storage.StorageBackend` interface + `sqlite.go` impl + `storage.go` wrapper |
| Add a background worker | New package with `Start(...)` + `Stop()`; wire in `cmd/server/main.go` |
| Add a protected endpoint | Route via `aa.RequireAdmin()` / `aa.RequireCanCreateShares()` wrapper |
| Add a webhook event | `internal/webhook/webhook.go` — add `NotifyXxx` method; call from handler |
| Add an OIDC claim | `auth/oidc.go::UserInfo` + `user_service.go::FindOrCreateOIDCUser` |
| Add a new role | `internal/models.Role` + middleware helper in `middleware/auth.go` |
| Add a config value | env var read in `main.go` OR `tunnel_config.json` field via `loadTunnelConfig` |

The important constraint: **never reach into another package's state
directly.** Storage is behind an interface, handlers hold a `*Handler`
struct with explicitly-set dependencies, middleware is composable.
This is what made the v2.2 router extraction possible without a full
rewrite.

---

## 10. Known technical debt

1. **Storage cleanup goroutine** in `sqlite.go` uses a different
   shutdown pattern than the other workers. Should migrate to the
   `Stop()` method style.
2. **`cmd/server/main.go` dependency wiring** is still manual. Could
   use a small DI helper, but the current shape is readable enough
   that introducing a framework would cost more than it saves.
3. **No OIDC integration test.** Would need a mock IdP. Meaningful
   work, deferred.
4. **Frontend is vanilla JS with ad-hoc i18n.** Works fine, but any
   non-trivial UI addition should probably introduce a lightweight
   framework or templating system.
5. **`entrypoint.sh` auto-detection** is the only Bash on the system.
   Porting it into Go would unlock a true `FROM scratch` default and
   remove one more attack-surface category.
6. **Webhook HMAC secret + OIDC client secret** are plaintext at rest.
   Envelope encryption is a threat-model decision.

If you're looking for a clean starter task: **#3 (OIDC integration
test)** is the highest-leverage item for test coverage, and **#5
(porting entrypoint.sh to Go)** is the highest-leverage item for
infrastructure cleanup.
