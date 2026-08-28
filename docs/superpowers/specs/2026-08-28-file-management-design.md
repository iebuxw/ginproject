# 文件管理功能设计

日期：2026-08-28
状态：已与用户确认

## 背景

参照另一后台系统的截图，为本平台（Go/Gin + Vue 2 + Element UI）新增「文件管理」：公共文件库，支持上传、按文件名搜索、列表展示、下载、删除。菜单挂在「运维管理」目录下，与定时任务、数据库备份并列。

## 已确认的需求决策

- **上传限制**：单文件 ≤100MB；扩展名黑名单仅挡可执行文件（exe/dll/bat/cmd/com/msi/vbs/sh/reg，大小写不敏感），其余类型不限。
- **预览**：图片显示缩略图（点击可看大图）；非图片显示通用文件图标，不做在线预览。
- **可见范围**：公共文件库，有「文件管理」权限的用户都能看到/下载全部文件；上传者列仅作展示。
- **删除**：删记录 + 删物理文件一起；物理删除失败仅记日志不阻断。
- **菜单位置**：本平台无「运维管理」目录（实际一级目录为 系统管理/日志管理/数据字典/任务管理），按用户决定**新建一级目录「运维管理」**，文件管理作为其下二级菜单。

## 架构选型

采用**本地磁盘 + 复用现有静态服务**（方案 A）：

- 文件存 `./uploads/files/<randomHex(16)><ext>`，沿用 `internal/controller/upload_controller.go` 的随机重命名方式，天然防重名与路径穿越。
- 图片预览复用已有静态路由 `r.Static("/api/uploads", "./uploads")`（router.go），缩略图零额外代码。
- 下载走专用接口 `ctx.FileAttachment`，返回原始文件名（数据库备份下载同款模式）。
- 不引入对象存储/DB 存文件等新依赖。

否决方案：文件存 MySQL LONGBLOB（大文件性能差、备份暴涨）；MinIO/S3（引入新依赖，过度设计）。

## 数据库（迁移 000013）

新表 `files`：

| 字段 | 类型 | 说明 |
|---|---|---|
| id | BIGINT PK AUTO_INCREMENT | |
| original_name | VARCHAR(255) | 上传时的原始文件名（允许重名） |
| stored_name | VARCHAR(128) | 磁盘存储名 `randomHex(16)+ext` |
| size | BIGINT | 字节数 |
| ext | VARCHAR(32) | 扩展名（不含点，如 jpg） |
| uploader_id | BIGINT | 上传者用户 id |
| uploader_name | VARCHAR(64) | 上传者用户名快照（用户被删后仍可展示，免 join） |
| created_at | DATETIME | 复用 `model.DateTime`，手动赋值 |

迁移文件 `migrations/000013_add_file_management.up.sql` / `.down.sql`：

菜单种子（已核实运行库真实菜单树：现有菜单 id 最大 55，一级目录 path 前缀均为 `/system`）：

- 一级目录：id=56「运维管理」，path `/system/ops-mgr`，icon `el-icon-s-tools`，type=1，sort=5。
- 二级菜单：id=57「文件管理」，parent=56，path `/system/file`，permission `file:list`，icon `el-icon-folder`，type=2，sort=1。
- 按钮（type=3，parent=57）：id=58「上传」`file:upload`、id=59「下载」`file:download`、id=60「删除」`file:delete`（`file:list` 挂在页面菜单上，与 db_backup 模式一致）。
- `INSERT IGNORE INTO role_menus` 将 56/57/58/59/60 绑定角色 1。全部用显式 id + `INSERT IGNORE` 保证幂等，写法照抄 `000011_add_backup_menus_and_tasks.up.sql`。
- down：删 role_menus 绑定、删菜单（id 56-60）、删表。
- 不修改任何已应用的迁移文件。

## 后端

沿用 router → controller → service → dao → model 分层，直接依赖具体类型（项目惯例，无接口抽象）。

新增文件：

