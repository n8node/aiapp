#!/bin/sh
set -eu

# Studio defaults to 127.0.0.1. Nginx and the Go API reach it via Docker DNS,
# so it must bind 0.0.0.0 or every hop returns 502.
LOCALE="${RIGINTEL_STUDIO_LOCALE:-ru}"
AUTH_DIR="${UNSLOTH_STUDIO_AUTH_DIR:-/home/unsloth/.unsloth/auth}"
AUTH_HOME="${UNSLOTH_STUDIO_HOME:-/home/unsloth/.unsloth}"

mkdir -p "$AUTH_DIR" /tmp/rigintel-studio 2>/dev/null || true

printf '%s\n' "$LOCALE" > /tmp/rigintel-studio/locale
if [ -d "$AUTH_HOME" ]; then
  printf '%s\n' "$LOCALE" > "$AUTH_HOME/rigintel_locale" || true
fi

# Re-passing UNSLOTH_STUDIO_PASSWORD after auth exists is a hard process exit.
auth_seeded=0
if [ -d "$AUTH_DIR" ]; then
  for f in "$AUTH_DIR"/* "$AUTH_DIR"/.[!.]*; do
    [ -e "$f" ] || continue
    auth_seeded=1
    break
  done
fi
if [ "$auth_seeded" = 1 ]; then
  unset UNSLOTH_STUDIO_PASSWORD
fi

export PATH="/opt/conda/bin:/usr/local/bin:${PATH:-/usr/bin}"
if [ -f /opt/conda/etc/profile.d/conda.sh ]; then
  # shellcheck disable=SC1091
  . /opt/conda/etc/profile.d/conda.sh
fi

if [ "$#" -gt 0 ]; then
  exec "$@"
fi

exec unsloth studio -H 0.0.0.0 -p 8000
