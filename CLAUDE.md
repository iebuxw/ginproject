# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Backend (Go)
```bash
# 本地运行（依赖本地 .env 中的 MYSQL_HOST/REDIS_HOST 指向 docker 服务）
go run cmd/server/main.go

# 编译
go build -o server ./cmd/server/

# Swagger 文档（修改注释后必须重新生成，访问 http://localhost:8000/swagger/index.html）
swag init -g cmd/server/main.go

# Docker 中重新构建并重启 Go 服务
docker compose up -d --build go-app
```

### Frontend (Vue)
```bash
cd web

# 开发热重载（端口 3000，proxy /api → localhost:8000）
npm run serve

# 构建到 dist/
npm run build

# Docker 中重新构建并重启 nginx（含 Vue 构建）
cd .. && docker compose up -d --build nginx
```

### Docker（整体管理）
```bash
docker compose up -d          # 启动所有服务
docker compose up -d --build  # 重新构建所有镜像
docker compose restart redis  # 重启单个服务
```

**访问地址：** http://localhost:8080（nginx → Vue SPA + /api 代理到 go-app:8000）

**默认管理员：** `admin` / `admin`（迁移种子 000003 写入）；但 DB 中实际密码已被改为 `123456`，登录受阻先用 `123456`，**未经用户明确同意不得擅自重置密码/用户数据**。

### 本地运行须知

`.env` 被 gitignore，仓库内的 `.env` 填的是 docker 服务名（`mysql`/`redis`/`rabbitmq`/`elasticsearch`）。本机 `go run` 需覆盖：`MYSQL_PORT=3307`、`REDIS_PORT=6380`、`RABBITMQ_HOST=127.0.0.1`、`RABBITMQ_PORT=5672`、`ES_HOST=127.0.0.1`。

### 测试与 Lint

**测试极少：仅 `internal/middleware/logger_test.go`（操作日志密码脱敏），用 `go test ./...` 运行；无 lint/typecheck、无 CI。** DDL 和种子数据由 golang-migrate 管理（`migrations/` 目录，具体文件以目录现状为准），启动时自动执行。新增迁移按 `00000N_xxx` 递增创建成对 .up/.down 文件，建表用 `IF NOT EXISTS`、种子用 `INSERT IGNORE` + 显式 id 保证幂等。

## 架构

```
cmd/server/main.go          # 入口：手动组装依赖链，启动时执行 golang-migrate 迁移
internal/
  config/                    # Viper 读取 .env
  router/                    # Gin 路由注册，中间件链
  middleware/                # CORS → JWT → RequirePerm → RBAC → OperationLogger
  controller/                # 请求处理，参数绑定，调用 service
  service/                   # 业务逻辑（密码哈希、菜单树、token 黑名单）
  dao/                       # GORM 数据访问
  model/                     # 结构体：User、Role、Menu、OperationLog、LoginLog、DictType、DictData、CronTask、CronTaskExecution、DateTime
  scheduler/                 # 定时任务调度器（robfig/cron/v3）+ 预定义命令注册表
```

实际还有 `es/`（Elasticsearch 客户端 + LogRepo，操作日志全文检索）、`worker/`（导出/邮件后台 worker，消费 RabbitMQ）、`ws/`（WebSocket Hub）、`utils/`（response/jwt/hash/uuid）。

**分层：** router → middleware → controller → service → dao → model（无接口抽象，直接依赖具体类型）

## 关键设计

### 权限模型（RBAC）

```
User ──N:M── Role ──N:M── Menu
                              │
                    Permission 字段（如 "user:add"）
```

- `Menu.Type`：1=目录，2=菜单页，3=按钮/权限点
- 用户权限 = 其所有角色绑定的所有 Menu.Permission 的并集
- 前端路由由后端返回的菜单树动态生成（`permission.js` 中 `generateRoutes`）
- 每个 API 路由硬编码所需权限字符串（如 `RequirePerm("user:add")`）

### JWT 认证

- HS256 签名，过期时间 24h（可配）
- Redis 存黑名单 `blacklist:<token>`（登出时写入，TTL = 令牌剩余有效期）
- JWTAuth 中间件启动时检查黑名单

### OperationLogger 中间件

- 仅记录 POST/PUT/DELETE，跳过 GET
- 捕获请求 body 作为 params，无 body 时回退用请求路径
- params 落库前经 `maskSensitiveParams` 脱敏（password 类字段值替换为 `***`，JSON 与非 JSON 均处理）
- 创建时间用自定义 `DateTime` 类型，JSON 格式 `2006-01-02 15:04:05`

