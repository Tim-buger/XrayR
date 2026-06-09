# Build go
FROM golang:1.25.3-alpine AS builder
WORKDIR /app
ENV CGO_ENABLED=0
ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}

# 先复制依赖清单，源码变化时可复用依赖缓存
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -v -o /out/XrayR -trimpath -ldflags "-s -w -buildid="

# 运行阶段：只保留二进制和 TLS/时区所需文件
FROM alpine:3.21
RUN apk --no-cache add tzdata ca-certificates \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && mkdir -p /etc/XrayR/cert
WORKDIR /etc/XrayR
COPY --from=builder /out/XrayR /usr/local/bin/XrayR

STOPSIGNAL SIGTERM
ENTRYPOINT ["/usr/local/bin/XrayR", "--config", "/etc/XrayR/config.yml"]
