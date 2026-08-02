# 异步导出操作日志 —— 设计文档

## 动机

当前 `GET /api/logs/export` 同步导出：查询全量操作日志 → 生成 Excel → 返回文件流。几十万条数据下前后端必然超时。需要改为异步模式。

## 架构

```
POST /api/logs/export → Gin:
  1. UUID 生成 taskID
  2. HSET excel:task:{taskID} status=processing ...
  3. Publish RabbitMQ {task_id}
  4. 返回 {task_id}

RabbitMQ "excel.export" → Worker Consume:
  1. HSET status=processing
  2. 分批查 operation_logs (5000/批) + excelize StreamWriter 流式写
  3. 存入 ./exports/{taskID}.xlsx
  4. HSET status=success filename=...
  5. hub.Send(userID, export_complete/{download_url})

WebSocket → 浏览器收到通知 → 显示下载链接

GET /api/logs/download/:taskID → Gin:
  1. HGET excel:task:{taskID} → 校验 user_id
  2. c.File() 返回
  3. os.Remove() 删除文件
```

## 组件

| 组件 | 用途 |
|---|---|
| RabbitMQ | 任务队列，durable queue，ACK 确认 |
| Redis Hash | 任务状态：status / user_id / module / method / filename / error，24h TTL |
| WebSocket | gorilla/websocket，Hub 管理连接，Worker 直接 hub.Send 推送 |
| 磁盘 | ./exports/ 存 xlsx，下载即删 |

## 接口

| 方法 | 路径 | 权限 | 说明 |
|---|---|---|---|
| POST | /api/logs/export | log:export | 创建导出任务 |
| GET | /api/logs/download/:taskID | log:export | 下载文件，校验归属后删 |
| GET | /api/logs/export-status | log:export | 查询任务状态（轮询降级） |
| GET | /api/ws | JWT (?token=) | WebSocket 连接 |

## WebSocket 消息

```json
{"type":"export_complete","task_id":"...","filename":"...","download_url":"..."}
{"type":"export_failed","task_id":"...","error":"..."}
{"type":"heartbeat"}
```

## 大数据处理

- excelize `NewStreamWriter` 流式写入
- DAO `FindBatch(module, method, offset, limit)` 分 5000 条/批
- Worker 循环：查一批 → 流式写一批，内存恒定

## 降级

前端同时监听 WS + 2s 轮询 GET /api/logs/export-status。WS 断开时轮询兜底。

## 安全

- 下载校验 `excel:task:{taskID}` 中 user_id == 当前用户
- WS 认证通过 `?token=` 查询参数传递 JWT

## 新增依赖

- `github.com/gorilla/websocket v1.5.1`
- `github.com/rabbitmq/amqp091-go v1.8.1`
- Docker: `rabbitmq:3-management`

## 新增文件

- `internal/utils/uuid.go`
- `internal/ws/hub.go`、`internal/ws/upgrader.go`
- `internal/controller/ws_controller.go`
- `internal/worker/export_worker.go`
- `web/src/utils/ws.js`

## 修改文件

- `go.mod`、`docker-compose.yml`、`.env`
- `internal/config/config.go`、`cmd/server/main.go`
- `internal/router/router.go`、`internal/controller/log_controller.go`
- `internal/dao/log_dao.go`、`internal/service/log_service.go`
- `web/src/api/log.js`、`web/src/views/log/index.vue`
- `web/src/layout/index.vue`、`web/src/store/modules/user.js`
- `.gitignore`
