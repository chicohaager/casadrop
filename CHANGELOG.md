# Changelog

All notable changes to CasaDrop will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased] — review follow-ups + UI feedback from daily use

### Added (CI, from zero)
- **There was no CI of any kind.** `.github/workflows/ci.yml` now builds, vets,
  gofmt-checks and tests on every push and PR — and, more to the point, **builds
  the image, runs it, and waits for Docker's own `HEALTHCHECK` verdict**. Twice
  a bug has shipped that lived in the image rather than in the Go code (most
  recently `wget --spider` sending HEAD at a GET-only route); every unit test
  stayed green because nothing ever executed the `HEALTHCHECK` line. A second
  container runs with `PORT=9000` to guard the fix below. Measured locally
  before committing: the new image reaches `healthy` on both ports, the previous
  one goes **`unhealthy` after 71 s** with `PORT=9000` while serving perfectly.
- **`tests/entrypoint_localip_test.sh`** feeds real `ip -4 route get` output
  through the `LOCAL_IP` parser — the thing that silently broke when BusyBox
  grep turned out to have no `-P`. Writing it turned up a live edge case: an
  interface named `src` made the parser return the literal string `src`, so it
  now requires the captured token to look like an IPv4 address.

### Fixed (media types, measured)
- **`.m4a`/`.m4b` render as audio instead of a black video rectangle.** They
  carry an `ftyp` box, so the sniffer commits to `video/mp4` — a *recognised*
  type, which the refinement deliberately never overrode. These two extensions
  are now listed as ambiguous with `video/mp4`, the same way `.mkv` is with
  `video/webm`. The migration's candidate set was widened to match, so existing
  shares are corrected too.
- **`.weba` (WebM audio) rendered a `<video>` element.** It shares EBML magic
  with `.webm`, so it sniffed as `video/webm`; it now resolves to `audio/webm`.
- **WAV shares advertise `audio/wav`, not `audio/wave`.** Go's sniffer produces
  the latter; measured in Chrome, `canPlayType("audio/wave")` returns the empty
  string — a flat reject — while `audio/wav` returns `maybe`. Emitting the
  sniffed spelling would have made the browser discard the source *without
  fetching it*. Only the `<source type>` attribute is rewritten; the stored type
  is untouched.
- **`mime.TypeByExtension` is gone as a fallback.** It consulted the OS mime
  database, which would happily answer `image/svg+xml` for an extension somebody
  added to the ambiguous list without adding a resolution — turning the "inert
  types only" safety property into a hope. A new test asserts every value in
  both maps is an `audio/*` or `video/*` type and that every listed extension
  has an explicit resolution.
- **"Detection never ran" is no longer stored as `application/octet-stream`.**
  Four writers used that string as a default when the file could not be opened
  or read, which is indistinguishable from a genuine sniff result — so the
  migration later promoted such rows by file name alone, flipping shares that
  nobody had ever looked at from download-only to rendered inline. They now
  store an empty value (excluded from the migration for free, since SQL `IN`
  matches neither `''` nor NULL) and **log** the reason. Responses run the
  stored value through `utils.ServingMimeType`, so an unset type still goes on
  the wire as `application/octet-stream` rather than as an empty Content-Type.
- **`/readyz` now reads.** `storage.Ping` called `db.Ping()`, which the modernc
  driver answers with `select 1` — without looking at application data. It reads
  a real row now, and the 503 path finally **logs** why. Its remaining limit is
  recorded rather than glossed over: a database file clobbered underneath an
  open connection is answered from SQLite's warm page cache and stays green.
- **`/stream/{id}` was the one public share route with no rate limit**, and a
  plain GET there increments the download counter — a short curl loop could
  exhaust a `max_downloads: 1` share before its recipient opened the link. It is
  throttled now, but only for the requests that count: Range requests, which a
  video player issues in bursts while seeking, pass through untouched.
- **The maintainer's personal username** no longer appears in the `author` and
  `developer` fields of the two shipped compose files; they carry the project's
  public handle instead.

