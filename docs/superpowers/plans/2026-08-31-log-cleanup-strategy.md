# 日志清理自动策略 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现日志清理自动策略，支持通过系统界面配置保留天数和清理范围，定时任务和手动 API 自动读取配置执行清理

**Architecture:** 新增独立 API 端点 `/api/log-settings` 管理日志清理配置，底层存储在 `system_settings` 表；前端新增日志设置页面，集成到日志管理菜单下；定时任务和手动 API 读取配置决定清理参数

**Tech Stack:** Go 1.18 + Gin + GORM（后端），Vue 2 + Element UI（前端），MySQL 5.7（system_settings 表）

---

## 现有代码参考

- 定时任务命令注册：`internal/scheduler/commands.go:18-40`
- clean_logs Handler 注入：`cmd/server/main.go:126-145`
- 系统配置 Service：`internal/service/system_setting_service.go`
- 系统配置 API：`web/src/api/setting.js`（`GET/PUT /settings`）
- 前端路由映射：`web/src/store/modules/permission.js:4-18`

---

### Task 1: 创建数据库迁移

**Files:**
- Modify: `migrations/000021_add_log_cleanup_config.up.sql`
- Modify: `migrations/000021_add_log_cleanup_config.down.sql`

- [ ] **Step 1: 更新 up 迁移文件**

```sql
-- migrations/000021_add_log_cleanup_config.up.sql

-- 日志清理策略配置
INSERT IGNORE INTO system_settings (setting_key, setting_value) VALUES
('log_cleanup_days', '180'),
('log_cleanup_scope', '["operation","login"]');

-- 日志设置菜单（父级：日志管理）
INSERT INTO menus (name, path, parent_id, type, permission, icon, sort_order)
SELECT '日志设置', '/system/log-setting', id, 2, 'log:setting', 'el-icon-setting', 99
FROM menus WHERE path = '/system/log' AND type = 1;
```

- [ ] **Step 2: 更新 down 迁移文件**

```sql
-- migrations/000021_add_log_cleanup_config.down.sql

DELETE FROM menus WHERE path = '/system/log-setting';
DELETE FROM system_settings WHERE setting_key IN ('log_cleanup_days', 'log_cleanup_scope');
```

- [ ] **Step 3: Commit**

```bash
git add migrations/000021_add_log_cleanup_config.*
git commit -m "feat: 添加日志清理策略配置项和菜单种子"
```

---

### Task 2: 创建 LogSettingController

**Files:**
- Create: `internal/controller/log_setting_controller.go`

- [ ] **Step 1: 创建 LogSettingController**

