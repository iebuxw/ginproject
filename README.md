# ginadmin 后台管理系统

Go(Gin) + Vue2 (Element UI) 单体仓库。UI 全中文，注释倾向中文。

## 技术栈

| 分类 | 技术 |
|---|---|
| 后端 | Go、Gin、GORM、golang-migrate |
| 前端 | Vue 2、Element UI |
| 数据库/缓存 | MySQL 5.7、Redis |
| 中间件 | RabbitMQ（amqp091-go）、Elasticsearch 7.17（IK 分词，操作日志全文检索） |
| 实时通信 | WebSocket（gorilla/websocket） |
| 部署 | Docker、docker-compose、nginx（HTTPS/TLS 终止） |
| 其他 | robfig/cron（定时任务）、excelize（Excel 导出）、JWT 认证 + RBAC 权限、Swagger |

## 快速开始

```bash
./gen-cert.sh              # 首次部署/换机器：生成自签证书（certs/ 不入库）
docker compose up -d       # 启动所有服务
```

- 访问地址：https://localhost:8443（自签证书，浏览器首次访问需点"高级 → 继续访问"）
- HTTP：http://localhost:8080（不自动跳转 HTTPS）
- 默认账号：`admin` / `123456`

## 线上部署（阿里云 ECS）

### 部署步骤

```bash
# 1. 克隆代码
git clone <repo-url> && cd ginproject

# 2. 创建生产 .env（不入库，需手动创建）
cp .env.example .env
vi .env    # 替换为生产密码（MYSQL_PASSWORD、JWT_SECRET、REDIS_PASSWORD、RABBITMQ_USER/PASSWORD 等）

# 3. 替换 SSL 证书（可选，没域名先用自签证书）
# 把真实证书放到 certs/server.crt 和 certs/server.key

# 4. 启动（生产配置只暴露 80/443，内部服务无端口映射）
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

### 本地与线上区别

| | 本地开发 | 线上部署 |
|---|---|---|
| 启动命令 | `docker compose up -d` | `docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d` |
| 端口映射 | 全开（3307/6380/8080/8443 等） | 只开 80/443 |
| `.env` | 项目根目录已有 | 手动创建，填生产密码 |
| 证书 | 自签 localhost 证书 | 真实域名证书（可选） |

### 离线部署（服务器无法联网）

服务器无法拉取镜像时，先在本地（有网环境）构建并导出：

```bash
# 本地构建
docker compose build

# 导出所有镜像
docker save ginproject-go-app ginproject-nginx ginproject-elasticsearch mysql:5.7 redis:3.2-alpine rabbitmq:3-management kibana:7.17.15 -o images.tar

# 传到服务器
scp images.tar root@服务器IP:/root/
```

服务器上导入并启动：

```bash
docker load < images.tar
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

## HTTPS 与证书

- `certs/` 目录内为自签 localhost 证书，随代码入库，供本地开发使用
- **线上部署**：用真实域名证书替换 `certs/server.crt` 和 `certs/server.key`，重启 nginx 即可
- **换机器或重新克隆后**：先运行 `./gen-cert.sh` 生成自签证书，再 `docker compose up -d --build nginx`
- 脚本幂等：`certs/server.crt` 与 `certs/server.key` 已存在时跳过，不会覆盖既有证书

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
| nginx | 8080（HTTPS 8443） | - |
| mysql | 容器3306 / 宿主3307 | root / admin123 |
| redis | 容器6379 / 宿主6380 | 无密码 |
| rabbitmq | 5672（管理台 15672） | guest / guest |
| elasticsearch | 9200 | - |
| kibana | 5601 | - |

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

`migrations/` 目录存放成对的 `.up.sql` / `.down.sql` 文件，按 `00000N` 递增编号，具体文件以目录现状为准。

### 新增迁移

1. 在 `migrations/` 目录创建新的 `.up.sql` 和 `.down.sql` 文件
2. 文件名递增，如 `000012_add_xxx.up.sql`
3. 重启应用自动执行：`docker compose up -d --build go-app`

### 迁移规范

- 建表用 `CREATE TABLE IF NOT EXISTS`（幂等）
- 种子数据用 `INSERT IGNORE`（已存在则跳过）
- 种子数据必须指定 `id` 列，防止自动分配导致重复
- `down.sql` 按外键依赖逆序删除

## 功能模块

| 模块 | 说明 | 前端页面 | 权限点 |
|---|---|---|---|
| 仪表盘 | CPU/内存/磁盘/Go 运行时信息 | /dashboard（登录默认页） | - |
| 管理员管理 | 管理员 CRUD | /system/user | user:* |
| 角色管理 | 角色 CRUD 与菜单授权 | /system/role | role:* |
| 菜单管理 | 菜单/权限点维护，前端路由由菜单树动态生成 | /system/menu | menu:* |
| 数据字典 | 字典类型+字典项（下拉框配置） | /system/dict-type | dict:* |
| 操作日志 | 请求记录 + ES 全文检索，支持 Excel 异步导出 | /system/log | log:* |
| 登录日志 | 登录成功/失败记录 | /system/login-log | login-log:list |
| 定时任务 | 6 段 cron，命令/HTTP 两种模式，执行日志，立即执行 | /system/task（执行日志 /system/task-logs） | cron:* |
| 数据库备份 | mysqldump 备份/恢复，恢复前自动创建快照，恢复需输入"确认恢复" | /system/backup | db_backup:* |

### 后台能力（无页面）

- **登录异常邮件告警**：登录失败发布 RabbitMQ 消息 + Redis 限频，worker 消费后 SMTP 发信
- **异步导出**：RabbitMQ 队列消费生成 Excel，完成后经 WebSocket（`/api/ws`，按 user_id 推送）通知前端
- **操作日志双写**：MySQL 落库 + Elasticsearch（IK 中文分词）全文检索，ES 不可用时自动降级为 MySQL

## 数据字典

管理系统中的下拉框选项、状态码、类型枚举等配置数据。

### 功能

- **字典类型**：定义字典分类（如性别、状态），code 唯一
- **字典数据**：每个类型下的具体选项（如 男=1，女=2）
- **前端路由**：`/system/dict-type`，类型列表 + 字典项抽屉

### API

| 方法 | 路径 | 说明 |
|---|---|---|
| GET/POST | `/api/dict-types` | 字典类型列表/新增 |
| GET/PUT/DELETE | `/api/dict-types/:id` | 字典类型详情/编辑/删除 |
| GET/POST | `/api/dict-data` | 字典数据列表/新增 |
| GET | `/api/dict-data/by-code/:code` | 按类型编码取字典数据（下拉框用） |
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
