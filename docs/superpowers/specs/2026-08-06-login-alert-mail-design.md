# 登录异常邮件告警 设计文档

日期：2026-08-06
状态：已实现

## 目标

当用户登录因「用户名或密码错误」失败时，向固定的告警邮箱发送一封通知邮件，不阻塞登录接口，并通过限频防止暴力破解时的邮件轰炸。

## 现状分析与决策

- 项目原先无任何邮件代码；登录失败仅写入 `login_logs`（status=0）。
- 决策点（已与需求方确认）：
  - **收件人**：固定告警邮箱（`.env` 的 `SMTP_TO`），发给用户本人。
  - **触发条件**：仅「用户名或密码错误」（`ErrInvalidCredentials`）；账号禁用、参数错误不触发。
  - **限频**：同一 IP 5 分钟窗口内最多发 1 封（Redis `alert_mail:<ip>`）。
  - **投递方式**：沿用项目现有「RabbitMQ 同步发布 + 后台 Worker 消费」模式（与日志导出 worker 一致），而非 goroutine 直发。原因：进程重启不丢在途告警、SMTP 慢时不会无界堆积 goroutine、与现有架构统一。
  - **SMTP 客户端**：Go 标准库 `net/smtp`，不引入第三方依赖；`DialTimeout` 5s。
  - **失败处理**：邮件为可容忍丢失的告警，发送失败仅记日志并 Ack（不积压队列），任何环节失败均不影响登录接口响应。

## 架构与数据流

```
登录请求 → AuthController.Login
  → 失败且 errors.Is(err, ErrInvalidCredentials)
  → Redis SetNX alert_mail:<ip> (5min TTL)，限频命中则跳过
  → Publish JSON{username, ip, message} → 队列 mail.send（持久队列）
  → MailWorker.Start() 后台消费
  → AlertMailService.SendLoginAlert → net/smtp 发邮件到 SMTP_TO
```

## 改动清单

| 文件 | 改动 |
| --- | --- |
| `internal/config/config.go` | 新增 `MailConfig`（SMTP_HOST/PORT/USER/PASSWORD/FROM/TO） |
| `.env` / `.env.example` | 补 SMTP 变量；`.env.example` 顺带补缺失的 RabbitMQ 变量 |
| `internal/service/auth_service.go` | 新增哨兵错误 `ErrInvalidCredentials`，Login 两处返回改用它 |
| `internal/service/alert_mail.go`（新增） | `AlertMailService.SendLoginAlert`：SMTP 发送，未配置时静默跳过；定义队列常量 `LoginAlertQueue = "mail.send"` |
| `internal/worker/mail_worker.go`（新增） | `MailWorker.Start()`：消费 `mail.send`，调发送器后 Ack |
| `internal/controller/auth_controller.go` | Login 失败分支精确匹配哨兵错误 → Redis 限频 → 发布任务；新增注入 `rdb`、`publishCh` |
| `cmd/server/main.go` | 组装 `AlertMailService`、`MailWorker`（`go mailWorker.Start()`），`AuthController` 注入 rdb/publishCh |

## 边界与约束

- SMTP_HOST/TO 为空时邮件功能整体禁用，不影响登录。
- 发布失败、发送失败、Redis 不可用时均只记日志，登录接口照常响应。
- MQ 消费 channel 各 worker 独立创建，共享同一 `amqpConn` 连接（与 export worker 一致）。
- Redis 版本为 3.2-alpine（不支持 HSET 多字段），限频使用单字段 `SetNX`，符合约定。

## 验证

- `go build ./...`、`go vet ./...` 均通过。
- 手动验证路径：配置真实 SMTP → 用错误密码登录 → 观察登录日志出现发布记录、`SMTP_TO` 收到「登录异常告警」邮件；5 分钟内同 IP 重复失败不再重复发送。