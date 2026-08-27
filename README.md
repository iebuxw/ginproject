# ginadmin 后台管理系统

Go(Gin) + Vue2 (Element UI) 单体仓库。UI 全中文，注释倾向中文。

## 快速开始

```bash
./gen-cert.sh              # 首次部署/换机器：生成自签证书（certs/ 不入库）
docker compose up -d       # 启动所有服务
```

- 访问地址：https://localhost:8443（自签证书，浏览器首次访问需点"高级 → 继续访问"）
- 访问 http://localhost:8080 会自动 301 跳转到 https://localhost:8443
- 默认账号：`admin` / `123456`

## HTTPS 与证书

- `certs/` 目录不入库（`.gitignore` 已排除），私钥不随代码分发
- **换机器或重新克隆后**：先运行 `./gen-cert.sh` 生成证书，再 `docker compose up -d --build nginx`
- 脚本幂等：`certs/server.crt` 与 `certs/server.key` 已存在时跳过，不会覆盖既有证书
- 证书有效期 10 年（SAN: localhost / 127.0.0.1），过期后删除 `certs/` 重跑脚本即可

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

## 数据库迁移

使用 [golang-migrate](https://github.com/golang-migrate/migrate) 管理数据库 schema 和种子数据，启动时自动执行未运行的迁移。

### 迁移文件结构

```
migrations/
  000001_create_schema.up.sql          # 建表（10张表）
  000001_create_schema.down.sql        # 删表（回滚）
  000002_seed_menus.up.sql             # 菜单种子数据（31条）
  000002_seed_menus.down.sql           # 清空菜单
  000003_seed_admin_and_dict.up.sql    # admin用户、角色、字典种子
  000003_seed_admin_and_dict.down.sql  # 清空用户/角色/字典
```

### 新增迁移

1. 在 `migrations/` 目录创建新的 `.up.sql` 和 `.down.sql` 文件
2. 文件名递增，如 `000004_add_xxx.up.sql`
3. 重启应用自动执行：`docker compose up -d --build go-app`

### 迁移规范

- 建表用 `CREATE TABLE IF NOT EXISTS`（幂等）
- 种子数据用 `INSERT IGNORE`（已存在则跳过）
- 种子数据必须指定 `id` 列，防止自动分配导致重复
- `down.sql` 按外键依赖逆序删除

## 数据字典

管理系统中的下拉框选项、状态码、类型枚举等配置数据。

### 功能

- **字典类型**：定义字典分类（如性别、状态），code 唯一
- **字典数据**：每个类型下的具体选项（如 男=1，女=2）
- **前端路由**：`/system/dict-type`，左右分栏布局

### API

| 方法 | 路径 | 说明 |
|---|---|---|
| GET/POST | `/api/dict-types` | 字典类型列表/新增 |
| GET/PUT/DELETE | `/api/dict-types/:id` | 字典类型详情/编辑/删除 |
| GET/POST | `/api/dict-data` | 字典数据列表/新增 |
| GET/PUT/DELETE | `/api/dict-data/:id` | 字典数据详情/编辑/删除 |

### 使用示例

```js
// 前端获取字典数据（用于下拉框）
import { getDictData } from '@/api/dict'
const res = await getDictData({ dict_type_id: 1 })
const options = res.data.list // [{label:"男", value:"1"}, ...]
```

### 权限

`dict:list`、`dict:query`、`dict:add`、`dict:edit`、`dict:delete`