### 数据字典

- `dict_type`（类型）+ `dict_data`（字典数据）两张表，DAO 为 `DictTypeDAO`/`DictDataDAO`，Service 为 `DictTypeService`/`DictDataService`（前者持有 dictDataDAO）
- 前端页面 `web/src/views/dict/`，路由 `dict:list` 等权限点

### 登录异常邮件告警

- 登录失败触发告警：发布 RabbitMQ 消息 + Redis 限频，`worker.MailWorker` 消费并发送（SMTP 支持 465 隐式 SSL）

### 操作日志 ES 双写（学习功能）

- 中间件写 MySQL 后同步写 ES（`es.LogRepo.Index`，`_id`=MySQL 主键，`Refresh:"true"` 立即可见）；ES 不可用仅 `log.Printf` 告警，不阻断请求
- `GET /logs` 优先走 ES（bool / multi_match / range / highlight，IK 中文分词），ES 失败自动回退 MySQL；响应带 `data_source` 标记（`es`/`mysql`）
- ES 初始化失败不 fatal：`main.go` 置 `esClient=nil`，`logRepo` 始终非 nil，`Enabled()` 返回 false 即整体降级
- **版本强绑定**：IK 插件必须与 ES 严格同版本（当前均 7.17.15），镜像在 `docker/elasticsearch.Dockerfile` 从 `get.infini.cloud/elasticsearch/analysis-ik` 下载
- `es.NewLogRepo(cli *es.Client)` 接受包装的 `*es.Client`（可传 nil），内部经 `cli.RawClient()` 拿原生 client

### 异步导出流程（命名看不出来）

1. `POST /api/logs/export`：写 Redis task `excel:task:<id>`（pending、user_id、method）+ 发布到 RabbitMQ 队列 `excel.export`
2. `worker.Start()`（main.go 里 goroutine）消费 -> 用 excelize `StreamWriter` 流式写 `exports/<taskID>.xlsx`（`exports/` 已 gitignore）-> 回写 task 状态
3. WebSocket 推 `export_complete` / `export_failed`
4. 前端轮询 `GET /api/logs/export-status?task_id=`，完成后 `GET /api/logs/download/:taskID` 下载（下载后服务端删除该 xlsx）

**Excel 写逻辑务必用 `StreamWriter.SetRow`，不要和 `SetCellValue` 混用**（混用会导致表头丢失）。

### 数据库备份与恢复

- `db_backups` 表存储备份记录，`backups/` 目录存放 `.sql.gz` 文件（已 gitignore）
- 备份通过 `os/exec` 调用 `mysqldump | gzip`，恢复用 `gunzip -c | mysql`
- Docker 镜像需安装 `mysql-client` + `gzip`（见 `docker/Dockerfile`）
- `backup_db` 和 `clean_backup` 为预定义命令，注册在 `scheduler/commands.go`，在 `main.go` 注入真实实现
- 前端 `web/src/views/backup/index.vue`，路由权限 `db_backup:*`
- 恢复操作需在对话框中输入"确认恢复"才能点击确认按钮

### 定时任务（CronTask）

- `cron_tasks`（任务）+ `cron_task_executions`（执行日志）两张表；Cron 表达式为 **6 段（秒 分 时 日 月 周）**，非标准 5 段
- 任务两种模式：`command` 非空走**预定义命令**（注册表 `scheduler/commands.go`，提供 name/label/handler，进程内直接调用，任务的 url/headers/body 被忽略）；为空走自定义 HTTP。前端编辑对话框命令下拉中 `_custom` 即自定义模式，命令列表来自 `GET /api/cron-tasks/commands`
- 任务增删改/启停后由 Service 调 `Reload()` 全量重建（热更新）；`running` sync.Map 防重叠，上次未执行完记跳过；页面「立即执行」走 `RunNow`，trigger=manual
- 执行状态：0=成功，1=失败，2=跳过；响应体截断至 2000 字符
- 前端 `web/src/views/task/`（index.vue 任务管理 + logs.vue 执行日志），权限点 `cron:list/query/add/edit/delete/run/log`

### WebSocket

`/api/ws` 是**公开路由**（不走 JWT），由 `ws.Hub` 维护连接并按 `user_id` 推送。nginx 需转发 `Upgrade` 头（已在 `docker/nginx.conf` 配置）。

