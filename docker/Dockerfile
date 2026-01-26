FROM golang:1.25-alpine3.22 AS Builder
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

# 如果没有提供 BUILD_DATE，则使用当前时间
# 使用 TARGETOS 和 TARGETARCH 支持多平台构建（由 Docker Buildx 自动提供）
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN BUILD_DATE=${BUILD_DATE:-$(date +%FT%T%z)} && \
    GO111MODULE=on GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 go build -ldflags "-w -s -X 'github.com/soulteary/version-kit.Version=$VERSION' -X 'github.com/soulteary/version-kit.Commit=$COMMIT' -X 'github.com/soulteary/version-kit.BuildDate=$BUILD_DATE'" -o stargate ./src/cmd/stargate
# upx 压缩（如果失败则跳过，确保多平台构建兼容性）
RUN if command -v upx > /dev/null 2>&1; then \
      upx -9 -o stargate.minify stargate 2>/dev/null && mv stargate.minify stargate || true; \
    fi

FROM alpine:3.22
COPY --from=Builder /app/stargate /bin/stargate
COPY --from=Builder /app/src/internal/web/templates /app/web/templates
RUN apk add --no-cache curl
WORKDIR /app
EXPOSE 80
CMD ["stargate"]