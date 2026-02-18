# syntax=docker/dockerfile:1

# =============================================================================
# Stage 1: Build
# =============================================================================
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy dependency files first for layer caching
COPY go.mod ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary (no CGO needed for BFF)
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /app/bff \
    ./main.go

# =============================================================================
# Stage 2: Runtime
# =============================================================================
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata

# Create non-root user for security
RUN addgroup -g 1000 bff && \
    adduser -u 1000 -G bff -s /bin/sh -D bff

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bff .

# Own the app directory
RUN chown -R bff:bff /app

# Switch to non-root user
USER bff

# Expose the BFF port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3000/health || exit 1

# Run the binary
ENTRYPOINT ["/app/bff"]
