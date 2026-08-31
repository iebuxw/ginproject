# 登录验证码实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为登录页面添加图片验证码功能，防止暴力破解，支持通过系统设置页面开关控制。

**Architecture:** 使用 `dchest/captcha` 库生成 4 位数字图片验证码，答案存 Redis（TTL 2 分钟），前端通过 base64 图片展示。登录时后端校验验证码，一次性消耗。

**Tech Stack:** Go / dchest/captcha / Redis / Vue 2 / Element UI

---

## File Structure

| Action | File | Responsibility |
|--------|------|----------------|
| Modify | `go.mod` | 添加 dchest/captcha 依赖 |
| Create | `internal/captcha/store.go` | Redis 适配的 captcha.Store 实现 |
| Create | `internal/controller/captcha_controller.go` | 验证码生成接口 |
| Modify | `internal/controller/auth_controller.go` | 登录时校验验证码 |
| Modify | `internal/router/router.go` | 注册 captcha 路由 |
| Modify | `cmd/server/main.go` | 依赖注入 |
| Create | `migrations/000017_add_captcha_setting.up.sql` | 验证码开关种子 |
| Create | `migrations/000017_add_captcha_setting.down.sql` | 回滚 |
| Modify | `web/src/api/auth.js` | 新增 getCaptcha API |
| Modify | `web/src/views/login/index.vue` | 验证码输入框 + 图片 |
| Modify | `web/src/views/setting/index.vue` | 验证码开关 |

---

### Task 1: 添加 dchest/captcha 依赖

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: 安装依赖**

```bash
cd D:\phpStudy\PHPTutorial\WWW\ginproject
go get github.com/dchest/captcha
```

- [ ] **Step 2: 验证依赖安装成功**

```bash
grep "dchest/captcha" go.mod
```

Expected: 行中包含 `github.com/dchest/captcha`

- [ ] **Step 3: 提交**

```bash
git add go.mod go.sum
git commit -m "deps: 添加 dchest/captcha 验证码库依赖"
```

---

### Task 2: 实现 Redis Captcha Store

**Files:**
- Create: `internal/captcha/store.go`

`dchest/captcha` 默认使用内存 store，不支持分布式。需要实现 `captcha.Store` 接口，用 Redis 存储验证码答案。

- [ ] **Step 1: 创建 Redis captcha store**

```go
package captcha

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// captchaTTL 验证码有效期
const captchaTTL = 2 * time.Minute

// RedisStore 实现 dchest/captcha.Store 接口，使用 Redis 存储验证码
type RedisStore struct {
	rdb *redis.Client
}

func NewRedisStore(rdb *redis.Client) *RedisStore {
	return &RedisStore{rdb: rdb}
}

// Set 将验证码答案存入 Redis
func (s *RedisStore) Set(id string, digits []byte) {
	key := "captcha:" + id
	if err := s.rdb.Set(context.Background(), key, string(digits), captchaTTL).Err(); err != nil {
		log.Printf("验证码存储失败: %v", err)
	}
}

// Get 从 Redis 获取验证码答案；remove=true 时读后删除（一次性消耗）
func (s *RedisStore) Get(id string, remove bool) string {
	key := "captcha:" + id
	ctx := context.Background()
	val, err := s.rdb.Get(ctx, key).Result()
	if err != nil {
		return ""
	}
	if remove {
		s.rdb.Del(ctx, key)
	}
	return val
}

// Collect 清理过期验证码（Redis 自动过期，此方法为空实现）
func (s *RedisStore) Collect() {}
```

- [ ] **Step 2: 验证编译通过**

```bash
go build ./internal/captcha/
```

Expected: 无错误输出

- [ ] **Step 3: 提交**

```bash
git add internal/captcha/store.go
git commit -m "feat: 实现 Redis 验证码存储适配器"
```

---

### Task 3: 实现验证码控制器

**Files:**
- Create: `internal/controller/captcha_controller.go`

- [ ] **Step 1: 创建 CaptchaController**

