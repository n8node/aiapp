# RigIntel Studio image: official Unsloth plus holding-portal overlays.
# Deeper source patches belong in overlay/; do not vendor the upstream tree here.
ARG UNSLOTH_BASE=unsloth/unsloth:latest
FROM ${UNSLOTH_BASE}

ENV UNSLOTH_STUDIO_TRUST_FORWARDED=1 \
    UNSLOTH_STUDIO_DISABLE_PUBLIC_CHECK=1 \
    UNSLOTH_STUDIO_NO_AUTH=1 \
    RIGINTEL_STUDIO_LOCALE=ru

COPY overlay/ /opt/rigintel/studio/
COPY entrypoint.sh /opt/rigintel/studio-entrypoint.sh

USER root
RUN chmod 755 /opt/rigintel/studio-entrypoint.sh
USER unsloth

# Keep the upstream image ENTRYPOINT so Jupyter/Studio still start.
# Optional wrapper: /opt/rigintel/studio-entrypoint.sh
