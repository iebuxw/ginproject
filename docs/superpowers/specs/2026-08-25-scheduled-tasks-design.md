# 定时任务功能设计文档

日期：2026-08-25
状态：已确认（用户逐节审批通过）

## 1. 目标

为 ginproject 后台管理系统新增定时任务功能：管理员可以在页面上创建、编辑、启停、删除定时任务，任务到点由服务端发起 HTTP 回调；页面可查看每次执行的结果日志，并支持手动"立即执行一次"。

## 2. 范围（YAGNI 裁剪）

**包含：**

- 任务 CRUD + 启停（页面管理）
- HTTP 回调执行（GET/POST，可配 headers/body/超时）
- 执行日志（每次执行记录状态/耗时/HTTP 状态码/错误）
- 立即执行一次（手动触发，写 manual 日志）
- cron 表达式校验（6 段含秒）
- 调度器热更新（增删改/启停即时生效）
- 重叠执行防护（上次未执行完则跳过本次）

**明确不包含：**

- 失败自动重试
- 失败邮件告警（复用现有 AlertMailService 的能力已具备，但本功能暂不接）
- 分布式调度/多实例锁（项目为 docker compose 单实例部署）
- Shell 命令任务、内置业务函数任务
- 执行日志自动清理

## 3. 技术选型

- 调度器：`github.com/robfig/cron/v3`（兼容 go 1.18，业界 Go 项目事实标准）
  - 解析器：`cron.NewParser(cron.Second|Minute|Hour|Dom|Month|Dow)`，支持 6 段含秒表达式（`秒 分 时 日 月 周`，对齐 XXL-JOB 习惯）
- 部署形态：单机调度，服务启动时从 DB 全量加载启用任务；重启后天然恢复
- 分层：照搬项目现有模式（model → dao → service → controller → router，无接口抽象）

## 4. 数据模型

迁移文件：`migrations/000005_create_cron_tables.up.sql` / `.down.sql`（新表 + 菜单种子合并一个迁移，`IF NOT EXISTS` + `INSERT IGNORE` 幂等）

### 4.1 表 `cron_tasks`（任务定义）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT PK AUTO_INCREMENT | |
| name | VARCHAR(64) NOT NULL | 任务名称 |
| url | VARCHAR(255) NOT NULL | HTTP 回调地址 |
| method | VARCHAR(8) NOT NULL DEFAULT 'GET' | GET / POST |
| headers | TEXT | JSON 自定义请求头（可选，创建时校验 JSON） |
| body | TEXT | POST 请求体（可选） |
| cron | VARCHAR(32) NOT NULL | 6 段 cron 表达式（秒 分 时 日 月 周） |
| timeout | INT NOT NULL DEFAULT 30 | HTTP 超时秒数（1-300） |
| status | TINYINT NOT NULL DEFAULT 1 | 1=启用 0=停用 |
| remark | VARCHAR(255) DEFAULT '' | 备注 |
| created_at | DATETIME | |
| updated_at | DATETIME | |

### 4.2 表 `cron_task_executions`（执行日志）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT PK AUTO_INCREMENT | |
| task_id | BIGINT NOT NULL, INDEX | 关联任务 |
| trigger | VARCHAR(16) NOT NULL | `cron` / `manual` |
| status | TINYINT NOT NULL | 0=成功 1=失败 2=跳过（上次未执行完） |
| http_status | INT | HTTP 响应状态码（失败/跳过后为空） |
| response | TEXT | 响应体摘要（截断 2000 字符） |
| error_msg | VARCHAR(255) DEFAULT '' | 失败原因（超时/网络错误/非 2xx 状态码） |
| duration_ms | INT | 执行耗时（毫秒） |
| created_at | DATETIME | |

### 4.3 菜单种子（同迁移文件）

> 注：原设计目录 id=4 与现有种子冲突（id=4 已被「用户管理」占用，000002 种子），第一次修正为 32 起；实际执行时发现现有库菜单 id 已用到 34（历史数据与迁移种子不同：32-34 已被字典编辑/删除/类型占用），故最终从 41 起顺延。

- 一级目录 id=41「任务管理」icon `el-icon-alarm-clock` path `/system/task-mgr` sort=4
- 二级菜单 id=42「定时任务」path `/system/task` icon `el-icon-alarm-clock` sort=1
- 按钮 id 43-49：
  - 43 `cron:list` 任务列表
  - 44 `cron:query` 任务查询
  - 45 `cron:add` 任务新增
  - 46 `cron:edit` 任务编辑
  - 47 `cron:delete` 任务删除
  - 48 `cron:run` 立即执行
  - 49 `cron:log` 执行日志
- role_menus 关联：admin（role_id=1）补 `(1,41)…(1,49)` 全量关联（对齐现有库已绑 1-34 的做法）

## 5. 后端架构

### 5.1 新增 `internal/scheduler/` 包

```go
type Scheduler struct {
    cron    *cron.Cron
    taskDAO *dao.CronTaskDAO
    execDAO *dao.CronTaskExecutionDAO
    running sync.Map // taskID → bool，防重叠执行
}

func NewScheduler(taskDAO, execDAO) *Scheduler
func (s *Scheduler) Start()         // main.go goroutine 调用：全量加载启用任务并启动
func (s *Scheduler) Reload()        // 任务增删改/启停后调用：旧实例 Stop → 重建 → 注册全部启用任务 → Start
func (s *Scheduler) RunNow(id uint) // 立即执行一次（Controller 调用），写 trigger=manual 日志
func (s *Scheduler) execute(task *model.CronTask)
```

关键行为：

