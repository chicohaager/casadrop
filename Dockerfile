# syntax=docker/dockerfile:1.7

########################################
# STAGE 1 — Builder                    #
########################################
# CasaDrop now uses modernc.org/sqlite (pure-Go), so no CGO/gcc/musl-dev
# are required at build time. This shrinks the builder image from ~600 MB
# to the base golang:alpine and removes glibc/musl from the transitive
# dependency graph entirely.
FROM golang:1.25-alpine AS builder

WORKDIR /app

# ca-certificates + tzdata are copied into the runtime image so HTTPS
# handshakes succeed and timestamps render with the configured TZ.
RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/

# Build flags (see "The Anatomy of a 2.5 MB Container"):
#   CGO_ENABLED=0     → static binary, runs on any kernel/glibc-less image
#   -ldflags="-w -s"  → strip DWARF (-w) and symbol table (-s), ~25% smaller
#   -trimpath         → remove local filesystem paths for reproducibility +
#                       so stack traces don't leak the build host's layout
RUN CGO_ENABLED=0 GOOS=linux \
    go build \
        -ldflags="-w -s" \
        -trimpath \
        -o /out/casadrop ./cmd/server

########################################
# STAGE 2 — Runtime (alpine)           #
########################################
# We stay on alpine (not scratch) for the default image because
# scripts/entrypoint.sh does LAN/Tailscale/EasyTier auto-detection via
# /sbin/ip, tailscale, easytier-cli, and su-exec. If you don't need that,
# use Dockerfile.scratch for a ~3 MB image instead.
FROM alpine:3.21

# Minimal runtime utilities:
#   ca-certificates → HTTPS (webhooks, OIDC)
#   tzdata          → configurable TZ
#   wget            → HEALTHCHECK
#   iproute2        → LAN IP detection in entrypoint.sh
#   su-exec         → drop privileges from root to casadrop user
# We deliberately do NOT install bash, curl, git, or package managers —
# attack-surface reduction per the article's Section 1.
RUN apk add --no-cache \
        ca-certificates \
        tzdata \
        wget \
        iproute2 \
        su-exec \
 && rm -rf /var/cache/apk/*

WORKDIR /app

# Non-root user (UID 10001 is high enough to avoid colliding with
# host-managed UIDs on bind mounts).
RUN addgroup -g 10001 -S casadrop \
 && adduser  -u 10001 -S casadrop -G casadrop \
 && mkdir -p /data/uploads /data/thumbnails \
 && chown -R casadrop:casadrop /data /app

COPY --from=builder /out/casadrop /app/casadrop
COPY --chmod=0755 web/        /app/web/
COPY --chmod=0644 logo.png    /app/web/static/logo.png
COPY --chmod=0755 scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh /app/casadrop

ENV PORT=8080 \
    DATA_DIR=/data \
    TEMPLATES_DIR=/app/web/templates \
    STATIC_DIR=/app/web/static

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/api/auth/status >/dev/null 2>&1 || exit 1

ENTRYPOINT ["/entrypoint.sh"]
