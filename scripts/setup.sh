#!/bin/bash
set -euo pipefail

echo "=== RigIntel setup ==="

mkdir -p nginx/ssl nginx/certbot/www
chmod +x scripts/*.sh wordpress/docker-entrypoint-custom.sh 2>/dev/null || true

if [ ! -f .env ]; then
  if [ "${ENVIRONMENT:-}" = "production" ]; then
    cp .env.production.example .env
  else
    cp .env.example .env
  fi
  echo "Created .env — replace CHANGE_ME values before starting production"
fi

echo "=== Setup complete ==="
echo "  make up     — local HTTP stack"
echo "  make prod   — production HTTPS stack"
