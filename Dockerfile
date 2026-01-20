FROM golang:1.25-alpine AS builder
ARG VERSION=dev
WORKDIR /build
COPY ./go.mod ./
COPY ./go.sum ./
ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn
RUN go mod download
COPY ./src ./src
COPY ./pkg ./pkg
RUN CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/soulteary/stargate/src/cmd/stargate.Version=${VERSION}" -o ./main ./src/cmd/stargate

FROM scratch
WORKDIR /app
COPY --from=builder /build/main ./main
COPY --from=builder /build/src/internal/web/templates /app/web/templates
EXPOSE 80
ENTRYPOINT ["./main"]
