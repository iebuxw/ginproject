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

**访问地址：** nginx 强制把 http://localhost:8080 301 到 **https://localhost:8443**（自签证书，见 docker/nginx.conf；Vue SPA + /api 代理到 go-app:8000）

**默认管理员：** `admin` / `admin`（迁移种子 000003 写入）；但 DB 中实际密码已被改为 `123456`，登录受阻先用 `123456`，**未经用户明确同意不得擅自重置密码/用户数据**。

** DDL 和种子数据由 golang-migrate 管理（`migrations/` 目录，具体文件以目录现状为准），启动时自动执行。新增迁移按 `00000N_xxx` 递增创建成对 .up/.down 文件，建表用 `IF NOT EXISTS`、种子用 `INSERT IGNORE` + 显式 id 保证幂等。

### 本地运行须知

`.env` 被 gitignore，仓库内的 `.env` 填的是 docker 服务名（`mysql`/`redis`/`rabbitmq`/`elasticsearch`）。本机 `go run` 需覆盖：`MYSQL_PORT=3307`、`REDIS_PORT=6380`、`RABBITMQ_HOST=127.0.0.1`、`RABBITMQ_PORT=5672`、`ES_HOST=127.0.0.1`。

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
  model/                     # 结构体：User、Role、Menu等
  scheduler/                 # 定时任务调度器（robfig/cron/v3）+ 预定义命令注册表
```

实际还有 `es/`（Elasticsearch 客户端 + LogRepo，操作日志全文检索）、`worker/`（导出/邮件后台 worker，消费 RabbitMQ）、`ws/`（WebSocket Hub）、`utils/`（response/jwt/hash/uuid）。

**分层：** router → middleware → controller → service → dao → model（无接口抽象，直接依赖具体类型）

## 版本与部署

- **Go** 1.18 / **Gin** v1.9.1 / **GORM** v1.25.5
- **Vue** 2.7.16 / **Element UI** 2.15.14
- **MySQL** 5.7 / **Redis** 3.2-alpine / **RabbitMQ** 3-management / **Elasticsearch** 7.17.15（IK 同版本）
- **部署**：Docker Compose 编排，Go 多阶段构建（golang:1.18-alpine → alpine:3），nginx 自签证书反代（http:80→https:8443），API 代理到 go-app:8000

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

**Excel 写逻辑务必用 `StreamWriter.SetRow`，不要和 `SetCellValue` 混用**（混用会导致表头丢失）。

### 数据库备份与恢复

- **mysqldump 必须排除 `db_backups` 表**（`--ignore-table`），否则恢复时会丢失备份记录

## 注意事项
- Redis 是 `redis:3.2-alpine`，**不支持 HSET 多字段**（4.0+ 才支持），多字段需拆成单字段调用
- 前端是 **Vue 2 + Element UI**（不是 Vue 3）；`web/src/store/modules/permission.js` 用后端菜单树动态生成路由，**新增菜单必须在 `componentMap` 中添加路由映射**
- `DateTime` 类型不会触发 GORM 自动时间戳，需手动设置 `CreatedAt`
- 手动操作 MySQL 插入中文时需加 `--default-character-set=utf8mb4`，否则乱码
- 提交习惯：按功能模块分批提交，不同功能不混在一个 commit（如"修复字典操作列"和"新增用户描述字段"分开提交）

## 工作方式

- 需求有歧义、风险高或影响大时，先澄清并等待批准再写代码（Spec Coding，不做 Vibe Coding）
- 实现前先说明方法；Plan 只写方案不写代码
- 复杂任务拆分为低耦合、可独立验证的子任务，分步推进
- 同一故障尝试修正超 5 次仍未解决时停止，汇报现状等待反馈
- 清理临时代码：交付前必须自查并移除所有调试日志（如 console.log / fmt.Println等）及临时注释
- 写代码时对必要的、重要的位置补充注释（字段含义、非显而易见的业务逻辑与设计决策等）；注释保持简洁，不写无意义的逐行注释
- 每次交付的改动需保持 Git 提交粒度清晰，避免将多个独立的子任务混在一个 Commit 中
- 改动代码时，若存在相关文档或函数注释（如 JSDoc / GoDoc / PHPDoc 等），必须同步更新，保持代码与注释一致；若原代码无注释则无需专门补写
- Bash `$()` 嵌套子命令优先拆开分步执行，不要询问用户

### 小改动纪律（防范围膨胀）

- 复杂或涉及多文件的任务：开工前列出改动文件清单和改动点，确认后再动手
- 单文件/微小 Bug 修复：直接改，不需要写计划
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
- 自动化验证：改动后不要本地 go run，直接 Docker 重建对应容器并验证：
  - 后端改动：`docker compose up -d --build go-app`，然后 curl 调用新增/修改的 API 确认返回值
  - 前端改动：`docker compose up -d --build nginx`，然后浏览器打开页面确认渲染
  - agent-browser 自动化验证（登录后用 `eval` 查 DOM）：`agent-browser --ignore-https-errors --args "--no-sandbox" open "https://localhost:8443"`。**全局选项（`--ignore-https-errors`、`--args`）必须放在 `open` 子命令之前**；本机 Chrome 不加 `--no-sandbox` 会启动失败（exit 3）；忽略证书**必须用官方标志 `--ignore-https-errors`**，把 Chrome 原生 `--ignore-certificate-errors` 塞进 `--args` 会与 `--no-sandbox` 冲突导致 Chrome 无法启动
  - 无法手工回归测试的，生成手工回归测试建议清单，交由用户验证

### 复用优先（防重复造轮子）

- 写任何工具/函数前，先搜索：标准库 → 项目已有依赖（go.mod / web/package.json）→ 项目内已有实现（internal/、utils/）是否覆盖该功能
- 已有依赖能力范围内直接用库 API（JWT/Excel/WebSocket/Redis/MQ/GORM 均已引入），不手写
- 项目已有现成实现（utils/hash、utils/jwt、utils/uuid、utils/response 等）直接复用，不另起炉灶
- 标准库和已有依赖都没有时，优先引入成熟三方库（选主流、维护活跃），不自己造轮子；引入前说明理由并经同意
- 只有标准库、已有依赖、成熟三方库都不合适时，才允许自研实现
