# Zap 结构化日志改造 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将项目中所有 Go 标准 `log` 调用替换为 zap 结构化日志，支持文件轮转和级别控制。

**Architecture:** 新增 `internal/logger` 包封装 zap，控制台彩色文本 + 文件 JSON 双输出，lumberjack 按大小轮转。通过 .env 配置级别/路径/轮转参数。gin.Logger() 保持不动。

**Tech Stack:** go.uber.org/zap, gopkg.in/natefinish/lumberjack.v2, Go 1.18, Gin 1.9.1, Docker Compose

---

## File Structure

**Already modified (brainstorming phase):**
- `internal/logger/logger.go` — zap 初始化包（已创建，lumberjack 导入待修复）
- `internal/config/config.go` — LogConfig 结构体和 Load() 读取（已完成）
- `.env` — 日志配置项（已完成）
- `docker-compose.yml` — go-logs volume（已完成）
- `docker/Dockerfile` — mkdir logs（已完成）

**Needs modification:**
- `go.mod` / `go.sum` — 添加 lumberjack 依赖
- `cmd/server/main.go` — 初始化 logger + 替换 9 处 log 调用
- `internal/captcha/store.go` — 替换 1 处
- `internal/controller/auth_controller.go` — 替换 3 处
- `internal/controller/log_controller.go` — 替换 1 处
- `internal/scheduler/scheduler.go` — 替换 5 处
- `internal/middleware/logger.go` — 替换 1 处
- `internal/es/log_repo.go` — 替换 1 处
- `internal/worker/export_worker.go` — 替换 1 处
- `internal/worker/mail_worker.go` — 替换 2 处
- `internal/service/alert_mail.go` — 替换 1 处
- `internal/service/file_service.go` — 替换 1 处
- `internal/service/log_service.go` — 替换 2 处
- `internal/service/notification_service.go` — 替换 1 处

---

### Task 1: 安装 lumberjack 依赖

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: 安装 lumberjack**

```bash
cd D:/phpStudy/PHPTutorial/WWW/ginproject
GOPROXY=https://goproxy.cn,direct go get gopkg.in/natefinish/lumberjack.v2@latest
```

Expected: go.mod 中出现 `gopkg.in/natefinish/lumberjack.v2` 依赖

- [ ] **Step 2: 验证依赖安装成功**

```bash
go mod tidy
```

Expected: 无报错，go.mod 和 go.sum 更新

- [ ] **Step 3: 修复 logger.go 的导入路径（如需要）**

检查 `internal/logger/logger.go` 第 9 行的导入路径是否与安装的包一致。如果 `gopkg.in/natefinish/lumberjack.v2` 安装失败，改用 `github.com/natefinish/lumberjack/v2`。

- [ ] **Step 4: 验证编译通过**

```bash
go build ./internal/logger/
```

Expected: 无报错

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum internal/logger/logger.go
git commit -m "deps: 添加 zap 和 lumberjack 日志依赖"
```

---

### Task 2: 替换 cmd/server/main.go 的 log 调用

**Files:**
- Modify: `cmd/server/main.go`

9 处调用需要替换。main.go 还需要在 `main()` 开头初始化 logger。

- [ ] **Step 1: 修改 import**

将 `"log"` 替换为 `"ginproject/internal/logger"` 和 `"go.uber.org/zap"`。

删除 `"log"` import，新增：
```go
"ginproject/internal/logger"
"go.uber.org/zap"
```

- [ ] **Step 2: 在 main() 开头添加 logger 初始化**

在 `cfg := config.Load()` 之后添加：

```go
logger.Init(logger.LogConfig{
    Level:      cfg.Log.Level,
    FilePath:   cfg.Log.FilePath,
    MaxSize:    cfg.Log.MaxSize,
    MaxBackups: cfg.Log.MaxBackups,
    MaxDays:    cfg.Log.MaxDays,
})
defer logger.Sync()
```

- [ ] **Step 3: 替换 9 处 log 调用**

| 行号 | 原代码 | 替换为 |
|------|--------|--------|
| 49 | `log.Fatalf("MySQL 连接失败: %v", err)` | `logger.Fatal("MySQL 连接失败", zap.Error(err))` |
| 60 | `log.Fatalf("数据库迁移失败: %v", err)` | `logger.Fatal("数据库迁移失败", zap.Error(err))` |
| 66 | `log.Printf("警告: Elasticsearch 不可用...%v", esErr)` | `logger.Warn("Elasticsearch 不可用，日志全文搜索将回退 MySQL", zap.Error(esErr))` |
| 72 | `log.Printf("警告: ES 索引初始化失败: %v", err)` | `logger.Warn("ES 索引初始化失败", zap.Error(err))` |
| 108 | `log.Fatalf("RabbitMQ 连接失败: %v", err)` | `logger.Fatal("RabbitMQ 连接失败", zap.Error(err))` |
| 114 | `log.Fatalf("RabbitMQ Channel 创建失败: %v", err)` | `logger.Fatal("RabbitMQ Channel 创建失败", zap.Error(err))` |
| 246 | `log.Printf("Server running on :%s", cfg.Server.Port)` | `logger.Info("Server running", zap.String("port", cfg.Server.Port))` |
| 248 | `log.Fatalf("启动失败: %v", err)` | `logger.Fatal("启动失败", zap.Error(err))` |
| 267 | `log.Println("数据库迁移完成")` | `logger.Info("数据库迁移完成")` |

- [ ] **Step 4: 验证编译**

```bash
go build ./cmd/server/
```

Expected: 无报错

- [ ] **Step 5: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: main.go 初始化 zap 日志并替换 log 调用"
```

