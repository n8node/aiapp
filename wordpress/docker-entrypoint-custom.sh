#!/bin/bash
set -euo pipefail

fix_wp_content_permissions() {
	local wp_content="/var/www/html/wp-content"
	for subdir in uploads themes plugins mu-plugins upgrade cache; do
		local dir="${wp_content}/${subdir}"
		mkdir -p "${dir}"
		chown -R www-data:www-data "${dir}" 2>/dev/null || true
		chmod -R ug+rwX "${dir}" 2>/dev/null || true
	done
}

fix_wp_content_permissions

exec docker-entrypoint.sh "$@"