- **热更新**：Reload 全量重建实例（任务量小，简单可靠）。`cron.Stop()` 会等待运行中的 job 结束，无 goroutine 泄漏。
- **防重叠**：`execute` 前 Check-and-Set `running`；已运行则跳过，写一条 `status=2`（跳过）日志；执行完删除标记。
- **执行**：`http.Client{Timeout: timeout * time.Second}`；GET 直接请求；POST 带 body + headers（含 `Content-Type` 由 headers 指定）；响应体截断 2000 字符；HTTP 非 2xx 记为失败（error_msg 含状态码）；`execute` 内 `defer recover()` 防 panic 拖垮调度器；日志写库失败仅 `log.Printf`，不中断执行。
- **停止**：应用退出时无需特殊处理（进程退出即停）。

### 5.2 分层文件（照搬现有模式）

| 文件 | 内容 |
|------|------|
| `internal/model/cron_task.go` | CronTask、CronTaskExecution 两个 struct + TableName |
| `internal/dao/cron_task_dao.go` | CronTaskDAO：Create/Update/Delete/FindByID/FindPage(关键词)/FindEnabled；CronTaskExecutionDAO：Create/FindByTaskIDPage |
| `internal/service/cron_task_service.go` | 业务校验（cron 表达式解析、URL 非空、method 合法、timeout 范围、headers JSON）+ 调 DAO + 变更后调 `scheduler.Reload()`；RunNow 调 `scheduler.RunNow` |
| `internal/controller/cron_task_controller.go` | 8 个 handler，Swagger 注解齐全（格式参照 dict_controller.go） |

Service 持有 Scheduler 引用（`NewCronTaskService(taskDAO, execDAO, scheduler)`）。

### 5.3 main.go 组装

```go
cronTaskDAO := dao.NewCronTaskDAO(db)
cronTaskExecutionDAO := dao.NewCronTaskExecutionDAO(db)
taskScheduler := scheduler.NewScheduler(cronTaskDAO, cronTaskExecutionDAO)
go taskScheduler.Start()
cronTaskService := service.NewCronTaskService(cronTaskDAO, cronTaskExecutionDAO, taskScheduler)
cronTaskCtrl := controller.NewCronTaskController(cronTaskService)
// router.Setup(...) 追加 cronTaskCtrl 参数
```

### 5.4 路由（`/api`，权限点对齐按钮）

| 方法 | 路径 | 权限 | 中间件 |
|------|------|------|--------|
| GET | /cron-tasks | cron:list | RequirePerm + RBAC |
| GET | /cron-tasks/:id | cron:query | RequirePerm + RBAC |
| POST | /cron-tasks | cron:add | RequirePerm + RBAC + OperationLogger |
| PUT | /cron-tasks/:id | cron:edit | RequirePerm + RBAC + OperationLogger |
| DELETE | /cron-tasks/:id | cron:delete | RequirePerm + RBAC + OperationLogger |
| PUT | /cron-tasks/:id/status | cron:edit | RequirePerm + RBAC + OperationLogger |
| POST | /cron-tasks/:id/run | cron:run | RequirePerm + RBAC + OperationLogger |
| GET | /cron-tasks/:id/executions | cron:log | RequirePerm + RBAC |

（OperationLogger 仅记录写操作，GET 不记录，符合现有中间件行为）

### 5.5 错误处理

- 创建/编辑：参数校验失败返回业务错误（HTTP 200 + code，走 `utils.Error`）
- 执行失败：只写入执行日志，不影响调度器继续跑
- 重叠执行：跳过 + status=2 日志
- 调度器崩溃恢复：进程重启后 Start() 从 DB 加载，天然恢复
- HTTP 客户端复用（`http.DefaultClient` 包装或包内共享 client），避免每次新建

## 6. 前端

| 文件 | 内容 |
|------|------|
| `web/src/api/cron.js` | 8 个接口封装（参照 dict.js） |
| `web/src/views/task/index.vue` | 任务管理单页面 |
| `web/src/store/modules/permission.js` | componentMap 加一行：`'/system/task': () => import('@/views/task/index.vue')` |

页面结构：

- 顶部：新建按钮 + 名称/URL 关键词搜索框
- 表格列：任务名称、URL、方法（el-tag）、cron 表达式、状态（el-switch 直接切换调 status 接口）、最近执行状态、操作（编辑 / 立即执行 / 日志 / 删除）
- 新建/编辑对话框（el-dialog + el-form）：名称、URL、method 下拉（GET/POST）、headers（JSON 文本域）、body（文本域）、cron 表达式（带提示"秒 分 时 日 月 周"）、超时秒、备注；校验：必填项、URL 格式、cron 非空
- 执行日志对话框：内嵌表格（触发方式 tag、状态 tag、HTTP 状态码、耗时、错误信息、执行时间）+ 分页
- 立即执行：确认弹窗 → 调 run 接口 → 成功提示"已触发执行"
- 删除：确认弹窗（对齐现有页面风格）
- 状态列使用 el-switch，切换即调 status 接口，失败回滚

## 7. 测试策略

- 项目现状无测试、无 lint/typecheck/CI（CLAUDE.md 明确），本功能遵循现状不引入单测框架
- cron 表达式解析校验由 robfig/cron 库自身保证
- 验证方式：docker 起服务 → 页面手动验证全流程：
  1. 新建任务（配一个本地回显接口或 httpbin 类 URL）
  2. 等 cron 到点，确认执行日志落库
  3. 立即执行一次，确认 manual 日志
  4. 停用/启用，确认热更新生效
  5. 改 cron 表达式，确认立即生效
  6. 配一个不可达 URL，确认失败日志（error_msg）
- Swagger 注解齐全，完成后 `swag init -g cmd/server/main.go` 重新生成

## 8. 提交拆分

按 CLAUDE.md 提交习惯，定时任务是单一功能模块，参照"新增数据字典功能"先例，一个 commit 包含后端 + 迁移 + 前端。