---

### Task 3: 替换 internal/captcha/store.go

**Files:**
- Modify: `internal/captcha/store.go:27`

- [ ] **Step 1: 替换 import 和 log 调用**

删除 `"log"` import，新增 `"ginproject/internal/logger"` 和 `"go.uber.org/zap"`。

第 27 行：
```go
// 原：log.Printf("验证码存储失败: %v", err)
logger.Error("验证码存储失败", zap.Error(err))
```

- [ ] **Step 2: Commit**

```bash
git add internal/captcha/store.go
git commit -m "feat: captcha/store.go 替换 log 为 zap"
```

---

### Task 4: 替换 internal/controller/auth_controller.go

**Files:**
- Modify: `internal/controller/auth_controller.go:90,105,141`

- [ ] **Step 1: 替换 import 和 3 处 log 调用**

删除 `"log"` import，新增 `"ginproject/internal/logger"` 和 `"go.uber.org/zap"`。

```go
// 第90行 原：log.Printf("读取系统设置失败: %v", err)
logger.Warn("读取系统设置失败", zap.Error(err))

// 第105行 原：log.Printf("验证码读取失败（可能已过期）: %v", err)
logger.Warn("验证码读取失败（可能已过期）", zap.Error(err))

// 第141行 原：log.Printf("登录告警任务发布失败: %v", err)
logger.Warn("登录告警任务发布失败", zap.Error(err))
```

- [ ] **Step 2: Commit**

```bash
git add internal/controller/auth_controller.go
git commit -m "feat: auth_controller.go 替换 log 为 zap"
```

---

### Task 5: 替换 internal/controller/log_controller.go

**Files:**
- Modify: `internal/controller/log_controller.go:75`

- [ ] **Step 1: 替换 import 和 log 调用**

删除 `"log"` import，新增 `"ginproject/internal/logger"` 和 `"go.uber.org/zap"`。

```go
// 第75行 原：log.Printf("ES 查询失败，回退 MySQL: %v", err)
logger.Warn("ES 查询失败，回退 MySQL", zap.Error(err))
```

- [ ] **Step 2: Commit**

```bash
git add internal/controller/log_controller.go
git commit -m "feat: log_controller.go 替换 log 为 zap"
```

---

### Task 6: 替换 internal/scheduler/scheduler.go

**Files:**
- Modify: `internal/scheduler/scheduler.go:64,73,78,94,179`

- [ ] **Step 1: 替换 import 和 5 处 log 调用**

删除 `"log"` import，新增 `"ginproject/internal/logger"` 和 `"go.uber.org/zap"`。

```go
// 第64行 原：log.Printf("调度器加载任务失败: %v", err)
logger.Error("调度器加载任务失败", zap.Error(err))

// 第73行 原：log.Printf("任务 %d 注册失败: %v", task.ID, err)
logger.Error("任务注册失败", zap.Uint("task_id", task.ID), zap.Error(err))

// 第78行 原：log.Printf("调度器已加载 %d 个启用任务", len(tasks))
logger.Info("调度器已加载启用任务", zap.Int("count", len(tasks)))

// 第94行 原：log.Printf("任务 %d 执行 panic: %v", task.ID, r)
logger.Error("任务执行 panic", zap.Uint("task_id", task.ID), zap.Any("recover", r))

// 第179行 原：log.Printf("任务 %d 执行日志写入失败: %v", taskID, err)
logger.Error("任务执行日志写入失败", zap.Uint("task_id", taskID), zap.Error(err))
```

- [ ] **Step 2: Commit**

```bash
git add internal/scheduler/scheduler.go
git commit -m "feat: scheduler.go 替换 log 为 zap"
```

---

### Task 7: 替换 internal/middleware/logger.go

**Files:**
- Modify: `internal/middleware/logger.go:55`

- [ ] **Step 1: 替换 import 和 log 调用**

删除 `"log"` import，新增 `"ginproject/internal/logger"` 和 `"go.uber.org/zap"`。

```go
// 第55行 原：log.Printf("ES 日志写入失败: %v", err)
logger.Warn("ES 日志写入失败", zap.Error(err))
```

- [ ] **Step 2: Commit**

```bash
git add internal/middleware/logger.go
git commit -m "feat: middleware/logger.go 替换 log 为 zap"
```

---

### Task 8: 替换 internal/es/log_repo.go

**Files:**
- Modify: `internal/es/log_repo.go:126`

- [ ] **Step 1: 替换 import 和 log 调用**

