#!/bin/bash
set -e

BASE_URL="${BASE_URL:-http://localhost:8080}"
TEST_DATA="${TEST_DATA:-./scripts/acceptence_cases.json}"

echo "🚀 Starting Overture Acceptance Suite..."

tmp_file=$(mktemp)
cleanup() {
    rm -f "$tmp_file"
}
trap cleanup EXIT

# 1. Create a temporary playlist for this test run
PLAYLIST_NAME="CI_TEST_$(date +%s)"
CREATE_RESP=$(curl -s -X POST "$BASE_URL/playlists" -H "Content-Type: application/json" -d "{\"name\": \"$PLAYLIST_NAME\"}")
PLAYLIST_ID=$(echo "$CREATE_RESP" | jq -r '.id')

echo "✅ Created Test Playlist: $PLAYLIST_ID"

# 2. Iterate through cases
cat "$TEST_DATA" | jq -c '.[]' | while read -r test; do
    ID=$(echo "$test" | jq -r '.id')
    METHOD=$(echo "$test" | jq -r '.method')
    # Replace placeholder with actual ID
    PATH_RAW=$(echo "$test" | jq -r '.path')
    URL="${BASE_URL}${PATH_RAW//\{\{playlist_id\}\}/$PLAYLIST_ID}"
    EXPECTED=$(echo "$test" | jq -r '.expected_status')
    PAYLOAD=$(echo "$test" | jq -c '.payload // empty')
    BODY_CONTAINS=$(echo "$test" | jq -r '.expected_body_contains // empty')
    JSON_PATH=$(echo "$test" | jq -r '.expected_json_path // empty')
    JSON_MIN=$(echo "$test" | jq -r '.expected_json_min // empty')

    echo -n "🧪 Testing $ID... "

    if [[ -n "$PAYLOAD" ]]; then
        STATUS=$(curl -s -o "$tmp_file" -w "%{http_code}" -X "$METHOD" "$URL" -H "Content-Type: application/json" -d "$PAYLOAD")
    else
        STATUS=$(curl -s -o "$tmp_file" -w "%{http_code}" -X "$METHOD" "$URL")
    fi

    if [ "$STATUS" -ne "$EXPECTED" ]; then
        echo "❌ FAILED (Expected $EXPECTED, got $STATUS)"
        cat "$tmp_file"
        exit 1
    fi

    if [ -n "$BODY_CONTAINS" ] && ! grep -q "$BODY_CONTAINS" "$tmp_file"; then
        echo "❌ FAILED (Body missing $BODY_CONTAINS)"
        cat "$tmp_file"
        exit 1
    fi

    if [ -n "$JSON_PATH" ] && [ -n "$JSON_MIN" ]; then
        actual=$(jq -r "$JSON_PATH" "$tmp_file" 2>/dev/null || true)
        if ! [[ "$actual" =~ ^-?[0-9]+([.][0-9]+)?$ ]]; then
            echo "❌ FAILED (Expected numeric result at $JSON_PATH)"
            cat "$tmp_file"
            exit 1
        fi
        if ! awk -v a="$actual" -v b="$JSON_MIN" 'BEGIN { exit !(a >= b) }'; then
            echo "❌ FAILED (Expected $JSON_PATH >= $JSON_MIN, got $actual)"
            cat "$tmp_file"
            exit 1
        fi
    fi

    echo "✅ (Status $STATUS)"
done

echo "🎉 All acceptance tests passed!"