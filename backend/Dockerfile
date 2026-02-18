# syntax=docker/dockerfile:1

# =============================================================================
# Stage 1: Build
# =============================================================================
FROM golang:1.25.6-alpine3.21 AS builder

# Install build dependencies (CGO for sqlite3)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy dependency files first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary with CGO enabled for sqlite3
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /app/overture \
    ./cmd/api

# =============================================================================
# Stage 2: Runtime
# =============================================================================
FROM alpine:3.21.3

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    sqlite-libs \
    tzdata

# Create non-root user for security
RUN addgroup -g 1000 overture && \
    adduser -u 1000 -G overture -s /bin/sh -D overture

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/overture .

# Own the app directory
RUN chown -R overture:overture /app

# Switch to non-root user
USER overture

# Expose the API port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
ENTRYPOINT ["/app/overture"]