### Fixed (review follow-ups)
- **A failed MIME migration is no longer indistinguishable from a clean one.**
  `refineMimeColumn` returned `(0, nil)` for *any* query error, justified in a
  comment by "a table may not exist yet on a fresh database". That cannot happen:
  `initBaseSchema` creates all three tables and is fatal on failure, and it runs
  first. So the catch-all could only ever hide real errors — and hid them
  perfectly, because with `updated == 0` the summary log line is skipped too. The
  error is now returned and logged. Two new tests pin it: one asserts the error
  is reported for a missing table and a missing column, one drives the migration
  through `NewSQLiteStorage` (the path production takes) with a row in each of
  the three tables. Both were mutation-checked — they fail when the fix is
  reverted or the startup call is removed.
- **The "your browser cannot play this file" note no longer appears on
  password-protected media shares that play perfectly.** On a locked share the
  `<source>` elements carry `data-src` and no `src`, and the browser fires
  `error` on such a source on its own (measured: an srcless `<source>` appended
  to an `<audio>` fires once). The note was shown then and never hidden again, so
  it sat above a working player. The handler now stays disarmed until
  `revealContent` has installed the real sources, and clears any note left over.
  Verified in both directions: while locked the event no longer shows the note,
  and after unlocking a genuine media error still does.
- **The container healthcheck follows `$PORT`.** It probed a hard-coded 8080, so
  a container started with `PORT=9000` reported unhealthy forever while serving
  correctly. Compose files that pin `PORT=8080` themselves were left as they are;
  the three example stacks in `docs/` dropped their redundant override entirely,
  since the image's own healthcheck now does the right thing. A compose-level
  `${PORT}` would *not* work here — compose substitutes it from your shell while
  parsing, not from the container's environment at check time.
- **`docs/tailscale.md` described the Tailscale URL precedence backwards.** It
  said the environment variable wins and the settings field lives in the
  database. Both halves are wrong: `handlers.go` takes the stored value first and
  falls back to `TAILSCALE_URL` only when it is empty, and the value is written
  to `data/tunnel_config.json`, not the database.
- **A LAN IP was visible in a shipped screenshot.** `docs/images/tailscale/01-…`
  showed a private-range address in the field and again in the "Detected:" line.
  Every text-based check passed, because the address lived in pixels. Replaced
  with `192.168.x.x`. The other three screenshots were re-checked by eye and are
  properly anonymised.

### Added (UI feedback from daily use)
- **Thumbnails for the pictures you picked, before they are uploaded.** The
  selection list showed a coloured `IMG` badge for every image, which is no help
  at all when you just dropped a folder of forty photos and want to check the
  selection. Each raster image (`png`, `jpeg`, `gif`, `webp`, `bmp`, `avif`) now
  renders its own thumbnail from a local object URL — nothing is uploaded to
  produce it. SVG deliberately keeps the badge: rendering an untrusted SVG in
  the admin page is an XSS surface. An image the browser cannot decode falls
  back to the badge instead of leaving a broken-image icon. Object URLs are
  revoked on every re-render, so picking and clearing a folder repeatedly does
  not pin the files in memory (verified: 2 of 2 previous URLs revoked).

### Fixed (UI feedback from daily use)
- **"Unlimited" no longer shows a made-up expiry when editing a share.** Opening
  a share created with *Never (unlimited)* showed the toggle switched on — and
  right above it a greyed-out "24" in *Expires in (hours)*, which reads like the
  share dies tomorrow. The field is now empty with `unlimited` as its
  placeholder; the 24-hour default only appears once the toggle is switched off,
  when a number is actually needed.
- **Editing a share with an empty expiry no longer reports success without
  changing anything.** With the toggle off and the field cleared, the dialog sent
  a request that omitted the expiry entirely, closed, and toasted "updated" while
  the share kept its old expiry. It now says what is wrong and stays open.
  New translation key `shares.expiryRequired`, present in all 14 languages.
- **Long file names no longer break out of the Edit / E-mail / Taildrop
  dialogs.** The file name was rendered in a bare paragraph, so a name without
  spaces — the kind photo exports produce — ran straight past the right edge and
  gave the dialog a horizontal scrollbar. It now breaks mid-word, is clamped to
  three lines so the buttons stay reachable, and keeps the full name in a
  tooltip. The shares list and the public share page already handled this.

## [2.4.2] - 2026-07-24 — Healthcheck & media MIME fixes