```go
// internal/controller/log_setting_controller.go

package controller

import (
	"encoding/json"
	"net/http"
	"strconv"

	"ginproject/internal/service"
	"ginproject/internal/utils"

	"github.com/gin-gonic/gin"
)

type LogSettingController struct {
	settingService *service.SystemSettingService
}

func NewLogSettingController(settingService *service.SystemSettingService) *LogSettingController {
	return &LogSettingController{settingService: settingService}
}

// Get 获取日志清理配置
// @Summary 获取日志清理配置
// @Tags 日志设置
// @Produce json
// @Success 200 {object} utils.Response{data=object{days=int,scope=[]string}} "成功"
// @Router /log-settings [get]
func (ctl *LogSettingController) Get(c *gin.Context) {
	cfg, err := ctl.settingService.GetAll()
	if err != nil {
		utils.Error(c, 500, "配置读取失败: "+err.Error())
		return
	}

	// 保留天数，默认 180
	days := 180
	if v, ok := cfg["log_cleanup_days"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 3650 {
			days = n
		}
	}

	// 清理范围，默认全部
	scope := []string{"operation", "login"}
	if v, ok := cfg["log_cleanup_scope"]; ok && v != "" {
		var parsed []string
		if json.Unmarshal([]byte(v), &parsed) == nil && len(parsed) > 0 {
			scope = parsed
		}
	}

	utils.Success(c, gin.H{
		"days":  days,
		"scope": scope,
	})
}

// Update 保存日志清理配置
// @Summary 保存日志清理配置
// @Tags 日志设置
// @Accept json
// @Produce json
// @Param body body object{days=int,scope=[]string} true "配置"
// @Success 200 {object} utils.Response "成功"
// @Failure 200 {object} utils.Response "参数非法"
// @Router /log-settings [put]
func (ctl *LogSettingController) Update(c *gin.Context) {
	var req struct {
		Days  int      `json:"days"`
		Scope []string `json:"scope"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数格式错误")
		return
	}

	// 校验保留天数
	if req.Days < 1 || req.Days > 3650 {
		utils.Error(c, 400, "保留天数必须在 1~3650 之间")
		return
	}

	// 校验清理范围
	if len(req.Scope) == 0 {
		utils.Error(c, 400, "清理范围不能为空")
		return
	}
	validScopes := map[string]bool{"operation": true, "login": true}
	for _, s := range req.Scope {
		if !validScopes[s] {
			utils.Error(c, 400, "无效的清理范围: "+s)
			return
		}
	}

	// 保存配置
	scopeJSON, _ := json.Marshal(req.Scope)
	settings := map[string]string{
		"log_cleanup_days":  strconv.Itoa(req.Days),
		"log_cleanup_scope": string(scopeJSON),
	}
	if err := ctl.settingService.Save(settings); err != nil {
		utils.Error(c, 500, "保存失败: "+err.Error())
		return
	}

	utils.Success(c, nil)
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/controller/log_setting_controller.go
git commit -m "feat: 新增 LogSettingController"
```

---

### Task 3: 注册路由和依赖注入

**Files:**
- Modify: `internal/router/router.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: 在 router.go 添加路由**

在 `internal/router/router.go` 的 `Setup` 函数签名中添加参数：
```go
func Setup(
	cfg *config.Config,
	authCtrl *controller.AuthController,
	userCtrl *controller.UserController,
	roleCtrl *controller.RoleController,
	menuCtrl *controller.MenuController,
	logCtrl *controller.LogController,
	loginLogCtrl *controller.LoginLogController,
	wsCtrl *controller.WSController,
	uploadCtrl *controller.UploadController,
	authService *service.AuthService,
	userDAO *dao.UserDAO,
	menuDAO *dao.MenuDAO,
	logDAO *dao.LogDAO,
	logRepo *es.LogRepo,
	dictTypeCtrl *controller.DictTypeController,
	dictDataCtrl *controller.DictDataController,
	taskCtrl *controller.CronTaskController,
	dbBackupCtrl *controller.DbBackupController,
	fileCtrl *controller.FileController,
	dashboardCtrl *controller.DashboardController,
	healthCtrl *controller.HealthController,
	settingCtrl *controller.SystemSettingController,
	notificationCtrl *controller.NotificationController,
	logSettingCtrl *controller.LogSettingController,  // 新增
) *gin.Engine {
```

在 `authorized` group 中添加路由（约第 200 行后）：
```go
		// 日志设置
		authorized.GET("/log-settings",
			middleware.RequirePerm("log:setting"), middleware.RBAC(menuDAO), logSettingCtrl.Get)
		authorized.PUT("/log-settings",
			middleware.RequirePerm("log:setting"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), logSettingCtrl.Update)
```

- [ ] **Step 2: 在 main.go 注入 Controller**

在 `cmd/server/main.go` 中创建 Controller 实例（约第 197 行附近）：
```go
	logSettingCtrl := controller.NewLogSettingController(systemSettingService)
```

在调用 `Setup` 时添加参数（约第 220 行附近）：
```go
	r := router.Setup(
		cfg,
		authCtrl,
		userCtrl,
		roleCtrl,
		menuCtrl,
		logCtrl,
		loginLogCtrl,
		wsCtrl,
		uploadCtrl,
		authService,
		userDAO,
		menuDAO,
		logDAO,
		logRepo,
		dictTypeCtrl,
		dictDataCtrl,
		taskCtrl,
		dbBackupCtrl,
		fileCtrl,
		dashboardCtrl,
		healthCtrl,
		settingCtrl,
		notificationCtrl,
		logSettingCtrl,  // 新增
	)
```

