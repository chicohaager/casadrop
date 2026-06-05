# Security Review - CasaDrop v2.0

## Executive Summary

CasaDrop (formerly Zima-Share) is a self-hosted file sharing application for homelabs. This security review covers the authentication system, data handling, network security, and potential attack vectors.

**Overall Assessment**: The application implements strong security controls for a self-hosted file sharing solution. Key strengths include SQLite database with WAL mode, bcrypt password hashing, session management, rate limiting, comprehensive security headers, and secure receive link handling. Areas for improvement are noted below.

---

## Authentication & Session Management

### Admin Authentication

| Feature | Implementation | Status |
|---------|---------------|--------|
| Password Storage | bcrypt with default cost (10 rounds) | ✅ Good |
| Session Tokens | 32 bytes, crypto/rand, base64 encoded | ✅ Good |
| Session Persistence | JSON file with 0600 permissions | ✅ Acceptable |
| Session TTL | 24 hours, auto-extended on activity | ✅ Good |
| Brute Force Protection | 500ms delay on failed attempts | ✅ Good |
| Rate Limiting | 5 attempts/minute per IP | ✅ Good |

**Strengths:**
- Constant-time comparison for environment password (`crypto/subtle`)
- Sessions survive server restarts (persistent storage)
- Rate limiting per IP address
- Session auto-extension prevents unnecessary re-authentication

**Potential Improvements:**
- Consider adding account lockout after N failed attempts
- Add CSRF tokens for form submissions
- Implement session binding to IP/User-Agent (with opt-out for mobile users)
- Add audit logging for authentication events

### Session Cookie Security

```go
http.Cookie{
    Name:     "zima_session",
    HttpOnly: true,                    // ✅ Prevents XSS access
    Secure:   (TLS || X-Forwarded-Proto), // ✅ HTTPS only when applicable
    SameSite: http.SameSiteStrictMode, // ✅ CSRF protection
    MaxAge:   86400,                   // 24 hours
}
```

**Status**: ✅ Well configured

### Share Password Protection

- Per-share optional password
- bcrypt hashed (same as admin password)
- No rate limiting on share password attempts

**Recommendation**: Add rate limiting to share password verification to prevent brute force attacks on password-protected shares.

---

## Security Headers

Implemented via `middleware/security.go`:

```
Content-Security-Policy: default-src 'self'; script-src 'self';
                         style-src 'self'; img-src 'self' data: https://api.qrserver.com;
                         font-src 'self'; connect-src 'self'; frame-ancestors 'none';
                         base-uri 'self'; form-action 'self'
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: accelerometer=(), camera=(), geolocation=(), gyroscope=(),
                    magnetometer=(), microphone=(), payment=(), usb=()
```

| Header | Status | Notes |
|--------|--------|-------|
| CSP | ✅ Strict | No `unsafe-inline` - all scripts and styles in external files |
| X-Content-Type-Options | ✅ Good | Prevents MIME sniffing |
| X-Frame-Options | ✅ Good | Prevents clickjacking |
| X-XSS-Protection | ✅ Good | Legacy browser protection |
| Referrer-Policy | ✅ Good | Limits referrer leakage |
| Permissions-Policy | ✅ Good | Restricts browser features |

**Status**: ✅ Implemented - all inline scripts and styles moved to external files.

---

## Input Validation & Sanitization

### File Upload

| Check | Implementation | Status |
|-------|---------------|--------|
| File Extension Blocking | Configurable blocklist (exe, bat, etc.) | ✅ Good |
| File Extension Whitelist | Optional allowlist | ✅ Good |
| File Size Limit | Configurable 1-100 GB | ✅ Good |
| Filename Sanitization | UUID-based storage names | ✅ Good |

**File Storage Pattern:**
- Original filename stored in metadata
- File stored with UUID name: `uploads/<uuid>/<original_filename>`
- Symlinks used for server-path shares (validated against allowlist)

