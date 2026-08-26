# 定时任务易用性改进 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 改进定时任务系统易用性：Cron 快捷按钮、预定义命令下拉、执行日志独立页面。

**Architecture:** 后端新增命令注册表（`scheduler/commands.go`），调度器执行时按 command 字段查表取配置；`cron_tasks` 表加 `command` 字段。前端编辑对话框改为"预定义命令模式"（只显示名称/命令/Cron/备注）和"自定义模式"（展开全部 HTTP 字段），Cron 输入框加快捷按钮。执行日志从弹窗改为独立页面，支持筛选。

**Tech Stack:** Go (Gin, GORM, robfig/cron/v3), Vue 2 + Element UI, MySQL, golang-migrate

---

## Task 1: 后端 — 命令注册表 + Model + DAO + 迁移

**Files:**
- Create: `internal/scheduler/commands.go`
- Modify: `internal/model/cron_task.go:8` — CronTask 加 Command 字段
- Modify: `internal/dao/cron_task_dao.go:33-44` — FindPage 搜索条件加 command
- Create: `migrations/000008_add_command_to_cron_tasks.up.sql`
- Create: `migrations/000008_add_command_to_cron_tasks.down.sql`

- [ ] **Step 1: 创建命令注册表 `internal/scheduler/commands.go`**

```go
package scheduler

// CommandDef 预定义命令定义
type CommandDef struct {
	Name    string            // 命令标识，如 "clean_logs"
	Label   string            // 中文名称，如 "清理过期日志"
	Method  string            // GET 或 POST
	URL     string            // 回调地址
	Headers map[string]string // 固定请求头（可选）
	Body    string            // 固定请求体（可选）
}

// Commands 预定义命令注册表
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

// CommandList 返回所有预定义命令的 name + label（供前端下拉使用）
type CommandOption struct {
	Name  string `json:"name"`
	Label string `json:"label"`
}

func CommandList() []CommandOption {
	opts := make([]CommandOption, 0, len(Commands))
	for _, c := range Commands {
		opts = append(opts, CommandOption{Name: c.Name, Label: c.Label})
	}
	return opts
}
```

- [ ] **Step 2: Model 加 Command 字段**

修改 `internal/model/cron_task.go`，在 `Name` 字段后加 `Command`：

```go
// CronTask 定时任务
type CronTask struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Name           string    `gorm:"size:64;not null" json:"name"`
	Command        string    `gorm:"size:64;default:''" json:"command"`
	URL            string    `gorm:"size:255;not null" json:"url"`
	Method         string    `gorm:"size:8;not null;default:GET" json:"method"`
	Headers        string    `gorm:"type:text" json:"headers"`
	Body           string    `gorm:"type:text" json:"body"`
	Cron           string    `gorm:"size:32;not null" json:"cron"`
	Timeout        int       `gorm:"not null;default:30" json:"timeout"`
	Status         int       `gorm:"not null;default:1" json:"status"`
	Remark         string    `gorm:"size:255;default:''" json:"remark"`
	LastExecStatus int       `gorm:"->" json:"last_exec_status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
