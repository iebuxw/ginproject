# 登录失败锁定 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 同一用户名连续登录失败达阈值后临时锁定账号，到期自动解锁。

**Architecture:** Redis 单 key `login_fail:<username>` 存连续失败计数。`AuthService.Login` 开头检查计数是否达阈值，达阈值返回 `ErrAccountLocked`（含剩余分钟数）；密码错误时 INCR，首次失败设 EXPIRE 窗口，达阈值刷新 TTL 为完整锁定时长；登录成功 DEL 清零。controller 对 `ErrAccountLocked` 记登录日志并返回提示，但不发告警邮件。

**Tech Stack:** Go 1.18 / Gin / go-redis v9 / Viper。Redis 3.2 兼容：仅用 GET/INCR/EXPIRE/TTL/DEL。

**规格要点（对应需求）：**
- 维度：按用户名（含不存在的用户名，防枚举差异）
- 阈值/时长可配置：`LOGIN_LOCK_MAX_ATTEMPTS`（默认 5）、`LOGIN_LOCK_DURATION`（分钟，默认 15）
- 自动解锁：TTL 到期 key 消失、计数归零
- 失败累计窗口 = 锁定时长（15 分钟内累计 5 次即锁）
- 不发告警邮件的条件不变：仅 `ErrInvalidCredentials` 触发；锁定期间的失败（`ErrAccountLocked`）不重复发
- 锁定期间即使密码正确也拒绝登录（check 在密码校验之前）
- 对外契约不变：路由、请求/响应 JSON 结构（`{code, message, data}`）不变，仅 message 文案新增

**注意（来自 CLAUDE.md）：**
- 项目无统一测试基建，仅少量纯函数单测；本计划为 Redis+GORM 依赖逻辑，不写单测，靠 Docker 重建 + curl 验证
- 验证用 `docker compose up -d --build go-app` + curl，不本地 go run
- DB 密码是 `123456`（不是 admin）
- 原则上不写逐行注释，仅在非显而易见处（如 TTL 刷新语义）写简短注释

---

### Task 1: Config 新增 LoginLockConfig

**Files:**
- Modify: `internal/config/config.go`
- Modify: `.env.example`

- [ ] **Step 1: 修改 `internal/config/config.go`**

`Config` 结构体在 `JWT` 字段后新增 `LoginLock LoginLockConfig`：

```go
type Config struct {
	Server           ServerConfig
	Database         DatabaseConfig
	Redis            RedisConfig
	JWT              JWTConfig
	LoginLock        LoginLockConfig
	RabbitMQ         RabbitMQConfig
	Mail             MailConfig
	Elasticsearch    ElasticsearchConfig
	LogCleanupSecret string
}
```

在 `JWTConfig` 定义后新增（紧跟现有风格）：

```go
// LoginLockConfig 登录失败锁定配置；MaxAttempts 次失败内累计触发，DurationMinutes 为累计窗口与锁定时长（分钟）
type LoginLockConfig struct {
	MaxAttempts    int
	DurationMinutes int
}
```

`Load()` 中 `viper.SetDefault("JWT_EXPIRE_HOURS", 24)` 后新增：

```go
	viper.SetDefault("LOGIN_LOCK_MAX_ATTEMPTS", 5)
	viper.SetDefault("LOGIN_LOCK_DURATION", 15)
```

`Load()` 返回值中 `JWT` 之后新增：

```go
		LoginLock: LoginLockConfig{
			MaxAttempts:     viper.GetInt("LOGIN_LOCK_MAX_ATTEMPTS"),
			DurationMinutes: viper.GetInt("LOGIN_LOCK_DURATION"),
		},
```

- [ ] **Step 2: 修改 `.env.example`**

`JWT_EXPIRE_HOURS=24` 之后新增：

```
# 登录失败锁定：15 分钟内连续失败 5 次锁定账号 15 分钟（到期自动解锁）
LOGIN_LOCK_MAX_ATTEMPTS=5
LOGIN_LOCK_DURATION=15
```

- [ ] **Step 3: 编译验证**

Run: `go build ./...`
Expected: 无输出（编译通过）

- [ ] **Step 4: Commit**

```bash
git add internal/config/config.go .env.example
git commit -m "feat: 新增登录失败锁定配置项 LOGIN_LOCK_MAX_ATTEMPTS/LOGIN_LOCK_DURATION"
```

---

### Task 2: AuthService 登录锁定逻辑

**Files:**
- Modify: `internal/service/auth_service.go`

- [ ] **Step 1: 新增锁定错误变量**

