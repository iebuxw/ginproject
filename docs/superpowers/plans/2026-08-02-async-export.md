# 异步导出操作日志 —— 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将操作日志 Excel 导出从同步改为 RabbitMQ 队列 + WebSocket 通知的异步模式

**Architecture:** 前端 POST 创建导出任务 → Gin HSET Redis + Publish RabbitMQ → Worker Consume 分批流式生成 Excel → hub.Send WebSocket 推送 → 前端下载后删文件

**Tech Stack:** Go 1.18, Gin, GORM, go-redis/v9, RabbitMQ (amqp091-go), gorilla/websocket, Vue 2.7, Element UI

---

### Task 1: 新增 Go 依赖

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: 添加依赖**

在 `go.mod` 的 `require` 块中添加两条依赖。执行：

```bash
cd D:/phpStudy/PHPTutorial/WWW/ginproject
go get github.com/gorilla/websocket@v1.5.1
go get github.com/rabbitmq/amqp091-go@v1.8.1
go mod tidy
```

- [ ] **Step 2: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: 新增 gorilla/websocket 和 rabbitmq/amqp091-go"

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
```

---

### Task 2: Docker Compose 和配置

**Files:**
- Modify: `docker-compose.yml`
- Modify: `.env`
- Modify: `internal/config/config.go`

- [ ] **Step 1: 添加 RabbitMQ 服务到 docker-compose.yml**

在 `redis` 服务和 `go-app` 服务之间插入：

```yaml
  rabbitmq:
    image: rabbitmq:3-management
    container_name: ginadmin-rabbitmq
    restart: always
    ports:
      - "15672:15672"
      - "5672:5672"
    environment:
      RABBITMQ_DEFAULT_USER: guest
      RABBITMQ_DEFAULT_PASS: guest
    healthcheck:
      test: ["CMD", "rabbitmq-diagnostics", "-q", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
```

修改 `go-app` 的 `depends_on`，增加 rabbitmq：

```yaml
    depends_on:
      mysql:
        condition: service_healthy
      redis:
        condition: service_healthy
      rabbitmq:
        condition: service_healthy
```

- [ ] **Step 2: 添加 RabbitMQ 环境变量到 .env**

```env
RABBITMQ_HOST=rabbitmq
RABBITMQ_PORT=5672
RABBITMQ_USER=guest
RABBITMQ_PASSWORD=guest
```

- [ ] **Step 3: 添加 RabbitMQConfig 到 config.go**

在 `Config` 结构体增加 `RabbitMQ RabbitMQConfig`，新增类型：

```go
type RabbitMQConfig struct {
    Host     string
    Port     string
    User     string
    Password string
}

func (c RabbitMQConfig) DSN() string {
    return fmt.Sprintf("amqp://%s:%s@%s:%s/", c.User, c.Password, c.Host, c.Port)
}
```

在 `Load()` 函数返回的 `Config{}` 中添加：

```go
RabbitMQ: RabbitMQConfig{
    Host:     viper.GetString("RABBITMQ_HOST"),
    Port:     viper.GetString("RABBITMQ_PORT"),
    User:     viper.GetString("RABBITMQ_USER"),
    Password: viper.GetString("RABBITMQ_PASSWORD"),
},
```

需要在 config.go 顶部增加 `"fmt"` import。

- [ ] **Step 4: Commit**

```bash
git add docker-compose.yml .env internal/config/config.go
git commit -m "config: 新增 RabbitMQ 配置和服务

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
```

---

### Task 3: UUID 工具

**Files:**
- Create: `internal/utils/uuid.go`

- [ ] **Step 1: 创建文件**

```go
package utils

import (
    "crypto/rand"
    "encoding/hex"
)

func NewUUID() string {
    b := make([]byte, 16)
    rand.Read(b)
    b[6] = (b[6] & 0x0f) | 0x40
    b[8] = (b[8] & 0x3f) | 0x80
    return hex.EncodeToString(b)
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/utils/uuid.go
git commit -m "feat: 新增 UUID 生成工具

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
```

---

### Task 4: WebSocket Hub 和 Upgrader

**Files:**
- Create: `internal/ws/hub.go`
- Create: `internal/ws/upgrader.go`

- [ ] **Step 1: 创建 upgrader.go**

```go
package ws

import (
    "net/http"

    "github.com/gorilla/websocket"
)

var Upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}
```

- [ ] **Step 2: 创建 hub.go**

```go
package ws

import (
    "sync"

    "github.com/gorilla/websocket"
)

type Message struct {
    Type        string `json:"type"`
    TaskID      string `json:"task_id,omitempty"`
    Filename    string `json:"filename,omitempty"`
    DownloadURL string `json:"download_url,omitempty"`
    Error       string `json:"error,omitempty"`
}

type userConn struct {
    conn *websocket.Conn
}

type Hub struct {
    mu    sync.RWMutex
    conns map[uint]*userConn
}

func NewHub() *Hub {
    return &Hub{conns: make(map[uint]*userConn)}
}

func (h *Hub) Register(userID uint, conn *websocket.Conn) {
    h.mu.Lock()
    defer h.mu.Unlock()
    if old, ok := h.conns[userID]; ok {
        old.conn.Close()
    }
    h.conns[userID] = &userConn{conn: conn}
}

func (h *Hub) Unregister(userID uint) {
    h.mu.Lock()
    defer h.mu.Unlock()
    delete(h.conns, userID)
}

func (h *Hub) Send(userID uint, msg Message) error {
    h.mu.RLock()
    uc, ok := h.conns[userID]
    h.mu.RUnlock()
    if !ok {
        return nil // 用户不在线，静默忽略
    }
    uc.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
    return uc.conn.WriteJSON(msg)
}
```

需要在 hub.go 顶部增加 `"time"` import。

- [ ] **Step 4: Commit**

```bash
git add internal/ws/
git commit -m "feat: 新增 WebSocket Hub 和 Upgrader

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
```

---

### Task 5: WebSocket Controller

**Files:**
- Create: `internal/controller/ws_controller.go`

- [ ] **Step 1: 创建文件**

```go
package controller

import (
    "context"
    "time"

    "ginproject/internal/config"
    "ginproject/internal/utils"
    "ginproject/internal/ws"

    "github.com/gin-gonic/gin"
    "github.com/redis/go-redis/v9"
)

type WSController struct {
    hub *ws.Hub
    rdb *redis.Client
    cfg *config.Config
}

func NewWSController(hub *ws.Hub, rdb *redis.Client, cfg *config.Config) *WSController {
    return &WSController{hub: hub, rdb: rdb, cfg: cfg}
}

func (ctl *WSController) Handle(c *gin.Context) {
    token := c.Query("token")
    if token == "" {
        c.JSON(401, gin.H{"code": 401, "message": "缺少token"})
        return
    }

    // 检查黑名单
    _, err := ctl.rdb.Get(context.Background(), "blacklist:"+token).Result()
    if err == nil {
        c.JSON(401, gin.H{"code": 401, "message": "Token已失效"})
        return
    }

    claims, err := utils.ParseToken(token, ctl.cfg.JWT.Secret)
    if err != nil {
        c.JSON(401, gin.H{"code": 401, "message": "Token无效"})
        return
    }

    conn, err := ws.Upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        return
    }

    userID := claims.UserID
    ctl.hub.Register(userID, conn)
    defer ctl.hub.Unregister(userID)
    defer conn.Close()

    stopCh := make(chan struct{})

    // 读协程：检测客户端断开
    go func() {
        defer close(stopCh)
        for {
            if _, _, err := conn.NextReader(); err != nil {
                break
            }
        }
    }()

    // 心跳
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    conn.SetPongHandler(func(string) error {
        conn.SetReadDeadline(time.Now().Add(60 * time.Second))
        return nil
    })

    for {
        select {
        case <-ticker.C:
            conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if err := conn.WriteJSON(ws.Message{Type: "heartbeat"}); err != nil {
                return
            }
        case <-stopCh:
            return
        }
    }
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/controller/ws_controller.go
git commit -m "feat: 新增 WebSocket 控制器

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
```

---

### Task 6: 分批查询支持

**Files:**
- Modify: `internal/dao/log_dao.go`
- Modify: `internal/service/log_service.go`

- [ ] **Step 1: 在 log_dao.go 新增 FindBatch**

在现有的 `FindAll` 方法后添加：

```go
func (d *LogDAO) FindBatch(module, method string, offset, limit int) ([]model.OperationLog, error) {
    var logs []model.OperationLog
    q := d.db.Model(&model.OperationLog{})
    if module != "" {
        q = q.Where("module = ?", module)
    }
    if method != "" {
        q = q.Where("method = ?", method)
    }
    err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&logs).Error
    return logs, err
}
```

- [ ] **Step 2: 在 log_service.go 新增 FindBatch**

```go
func (s *LogService) FindBatch(module, method string, offset, limit int) ([]model.OperationLog, error) {
    return s.logDAO.FindBatch(module, method, offset, limit)
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/dao/log_dao.go internal/service/log_service.go
git commit -m "feat: 新增操作日志分批查询方法

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
```

---

### Task 7: 导出 Worker

**Files:**
- Create: `internal/worker/export_worker.go`

- [ ] **Step 1: 创建文件**

```go
package worker

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "time"

    "ginproject/internal/model"
    "ginproject/internal/service"
    "ginproject/internal/ws"

    "github.com/rabbitmq/amqp091-go"
    "github.com/redis/go-redis/v9"
    "github.com/xuri/excelize/v2"
)