```

- [ ] **Step 3: DAO FindPage 搜索条件加 command**

修改 `internal/dao/cron_task_dao.go` 的 `FindPage` 方法，`keyword` 搜索条件加 `command`：

```go
if keyword != "" {
    q = q.Where("name LIKE ? OR url LIKE ? OR command LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
}
```

- [ ] **Step 4: 创建迁移 `migrations/000008_add_command_to_cron_tasks.up.sql`**

```sql
ALTER TABLE cron_tasks ADD COLUMN command VARCHAR(64) DEFAULT '' AFTER name;
ALTER TABLE cron_tasks ADD INDEX idx_command (command);
```

- [ ] **Step 5: 创建回滚 `migrations/000008_add_command_to_cron_tasks.down.sql`**

```sql
ALTER TABLE cron_tasks DROP INDEX idx_command;
ALTER TABLE cron_tasks DROP COLUMN command;
```

- [ ] **Step 6: 编译验证**

```bash
go build ./cmd/server/
```

- [ ] **Step 7: 提交**

```bash
git add internal/scheduler/commands.go internal/model/cron_task.go internal/dao/cron_task_dao.go migrations/000008_add_command_to_cron_tasks.up.sql migrations/000008_add_command_to_cron_tasks.down.sql
git commit -m "feat: 添加命令注册表和 cron_tasks.command 字段"
```

---

## Task 2: 后端 — 调度器支持 command 字段

**Files:**
- Modify: `internal/scheduler/scheduler.go:91-144` — execute() 方法

- [ ] **Step 1: 修改 execute() 方法，支持 command 字段解析**

在 `execute()` 方法中，`start := time.Now()` 之后、构造 `body` 之前，加入 command 解析逻辑。完整替换 `execute` 方法：

```go
func (s *Scheduler) execute(task *model.CronTask, trigger string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("任务 %d 执行 panic: %v", task.ID, r)
		}
	}()

	// 防重叠：上次未执行完则跳过本次
	if _, loaded := s.running.LoadOrStore(task.ID, true); loaded {
		s.saveExec(task.ID, trigger, ExecStatusSkipped, 0, "", "上次未执行完，跳过本次", 0)
		return
	}
	defer s.running.Delete(task.ID)

	// 解析执行配置：预定义命令 or 自定义 HTTP
	method := task.Method
	url := task.URL
	headers := task.Headers
	bodyStr := task.Body

	if task.Command != "" {
		if cmd, ok := Commands[task.Command]; ok {
			method = cmd.Method
			url = cmd.URL
			if cmd.Headers != nil {
				if b, err := json.Marshal(cmd.Headers); err == nil {
					headers = string(b)
				}
			}
			bodyStr = cmd.Body
		} else {
			s.saveExec(task.ID, trigger, ExecStatusFailed, 0, "", "未知命令: "+task.Command, 0)
			return
		}
	}

	start := time.Now()
	var body io.Reader
	if method == "POST" {
		body = strings.NewReader(bodyStr)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		s.saveExec(task.ID, trigger, ExecStatusFailed, 0, "", "请求创建失败: "+err.Error(), int(time.Since(start).Milliseconds()))
		return
	}
	// 请求头（JSON 对象，创建时已校验）
	if strings.TrimSpace(headers) != "" {
		var headerMap map[string]string
		if err := json.Unmarshal([]byte(headers), &headerMap); err == nil {
			for k, v := range headerMap {
				req.Header.Set(k, v)
			}
		}
	}
	ctx, cancel := context.WithTimeout(req.Context(), time.Duration(task.Timeout)*time.Second)
	defer cancel()

	resp, err := s.client.Do(req.WithContext(ctx))
	if err != nil {
		s.saveExec(task.ID, trigger, ExecStatusFailed, 0, "", err.Error(), int(time.Since(start).Milliseconds()))
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	text := strings.TrimSpace(string(respBody))
	if len(text) > maxResponseLen {
		text = text[:maxResponseLen]
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.saveExec(task.ID, trigger, ExecStatusFailed, resp.StatusCode, text,
			"HTTP 状态码非 2xx: "+resp.Status, int(time.Since(start).Milliseconds()))
		return
	}
	s.saveExec(task.ID, trigger, ExecStatusSuccess, resp.StatusCode, text, "", int(time.Since(start).Milliseconds()))
}
```

- [ ] **Step 2: 编译验证**

```bash
go build ./cmd/server/
```

- [ ] **Step 3: 提交**

```bash
git add internal/scheduler/scheduler.go
git commit -m "feat: 调度器支持 command 字段，按注册表执行预定义命令"
```

---

## Task 3: 后端 — Service 校验改造 + Controller 新增 Commands/ListAllExecutions

**Files:**
- Modify: `internal/service/cron_task_service.go:22-45,97-107` — validate() + 新增 ListCommands/FindAllExecutions
- Modify: `internal/controller/cron_task_controller.go` — 新增 Commands/ListAllExecutions handler
- Modify: `internal/router/router.go:131-147` — 注册新路由

- [ ] **Step 1: 修改 Service validate() 方法**

替换 `internal/service/cron_task_service.go` 的 `validate` 方法，支持 command 字段校验：

```go
func (s *CronTaskService) validate(t *model.CronTask) error {
	if strings.TrimSpace(t.Name) == "" {
		return errors.New("任务名称不能为空")
	}
	// command 校验
	cmd := strings.TrimSpace(t.Command)
	if cmd != "" && cmd != "_custom" {
		// 预定义命令：必须在注册表中
		if _, ok := scheduler.Commands[cmd]; !ok {
			return errors.New("未知的预定义命令")
		}
	}
	if cmd == "_custom" {
		t.Command = "" // 转为空字符串，走自定义模式
	}
	// 自定义模式下校验 HTTP 字段
	if t.Command == "" {
		if strings.TrimSpace(t.URL) == "" {
			return errors.New("回调地址不能为空")
		}
		if t.Method != "GET" && t.Method != "POST" {
			return errors.New("请求方式仅支持 GET/POST")
		}
		if strings.TrimSpace(t.Headers) != "" {
			var m map[string]string
			if err := json.Unmarshal([]byte(t.Headers), &m); err != nil {
				return errors.New("请求头必须是 JSON 对象")
			}
		}
	}
	if t.Timeout < 1 || t.Timeout > 300 {
		return errors.New("超时时间需在 1-300 秒之间")
	}
	if _, err := scheduler.ParseCron(t.Cron); err != nil {
		return errors.New("cron 表达式不合法（格式：秒 分 时 日 月 周）")
	}
	return nil
}
```

- [ ] **Step 2: Service 新增 ListCommands 和 FindAllExecutions**

在 `internal/service/cron_task_service.go` 末尾追加：

```go
func (s *CronTaskService) ListCommands() []scheduler.CommandOption {
	return scheduler.CommandList()
}

