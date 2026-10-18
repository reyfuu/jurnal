#!/usr/bin/env bash
set -euo pipefail

# Laporan Bisnis by Akademi Pajak - VPS Deployment Script
# Usage: ./deploy.sh [PORT]

APP_PORT="${1:-8080}"
IMAGE_NAME="laporan-bisnis:latest"
CONTAINER_NAME="laporan-bisnis"

echo "=== [1/4] Pulling latest updates from origin main ==="
git pull origin main

echo "=== [2/4] Building Docker container ==="
docker build -t "$IMAGE_NAME" .

echo "=== [3/4] Restarting container on port $APP_PORT ==="
docker rm -f "$CONTAINER_NAME" 2>/dev/null || true
docker run -d \
  --name "$CONTAINER_NAME" \
  --restart unless-stopped \
  -p "${APP_PORT}:8080" \
  "$IMAGE_NAME"

echo "=== [4/4] Verifying deployment ==="
sleep 2
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${APP_PORT}/" || echo "000")
if [ "$HTTP_CODE" = "200" ]; then
  echo "✓ Laporan Bisnis successfully deployed & running on http://localhost:${APP_PORT}"
else
  echo "⚠ Warning: Expected HTTP 200, got: $HTTP_CODE"
fi
