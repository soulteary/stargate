# Release binaries are cross-compiled once by the validated release job.
# Packaging them avoids compiling Go under QEMU and preserves byte identity
# between the downloadable binary and the binary shipped in the image.
FROM alpine:3.24@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf
ARG TARGETARCH
ARG ALPINE_PACKAGE_REFRESH=local
COPY dist/stargate-linux-${TARGETARCH} /bin/stargate
COPY src/internal/web/templates /app/web/templates
RUN echo "Alpine package refresh: ${ALPINE_PACKAGE_REFRESH}" && \
    apk upgrade --no-cache && \
    apk add --no-cache ca-certificates && \
    addgroup -S stargate && \
    adduser -S -D -H -u 10001 -G stargate stargate && \
    chown -R stargate:stargate /app
WORKDIR /app
ENV PORT=8080
USER 10001:10001
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -q -O - "http://127.0.0.1:${PORT:-8080}/healthz" >/dev/null || exit 1
CMD ["stargate"]
