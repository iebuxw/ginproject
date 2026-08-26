# 定时任务易用性改进设计

日期：2026-08-26

## 目标

改进定时任务系统的用户体验，让非技术人员也能轻松创建和管理定时任务。

## 改动范围

1. **Cron 快捷按钮** — 前端 UI，一键生成常用 Cron 表达式
2. **预定义命令别名** — 后端命令注册表 + 前端下拉选择，隐藏 HTTP 细节
3. **执行日志独立页面** — 替代弹窗，支持筛选和查看输出

**不包含**：失败告警、分布式调度、Shell 命令任务。

---

## 一、Cron 快捷按钮（前端）

### 交互设计

在 cron 表达式输入框下方添加一排按钮，点击后自动填入表达式。按钮旁显示一个小时间选择器用于调整小时（默认凌晨 3 点）。

| 按钮 | 生成的 Cron（6 段） | 说明 |
|------|---------------------|------|
| 每分钟 | `0 * * * * *` | 每分钟执行 |
| 每小时 | `0 0 * * * *` | 每小时整点执行 |
| 每天 | `0 0 {hour} * * *` | 每天指定小时执行 |
| 每周 | `0 0 {hour} ? * 1` | 每周一指定小时执行 |
| 每月 | `0 0 {hour} 1 * *` | 每月 1 号指定小时执行 |

- 默认小时为 3（凌晨 3 点）
- 点击按钮后 cron 输入框更新，用户仍可手动微调
- 按钮使用 `el-button-group`，样式紧凑

---

## 二、预定义命令别名

### 2.1 后端命令注册表

在 `internal/scheduler/` 中新增 `commands.go`，定义命令注册表：

```go
type CommandDef struct {
    Name    string            // 命令标识，如 "clean_logs"
    Label   string            // 中文名称，如 "清理过期日志"
    Method  string            // GET 或 POST
    URL     string            // 回调地址
    Headers map[string]string // 固定请求头
    Body    string            // 固定请求体
}

var Commands = map[string]CommandDef{
    "clean_logs": {
        Name:   "clean_logs",
        Label:  "清理过期日志",
        Method: "POST",
        URL:    "/api/logs/cleanup?days=30",
        Headers: map[string]string{
            "X-Cleanup-Secret": "{{CLEANUP_SECRET}}",
        },
    },
    "backup_db": {
        Name:   "backup_db",
        Label:  "数据库备份",
        Method: "POST",
        URL:    "/api/db/backup",
    },
}
```

- `{{CLEANUP_SECRET}}` 是占位符，启动时由 `InjectCleanupSecret` 替换为 .env 实际值
- 调度器执行任务时，若 `command` 字段非空，从注册表取配置执行；为空则用任务记录的 HTTP 配置

### 2.2 新增 API

`GET /api/cron-tasks/commands` — 返回可用命令列表（不需认证也可考虑，但保持在 authorized 组内）

响应格式：
```json
{
  "code": 200,
  "data": [
    { "name": "clean_logs", "label": "清理过期日志" },
    { "name": "backup_db", "label": "数据库备份" },
    { "name": "_custom", "label": "自定义" }
  ]
}
```

`_custom` 是前端约定的特殊值，表示用户要自定义 HTTP 回调。

### 2.3 数据库改动

`cron_tasks` 表新增字段（通过新迁移）：

```sql
ALTER TABLE cron_tasks ADD COLUMN command VARCHAR(64) DEFAULT '' AFTER name;
ALTER TABLE cron_tasks ADD INDEX idx_command (command);
```

- `command` 为空 = 自定义 HTTP 回调模式
- `command` 非空 = 预定义命令模式，执行时从注册表取配置

现有种子任务 `clean_logs` 需更新：设置 `command='clean_logs'`，清空 url/method/headers/body。

### 2.4 调度器改动

`internal/scheduler/scheduler.go` 的 `execute()` 方法增加命令解析逻辑：

```
1. 读取任务的 command 字段
2. 若 command 非空 → 从 Commands 注册表查找，用其 URL/Method/Headers/Body
3. 若 command 为空 → 用任务记录的 URL/Method/Headers/Body（现有逻辑不变）
4. 后续 HTTP 请求发送逻辑不变
```

### 2.5 Service 层改动

`CronTaskService.Create/Update` 增加校验：
- 若 command 非空，必须在注册表中存在
- 若 command 为 `_custom`，转为空字符串，要求 url 非空
- 若 command 非空且非 `_custom`，url/method/headers/body 可为空（由注册表填充）

### 2.6 前端改动

`web/src/views/task/index.vue` 编辑对话框改造：

