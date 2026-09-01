#!/bin/sh
set -eu

# Studio defaults to 127.0.0.1. Nginx and the Go API reach it via Docker DNS,
# so it must bind 0.0.0.0 or every hop returns 502.
LOCALE="${RIGINTEL_STUDIO_LOCALE:-ru}"
AUTH_DIR="${UNSLOTH_STUDIO_AUTH_DIR:-/home/unsloth/.unsloth/auth}"
AUTH_HOME="${UNSLOTH_STUDIO_HOME:-/home/unsloth/.unsloth}"
# Upstream STUDIO_HOME is ~/.unsloth/studio; do not reuse AUTH_HOME for it.
STUDIO_HOME="${STUDIO_HOME:-/home/unsloth/.unsloth/studio}"
INSTALLER_BIN="${STUDIO_HOME}/unsloth_studio/bin/unsloth"
DIST_STORE="/workspace/work/.rigintel-studio-frontend"

mkdir -p "$AUTH_DIR" /tmp/rigintel-studio /workspace/work 2>/dev/null || true

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

# Keep Hugging Face downloads on the work volume; container recreate otherwise
# drops /workspace/.cache and leaves 0-byte .incomplete blobs.
if [ -z "${HF_HOME:-}" ]; then
  export HF_HOME="/workspace/work/.hf-cache"
fi
mkdir -p "$HF_HOME" 2>/dev/null || true

apply_outbound_proxy() {
  token="${RIGINTEL_INTERNAL_TOKEN:-}"
  [ -n "$token" ] || return 0
  i=0
  while [ "$i" -lt 8 ]; do
    code=$(curl -sS -o /tmp/rigintel-https-proxy -w "%{http_code}" --max-time 3 \
      -H "X-RigIntel-Internal: ${token}" \
      http://backend:8080/api/v1/internal/outbound-proxy 2>/dev/null || printf '000')
    if [ "$code" = "200" ]; then
      PROXY_URL=$(tr -d '\r\n' < /tmp/rigintel-https-proxy)
      rm -f /tmp/rigintel-https-proxy
      if [ -n "$PROXY_URL" ]; then
        export HTTP_PROXY="$PROXY_URL"
        export HTTPS_PROXY="$PROXY_URL"
        export http_proxy="$PROXY_URL"
        export https_proxy="$PROXY_URL"
        export ALL_PROXY="$PROXY_URL"
        export NO_PROXY="localhost,127.0.0.1,backend,postgres,nginx,model-gateway,ai-runtime,unsloth-studio"
        export no_proxy="$NO_PROXY"
        echo "Outbound HTTP proxy enabled for Hugging Face" >&2
      fi
      return 0
    fi
    i=$((i + 1))
    sleep 1
  done
  rm -f /tmp/rigintel-https-proxy
}

apply_outbound_proxy

# Installer binary must win over conda/venv `unsloth` (no frontend dist).
export PATH="/home/unsloth/.bun/bin:${STUDIO_HOME}/unsloth_studio/bin:/opt/conda/bin:/usr/local/bin:${PATH:-/usr/bin}"
if [ -f /opt/conda/etc/profile.d/conda.sh ]; then
  # shellcheck disable=SC1091
  . /opt/conda/etc/profile.d/conda.sh
fi

has_dist() {
  [ -f "$1/index.html" ]
}

resolve_frontend() {
  if has_dist "$DIST_STORE"; then
    printf '%s\n' "$DIST_STORE"
    return 0
  fi
  for p in \
    "${STUDIO_HOME}/unsloth_studio/lib/python3.12/site-packages/studio/frontend/dist" \
    "${STUDIO_HOME}/unsloth_studio/lib/python3.11/site-packages/studio/frontend/dist" \
    /opt/venv/lib/python3.12/site-packages/studio/frontend/dist \
    /opt/venv/lib/python3.11/site-packages/studio/frontend/dist \
    /workspace/studio/frontend/dist
  do
    if has_dist "$p"; then
      printf '%s\n' "$p"
      return 0
    fi
  done
  return 1
}

build_frontend() {
  for src in \
    /opt/venv/lib/python3.12/site-packages/studio/frontend \
    /opt/venv/lib/python3.11/site-packages/studio/frontend \
    "${STUDIO_HOME}/unsloth_studio/lib/python3.12/site-packages/studio/frontend" \
    /workspace/studio/frontend
  do
    [ -f "$src/package.json" ] || continue
    echo "Building Studio frontend from $src (skipping vendor npm/flash-attn first-run)" >&2
    cd "$src"
    NODE_DIR="/workspace/work/.rigintel-node"
    if [ ! -x "$NODE_DIR/bin/npm" ]; then
      echo "Fetching Node.js into $NODE_DIR" >&2
      curl -fsSL --max-time 120 https://nodejs.org/dist/v22.14.0/node-v22.14.0-linux-x64.tar.gz -o /tmp/rigintel-node.tar.gz
      mkdir -p /tmp/rigintel-node
      tar -C /tmp/rigintel-node -xzf /tmp/rigintel-node.tar.gz
      mkdir -p "$NODE_DIR"
      cp -a /tmp/rigintel-node/node-v22.14.0-linux-x64/. "$NODE_DIR/"
    fi
    export PATH="$NODE_DIR/bin:$PATH"
    npm install --ignore-scripts --fetch-timeout=60000 --fetch-retries=2
    npm run build || npx vite build
    mkdir -p "$DIST_STORE"
    cp -a "$src/dist/." "$DIST_STORE/"
    printf '%s\n' "$DIST_STORE"
    return 0
  done
  echo "Studio frontend source not found" >&2
  return 1
}

FRONTEND="$(resolve_frontend || true)"
if [ -z "$FRONTEND" ]; then
  FRONTEND="$(build_frontend)"
fi

launch_studio() {
  export PATH="/workspace/work/.rigintel-node/bin:/home/unsloth/.bun/bin:${STUDIO_HOME}/unsloth_studio/bin:/opt/venv/bin:/opt/conda/bin:/usr/local/bin:${PATH:-/usr/bin}"
  if [ -x "$INSTALLER_BIN" ]; then
    exec "$INSTALLER_BIN" studio -H 0.0.0.0 -p 8000 --frontend "$FRONTEND"
  fi
  for b in /opt/venv/bin/unsloth /opt/conda/bin/unsloth; do
    if [ -x "$b" ]; then
      exec "$b" studio -H 0.0.0.0 -p 8000 --frontend "$FRONTEND"
    fi
  done
  exec unsloth studio -H 0.0.0.0 -p 8000 --frontend "$FRONTEND"
}

if [ "$#" -eq 0 ]; then
  launch_studio
fi

# Compose passes `unsloth studio ...` — keep host/port, always pin frontend dist.
case " $* " in
  *" --frontend "*)
    exec "$@"
    ;;
esac
if [ "$1" = "unsloth" ] || [ "$1" = "$INSTALLER_BIN" ]; then
  exec "$1" studio -H 0.0.0.0 -p 8000 --frontend "$FRONTEND"
fi
exec "$@" --frontend "$FRONTEND"
