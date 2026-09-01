# Docker 多环境部署设计

## 背景

项目即将部署到阿里云 ECS，当前 Docker 配存在以下问题：
- `.env` 通过 `COPY` 打包进镜像，密钥随镜像泄露
- 所有服务端口（MySQL/Redis/RabbitMQ/ES/Kibana）暴露到宿主机
- nginx 自签证书和 `server_name localhost` 无法用于生产
- 密码硬编码在 `docker-compose.yml` 中，与 `.env` 重复维护

## 方案

使用 docker-compose 多文件机制拆分本地和生产配置，Go/Vue 代码零改动。

## 文件结构

```
docker-compose.yml              # 基础服务定义（两边共用）
docker-compose.override.yml     # 本地开发（端口映射）— 自动加载
docker-compose.prod.yml         # 生产部署（安全收紧）— 手动指定
docker/nginx.conf               # 唯一 nginx 配置（server_name _）
docker/Dockerfile               # 删掉 COPY .env
.dockerignore                   # 排除敏感文件
.env.example                    # 占位符，不含真实密码
```

## 用法

本地（行为不变）：
```bash
docker compose up -d
```

生产：
```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

## 详细设计

###1. docker-compose.yml（基础）

移除所有 `ports` 映射，密钥改用 `${...}` 引用 `.env`：

```yaml
services:
  mysql:
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_PASSWORD}
      MYSQL_DATABASE: ginadmin
      TZ: Asia/Shanghai
    # 无 ports

  redis:
    # 无 ports

  rabbitmq:
    environment:
      RABBITMQ_DEFAULT_USER: ${RABBITMQ_USER}
      RABBITMQ_DEFAULT_PASS: ${RABBITMQ_PASSWORD}
    # 无 ports

  go-app:
    volumes:
      - uploads:/app/uploads
      - go-logs:/app/logs
      - ./.env:/app/.env:ro    # 运行时挂载，不打包进镜像
    # 无 ports

  elasticsearch:
    # 无 ports

  kibana:
    # 无 ports

  nginx:
    volumes:
      - ./certs:/etc/nginx/certs:ro
      - ./docker/nginx.conf:/etc/nginx/conf.d/default.conf:ro
    # 无 ports（由 override/prod 定义）
```

###2. docker-compose.override.yml（本地）

自动加载，添加端口映射方便调试：

```yaml
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

###3. docker-compose.prod.yml（生产）

只暴露 nginx 的 80 和 443：

```yaml
services:
  nginx:
    ports:
      - "80:80"
      - "443:443"
```

###4. Dockerfile 变化

删除 `COPY .env .env`，其余不变：

```diff
  COPY --from=builder /build/server .
- COPY .env .env
  COPY migrations/ ./migrations/
```

###5. nginx.conf 变化

两处修改：

1. 删除80→443 重定向 server 块（生产环境 8443 端口不存在，重定向会失败）
2. `server_name localhost` 改为 `server_name _`

```diff
- server {
-     listen 80;
-     server_name localhost;
-     return 301 https://$host:8443$request_uri;
- }

  server {
      listen 443 ssl http2;
-     server_name localhost;
+     server_name _;
      ...
  }
```

本地开发直接访问 `https://localhost:8443`。HTTP (`http://localhost:8080`) 不再自动跳转，如需 HTTP 访问可直接用。

###6. .dockerignore（新建）

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

###7. .env.example

移除所有真实密码，改为占位符：

```
MYSQL_PASSWORD=<your-mysql-password>
JWT_SECRET=<your-jwt-secret>
REDIS_PASSWORD=<your-redis-password>
RABBITMQ_USER=<your-rabbitmq-user>
RABBITMQ_PASSWORD=<your-rabbitmq-password>
SMTP_PASSWORD=<your-smtp-password>
```

## 服务器部署步骤

```bash
git clone <repo>
cd ginproject
cp .env.example .env
vi .env                                    # 填生产密码
# 可选：替换 certs/ 下的证书
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

## 安全改进

| 问题 | 解决 |
|------|------|
| `.env` 打包进镜像 | 改为 volume 挂载，不进镜像 |
| 内部服务端口暴露 | 生产环境不映射内部端口 |
| 密码硬编码 | 统一从 `.env` 读取 |
| `.env.example` 含真实密码 | 替换为占位符 |
| `.dockerignore` 缺失 | 新建，排除敏感文件 |
