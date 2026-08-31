# 日志清理自动策略 设计文档

## 1. 概述

### 1.1 目标

实现日志清理自动策略，支持通过系统界面配置保留天数和清理范围，定时任务和手动 API 自动读取配置执行清理。

### 1.2 背景

现有日志清理功能存在以下问题：
- 保留天数硬编码为 30 天（`cmd/server/main.go:130-132`）
- 清理范围固定清理所有日志（操作日志 + 登录日志）
- 无法通过界面配置，修改需改代码重新部署

### 1.3 成功标准

- 管理员可通过「日志设置」页面配置保留天数和清理范围
- 定时任务 `clean_logs` 每天自动读取配置执行清理
- 手动 API `POST /logs/cleanup` 也读取配置（API 传参优先，向后兼容）
- 配置即时生效，无需重启服务

## 2. 配置项设计

### 2.1 配置项

| setting_key | 类型 | 默认值 | 说明 |
|---|---|---|---|
| `log_cleanup_days` | int（字符串存储） | `180` | 保留天数，删除 N 天前的日志 |
| `log_cleanup_scope` | JSON array（字符串存储） | `["operation","login"]` | 清理范围：operation=操作日志，login=登录日志 |

### 2.2 存储

使用现有 `system_settings` 表，通过 `INSERT IGNORE` 种子数据保证幂等。

## 3. 后端设计

### 3.1 新增 API

| 端点 | 方法 | 权限 | 说明 |
|---|---|---|---|
| `/api/log-settings` | GET | `log:setting` | 获取日志清理配置 |
| `/api/log-settings` | PUT | `log:setting` | 保存日志清理配置 |

### 3.2 请求/响应

```json
// GET /api/log-settings
{
  "code": 200,
  "data": {
    "days": 180,
    "scope": ["operation", "login"]
  }
}

// PUT /api/log-settings
// 请求体
{
  "days": 180,
  "scope": ["operation", "login"]
}
// 响应
{
  "code": 200,
  "message": "保存成功"
}
```

### 3.3 Controller

新增 `LogSettingController`，位于 `internal/controller/log_setting_controller.go`：
- `Get(c *gin.Context)` - 读取配置并返回
- `Update(c *gin.Context)` - 保存配置到 system_settings

### 3.4 路由注册

在 `internal/router/router.go` 中，`authorized` group 下添加：
```go
// 日志设置
authorized.GET("/log-settings",
    middleware.RequirePerm("log:setting"), middleware.RBAC(menuDAO), logSettingCtrl.Get)
authorized.PUT("/log-settings",
    middleware.RequirePerm("log:setting"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), logSettingCtrl.Update)
```

### 3.5 定时任务集成

修改 `cmd/server/main.go` 中 `clean_logs` Handler：

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

### 3.6 手动 API 修改

修改 `LogController.Cleanup` 方法，增加从配置读取的逻辑：
- `days` 参数无效时，读取 `log_cleanup_days` 配置
- `scope` 参数无效时，读取 `log_cleanup_scope` 配置
- 向后兼容：API 传参优先于配置

## 4. 前端设计

### 4.1 页面位置

日志管理（目录）→ 日志设置（子菜单）

### 4.2 菜单结构

```
日志管理（目录，已有）
├── 操作日志（菜单页，已有，path=/system/log）
├── 登录日志（菜单页，已有，path=/system/login-log）
└── 日志设置（菜单页，新增）
    - path: /system/log-setting
    - permission: log:setting
    - icon: el-icon-setting
    - sort_order: 99
```

### 4.3 路由映射

在 `web/src/store/modules/permission.js` 的 `componentMap` 中添加：
```javascript
'/system/log-setting': () => import('@/views/log/setting.vue')
```

### 4.4 页面设计

```
┌─────────────────────────────────────────────┐
│ 日志清理设置                                │
├─────────────────────────────────────────────┤
│                                             │
│ 保留天数:  [____180____] 天                 │
│            提示：删除 N 天前的日志           │
│                                             │
│ 清理范围:  ☑ 操作日志                       │
│            ☑ 登录日志                       │
│                                             │
│ [保存]                                      │
│                                             │
└─────────────────────────────────────────────┘
```

### 4.5 页面组件

新建 `web/src/views/log/setting.vue`：
- 表单：`el-input-number`（保留天数）+ `el-checkbox-group`（清理范围）
- 加载：调用 `getSettings()` 读取配置
- 保存：调用 `updateSettings()` 写入配置

### 4.6 API 调用

复用现有 `web/src/api/setting.js` 中的 `getSettings` 和 `updateSettings`。

## 5. 数据库迁移

### 5.1 迁移文件

文件：`migrations/000021_add_log_cleanup_config.up.sql`

```sql
-- 日志清理策略配置
INSERT IGNORE INTO system_settings (setting_key, setting_value) VALUES
('log_cleanup_days', '180'),
('log_cleanup_scope', '["operation","login"]');

-- 日志设置菜单（父级：日志管理）
INSERT INTO menus (name, path, parent_id, type, permission, icon, sort_order)
SELECT '日志设置', '/system/log-setting', id, 2, 'log:setting', 'el-icon-setting', 99
FROM menus WHERE path = '/system/log' AND type = 1;
```

### 5.2 回滚脚本

文件：`migrations/000021_add_log_cleanup_config.down.sql`

```sql
DELETE FROM menus WHERE path = '/system/log-setting';
DELETE FROM system_settings WHERE setting_key IN ('log_cleanup_days', 'log_cleanup_scope');
```

## 6. 权限设计

### 6.1 权限点

新增权限点：`log:setting`（日志设置）

### 6.2 权限分配

- 管理员角色默认拥有 `log:setting` 权限
- 需在迁移中为管理员角色添加此权限（或手动在菜单管理中配置）

## 7. 错误处理

### 7.1 配置读取失败

- 定时任务：fallback 到默认值（30 天，全部日志）
- API：返回错误响应，提示配置读取失败

### 7.2 配置保存失败

- 前端显示错误提示
- 不影响现有配置

### 7.3 配置格式异常

- `log_cleanup_days` 非数字：使用默认值 180
- `log_cleanup_scope` 非法 JSON：使用默认值 `["operation","login"]`
- 清理范围为空数组：使用默认值

## 8. 测试策略

### 8.1 单元测试

- Controller 层：测试配置读取和保存的正确性
- Service 层：测试配置解析和 fallback 逻辑

### 8.2 集成测试

- 定时任务：模拟不同配置，验证清理行为
- API：测试参数优先级（API 传参 vs 配置）

### 8.3 前端测试

- 页面加载：配置正确显示
- 保存功能：配置正确持久化
- 边界情况：输入非法值的处理

## 9. 部署计划

### 9.1 迁移执行

1. 创建迁移文件
2. Docker 重建 go-app 容器
3. 验证迁移执行成功

### 9.2 功能验证

1. 访问日志设置页面
2. 修改配置并保存
3. 触发定时任务或手动 API，验证清理行为

### 9.3 回滚方案

如需回滚：
1. 执行 down 迁移
2. 重建 go-app 容器
3. 恢复原有清理逻辑

## 10. 后续扩展

### 10.1 更多日志类型

如需支持其他日志类型（如安全日志、审计日志），可扩展 `log_cleanup_scope` 的可选值。

### 10.2 清理统计

可添加清理统计功能，记录每次清理的日志数量和耗时。

### 10.3 清理策略

可扩展更复杂的清理策略，如按模块设置不同的保留天数。