删除 `"log"` import，新增 `"ginproject/internal/logger"` 和 `"go.uber.org/zap"`。

```go
// 第126行 原：log.Printf("ES 命中解析失败: %v", err)
logger.Warn("ES 命中解析失败", zap.Error(err))
```

- [ ] **Step 2: Commit**

```bash
git add internal/es/log_repo.go
git commit -m "feat: es/log_repo.go 替换 log 为 zap"
```

---

### Task 9: 替换 internal/worker/export_worker.go

**Files:**
- Modify: `internal/worker/export_worker.go:102`

- [ ] **Step 1: 替换 import 和 log 调用**

删除 `"log"` import，新增 `"ginproject/internal/logger"` 和 `"go.uber.org/zap"`。

```go
// 第102行 原：log.Printf("导出任务 %s 失败: %v", taskID, err)
logger.Error("导出任务失败", zap.String("task_id", taskID), zap.Error(err))
```

- [ ] **Step 2: Commit**

```bash
git add internal/worker/export_worker.go
git commit -m "feat: export_worker.go 替换 log 为 zap"
```

---

### Task 10: 替换 internal/worker/mail_worker.go

**Files:**
- Modify: `internal/worker/mail_worker.go:50,55`

- [ ] **Step 1: 替换 import 和 2 处 log 调用**

删除 `"log"` import，新增 `"ginproject/internal/logger"` 和 `"go.uber.org/zap"`。

```go
// 第50行 原：log.Printf("邮件任务解析失败: %v", err)
logger.Error("邮件任务解析失败", zap.Error(err))

// 第55行 原：log.Printf("登录告警邮件发送失败: %v", err)
logger.Error("登录告警邮件发送失败", zap.Error(err))
```

- [ ] **Step 2: Commit**

```bash
git add internal/worker/mail_worker.go
git commit -m "feat: mail_worker.go 替换 log 为 zap"
```

---

### Task 11: 替换 internal/service/ 下 4 个文件

**Files:**
- Modify: `internal/service/alert_mail.go:33`
- Modify: `internal/service/file_service.go:94`
- Modify: `internal/service/log_service.go:59,61`
- Modify: `internal/service/notification_service.go:61`

- [ ] **Step 1: 替换 alert_mail.go**

删除 `"log"` import，新增 `"ginproject/internal/logger"` 和 `"go.uber.org/zap"`。

```go
// 第33行 原：log.Printf("登录告警邮件未发送：SMTP 未配置")
logger.Warn("登录告警邮件未发送：SMTP 未配置")
```

- [ ] **Step 2: 替换 file_service.go**

删除 `"log"` import，新增 `"ginproject/internal/logger"` 和 `"go.uber.org/zap"`。

```go
// 第94行 原：log.Printf("警告: 物理文件删除失败 %s: %v", savePath, err)
logger.Warn("物理文件删除失败", zap.String("path", savePath), zap.Error(err))
```

- [ ] **Step 3: 替换 log_service.go**

删除 `"log"` import，新增 `"ginproject/internal/logger"` 和 `"go.uber.org/zap"`。

```go
// 第59行 原：log.Printf("ES 清理旧操作日志失败（已降级仅清 MySQL）: %v", err)
logger.Warn("ES 清理旧操作日志失败（已降级仅清 MySQL）", zap.Error(err))

// 第61行 原：log.Printf("ES 已清理 %d 条旧操作日志", n)
logger.Info("ES 已清理旧操作日志", zap.Int("count", n))
```

- [ ] **Step 4: 替换 notification_service.go**

删除 `"log"` import，新增 `"ginproject/internal/logger"` 和 `"go.uber.org/zap"`。

```go
// 第61行 原：log.Printf("系统事件消息落库失败: %v", err)
logger.Error("系统事件消息落库失败", zap.Error(err))
```

- [ ] **Step 5: Commit**

```bash
git add internal/service/alert_mail.go internal/service/file_service.go internal/service/log_service.go internal/service/notification_service.go
git commit -m "feat: service 层 4 个文件替换 log 为 zap"
```

---

### Task 12: Docker 构建验证

**Files:**
- None (verification only)

- [ ] **Step 1: Docker 重建 go-app**

```bash
cd D:/phpStudy/PHPTutorial/WWW/ginproject
docker compose up -d --build go-app
```

Expected: 构建成功，容器启动正常

- [ ] **Step 2: 确认日志文件生成**

```bash
docker compose exec go-app ls -la logs/
```

Expected: 看到 `app.log` 文件

- [ ] **Step 3: 确认 JSON 格式**

```bash
docker compose exec go-app head -5 logs/app.log
```

Expected: JSON 格式，包含 `ts`、`level`、`msg` 等字段

- [ ] **Step 4: 确认控制台输出**

```bash
docker compose logs --tail 20 go-app
```

Expected: 控制台仍有彩色文本日志输出

- [ ] **Step 5: 调用 API 触发业务日志**

```bash
curl -k https://localhost:8443/api/health
```

Expected: 日志文件中出现对应的请求日志

- [ ] **Step 6: 最终 Commit（如有修复）**

如果有任何修复，单独提交。
