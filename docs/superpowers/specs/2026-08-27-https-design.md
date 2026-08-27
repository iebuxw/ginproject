# 内网 HTTPS 支持设计

日期：2026-08-27

## 背景与目标

老板指出项目没有 HTTPS。当前部署为内网、无域名，nginx 容器仅监听 80（宿主机 8080）。

目标：
- 提供 HTTPS 访问：`https://localhost:8443`
- `http://localhost:8080` 自动 301 跳转到 HTTPS
- 证书采用 openssl 自签，宿主机生成后挂载进容器

## 需求确认

| 问题 | 结论 |
|------|------|
| 部署环境 | 内网 / 无域名 |
| 证书来源 | 自签证书（openssl） |
| HTTPS 端口 | 8443（与 8080 并存） |
| HTTP 8080 处理 | 自动 301 跳转到 8443 |
| 证书生成方式 | 宿主机生成一次 + docker 挂载（重建容器证书不变） |

## 架构

```
浏览器 ── http://localhost:8080 ──┐ 301 ──→ https://localhost:8443
                                  └─(nginx)─→ /api/ → go-app:8000
                                              /api/ws → go-app:8000 (wss)
```

- TLS 终止在 nginx 容器（443 端口）
- 证书文件来自宿主机 `certs/` 目录（只读挂载 `/etc/nginx/certs`）
- WebSocket 经 443 后自动升级为 wss，nginx 代理配置不变（Upgrade 头保留）

## 改动清单

### 1. `gen-cert.sh`（新增，根目录）

```bash
#!/bin/sh
# 自签证书生成脚本：已存在则跳过，不覆盖
CERT_DIR="$(dirname "$0")/certs"
mkdir -p "$CERT_DIR"
[ -f "$CERT_DIR/server.crt" ] && [ -f "$CERT_DIR/server.key" ] && { echo "证书已存在，跳过"; exit 0; }
openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
  -keyout "$CERT_DIR/server.key" -out "$CERT_DIR/server.crt" \
  -subj "/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
```

- 有效期 3650 天，SAN 含 localhost/127.0.0.1（避免 Chrome 对 IP 证书的额外报错）
- 幂等：证书存在时跳过，不覆盖既有证书

### 2. `.gitignore`：追加 `certs/`

私钥不入库。

### 3. `docker/nginx.conf`

- 原 `listen 80` server 块改为仅跳转：

```nginx
server {
    listen 80;
    server_name localhost;
    return 301 https://$host:8443$request_uri;
}
```

（`$host` 不带端口，跳转目标端口需显式写 8443）

- 新增 443 server 块，复用原 location 配置：

```nginx
server {
    listen 443 ssl http2;
    server_name localhost;

    ssl_certificate     /etc/nginx/certs/server.crt;
    ssl_certificate_key /etc/nginx/certs/server.key;

    # 原 root/index + location / + location /api/ws + location /api/ 配置原样保留
}
```

### 4. `docker-compose.yml`：nginx 服务

```yaml
ports:
  - "8080:80"
  - "8443:443"
volumes:
  - ./certs:/etc/nginx/certs:ro
```

## 错误处理

- 证书缺失：nginx 启动失败。解决：先运行 `./gen-cert.sh` 再 `docker compose up`
- 自签证书首次访问：浏览器显示不受信任警告，需"高级 → 继续访问"（内网常态）
- 证书过期（10 年后）：删除 certs/ 重跑脚本生成新证书

## 验证

1. `./gen-cert.sh` 生成证书，`docker compose up -d --build nginx`
2. `curl -k https://localhost:8443` → 200（SPA index.html）
3. `curl -I http://localhost:8080` → 301，`Location: https://localhost:8443/`
4. 浏览器打开 `https://localhost:8443`：SPA 渲染、登录、WebSocket（wss）连接、`/api` 代理均正常
