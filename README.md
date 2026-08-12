# ginadmin 后台管理系统

Go(Gin) + Vue2 (Element UI) 单体仓库。UI 全中文，注释倾向中文。

## 快速开始

```bash
docker compose up -d           # 启动所有服务
```

- 访问地址：http://localhost:8080（nginx → Vue SPA，`/api` 代理到 go-app:8000）
- 默认账号：`admin` / `admin`

## 数据库连接命令

```bash
# 方式一：容器内一键连接（推荐）
docker exec -it ginadmin-mysql mysql -uroot -padmin123 --default-character-set=utf8mb4 ginadmin

# 方式二：本机 mysql 客户端连宿主机 3307 端口
mysql -h127.0.0.1 -P3307 -uroot -padmin123 --default-character-set=utf8mb4 ginadmin

# Redis（宿主机 6380 端口）
docker exec -it ginadmin-redis redis-cli

# RabbitMQ 管理台
# http://localhost:15672   guest / guest
```

## 端口与账号速查

| 服务 | 端口 | 账号 |
|---|---|---|
| go-app | 8000 | - |
| nginx | 8080 | - |
| mysql | 容器3306 / 宿主3307 | root / admin123 |
| redis | 容器6379 / 宿主6380 | 无密码 |
| rabbitmq | 5672（管理台 15672） | guest / guest |

## 常用开发命令

```bash
# 后端
go run cmd/server/main.go              # 本地运行/开发
go build -o server ./cmd/server/       # 编译
docker compose up -d --build go-app    # Docker 重建并重启 go

# 前端（Vue 2 + Element UI）
cd web && npm run serve                # 热重载，端口 3000，proxy /api -> localhost:8000
npm run build                          # 构建到 web/dist/

# Docker（nginx 镜像会先构建 Vue 再打包）
docker compose up -d --build nginx
docker compose up -d                   # 启动全部
```
