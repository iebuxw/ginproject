# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Backend (Go)
```bash
# 本地运行（依赖本地 .env 中的 MYSQL_HOST/REDIS_HOST 指向 docker 服务）
go run cmd/server/main.go

# 编译
go build -o server ./cmd/server/

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

**默认管理员：** admin / admin

## 架构

```
cmd/server/main.go          # 入口：手动组装依赖链，自动建表，种子数据
internal/
  config/                    # Viper 读取 .env
  router/                    # Gin 路由注册，中间件链
  middleware/                # CORS → JWT → RequirePerm → RBAC → OperationLogger
  controller/                # 请求处理，参数绑定，调用 service
  service/                   # 业务逻辑（密码哈希、菜单树、token 黑名单）
  dao/                       # GORM 数据访问
  model/                     # 结构体：User、Role、Menu、OperationLog、DateTime
```

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
- 创建时间用自定义 `DateTime` 类型，JSON 格式 `2006-01-02 15:04:05`

## 注意事项

- `utils.Error` 返回 HTTP 200（业务错误码），需要改 HTTP 状态码用 `ErrorWithStatus`
- `OperationLog.Module` / `Action` 字段中间件未填充，当前始终为空
- 用户管理 CRUD 不支持分配角色
- `.env` 和 `web/dist/` 被 gitignore，Docker 在构建阶段自行编译前端
- `DateTime` 类型不会触发 GORM 自动时间戳，需手动设置 `CreatedAt`
- 手动操作 MySQL 插入中文时需加 `--default-character-set=utf8mb4`，否则乱码
- UI 文案全部中文
