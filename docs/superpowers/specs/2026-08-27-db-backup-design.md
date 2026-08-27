# 数据库备份功能设计

## 概述

为项目添加数据库备份与恢复功能，包括：手动/定时备份、备份列表管理、恢复、下载、自动清理过期备份。复用现有定时任务系统和预定义命令模式。

## 技术方案

**方案：进程内调用 mysqldump（方案 A）**

go-app 容器安装 `mysql-client` + `gzip`，Go 通过 `os/exec` 调用 `mysqldump` 生成 `.sql.gz` 文件到 `backups/` 目录，元数据存 MySQL `db_backups` 表。恢复用 `mysql` CLI 导入。

**选择理由：**
- `mysqldump` 是 MySQL 官方工具，备份最完整（含视图、存储过程、触发器）
- 与现有 `clean_logs` 命令注入模式完全一致
- Dockerfile 改动极小（加一行 `apt-get install mysql-client`）
- 恢复也用标准 `mysql` CLI，可靠且简单

## 数据模型

### 存储

- **位置：** 项目根目录下 `backups/` 目录（和 `exports/` 模式一致），gitignore
- **文件命名：** `{项目名}_{YYYYMMDD_HHMMSS}.sql.gz`，如 `ginadmin_20260827_020000.sql.gz`

### 数据库表 `db_backups`

| 字段 | 类型 | 说明 |
|---|---|---|
| id | BIGINT PK AUTO_INCREMENT | 主键 |
| filename | VARCHAR(255) NOT NULL | 文件名 |
| file_size | BIGINT | 文件大小（字节） |
| trigger_type | VARCHAR(20) | 触发方式：`cron` / `manual` |
| status | TINYINT | 状态：0=成功，1=失败 |
| type | VARCHAR(20) | 类型：`backup`（常规备份） |
| remark | TEXT | 备注 |
| created_at | DATETIME | 创建时间 |

### Go Model（`internal/model/db_backup.go`）

```go
type DbBackup struct {
    ID          int64    `json:"id" gorm:"primaryKey;autoIncrement"`
    Filename    string   `json:"filename" gorm:"size:255;not null"`
    FileSize    int64    `json:"file_size"`
    TriggerType string   `json:"trigger_type" gorm:"size:20"`
    Status      int      `json:"status"`
    Type        string   `json:"type" gorm:"size:20"`
    Remark      string   `json:"remark" gorm:"type:text"`
    CreatedAt   DateTime `json:"created_at"`
}
```

## 备份/恢复/清理逻辑

### 备份流程

1. 组装文件名：`{DB_NAME}_{YYYYMMDD_HHMMSS}.sql.gz`
2. 构造 mysqldump 命令：`mysqldump -h{host} -P{port} -u{user} -p{pass} {db} | gzip > {filepath}`
3. Go 通过 `os/exec.Command` 执行，捕获 stderr
4. 执行成功后获取文件大小（`os.Stat`），写入 `db_backups` 表
5. 失败时记录错误信息到 remark

### 恢复流程

1. 前端弹出**输入确认对话框**：用户必须在输入框中输入"确认恢复"才能点击确认按钮
2. 警告文案："恢复操作将用备份文件覆盖当前数据库，此操作不可撤销。"
3. 确认后调用恢复 API
4. Service 用 `os/exec` 执行：`gunzip -c {filepath} | mysql -h{host} -P{port} -u{user} -p{pass} {db}`
5. 恢复前先**关闭外键约束**（`SET FOREIGN_KEY_CHECKS=0`），恢复后**重新开启**
6. 恢复完成后返回成功/失败

### 清理流程（`clean_backup` 命令）

1. 查询 `db_backups` 表中 `created_at < NOW() - INTERVAL ? DAY` 的记录
2. 删除对应的物理文件（`os.Remove`）
3. 删除数据库记录
4. 返回删除数量

### 关键细节

- `mysqldump` 和 `mysql` 命令从 go-app 容器内执行，连接 `ginadmin-mysql:3306`
- 数据库连接信息从 `.env` 读取（和现有 MySQL 配置共用）
- 备份/恢复是耗时操作，命令执行超时设置为 300 秒（5 分钟）
- 恢复操作不自动创建快照

## API 接口

| 方法 | 路径 | 权限 | 说明 |
|---|---|---|---|
| GET | `/api/db-backups` | `db_backup:list` | 备份列表（分页，支持日期范围筛选） |
| POST | `/api/db-backups` | `db_backup:add` | 手动创建备份 |
| POST | `/api/db-backups/:id/restore` | `db_backup:restore` | 恢复备份 |
| DELETE | `/api/db-backups/:id` | `db_backup:delete` | 删除备份（删文件+删记录） |
| GET | `/api/db-backups/:id/download` | `db_backup:download` | 下载备份文件 |