`POST /api/logs/cleanup` 同为**公开路由**：请求头 `X-Cleanup-Secret`（优先）或 query `secret` 与 `.env` 的 `LOG_CLEANUP_SECRET` 比对，通过后分批清理过期日志（days 缺省 30，校验失败返回业务码 400 而非 HTTP 状态码）。

## 注意事项

- `utils.Error` 返回 HTTP 200（业务错误码），需要改 HTTP 状态码用 `ErrorWithStatus`
- `OperationLog.Module` / `Action` 字段中间件未填充，当前始终为空
- 用户管理 CRUD 不支持分配角色
- `.env`、`web/dist/`、`exports/`、`backups/` 均 gitignore，Docker 在构建阶段自行编译前端
- Redis 是 `redis:3.2-alpine`，**不支持 HSET 多字段**（4.0+ 才支持），多字段需拆成单字段调用
- 前端是 **Vue 2 + Element UI**（不是 Vue 3）；`web/src/store/modules/permission.js` 用后端菜单树动态生成路由，**新增菜单必须在 `componentMap` 中添加路由映射**
- RabbitMQ 是 `rabbitmq:3-management`，通过 `amqp091-go` 连接
- WebSocket 走 `gorilla/websocket`，nginx 需配置 `proxy_set_header Upgrade $http_upgrade` 转发 WebSocket 升级头
- `DateTime` 类型不会触发 GORM 自动时间戳，需手动设置 `CreatedAt`
- 手动操作 MySQL 插入中文时需加 `--default-character-set=utf8mb4`，否则乱码
- 提交习惯：按功能模块分批提交，不同功能不混在一个 commit（如"修复字典操作列"和"新增用户描述字段"分开提交）
- UI 文案全部中文

## 工作方式

- 需求有歧义、风险高或影响大时，先澄清并等待批准再写代码（Spec Coding，不做 Vibe Coding）
- 实现前先说明方法；Plan 只写方案不写代码
- 复杂任务拆分为低耦合、可独立验证的子任务，分步推进
- 同一故障尝试修正超 5 次仍未解决时停止，汇报现状等待反馈
- 清理临时代码：交付前必须自查并移除所有调试日志（如 console.log / fmt.Println等）及临时注释
- 每次交付的改动需保持 Git 提交粒度清晰，避免将多个独立的子任务混在一个 Commit 中
- 改动代码时，若存在相关文档或函数注释（如 JSDoc / GoDoc / PHPDoc 等），必须同步更新，保持代码与注释一致；若原代码无注释则无需专门补写

### 小改动纪律（防范围膨胀）

- 复杂或涉及多文件的任务：开工前列出改动文件清单和改动点，确认后再动手
- 单文件/微小 Bug 修复：可直接按最小改动原则修改，完成后供用户审查
- 最小改动：只改任务必需的文件与行，不顺手重构、不修无关 bug、不改无关代码
- 不新增依赖：引入第三方库或新依赖前必须说明理由并征得同意
- 不新增过度抽象：默认沿用现有分层与结构。不新建公共接口、通用工具或配置项；仅在同一个文件内部允许提取极简的私有辅助函数（Helper），绝不随意扩展全局抽象
- 完成后用 git diff 自查，撤销超出任务范围的行

### 历史代码保护（防回归）

- 改动前先完整阅读目标函数及其调用链，不基于片段猜测
- 修改公共代码（中间件、utils、model、DAO/service 签名）前，grep 全局所有调用方确认不破坏
- 不改变对外契约：路由、请求/响应 JSON 字段、权限点、菜单树结构保持兼容
- 不修改已应用的迁移文件（migrations/），只能新增成对迁移
- 不删除看似无用的历史代码/字段/路由，删除前先询问
- 自动化验证：改动后必须执行编译/构建检查（如 go build / npm run build），并运行受影响的单元测试/集成测试。并对受影响接口手工回归。无法手工回归测试的，生成手工回归测试建议清单，交由用户验证

### 复用优先（防重复造轮子）

- 写任何工具/函数前，先搜索：标准库 → 项目已有依赖（go.mod / web/package.json）→ 项目内已有实现（internal/、utils/）是否覆盖该功能
- 已有依赖能力范围内直接用库 API（JWT/Excel/WebSocket/Redis/MQ/GORM 均已引入），不手写
- 项目已有现成实现（utils/hash、utils/jwt、utils/uuid、utils/response 等）直接复用，不另起炉灶
- 标准库和已有依赖都没有时，优先引入成熟三方库（选主流、维护活跃），不自己造轮子；引入前说明理由并经同意
- 只有标准库、已有依赖、成熟三方库都不合适时，才允许自研实现
