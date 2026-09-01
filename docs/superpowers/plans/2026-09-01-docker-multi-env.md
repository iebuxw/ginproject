# Docker 多环境部署实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 Docker 配置拆分为本地开发和生产部署两套，用 docker-compose 多文件机制实现，Go/Vue 代码零改动。

**Architecture:** 基础服务定义保留在 `docker-compose.yml`（移除端口映射和硬编码密码），本地端口映射放 `docker-compose.override.yml`（自动加载），生产安全配置放 `docker-compose.prod.yml`（手动指定）。Dockerfile 不再打包 `.env`，改为运行时 volume 挂载。

**Tech Stack:** Docker Compose multi-file override, nginx, self-signed SSL

---

### Task1: 创建 .dockerignore

**Files:**
- Create: `.dockerignore`

- [ ] **Step1: 创建 .dockerignore**

在项目根目录创建 `.dockerignore`，排除不需要进入构建上下文的文件：

```
.env
.env.*
.git
node_modules
web/node_modules
certs
docs
*.md
.dockerignore
docker-compose*.yml
```

- [ ] **Step2: 提交**

```bash
git add .dockerignore
git commit -m "chore: add .dockerignore to exclude sensitive files from build context"
```

---

### Task2: 修改 Dockerfile，移除 .env 打包

**Files:**
- Modify: `docker/Dockerfile:25`

- [ ] **Step1: 删除 COPY .env 行**

在 `docker/Dockerfile` 中删除第25行 `COPY .env .env`：

```diff
  COPY --from=builder /build/server .
- COPY .env .env
  COPY migrations/ ./migrations/
```

- [ ] **Step2: 验证构建**

```bash
docker compose build go-app
```

Expected: 构建成功（.env 将通过 volume 挂载，不再打包进镜像）

- [ ] **Step3: 提交**

```bash
git add docker/Dockerfile
git commit -m "fix: remove COPY .env from Dockerfile to prevent secrets leaking into image"
```

---

### Task3: 修改 nginx.conf

**Files:**
- Modify: `docker/nginx.conf`

- [ ] **Step1: 删除重定向 server 块，修改 server_name**

将 `docker/nginx.conf` 从：

```nginx
server {
    listen 80;
    server_name localhost;
    return 301 https://$host:8443$request_uri;
}

server {
    listen 443 ssl http2;
    server_name localhost;

    root /usr/share/nginx/html;
    index index.html;

    ssl_certificate     /etc/nginx/certs/server.crt;
    ssl_certificate_key /etc/nginx/certs/server.key;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api/ws {
        proxy_pass http://go-app:8000/api/ws;
        proxy_set_header Host $host;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 86400s;
        proxy_send_timeout 86400s;
    }

    location /api/ {
        proxy_pass http://go-app:8000/api/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        client_max_body_size 101m;
    }
}
```

改为：

```nginx
server {
    listen 443 ssl http2;
    server_name _;

    root /usr/share/nginx/html;
    index index.html;

    ssl_certificate     /etc/nginx/certs/server.crt;
    ssl_certificate_key /etc/nginx/certs/server.key;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api/ws {
        proxy_pass http://go-app:8000/api/ws;
        proxy_set_header Host $host;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 86400s;
        proxy_send_timeout 86400s;
    }

    location /api/ {
        proxy_pass http://go-app:8000/api/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        client_max_body_size 101m;
    }
}
```

变更点：
1. 删除了第一个 `server` 块（80→443 重定向，生产环境 8443 端口不存在会失败）
2. `server_name localhost` 改为 `server_name _`（匹配任意域名/IP）

- [ ] **Step2: 提交**

```bash
git add docker/nginx.conf
git commit -m "fix: remove80→443 redirect and use wildcard server_name for multi-env support"
```

---

### Task4: 修改 docker-compose.yml

**Files:**
- Modify: `docker-compose.yml`

- [ ] **Step1: 重写 docker-compose.yml**

将 `docker-compose.yml` 改为以下内容（移除所有 ports 映射，密钥改 `${...}` 引用，go-app 加 .env 挂载）：

```yaml
version: '3.8'

services:
  mysql:
    image: mysql:5.7
    container_name: ginadmin-mysql
    restart: always
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_PASSWORD}
      MYSQL_DATABASE: ginadmin
      TZ: Asia/Shanghai
    volumes:
      - mysql-data:/var/lib/mysql
    command: --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:3.2-alpine
    container_name: ginadmin-redis
    restart: always
    volumes:
      - redis-data:/data
      - ./docker/redis.conf:/usr/local/etc/redis/redis.conf
    command: redis-server /usr/local/etc/redis/redis.conf
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  rabbitmq:
    image: rabbitmq:3-management
    container_name: ginadmin-rabbitmq
    restart: always
    environment:
      RABBITMQ_DEFAULT_USER: ${RABBITMQ_USER}
      RABBITMQ_DEFAULT_PASS: ${RABBITMQ_PASSWORD}
    healthcheck:
      test: ["CMD", "rabbitmq-diagnostics", "-q", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  go-app:
    build:
      context: .
      dockerfile: docker/Dockerfile
    container_name: ginadmin-api
    restart: always
    volumes:
      - uploads:/app/uploads
      - go-logs:/app/logs
      - ./.env:/app/.env:ro
    depends_on:
      mysql:
        condition: service_healthy
      redis:
        condition: service_healthy
      rabbitmq:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8000/api/health"]
      interval: 15s
      timeout: 5s
      retries: 3
      start_period: 10s

  elasticsearch:
    build:
      context: .
      dockerfile: docker/elasticsearch.Dockerfile
    container_name: ginadmin-elasticsearch
    restart: always
    environment:
      - discovery.type=single-node
      - ES_JAVA_OPTS=-Xms512m -Xmx512m
      - xpack.security.enabled=false
    volumes:
      - es-data:/usr/share/elasticsearch/data
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9200/_cluster/health?wait_for_status=yellow"]
      interval: 15s
      timeout: 10s
      retries: 10

  kibana:
    image: kibana:7.17.15
    container_name: ginadmin-kibana
    restart: always
    environment:
      - ELASTICSEARCH_HOSTS=http://elasticsearch:9200
    depends_on:
      elasticsearch:
        condition: service_healthy

  nginx:
    build:
      context: .
      dockerfile: docker/Dockerfile.nginx
    container_name: ginadmin-nginx
    restart: always
    volumes:
      - ./certs:/etc/nginx/certs:ro
      - ./docker/nginx.conf:/etc/nginx/conf.d/default.conf:ro
    depends_on:
      go-app:
        condition: service_healthy

volumes:
  mysql-data:
  redis-data:
  es-data:
  uploads:
  go-logs:
```