### 列表接口响应

```json
{
  "code": 200,
  "data": {
    "list": [{
      "id": 1,
      "filename": "ginadmin_20260827_020000.sql.gz",
      "file_size": 11520,
      "trigger_type": "cron",
      "status": 0,
      "type": "backup",
      "remark": "",
      "created_at": "2026-08-27 02:00:00"
    }],
    "total": 10,
    "page": 1,
    "page_size": 20
  }
}
```

## 定时任务集成

复用现有定时任务系统。`backup_db` 和 `clean_backup` 作为预定义命令注册在 `scheduler/commands.go`，在 `main.go` 启动时注入真实实现。

### 种子数据

通过迁移 `000011_add_backup_menus_and_tasks` 写入 `cron_tasks` 表：

| 任务名 | 命令 | Cron 表达式 | 状态 | 备注 |
|---|---|---|---|---|
| 数据库备份 | `backup_db` | `0 0 2 * * *` | 启用 | 每天凌晨2点自动备份数据库 |
| 清理过期备份 | `clean_backup` | `0 0 4 * * *` | 启用 | 每天凌晨4点清理过期备份 |

种子使用 `INSERT IGNORE` + 显式 id 保证幂等。

### 命令注册

`scheduler/commands.go` 新增 `clean_backup` 命令条目（初始返回"命令未实现"）。`main.go` 启动时替换 `backup_db` 和 `clean_backup` 的 handler 为真实实现。

## 前端页面

### 页面文件

- `web/src/views/backup/index.vue` — 备份列表页
- `web/src/api/backup.js` — API 调用

### 页面布局

**顶部操作栏：**
- 左侧：日期范围选择器（Element UI DateRangePicker）+ 搜索按钮 + 重置按钮
- 右侧：「新增备份」按钮（蓝色），点击直接创建备份

**表格列：**

| 列名 | 字段 | 宽度 |
|---|---|---|
| ID | id | 80px |
| 文件名 | filename | auto |
| 文件大小 | file_size（格式化为 KB/MB） | 120px |
| 触发方式 | trigger_type（定时/手动） | 100px |
| 状态 | status（成功/失败） | 80px |
| 类型 | type（常规备份） | 100px |
| 备注 | remark | auto |
| 创建时间 | created_at | 180px |
| 操作 | — | 200px |

**操作列按钮：**
- 恢复（蓝色）→ 弹出输入确认对话框（输入"确认恢复"后才可点击确认）
- 下载（蓝色）→ 浏览器下载 .sql.gz 文件
- 删除（红色）→ 确认后删除

### 恢复确认对话框

- 标题：确认恢复
- 警告文案（橙色）："恢复操作将用备份文件覆盖当前数据库，此操作不可撤销。"
- 输入提示："请输入 确认恢复 以继续："
- 输入框：用户必须输入"确认恢复"四个字
- 按钮：取消（灰色）+ 确认恢复（红色，输入正确后才可点击）

### 菜单权限种子

在迁移 `000011` 中插入菜单：

- 「数据库备份」菜单（ID 51，parent_id=26 运维管理下，type=2）
- 权限按钮：`db_backup:list`、`db_backup:add`、`db_backup:delete`、`db_backup:restore`、`db_backup:download`（type=3，parent_id=51）

## 文件变更清单

### 新增文件

| 文件 | 说明 |
|---|---|
| `internal/model/db_backup.go` | DbBackup 模型 |
| `internal/dao/db_backup_dao.go` | 备份数据访问 |
| `internal/service/db_backup_service.go` | 备份业务逻辑（调 mysqldump/mysql CLI） |
| `internal/controller/db_backup_controller.go` | 备份 API 控制器 |
| `web/src/views/backup/index.vue` | 备份列表前端页面 |
| `web/src/api/backup.js` | 前端 API 调用 |
| `migrations/000011_add_backup_menus_and_tasks.up.sql` | 迁移：建表 + 菜单 + 种子任务 |
| `migrations/000011_add_backup_menus_and_tasks.down.sql` | 回滚迁移 |

### 修改文件

| 文件 | 说明 |
|---|---|
| `internal/router/router.go` | 注册备份路由 |
| `cmd/server/main.go` | 注入 backupService，替换 backup_db 和 clean_backup handler |
| `internal/scheduler/commands.go` | 注册 clean_backup 命令 |
| `docker/Dockerfile` | 安装 mysql-client |
| `.gitignore` | 添加 backups/ |
| `CLAUDE.md` | 更新架构说明和注意事项 |
