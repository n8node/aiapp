# Overlay image for Unsloth Studio. Do not change USER: upstream is unsloth:runtimeusers.
ARG UNSLOTH_BASE=unsloth/unsloth:latest
FROM ${UNSLOTH_BASE}

ENV UNSLOTH_STUDIO_TRUST_FORWARDED=1 \
    UNSLOTH_STUDIO_DISABLE_PUBLIC_CHECK=1 \
    RIGINTEL_STUDIO_LOCALE=ru

COPY --chmod=755 overlay/ /opt/rigintel/studio/
COPY --chmod=755 entrypoint.sh /opt/rigintel/studio-entrypoint.sh
ENTRYPOINT ["/opt/rigintel/studio-entrypoint.sh"]