```go
package controller

import (
	"bytes"
	"encoding/base64"

	"ginproject/internal/utils"

	"github.com/dchest/captcha"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// CaptchaController 验证码接口
type CaptchaController struct {
	rdb *redis.Client
}

func NewCaptchaController(rdb *redis.Client) *CaptchaController {
	return &CaptchaController{rdb: rdb}
}

// Generate 生成验证码
// @Summary 获取验证码
// @Description 生成 4 位数字图片验证码，返回 captcha_id 和 base64 编码的图片
// @Tags 认证
// @Produce json
// @Success 200 {object} utils.Response{data=object{captcha_id=string,captcha_image=string}} "成功"
// @Router /auth/captcha [get]
func (ctl *CaptchaController) Generate(c *gin.Context) {
	id := captcha.NewLen(4)

	var buf bytes.Buffer
	if err := captcha.WriteImage(&buf, id, 240, 80); err != nil {
		utils.Error(c, 500, "验证码生成失败")
		return
	}

	imgBase64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	utils.Success(c, gin.H{
		"captcha_id":    id,
		"captcha_image": "data:image/png;base64," + imgBase64,
	})
}
```

- [ ] **Step 2: 验证编译通过**

```bash
go build ./internal/controller/
```

Expected: 无错误输出

- [ ] **Step 3: 提交**

```bash
git add internal/controller/captcha_controller.go
git commit -m "feat: 实现验证码生成控制器"
```

---

### Task 4: 修改 AuthController 支持验证码校验

**Files:**
- Modify: `internal/controller/auth_controller.go`

- [ ] **Step 1: 修改 LoginRequest 结构体，新增验证码字段**

将：
```go
type LoginRequest struct {
	Username string `json:"username" binding:"required" example:"admin"`
	Password string `json:"password" binding:"required" example:"123456"`
}
```

改为：
```go
type LoginRequest struct {
	Username    string `json:"username" binding:"required" example:"admin"`
	Password    string `json:"password" binding:"required" example:"123456"`
	CaptchaID   string `json:"captcha_id"`
	CaptchaCode string `json:"captcha_code"`
}
```

- [ ] **Step 2: 给 AuthController 新增 settingService 依赖**

将：
```go
type AuthController struct {
	authService     *service.AuthService
	menuDAO         *dao.MenuDAO
	userDAO         *dao.UserDAO
	loginLogService *service.LoginLogService
	rdb             *redis.Client
	publishCh       *amqp091.Channel
}

func NewAuthController(authService *service.AuthService, menuDAO *dao.MenuDAO, userDAO *dao.UserDAO, loginLogService *service.LoginLogService, rdb *redis.Client, publishCh *amqp091.Channel) *AuthController {
	return &AuthController{authService, menuDAO, userDAO, loginLogService, rdb, publishCh}
}
```

改为：
```go
type AuthController struct {
	authService     *service.AuthService
	menuDAO         *dao.MenuDAO
	userDAO         *dao.UserDAO
	loginLogService *service.LoginLogService
	rdb             *redis.Client
	publishCh       *amqp091.Channel
	settingService  *service.SystemSettingService
}

func NewAuthController(authService *service.AuthService, menuDAO *dao.MenuDAO, userDAO *dao.UserDAO, loginLogService *service.LoginLogService, rdb *redis.Client, publishCh *amqp091.Channel, settingService *service.SystemSettingService) *AuthController {
	return &AuthController{authService, menuDAO, userDAO, loginLogService, rdb, publishCh, settingService}
}
```

- [ ] **Step 3: 在 Login 方法开头添加验证码校验逻辑**

将 `Login` 方法的开头（从 `var req LoginRequest` 到 `token, user, err := ctl.authService.Login(...)` 之前）替换为：

```go
func (ctl *AuthController) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}

	// 验证码校验（根据系统设置决定是否启用）
	if err := ctl.checkCaptcha(req.CaptchaID, req.CaptchaCode); err != nil {
		utils.Error(c, 400, err.Error())
		return
	}

	token, user, err := ctl.authService.Login(req.Username, req.Password)
```

- [ ] **Step 4: 在 Login 方法之后添加 checkCaptcha 私有方法**

在 `publishLoginAlert` 方法之前添加：

