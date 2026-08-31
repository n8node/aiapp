#!/bin/bash
set -euo pipefail

fix_wp_content_permissions() {
	local wp_content="/var/www/html/wp-content"
	mkdir -p \
		"${wp_content}/uploads" \
		"${wp_content}/themes" \
		"${wp_content}/plugins" \
		"${wp_content}/mu-plugins" \
		"${wp_content}/upgrade" \
		"${wp_content}/cache"
	chown -R www-data:www-data "${wp_content}"
	chmod -R ug+rwX "${wp_content}"
}

fix_wp_content_permissions

if [ "${1:-}" = "apache2-foreground" ]; then
	docker-entrypoint.sh apache2-foreground &
	pid=$!
	for _ in $(seq 1 50); do
		if [ -f /var/run/apache2/apache2.pid ] || [ -f /run/apache2/apache2.pid ]; then
			break
		fi
		sleep 0.2
	done
	fix_wp_content_permissions
	wait "$pid"
	exit $?
fi

exec docker-entrypoint.sh "$@"
