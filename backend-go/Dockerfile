# syntax=docker/dockerfile:1.7
#
# DevOps Journey Lab - backend image.
#
# Runtime base choice: alpine (non-root) instead of distroless.
# The compose/Docker healthcheck needs an HTTP client inside the image and the
# server binary has no built-in "healthcheck" subcommand, so a distroless image
# could not be health-checked. alpine adds ~8 MB and gives us busybox wget;
# the process still runs as an unprivileged user on a read-only root filesystem.
# If the binary gains a `healthcheck` subcommand, switch the runtime stage to
# gcr.io/distroless/static-debian12:nonroot and drop wget.

ARG GO_VERSION=1.27
ARG ALPINE_VERSION=3.24

FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /src

# Download modules first so they are cached independently of source changes.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ARG VERSION=dev
ARG TARGETOS=linux
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" \
      -o /out/server ./cmd/server

FROM alpine:${ALPINE_VERSION}

ARG VERSION=dev
LABEL org.opencontainers.image.title="devopslab-backend" \
      org.opencontainers.image.description="DevOps Journey Lab API server" \
      org.opencontainers.image.source="https://github.com/ismoilovdevml/devopsjourney-lab" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.version="${VERSION}"

# The alpine base already ships ca-certificates-bundle and busybox wget, so no
# packages are installed (no network access needed at this stage).
RUN addgroup -S -g 10001 app \
    && adduser -S -D -H -u 10001 -G app -s /sbin/nologin app

WORKDIR /app

COPY --from=builder --chown=root:root /out/server /app/server
COPY --chown=root:root tasks /app/tasks

ENV PORT=8080 \
    GIN_MODE=release \
    TASKS_DIR=/app/tasks

USER 10001:10001

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD ["wget", "-q", "-O", "/dev/null", "http://127.0.0.1:8080/healthz"]

ENTRYPOINT ["/app/server"]
