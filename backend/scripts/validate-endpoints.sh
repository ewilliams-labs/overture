#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root_dir="$(cd "$script_dir/.." && pwd)"

server_pid=""
started_server=""
cleanup() {
  if [ -n "$started_server" ] && [ -n "$server_pid" ]; then
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
  fi
}
trap cleanup EXIT

stop_existing_server() {
  local pid=""
  if command -v lsof >/dev/null 2>&1; then
    pid="$(lsof -ti tcp:8080 2>/dev/null || true)"
  elif command -v fuser >/dev/null 2>&1; then
    pid="$(fuser -n tcp 8080 2>/dev/null || true)"
  fi
  if [ -n "$pid" ]; then
    echo "Stopping existing server on :8080 (pid: $pid)..."
    kill $pid 2>/dev/null || true
    sleep 0.5
  fi
}

stop_existing_server

echo "Starting server..."
(
  cd "$root_dir"
  go run cmd/api/main.go
) &
server_pid="$!"
started_server="yes"

ready=""
for _ in {1..30}; do
  status=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/health" || true)
  if [ "$status" = "200" ]; then
    ready="yes"
    break
  fi
  sleep 0.5
done

if [ -z "$ready" ]; then
  echo "Server did not become ready at $BASE_URL" >&2
  exit 1
fi

echo "Running API validation..."
BASE_URL="$BASE_URL" bash "$script_dir/data-validation.sh"