**Potential Risk**: Symlink shares (`share-from-path`) depend on `SHARE_ALLOWED_PATHS` configuration. Misconfiguration could expose sensitive files.

### URL Validation

```go
func validateURL(urlStr string, requireHTTPS bool) error {
    // Validates scheme (http/https) and host presence
}
```

- Tailscale, Cloudflare, Pangolin URLs require HTTPS
- Custom URLs allow HTTP for local reverse proxies
- Empty URLs are valid (optional fields)

**Status**: ✅ Acceptable

### Path Traversal Protection

- File browser restricted to `SHARE_ALLOWED_PATHS`
- Path normalization with `filepath.Clean()`
- Checks for `..` and absolute paths

**Status**: ✅ Good

---

## Network Security

### Exposed Endpoints (No Auth Required)

| Endpoint | Purpose | Risk Level |
|----------|---------|------------|
| `/login`, `/setup` | Authentication | Low |
| `/s/{id}` | Share page | Low (requires valid share ID) |
| `/d/{id}` | Download | Low (rate limited, optional password) |
| `/stream/{id}` | Media streaming | Low (same as download) |
| `/qr/{id}` | QR code | Low |
| `/thumbnail/{id}` | Image thumbnail | Low (requires valid share ID) |
| `/folder/{id}/contents` | Folder contents | Low (password protected at handler) |
| `/folder/{id}/download` | Folder file download | Low (password protected at handler) |
| `/folder/{id}/zip` | Folder ZIP download | Low (password protected at handler) |
| `/r/{id}` | Receive link page | Low (optional password) |
| `/r/{id}/upload` | Upload to receive link | Low (password/limits enforced) |
| `/api/auth/status` | Auth status check | Low |
| `/metrics` | Prometheus metrics | Low (no sensitive data) |

**Notes:**
- `/api/network` and `/api/tunnel` now require authentication (v1.5+)
- Receive links have configurable upload limits and allowed extensions
- Folder shares support optional password protection

### Rate Limiting

| Endpoint | Limit | Status |
|----------|-------|--------|
| Login | 5/min per IP | ✅ |
| Download | 10/min per IP | ✅ |
| Upload | No specific limit | ⚠️ Consider adding |
| API | No specific limit | ⚠️ Consider adding |

### Tailscale Funnel Integration

When using Tailscale Funnel:
- Traffic is encrypted end-to-end via WireGuard
- Public access without port forwarding
- Authentication handled by Zima-Share, not Tailscale

**Status**: ✅ Good integration

---

## Data Security

### Storage (v2.0 - SQLite)

| Data | Location | Protection |
|------|----------|------------|
| All metadata | `shares.db` | SQLite with WAL mode, file permissions |
| Admin config | `admin_config.json` | 0600, bcrypt hash |
| Sessions | `sessions.json` | 0600 |
| Tunnel config | `tunnel_config.json` | File permissions |
| Uploaded files | `uploads/` | Directory permissions |
| Received files | `uploads/received/{id}/` | Directory permissions |
| Thumbnails | `thumbnails/` | Directory permissions |

**SQLite Security:**
- WAL mode for concurrent reads and atomic writes
- Foreign key constraints enabled
- Parameterized queries prevent SQL injection
- Database file has restricted permissions

### Sensitive Data Handling

- Passwords: bcrypt hashed, never logged
- Session tokens: Masked in API responses (`token[:8]...`)
- Share IDs: UUID v4 (8 chars), not guessable
- Receive Link IDs: UUID v4 (8 chars), not guessable

**Recommendations:**
- Ensure `DATA_DIR` has restrictive permissions (700 or 750)
- Consider encrypting sensitive config files at rest

---

## Container Security

### Dockerfile Analysis

```dockerfile
# Non-root user
USER zima (uid=1000)

# Minimal base image
FROM alpine:3.19

# No unnecessary packages
RUN apk add --no-cache ca-certificates tzdata wget
```

| Feature | Status |
|---------|--------|
| Non-root user | ✅ Good |
| Minimal base image | ✅ Good |
| No shell access needed | ✅ Good |
| Health check | ✅ Good |
| Read-only volumes for host paths | ✅ Good (`:ro` in compose) |