const (
    queueName  = "excel.export"
    taskPrefix = "excel:task:"
    exportDir  = "exports"
    taskTTL    = 24 * time.Hour
    batchSize  = 5000
)

type ExportWorker struct {
    rdb        *redis.Client
    amqpConn   *amqp091.Connection
    logService *service.LogService
    hub        *ws.Hub
}

type queueMessage struct {
    TaskID string `json:"task_id"`
}

func NewExportWorker(rdb *redis.Client, amqpConn *amqp091.Connection, logService *service.LogService, hub *ws.Hub) *ExportWorker {
    return &ExportWorker{rdb: rdb, amqpConn: amqpConn, logService: logService, hub: hub}
}

func (w *ExportWorker) Start() {
    os.MkdirAll(exportDir, 0755)

    ch, err := w.amqpConn.Channel()
    if err != nil {
        panic("RabbitMQ channel 创建失败: " + err.Error())
    }
    defer ch.Close()

    q, err := ch.QueueDeclare(queueName, true, false, false, false, nil)
    if err != nil {
        panic("RabbitMQ 队列声明失败: " + err.Error())
    }

    ch.Qos(1, 0, false)

    msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
    if err != nil {
        panic("RabbitMQ 消费注册失败: " + err.Error())
    }

    for msg := range msgs {
        var qm queueMessage
        if json.Unmarshal(msg.Body, &qm) == nil {
            w.processTask(qm.TaskID)
        }
        msg.Ack(false)
    }
}