- `internal/model/file.go`：File 结构体 + `TableName()`。
- `internal/dao/file_dao.go`：`FindPage(page, pageSize int, name string)`（name 对 original_name 模糊匹配，`Order("id DESC")`）、`Create`、`FindByID`、`Delete`。
- `internal/service/file_service.go`：
  - `Upload`：校验扩展名黑名单与 `header.Size ≤ 100MB`（写盘前先查 Size）→ 保存到 `./uploads/files/` → 落库（记录 uploader_id + uploader_name 快照）。
  - `Delete`：删记录 + 删物理文件；物理删除失败仅 `log.Printf`，不返回错误。
  - `GetForDownload`：按 id 查记录，返回磁盘路径 + 原始文件名。
  - 构造时 `os.MkdirAll("./uploads/files", 0755)`（db_backup_service 同款）。
- `internal/controller/file_controller.go`：
  - `GET /api/files?page=&page_size=&name=`：分页列表，响应 `{list,total,page,page_size}`。
  - `POST /api/files/upload`：multipart，字段名 `file`。
  - `GET /api/files/download/:id`：`ctx.FileAttachment` 返回原始文件名。
  - `DELETE /api/files/:id`。

路由注册（`internal/router/router.go` authorized 组）：每条路由挂 `RequirePerm("file:xxx") + RBAC + OperationLogger`；**上传路由不挂 OperationLogger**——该中间件会把整个 body 读进内存并把二进制写进日志表/ES，现状头像上传路由同样刻意未挂。

不新增配置项，存储目录沿用硬编码惯例（与 db_backup 的 `backupDir: "backups"` 一致）。

## 前端（Vue 2 + Element UI，文案全中文）

- `web/src/api/file.js`：list / upload / download(blob) / delete，照 `web/src/api/backup.js` 风格。
- `web/src/views/file/index.vue`（照 `web/src/views/backup/index.vue` 结构）：
  - 搜索栏：文件名输入框 + 搜索/重置按钮。
  - 右上「上传文件」按钮：el-upload 手动上传，`before-upload` 校验扩展名黑名单与 100MB。
  - 表格列：ID / 预览 / 文件名 / 大小（格式化 KB/MB）/ 类型（ext）/ 上传者 / 上传时间（`yyyy-MM-dd HH:mm:ss`）/ 操作（下载、删除；删除带确认对话框）。
  - 预览列：图片用 `el-image` 缩略图（带大图预览，src 走 `/api/uploads/files/<stored_name>`）；非图片显示 `el-icon-document` 图标。
  - 分页：el-pagination，`@current-change` 翻页。
- `web/src/store/modules/permission.js`：componentMap 新增 `'/system/file'` → `@/views/file/index.vue` 映射（不加则路由不生成）。

## 错误处理与边界

- 业务错误（超限、类型拒绝、记录不存在）统一 `utils.Error` 业务码 + HTTP 200（项目惯例）。
- 磁盘存储名随机化，原始文件名允许重复。
- 删除时物理文件删除失败不阻断（记录已删，孤儿文件留待手工清理）。
- **已知边界**：`/api/uploads` 静态路由是公开的（不走 JWT，与现有头像一致），拿到存储名 URL 即可匿名取文件；权限控制在页面与接口层（列表/受控下载/删除），与现状保持同级别。
- `uploads/` 若未被 .gitignore 覆盖则补充忽略。

## 测试与验证

项目无单测惯例（仅 logger_test.go），验证以 Docker 重建 + 手工/自动化回归为主：

1. 后端改动：`docker compose up -d --build go-app`，curl 验证：上传正常文件、上传 exe（拒绝）、上传超 100MB（拒绝）、列表搜索、下载内容比对、删除后确认磁盘文件消失、无权限角色访问被拒。
2. 前端改动：`docker compose up -d --build nginx`，agent-browser 打开 `https://localhost:8443` 验证页面渲染、上传、预览、下载、删除全流程。
3. 迁移验证：重启后确认 `files` 表与菜单种子就位，admin 角色可见「运维管理 → 文件管理」菜单。