func (s *CronTaskService) FindAllExecutions(taskID uint, status int, startTime, endTime string, page, pageSize int) ([]model.CronTaskExecution, int64, error) {
	return s.execDAO.FindAllPage(taskID, status, startTime, endTime, page, pageSize)
}
```

- [ ] **Step 3: DAO 新增 FindAllPage**

在 `internal/dao/cron_task_dao.go` 的 `CronTaskExecutionDAO` 部分末尾追加：

```go
func (d *CronTaskExecutionDAO) FindAllPage(taskID uint, status int, startTime, endTime string, page, pageSize int) ([]model.CronTaskExecution, int64, error) {
	var list []model.CronTaskExecution
	var total int64
	q := d.db.Model(&model.CronTaskExecution{})
	if taskID > 0 {
		q = q.Where("task_id = ?", taskID)
	}
	if status >= 0 {
		q = q.Where("status = ?", status)
	}
	if startTime != "" {
		q = q.Where("created_at >= ?", startTime)
	}
	if endTime != "" {
		q = q.Where("created_at <= ?", endTime+" 23:59:59")
	}
	q.Count(&total)
	err := q.Offset((page - 1) * pageSize).Limit(pageSize).Order("id DESC").Find(&list).Error
	return list, total, err
}
```

- [ ] **Step 4: Controller 新增 Commands 和 ListAllExecutions handler**

在 `internal/controller/cron_task_controller.go` 末尾追加：

```go
// Commands 获取预定义命令列表
// @Summary 获取预定义命令列表
// @Description 返回所有可用的预定义命令（名称+标识），供前端下拉使用
// @Tags 定时任务
// @Security BearerAuth
// @Produce json
// @Success 200 {object} utils.Response{data=[]scheduler.CommandOption} "成功"
// @Router /cron-tasks/commands [get]
func (ctl *CronTaskController) Commands(c *gin.Context) {
	utils.Success(c, ctl.cronTaskService.ListCommands())
}

