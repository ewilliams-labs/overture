#!/bin/bash
# env-setup.sh - Environment detection for WSL2 and Ollama connectivity
set -euo pipefail

# Detect Windows Host IP (for WSL2 connectivity)
if grep -qi microsoft /proc/version 2>/dev/null; then
    WIN_HOST_IP=$(ip route show | grep default | awk '{print $3}')
    export OLLAMA_HOST="http://${WIN_HOST_IP}:11434"
    echo "🔍 WSL2 detected. Using Windows host at ${WIN_HOST_IP}"
else
    export OLLAMA_HOST="${OLLAMA_HOST:-http://localhost:11434}"
    echo "🔍 Native Linux detected. Using OLLAMA_HOST=${OLLAMA_HOST}"
fi

# Check if Ollama is reachable
OLLAMA_STATUS=$(curl -s --connect-timeout 2 -o /dev/null -w "%{http_code}" "${OLLAMA_HOST}/api/tags" 2>/dev/null || echo "000")

if [ "$OLLAMA_STATUS" = "200" ]; then
    export RUN_AI_TESTS="true"
    echo "✅ Ollama detected at ${OLLAMA_HOST}. Enabling AI tests."
    
    # Auto-detect model if not specified
    if [ -z "${OLLAMA_MODEL:-}" ]; then
        OLLAMA_MODEL=$(curl -s "${OLLAMA_HOST}/api/tags" 2>/dev/null | jq -r '.models[0].name // empty')
        if [ -n "$OLLAMA_MODEL" ]; then
            export OLLAMA_MODEL
            echo "📦 Using model: ${OLLAMA_MODEL}"
        fi
    fi
else
    export RUN_AI_TESTS="false"
    echo "⚠️  Ollama not reachable (status: ${OLLAMA_STATUS}). Skipping AI tests."
fi

# Pass control to the next command
exec "$@"