func (w *ExportWorker) processTask(taskID string) {
    taskKey := taskPrefix + taskID
    ctx := context.Background()

    w.rdb.HSet(ctx, taskKey, "status", "processing")

    userIDStr, _ := w.rdb.HGet(ctx, taskKey, "user_id").Result()
    module, _ := w.rdb.HGet(ctx, taskKey, "module").Result()
    method, _ := w.rdb.HGet(ctx, taskKey, "method").Result()

    var uid uint
    fmt.Sscanf(userIDStr, "%d", &uid)

    filename := fmt.Sprintf("操作日志_%s.xlsx", time.Now().Format("20060102_150405"))
    filePath := filepath.Join(exportDir, taskID+".xlsx")

    err := w.buildExcel(module, method, filePath)
    if err != nil {
        w.rdb.HSet(ctx, taskKey, "status", "failed", "error", err.Error())
        w.hub.Send(uid, ws.Message{
            Type:   "export_failed",
            TaskID: taskID,
            Error:  err.Error(),
        })
        return
    }

    w.rdb.HSet(ctx, taskKey,
        "status", "success",
        "filename", filename,
    )
    w.rdb.Expire(ctx, taskKey, taskTTL)

    w.hub.Send(uid, ws.Message{
        Type:        "export_complete",
        TaskID:      taskID,
        Filename:    filename,
        DownloadURL: "/api/logs/download/" + taskID,
    })
}

