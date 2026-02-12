#!/bin/bash
set -e

BASE_URL="http://localhost:8080"
TEST_DATA="tests/acceptance_cases.json"

echo "🚀 Starting Overture Acceptance Suite..."

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

    echo -n "🧪 Testing $ID... "

    if [[ -n "$PAYLOAD" ]]; then
        STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X "$METHOD" "$URL" -H "Content-Type: application/json" -d "$PAYLOAD")
    else
        STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X "$METHOD" "$URL")
    fi

    if [ "$STATUS" -eq "$EXPECTED" ]; then
        echo "✅ (Status $STATUS)"
    else
        echo "❌ FAILED (Expected $EXPECTED, got $STATUS)"
        exit 1
    fi
done

echo "🎉 All acceptance tests passed!"