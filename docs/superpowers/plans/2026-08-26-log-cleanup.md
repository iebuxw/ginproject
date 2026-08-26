# 日志定时清理 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 新增日志清理 HTTP 端点（secret 校验、按保留天数分批删除 operation_logs/login_logs + ES 同步清理），配合现有定时任务模块在管理页配置一条任务即可定时清理旧日志。

**Architecture:** 复用现有 HTTP 回调式定时任务模块，零调度器改动。新增公开路由 `POST /api/logs/cleanup`（调度器发请求不带 JWT，防滥用靠 `LOG_CLEANUP_SECRET` 比对）。按 `created_at` 分批 LIMIT 1000 循环删除防大表锁，ES 用 `delete_by_query` 同步清理且降级不阻断。迁移 000006 为两表 created_at 加索引。

**Tech Stack:** Go 1.x + Gin + GORM + go-elasticsearch v7 + golang-migrate + robfig/cron（现有，无新依赖）

**验证模式说明：** 项目无测试/CI 基建（见 CLAUDE.md），本计划以 `go build` 编译验证 + curl 手工回归为验收手段，不引入测试框架。

---

### Task 1: 迁移 000006 加 created_at 索引

**Files:**
- Create: `migrations/000006_add_logs_created_at_index.up.sql`
- Create: `migrations/000006_add_logs_created_at_index.down.sql`

- [ ] **Step 1: 创建 up 迁移**

`migrations/000006_add_logs_created_at_index.up.sql`：

```sql
-- 日志按 created_at 清理/查询索引（MySQL 不支持 CREATE INDEX IF NOT EXISTS，迁移只执行一次，幂等性由 golang-migrate 保证）
CREATE INDEX idx_operation_logs_created_at ON operation_logs (created_at);
CREATE INDEX idx_login_logs_created_at ON login_logs (created_at);
```

- [ ] **Step 2: 创建 down 迁移**

`migrations/000006_add_logs_created_at_index.down.sql`：

```sql
DROP INDEX idx_operation_logs_created_at ON operation_logs;
DROP INDEX idx_login_logs_created_at ON login_logs;
```

- [ ] **Step 3: 提交**

```bash
git add migrations/000006_add_logs_created_at_index.up.sql migrations/000006_add_logs_created_at_index.down.sql
git commit -m "feat: 迁移 000006 为日志表 created_at 加索引（支撑按时间清理）"
```

---

### Task 2: 配置项 LOG_CLEANUP_SECRET

**Files:**
- Modify: `internal/config/config.go`
- Modify: `.env.example`
- Modify: `.env`（gitignore，本地 go run 需要）

- [ ] **Step 1: config.go 加字段**

`internal/config/config.go` 的 `Config` 结构体（第 9-17 行）加一行：

```go
type Config struct {
	Server        ServerConfig
	Database      DatabaseConfig
	Redis         RedisConfig
	JWT           JWTConfig
	RabbitMQ      RabbitMQConfig
	Mail          MailConfig
	Elasticsearch ElasticsearchConfig
	LogCleanupSecret string
}
```

`Load()` 返回值（第 77-115 行）加一项：

```go
	return &Config{
		Server: ServerConfig{Port: viper.GetString("SERVER_PORT")},
		// ...原有项不动...
		Elasticsearch: ElasticsearchConfig{
			Host:     viper.GetString("ES_HOST"),
			Port:     viper.GetString("ES_PORT"),
			Username: viper.GetString("ES_USERNAME"),
			Password: viper.GetString("ES_PASSWORD"),
		},
		LogCleanupSecret: viper.GetString("LOG_CLEANUP_SECRET"),
	}
```

- [ ] **Step 2: .env.example 与 .env 各加一行**（末尾追加）

```
# 日志定时清理接口密钥（公开路由防滥用，定时任务 URL 中携带）
LOG_CLEANUP_SECRET=请修改为随机字符串
```

