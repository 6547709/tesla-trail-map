# syntax=docker/dockerfile:1.7
# ─────────────────────────────────────────────────────────────────────────────
# Multi-arch Dockerfile for tesla-trail-map
# Builds on x86_64 / arm64 hosts via `docker buildx build --platform=...`.
# Buildx auto-fills TARGETOS / TARGETARCH; we pass them to `go build` to
# cross-compile the Go binary natively (no QEMU compile step needed).
# ─────────────────────────────────────────────────────────────────────────────

ARG GO_VERSION=1.25

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS build

# Buildx auto-populates these — never set them by hand.
ARG TARGETOS
ARG TARGETARCH
# App version baked into the binary via -ldflags.
ARG VERSION=dev

WORKDIR /src

# Cache go modules independently from the source so most rebuilds skip download.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY . .

# Cross-compile to the requested target. CGO off keeps the binary static.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build \
      -trimpath \
      -ldflags="-s -w -X main.Version=${VERSION#v}" \
      -o /out/app .

# ─────────────────────────────────────────────────────────────────────────────
# Runtime image — small, multi-arch, with timezone data + CA bundle.
# ─────────────────────────────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S -G app app
# TZ is intentionally NOT defaulted here — tzdata is installed above, so
# the runtime can set it via `docker run -e TZ=...` or compose. Default
# would otherwise depend on the image maintainer's region (Asia/Shanghai
# is the wrong default for anyone outside China).

WORKDIR /app
COPY --from=build /out/app /app/app
COPY --from=build /src/map.html /app/map.html

USER app
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/health >/dev/null 2>&1 || exit 1

ENTRYPOINT ["/app/app"]

# OCI labels — populated by the GitHub Actions workflow on release.
ARG VCS_REF=unknown
ARG BUILD_DATE=unknown
LABEL org.opencontainers.image.title="tesla-trail-map" \
      org.opencontainers.image.description="TeslaMate drive-trail playback (Go + Leaflet)" \
      org.opencontainers.image.source="https://github.com/6547709/tesla-trail-map" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${VCS_REF}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.licenses="UNLICENSED"
