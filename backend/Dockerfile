# 1. Use the patched Go version (matches fixed version in logs)
FROM golang:1.25-alpine3.21 AS builder

# 2. Install build dependencies AND upgrade existing packages to fix OpenSSL
RUN apk update && apk upgrade --no-cache && apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Build with CGO enabled for SQLite
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /app/overture \
    ./cmd/api

FROM alpine:3.21.3


RUN apk update && apk upgrade --no-cache && apk add --no-cache gcc musl-dev

WORKDIR /app

# Non-root user setup remains the same
RUN addgroup -g 1000 overture && \
    adduser -u 1000 -G overture -s /bin/sh -D overture

COPY --from=builder /app/overture .
RUN chown -R overture:overture /app

USER overture
EXPOSE 8080
CMD ["./overture"]