与原文件的差异：
1. 移除所有 `ports` 映射（mysql/redis/rabbitmq/go-app/elasticsearch/kibana/nginx）
2. `MYSQL_ROOT_PASSWORD` 从 `admin123` 改为 `${MYSQL_PASSWORD}`
3. `RABBITMQ_DEFAULT_USER/PASS` 从 `guest/guest` 改为 `${RABBITMQ_USER}/${RABBITMQ_PASSWORD}`
4. go-app 新增 `volumes: ./.env:/app/.env:ro`
5. nginx 新增 `volumes: ./docker/nginx.conf:/etc/nginx/conf.d/default.conf:ro`

- [ ] **Step2: 验证语法**

```bash
docker compose config
```

Expected: 输出完整的合并配置，无报错

- [ ] **Step3: 提交**

```bash
git add docker-compose.yml
git commit -m "refactor: remove hardcoded ports and secrets from docker-compose.yml"
```

---

### Task5: 创建 docker-compose.override.yml

**Files:**
- Create: `docker-compose.override.yml`

- [ ] **Step1: 创建本地开发 override 文件**

在项目根目录创建 `docker-compose.override.yml`：

```yaml
# 本地开发配置，docker compose up 自动加载
services:
  mysql:
    ports:
      - "3307:3306"

  redis:
    ports:
      - "6380:6379"

  rabbitmq:
    ports:
      - "15672:15672"
      - "5672:5672"

  go-app:
    ports:
      - "8000:8000"

  elasticsearch:
    ports:
      - "9200:9200"

  kibana:
    ports:
      - "5601:5601"

  nginx:
    ports:
      - "8080:80"
      - "8443:443"
```

- [ ] **Step2: 验证合并配置**

```bash
docker compose config
```

Expected: 输出包含所有端口映射（从 override 合并），无报错

- [ ] **Step3: 提交**

```bash
git add docker-compose.override.yml
git commit -m "feat: add docker-compose.override.yml for local development ports"
```

---

### Task6: 创建 docker-compose.prod.yml

**Files:**
- Create: `docker-compose.prod.yml`

- [ ] **Step1: 创建生产部署配置文件**

在项目根目录创建 `docker-compose.prod.yml`：

```yaml
# 生产部署配置
# 用法: docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
services:
  nginx:
    ports:
      - "80:80"
      - "443:443"
```

- [ ] **Step2: 验证生产配置**

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml config
```

Expected: 输出只包含 nginx 的 80/443 端口映射，无其他端口暴露

- [ ] **Step3: 提交**

```bash
git add docker-compose.prod.yml
git commit -m "feat: add docker-compose.prod.yml for production deployment"
```

---

### Task7: 更新 .env.example

**Files:**
- Modify: `.env.example`

- [ ] **Step1: 确认 .env.example 无真实密码**

当前 `.env.example` 已使用中文占位符（如"请修改为你的密码"），无需修改内容。确认文件中不包含任何真实密码即可。

- [ ] **Step2: 提交（如有变更）**

如果文件无需变更，则跳过提交。

---

### Task8: 端到端验证

- [ ] **Step1: 停止现有容器**

```bash
docker compose down
```

- [ ] **Step2: 本地启动验证**

```bash
docker compose up -d --build
```

Expected: 所有服务启动成功

- [ ] **Step3: 检查服务状态**

```bash
docker compose ps
```

Expected: 所有服务状态为 healthy 或 running

- [ ] **Step4: 验证 HTTPS 访问**

浏览器访问 `https://localhost:8443`，确认页面正常加载（接受自签证书警告）

- [ ] **Step5: 验证 API 访问**

```bash
curl -k https://localhost:8443/api/health
```

Expected: 返回健康检查响应

- [ ] **Step6: 验证 .env 未打包进镜像**

```bash
docker run --rm ginproject-go-app cat /app/.env 2>&1 || echo "PASS: .env not in image"
```

Expected: 报错文件不存在（.env 通过 volume 挂载，不在镜像中）

- [ ] **Step7: 验证生产配置语法**

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml config
```

Expected: 输出只包含 nginx 80/443 端口，无内部服务端口