func (w *ExportWorker) buildExcel(module, method, filePath string) error {
    f := excelize.NewFile()
    defer f.Close()

    sheet := "操作日志"
    f.SetSheetName("Sheet1", sheet)

    headers := []string{"ID", "操作人ID", "请求方式", "请求路径", "参数", "耗时(ms)", "IP", "操作时间"}
    for i, h := range headers {
        cell, _ := excelize.CoordinatesToCellName(i+1, 1)
        f.SetCellValue(sheet, cell, h)
    }

    headerStyle, _ := f.NewStyle(&excelize.Style{
        Font: &excelize.Font{Bold: true},
        Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
    })
    f.SetCellStyle(sheet, "A1", fmt.Sprintf("%c1", 'A'+len(headers)-1), headerStyle)

    sw, err := f.NewStreamWriter(sheet)
    if err != nil {
        return err
    }

    offset := 0
    row := 2

    for {
        logs, err := w.logService.FindBatch(module, method, offset, batchSize)
        if err != nil {
            return err
        }
        if len(logs) == 0 {
            break
        }

        for _, log := range logs {
            cell, _ := excelize.CoordinatesToCellName(1, row)
            values := []interface{}{
                log.ID, log.OperatorID, log.Method, log.Path,
                log.Params, log.Duration, log.IP,
                time.Time(log.CreatedAt).Format("2006-01-02 15:04:05"),
            }
            if err := sw.SetRow(cell, values); err != nil {
                return err
            }
            row++
        }

        offset += batchSize
        if len(logs) < batchSize {
            break
        }
    }

    if err := sw.Flush(); err != nil {
        return err
    }

    return f.SaveAs(filePath)
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/worker/export_worker.go
git commit -m "feat: 新增异步导出 Worker（RabbitMQ 消费 + 流式生成 Excel）

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
```

---

### Task 8: 改造 LogController

**Files:**
- Modify: `internal/controller/log_controller.go`

- [ ] **Step 1: 重写文件**

```go
package controller

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "time"

    "ginproject/internal/service"
    "ginproject/internal/utils"

    "github.com/gin-gonic/gin"
    "github.com/rabbitmq/amqp091-go"
    "github.com/redis/go-redis/v9"
)

type LogController struct {
    logService *service.LogService
    rdb        *redis.Client
    amqpCh     *amqp091.Channel
}

func NewLogController(logService *service.LogService, rdb *redis.Client, amqpCh *amqp091.Channel) *LogController {
    return &LogController{logService: logService, rdb: rdb, amqpCh: amqpCh}
}

func (ctl *LogController) List(c *gin.Context) {
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
    module := c.Query("module")
    method := c.Query("method")
    logs, total, err := ctl.logService.FindPage(page, pageSize, module, method)
    if err != nil {
        utils.Error(c, 500, err.Error())
        return
    }
    utils.Success(c, gin.H{"list": logs, "total": total})
}

func (ctl *LogController) Export(c *gin.Context) {
    var req struct {
        Module string `json:"module"`
        Method string `json:"method"`
    }
    c.ShouldBindJSON(&req)
    userID, _ := c.Get("user_id")

    taskID := utils.NewUUID()
    taskKey := "excel:task:" + taskID

    ctx := context.Background()
    ctl.rdb.HSet(ctx, taskKey,
        "status", "pending",
        "user_id", fmt.Sprintf("%d", userID),
        "module", req.Module,
        "method", req.Method,
    )
    ctl.rdb.Expire(ctx, taskKey, 24*time.Hour)

    body, _ := json.Marshal(map[string]string{"task_id": taskID})
    ctl.amqpCh.Publish("", "excel.export", false, false, amqp091.Publishing{
        ContentType: "application/json",
        Body:        body,
    })

    utils.Success(c, gin.H{"task_id": taskID})
}

func (ctl *LogController) ExportStatus(c *gin.Context) {
    taskID := c.Query("task_id")
    if taskID == "" {
        utils.Error(c, 400, "缺少task_id")
        return
    }
    fields, err := ctl.rdb.HGetAll(context.Background(), "excel:task:"+taskID).Result()
    if err != nil || len(fields) == 0 {
        utils.Error(c, 404, "任务不存在或已过期")
        return
    }
    utils.Success(c, fields)
}

