#!/bin/bash
set -euo pipefail

echo "=== RigIntel server bootstrap (Ubuntu 24.04) ==="

if [ "$(id -u)" -ne 0 ]; then
  echo "Run as root" >&2
  exit 1
fi

export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y git make curl ca-certificates ufw certbot

# Host nginx/apache would steal :80/:443 from the Compose edge.
systemctl disable --now apache2 nginx 2>/dev/null || true

if ! command -v docker >/dev/null 2>&1; then
  apt-get install -y docker.io docker-compose-v2
fi

systemctl enable --now docker
docker --version
docker compose version

ufw allow OpenSSH
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable

mkdir -p /opt
echo "=== Bootstrap complete ==="
echo "Next: clone to /opt/aiapp and follow scripts/deploy-server.md"
echo "Occupied HTTP/HTTPS sockets:"
ss -tlnp | grep -E ':80|:443' || echo "(none)"