`.env` 中填一个实际随机字符串（本机 go run 与 docker 需一致，否则定时任务调清理接口会 403）。

- [ ] **Step 3: 提交**

```bash
git add internal/config/config.go .env.example
git commit -m "feat: 配置项新增 LOG_CLEANUP_SECRET（日志清理接口密钥）"
```

（`.env` 已 gitignore，无需 add）

---

### Task 3: DAO 层按时间分批删除

**Files:**
- Modify: `internal/dao/log_dao.go`
- Modify: `internal/dao/login_log_dao.go`

- [ ] **Step 1: LogDAO 加 DeleteOlderThan**

`internal/dao/log_dao.go` 末尾（`FindBatch` 之后）加：

```go
// DeleteOlderThan 删除创建时间早于 before 的日志，最多删除 limit 条；返回实际删除行数
func (d *LogDAO) DeleteOlderThan(before time.Time, limit int) (int64, error) {
	res := d.db.Where("created_at < ?", before).Limit(limit).Delete(&model.OperationLog{})
	return res.RowsAffected, res.Error
}
```

文件头 import 增加 `"time"`。

- [ ] **Step 2: LoginLogDAO 加 DeleteOlderThan**

`internal/dao/login_log_dao.go` 末尾（`FindPage` 之后）加：

```go
// DeleteOlderThan 删除创建时间早于 before 的登录日志，最多删除 limit 条；返回实际删除行数
func (d *LoginLogDAO) DeleteOlderThan(before time.Time, limit int) (int64, error) {
	res := d.db.Where("created_at < ?", before).Limit(limit).Delete(&model.LoginLog{})
	return res.RowsAffected, res.Error
}
```

文件头 import 增加 `"time"`。

- [ ] **Step 3: 编译验证**

```bash
go build ./internal/dao/
```

预期：无输出（编译通过）。

- [ ] **Step 4: 提交**

```bash
git add internal/dao/log_dao.go internal/dao/login_log_dao.go
git commit -m "feat: 日志 DAO 新增按时间分批删除方法"
```

---

### Task 4: ES 按时间删除旧文档

**Files:**
- Modify: `internal/es/log_repo.go`

- [ ] **Step 1: 加 DeleteByTime**

`internal/es/log_repo.go` 末尾（`sanitizeHighlight` 之后）加：

```go
// DeleteByTime 删除创建时间早于 before 的操作日志文档（delete_by_query），返回删除条数
func (r *LogRepo) DeleteByTime(before time.Time) (int64, error) {
	if r.cli == nil {
		return 0, fmt.Errorf("ES 未启用")
	}
	body := map[string]interface{}{
		"query": map[string]interface{}{
			"range": map[string]interface{}{
				"created_at": map[string]interface{}{
					"lt": before.Format("2006-01-02 15:04:05"),
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return 0, err
	}
	refresh := true
	res, err := (&esapi.DeleteByQueryRequest{
		Index:   []string{LogIndex},
		Body:    bytes.NewReader(buf.Bytes()),
		Refresh: &refresh,
	}).Do(context.Background(), r.cli.RawClient())
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return 0, fmt.Errorf("删除失败: %s", res.String())
	}
	var parsed struct {
		Deleted int64 `json:"deleted"`
	}
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return 0, err
	}
	return parsed.Deleted, nil
}
```

文件头 import 增加 `"time"`（现有：bytes/context/encoding/json/fmt/html/log/strings + ginproject/internal/model + esapi；`fmt`、`bytes`、`json`、`context`、`esapi` 均已存在）。

- [ ] **Step 2: 编译验证**

```bash
go build ./internal/es/
```

预期：无输出。若 `Refresh` 字段类型报错（该字段在 go-elasticsearch v7 中为 `*bool`，已确认），不要改结构体，用局部变量 `refresh := true` 传址。

- [ ] **Step 3: 提交**

```bash
git add internal/es/log_repo.go
git commit -m "feat: ES LogRepo 新增按时间删除旧操作日志文档"
```