---

## Receive Links Security (v2.0)

### Features
- Password protection (optional, bcrypt hashed)
- Upload limits (max uploads per link)
- File size limits (per link or global)
- Allowed extensions filtering
- Automatic expiration
- Per-link webhook notifications

### Security Measures
| Feature | Implementation | Status |
|---------|---------------|--------|
| Password | bcrypt hashed | ✅ Good |
| Upload limit | Server-side counter | ✅ Good |
| File size | MaxBytesReader | ✅ Good |
| Extensions | Whitelist validation | ✅ Good |
| Expiration | Auto-cleanup | ✅ Good |

### Potential Risks
- **Disk exhaustion**: Many uploads could fill disk
  - Mitigation: Set max_uploads and max_file_size
- **Malicious files**: Users could upload malware
  - Mitigation: Extension whitelist, no execution
- **DoS via uploads**: Repeated large uploads
  - Mitigation: Upload limits per link

---

## Folder Shares Security (v2.0)

### Features
- Browse folder contents via web UI
- Download individual files
- Download entire folder as ZIP
- Optional password protection

### Security Measures
| Feature | Implementation | Status |
|---------|---------------|--------|
| Path traversal | Relative path validation | ✅ Good |
| Access control | Password or public | ✅ Good |
| ZIP creation | On-demand, temp files cleaned | ✅ Good |

### Potential Risks
- **Large ZIP creation**: Could use memory/CPU
  - Mitigation: ZIP created on-demand, not cached
- **Symlink traversal**: Folders could contain symlinks
  - Mitigation: filepath.Walk follows symlinks but within share root

---

## Prometheus Metrics Security (v2.0)

### Exposed Metrics
- `zima_uploads_total` - Upload count by status
- `zima_downloads_total` - Download count
- `zima_active_shares` - Current share count
- `zima_http_requests_total` - HTTP request count
- `zima_http_request_duration_seconds` - Request latency

### Security Considerations
- Endpoint `/metrics` is public (no auth)
- No sensitive data exposed (no IPs, filenames, passwords)
- Consider firewall rules if metrics should be internal only

---

## Potential Attack Vectors

### 1. Brute Force Attacks

**Target**: Admin login, share passwords

**Mitigations in place**:
- Rate limiting (5/min for login)
- Delay on failed attempts (500ms)

**Additional recommendations**:
- Account lockout after 10 failed attempts
- CAPTCHA after 3 failed attempts
- Fail2ban integration for persistent attackers

### 2. Session Hijacking

**Mitigations in place**:
- HttpOnly cookies
- SameSite=Strict
- Secure flag on HTTPS

**Additional recommendations**:
- Bind sessions to IP address (opt-in)
- Session regeneration after privilege changes

### 3. Path Traversal

**Mitigations in place**:
- Allowlist-based path restriction
- Path normalization
- UUID-based file storage

**Status**: ✅ Well protected

### 4. XSS (Cross-Site Scripting)

**Mitigations in place**:
- Strict CSP headers (no `unsafe-inline`)
- Go template auto-escaping
- All scripts and styles in external files

**Status**: ✅ Well protected

### 5. CSRF (Cross-Site Request Forgery)

**Mitigations in place**:
- SameSite=Strict cookies

**Missing**:
- CSRF tokens for forms

**Recommendation**: Implement CSRF tokens for state-changing operations

### 6. Information Disclosure

**Exposed information**:
- Internal IP addresses via `/api/network`
- Tunnel URLs via `/api/tunnel`

**Recommendation**: Require authentication for these endpoints or limit exposed data

---

## Recommendations Summary

### High Priority - ✅ IMPLEMENTED (v1.5)

1. ✅ **CSRF tokens** for form submissions (login, setup)
2. ✅ **Authentication required** for `/api/network` and `/api/tunnel` (GET)
3. ✅ **Rate limiting** for share password attempts (5 attempts/15 min per share per IP)

