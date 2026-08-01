# 后台管理骨架 - 设计文档

## 概述

基于 Go/Gin + GORM + MySQL + Redis + Vue2 + Nginx 的标准后台管理系统骨架，支持 Docker Compose 一键编排启动。

## 技术栈

| 层 | 技术 | 版本 |
|---|---|---|
| 后端框架 | Gin | v1.9+ |
| ORM | GORM | v1.25+ |
| 数据库 | MySQL | 5.7 |
| 缓存 | Redis | 3.2-alpine |
| 鉴权 | JWT (golang-jwt/v4) | v4 |
| 配置 | Viper | v1 |
| 前端 | Vue2 + Element UI | Vue2 |
| 反向代理 | Nginx | latest-alpine |
| 容器编排 | Docker Compose | 3.8 |
| Go 版本 | Go | 1.18.9 |

## 功能模块

1. **认证模块** — 登录/登出，JWT + Redis 黑名单
2. **用户管理** — CRUD + 分页搜索
3. **角色管理** — CRUD + 菜单权限分配（RBAC）
4. **菜单管理** — 树形菜单 CRUD（目录/菜单/按钮三级）
5. **操作日志** — 操作记录分页查询+筛选

## 数据库设计

### users
| 字段 | 类型 | 说明 |
|---|---|---|
| id | bigint PK | 自增 |
| username | varchar(64) UNIQUE | 用户名 |
| password | varchar(255) | bcrypt 哈希 |
| email | varchar(128) | 邮箱 |
| phone | varchar(20) | 手机号 |
| status | tinyint | 1=启用 0=禁用 |
| last_login_at | datetime | 最后登录时间 |
| created_at / updated_at | datetime | GORM 自动管理 |

### roles
| 字段 | 类型 | 说明 |
|---|---|---|
| id | bigint PK | 自增 |
| name | varchar(64) | 角色名 |
| code | varchar(64) UNIQUE | 角色标识 |
| description | varchar(255) | 描述 |
| status | tinyint | 1=启用 0=禁用 |
| created_at / updated_at | datetime | GORM 自动管理 |

### user_roles (多对多)
| 字段 | 类型 | 说明 |
|---|---|---|
| user_id | bigint FK | 用户ID |
| role_id | bigint FK | 角色ID |

### menus
| 字段 | 类型 | 说明 |
|---|---|---|
| id | bigint PK | 自增 |
| parent_id | bigint | 父级ID，0=顶级 |
| name | varchar(64) | 菜单名 |
| icon | varchar(64) | 图标 |
| path | varchar(255) | 路由路径 |
| type | tinyint | 1=目录 2=菜单 3=按钮 |
| permission | varchar(128) | 权限标识，如 user:add |
| sort | int | 排序 |
| status | tinyint | 1=启用 0=禁用 |
| created_at / updated_at | datetime | GORM 自动管理 |

### role_menus (多对多)
| 字段 | 类型 | 说明 |
|---|---|---|
| role_id | bigint FK | 角色ID |
| menu_id | bigint FK | 菜单ID |

### operation_logs
| 字段 | 类型 | 说明 |
|---|---|---|
| id | bigint PK | 自增 |
| operator_id | bigint | 操作人ID |
| module | varchar(64) | 操作模块 |
| action | varchar(64) | 操作类型 |
| method | varchar(10) | HTTP 方法 |
| path | varchar(255) | 请求路径 |
| params | text | 请求参数 |
| response | text | 响应内容 |
| duration | int | 耗时(ms) |
| ip | varchar(45) | 操作IP |
| created_at | datetime | GORM 自动管理 |

## API 设计

统一响应格式：
```json
{ "code": 200, "message": "success", "data": {} }
```

| 方法 | 路径 | 说明 | 权限 |
|---|---|---|---|
| POST | /api/auth/login | 登录 | 公开 |
| POST | /api/auth/logout | 登出 | 需登录 |
| GET | /api/auth/userinfo | 当前用户信息+菜单+权限 | 需登录 |
| GET | /api/users | 用户列表(分页) | user:list |
| GET | /api/users/:id | 用户详情 | user:query |
| POST | /api/users | 新增用户 | user:add |
| PUT | /api/users/:id | 编辑用户 | user:edit |
| DELETE | /api/users/:id | 删除用户 | user:delete |
| GET | /api/roles | 角色列表(分页) | role:list |
| GET | /api/roles/:id | 角色详情+菜单ID列表 | role:query |
| POST | /api/roles | 新增角色 | role:add |
| PUT | /api/roles/:id | 编辑角色(含菜单分配) | role:edit |
| DELETE | /api/roles/:id | 删除角色 | role:delete |
| GET | /api/menus | 菜单树 | menu:list |
| POST | /api/menus | 新增菜单 | menu:add |
| PUT | /api/menus/:id | 编辑菜单 | menu:edit |
| DELETE | /api/menus/:id | 删除菜单(有子项拒绝) | menu:delete |
| GET | /api/logs | 日志列表(分页+筛选) | log:list |

## 中间件链

```
Request → CORS → JWT（公开路由跳过）→ RBAC（检查权限标识）→ Logger（记录操作日志）→ Controller
```

## Go 项目结构

```
ginproject/
├── cmd/server/main.go
├── internal/
│   ├── config/config.go
│   ├── model/          # GORM 模型
│   ├── dao/            # 数据访问层
│   ├── service/        # 业务逻辑层
│   ├── controller/     # HTTP 处理
│   ├── middleware/      # JWT/RBAC/CORS/Logger
│   ├── router/router.go
│   └── utils/           # response/jwt/hash
├── web/                 # Vue2 前端
├── docker/
│   ├── Dockerfile.go
│   └── nginx.conf
├── docker-compose.yml
├── go.mod
└── .env
```

## 前端设计

- 框架：Vue2 + Element UI
- 布局：侧边栏 + 顶部导航 + 内容区三段式
- 动态路由：根据后端返回的菜单树动态生成
- 状态管理：Vuex 管理用户 token、权限列表、菜单树
- 页面：登录、Dashboard、用户管理、角色管理、菜单管理、操作日志

## Docker 编排拓扑

```
nginx (:8080)
  ├── /          → web/ 静态文件
  └── /api/*     → go-app:8000

go-app (:8000 内部)
  ├── 环境变量读取 MYSQL_HOST / REDIS_HOST
  ├── GORM AutoMigrate 建表
  ├── 首次启动创建 admin/admin 默认管理员
  └── depends_on → mysql (healthy), redis (healthy)

mysql:5.7 (:3306)
  └── volume: mysql-data

redis:3.2-alpine (:6379)
  └── volume: redis-data
```

## 默认数据

- 管理员：`admin` / `admin`
- 自动创建"超级管理员"角色，分配全部菜单权限
- 预置菜单：系统管理（目录）→ 用户管理/角色管理/菜单管理（菜单）→ 各 CRUD 按钮
- Redis 首次登录前无黑名单，按需写入

## 验证方式

1. `docker-compose up -d` 一键启动所有服务
2. 访问 `http://localhost:8080` 进入登录页
3. 使用 admin/admin 登录
4. 对用户/角色/菜单执行 CRUD 操作
5. 查看操作日志记录
6. 登出后再次使用旧 token 请求 → 返回 401（黑名单生效）