```go
// checkCaptcha 校验验证码；未启用时跳过；Redis 不可用时降级放行
func (ctl *AuthController) checkCaptcha(captchaID, captchaCode string) error {
	settings, err := ctl.settingService.GetAll()
	if err != nil {
		log.Printf("读取系统设置失败: %v", err)
		return nil // 降级放行
	}
	if settings["captcha_enabled"] != "1" {
		return nil
	}

	if captchaID == "" || captchaCode == "" {
		return fmt.Errorf("请输入验证码")
	}

	key := "captcha:" + captchaID
	ctx := context.Background()
	stored, err := ctl.rdb.Get(ctx, key).Result()
	if err != nil {
		log.Printf("验证码读取失败（可能已过期）: %v", err)
		return fmt.Errorf("验证码已过期，请重新获取")
	}
	// 一次性消耗
	ctl.rdb.Del(ctx, key)

	if stored != captchaCode {
		return fmt.Errorf("验证码错误")
	}
	return nil
}
```

- [ ] **Step 5: 确保 import 中包含 fmt**

在 import 块中确认有 `"fmt"`，如果没有则添加。

- [ ] **Step 6: 验证编译通过**

```bash
go build ./internal/controller/
```

Expected: 编译错误提示 `cmd/server/main.go` 中 `NewAuthController` 参数不匹配（预期行为，下一步修复）

- [ ] **Step 7: 提交**

```bash
git add internal/controller/auth_controller.go
git commit -m "feat: 登录接口支持验证码校验"
```

---

### Task 5: 注册路由 + 依赖注入