---

### Task 5: Service 层清理编排

**Files:**
- Modify: `internal/service/log_service.go`
- Modify: `internal/service/login_log_service.go`

- [ ] **Step 1: LogService 加 Cleanup（含 ES 降级）**

`internal/service/log_service.go` 末尾（`FindBatch` 之后）加：

```go
// Cleanup 分批删除保留天数之外的操作日志，并同步清理 ES；ES 不可用仅告警不阻断
func (s *LogService) Cleanup(days int) (int64, error) {
	before := time.Now().AddDate(0, 0, -days)
	var total int64
	for {
		n, err := s.logDAO.DeleteOlderThan(before, 1000)
		if err != nil {
			return total, err
		}
		total += n
		if n == 0 {
			break
		}
	}
	if n, err := s.logRepo.DeleteByTime(before); err != nil {
		log.Printf("ES 清理旧操作日志失败（已降级仅清 MySQL）: %v", err)
	} else if n > 0 {
		log.Printf("ES 已清理 %d 条旧操作日志", n)
	}
	return total, nil
}
```

文件头 import 增加 `"log"`、`"time"`。

- [ ] **Step 2: LoginLogService 加 Cleanup**

`internal/service/login_log_service.go` 末尾（`FindPage` 之后）加：

```go
// Cleanup 分批删除保留天数之外的登录日志
func (s *LoginLogService) Cleanup(days int) (int64, error) {
	before := time.Now().AddDate(0, 0, -days)
	var total int64
	for {
		n, err := s.loginLogDAO.DeleteOlderThan(before, 1000)
		if err != nil {
			return total, err
		}
		total += n
		if n == 0 {
			break
		}
	}
	return total, nil
}
```

文件头 import 增加 `"time"`。

- [ ] **Step 3: 编译验证**

```bash
go build ./internal/service/
```

预期：无输出。

- [ ] **Step 4: 提交**

```bash
git add internal/service/log_service.go internal/service/login_log_service.go
git commit -m "feat: 日志 Service 新增清理编排（分批删除 + ES 降级）"
```

---

### Task 6: Controller 清理处理器

**Files:**
- Modify: `internal/controller/log_controller.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: 修改 LogController 结构与构造**

`internal/controller/log_controller.go`：

```go
type LogController struct {
	logService      *service.LogService
	loginLogService *service.LoginLogService
	rdb             *redis.Client
	amqpCh          *amqp091.Channel
	cleanupSecret   string
}

