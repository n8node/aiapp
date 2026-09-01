#!/bin/sh
set -eu

LOCALE="${RIGINTEL_STUDIO_LOCALE:-ru}"
AUTH_HOME="${UNSLOTH_STUDIO_HOME:-/home/unsloth/.unsloth}"
mkdir -p "$AUTH_HOME" /tmp/rigintel-studio 2>/dev/null || true

# Persist the platform locale so Studio hydrates Russian (or the admin choice)
# instead of English on first paint. Studio's key is unsloth_locale.
printf '%s\n' "$LOCALE" > /tmp/rigintel-studio/locale
if [ -d "$AUTH_HOME" ]; then
  printf '%s\n' "$LOCALE" > "$AUTH_HOME/rigintel_locale" || true
fi

# Prefer the upstream image command; keep GPU/Jupyter launch intact.
if [ "$#" -gt 0 ]; then
  exec "$@"
fi

for candidate in \
  /opt/unsloth/entrypoint.sh \
  /usr/local/bin/unsloth-entrypoint \
  /usr/local/bin/studio \
  /opt/unsloth/start.sh
do
  if [ -x "$candidate" ]; then
    exec "$candidate"
  fi
done

if command -v unsloth >/dev/null 2>&1; then
  exec unsloth studio --host 0.0.0.0 --port 8000
fi

echo "rigintel-studio: no upstream entrypoint found" >&2
exit 1