### Medium Priority - ✅ IMPLEMENTED (v1.5)

4. ✅ **Account lockout** after 10 failed login attempts (15 min lockout)
5. ✅ **Audit logging** for security events (login, logout, lockout, CSRF violations)
6. ✅ **Remove `unsafe-inline`** from CSP - Implemented (inline styles moved to auth.css, inline event handlers converted to JS event listeners)

### Low Priority - Not Yet Implemented

7. Add optional IP binding for sessions
8. Implement session regeneration on privilege changes
9. Consider config file encryption at rest

---

## Compliance Notes

### GDPR Considerations

- No user tracking implemented
- Share metadata includes uploader IP (consider anonymization)
- Session data includes IP and User-Agent (for security)
- Recommend adding data retention policy

### Self-Hosted Security

As a self-hosted application, security depends heavily on:
- Host system security
- Network configuration (firewall, reverse proxy)
- Docker security settings
- Admin password strength

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 2.2 | 2026-04-13 | Three-pass code review — see "Pass 1/2/3 Review" block below |
| 2.0 | 2025-12-18 | SQLite migration, Prometheus metrics, thumbnails, folder shares, receive links |
| 1.7 | 2025-12-18 | Prometheus metrics endpoint |
| 1.6 | 2025-12-18 | SQLite database migration (from JSON) |
| 1.5 | 2025-12-18 | CSRF protection, account lockout, audit logging, share password rate limiting, API endpoint protection |
| 1.4 | 2025-12-17 | Added professional auth system, persistent sessions |
| 1.3 | 2025-12 | Network selection, file extension filtering |

---

## v2.2 — Three-Pass Review (2026-04-13)

Full-codebase security audit in three iterations with parallel specialized
auditors (auth/sessions, storage/SQL, handlers, concurrency). 30+ items
flagged, each verified against the actual source before fixing. Findings
below are the ones that resulted in a code change.

### Fixed — High / Medium severity

| # | Severity | Location | Bug | Fix |
|---|---|---|---|---|
| 1 | HIGH | `handlers/download.go:251` | CORS origin bypass via `strings.Contains(origin, host)`. `evil.com/legit-host` matched `legit-host`. | Exact `url.Parse(origin).Host == r.Host` + `Vary: Origin`. |
| 2 | HIGH | `auth/oidc.go:432` | `generateRandomString` truncated base64 output to `length` characters, losing ~25% of entropy on OIDC state/nonce. | `base64.RawURLEncoding.EncodeToString(b)` (full width). Test updated. |
| 3 | MEDIUM | `handlers/receive.go:452` | On `IncrementReceiveLinkUploads` race-loss the just-saved file, DB record, and any auto-created share were leaked. | Full rollback of file + received-file row + share. |
| 4 | HIGH (footgun) | `storage/migrate_users.go:159` | `PRAGMA table_info(" + table + ")` built with string concat. Callers pass literals so not exploitable. | `allowedMigrationTables` whitelist. |
| 5 | MEDIUM | `auth/auth.go` + `middleware/auth.go` | bcrypt cost was `bcrypt.DefaultCost` (10). | New `auth.BcryptCost = 12`. Middleware uses 12 directly (import-cycle avoidance). |
| 6 | MEDIUM | `handlers/folder.go:524` | ZIP entries written with OS-native separator; no defence-in-depth `..` check. | `filepath.ToSlash` on `header.Name` + reject `relPath` containing `..`. |
| 7 | MEDIUM | `handlers/upload.go:335` | `FinalizeChunkUpload` read `len(upload.ChunksReceived)` after releasing the lock → race with in-flight `UploadChunk`. | Snapshot under the lock. |
| 8 | HIGH (pass 2) | all background workers | No shutdown drain — goroutines leaked on SIGTERM. | `Stop()` methods on `RateLimiter`, `AdminAuth`, OIDC `Provider`, `EmailHandler`, chunk cleanup worker; called from `main.go` after `srv.Shutdown()`. |
| 9 | LOW | `middleware/auth.go:792` | Logout cookie clear missing `Secure` + `SameSite=Strict`. | Flags added to match login-time cookie. |
| 10 | MEDIUM (pass 3) | `handlers/receive.go:325`, `upload.go:94`, `upload.go:495` | `ParseMultipartForm(32<<20)` allowed 32 MB of RAM per request via many small form fields, before `MaxBytesReader` could enforce. | Budget reduced to 8 MB. |
| 11 | MEDIUM | `middleware/auth.go::ValidatePassword` | Env-password path returned before bcrypt, leaking auth-mode via response time. | Always runs a bcrypt compare (dummy hash when no credential configured). |
| 12 | HIGH | `auth/oidc_handlers.go::LogoutHandler` | Post-logout redirect URI built from client-controlled `X-Forwarded-Host` → open redirect via IdP bounce. | Uses `r.Host` only; explicit comment. |
| 13 | MEDIUM | `auth/user_service.go::FindOrCreateOIDCUser` | OIDC auto-provisioning unbounded — compromised IdP could flood the user table. | Per-issuer rate limit (20/hour). |
| 14 | LOW | `handlers/folder.go::DownloadFolderZip` | No byte budget on streamed ZIP → long-running connection hold. | `MAX_FOLDER_ZIP_GB` env (default 10 GB), abort when exceeded. |