在 `ErrInvalidCredentials` 定义（auth_service.go:24）后新增：

```go
// ErrAccountLocked 账号因连续登录失败被临时锁定，剩余等待分钟数由 controller 拼入提示文案
var ErrAccountLocked = errors.New("账号已锁定")
```

- [ ] **Step 2: 新增私有辅助方法**

在 `Login` 方法后新增三个私有方法（同一文件内 Helper，符合项目纪律）：

```go
// loginFailKey 用户名维度的失败计数 key，TTL 即累计窗口与锁定时长
func loginFailKey(username string) string {
	return fmt.Sprintf("login_fail:%s", username)
}

// isLocked 失败计数已达阈值即锁定；Redis 异常时放行（降级不阻断登录）
func (s *AuthService) isLocked(username string) (bool, int) {
	val, err := s.rdb.Get(context.Background(), loginFailKey(username)).Int()
	if err != nil {
		return false, 0
	}
	if val < s.cfg.LoginLock.MaxAttempts {
		return false, 0
	}
	ttl, err := s.rdb.TTL(context.Background(), loginFailKey(username)).Result()
	if err != nil || ttl <= 0 {
		return true, 1
	}
	// 剩余分钟数向上取整，避免显示"0分钟后"
	return true, int((ttl + time.Minute - 1) / time.Minute)
}

// recordFailure 记一次失败：首次失败设累计窗口，达阈值刷新 TTL 为完整锁定时长
func (s *AuthService) recordFailure(username string) {
	key := loginFailKey(username)
	ctx := context.Background()
	val, err := s.rdb.Incr(ctx, key).Result()
	if err != nil {
		return
	}
	window := time.Duration(s.cfg.LoginLock.DurationMinutes) * time.Minute
	if val == int64(s.cfg.LoginLock.MaxAttempts) {
		// 达到阈值：刷新 TTL 为完整锁定时长，此刻起锁定开始计时
		s.rdb.Expire(ctx, key, window)
	} else if val == 1 {
		s.rdb.Expire(ctx, key, window)
	}
}

// clearFailures 登录成功清零计数
func (s *AuthService) clearFailures(username string) {
	s.rdb.Del(context.Background(), loginFailKey(username))
}
```

- [ ] **Step 3: 修改 `Login` 方法**

在用户查询之前（方法开头、`user, err := s.userDAO.FindByUsername(username)` 之前）插入锁定检查：

```go
	if locked, remain := s.isLocked(username); locked {
		return "", nil, fmt.Errorf("%w，请 %d 分钟后再试", ErrAccountLocked, remain)
	}
```

密码错误分支（`if !utils.CheckPassword(password, user.Password)`）改为：

```go
	if !utils.CheckPassword(password, user.Password) {
		s.recordFailure(username)
		return "", nil, ErrInvalidCredentials
	}
```

用户不存在分支（`errors.Is(err, gorm.ErrRecordNotFound)`）同样加 `s.recordFailure(username)`：

```go
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.recordFailure(username)
			return "", nil, ErrInvalidCredentials
		}
```

登录成功分支、生成 token 之前加清零：

```go
	s.clearFailures(username)
```

注意：用户被禁用（`user.Status != 1`）不计失败、不清零，保持现状直接返回错误。

- [ ] **Step 4: 编译验证**

Run: `go build ./...`
Expected: 无输出

- [ ] **Step 5: Commit**

```bash
git add internal/service/auth_service.go
git commit -m "feat: 登录连续失败达阈值后临时锁定账号（Redis 计数 + TTL 自动解锁）"
```

---

### Task 3: Controller 处理 ErrAccountLocked

**Files:**
- Modify: `internal/controller/auth_controller.go`

- [ ] **Step 1: 修改 `Login` 错误处理分支**

现有错误分支（auth_controller.go:58-66）中，`errors.Is(err, service.ErrInvalidCredentials)` 的告警邮件条件保持不变，锁定错误自然不满足该条件、不会发邮件。仅需确认错误信息完整透传（`fmt.Errorf("%w...")` 包装后 `err.Error()` 已含剩余分钟数）。

错误分支改为（新增锁定分支注释，逻辑结构不变）：

```go
	token, user, err := ctl.authService.Login(req.Username, req.Password)
	if err != nil {
		_ = ctl.loginLogService.Create(&model.LoginLog{
			Username: req.Username, Status: 0, Message: err.Error(), IP: c.ClientIP(), CreatedAt: model.DateTime(time.Now()),
		})
		// 仅凭据错误发告警邮件；锁定期间（ErrAccountLocked）不重复发
		if errors.Is(err, service.ErrInvalidCredentials) {
			ctl.publishLoginAlert(req.Username, c.ClientIP(), err.Error())
		}
		utils.Error(c, 401, err.Error())
		return
	}
```

