# Zap 结构化日志改造设计

## 背景

项目当前使用 Go 标准库 `log` 输出日志到控制台，无文件持久化、无结构化格式、无级别控制。生产环境排查问题困难，无法检索和告警。

## 目标

- 引入 zap 结构化日志，支持 debug/info/warn/error 四级
- 日志同时输出到控制台（彩色文本）和文件（JSON）
- 文件按大小轮转（lumberjack），保留最近 7 个文件、30 天
- 通过 .env 配置日志级别、文件路径、轮转参数
- Docker 挂载日志目录到宿主机

## 架构

```
main.go 启动
  → config.Load() 读取 .env 日志配置
  → logger.Init() 初始化 zap
  → 业务代码用 logger.Info/Warn/Error/Fatal 替代 log.Printf
  → gin.Logger() 保持不动
```

## 新增文件

### `internal/logger/logger.go`

- 全局变量 `L *zap.Logger`
- `Init(cfg LogConfig)` 初始化：控制台彩色文本 + 文件 JSON，lumberjack 轮转
- 便捷方法：`Info/Warn/Error/Fatal(msg, fields...)`
- `Sync()` 刷盘，进程退出前调用

## 修改文件

### `internal/config/config.go`

新增 `LogConfig` 结构体和对应字段：

```go
type LogConfig struct {
    Level      string // debug, info, warn, error
    FilePath   string // 日志文件路径，空则仅输出控制台
    MaxSize    int    // 单文件最大 MB
    MaxBackups int    // 保留旧文件个数
    MaxDays    int    // 保留天数
}
```

Load() 中新增默认值和读取逻辑。

### `.env` 新增配置项

```
LOG_LEVEL=info
LOG_FILE_PATH=logs/app.log
LOG_MAX_SIZE=100
LOG_MAX_BACKUPS=7
LOG_MAX_DAYS=30
```

### `docker-compose.yml`

go-app 服务新增 volume：`go-logs:/app/logs`，底部 volumes 新增 `go-logs`。

### `docker/Dockerfile`

运行阶段新增 `RUN mkdir -p logs`。

### 业务文件替换（13 个文件，约 27 处调用）

| 文件 | log 调用 | 替换为 |
|------|----------|--------|
| `cmd/server/main.go` | log.Fatalf(MySQL 连接失败) | logger.Fatal(...) |
| `cmd/server/main.go` | log.Fatalf(数据库迁移失败) | logger.Fatal(...) |
| `cmd/server/main.go` | log.Printf(ES 不可用警告) | logger.Warn(...) |
| `cmd/server/main.go` | log.Printf(ES 索引初始化失败) | logger.Warn(...) |
| `cmd/server/main.go` | log.Fatalf(RabbitMQ 连接失败) | logger.Fatal(...) |
| `cmd/server/main.go` | log.Fatalf(RabbitMQ Channel 失败) | logger.Fatal(...) |
| `cmd/server/main.go` | log.Printf(Server running) | logger.Info(...) |
| `cmd/server/main.go` | log.Fatalf(启动失败) | logger.Fatal(...) |
| `cmd/server/main.go` | log.Println(数据库迁移完成) | logger.Info(...) |
| `internal/captcha/store.go` | log.Printf(验证码存储失败) | logger.Error(...) |
| `internal/controller/auth_controller.go` | log.Printf(读取系统设置失败) | logger.Warn(...) |
| `internal/controller/auth_controller.go` | log.Printf(验证码读取失败) | logger.Warn(...) |
| `internal/controller/auth_controller.go` | log.Printf(登录告警任务发布失败) | logger.Warn(...) |
| `internal/controller/log_controller.go` | log.Printf(ES 查询失败) | logger.Warn(...) |
| `internal/scheduler/scheduler.go` | log.Printf(调度器加载任务失败) | logger.Error(...) |
| `internal/scheduler/scheduler.go` | log.Printf(任务注册失败) | logger.Error(...) |
| `internal/scheduler/scheduler.go` | log.Printf(调度器已加载任务) | logger.Info(...) |
| `internal/scheduler/scheduler.go` | log.Printf(任务执行 panic) | logger.Error(...) |
| `internal/scheduler/scheduler.go` | log.Printf(执行日志写入失败) | logger.Error(...) |
| `internal/middleware/logger.go` | log.Printf(ES 日志写入失败) | logger.Warn(...) |
| `internal/es/log_repo.go` | log.Printf(ES 命中解析失败) | logger.Warn(...) |
| `internal/worker/export_worker.go` | log.Printf(导出任务失败) | logger.Error(...) |
| `internal/worker/mail_worker.go` | log.Printf(邮件任务解析失败) | logger.Error(...) |
| `internal/worker/mail_worker.go` | log.Printf(登录告警邮件发送失败) | logger.Error(...) |
| `internal/service/alert_mail.go` | log.Printf(SMTP 未配置) | logger.Warn(...) |
| `internal/service/file_service.go` | log.Printf(物理文件删除失败) | logger.Warn(...) |
| `internal/service/log_service.go` | log.Printf(ES 清理失败) | logger.Warn(...) |
| `internal/service/log_service.go` | log.Printf(ES 已清理) | logger.Info(...) |
| `internal/service/notification_service.go` | log.Printf(消息落库失败) | logger.Error(...) |

## 依赖

- `go.uber.org/zap` — 结构化日志库
- `gopkg.in/natefinish/lumberjack.v2` — 文件轮转

## 不改动

- `gin.Logger()` 保持原样，HTTP 请求日志不接管
- 业务操作日志（OperationLogger 中间件写 MySQL/ES）不受影响
- 不改变任何业务逻辑

## 验证

1. `docker compose up -d --build go-app`
2. `docker compose exec go-app ls -la logs/` — 确认日志文件生成
3. `docker compose exec go-app cat logs/app.log` — 确认 JSON 格式
4. `docker compose logs --tail 20 go-app` — 确认控制台仍有彩色文本输出
5. 登录系统，操作几个页面，确认业务日志正常写入
