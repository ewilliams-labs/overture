#!/bin/bash
# with-server.sh - Wait-and-Cleanup pattern for server lifecycle
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
LOG_FILE="${LOG_FILE:-/tmp/overture-server.log}"
HEALTH_URL="${HEALTH_URL:-http://localhost:8080/health}"
MAX_RETRIES="${MAX_RETRIES:-20}"
RETRY_INTERVAL="${RETRY_INTERVAL:-0.5}"

# Build the Go binary
echo "🔨 Building server..."
cd "$PROJECT_ROOT"
go build -o overture ./cmd/api

# Start server in background
echo "🚀 Starting server..."
./overture > "$LOG_FILE" 2>&1 &
PID=$!

# Cleanup on exit
cleanup() {
    echo "🧹 Cleaning up..."
    kill "$PID" 2>/dev/null || true
    rm -f "$PROJECT_ROOT/overture" 2>/dev/null || true
}
trap cleanup EXIT

# Wait for server to become healthy
echo "⏳ Waiting for server health..."
ready=""
for i in $(seq 1 "$MAX_RETRIES"); do
    status=$(curl -s --connect-timeout 1 --max-time 2 -o /dev/null -w "%{http_code}" "$HEALTH_URL" 2>/dev/null || echo "000")
    if [ "$status" = "200" ]; then
        ready="yes"
        echo "✅ Server healthy after $i attempts"
        break
    fi
    sleep "$RETRY_INTERVAL"
done

if [ -z "$ready" ]; then
    echo "❌ Server did not become healthy after $MAX_RETRIES attempts"
    echo "📋 Server logs:"
    cat "$LOG_FILE"
    exit 1
fi

# Pass control to the test command
exec "$@"
