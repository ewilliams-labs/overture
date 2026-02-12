#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
TEST_DATA="${TEST_DATA:-./scripts/acceptence_cases.json}"

wait_for_ready() {
  local ready=""
  for _ in $(seq 1 30); do
    status=$(curl -s --connect-timeout 1 --max-time 2 -o /dev/null -w "%{http_code}" "$BASE_URL/health" || true)
    if [ "$status" = "200" ]; then
      ready="yes"
      break
    fi
    sleep 0.5
  done

  if [ -z "$ready" ]; then
    echo "Server did not become ready at $BASE_URL" >&2
    return 1
  fi
}

wait_for_ready
TEST_DATA="$TEST_DATA" BASE_URL="$BASE_URL" bash ./scripts/validate-api.sh
