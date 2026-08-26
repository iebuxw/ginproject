# 日志定时清理 设计文档

日期：2026-08-26

## 背景

操作日志（`operation_logs`）与登录日志（`login_logs`）持续增长，无清理机制。需要定时清理旧日志，防止数据量膨胀。

## 需求

- 清理范围：`operation_logs` + `login_logs` 两张表
- 保留策略：保留最近 N 天，N 由接口参数控制（定时任务配置时自定）
- 触发方式：复用现有定时任务模块（HTTP 回调式），在任务管理页配置一条任务
- ES 双写：同步清理 ES 中的旧操作日志文档（`delete_by_query`），ES 不可用时降级仅清 MySQL
- 为 `created_at` 加索引，保证按时间删除不产生全表扫描

## 方案

### 1. 新增清理端点（后端）

```
POST /api/logs/cleanup?secret=xxx&days=30&scope=all
```

- **公开路由**（不走 JWT）：调度器发请求不带 token，防滥用靠 `secret` 参数与 `.env` 中 `LOG_CLEANUP_SECRET` 比对
- **参数**：
  - `secret`：必填，与配置比对，不一致返回 403
  - `days`：保留天数，校验为 1~3650 的整数，缺省 30（实现为 `DefaultQuery("days", "30")`）
  - `scope`：可选，`operation` / `login` / `all`，默认 `all`
- **流程**：
  1. 校验密钥 → 计算截止时间 `now - days`
  2. 分批删除：循环 `DELETE FROM xxx WHERE created_at < ? LIMIT 1000` 直到影响行数为 0（防大表长锁）
  3. 同步清 ES：`delete_by_query`（range created_at < 截止时间），ES 不可用仅告警、不阻断
  4. 返回 `{"operation_deleted": n, "login_deleted": n}` 统计

### 2. 改动文件清单

| 文件 | 改动点 |
|---|---|
| `migrations/000006_add_logs_created_at_index.up.sql` | `CREATE INDEX idx_operation_logs_created_at ON operation_logs(created_at)`、`CREATE INDEX idx_login_logs_created_at ON login_logs(created_at)` |
| `migrations/000006_add_logs_created_at_index.down.sql` | 对应 `DROP INDEX` |
| `internal/dao/log_dao.go` | 加 `DeleteOlderThan(before time.Time, limit int) (int64, error)`，按 `created_at < ?` 分批删除 |
| `internal/dao/login_log_dao.go` | 加 `DeleteOlderThan(before time.Time, limit int) (int64, error)` |
| `internal/es/log_repo.go` | 加 `DeleteByTime(before time.Time) (int64, error)`：`delete_by_query` range 删除，Refresh=true |
| `internal/service/log_service.go` | 加 `Cleanup(days int, scope string) (CleanupResult, error)`：编排分批删两表 + ES 降级 |
| `internal/controller/log_controller.go` | 加 `Cleanup` 处理器：解析参数、比对密钥、调 service、返回统计 |
| `internal/router/router.go` | 公开路由区加 `POST /logs/cleanup` |
| `internal/config/config.go` | 新增 `LogCleanupSecret` 配置项 |
| `.env` / `.env.example` | 新增 `LOG_CLEANUP_SECRET` |

### 3. 定时任务配置（纯数据，无代码）

任务管理页新建一条任务：

- URL：`http://go-app:8000/api/logs/cleanup?secret=<LOG_CLEANUP_SECRET>&days=30`
- Method：POST（body 为空，参数走 URL 查询串）
- Cron：`0 0 3 * * *`（每天凌晨 3 点）
- 不进迁移种子：不同环境 URL/密钥不同，页面自建

### 4. 明确不做（YAGNI）

- 不加新权限点/菜单、不加前端页面（复用现有任务管理页）
- 不动 `cron_task_executions` 清理（不在本次范围）
- 不改调度器（任务仍是纯 HTTP 回调）
- 不加清理按钮（无前端页面）

## 错误处理

- `secret` 不匹配：HTTP 403 + 业务错误
- `days` 非法：业务码 400（`utils.Error` 惯例，HTTP 200）
- ES 不可用：`log.Printf` 告警，继续返回 MySQL 清理结果
- DAO 删除失败：返回错误，清理中断（下次调度重试）

## 验证方式

无测试基建，手工回归：

1. 启动服务，造若干 30 天前的旧日志 + 若干新日志
2. `curl -X POST "http://localhost:8000/api/logs/cleanup?secret=xxx&days=30"` 验证旧日志被删、新日志保留、返回统计正确
3. 错误 secret / 非法 days 分别验证 403 / 400
4. 验证 ES 中旧文档被删除（`GET /logs` 检索不到旧数据）
5. 任务管理页建任务 → "立即执行" → 执行日志状态成功，日志表清理生效
6. 验证 `go build` 编译通过
