# --- BUILD STAGE ---
FROM golang:1.25-alpine3.21 AS builder

# Install build dependencies and upgrade existing packages
RUN apk update && apk upgrade --no-cache && apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy mod files FIRST for caching
COPY go.mod go.sum ./

# Force 1.25.7 download if needed
RUN GOTOOLCHAIN=auto go mod download

COPY . .

# Build Backend binary
RUN GOTOOLCHAIN=auto CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /app/overture \
    ./cmd/api


# --- RUNTIME STAGE ---
FROM alpine:3.23.3

# Install RUNTIME dependencies and patch OS for vulnerabilities
RUN apk update && apk upgrade --no-cache && apk add --no-cache ca-certificates sqlite-libs tzdata

WORKDIR /app

# Non-root user setup
RUN addgroup -g 1000 overture && \
    adduser -u 1000 -G overture -s /bin/sh -D overture

COPY --from=builder /app/overture .
RUN chown -R overture:overture /app

USER overture
EXPOSE 8080

# Execute Backend
CMD ["./overture"]