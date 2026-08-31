#!/bin/bash
set -euo pipefail

WP_URL="${WP_URL:-https://rigintel.ai}"
WP_TITLE="${WP_TITLE:-RigIntel}"
WP_ADMIN_USER="${WP_ADMIN_USER:-admin}"
WP_ADMIN_EMAIL="${WP_ADMIN_EMAIL:-admin@rigintel.ai}"
WP_ADMIN_PASSWORD="${WP_ADMIN_PASSWORD:?Set WP_ADMIN_PASSWORD}"

COMPOSE="docker compose --env-file .env -f docker-compose.yml -f docker-compose.prod.yml"
WP="$COMPOSE exec -T wordpress wp --allow-root"

echo "=== WordPress bootstrap @ $WP_URL ==="

if $WP core is-installed 2>/dev/null; then
  echo "WordPress already installed, skipping core install"
else
  $WP core install \
    --url="$WP_URL" \
    --title="$WP_TITLE" \
    --admin_user="$WP_ADMIN_USER" \
    --admin_email="$WP_ADMIN_EMAIL" \
    --admin_password="$WP_ADMIN_PASSWORD" \
    --skip-email
  echo "WordPress installed"
fi

$WP rewrite structure '/%postname%/' --hard
$WP option update blogdescription 'Корпоративная AI-платформа'
$WP option update timezone_string 'Europe/Moscow'

PAGE_ID=$($WP post list --post_type=page --name=home --field=ID 2>/dev/null || true)
if [ -z "$PAGE_ID" ]; then
  PAGE_ID=$($WP post create \
    --post_type=page \
    --post_title='RigIntel' \
    --post_status=publish \
    --post_content="<p>Корпоративная AI-платформа для сверки документов и чата по базе знаний.</p><p><a href=\"/app\">Открыть приложение →</a></p>" \
    --porcelain)
fi
$WP option update show_on_front page
$WP option update page_on_front "$PAGE_ID"

echo "=== Done ==="
echo "  Site: $WP_URL"
echo "  App:  ${WP_URL}/app"
echo "  Admin: ${WP_URL}/wp-admin"