**Files:**
- Modify: `internal/router/router.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: 在 router.go 中新增 captchaCtrl 参数**

修改 `Setup` 函数签名，新增 `captchaCtrl *controller.CaptchaController` 参数。在 `settingCtrl` 参数之后添加：

```go
func Setup(
	cfg *config.Config,
	authCtrl *controller.AuthController,
	// ... 其他参数不变 ...
	settingCtrl *controller.SystemSettingController,
	captchaCtrl *controller.CaptchaController,
	notificationCtrl *controller.NotificationController,
	logSettingCtrl *controller.LogSettingController,
) *gin.Engine {
```

- [ ] **Step 2: 在公开路由中注册 captcha 路由**

在 `api.POST("/auth/login", authCtrl.Login)` 之后添加：

```go
api.GET("/auth/captcha", captchaCtrl.Generate)
```

- [ ] **Step 3: 在 main.go 中创建 CaptchaController 并注入**

在 `settingCtrl` 创建之后添加：

```go
captchaCtrl := controller.NewCaptchaController(rdb)
```

- [ ] **Step 4: 修改 AuthController 创建，传入 settingService**

将：
```go
authCtrl := controller.NewAuthController(authService, menuDAO, userDAO, loginLogService, rdb, publishCh)
```

改为：
```go
authCtrl := controller.NewAuthController(authService, menuDAO, userDAO, loginLogService, rdb, publishCh, systemSettingService)
```

- [ ] **Step 5: 修改 router.Setup 调用，传入 captchaCtrl**

在 `settingCtrl` 之后添加 `captchaCtrl`：

```go
r := router.Setup(cfg, authCtrl, userCtrl, roleCtrl, menuCtrl, logCtrl, loginLogCtrl, wsCtrl, uploadCtrl, authService, userDAO, menuDAO, logDAO, logRepo, dictTypeCtrl, dictDataCtrl, cronTaskCtrl, dbBackupCtrl, fileCtrl, dashboardCtrl, healthCtrl, settingCtrl, captchaCtrl, notificationCtrl, logSettingCtrl)
```

- [ ] **Step 6: 注册 Redis captcha store**

在 `captchaCtrl` 创建之前，设置 captcha 使用 Redis store：

```go
captchaStore := captcha.NewRedisStore(rdb)
captcha.SetCustomStore(captchaStore)
```

并在 import 中添加：
```go
captchapkg "ginproject/internal/captcha"
"github.com/dchest/captcha"
```

注意：需要解决 captcha 包名冲突——将内部 captcha 包 import 别名为 `captchapkg`，标准库 captcha 保持原名。

- [ ] **Step 7: 验证编译通过**

```bash
go build ./cmd/server/
```

Expected: 无错误输出

- [ ] **Step 8: 提交**

```bash
git add internal/router/router.go cmd/server/main.go
git commit -m "feat: 注册验证码路由和依赖注入"
```

---

### Task 6: 数据库迁移

**Files:**
- Create: `migrations/000017_add_captcha_setting.up.sql`
- Create: `migrations/000017_add_captcha_setting.down.sql`

- [ ] **Step 1: 创建 up 迁移**

```sql
INSERT IGNORE INTO `system_settings` (`setting_key`, `setting_value`) VALUES
('captcha_enabled', '0');
```

- [ ] **Step 2: 创建 down 迁移**

```sql
DELETE FROM `system_settings` WHERE `setting_key` = 'captcha_enabled';
```

- [ ] **Step 3: 提交**

```bash
git add migrations/000017_add_captcha_setting.up.sql migrations/000017_add_captcha_setting.down.sql
git commit -m "feat: 添加验证码开关数据库迁移"
```

---

### Task 7: Docker 构建验证后端

**Files:** 无新增/修改

- [ ] **Step 1: Docker 重建 go-app**

```bash
docker compose up -d --build go-app
```

- [ ] **Step 2: 验证容器启动正常**

```bash
docker compose logs go-app --tail 20
```

Expected: 无 panic/fatal，输出 "Server running on :8000"

- [ ] **Step 3: 测试获取验证码接口**

```bash
curl -k https://localhost:8443/api/auth/captcha
```

Expected: 返回 JSON，包含 `captcha_id` 和 `captcha_image` 字段

- [ ] **Step 4: 测试验证码开关为禁用时登录（应跳过验证码）**

```bash
curl -k -X POST https://localhost:8443/api/auth/login -H "Content-Type: application/json" -d '{"username":"admin","password":"123456"}'
```

Expected: 正常返回 token（验证码默认禁用）

- [ ] **Step 5: 开启验证码开关后测试登录（无验证码应失败）**

先通过已登录的管理员 token 开启验证码，再测试无验证码登录：

```bash
# 用返回的 token 开启验证码
curl -k -X PUT https://localhost:8443/api/settings -H "Content-Type: application/json" -H "Authorization: Bearer <token>" -d '{"captcha_enabled":"1"}'

# 无验证码登录应失败
curl -k -X POST https://localhost:8443/api/auth/login -H "Content-Type: application/json" -d '{"username":"admin","password":"123456"}'
```

Expected: 返回 "请输入验证码"

---

### Task 8: 前端 API 和登录页面改造

**Files:**
- Modify: `web/src/api/auth.js`
- Modify: `web/src/views/login/index.vue`

- [ ] **Step 1: 在 auth.js 中新增 getCaptcha 函数**

```js
export const getCaptcha = () => request.get('/auth/captcha')
```

- [ ] **Step 2: 改造登录页面模板**

替换整个 `<template>` 部分：

```vue
<template>
  <div class="login-container" style="display:flex;justify-content:center;align-items:center;height:100vh;background:#f0f2f5">
    <el-card style="width:400px">
      <h2 style="text-align:center;margin-bottom:20px">{{ siteName }}</h2>
      <el-form ref="form" :model="form" :rules="rules">
        <el-form-item prop="username">
          <el-input v-model="form.username" placeholder="用户名" prefix-icon="el-icon-user"></el-input>
        </el-form-item>
        <el-form-item prop="password">
          <el-input v-model="form.password" type="password" placeholder="密码" prefix-icon="el-icon-lock" @keyup.enter.native="handleLogin"></el-input>
        </el-form-item>
        <el-form-item v-if="captchaEnabled" prop="captcha_code">
          <div style="display:flex;align-items:center">
            <el-input v-model="form.captcha_code" placeholder="验证码" style="flex:1" @keyup.enter.native="handleLogin"></el-input>
            <img v-if="captchaImage" :src="captchaImage" alt="验证码" style="height:40px;margin-left:10px;cursor:pointer;border-radius:4px" @click="refreshCaptcha" title="点击刷新" />
          </div>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" style="width:100%" @click="handleLogin" :loading="loading">登录</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>
```

- [ ] **Step 3: 改造登录页面脚本**

替换整个 `<script>` 部分：

```vue
<script>
import { getSettings } from '@/api/setting'
import { getCaptcha } from '@/api/auth'
export default {
  data() {
    return {
      form: { username: '', password: '', captcha_id: '', captcha_code: '' },
      rules: {
        username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
        password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
      },
      loading: false,
      siteName: 'GinAdmin',
      captchaEnabled: false,
      captchaImage: ''
    }
  },
  created() {
    getSettings().then(res => {
      if (res.code === 200) {
        if (res.data.site_name) {
          this.siteName = res.data.site_name
          document.title = res.data.site_name
        }
        if (res.data.captcha_enabled === '1') {
          this.captchaEnabled = true
          this.refreshCaptcha()
        }
      }
    }).catch(() => {})
  },
  methods: {
    async refreshCaptcha() {
      try {
        const res = await getCaptcha()
        if (res.code === 200) {
          this.form.captcha_id = res.data.captcha_id
          this.captchaImage = res.data.captcha_image
          this.form.captcha_code = ''
        }
      } catch {}
    },
    async handleLogin() {
      const valid = await this.$refs.form.validate().catch(() => false)
      if (!valid) return
      this.loading = true
      try {
        await this.$store.dispatch('user/login', this.form)
        this.$router.push('/')
      } catch {
        if (this.captchaEnabled) {
          this.refreshCaptcha()
        }
        this.loading = false
      }
    }
  }
}
</script>
```

- [ ] **Step 4: 提交**

```bash
git add web/src/api/auth.js web/src/views/login/index.vue
git commit -m "feat: 登录页面添加验证码输入和展示"
```

---

### Task 9: 系统设置页面添加验证码开关

**Files:**
- Modify: `web/src/views/setting/index.vue`

- [ ] **Step 1: 在 form 中添加 captcha_enabled 字段**

修改 `data()` 中的 `form`：

```js
form: { site_name: '', site_logo: '', captcha_enabled: '0' },
```

- [ ] **Step 2: 在模板的 Logo 表单项之后、保存按钮之前添加验证码开关**

```vue
<el-form-item label="登录验证码">
  <el-switch
    v-model="form.captcha_enabled"
    active-value="1"
    inactive-value="0"
    active-text="启用"
    inactive-text="禁用"
  ></el-switch>
  <div style="color:#999;font-size:12px;margin-top:4px">启用后登录时需要输入图片验证码</div>
</el-form-item>
```

- [ ] **Step 3: 修改 fetchSettings 方法，读取 captcha_enabled**

在 `fetchSettings` 的 `this.form` 赋值中添加：

```js
this.form = {
  site_name: res.data.site_name || '',
  site_logo: res.data.site_logo || '',
  captcha_enabled: res.data.captcha_enabled || '0'
}
```

- [ ] **Step 4: 提交**

```bash
git add web/src/views/setting/index.vue
git commit -m "feat: 系统设置页面添加验证码开关"
```

---

### Task 10: 前端 Docker 构建验证

**Files:** 无新增/修改

- [ ] **Step 1: Docker 重建 nginx**

```bash
docker compose up -d --build nginx
```

- [ ] **Step 2: 浏览器验证登录页**

打开 `https://localhost:8443`，确认：
1. 默认状态下（验证码禁用）看不到验证码输入框
2. 登录功能正常

- [ ] **Step 3: 浏览器验证系统设置页**

登录后进入「系统配置」页面，确认：
1. 验证码开关显示为"禁用"
2. 切换到"启用"并保存

- [ ] **Step 4: 浏览器验证验证码启用后的登录**

退出登录，回到登录页，确认：
1. 出现验证码输入框和图片
2. 点击图片可刷新
3. 输入正确验证码后可登录
4. 输入错误验证码提示"验证码错误"
5. 登录失败后验证码自动刷新

- [ ] **Step 5: 关闭验证码开关**

重新登录后到设置页面关闭验证码，确认登录页不再显示验证码。
