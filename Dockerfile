# 构建阶段 - 使用官方Go镜像
FROM golang:1.24-alpine AS builder

WORKDIR /app

ENV GOPROXY=https://proxy.golang.com.cn,https://goproxy.cn,direct

COPY . .

# 构建可执行文件（启用静态编译）
RUN go clean -modcache && \
    go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/main ./demo/internal/main

# 运行阶段 - 使用Alpine基础镜像
FROM alpine:3.21.3

# 安装运行时依赖（如有需要）
RUN apk add --no-cache ca-certificates && \
    apk --no-cache add tzdata

WORKDIR /app

# 从构建阶段复制可执行文件
COPY --from=builder /app/main /app/main
COPY init.yml /app/init.yml
COPY bilibili_video.csv /app/bilibili_video.csv

# 暴露端口
EXPOSE 7887

# 启动命令
CMD ["/app/main"]