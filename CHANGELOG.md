# Changelog

All notable changes to CasaDrop will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- OIDC/OAuth2 authentication support (Authentik, Keycloak, etc.)
- Environment variable protection for OIDC configuration
- CI/CD pipeline with GitHub Actions
- Multi-architecture Docker builds (amd64, arm64)
- Dependabot for dependency updates

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