- [ ] **Step 3: Commit**

```bash
git add internal/router/router.go cmd/server/main.go
git commit -m "feat: 注册日志设置路由和依赖注入"
```

---

### Task 4: 修改 clean_logs Handler

**Files:**
- Modify: `cmd/server/main.go:126-145`

- [ ] **Step 1: 修改 clean_logs Handler**

在 `cmd/server/main.go` 中，找到 `clean_logs` Handler（约第 126 行），替换为：

```go
	scheduler.Commands["clean_logs"] = scheduler.CommandDef{
		Name:  "clean_logs",
		Label: "清理过期日志",
		Handler: func(days int) (scheduler.CommandResult, error) {
			// 1. 保留天数：入参无效时读取配置，再无效 fallback 30
			if days < 1 || days > 3650 {
				cfg, _ := systemSettingService.GetAll()
				if v, ok := cfg["log_cleanup_days"]; ok {
					if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 3650 {
						days = n
					}
				}
				if days < 1 || days > 3650 {
					days = 30
				}
			}

			// 2. 清理范围：读取配置，默认全部
			scope := []string{"operation", "login"}
			cfg, _ := systemSettingService.GetAll()
			if v, ok := cfg["log_cleanup_scope"]; ok && v != "" {
				var parsed []string
				if json.Unmarshal([]byte(v), &parsed) == nil && len(parsed) > 0 {
					scope = parsed
				}
			}

			// 3. 按 scope 执行清理
			var msgs []string
			for _, s := range scope {
				switch s {
				case "operation":
					n, err := logService.Cleanup(days)
					if err != nil {
						return scheduler.CommandResult{}, err
					}
					msgs = append(msgs, fmt.Sprintf("操作日志 %d 条", n))
				case "login":
					n, err := loginLogService.Cleanup(days)
					if err != nil {
						return scheduler.CommandResult{}, err
					}
					msgs = append(msgs, fmt.Sprintf("登录日志 %d 条", n))
				}
			}
			return scheduler.CommandResult{
				Message: "清理完成：" + strings.Join(msgs, "，"),
			}, nil
		},
	}
```

- [ ] **Step 2: 确保导入包含必要包**

检查 `cmd/server/main.go` 导入部分包含：
```go
import (
	"encoding/json"
	"strconv"
	"strings"
	// ... 其他导入
)
```

- [ ] **Step 3: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: clean_logs 定时任务读取系统配置决定清理参数"
```

---

### Task 5: 创建前端 API 文件

**Files:**
- Create: `web/src/api/log-setting.js`

- [ ] **Step 1: 创建 API 文件**

```javascript
// web/src/api/log-setting.js
import request from './request'

export const getLogSettings = () => request.get('/log-settings')
export const updateLogSettings = (data) => request.put('/log-settings', data)
```

- [ ] **Step 2: Commit**

```bash
git add web/src/api/log-setting.js
git commit -m "feat: 新增日志设置 API 文件"
```

---

### Task 6: 创建前端页面

**Files:**
- Create: `web/src/views/log/setting.vue`

- [ ] **Step 1: 创建日志设置页面**

```vue
<!-- web/src/views/log/setting.vue -->
<template>
  <div>
    <el-card>
      <div slot="header"><span>日志清理设置</span></div>
      <el-form ref="form" :model="form" label-width="100px" style="max-width:500px" v-loading="loading">
        <el-form-item label="保留天数" prop="days">
          <el-input-number v-model="form.days" :min="1" :max="3650" placeholder="保留天数"></el-input-number>
          <span style="margin-left:8px;color:#909399;font-size:12px">删除 N 天前的日志</span>
        </el-form-item>
        <el-form-item label="清理范围" prop="scope">
          <el-checkbox-group v-model="form.scope">
            <el-checkbox label="operation">操作日志</el-checkbox>
            <el-checkbox label="login">登录日志</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script>
