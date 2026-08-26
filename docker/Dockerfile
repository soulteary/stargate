FROM golang:1.27.0-alpine3.24 AS builder
RUN apk add git bash gcc musl-dev upx
WORKDIR /app
ENV GO111MODULE=on
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

# 定义构建参数，提供默认值以支持本地构建
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE

# 多平台：TARGETOS/TARGETARCH 由 Docker Buildx 注入；本地 build 时用当前内核架构，避免 exec format error（如 ARM Mac）
ARG TARGETOS
ARG TARGETARCH
RUN BUILD_DATE=${BUILD_DATE:-$(date +%FT%T%z)} && \
    ARCH=${TARGETARCH:-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/;s/armv7l/arm/')} && \
    OS=${TARGETOS:-linux} && \
    GO111MODULE=on GOOS=$OS GOARCH=$ARCH CGO_ENABLED=0 go build -ldflags "-w -s -X 'github.com/soulteary/version-kit/v2.Version=$VERSION' -X 'github.com/soulteary/version-kit/v2.Commit=$COMMIT' -X 'github.com/soulteary/version-kit/v2.BuildDate=$BUILD_DATE'" -o stargate ./src/cmd/stargate
# upx 压缩（如果失败则跳过，确保多平台构建兼容性）
RUN if command -v upx > /dev/null 2>&1; then \
      upx -9 -o stargate.minify stargate 2>/dev/null && mv stargate.minify stargate || true; \
    fi

FROM alpine:3.24
COPY --from=builder /app/stargate /bin/stargate
COPY --from=builder /app/src/internal/web/templates /app/web/templates
RUN apk add --no-cache ca-certificates && \
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