func NewLogController(logService *service.LogService, loginLogService *service.LoginLogService, rdb *redis.Client, amqpCh *amqp091.Channel, cleanupSecret string) *LogController {
	return &LogController{logService: logService, loginLogService: loginLogService, rdb: rdb, amqpCh: amqpCh, cleanupSecret: cleanupSecret}
}
```

- [ ] **Step 2: 加 Cleanup 处理器**（放在 `Download` 方法之后）

```go
// Cleanup 定时清理旧日志（公开路由，secret 校验）
// @Summary 清理旧日志
// @Description 按保留天数分批删除旧日志（操作日志/登录日志），ES 同步清理。供定时任务调用，需携带 secret
// @Tags 操作日志
// @Produce json
// @Param secret query string true "清理密钥（与 LOG_CLEANUP_SECRET 比对）"
// @Param days query int true "保留天数（删除创建时间早于 now-days 的日志）"
// @Param scope query string false "清理范围：operation/login/all，默认 all"
// @Success 200 {object} utils.Response{data=object{operation_deleted=int,login_deleted=int}} "成功"
// @Failure 200 {object} utils.Response "参数非法"
// @Router /logs/cleanup [post]
func (ctl *LogController) Cleanup(c *gin.Context) {
	secret := c.Query("secret")
	if ctl.cleanupSecret == "" || secret != ctl.cleanupSecret {
		utils.ErrorWithStatus(c, http.StatusForbidden, 403, "密钥无效")
		return
	}
	days, err := strconv.Atoi(c.DefaultQuery("days", "30"))
	if err != nil || days < 1 || days > 3650 {
		utils.Error(c, 400, "days 参数非法（1~3650）")
		return
	}
	scope := c.DefaultQuery("scope", "all")
	if scope != "operation" && scope != "login" && scope != "all" {
		utils.Error(c, 400, "scope 参数非法（operation/login/all）")
		return
	}
	result := gin.H{}
	if scope == "operation" || scope == "all" {
		n, err := ctl.logService.Cleanup(days)
		if err != nil {
			utils.Error(c, 500, "操作日志清理失败: "+err.Error())
			return
		}
		result["operation_deleted"] = n
	}
	if scope == "login" || scope == "all" {
		n, err := ctl.loginLogService.Cleanup(days)
		if err != nil {
			utils.Error(c, 500, "登录日志清理失败: "+err.Error())
			return
		}
		result["login_deleted"] = n
	}
	utils.Success(c, result)
}
```

文件头 import 增加 `"net/http"`（现有：context/encoding/json/fmt/log/os/path/filepath/strconv/time + ginproject 三件 + gin/amqp091/redis；`strconv`、`gin` 已有）。

- [ ] **Step 3: main.go 接线**

`cmd/server/main.go` 第 126 行：

```go
	logCtrl := controller.NewLogController(logService, loginLogService, rdb, publishCh, cfg.LogCleanupSecret)
```

（原来只有 logService/rdb/publishCh 三个参数，注意 loginLogService 变量已存在，见第 87 行）

- [ ] **Step 4: 编译验证**

```bash
go build ./...
```

预期：无输出。

- [ ] **Step 5: 提交**

```bash
git add internal/controller/log_controller.go cmd/server/main.go
git commit -m "feat: 新增日志清理接口（secret 校验 + 分批清理）"
```

---

### Task 7: 注册公开路由

**Files:**
- Modify: `internal/router/router.go`

- [ ] **Step 1: 加公开路由**

`internal/router/router.go` 第 42 行（`api.POST("/auth/login", authCtrl.Login)` 之后）加：

```go
	// 日志清理（公开：定时任务调度器无 JWT，靠 secret 参数防滥用）
	api.POST("/logs/cleanup", logCtrl.Cleanup)
