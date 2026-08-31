#!/usr/bin/env bash
set -euo pipefail

COMPOSE=(
  docker compose --env-file .env
  -f docker-compose.yml
  -f docker-compose.prod.yml
)
DOMAIN="${RIGINTEL_PUBLIC_URL:-https://rigintel.ai}"
ATTEMPTS="${VERIFY_ATTEMPTS:-12}"
SLEEP_SECONDS="${VERIFY_SLEEP_SECONDS:-2}"

require_running() {
  local service="$1"
  if ! "${COMPOSE[@]}" ps --status running --services | grep -qx "$service"; then
    echo "Service is not running: $service" >&2
    "${COMPOSE[@]}" logs --tail=40 "$service" >&2 || true
    return 1
  fi
}

for service in nginx backend frontend; do
  require_running "$service"
done

for attempt in $(seq 1 "$ATTEMPTS"); do
  if curl --fail --silent --show-error --max-time 10 \
    --resolve "rigintel.ai:443:127.0.0.1" \
    "$DOMAIN/app/health" >/dev/null \
    && curl --fail --silent --show-error --max-time 10 --location \
      --resolve "rigintel.ai:443:127.0.0.1" \
      "$DOMAIN/app/" >/dev/null; then
    echo "Release verified: $DOMAIN/app/health and $DOMAIN/app/"
    exit 0
  fi

  if [ "$attempt" -lt "$ATTEMPTS" ]; then
    sleep "$SLEEP_SECONDS"
  fi
done

echo "Release verification failed. Recent service logs:" >&2
"${COMPOSE[@]}" logs --tail=80 nginx frontend backend >&2
exit 1
