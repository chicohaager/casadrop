# Changelog

All notable changes to CasaDrop will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/user/casadrop/compare/v2.0.0...HEAD
[2.0.0]: https://github.com/user/casadrop/compare/v1.0.0...v2.0.0
[1.0.0]: https://github.com/user/casadrop/releases/tag/v1.0.0