// ListAllExecutions 获取全部执行日志（分页+筛选）
// @Summary 获取全部执行日志
// @Description 分页查询所有任务的执行日志，支持按任务/状态/时间范围筛选
// @Tags 定时任务
// @Security BearerAuth
// @Produce json
// @Param task_id query int false "任务 ID"
// @Param status query int false "状态（0成功/1失败/2跳过）"
// @Param start_time query string false "开始日期 YYYY-MM-DD"
// @Param end_time query string false "结束日期 YYYY-MM-DD"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} utils.Response{data=object{list=[]model.CronTaskExecution,total=int}} "成功"
// @Router /cron-tasks/executions [get]
func (ctl *CronTaskController) ListAllExecutions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	taskID, _ := strconv.Atoi(c.DefaultQuery("task_id", "0"))
	status, _ := strconv.Atoi(c.DefaultQuery("status", "-1"))
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")
	list, total, err := ctl.cronTaskService.FindAllExecutions(uint(taskID), status, startTime, endTime, page, pageSize)
	if err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, gin.H{"list": list, "total": total})
}
```

- [ ] **Step 5: 路由注册新接口**

在 `internal/router/router.go` 的定时任务路由块中，在 `authorized.GET("/cron-tasks/:id/executions"` 之前插入：

```go
authorized.GET("/cron-tasks/commands",
    middleware.RequirePerm("cron:list"), middleware.RBAC(menuDAO), taskCtrl.Commands)
authorized.GET("/cron-tasks/executions",
    middleware.RequirePerm("cron:log"), middleware.RBAC(menuDAO), taskCtrl.ListAllExecutions)
```

注意：这两个路由必须在 `/cron-tasks/:id/executions` 之前注册，否则 Gin 会把 `commands` 和 `executions` 当作 `:id` 参数匹配。

- [ ] **Step 6: 编译验证**

```bash
go build ./cmd/server/
```

- [ ] **Step 7: 提交**

```bash
git add internal/service/cron_task_service.go internal/controller/cron_task_controller.go internal/dao/cron_task_dao.go internal/router/router.go
git commit -m "feat: Service/Controller 支持 command 校验、命令列表、全局执行日志查询"
```

---

## Task 4: 后端 — 种子数据更新

**Files:**
- Create: `migrations/000009_update_seed_task_command.up.sql`
- Create: `migrations/000009_update_seed_task_command.down.sql`

- [ ] **Step 1: 创建迁移 `migrations/000009_update_seed_task_command.up.sql`**

```sql
-- 将现有日志清理任务改为使用预定义命令模式
UPDATE cron_tasks
SET command = 'clean_logs',
    url = '',
    method = 'POST',
    headers = '',
    body = ''
WHERE url LIKE '%logs/cleanup%';
```

- [ ] **Step 2: 创建回滚 `migrations/000009_update_seed_task_command.down.sql`**

```sql
-- 恢复为自定义 HTTP 模式
UPDATE cron_tasks
SET command = '',
    url = 'http://go-app:8000/api/logs/cleanup?days=30',
    method = 'POST',
    headers = '{"X-Cleanup-Secret":"__LOG_CLEANUP_SECRET__"}',
    body = ''
WHERE command = 'clean_logs';
```

- [ ] **Step 3: 编译验证**

```bash
go build ./cmd/server/
```

- [ ] **Step 4: 提交**

```bash
git add migrations/000009_update_seed_task_command.up.sql migrations/000009_update_seed_task_command.down.sql
git commit -m "feat: 迁移种子清理任务为 command 模式"
```

---

## Task 5: 后端 — 执行日志菜单迁移

**Files:**
- Create: `migrations/000010_add_task_logs_menu.up.sql`
- Create: `migrations/000010_add_task_logs_menu.down.sql`

- [ ] **Step 1: 创建迁移 `migrations/000010_add_task_logs_menu.up.sql`**

```sql
-- 执行日志菜单（挂在"任务管理"目录下，id=41）
INSERT IGNORE INTO menus (id, parent_id, name, path, component, permission, type, icon, sort, created_at, updated_at)
VALUES (50, 41, '执行日志', 'task-logs', 'task/logs', 'cron:log', 2, 'document', 2, NOW(), NOW());

-- admin 角色绑定
INSERT IGNORE INTO role_menus (role_id, menu_id) VALUES (1, 50);
```

- [ ] **Step 2: 创建回滚 `migrations/000010_add_task_logs_menu.down.sql`**

```sql
DELETE FROM role_menus WHERE menu_id = 50;
DELETE FROM menus WHERE id = 50;
```

- [ ] **Step 3: 编译验证**

```bash
go build ./cmd/server/
```

- [ ] **Step 4: 提交**

```bash
git add migrations/000010_add_task_logs_menu.up.sql migrations/000010_add_task_logs_menu.down.sql
git commit -m "feat: 迁移新增执行日志菜单"
```

---

## Task 6: 前端 — API 层 + 编辑对话框改造 + Cron 快捷按钮

**Files:**
- Modify: `web/src/api/cron.js` — 新增 commands/listAllExecutions 接口
- Modify: `web/src/views/task/index.vue` — 编辑对话框全面改造

- [ ] **Step 1: 新增 API 接口**

在 `web/src/api/cron.js` 末尾追加：

```js
export const getCronCommands = () => request.get('/cron-tasks/commands')
export const getAllCronTaskExecutions = (params) => request.get('/cron-tasks/executions', { params })
```

- [ ] **Step 2: 改造 task/index.vue**

完整替换 `web/src/views/task/index.vue`：

```vue
<template>
  <div>
    <el-card>
      <div slot="header" class="clearfix">
        <span>定时任务</span>
        <el-button type="primary" size="small" style="float:right" @click="openDialog()">新建任务</el-button>
      </div>
      <el-input v-model="keyword" placeholder="搜索任务名称/命令" style="width:250px;margin-bottom:10px" @keyup.enter.native="fetchData" clearable @clear="fetchData"></el-input>
      <el-table :data="list" border>
        <el-table-column prop="id" label="ID" width="60"></el-table-column>
        <el-table-column prop="name" label="任务名称"></el-table-column>
        <el-table-column prop="command" label="命令" width="140">
          <template slot-scope="s">
            <span v-if="s.row.command">{{ commandLabel(s.row.command) }}</span>
            <span v-else style="color:#909399">自定义</span>
          </template>
        </el-table-column>
        <el-table-column prop="cron" label="Cron 表达式" width="160"></el-table-column>
        <el-table-column label="状态" width="90">
          <template slot-scope="s">
            <el-switch v-model="s.row.status" :active-value="1" :inactive-value="0" @change="val => handleStatusChange(s.row, val)"></el-switch>
          </template>
        </el-table-column>
        <el-table-column label="最近执行" width="90">
          <template slot-scope="s">
            <el-tag v-if="s.row.last_exec_status === 0" type="success" size="mini">成功</el-tag>
            <el-tag v-else-if="s.row.last_exec_status === 1" type="danger" size="mini">失败</el-tag>
            <el-tag v-else-if="s.row.last_exec_status === 2" type="warning" size="mini">跳过</el-tag>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220">
          <template slot-scope="s">
            <el-button size="mini" @click="openDialog(s.row)">编辑</el-button>
            <el-button size="mini" type="primary" @click="handleRun(s.row.id)">立即执行</el-button>
            <el-button size="mini" type="danger" @click="handleDelete(s.row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination style="margin-top:15px" @current-change="pageChange" :current-page="page" :page-size="pageSize" :total="total" layout="total,prev,pager,next"></el-pagination>
    </el-card>

    <!-- 新建/编辑对话框 -->
    <el-dialog :title="isEdit ? '编辑任务' : '新建任务'" :visible.sync="dialogVisible" width="600px">
      <el-form :model="form" label-width="110px">
        <el-form-item label="任务名称">
          <el-input v-model="form.name" placeholder="任务名称"></el-input>
        </el-form-item>
        <el-form-item label="命令">
          <el-select v-model="form.command" style="width:100%" @change="onCommandChange">
            <el-option v-for="c in commands" :key="c.name" :label="c.label" :value="c.name"></el-option>
            <el-option label="自定义" value="_custom"></el-option>
          </el-select>
        </el-form-item>
        <!-- 自定义模式字段 -->
        <template v-if="form.command === '_custom' || form.command === ''">
          <el-form-item label="回调地址">
            <el-input v-model="form.url" placeholder="http://example.com/callback"></el-input>
          </el-form-item>
          <el-form-item label="请求方式">
            <el-select v-model="form.method" style="width:100%">
              <el-option label="GET" value="GET"></el-option>
              <el-option label="POST" value="POST"></el-option>
            </el-select>
          </el-form-item>
          <el-form-item label="请求头">
            <el-input v-model="form.headers" type="textarea" :rows="2" placeholder='JSON 对象，如 {"Content-Type": "application/json"}'></el-input>
          </el-form-item>
          <el-form-item v-if="form.method === 'POST'" label="请求体">
            <el-input v-model="form.body" type="textarea" :rows="3" placeholder="POST 请求体"></el-input>
          </el-form-item>
          <el-form-item label="超时（秒）">
            <el-input-number v-model="form.timeout" :min="1" :max="300"></el-input-number>
          </el-form-item>
        </template>
        <!-- Cron 表达式 + 快捷按钮 -->
        <el-form-item label="Cron 表达式">
          <el-input v-model="form.cron" placeholder="秒 分 时 日 月 周，如 0 0/5 * * * ?"></el-input>
        </el-form-item>
        <el-form-item label="快捷设置">
          <el-button-group>
            <el-button size="small" @click="setCron('minute')">每分钟</el-button>
            <el-button size="small" @click="setCron('hourly')">每小时</el-button>
            <el-button size="small" @click="setCron('daily')">每天</el-button>
            <el-button size="small" @click="setCron('weekly')">每周</el-button>
            <el-button size="small" @click="setCron('monthly')">每月</el-button>
          </el-button-group>
          <span style="margin-left:10px;color:#909399;font-size:12px">{{ cronHint }}</span>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark"></el-input>
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit">确定</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import { getCronTasks, addCronTask, updateCronTask, deleteCronTask, updateCronTaskStatus, runCronTask, getCronCommands } from '@/api/cron'
export default {
  data() {
    return {
      list: [], page: 1, pageSize: 10, total: 0, keyword: '',
      dialogVisible: false, isEdit: false,
      form: { name: '', command: '', url: '', method: 'GET', headers: '', body: '', cron: '', timeout: 30, remark: '' },
      commands: [], cronHour: 3
    }
  },
  computed: {
    commandLabel() {
      return (name) => {
        const c = this.commands.find(x => x.name === name)
        return c ? c.label : name
      }
    },
    cronHint() {
      const c = this.form.cron
      if (!c) return ''
      const parts = c.split(' ')
      if (parts.length < 6) return ''
      const h = parts[2], m = parts[1], dom = parts[3], mon = parts[4], dow = parts[5]
      if (m === '*' && h === '*') return '每分钟执行'
      if (m === '0' && h === '*') return '每小时整点执行'
      if (dom === '*' && mon === '*' && dow === '*') return `每天 ${h.padStart(2, '0')}:${m.padStart(2, '0')} 执行`
      if (dom === '?' && mon === '*' && dow !== '*') {
        const dayMap = { '1': '一', '2': '二', '3': '三', '4': '四', '5': '五', '6': '六', '0': '日' }
        return `每周${dayMap[dow] || dow} ${h.padStart(2, '0')}:${m.padStart(2, '0')} 执行`
      }
      if (dom !== '*' && mon === '*' && dow === '?') return `每月 ${dom} 号 ${h.padStart(2, '0')}:${m.padStart(2, '0')} 执行`
      return ''
    }
  },
  created() {
    this.fetchData()
    this.fetchCommands()
  },
  methods: {
    async fetchData() {
      const res = await getCronTasks({ page: this.page, page_size: this.pageSize, keyword: this.keyword })
      this.list = res.data.list; this.total = res.data.total
    },
    async fetchCommands() {
      const res = await getCronCommands()
      this.commands = res.data
    },
    pageChange(p) { this.page = p; this.fetchData() },
    openDialog(row) {
      if (row) {
        this.isEdit = true
        this.form = { ...row, command: row.command || '_custom' }
      } else {
        this.isEdit = false
        this.form = { name: '', command: '', url: '', method: 'GET', headers: '', body: '', cron: '', timeout: 30, remark: '' }
      }
      this.dialogVisible = true
    },
    onCommandChange(val) {
      if (val === '_custom' || val === '') {
        // 自定义模式：保留现有字段
      } else {
        // 预定义命令：清空 HTTP 字段
        this.form.url = ''
        this.form.method = 'GET'
        this.form.headers = ''
        this.form.body = ''
      }
    },
    setCron(type) {
      const h = String(this.cronHour).padStart(2, '0')
      const map = {
        minute: '0 * * * * *',
        hourly: `0 0 * * * *`,
        daily: `0 0 ${h} * * *`,
        weekly: `0 0 ${h} ? * 1`,
        monthly: `0 0 ${h} 1 * *`
      }
      this.form.cron = map[type]
    },
    handleStatusChange(row, val) {
      updateCronTaskStatus(row.id, { status: val }).catch(() => {
        this.$message.error('状态切换失败')
        row.status = val === 1 ? 0 : 1
      })
    },
    async handleSubmit() {
      if (!this.form.name) { this.$message.warning('任务名称不能为空'); return }
      if (!this.form.command && !this.form.url) { this.$message.warning('回调地址不能为空'); return }
      if (!this.form.cron) { this.$message.warning('cron 表达式不能为空'); return }
      if (this.isEdit) { await updateCronTask(this.form.id, this.form) } else { await addCronTask(this.form) }
      this.dialogVisible = false; this.fetchData(); this.$message.success(this.isEdit ? '编辑成功' : '新增成功')
    },
    async handleRun(id) {
      await this.$confirm('确认立即执行该任务?', '提示', { type: 'warning' })
      await runCronTask(id); this.$message.success('已触发执行')
    },
    async handleDelete(id) {
      await this.$confirm('确认删除该任务?', '提示', { type: 'warning' })
      await deleteCronTask(id); this.fetchData(); this.$message.success('删除成功')
    }
  }
}
</script>
```

- [ ] **Step 3: 前端编译验证**

```bash
cd web && npm run build
```

- [ ] **Step 4: 提交**

```bash
git add web/src/api/cron.js web/src/views/task/index.vue
git commit -m "feat: 前端编辑对话框改造——命令下拉、Cron 快捷按钮"
```

---

## Task 7: 前端 — 执行日志独立页面 + 路由注册

**Files:**
- Create: `web/src/views/task/logs.vue`
- Modify: `web/src/store/modules/permission.js:10` — 注册路由映射

- [ ] **Step 1: 创建执行日志页面 `web/src/views/task/logs.vue`**

```vue
<template>
  <div>
    <el-card>
      <div slot="header"><span>执行日志</span></div>
      <!-- 筛选区 -->
      <el-form :inline="true" style="margin-bottom:15px">
        <el-form-item label="任务名称">
          <el-select v-model="filter.taskId" placeholder="全部" clearable style="width:180px" @change="handleSearch">
            <el-option v-for="t in taskOptions" :key="t.id" :label="t.name" :value="t.id"></el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="filter.status" placeholder="全部" clearable style="width:120px" @change="handleSearch">
            <el-option label="成功" :value="0"></el-option>
            <el-option label="失败" :value="1"></el-option>
            <el-option label="跳过" :value="2"></el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="日期范围">
          <el-date-picker v-model="filter.dateRange" type="daterange" range-separator="-" start-placeholder="开始日期" end-placeholder="结束日期" value-format="yyyy-MM-dd" style="width:260px" @change="handleSearch"></el-date-picker>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSearch">搜索</el-button>
          <el-button @click="handleReset">重置</el-button>
        </el-form-item>
      </el-form>
      <!-- 表格 -->
      <el-table :data="list" border>
        <el-table-column prop="id" label="ID" width="70"></el-table-column>
        <el-table-column label="任务名称" width="160">
          <template slot-scope="s">
            {{ taskName(s.row.task_id) }}
          </template>
        </el-table-column>
        <el-table-column label="命令" width="140">
          <template slot-scope="s">
            {{ taskCommand(s.row.task_id) || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template slot-scope="s">
            <el-tag v-if="s.row.status === 0" type="success" size="mini">成功</el-tag>
            <el-tag v-else-if="s.row.status === 1" type="danger" size="mini">失败</el-tag>
            <el-tag v-else-if="s.row.status === 2" type="warning" size="mini">跳过</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="耗时(秒)" width="100">
          <template slot-scope="s">
            {{ (s.row.duration_ms / 1000).toFixed(1) }}
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="执行时间" width="170"></el-table-column>
        <el-table-column label="操作" width="150">
          <template slot-scope="s">
            <el-button size="mini" type="info" @click="viewOutput(s.row)">查看输出</el-button>
            <el-button size="mini" type="danger" @click="handleDelete(s.row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination style="margin-top:15px" @current-change="pageChange" :current-page="page" :page-size="pageSize" :total="total" layout="total,prev,pager,next"></el-pagination>
    </el-card>

    <!-- 查看输出弹窗 -->
    <el-dialog title="执行输出" :visible.sync="outputVisible" width="700px">
      <pre style="max-height:400px;overflow:auto;background:#f5f7fa;padding:15px;border-radius:4px;font-size:13px;white-space:pre-wrap;word-break:break-all">{{ outputContent }}</pre>
      <span slot="footer"><el-button @click="outputVisible = false">关闭</el-button></span>
    </el-dialog>
  </div>
</template>

<script>
import { getAllCronTaskExecutions, getCronTasks } from '@/api/cron'
export default {
  data() {
    return {
      list: [], page: 1, pageSize: 10, total: 0,
      filter: { taskId: '', status: '', dateRange: null },
      taskOptions: [],
      outputVisible: false, outputContent: ''
    }
  },
  created() {
    this.fetchTasks()
    // 支持从任务列表页跳转过来带 task_id 参数
    if (this.$route.query.task_id) {
      this.filter.taskId = Number(this.$route.query.task_id)
    }
    this.fetchData()
  },
  methods: {
    async fetchTasks() {
      const res = await getCronTasks({ page: 1, page_size: 100 })
      this.taskOptions = res.data.list
    },
    async fetchData() {
      const params = { page: this.page, page_size: this.pageSize }
      if (this.filter.taskId) params.task_id = this.filter.taskId
      if (this.filter.status !== '' && this.filter.status !== null) params.status = this.filter.status
      if (this.filter.dateRange && this.filter.dateRange.length === 2) {
        params.start_time = this.filter.dateRange[0]
        params.end_time = this.filter.dateRange[1]
      }
      const res = await getAllCronTaskExecutions(params)
      this.list = res.data.list; this.total = res.data.total
    },
    taskName(taskId) {
      const t = this.taskOptions.find(x => x.id === taskId)
      return t ? t.name : taskId
    },
    taskCommand(taskId) {
      const t = this.taskOptions.find(x => x.id === taskId)
      return t ? t.command : ''
    },
    handleSearch() { this.page = 1; this.fetchData() },
    handleReset() { this.filter = { taskId: '', status: '', dateRange: null }; this.handleSearch() },
    pageChange(p) { this.page = p; this.fetchData() },
    viewOutput(row) {
      this.outputContent = row.response || row.error_msg || '(无输出)'
      this.outputVisible = true
    },
    async handleDelete(id) {
      await this.$confirm('确认删除该日志?', '提示', { type: 'warning' })
      // 复用已有删除接口（如有）或直接调用
      const { default: request } = await import('@/api/request')
      await request.delete('/cron-tasks/executions/' + id)
      this.fetchData(); this.$message.success('删除成功')
    }
  }
}
</script>
```

- [ ] **Step 2: 注册路由映射**

修改 `web/src/store/modules/permission.js`，在 `componentMap` 中追加：

```js
'/system/task-logs': () => import('@/views/task/logs.vue')
```

- [ ] **Step 3: 前端编译验证**

```bash
cd web && npm run build
```

- [ ] **Step 4: 提交**

```bash
git add web/src/views/task/logs.vue web/src/store/modules/permission.js
git commit -m "feat: 执行日志独立页面，支持任务/状态/日期筛选"
```

---

## Task 8: 整体验证

- [ ] **Step 1: 后端编译**

```bash
go build ./cmd/server/
```

- [ ] **Step 2: 前端编译**

```bash
cd web && npm run build
```

- [ ] **Step 3: Docker 构建验证**

```bash
docker compose up -d --build go-app
```

- [ ] **Step 4: Swagger 文档重新生成**

```bash
swag init -g cmd/server/main.go
```

- [ ] **Step 5: 手工回归测试**

1. 登录后台 → 定时任务页面，确认任务列表显示"命令"列
2. 新建任务 → 选择预定义命令 → 确认 URL/Method/Headers/Body 字段隐藏
3. 新建任务 → 选择"自定义" → 确认全部 HTTP 字段展开
4. Cron 快捷按钮点击验证
5. 执行日志页面 → 筛选、查看输出
6. 编辑已有种子任务（日志清理）→ 确认显示为预定义命令模式

- [ ] **Step 6: 提交 Swagger 文档**

```bash
git add docs/
git commit -m "docs: 重新生成 Swagger（定时任务命令列表、全局执行日志接口）"
```