```

- [ ] **Step 2: 编译验证**

```bash
go build ./...
```

预期：无输出。

- [ ] **Step 3: 提交**

```bash
git add internal/router/router.go
git commit -m "feat: 注册公开路由 POST /api/logs/cleanup"
```

---

### Task 8: 重新生成 Swagger 文档

**Files:**
- Modify: `docs/docs.go`、`docs/swagger.json`、`docs/swagger.yaml`（swag 自动生成）

- [ ] **Step 1: 重新生成**

```bash
swag init -g cmd/server/main.go
```

预期：生成成功，`docs/` 三个文件更新，包含 `/logs/cleanup` 路由。

- [ ] **Step 2: 提交**

```bash
git add docs/
git commit -m "docs: 重新生成 Swagger（新增日志清理接口）"
```

---

### Task 9: 手工回归验证

**前置条件：** 本地环境已按 CLAUDE.md 覆盖 .env（MYSQL_PORT=3307 等），服务可启动；`.env` 已配置 `LOG_CLEANUP_SECRET`。

- [ ] **Step 1: 启动服务**

```bash
go run cmd/server/main.go
```

预期：日志输出 "数据库迁移完成"（000006 索引已建）与 "Server running on :8000"。

- [ ] **Step 2: 造数据**（MySQL 手动插入旧日志，`--default-character-set=utf8mb4`）

```bash
mysql -h 127.0.0.1 -P 3307 -u root -p --default-character-set=utf8mb4 ginadmin -e "
INSERT INTO operation_logs (operator_id, module, action, method, path, params, response, duration, ip, created_at) VALUES
(1,'测试','清理','GET','/test/old','{}','{}',1,'127.0.0.1',DATE_SUB(NOW(), INTERVAL 40 DAY)),
(1,'测试','清理','GET','/test/new','{}','{}',1,'127.0.0.1',NOW());
INSERT INTO login_logs (username, status, message, ip, created_at) VALUES
('olduser',1,'旧', '127.0.0.1', DATE_SUB(NOW(), INTERVAL 40 DAY)),
('newuser',1,'新', '127.0.0.1', NOW());
"
```

- [ ] **Step 3: 验证非法参数**

```bash
curl -s -X POST "http://localhost:8000/api/logs/cleanup?secret=wrong&days=30"
```

预期：HTTP 403，body `{"code":403,"message":"密钥无效"}`

```bash
curl -s -X POST "http://localhost:8000/api/logs/cleanup?secret=<真实密钥>&days=abc"
```

预期：HTTP 200，body `{"code":400,"message":"days 参数非法（1~3650）"}`

- [ ] **Step 4: 验证正常清理**

```bash
curl -s -X POST "http://localhost:8000/api/logs/cleanup?secret=<真实密钥>&days=30"
```

预期：`{"code":200,"message":"success","data":{"login_deleted":1,"operation_deleted":1}}`（40 天前各 1 条被删，今天的新日志保留）

```bash
mysql -h 127.0.0.1 -P 3307 -u root -p ginadmin -e "SELECT COUNT(*) AS old_op FROM operation_logs WHERE path='/test/old'; SELECT COUNT(*) AS new_op FROM operation_logs WHERE path='/test/new'; SELECT COUNT(*) AS old_login FROM login_logs WHERE username='olduser'; SELECT COUNT(*) AS new_login FROM login_logs WHERE username='newuser';"
```

预期：old_op=0、new_op=1、old_login=0、new_login=1。

- [ ] **Step 5: 验证 scope 参数**

```bash
curl -s -X POST "http://localhost:8000/api/logs/cleanup?secret=<真实密钥>&days=30&scope=login"
```

预期：`data` 仅含 `login_deleted` 字段。

- [ ] **Step 6: 验证 ES 同步清理**（若 ES 可用）

先确认旧文档存在于 ES（`GET http://localhost:9200/operation_logs/_count?q=path:/test/old` 为 1），再调用清理接口后该查询为 0。

- [ ] **Step 7: 任务管理页端到端**

登录后台 → 任务管理 → 新建任务：
- URL：`http://go-app:8000/api/logs/cleanup?secret=<LOG_CLEANUP_SECRET>&days=30`
- Method：POST
- Cron：`0 0 3 * * *`（每天凌晨 3 点）
- 保存后点"立即执行"

预期：执行日志状态为成功，HTTP 200，响应含 `login_deleted`/`operation_deleted`；再次验证旧日志已删。

---

## Self-Review

**Spec coverage：**
- 清理端点公开 + secret 校验 → Task 6/7 ✓
- days 参数可配（1~3650 校验）→ Task 6 ✓
- 分批 LIMIT 1000 循环删除 → Task 3/5 ✓
- ES 同步清理 + 降级 → Task 4/5 ✓
- created_at 索引 → Task 1 ✓
- LOG_CLEANUP_SECRET 配置 → Task 2 ✓
- 任务页配置（纯数据）→ Task 9 Step 7 ✓
- 验证 → Task 9 ✓

**Placeholder scan：** 无 TBD/TODO；所有代码步骤含完整代码。

**Type consistency：** `DeleteOlderThan(before time.Time, limit int) (int64, error)` 在 Task 3 定义、Task 5 调用，签名一致；`DeleteByTime(before time.Time) (int64, error)` 同理；`NewLogController` 新签名在 Task 6 定义、main.go 调用点同步更新；`Cleanup(days int) (int64, error)` 两个 Service 方法签名一致。
