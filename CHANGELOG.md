# Changelog

All notable changes to CasaDrop will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- **Docker healthcheck reported a healthy container as failed.** The compose
  healthchecks used `wget --no-verbose --tries=1 --spider …`, which failed with
  `Remote file does not exist -- broken link!!!` (exit 8) even though the endpoint
  answered normally to `wget -qO-`. Two causes, both fixed:
  1. The alpine runtime image installs **GNU wget** (`apk add wget`), which
     shadows BusyBox's applet at `/usr/bin/wget`. GNU `--spider` issues a **HEAD**
     request (BusyBox's `--spider` uses GET — hence the confusing "BusyBox"
     symptom). All shipped healthchecks now do a plain `GET`
     (`wget -qO- … >/dev/null 2>&1 || exit 1`) against `/healthz`.
  2. `/healthz`, `/readyz` and `/api/auth/status` were registered with
     `.Methods("GET")` only, and gorilla/mux does not imply `HEAD` from `GET` —
     so a HEAD probe got a **404** from a perfectly healthy server. They now
     match `GET, HEAD`, so HEAD-based probes (`--spider`, some load balancers)
     work too. Regression test `TestHealthProbesAnswerHEAD`.
  - Touched: `Dockerfile`, `docker-compose.yaml`, `docker-compose.zimaos.yaml`,
    `dist/casadrop-install-2.3.0/casadrop-zimaos-app.yaml`, `internal/routes/routes.go`,
    `docs/docker-compose.md`, `docs/HOWTO.md`, `docs/zimaos.md`,
    `docs/docker-compose.{traefik,caddy,authentik}.yml`. The image healthcheck also
    moved from `/api/auth/status` to the purpose-built `/healthz`.
- **`entrypoint.sh` used `grep -P`, which BusyBox grep does not support.** Local-IP
  auto-detection printed a full grep usage dump into the container log on every
  start and always fell through to the `hostname -i` fallback. Replaced with an
  awk field scan; verified against BusyBox.

## [2.4.1] - 2026-07-13 — Security review follow-ups

### Security (multi-agent code + security review)
- **X-Forwarded-For spoofing (CWE-348) defeated rate-limit and login lockout.**
  Behind a trusted proxy, `GetClientIP` returned the *leftmost* XFF entry, which
  a conforming proxy leaves fully client-controlled (it appends the real peer on
  the right). A client could send `X-Forwarded-For: <rotating-fake>, <real>` to
  get a fresh, attacker-chosen key on every request, bypassing per-IP login
  lockout, the share-password limiter, and receive-upload limits. XFF is now read
  right-to-left, skipping trusted-proxy hops, so the real client IP can't be
  spoofed. Regression tests in `utils_test.go`.
- **IDOR: any authenticated user could delete another user's *expired* share or
  receive link by ID.** `DeleteShare`/`DeleteReceiveLink` fell through to an
  unconditional delete when the resource was expired (the ownership check sat
  behind a `Get` that filters expired rows). They now return 404 and leave
  reaping to the background cleaner. Regression test
  `TestDeleteExpiredShareRequiresOwnership`.
- **Extension-blocklist bypass via a trailing space or dot.** `"evil.exe "` and
  `"evil.exe."` slipped past the `.exe` block (and Windows strips trailing dots,
  so the latter is really executable). The name is now trimmed before the
  extension is extracted, in both `TunnelConfig` implementations. Regression
  tests in `tunnel_test.go` and `extension_allow_test.go`.
- **Data races on the admin config and setup token.** `config` and `setupToken`
  were read/written without consistent locking (setup wizard racing with the
  per-request auth checks, and with TOTP enable/disable). Both are now guarded by
  a dedicated `configMu`, and the setup token is validated-and-consumed in one
  atomic critical section so two concurrent setup POSTs can't both pass. Verified
  clean under `go test -race`.
