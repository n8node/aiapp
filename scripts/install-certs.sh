#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

LIVE_DIR="${LIVE_DIR:-/etc/letsencrypt/live/rigintel.ai}"
SSL_DIR="$ROOT/nginx/ssl"

if [ ! -f "$LIVE_DIR/fullchain.pem" ] || [ ! -f "$LIVE_DIR/privkey.pem" ]; then
  echo "Certificate files not found in $LIVE_DIR" >&2
  exit 1
fi

mkdir -p "$SSL_DIR"
cp "$LIVE_DIR/fullchain.pem" "$SSL_DIR/fullchain.pem"
cp "$LIVE_DIR/privkey.pem" "$SSL_DIR/privkey.pem"
chmod 600 "$SSL_DIR/privkey.pem"
chmod 644 "$SSL_DIR/fullchain.pem"

if docker compose --env-file .env -f docker-compose.yml -f docker-compose.prod.yml ps --status running --services 2>/dev/null | grep -qx nginx; then
  docker compose --env-file .env -f docker-compose.yml -f docker-compose.prod.yml exec -T nginx nginx -s reload
fi

echo "Installed TLS files into $SSL_DIR"
