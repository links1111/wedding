# syntax=docker/dockerfile:1

# ====== 构建阶段 ======
FROM golang:1.23-alpine AS builder

# CGO 禁用：go-sqlite3 需要 CGO，因此保留 CGO_ENABLED=1
# 安装 gcc/musl-dev 以编译 go-sqlite3
RUN apk add --no-cache gcc musl-dev

WORKDIR /build

# 先复制依赖文件，利用缓存层
COPY go.mod go.sum* ./
RUN go mod download

# 复制源码
COPY . .

# 静态编译（CGO 启用以支持 go-sqlite3，链接 musl）
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w -buildid=" -o wedding-server .

# ====== 运行阶段 ======
FROM alpine:3.20

# 安装 ca-certificates（HTTPS 请求）与 tzdata（时区）
# musl 运行时已随二进制静态链接，无需额外 gcc 运行时
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S -G app app

WORKDIR /app

# 复制编译产物
COPY --from=builder /build/wedding-server /app/wedding-server

# 复制默认静态资源和模板（可被 volume 挂载覆盖）
COPY --from=builder /build/web/static /app/web/static
COPY --from=builder /build/web/templates /app/web/templates

# 数据目录（SQLite 持久化），挂载卷到此处即可持久保存
RUN mkdir -p /app/data && chown -R app:app /app
VOLUME ["/app/data"]

# 切换非 root 用户
USER app

# 环境变量默认值（可在 docker run / compose 中覆盖）
ENV PORT=8080 \
    DB_PATH=/app/data/wedding.db \
STATIC_DIR=/app/web/static \
TEMPLATE_DIR=/app/web/templates \
    TZ=Asia/Shanghai

EXPOSE 8080

# 健康检查：访问根路由
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:${PORT}/ >/dev/null 2>&1 || exit 1

ENTRYPOINT ["/app/wedding-server"]
