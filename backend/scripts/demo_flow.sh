#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
mkdir -p "$script_dir"
root_dir=$(cd "$script_dir/.." && pwd)
state_file="$root_dir/.last_playlist_id"
base_url="${BASE_URL:-http://localhost:8080}"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

require_cmd curl
require_cmd jq

usage() {
  echo "Usage:" >&2
  echo "  $0 create-playlist [name]" >&2
  echo "  $0 add-track <title> <artist>" >&2
}

cmd="${1:-}"
case "$cmd" in
  create-playlist)
    name="${2:-Vibe Check}"
    resp=$(curl -s -X POST "$base_url/playlists" -H "Content-Type: application/json" -d "{\"name\":\"$name\"}")
    echo "$resp"
    playlist_id=$(echo "$resp" | jq -r '.id // empty')
    if [ -z "$playlist_id" ]; then
      echo "❌ Error: playlist id missing!" >&2
      exit 1
    fi
    echo "$playlist_id" > "$state_file"
    ;;
  add-track)
    title="${2:-}"
    artist="${3:-}"
    if [ -z "$title" ] || [ -z "$artist" ]; then
      usage
      exit 1
    fi
    if [ ! -f "$state_file" ]; then
      echo "❌ Error: no playlist id found. Run create-playlist first." >&2
      exit 1
    fi
    playlist_id=$(cat "$state_file")
    resp=$(curl -s -X POST "$base_url/playlists/$playlist_id/tracks" -H "Content-Type: application/json" -d "{\"title\":\"$title\",\"artist\":\"$artist\"}")
    echo "$resp"

    if echo "$resp" | jq -e '.error // empty' >/dev/null; then
      echo "❌ Error: add-track failed." >&2
      exit 1
    fi

    playlist_resp=$(curl -s "$base_url/playlists/$playlist_id")
    track_id=$(echo "$playlist_resp" | jq -r '.tracks[-1].id // empty')
    sleep 2
    verify_resp=$(curl -s "$base_url/playlists/$playlist_id")
    energy=$(echo "$verify_resp" | jq -r '.tracks[-1].features.energy // 0')
    if awk -v e="$energy" 'BEGIN { exit !(e == 0.95) }'; then
      echo "✅ [SUCCESS] Background Worker updated features (Energy: 0.95)."
      exit 0
    elif awk -v e="$energy" 'BEGIN { exit !(e > 0 && e != 0.95) }'; then
      echo "✅ [SUCCESS] No Preview URL found. Fallback logic is active (Energy: $energy)."
      exit 0
    else
      echo "❌ [FAILURE] Energy is 0.0. Data pipeline is broken."
      exit 1
    fi
    if [ -n "$track_id" ]; then
      echo "⚠️  Verify server logs show: 'Processed $track_id'"
    else
      echo "⚠️  Verify server logs show: 'Processed [TrackID]'"
    fi
    ;;
  *)
    usage
    exit 1
    ;;
esac