func (ctl *LogController) Download(c *gin.Context) {
    taskID := c.Param("taskID")
    taskKey := "excel:task:" + taskID

    userID, _ := c.Get("user_id")
    taskUserID, err := ctl.rdb.HGet(context.Background(), taskKey, "user_id").Result()
    if err != nil || taskUserID != fmt.Sprintf("%d", userID) {
        utils.Error(c, 403, "无权下载或任务不存在")
        return
    }

    status, _ := ctl.rdb.HGet(context.Background(), taskKey, "status").Result()
    if status != "success" {
        utils.Error(c, 400, "文件尚未生成或已失败")
        return
    }

    filename, _ := ctl.rdb.HGet(context.Background(), taskKey, "filename").Result()
    filePath := filepath.Join("exports", taskID+".xlsx")

    if _, err := os.Stat(filePath); os.IsNotExist(err) {
        utils.Error(c, 404, "文件已被下载或不存在")
        return
    }

    c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
    c.File(filePath)

    // 下载后删除
    os.Remove(filePath)
}
```

需要新增 import：`"context"`, `"encoding/json"`, `"fmt"`, `"os"`, `"path/filepath"`, `"strconv"`, `"time"`, `"github.com/rabbitmq/amqp091-go"`, `"github.com/redis/go-redis/v9"`。

删除旧的 import：`"github.com/xuri/excelize/v2"`。

- [ ] **Step 2: Commit**

```bash
git add internal/controller/log_controller.go
git commit -m "feat: 改造 LogController 支持异步导出

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
```

---

### Task 9: 修改路由

**Files:**
- Modify: `internal/router/router.go`

- [ ] **Step 1: 修改 Setup 函数签名，增加 wsCtrl 参数**

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
    authService *service.AuthService,
    userDAO *dao.UserDAO,
    menuDAO *dao.MenuDAO,
    logDAO *dao.LogDAO,
) *gin.Engine {
```

- [ ] **Step 2: 替换老的 Export 路由，新增路由**

在 authorized 组内，替换原来的 `GET /logs/export` 行（原来的第 85-86 行），改为：

```go
            // 操作日志
            authorized.GET("/logs",
                middleware.RequirePerm("log:list"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO), logCtrl.List)
            authorized.POST("/logs/export",
                middleware.RequirePerm("log:export"), middleware.RBAC(menuDAO), logCtrl.Export)
            authorized.GET("/logs/export-status",
                middleware.RequirePerm("log:export"), middleware.RBAC(menuDAO), logCtrl.ExportStatus)
            authorized.GET("/logs/download/:taskID",
                middleware.RequirePerm("log:export"), middleware.RBAC(menuDAO), logCtrl.Download)
```

- [ ] **Step 3: 在 authorized 组外添加 WebSocket 路由**

在 `return r` 之前添加：

```go
    // WebSocket
    r.GET("/api/ws", wsCtrl.Handle)
```

- [ ] **Step 4: Commit**

```bash
git add internal/router/router.go
git commit -m "feat: 路由增加异步导出和 WebSocket 端点

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
```

---

### Task 10: 连接 main.go

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: 增加依赖注入和 Worker 启动**

需要在 `main()` 中，在现有 Service 初始化之后、router.Setup 之前，增加以下代码：

```go
    // --- RabbitMQ ---
    amqpConn, err := amqp.Dial(cfg.RabbitMQ.DSN())
    if err != nil {
        log.Fatalf("RabbitMQ 连接失败: %v", err)
    }
    defer amqpConn.Close()

    publishCh, err := amqpConn.Channel()
    if err != nil {
        log.Fatalf("RabbitMQ Channel 创建失败: %v", err)
    }
    defer publishCh.Close()

    // --- WebSocket Hub ---
    hub := ws.NewHub()
    wsCtrl := controller.NewWSController(hub, rdb, cfg)

    // --- Export Worker ---
    exportWorker := worker.NewExportWorker(rdb, amqpConn, logService, hub)
    go exportWorker.Start()
```

- [ ] **Step 2: 修改 LogController 和 router.Setup 调用**

修改 logCtrl 构造：

```go
    logCtrl := controller.NewLogController(logService, rdb, publishCh)
```

修改 router.Setup 调用，增加 `wsCtrl` 参数：

```go
    r := router.Setup(cfg, authCtrl, userCtrl, roleCtrl, menuCtrl, logCtrl, loginLogCtrl, wsCtrl, authService, userDAO, menuDAO, logDAO)
```

- [ ] **Step 3: 增加 import**

需要在 import 块中新增：

```go
    "ginproject/internal/ws"
    "ginproject/internal/worker"

    "github.com/rabbitmq/amqp091-go"
```

- [ ] **Step 4: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: main.go 串联 RabbitMQ、WebSocket Hub 和 Worker

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
```

---

### Task 11: 前端 WebSocket 工具

**Files:**
- Create: `web/src/utils/ws.js`

- [ ] **Step 1: 创建文件**

```js
let ws = null
let reconnectTimer = null
const handlers = {}