### Added

- **`utils.ValidateExternalWebhookURL`** — rejects URLs whose host is a
  literal loopback/private/link-local IP. Wired into receive-link and
  webhook-config via `STRICT_WEBHOOK_URLS=true` env flag (opt-in for
  homelab use cases that legitimately target the same LAN).
- **`internal/routes/routes_test.go::TestAuthFlow`** — end-to-end
  integration test that spins up an `httptest.Server` against the real
  router and walks `/api/auth/status → /login (JSON) → /api/stats →
  /logout → /api/stats`. This is now the regression gate for any change
  to middleware, session management, or route wiring.

### Infrastructure

- **SQLite driver** swapped to `modernc.org/sqlite` (pure Go). No more
  CGO, fully static binary, compatible with `FROM scratch`.
- **Dockerfile** hardened per "The Anatomy of a 2.5 MB Container":
  `CGO_ENABLED=0 -ldflags="-w -s" -trimpath`, non-root
  `uid=10001`, no `gcc`/`musl-dev` in the builder stage. `Dockerfile.scratch`
  is the minimal variant (no shell, no auto-detection) for deployments
  that prioritise attack-surface reduction over ergonomics.
- **Graceful shutdown** — `main.go` now cleanly drains all background
  workers after `http.Server.Shutdown()` returns.

### Verified not a bug (false positives during audit)

- `UPDATE receive_links SET total_size = MAX(0, total_size - ?) WHERE …` —
  SQLite's `max(x,y)` is a valid scalar function when given two arguments.
- Download counter race — `IncrementDownloads` is already atomic via
  `UPDATE … WHERE downloads < max_downloads`.
- CSRF on API endpoints — session cookie uses `SameSite=Strict`, which
  blocks cross-origin form submission at the browser level. Token-based
  CSRF would be belt-and-braces, not a missing primitive.

### Deferred (scope / threat-model)

- **Webhook HMAC secret at-rest encryption** — stored plaintext in SQLite.
  Envelope encryption is a threat-model decision; DB-leak is not in the
  top-threat ranking for a homelab self-host.
- **Strict session IP/UA binding** — would break mobile users and VPN
  rotators. No default-safe fix.
- **`ValidateExternalWebhookURL` DNS resolution** — only blocks literal
  IPs. Hostnames resolving to private ranges still pass. Strict DNS
  pinning would break legitimate LAN webhooks.
- **OIDC integration test** — needs a mock IdP; bigger in scope than
  this pass.

---

*Last reviewed: 2026-04-13 (zuse/PAI). Regular security reviews are recommended as the application evolves.*