import { getLogSettings, updateLogSettings } from '@/api/log-setting'

export default {
  data() {
    return {
      form: { days: 180, scope: ['operation', 'login'] },
      loading: false,
      saving: false
    }
  },
  created() {
    this.fetchSettings()
  },
  methods: {
    async fetchSettings() {
      this.loading = true
      try {
        const res = await getLogSettings()
        if (res.code === 200) {
          this.form.days = res.data.days || 180
          this.form.scope = res.data.scope || ['operation', 'login']
        }
      } finally {
        this.loading = false
      }
    },
    async handleSave() {
      if (this.form.scope.length === 0) {
        this.$message.warning('请至少选择一项清理范围')
        return
      }
      this.saving = true
      try {
        const res = await updateLogSettings({
          days: this.form.days,
          scope: this.form.scope
        })
        if (res.code === 200) {
          this.$message.success('保存成功')
        }
      } finally {
        this.saving = false
      }
    }
  }
}
</script>
```

- [ ] **Step 2: Commit**

```bash
git add web/src/views/log/setting.vue
git commit -m "feat: 新增日志设置页面"
```

---

### Task 7: 添加前端路由映射

**Files:**
- Modify: `web/src/store/modules/permission.js:4-18`

- [ ] **Step 1: 在 componentMap 中添加路由映射**

在 `web/src/store/modules/permission.js` 的 `componentMap` 中添加：

```javascript
const componentMap = {
  '/system/user': () => import('@/views/user/index.vue'),
  '/system/role': () => import('@/views/role/index.vue'),
  '/system/menu': () => import('@/views/menu/index.vue'),
  '/system/log': () => import('@/views/log/index.vue'),
  '/system/login-log': () => import('@/views/loginlog/index.vue'),
  '/system/log-setting': () => import('@/views/log/setting.vue'),  // 新增
  '/system/dict-type': () => import('@/views/dict/index.vue'),
  '/system/task': () => import('@/views/task/index.vue'),
  '/system/task-logs': () => import('@/views/task/logs.vue'),
  '/system/backup': () => import('@/views/backup/index.vue'),
  '/system/file': () => import('@/views/file/index.vue'),
  '/system/setting': () => import('@/views/setting/index.vue'),
  '/system/notification': () => import('@/views/notification/index.vue'),
  '/system/notification-send': () => import('@/views/notification/send.vue')
}
```

- [ ] **Step 2: Commit**

```bash
git add web/src/store/modules/permission.js
git commit -m "feat: 添加日志设置路由映射"
```

---

### Task 8: Docker 重建验证

**Files:**
- 无（验证步骤）

- [ ] **Step 1: 重建 go-app 容器**

```bash
docker compose up -d --build go-app
```

- [ ] **Step 2: 验证迁移执行**

```bash
docker compose logs go-app | grep -i "migration\|000021"
```

Expected: 看到迁移执行成功的日志

- [ ] **Step 3: 验证配置 API**

```bash
curl -s http://localhost:8000/api/log-settings | python -m json.tool
```

Expected: 返回 `{"code":200,"data":{"days":180,"scope":["operation","login"]}}`

- [ ] **Step 4: 验证定时任务配置读取**

触发定时任务执行（或等待凌晨 3 点自动执行），检查执行日志是否显示配置的保留天数。

- [ ] **Step 5: 重建 nginx 容器**

```bash
docker compose up -d --build nginx
```

- [ ] **Step 6: 浏览器验证**

1. 打开 `https://localhost:8443`
2. 登录（admin / 123456）
3. 进入「日志管理」→「日志设置」
4. 修改保留天数和清理范围，保存
5. 刷新页面确认配置持久化

- [ ] **Step 7: 验证通过后最终 Commit**

```bash
git status  # 确认无意外改动
```