export function connectWS(token) {
  if (ws && ws.readyState === WebSocket.OPEN) return

  const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const url = `${protocol}//${location.host}/api/ws?token=${encodeURIComponent(token)}`

  ws = new WebSocket(url)

  ws.onopen = () => {}

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      if (msg.type === 'heartbeat') {
        ws.send(JSON.stringify({ type: 'pong' }))
        return
      }
      const cbs = handlers[msg.type]
      if (cbs) cbs.forEach(cb => cb(msg))
    } catch (e) {
      // ignore
    }
  }

  ws.onclose = () => {
    ws = null
    reconnectTimer = setTimeout(() => connectWS(token), 5000)
  }

  ws.onerror = () => {
    ws.close()
  }
}

export function disconnectWS() {
  if (reconnectTimer) clearTimeout(reconnectTimer)
  if (ws) ws.close()
  ws = null
}

export function onWSMessage(type, callback) {
  if (!handlers[type]) handlers[type] = []
  handlers[type].push(callback)
}

export function offWSMessage(type, callback) {
  if (!handlers[type]) return
  handlers[type] = handlers[type].filter(cb => cb !== callback)
}
```

- [ ] **Step 2: Commit**

```bash
git add web/src/utils/ws.js
git commit -m "feat: 新增前端 WebSocket 管理器

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
```

---

### Task 12: 前端 Store 和 Layout 接入 WebSocket

**Files:**
- Modify: `web/src/store/modules/user.js`
- Modify: `web/src/layout/index.vue`

- [ ] **Step 1: 在 user.js 的 login action 中连接 WS**

在文件顶部增加 import：

```js
import { connectWS } from '@/utils/ws'
```

在 `login` action 中，`return res` 之前增加：

```js
    connectWS(res.data.token)
```

- [ ] **Step 2: 在 layout/index.vue 中连接和断开 WS**

在 `<script>` 块中增加 import：

```js
import { connectWS, disconnectWS } from '@/utils/ws'
```

在 `created()` 中增加：

```js
    const token = this.$store.state.user.token
    if (token) connectWS(token)
```

在 `methods` 中增加对 logout 的监听。修改 `handleCommand` 中 `logout` 分支，在 dispatch 前先断开：

实际上，更优雅的方式是在 `beforeDestroy` 中断开。在 `created()` 后添加：

```js
  beforeDestroy() {
    disconnectWS()
  }
```

同时在 `handleCommand` 的 `logout` 分支增加 `disconnectWS()`：

```js
      } else if (cmd === 'logout') {
        disconnectWS()
        await this.$store.dispatch('user/logout')
        this.$router.push('/login')
      }
```

- [ ] **Step 3: Commit**

```bash
git add web/src/store/modules/user.js web/src/layout/index.vue
git commit -m "feat: Layout 和 Store 接入 WebSocket 连接生命周期

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
```

---

### Task 13: 前端导出页面改造

**Files:**
- Modify: `web/src/api/log.js`
- Modify: `web/src/views/log/index.vue`

- [ ] **Step 1: 修改 api/log.js**

将 `exportLogs` 函数替换为：

```js
export const exportLogs = (params) => request.post('/logs/export', params)

export const getExportStatus = (taskId) => request.get('/logs/export-status', { params: { task_id: taskId } })
```

删除旧的 blob 下载逻辑（`export const exportLogs = (params) => { ... }` 整个替换掉）。

文件最终内容：

```js
import request from './request'

export const getLogs = (params) => request.get('/logs', { params })

export const getLoginLogs = (params) => request.get('/login-logs', { params })

export const exportLogs = (params) => request.post('/logs/export', params)

export const getExportStatus = (taskId) => request.get('/logs/export-status', { params: { task_id: taskId } })
```

- [ ] **Step 2: 修改 views/log/index.vue — data 和 template**

在 `data()` 中增加：

```js
      exporting: false,
      currentTaskId: null,
      downloadUrl: null,
      downloadFilename: null,
```

在 `<el-form-item><el-button type="primary" @click="exportExcel">导出Excel</el-button></el-form-item>` 改为：

```html
        <el-form-item>
          <el-button type="primary" :disabled="exporting" @click="exportExcel">
            {{ exporting ? '导出中...' : '导出Excel' }}
          </el-button>
        </el-form-item>