**默认模式（command 非空且非 _custom）：**
- 任务名称
- 命令（el-select 下拉，从 API 获取）
- Cron 表达式 + 快捷按钮
- 备注

**自定义模式（command 为空或 _custom）：**
- 任务名称
- 命令（下拉选"自定义"）
- 回调地址、请求方式、请求头、请求体、超时
- Cron 表达式 + 快捷按钮
- 备注

切换命令时：
- 选预定义命令 → 隐藏 HTTP 字段
- 选"自定义" → 展开 HTTP 字段

表格中"方法"列在预定义命令模式下隐藏（因为用户不需要知道）。

---

## 三、执行日志独立页面

### 3.1 路由与菜单

- 新增路由 `/system/task-logs`，对应 `web/src/views/task/logs.vue`
- 菜单迁移中新增：id=50，父级=41（任务管理目录），type=2（菜单页），permission=`cron:log`，path=`task-logs`，component=`task/logs`
- admin 角色自动绑定（INSERT IGNORE）

### 3.2 页面设计（参考截图）

**筛选区：**
- 任务名称（el-select，从 cron_tasks 表加载全部任务名）
- 状态（el-select：全部/成功/失败/跳过）
- 日期范围（el-date-picker type="daterange"）
- 搜索按钮 + 重置按钮

**表格列：**
| 列名 | 字段 | 宽度 |
|------|------|------|
| ID | id | 70 |
| 任务名称 | task_name（关联查询） | - |
| 命令 | command（关联查询） | 120 |
| 状态 | status | 80 |
| 耗时(秒) | duration_ms / 1000 | 100 |
| 执行时间 | created_at | 170 |
| 操作 | 查看输出 / 删除 | 150 |

**查看输出**：点击后弹窗显示 response 内容（el-dialog + pre 标签，等宽字体）。

### 3.3 后端改动

`CronTaskExecutionDAO.FindPage` 增加筛选参数：
- `task_id` — 按任务筛选
- `status` — 按状态筛选
- `start_time` / `end_time` — 按时间范围筛选

`GET /api/cron-tasks/executions` — 新增全局执行日志查询（不传 task_id 查全部），支持上述筛选。

Controller 新增 `ListAllExecutions` handler。

### 3.4 任务列表页改动

原任务列表表格中移除"日志"按钮（改为跳转到执行日志页面并带 task_id 参数），或保留但改为 `router.push` 跳转。

---

## 四、文件改动清单

| 文件 | 改动 |
|------|------|
| `internal/scheduler/commands.go` | **新增** 命令注册表 |
| `internal/scheduler/scheduler.go` | 修改 execute()，支持 command 字段解析 |
| `internal/model/cron_task.go` | CronTask 结构体加 Command 字段 |
| `internal/dao/cron_task_dao.go` | FindEnabled/FindPage 支持 command 字段 |
| `internal/dao/cron_task_execution_dao.go` | FindPage 增加 status/时间范围筛选；新增 FindAllPage |
| `internal/service/cron_task_service.go` | Create/Update 增加 command 校验；新增 ListCommands |
| `internal/controller/cron_task_controller.go` | 新增 Commands/ListAllExecutions handler |
| `internal/router/router.go` | 注册新路由 |
| `migrations/000008_add_command_to_cron_tasks.up.sql` | **新增** ALTER TABLE |
| `migrations/000008_add_command_to_cron_tasks.down.sql` | **新增** 回滚 |
| `migrations/000009_update_seed_task_command.up.sql` | **新增** 更新种子任务 |
| `migrations/000009_update_seed_task_command.down.sql` | **新增** 回滚 |
| `migrations/000010_add_task_logs_menu.up.sql` | **新增** 执行日志菜单 |
| `migrations/000010_add_task_logs_menu.down.sql` | **新增** 回滚 |
| `web/src/api/cron.js` | 新增 commands/listAllExecutions 接口 |
| `web/src/views/task/index.vue` | 编辑对话框改造 + Cron 快捷按钮 |
| `web/src/views/task/logs.vue` | **新增** 执行日志独立页面 |
| `web/src/store/modules/permission.js` | 注册 task/logs 路由映射 |

---

## 五、提交拆分

1. **后端：命令注册表 + 调度器改造** — commands.go + scheduler.go + model + DAO + migration 000008
2. **后端：Service/Controller/路由 + 种子更新** — service + controller + router + migration 000009
3. **前端：编辑对话框改造 + Cron 快捷按钮** — task/index.vue + api/cron.js
4. **后端：执行日志全局查询** — execution DAO + controller + migration 000010
5. **前端：执行日志独立页面** — task/logs.vue + permission.js + api/cron.js