- **Attribute-injection hardening (defense-in-depth).** The frontend `escapeHtml`
  helper did not escape quotes, so a value placed into an HTML attribute (e.g. an
  uploaded filename in the admin's shares list `title=`) could break out. It now
  escapes quotes too; the strict CSP already prevented script execution, but this
  closes the markup-injection. Verified in a real browser: a `x" onmouseover=…`
  filename renders as an inert, fully-escaped `title` (no injected handler, canary
  never fires).

### Fixed
- **Copy-mode "share from host" didn't attribute the copied bytes to the owner,**
  so `GetUserUsage` never counted them and the storage quota could be exceeded by
  repeated copies. The share now records `UserID`/`UserEmail` like every other
  creation path (admin-only route, so low impact).
- **Orphaned files on expiry cleanup.** `cleanupExpiredShares` /
  `cleanupExpiredReceiveLinks` deleted DB rows with a re-evaluated `now()`
  predicate, so a resource crossing the expiry boundary mid-cleanup was removed
  from the DB while its file was never deleted. They now delete exactly the IDs
  whose files were removed.
- **Non-atomic received-file accounting.** `SaveReceivedFile` inserted the file
  row and bumped `receive_links.total_size` (which feeds quota accounting) in two
  separate statements; they're now one transaction, matching `DeleteReceivedFile`.
- **Create-user UI required a password,** contradicting the backend which allows
  passwordless (SSO-only) accounts. The field is now optional with an 8-char
  minimum only when set, plus a hint. Verified in a real browser (DE locale).
- **Silently swallowed errors** in the receive-link handlers: a DB error while
  counting received files rendered as `files_count: 0` instead of surfacing. The
  detail endpoint now returns 500 and both log the error.

### Known limitation
- Per-user quota enforcement is best-effort: two concurrent uploads by the same
  owner can each pass the check before either writes, slightly overshooting the
  limit. A fully atomic reservation needs a transactional counter across the
  storage interface and is deferred.

## [2.4.0] - 2026-07-13 — Storage Quotas, Malware Scanning & Abuse Protection

### Added
- **Per-user storage quota** (`users.quota_bytes`, 0 = unlimited), set inline by
  admins in User Management. Enforced with HTTP 413 on single/multi/chunked
  uploads and copy-mode host shares. Receive-link uploads count against the
  **link owner's** quota, so anonymous uploads can't bypass a user's limit.
  Symlink shares and in-place folder shares are excluded — they reference host
  files and consume no managed storage. Exposed via `/api/me` and `/api/users`.
- **ClamAV malware scanning** for the public receive-upload path
  (`CLAMAV_ADDR`, opt-in). Dependency-free clamd `INSTREAM` client
  (`internal/scan`). **Fail-closed:** an infected file is rejected (422) and any
  scanner error — including clamd being unreachable — rejects the upload (503)
  rather than waving it through.
- **Proof-of-work throttle** on public receive uploads (`RECEIVE_POW_BITS`,
  opt-in): stateless HMAC-signed, single-use challenge solved in-browser via
  WebCrypto (`internal/pow`). Plus an always-on per-IP rate limit
  (`RECEIVE_RATE_PER_HOUR`, default 30).
- **Unlimited expiry for shares** — a share can now be set to never expire, both
  at creation and when editing an existing one.
- **`docs/HOWTO.md`** — complete community guide: install (Docker/Compose/source),
  the first-run setup-token flow, sharing, receive links, users/roles/quotas,
  public access incl. the `TRUSTED_PROXY` fail-closed trap, an API cookbook of
  verified `curl` examples, a hardening checklist, backups and troubleshooting.

### Changed
- **Multi-arch images: `linux/amd64` + `linux/arm64`.** The Go build stage now
  runs natively on the build host and cross-compiles to `$TARGETARCH` (the binary
  is CGO-free), so Raspberry Pi 4+ and ARM NAS boxes can finally `docker pull`
  instead of building from source. Applies to `Dockerfile` and
  `Dockerfile.scratch`.
- Documentation corrected against the code: the Prometheus endpoint is
  `/api/metrics` and is **admin-only** (it was documented as a public `/metrics`);
  the database file is `shares.db` (not `casadrop.db`); `PANGOLIN_URL` and
  `ZEROTIER_IP` are not read by the application (Pangolin needs no variable —
  links follow `X-Forwarded-Host`; EasyTier uses `EASYTIER_IP`); the `SMTP_*`
  environment variables never existed — SMTP is configured in the admin UI.

### Fixed
- **SSO login silently wiped a user's storage quota.** `GetUserByEmail` and
  `GetUserByOIDC` did not `SELECT quota_bytes`, so they returned `QuotaBytes: 0`.
  The OIDC login path loads the user through exactly those queries and hands the
  same struct to `UpdateUser` (to refresh `last_login_at`), which persists every
  column — so each single sign-on wrote the quota back as `0`, i.e. *unlimited*,
  with no error and no log line. Admin-set quotas therefore survived only until
  the user's next SSO login. Both lookups now select the column. Regression test
  `TestLookupsPreserveQuota` (fails against the old code).
- **Only one local user account could ever be created.** The `users` table
  carries `UNIQUE(oidc_subject, oidc_issuer)`, and local (non-OIDC) accounts
  were inserted with `''` in both columns. SQLite treats two NULLs as distinct
  but two empty strings as equal, so the *second* local account always collided
  and the API answered a bare `500 Failed to create user` — with nothing in the
  log. Local accounts now store `NULL` (all read paths already `COALESCE` back
  to `""`), existing `''` rows are normalized on startup, and the handler logs
  the underlying error instead of swallowing it. Affects every release that
  shipped per-user local auth (≤ 2.3.0). Regression tests
  `TestCreateMultipleLocalUsers` and `TestOIDCIdentityStillUnique` (the latter
  proves the real OIDC uniqueness constraint still bites).

### Security (multi-agent code review — v2.4 follow-up)
- **IDOR fix — ownerless receive links are now admin-only.** `GetReceiveLink`,
  `GetReceivedFiles`, and `DownloadReceivedFile` guarded with
  `link.UserID != "" && link.UserID != user.ID`, so an ownerless link (created
  under the shared-admin login, `UserID==""`) skipped the check — any
  authenticated Viewer/User could read/download the admin's received files by
  ID. Now matches the share path (`link.UserID != user.ID`). Regression test
  `TestOwnerlessReceiveLinkIsAdminOnly`.
- **Chunked-upload disk-exhaustion / size+quota bypass.** `UploadChunk` enforced
  only a 10 MB per-chunk cap; a client could declare `TotalSize:1` and stream
  10 MB per index to disk until `FinalizeChunkUpload` finally rechecked. Added a
  running cumulative cap against the declared `TotalSize` (413 on overflow).
  Regression test `TestUploadChunkEnforcesCumulativeSize`.
- **Multi-file upload quota now fails closed.** A `GetUser`/`GetUserUsage`
  storage error left `quotaCap=0` (unlimited), letting a batch bypass a nearly
  full quota; it now returns 500 on error, matching the single-file path.
- **Folder-ZIP symlink leak.** `DownloadFolderZip` followed symlinks
  (`copyFileToZip` → `os.Open`), so a `link -> /etc/passwd` inside a shared
  folder leaked the target's content into the public ZIP, escaping the share
  root + `SHARE_ALLOWED_PATHS` and defeating the `MAX_FOLDER_ZIP_GB` budget.
  Symlinks are now skipped (consistent with the single-file path).
- **Email header injection.** The email `Subject` (derived from
  attacker-controlled transfer title/sender) was written verbatim; CR/LF are now
  stripped from header values (`stripHeaderValue`). Test `TestStripHeaderValue`.
- **Share/QR URL host is now fail-closed.** `GetBaseURL` honored
  `X-Forwarded-Host`/`-Proto` unconditionally; a direct client could spoof the
  host baked into generated share links and poison the cacheable public
  `/qr/{id}`. Forwarded headers are now trusted only from a `TRUSTED_PROXY` peer
  (matching `GetClientIP`/`IsRequestSecure`). Test adds a fail-closed case.
- **Hardening:** `DownloadFolderFile` now sends `X-Content-Type-Options:
  nosniff`; dropped the unused `api.qrserver.com` origin from the `img-src` CSP
  (QR is served locally); `X-XSS-Protection` set to `0` (modern guidance).
- **Dependency CVEs cleared.** Go toolchain 1.25.10 → 1.25.11 (net/textproto
  GO-2026-5039, crypto/x509 GO-2026-5037) and `golang.org/x/image` 0.14.0 →
  0.43.0 (webp-decode panic GO-2026-5061, reachable via the thumbnail path).
  `govulncheck ./...` now reports **0** affected vulnerabilities.

### Security (pre-public-release hardening review)
- **Setup-wizard takeover guard.** When no `ADMIN_PASSWORD` is set, the
  unauthenticated `/setup` wizard now requires a one-time **setup token** that is
  printed only to the server logs (`docker logs casadrop`). This closes the race
  where an internet-exposed, not-yet-configured instance (a tunnel can publish
  the URL before setup finishes) is claimed by whoever reaches `/setup` first.
  Setting `ADMIN_PASSWORD` skips the wizard entirely.
- **IDOR fix — ownerless items are now admin-only.** Shares/receive links with
  an empty `UserID` (created under the shared-admin login or via receive-link
  auto-share) were accessible to *every* authenticated user. The ownership guard
  changed from "deny only if owned by someone else" to "deny unless owned by me
  (or admin)", and auto-shared files now inherit the parent link's owner.
- **OIDC `disable_local_auth` now takes effect at runtime.** It was only honored
  when OIDC was enabled at startup; enabling OIDC via the admin API left the
  password login path open until restart. `AdminAuth` now consults the live
  provider as the single source of truth.
- **OIDC auto-provision refuses `OIDC_DEFAULT_ROLE=admin`** (would promote every
  IdP account to admin); falls back to `user` with a loud warning. Promote
  admins explicitly.
- **Receive-link password brute-force protection** — `POST /r/{id}/upload` now
  goes through the same per-(link,IP) rate limiter as share-download passwords.
- **Admin-lockout DoS fixed** — a locked IP no longer hard-blocks *before*
  credential verification, so an attacker can't lock the admin out of a shared
  NAT/egress IP. Correct credentials always pass; wrong ones are counted and
  throttled (escalating delay).
- **Filesystem browser is admin-only** — `/api/browse`, `/api/share-from-path`,
  `/api/share-folder` moved from "can create shares" (User+Admin) to Admin-only,
  so a non-admin can't exfiltrate files (e.g. `~/.ssh`) via `SHARE_ALLOWED_PATHS`.
- **Stored-XSS / same-origin phishing hardening** — `/stream/{id}` now serves
  only a strict media allow-list (video/audio/image-non-SVG/pdf) inline; HTML,
  SVG, XML, text and unknown types are forced to `attachment` +
  `application/octet-stream` + `nosniff`.
- **`WEBHOOK_STRICT_SSRF` now defaults to `true`** (fail-closed) — strict SSRF +
  DNS-rebinding protection for outbound webhooks is on unless explicitly disabled
  with `WEBHOOK_STRICT_SSRF=false` for LAN receivers.
- **Defense-in-depth CSRF** — mutating `/api/*` requests are rejected when a
  browser reports `Sec-Fetch-Site: cross-site` and authenticates by cookie
  (complements the SameSite=Strict session cookie).
- **Session tokens hashed at rest** — `sessions.json` and the in-memory session
  map now store only the SHA-256 of the bearer token (the raw token lives only in
  the client cookie), so a data-dir read / backup leak can't yield live
  session-hijack tokens. Existing sessions are invalidated on upgrade (re-login).
- **OIDC PKCE + state→browser binding** — the authorization-code flow now uses
  PKCE (S256), and the OAuth `state` is bound to the initiating browser via a
  short-lived HttpOnly cookie verified on callback (defeats authorization-code
  injection and login-CSRF / forced-login).
- **Hardening fixes**: share-expiry integer-overflow clamp on the main upload
  path; `Content-Disposition` filenames strip CR/LF (header-injection) across all
  download paths; `AuthStatusHandler` honors the absolute session-lifetime cap;
  thumbnail generation rejects decompression-bomb dimensions (>50 MP) before full
  decode; `Search` escapes `LIKE` wildcards; corrected the misleading
  `TRUSTED_PROXY` doc comment (the code is fail-closed).
- **Container/infra hardening**: `cap_drop: ALL` + minimal `cap_add` on the app
  service (both compose files); `no-new-privileges` + `cap_drop: ALL` on the
  tunnel service; dropped `SYS_MODULE` from the tailscale service; `Dockerfile.tunnel`
  pins/parameterizes the cloudflared version, builds multi-arch (`TARGETARCH`),
  fails on download error, and bumps to `alpine:3.21`; removed the dead/broken
  `Dockerfile.tailscale`; bounded the tunnel-wrapper log; `build-module.sh` now
  builds pure-Go (`CGO_ENABLED=0`); `start-with-tailscale.sh` chowns to uid 10001.

### Added
- **Per-user storage quota** — admins can cap managed storage per user
  (`users.quota_bytes`, 0 = unlimited), set inline in User Management (GB
  granularity). Usage = Σ uploaded/copied share `file_size` **plus** Σ
  receive-link `total_size`; symlink and in-place folder shares are excluded
  (they reference host files without copying). Enforced with HTTP 413 on
  `UploadFile`, `UploadMultipleFiles`, chunk `Init`+`Finalize` (re-checked with
  the actual assembled size), and copy-mode `ShareFromPath`. **Receive uploads
  count against the LINK OWNER's quota**, so anonymous uploads can't bypass a
  user's limit. `/api/me` and `/api/users` expose `quotaBytes` + `usageBytes`.
- **Optional ClamAV malware scanning of receive-link uploads** (`CLAMAV_ADDR`) —
  the only path where anonymous strangers upload. Dependency-free clamd
  `INSTREAM` client (`internal/scan`); `CLAMAV_TIMEOUT` (default 30s) bounds
  dial+scan. **Unset = disabled** (opt-in). When set, uploads are **fail-closed**:
  an infected file is rejected (422) and any scanner error — including clamd
  unreachable — rejects the upload (503) rather than waving it through.
- **Optional proof-of-work throttle on public receive uploads** (`RECEIVE_POW_BITS`)
  — anti-abuse for anonymous strangers. Value = required leading zero bits of
  `SHA-256(challenge+"."+solution)`, solved in-browser via WebCrypto
  (dependency-free, `internal/pow`, stateless HMAC-signed single-use challenge
  served at `GET /r/{id}/challenge`). **Unset/0 = disabled** (opt-in); needs a
  secure context (HTTPS/localhost) client-side. Plus an always-on per-IP baseline
  limit `RECEIVE_RATE_PER_HOUR` (default 30) on `POST /r/{id}/upload`.
- **Tailscale Taildrop** — send an existing share's file straight to one of your
  own tailnet devices ("send to my device"). Admin-only action in the shares
  list; the target must match a live `tailscale status` peer and the file is
  always resolved from a managed share (no arbitrary path), `tailscale file cp`
  runs without a shell. Endpoints `GET /api/taildrop/status`,
  `POST /api/taildrop/send`. Degrades gracefully (button hidden) when Tailscale
  is unavailable.
- **Custom fixed-domain field** in network settings — the "Custom Domain / URL"
  card is a clearly-labeled free-text input (example placeholder + hint) for a
  user's own fixed domain (reverse proxy, named Cloudflare tunnel, WireGuard);
  selectable as the primary network like any other.
- Restored the Cloudflare **Quick Tunnel** wrapper (`scripts/tunnel-wrapper.sh`)
  so the optional `tunnel` compose profile builds again and the free, no-account
  `*.trycloudflare.com` URL is auto-detected.
- OIDC/OAuth2 authentication support (Authentik, Keycloak, etc.)
- Environment variable protection for OIDC configuration
- CI/CD pipeline with GitHub Actions
- Multi-architecture Docker builds (amd64, arm64)
- Dependabot for dependency updates

### Security (pre-release hardening pass)
- **Dependency + toolchain bump to clear all known CVEs**: `go-jose/v3`
  v3.0.1 → v3.0.5 (GO-2025-3485 DoS in OIDC token parsing) and Go toolchain
  pinned to 1.25.10 (clears the stdlib advisories incl. `crypto/x509`
  GO-2025-4007). `govulncheck ./...` now reports 0 affected vulnerabilities.
- **Receive-link webhooks honor `WEBHOOK_STRICT_SSRF`**: `sendReceiveWebhook`
  previously only refused redirects; it now uses the same DNS-pinning transport
  as the global webhook service, closing a DNS-rebinding SSRF on per-link
  webhooks. Shared `utils.StrictSSRFTransport`.
- **TOTP anti-replay**: a 2FA code is now single-use within its acceptance
  window (last-consumed step counter persisted); the enrollment code is consumed
  on enable so it can't be replayed to log in.
- **`Secure` cookie flag no longer trusts a spoofable `X-Forwarded-Proto`** —
  the header is honored only from trusted proxies (or when `TRUSTED_PROXY` is
  unset, the always-proxied default), via `utils.IsRequestSecure`.
- **`OIDC_DISABLE_LOCAL_AUTH` is enforced on the endpoint**, not just hidden in
  the UI: both the form and JSON login reject the password path when disabled.
- **Path-handling consistency**: `ShareFromPath`/`ShareFolder` now operate on the
  symlink-resolved path for stat/open/walk (closes a validate-vs-use TOCTOU).
- **OIDC stored email is updated only when `email_verified`** on the
  subject fast-path (prevents an unverified email from overwriting the key used
  by local login / email-linking).
- **Bounded `tailscale status`/`funnel` execs** so a wedged `tailscaled` can't
  hold the config lock indefinitely.
- **Forwarded-header trust is now fail-closed** (behavior change): `X-Forwarded-For`
  /`-Real-IP`/`-Proto` are honored **only** from peers in `TRUSTED_PROXY`. When
  `TRUSTED_PROXY` is unset, forwarded headers are ignored and the socket peer is
  used, so a directly reachable client can't spoof its IP to evade rate-limit /
  lockout / the share-password limiter. **Action:** set `TRUSTED_PROXY` to your
  reverse proxy's IP/CIDR so the real client IP is recovered.

## [2.3.0] - 2026-06-01 — Security Review II + Per-User Local Auth

### Added
- **Optional TOTP two-factor authentication (2FA)** for the local admin login —
  authenticator-app second factor, enrolled via Settings (QR + manual secret),
  dependency-free RFC 6238 implementation (`internal/totp`). Endpoints under
  `/api/admin/2fa*` (admin only).
- **Per-user local authentication** — the login form accepts an optional email,
  so users in the `users` table sign in with email + password and their own
  role (Admin/User/Viewer). A blank email keeps the single-admin-password path
  for backward compatibility.
- **Health probes** — public `GET /healthz` (liveness) and `GET /readyz`
  (readiness, pings the storage backend).
- **`TRUSTED_PROXY`** env — comma-separated CIDRs/IPs; `X-Forwarded-For` is only
  honored from these peers (anti-spoofing for rate-limit/lockout). Unset =
  forwarded headers trusted (always-behind-proxy default).
- **`WEBHOOK_STRICT_SSRF`** env — when `true`, webhook delivery resolves the
  target host and refuses to dial private/loopback IPs, pinning the validated IP
  to defeat DNS rebinding.
- Regression tests for OIDC `email_verified` linking, per-user login, and
  request-host link derivation.

### Changed
- **Go module renamed** `zima-share` → `casadrop` (import paths only).
- **Share/receive/QR links follow the access path** — when reached via a public
  or tunnel host (Pangolin, Tailscale, custom domain via `X-Forwarded-Host`),
  links use that host; local/LAN access falls back to the configured primary
  network. (Emails always use the configured URL, never the request host.)
- **Strict CSP** — `script-src 'self'` (removed `'unsafe-inline'`); all inline
  handlers/scripts moved to external `/static/js/*`. Added **HSTS** over HTTPS.
- **Session absolute lifetime** (7 days) on top of the rolling idle timeout.
- Timestamp serialization unified across all tables.

### Fixed
- **OIDC account-takeover**: email-based linking/provisioning now requires
  `email_verified` from the IdP (fails closed otherwise).
- **Webhook SSRF**: redirects are refused (config webhooks *and* receive-link
  webhooks); env `WEBHOOK_URL` is validated.
- API-key role validated against known roles; `crypto/rand` errors checked.
- Email-notification HTML escaped; folder MIME read from the validated path;
  receive-upload rolls back on save failure; SQLite cleanup goroutine stops on
  `Close()`; expiry-hour and folder-ZIP byte-budget overflows clamped; Tailscale
  auth-key masking guarded against short keys.

## [2.2.0] - 2026-04-13 — Security + Infrastructure Review

Three-pass code review by zuse (PAI). No behavioural regressions; all
existing tests pass, one new integration test added. Full session notes
in `SESSION_NOTES.md`.

### Added
- **`internal/routes`** package — router setup extracted from
  `cmd/server/main.go`, testable via `routes.New(Deps{...})`.
- **`internal/routes/routes_test.go`** — end-to-end integration test
  walking the auth flow (`/api/auth/status` → login → protected → logout)
  with a real `httptest.Server`.
- **`Dockerfile.scratch`** — experimental minimal scratch-based image
  (no shell, no auto-detection) following "The Anatomy of a 2.5 MB
  Container".
- **`ARCHITECTURE.md`** — top-level architecture reference.
- **`SESSION_NOTES.md`** — review snapshot so work can resume without
  re-reading the chat log.
- **`utils.ValidateExternalWebhookURL`** — opt-in SSRF guard that
  rejects webhook URLs whose host is a literal loopback/private/link-local
  IP.
- **`STRICT_WEBHOOK_URLS`** env flag — when `true`, receive-link and
  webhook-config endpoints use the strict SSRF validator.
- **`MAX_FOLDER_ZIP_GB`** env var — caps the uncompressed byte budget
  for streamed folder-ZIP downloads (default 10 GB).
- **`auth.BcryptCost`** constant — cost 12 (up from `bcrypt.DefaultCost`
  = 10).
- **Per-issuer OIDC auto-provisioning rate limit** — 20 accounts/hour
  per issuer, so a compromised IdP cannot flood the user table.
- **Graceful-shutdown `Stop()` methods** on every long-lived background
  worker: `RateLimiter`, `AdminAuth`, OIDC `Provider`, `EmailHandler`,
  and the package-level chunk cleanup worker.

### Changed
- **SQLite driver** migrated from `github.com/mattn/go-sqlite3` (CGO)
  to `modernc.org/sqlite` (pure Go). Enables `CGO_ENABLED=0` static
  builds and `FROM scratch` container images.
- **Go toolchain** bumped to **1.25** (required by modernc.org/sqlite).
- **`Dockerfile`** now builds with
  `CGO_ENABLED=0 -ldflags="-w -s" -trimpath`, runs as non-root
  `uid=10001`, and no longer installs `gcc`/`musl-dev` in the builder
  stage. Final binary is fully static.
- **`cmd/server/main.go`** shrunk from 267 to ~145 lines; all route
  wiring lives in `internal/routes`.
- **`middleware.ValidatePassword`** now always performs a bcrypt compare
  (against a dummy hash when no credential is configured) to close a
  timing side-channel that distinguished auth modes.
- **Multipart in-memory budget** reduced from 32 MB to 8 MB on upload,
  multi-upload, and receive endpoints. Stops a DoS vector where many
  tiny form fields allocate RAM before `MaxBytesReader` can enforce.
- **OIDC logout** no longer trusts `X-Forwarded-Host` when building the
  post-logout redirect URI (open-redirect via IdP bounce).
- **Logout cookies** now carry `Secure` + `SameSite=Strict`, matching
  the flags set at login time.
- **Folder-ZIP entries** are written with forward slashes
  (`filepath.ToSlash`) and refuse relative paths containing `..`
  (defence-in-depth against zip-slip on the consumer side).

### Fixed
- **CORS origin bypass** in `download.go::StreamFile` — `strings.Contains`
  against the request host would match `evil.com/legit-host`. Now uses
  exact URL host comparison.
- **OIDC state/nonce entropy loss** — `generateRandomString` truncated
  the base64 output to `length` characters, losing ~25% of the entropy.
  Returns the full `RawURLEncoding` now.
- **Receive upload rollback** — when the atomic upload-limit check
  returned `!allowed`, the just-saved file, DB record, and any
  auto-created share were leaked. Full cleanup added.
- **PRAGMA table name whitelist** in `migrate_users.go`. Callers all
  pass literals so this was not exploitable, but the concat pattern is
  gone.
- **Chunk upload finalize race** — `len(upload.ChunksReceived)` is now
  read under the lock that guards the map, not after unlocking.

### Security

See `SECURITY.md` for the full classified list of findings from this
review and their resolution status.

## [2.0.0] - 2024-12-18

### Added
- **SQLite database** for metadata storage (replacing JSON files)
- **Prometheus metrics** endpoint (`/metrics`)
- **Thumbnail generation** for images and videos
- **Folder sharing** with ZIP download support
- **Receive links** (reverse shares) for accepting uploads
- **File browser** for sharing existing server files
- **Webhook notifications** for share events
- **Multi-network support** (Cloudflare, Tailscale, Pangolin, ZeroTier)
- **Configurable max file size** via admin settings
- **Bulk delete** for multiple shares
- **Persistent sessions** surviving container restarts

### Changed
- Rebranded from Zima-Share to **CasaDrop**
- Cookie name changed to `casadrop_session`
- Improved security headers and CSRF protection
- Enhanced rate limiting for downloads

### Security
- Added bcrypt password hashing
- Implemented account lockout after failed attempts
- Added CSRF token validation
- Security audit logging

## [1.0.0] - 2024-11-01

### Added
- Initial release
- File upload with drag & drop
- Password protection for shares
- Expiration dates (1h to 30 days)
- Download limits
- QR code generation
- Dark/Light theme
- i18n support (EN/DE)
- Cloudflare Tunnel integration
- ZimaOS/CasaOS support

[Unreleased]: https://github.com/chicohaager/casadrop/compare/v2.0.0...HEAD
[2.0.0]: https://github.com/chicohaager/casadrop/compare/v1.0.0...v2.0.0
[1.0.0]: https://github.com/chicohaager/casadrop/releases/tag/v1.0.0
