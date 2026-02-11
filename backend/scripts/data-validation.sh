#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
PLAYLIST_NAME="Vibe Check"
TRACK_TITLE="Happy"
TRACK_ARTIST="Pharrell Williams"

COLOR_GREEN="\033[0;32m"
COLOR_RESET="\033[0m"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

require_cmd curl
require_cmd jq

tmp_file="$(mktemp)"
cleanup() {
  rm -f "$tmp_file"
}
trap cleanup EXIT

echo "Checking health endpoint..."
health_status=$(curl -s -o "$tmp_file" -w "%{http_code}" "$BASE_URL/health" || true)
if [ "$health_status" != "200" ]; then
  echo "Health check failed with status $health_status" >&2
  cat "$tmp_file" >&2
  exit 1
fi
if ! jq -e '.status == "ok"' "$tmp_file" >/dev/null; then
  echo "Health response missing status=ok" >&2
  cat "$tmp_file" >&2
  exit 1
fi

echo "Creating playlist..."
create_status=$(curl -s -o "$tmp_file" -w "%{http_code}" \
  -H "Content-Type: application/json" \
  -d "{\"name\": \"$PLAYLIST_NAME\"}" \
  "$BASE_URL/playlists")
if [ "$create_status" != "201" ]; then
  echo "Create playlist failed with status $create_status" >&2
  cat "$tmp_file" >&2
  exit 1
fi
PLAYLIST_ID=$(jq -r '.id // empty' "$tmp_file")
if [ -z "$PLAYLIST_ID" ]; then
  echo "Create playlist response missing id" >&2
  cat "$tmp_file" >&2
  exit 1
fi
if ! jq -e --arg name "$PLAYLIST_NAME" '.name == $name' "$tmp_file" >/dev/null; then
  echo "Create playlist response missing expected name" >&2
  cat "$tmp_file" >&2
  exit 1
fi

echo "Playlist ID: $PLAYLIST_ID"

echo "Adding track..."
add_status=$(curl -s -o "$tmp_file" -w "%{http_code}" \
  -H "Content-Type: application/json" \
  -d "{\"title\": \"$TRACK_TITLE\", \"artist\": \"$TRACK_ARTIST\"}" \
  "$BASE_URL/playlists/$PLAYLIST_ID/tracks")
if [ "$add_status" != "201" ]; then
  echo "Add track failed with status $add_status" >&2
  cat "$tmp_file" >&2
  exit 1
fi
if ! jq -e --arg id "$PLAYLIST_ID" '.id == $id' "$tmp_file" >/dev/null; then
  echo "Add track response missing playlist id" >&2
  cat "$tmp_file" >&2
  exit 1
fi

printf "%b\n" "${COLOR_GREEN}Validation passed.${COLOR_RESET}"