### Fixed
- **Existing shares with a wrong `mime_type` are corrected on startup.** The
  detection fix below only applied to new uploads; rows written by older
  versions kept their wrong type, so an already-shared FLAC still had no player.
  A migration now re-runs the same disambiguation over the stored value — which
  *is* the old sniff result, so no uploaded file has to be re-read. Only
  non-committal values (`application/octet-stream`, `application/ogg`,
  `video/webm`) are candidates; recognised types are never touched, so the
  migration cannot promote anything into an active type. Covers `shares`,
  `received_files` and `folder_contents`, is idempotent, and logs how many rows
  it changed. Verified against a snapshot of a real instance: 3 of 9 shares
  corrected (`.mkv` → `video/x-matroska`, `.flac` → `audio/flac`, `.mp3` →
  `audio/mpeg`), the two genuine `.webm` shares left untouched.
- **Audio files that Go's sniffer doesn't recognise got no player at all.** MIME
  type was derived purely from `http.DetectContentType`, which carries
  signatures for only a handful of media formats. Measured before/after against
  two running builds:

  | upload | was | player | now | player |
  |---|---|---|---|---|
  | `.mp3` without ID3 header | `application/octet-stream` | **none** | `audio/mpeg` | ✅ |
  | `.flac` | `application/octet-stream` | **none** | `audio/flac` | ✅ |
  | `.opus` | `application/ogg` | **none** | `audio/ogg` | ✅ |
  | `.mkv` | `video/webm` | ✅ | `video/x-matroska` | ✅ |

  **Correction (2026-07-26): `.m4a` was listed here as fixed and is not.** The row
  claimed `application/octet-stream` → `audio/mp4`; both halves are wrong. Measured
  against this build: an M4A carries an `ftyp` box, so the sniffer recognises it
  and returns `video/mp4` — for both iTunes-style (`ftypM4A …M4A mp42isom`) and
  ffmpeg-style (`ftypisom…iso2mp41`) headers. `video/mp4` is a committal answer,
  so `RefineMimeType` leaves it alone by design, and `getMediaType` classifies the
  file as **video**: an M4A renders as a black `<video>` rectangle rather than an
  audio player. Correcting it means letting the extension override a *recognised*
  type, which is exactly the promotion this release deliberately forbids, so it
  needs its own change with its own safety argument. Tracked, not fixed here.

  `getMediaType` classifies by MIME prefix, so `application/octet-stream` meant
  `MediaTypeUnknown` and the share page rendered no `<audio>` element — the file
  could only be downloaded.
- **Matroska was labelled `video/webm`.** `.mkv` and `.webm` share the same EBML
  magic bytes, so the sniffer cannot tell them apart. A 4K HEVC movie was served
  as WebM, a container that only permits VP8/VP9/AV1 with Vorbis/Opus.
- New `internal/utils.DetectMimeType` fixes both while keeping sniffing
  authoritative: the file name — which the uploader controls — is consulted
  **only** when the sniffer returned a non-committal answer, and then only to
  promote into an allow-list of **inert media types**. It can never reach
  `text/html`, `image/svg+xml` or `application/xml`, so this does not open
  content-type confusion. Covered by tests including hostile cases (an HTML
  payload named `song.flac` stays `text/html`; `evil.svg` stays
  `application/octet-stream`). All six sniff sites now route through it
  (`upload.go` ×3, `handlers.go`, `receive.go`, `folder.go`).

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

### Added
- **Share page tells you when playback is limited instead of failing silently.**
  A Matroska share now carries a note that browsers play these only partially —
  AC-3/E-AC-3/DTS audio has no browser decoder, so the video runs silently. Plus
  an error note wired to the media element's `error` event, since a `<video>`
  that cannot decode otherwise shows a black rectangle and an `<audio>` stays
  mute, which reads as "the file is broken".
- `<source type="…">` is now omitted for containers a browser cannot evaluate
  via `canPlayType()`, so the browser decides from the bytes rather than
  skipping a source it could have played.
- **`docs/tailscale.md`** — HowTo for running CasaDrop behind Tailscale, covering
  both wiring variants (host Tailscale with mounted CLI/socket vs. the bundled
  `--profile tailscale` sidecar), `serve` vs. `funnel`, share-link host behaviour,
  Taildrop, and troubleshooting. With screenshots.

### Changed
- The Tailscale sidecar now accepts **either** `TAILSCALE_AUTHKEY` or `TS_AUTHKEY`
  in both `docker-compose.yaml` and `docker-compose.zimaos.yaml` — the two files
  previously documented different names for the same variable.

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
