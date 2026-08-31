#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

DOMAIN="${DOMAIN:-rigintel.ai}"
WWW_DOMAIN="${WWW_DOMAIN:-www.rigintel.ai}"
EMAIL="${CERTBOT_EMAIL:-}"
MODE="${1:-standalone}"
COMPOSE=(docker compose --env-file .env -f docker-compose.yml -f docker-compose.prod.yml)

if [ -f .env ]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
  EMAIL="${CERTBOT_EMAIL:-$EMAIL}"
fi

if [ -z "$EMAIL" ]; then
  echo "Set CERTBOT_EMAIL in .env" >&2
  exit 1
fi

if [ "$(id -u)" -ne 0 ]; then
  echo "Run as root so certbot can write /etc/letsencrypt" >&2
  exit 1
fi

echo "Requesting Let's Encrypt certificate for $DOMAIN and $WWW_DOMAIN"
echo "Do not include bitrix.rigintel.ai — it is a separate host."

if ! command -v certbot >/dev/null 2>&1; then
  echo "Installing certbot..."
  export DEBIAN_FRONTEND=noninteractive
  apt-get update
  apt-get install -y certbot
fi
if ! command -v certbot >/dev/null 2>&1; then
  echo "certbot is not installed. Run: apt-get install -y certbot" >&2
  exit 1
fi

port_in_use() {
  ss -tln | grep -qE ':80\s'
}

case "$MODE" in
  standalone)
    if "${COMPOSE[@]}" ps --services >/dev/null 2>&1; then
      "${COMPOSE[@]}" stop nginx >/dev/null 2>&1 || true
    fi
    for _ in $(seq 1 15); do
      if ! port_in_use; then
        break
      fi
      sleep 1
    done
    if port_in_use; then
      echo "Port 80 is still busy after stopping nginx." >&2
      ss -tlnp | grep -E ':80|:443' || true
      exit 1
    fi
    certbot certonly --standalone \
      --non-interactive --agree-tos --email "$EMAIL" \
      -d "$DOMAIN" -d "$WWW_DOMAIN"
    ;;
  webroot)
    mkdir -p "$ROOT/nginx/certbot/www"
    certbot certonly --webroot -w "$ROOT/nginx/certbot/www" \
      --non-interactive --agree-tos --email "$EMAIL" \
      -d "$DOMAIN" -d "$WWW_DOMAIN"
    ;;
  *)
    echo "Usage: $0 [standalone|webroot]" >&2
    exit 1
    ;;
esac

bash "$ROOT/scripts/install-certs.sh"

if ! crontab -l 2>/dev/null | grep -q "aiapp/scripts/install-certs.sh"; then
  (crontab -l 2>/dev/null || true; echo "17 3 * * * certbot renew --webroot -w $ROOT/nginx/certbot/www --deploy-hook $ROOT/scripts/install-certs.sh >/var/log/aiapp-certbot.log 2>&1") | crontab -
  echo "Installed daily certbot renew cron"
fi

if [ ! -f "$ROOT/nginx/ssl/fullchain.pem" ] || [ ! -f "$ROOT/nginx/ssl/privkey.pem" ]; then
  echo "TLS files were not installed; not starting nginx." >&2
  exit 1
fi

if "${COMPOSE[@]}" ps --services >/dev/null 2>&1; then
  "${COMPOSE[@]}" up -d nginx
fi

echo "Certificate ready."
