# RabbitMQ 速查复习笔记

本项目 RabbitMQ 只做两件事：**日志异步导出**（队列 `excel.export`）+ **登录告警邮件**（队列 `mail.send`）。
无自定义 Exchange，全部走**默认交换机**（空 exchange，routing key = 队列名），生产消费在同一应用。

代码位置：`cmd/server/main.go`(连接/channel 初始化)、`internal/controller/log_controller.go`、`internal/controller/auth_controller.go`(发布端)、`internal/worker/export_worker.go`、`internal/worker/mail_worker.go`(消费端)。

## 0. 架构一图流

```
main.go:74  amqp091.Dial(cfg.RabbitMQ.DSN())  → 全局唯一连接 amqpConn
                    │
        ┌───────────┼───────────────────────┐
        │ main.go:80 publishCh(一个channel)   │ main.go:92  go exportWorker.Start()
        │  ← controller 发布消息用          │ main.go:96  go mailWorker.Start()
        │                                  │ （每个 worker 内部再各自 Channel()）
        └───────────────────────────────────┘
```

要点：**一个 Connection 全局复用**；发布与两个消费者各用独立 Channel（Channel 非线程安全，惯例一个业务一个 Channel）。

## 1. 队列总览（默认交换机 → 路由键 = 队列名）

| 队列 | 发布端 | 消费端 | 消息体 |
|---|---|---|---|
| `excel.export` | `LogController.Export` log_controller.go:84 | `ExportWorker` export_worker.go:52 | `{"task_id": uuid}` |
| `mail.send`（`service.LoginAlertQueue` alert_mail.go:16） | `AuthController.publishLoginAlert` auth_controller.go:75 | `MailWorker` mail_worker.go:35 | `{"username","ip","message"}` |

两处发布都调 `Publish("", 队列名, ...)`：第一个参数为空 = 默认交换机，即"发到同名队列"。

## 2. 流程 1：日志异步导出（完整闭环）

1. **发布** `POST /api/logs/export`（log_controller.go:67）：
   生成 `taskID` → 写 Redis `excel:task:<id>`（`pending`/`user_id`/`method`，24h TTL）→ 发布 `{"task_id"}` 到 `excel.export`。**任务状态全靠 Redis，MQ 只传 taskID。**
2. **消费** `ExportWorker.Start`（export_worker.go:43）：
   `QueueDeclare(name, durable=true)` → `ch.Qos(1,0,false)`（预取 1，公平分发）→ `Consume(autoAck=false)` → 循环 `processTask`（export_worker.go:73）：
   - 置 `processing` → 从 Redis 取 `user_id`/`method`
   - `buildExcel`（export_worker.go:113）：excelize `StreamWriter` 分批 5000 行写 xlsx（`exports/<taskID>.xlsx`），**必须全用 SetRow，别混 SetCellValue**
   - 成功：Redis 置 `success`+`filename`；失败：置 `failed`+`error`
3. **通知**：WebSocket 推 `export_complete` / `export_failed`（export_worker.go:105,93）
4. **下载**：前端轮询 `GET /api/logs/export-status?task_id=`；成功后 `GET /api/logs/download/:taskID`（校验 task 的 user_id 归属，`c.File` 后**即删文件** log_controller.go:135）

## 3. 流程 2：登录告警邮件

`publishLoginAlert`（auth_controller.go:61）：
1. Redis `SetNX("alert_mail:"+ip, ..., 5min)` 限频（auth_controller.go:63），未拿到锁直接 return，**不重复发**
2. 发布 `mail.send` → `MailWorker.Start`（mail_worker.go:28）消费 → `SendLoginAlert` 发 SMTP 邮件
3. SMTP 未配置时静默跳过（alert_mail.go:30），发布失败仅 `log.Printf`，**不影响登录**

## 4. 可靠性现状速查（面试记忆点）

| 有 ✅ | 缺 ❌ |
|---|---|
| 队列持久化 `QueueDeclare(..., true, ...)` | 失败也无条件 `Ack`：export_worker.go:69 / mail_worker.go:57 → **处理失败消息即丢** |
| `Qos(1)` 预取 + 公平分发 | 无断线重连：`for msg := range msgs` 连接断了 goroutine 直接死（export_worker.go:64 / mail_worker.go:47） |
| 手动 Ack（`Consume(..., false, ...)`） | `Publish` 两处都不检查返回值 → 发布失败静默丢（log_controller.go:84 / auth_controller.go:75） |
| Redis 任务态兜底（导出失败可见可查） | 无 Exchange 设计/无死信队列/无重试/无幂等码 |
| | 单 goroutine 串行消费，Qos 未配合并发消费 |

**记住**：队列声明只在消费端（worker 里），发布端不声明 → **worker 还没起来时发布的消息直接丢弃**。

## 5. 概念对照（面试问答映射本项目）

- **默认交换机 = 隐式 direct**：固定空键 + 绑定同名队列 = 一对一。本项目两队列都是这种"一个队列一个用途"，够用。
- **direct 直连交换机**：显式声明 + `QueueBind` 任意 key；例：A 队列绑 `order.created`+`order.paid`、B 队列绑 `order.cancelled` —— **精确全等匹配，谁绑了谁收**；多个队列绑同 key 时消息复制给每个。适合"多源汇聚 / 按 key 分流"。
- **fanout**：广播，消息复制给所有绑定队列，不管 key——"一个事件通知所有系统"时用，默认交换机做不到。
- **topic**：路由键通配（`*` 一段、`#` 多段）——带层级的事件流。

**多副本两种含义（易混）**：
- 同队列多个消费者 = 消息**瓜分**（一条只被一个处理）→ 默认交换机就够，本项目加 worker goroutine 即可横向扩
- 多个队列都收 = **广播** → 必须换 fanout/topic

**可靠性三板斧（默认交换机也能做，与 Exchange 无关）**：失败按 Ack/Nack + 重试、断线重连、发布确认，本项目三块都没有。

## 6. 易忘点

- 默认交换机：`Publish("", 队列名, ...)`，第一参空串 = 默认交换机；交换机和队列绑定概念在此不涉及
- 队列 `durable=true` 只是队列持久，消息还要 `DeliveryMode: Persistent` 才会落盘（本项目未设）
- 两个 worker 出错都 `panic`（channel/消费注册失败）——启动期配置错直接崩，运行期断线则是 goroutine 静默退出
- Redis 是 3.2，HSet 只能单字段一调（见 AGENTS.md），导出任务态就是多行单字段 HSet
- 下载即删 xlsx：导出文件只存在生命周期内，DB/ES 不存文件