（若现有代码已完全一致则仅需补充注释，无逻辑改动。）

- [ ] **Step 2: 编译验证**

Run: `go build ./...`
Expected: 无输出

- [ ] **Step 3: Commit**

```bash
git add internal/controller/auth_controller.go
git commit -m "feat: 登录锁定错误记日志并返回剩余等待时间提示"
```

---

### Task 4: Swagger 注释与文档同步

**Files:**
- Modify: `internal/controller/auth_controller.go`（Login 的 Swagger 注释）

- [ ] **Step 1: 更新 Login Swagger 注释**

`@Description 使用用户名和密码登录，返回 JWT Token` 改为：

```
// @Description 使用用户名和密码登录，返回 JWT Token。连续失败达到阈值后账号临时锁定，到期自动解锁
```

- [ ] **Step 2: 重新生成 Swagger 文档（需本机有 swag）**

Run: `swag init -g cmd/server/main.go`
Expected: `docs/` 下文档更新，无报错。若本机无 swag 则跳过此步，在交付说明中注明。

- [ ] **Step 3: Commit**

```bash
git add internal/controller/auth_controller.go docs/
git commit -m "docs: 登录接口 Swagger 注释补充锁定行为说明"
```

---

### Task 5: Docker 重建 + curl 验证

**Files:** 无代码改动，纯验证。

- [ ] **Step 1: 重建后端容器**

Run: `docker compose up -d --build go-app`
Expected: 容器启动，`docker compose logs go-app --tail 20` 无报错、迁移正常。

- [ ] **Step 2: 验证锁定触发**

连续 6 次错误密码登录（密码用 `wrongpass`）：

```bash
for i in 1 2 3 4 5 6; do curl -sk -X POST http://localhost:8443/api/auth/login -H "Content-Type: application/json" -d '{"username":"admin","password":"wrongpass"}'; echo; done
```

Expected: 前 5 次返回 `用户名或密码错误`；第 6 次起返回 `账号已锁定，请 N 分钟后再试`（N ≤ 15）。

- [ ] **Step 3: 验证锁定期间正确密码也被拒**

```bash
curl -sk -X POST http://localhost:8443/api/auth/login -H "Content-Type: application/json" -d '{"username":"admin","password":"123456"}'
```

Expected: 返回 `账号已锁定，请 N 分钟后再试`（非登录成功）。

- [ ] **Step 4: 验证 Redis 计数状态**

```bash
docker compose exec redis redis-cli GET login_fail:admin
docker compose exec redis redis-cli TTL login_fail:admin
```

Expected: GET 返回 `5`（或达阈值后的值），TTL 为正数（秒）。

- [ ] **Step 5: 验证解锁恢复**

清掉计数模拟到期：

```bash
docker compose exec redis redis-cli DEL login_fail:admin
curl -sk -X POST http://localhost:8443/api/auth/login -H "Content-Type: application/json" -d '{"username":"admin","password":"123456"}'
```

Expected: 正确密码登录成功，返回 token。

- [ ] **Step 6: 验证登录日志**

登录管理后台查看登录日志，或：

```bash
docker compose exec mysql mysql -uroot -p"$MYSQL_PASSWORD" ginadmin --default-character-set=utf8mb4 -e "SELECT username,status,message FROM login_logs ORDER BY id DESC LIMIT 8;"
```

Expected: 失败记录 message 为 `用户名或密码错误` / `账号已锁定，请 N 分钟后再试`，最后一条 status=1。

（命令中 `$MYSQL_PASSWORD` 以 `.env` 实际值为准；不确定时先 `docker compose exec mysql printenv MYSQL_ROOT_PASSWORD` 查询。）

- [ ] **Step 7: 无需提交（纯验证任务）**

若验证全部通过，功能完成。

---

## 手工回归测试建议清单（自动化覆盖不到的部分）

1. 前端登录页输入错误密码 5 次，第 6 次提示"账号已锁定，请 N 分钟后再试"（文案正常渲染、无 JS 报错）
2. 等待 15 分钟（或 Redis DEL）后正确密码可登录
3. 登录成功后再次输错密码，计数从 0 重新累计（成功清零逻辑）
4. 告警邮件：锁定期间连续失败不再触发新邮件（同 IP 5 分钟限频本就存在，需观察锁定后是否还有邮件）
