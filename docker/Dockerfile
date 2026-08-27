# ---- 构建阶段 ----
FROM golang:1.18-alpine AS builder

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

WORKDIR /build

COPY go.mod go.sum ./
RUN GOPROXY=https://goproxy.cn,direct go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY docs/ ./docs/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o server ./cmd/server/

# ---- 运行阶段 ----
FROM alpine:3

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache ca-certificates tzdata mysql-client gzip \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

WORKDIR /app
COPY --from=builder /build/server .
COPY .env .env
COPY migrations/ ./migrations/

EXPOSE 8000
CMD ["./server"]