```

在 `</el-form>` 之后、`<el-table` 之前，增加下载提示：

```html
      <el-alert
        v-if="downloadUrl"
        title="导出完成"
        type="success"
        :closable="true"
        @close="downloadUrl = null"
        style="margin-bottom:15px"
      >
        <a :href="downloadUrl">{{ downloadFilename || '点击下载' }}</a>
      </el-alert>
```

- [ ] **Step 3: 修改 views/log/index.vue — methods 和 lifecycle**

替换 `exportExcel` 方法：

```js
    exportExcel() {
      this.exporting = true
      this.downloadUrl = null
      this.startPolling = false
      exportLogs({ method: this.filters.method }).then(res => {
        this.currentTaskId = res.data.task_id
        this._pollTimer = setInterval(() => {
          if (!this.currentTaskId) { clearInterval(this._pollTimer); return }
          getExportStatus(this.currentTaskId).then(r => {
            const d = r.data
            if (d.status === 'success') {
              this.onExportComplete({
                task_id: this.currentTaskId,
                filename: d.filename,
                download_url: '/api/logs/download/' + this.currentTaskId
              })
            } else if (d.status === 'failed') {
              this.onExportFailed({ task_id: this.currentTaskId, error: d.error || '导出失败' })
            }
          }).catch(() => {})
        }, 2000)
      }).catch(() => {
        this.exporting = false
        this.$message.error('导出请求失败')
      })
    },
    onExportComplete(msg) {
      if (msg.task_id !== this.currentTaskId) return
      this.exporting = false
      clearInterval(this._pollTimer)
      this.downloadUrl = msg.download_url
      this.downloadFilename = msg.filename
    },
    onExportFailed(msg) {
      if (msg.task_id !== this.currentTaskId) return
      this.exporting = false
      clearInterval(this._pollTimer)
      this.$message.error(msg.error || '导出失败')
    },
```

在 `export default {` 中增加 `mounted` 和 `beforeDestroy`：

```js
  mounted() {
    this._onComplete = this.onExportComplete.bind(this)
    this._onFailed = this.onExportFailed.bind(this)
    onWSMessage('export_complete', this._onComplete)
    onWSMessage('export_failed', this._onFailed)
  },
  beforeDestroy() {
    offWSMessage('export_complete', this._onComplete)
    offWSMessage('export_failed', this._onFailed)
    if (this._pollTimer) clearInterval(this._pollTimer)
  },
```

在 import 行增加：

```js
import { onWSMessage, offWSMessage } from '@/utils/ws'
```

将 `import { getLogs, exportLogs } from '@/api/log'` 改为：

```js
import { getLogs, exportLogs, getExportStatus } from '@/api/log'
```

- [ ] **Step 4: Commit**

```bash
git add web/src/api/log.js web/src/views/log/index.vue
git commit -m "feat: 操作日志导出改为异步模式（WS 通知 + 轮询降级）

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
```

---

### Task 14: .gitignore

**Files:**
- Modify: `.gitignore`

- [ ] **Step 1: 添加 exports/**

在 `.gitignore` 末尾添加一行：

```
exports/
```

- [ ] **Step 2: Commit**

```bash
git add .gitignore
git commit -m "chore: gitignore 增加导出目录

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
```

---

### Task 15: 构建和验证

- [ ] **Step 1: 启动 RabbitMQ**

```bash
docker compose up -d rabbitmq
```

等待 rabbitmq 健康检查通过（约 30 秒）。

- [ ] **Step 2: 重新构建并启动**

```bash
docker compose up -d --build go-app nginx
```

- [ ] **Step 3: 检查服务状态**

```bash
docker compose ps
```

确认 go-app、nginx、rabbitmq、mysql、redis 都是 Up 状态。

- [ ] **Step 4: 验证功能**

1. 打开 `http://localhost:8080`，登录 admin/admin
2. 进入"操作日志"页面
3. 点击"导出Excel" → 按钮应变为"导出中..."并禁用
4. 等待 WebSocket 推送完成通知 → 页面出现下载链接
5. 点击下载 → 浏览器下载 xlsx 文件、内容正确
6. 再次点击同一个下载链接 → 应返回"文件已被下载或不存在"
7. 用不同浏览器登录两个用户 → 同时导出 → 互不干扰
8. 打开 `http://localhost:15672`（guest/guest）→ 查看 excel.export 队列状态

- [ ] **Step 5: 检查 go-app 日志**

```bash
docker compose logs go-app | tail -20
```

确认没有 panic 或错误。
