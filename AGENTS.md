# AGENTS.md

Go(Gin) + Vue2 后端管理的单体仓库。UI 文案全部中文，代码注释倾向中文。

## 命令

```bash
# 后端
go run cmd/server/main.go              # 本地运行/开发
go build -o server ./cmd/server/       # 编译
docker compose up -d --build go-app    # Docker 重建并重启 go

# 前端 (Vue 2 + Element UI，不是 Vue 3)
cd web && npm run serve                # 热重载，端口 3000，proxy /api -> localhost:8000
npm run build                          # 构建到 web/dist/

# Docker（nginx 镜像会先构建 Vue 再打包）
docker compose up -d --build nginx
docker compose up -d                   # 启动全部
```

- 访问地址：http://localhost:8080（nginx → Vue SPA，`/api` 代理到 go-app:8000）
- 固定端口：go-app **8000**，nginx **8080**，容器内 mysql **3306**/宿主机 **3307**，redis 容器内 **6379**/宿主机 **6380**，rabbitmq **5672**（管理台 15672）
- 默认账号：`admin` / `admin`
- **无测试、无 lint/typecheck 配置**，也没有 CI。DDL 由 `AutoMigrate` 自动建表，无迁移文件。

## 本地运行须知（重要）

`.env` 被 gitignore、但当前已存在，且**内部填的是 docker 服务名**（`mysql`/`redis`/`rabbitmq`）。

- 只在 docker 里跑 go-app 时可直接用现成 `.env`。
- **本机 `go run` 需要把 host 指向 `127.0.0.1`，port 指向 docker 发布端口**：`MYSQL_PORT=3307`、`REDIS_PORT=6380`、`RABBITMQ_PORT=5672`、`RABBITMQ_HOST=127.0.0.1`。
- `.env.example` **缺少 RabbitMQ 变量**，但 `internal/config` 会读 `RABBITMQ_HOST/PORT/USER/PASSWORD`，新增/复制环境配置时易漏。

## 架构

```
cmd/server/main.go         # 入口：手动组装依赖链（无 DI 框架），AutoMigrate + 种子数据
internal/
  config/                  # Viper 读 .env
  router/                  # Gin 路由注册 + 中间件链
  middleware/              # CORS → JWT → RequirePerm → RBAC → OperationLogger
  controller/             # 参数绑定 + 调 service
  service/                 # 业务逻辑
  dao/                     # GORM 数据访问（无接口，直接具体类型）
  model/                   # User/Role/Menu/OperationLog/LoginLog + DateTime
  worker/                  # 导出后台 worker（消费 RabbitMQ 生成 Excel）
  ws/                      # WebSocket Hub
  utils/                   # response / jwt / hash / uuid
```

分层调用惯例（无接口抽象，直接依赖具体类型）：
router → middleware → controller → service → dao → model。新增功能照此链在 `main.go` 逐层 new 注入。

## 权限模型（RBAC）

```
User ──N:M── Role ──N:M── Menu（Menu.Permission 字段，如 "user:add"）
```
- `Menu.Type`：1=目录，2=菜单页，3=按钮/权限点
- 用户权限 = 所有角色绑定的 Menu.Permission 并集
- 前端路由由后端菜单树动态生成（`web/src/permission.js` 的 `generateRoutes`）
- 每个 API 路由硬编码权限，如 `middleware.RequirePerm("user:add")`
- 用户管理 CRUD **不支持分配角色**

## 关键约定

- `utils.Error` 返回 HTTP **200**（仅业务 code 非 200）；要改 HTTP 状态码用 `ErrorWithStatus`
- JWT：HS256，24h（`.env` 可配），Redis 黑名单 `blacklist:<token>`
- `DateTime` 类型**不触发 GORM 自动时间戳**，需手动 `CreatedAt = model.DateTime(time.Now())`
- `OperationLog.Module/Action` 中间件**未填充，恒为空**（业务需要时得自己补）
- Redis 是 `redis:3.2-alpine`，**不支持 HSET 一次多字段**（4.0+ 才行），代码用单字段 `HSet` 调用，新增代码保持此风格
- 手动用 mysql CLI 插入中文需加 `--default-character-set=utf8mb4`，否则乱码
- WebSocket `/api/ws` 是**公开路由**（不走 JWT）；nginx 需转发 Upgrade 头（已在 `docker/nginx.conf` 配置）

## 异步导出流程（不易从命名推断）

日志导出走「RabbitMQ + WebSocket + 流式 Excel」，三思 SQL/缓存时勿忘 redis：

1. `POST /api/logs/export`：写 Redis task `excel:task:<id>`（pending、user_id、method）= uuid，发布到 RabbitMQ 队列 `excel.export`
2. `worker.Start()`（main.go 里 goroutine）消费 → 用 excelize `StreamWriter` 流式写 `exports/<taskID>.xlsx`（`exports/` 已 gitignore）→ 回写 task 状态
3. 通过 WebSocket 通知客户端 `export_complete` / `export_failed`
4. 前端轮询 `GET /api/logs/export-status?task_id=`，完成后 `GET /api/logs/download/:taskID` 下载（下载后服务端删除该 xlsx）

改 Excel 写逻辑务必用 `StreamWriter.SetRow`，不要和 `SetCellValue` 混用（混用会导致表头丢失）。