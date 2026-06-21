#!/bin/sh
# Use PORT env var if set (Render injects it), otherwise default to 8080
PORT=${PORT:-8080}
TARGETS=${TARGETS:-"https://example.com,https://google.com,https://github.com"}
INTERVAL=${INTERVAL:-30s}

exec ./go-probe \
  --targets="$TARGETS" \
  --interval="$INTERVAL" \
  --addr=":$PORT"
