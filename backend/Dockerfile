# =============================================================================
# Go Backend - Multi-stage Dockerfile
# =============================================================================
# Build:  docker build -t drawgo-backend ./backend
# Run:    docker run -p 8080:8080 --env-file .env drawgo-backend
# =============================================================================

# -----------------------------------------------------------------------------
# Stage 1: Build
# -----------------------------------------------------------------------------
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Cache module downloads
COPY go.mod go.sum ./
RUN go mod download

# Build with optimizations: static binary, stripped debug info, embedded version info
COPY . .

# Build args: VERSION and COMMIT_SHA are auto-detected from git when available
# (requires additional_contexts in docker-compose.yml), or can be overridden.
# BUILD_TIME is always auto-detected from the build clock.
ARG VERSION=
ARG COMMIT_SHA=
COPY --from=gitdir . /tmp/.git/
RUN GIT_DIR=/tmp/.git; \
    : "${VERSION:=$(git --git-dir=$GIT_DIR describe --tags --always 2>/dev/null || echo dev)}"; \
    : "${COMMIT_SHA:=$(git --git-dir=$GIT_DIR rev-parse --short HEAD 2>/dev/null || echo unknown)}"; \
    BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ); \
    echo "Building: version=${VERSION} commit=${COMMIT_SHA} time=${BUILD_TIME}"; \
    CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w \
      -X github.com/darwinbatres/drawgo/backend/internal/pkg/buildinfo.Version=${VERSION} \
      -X github.com/darwinbatres/drawgo/backend/internal/pkg/buildinfo.CommitSHA=${COMMIT_SHA} \
      -X github.com/darwinbatres/drawgo/backend/internal/pkg/buildinfo.BuildTime=${BUILD_TIME}" \
    -o server ./cmd/server

# -----------------------------------------------------------------------------
# Stage 2: Run
# -----------------------------------------------------------------------------
FROM alpine:3.21

RUN apk --no-cache add ca-certificates wget postgresql16-client

# Non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/internal/database/migrations ./migrations

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
  CMD wget -q --spider http://localhost:8080/api/v1/health || exit 1

CMD ["./server"]
