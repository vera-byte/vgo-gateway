# 构建阶段
FROM golang:1.24.1-alpine AS builder

# 安装必要的工具
RUN apk add --no-cache git ca-certificates tzdata

# 设置工作目录
WORKDIR /app

# 复制工作区文件
COPY go.work go.work.sum ./

# 复制依赖的模块
COPY vgo-kit/ ./vgo-kit/
COPY vgo-gateway/ ./vgo-gateway/

# 设置工作目录到vgo-gateway
WORKDIR /app/vgo-gateway

# 下载依赖
RUN go mod download

# 构建应用
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X github.com/vera-byte/vgo-gateway/internal/version.Version=${VERSION} \
    -X github.com/vera-byte/vgo-gateway/internal/version.Commit=${COMMIT} \
    -X github.com/vera-byte/vgo-gateway/internal/version.BuildTime=${BUILD_TIME}" \
    -o vgo-gateway ./cmd

# 运行阶段
FROM alpine:latest

# 安装必要的工具
RUN apk --no-cache add ca-certificates tzdata curl

# 设置时区
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
RUN echo 'Asia/Shanghai' > /etc/timezone

# 创建非root用户
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/vgo-gateway/vgo-gateway .

# 创建配置目录
RUN mkdir -p /app/configs

# 更改文件所有者
RUN chown -R appuser:appgroup /app

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 8080 9090

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:9090/health || exit 1

# 启动应用
CMD ["./vgo-gateway"]