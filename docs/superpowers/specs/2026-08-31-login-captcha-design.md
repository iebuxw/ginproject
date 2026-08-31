# 登录验证码功能设计

**日期：** 2026-08-31
**状态：** 已批准

## 概述

为登录页面添加图片验证码功能，防止暴力破解攻击。使用 `dchest/captcha` 库生成 4 位数字验证码图片，答案存 Redis，支持通过系统设置页面配置开关。

## 需求

- 验证码类型：4 位数字图片验证码
- 触发时机：可通过系统设置页面开关控制（默认禁用）
- 配置位置：`system_settings` 表，key 为 `captcha_enabled`
- 验证码有效期：2 分钟
- 一次性消耗：校验后立即删除，无论成功失败

## 架构设计

### 数据流

```
前端登录页                    后端
    │                          │
    ├── GET /api/auth/captcha ──┤── 生成验证码图片 + 答案存 Redis
    │   ← {id, image_base64} ──┤     key: captcha:{id}, TTL: 2min
    │                          │
    │  [用户输入验证码]          │
    │                          │
    ├── POST /api/auth/login ──┤── 1. 读取 captcha_enabled 设置
    │   {username, password,   │     2. 若启用：校验验证码（比对 Redis）
    │    captcha_id,           │     3. 验证通过 → 走原有登录逻辑
    │    captcha_code}         │     4. 验证失败 → 返回错误，消耗验证码
    │   ← {token, user} ──────┤
```

### Redis Key 设计

- Key: `captcha:{captcha_id}`
- Value: 正确答案字符串（4 位数字）
- TTL: 120 秒
- 一次性消耗：校验后立即 DEL

## 后端改动

### 新增文件

**`internal/controller/captcha_controller.go`**
- `CaptchaController` 结构体，持有 `rdb`（Redis 客户端）
- `GenerateCaptcha(c *gin.Context)` 方法：
  - 调用 `captcha.New()` 生成 4 位数字验证码
  - 将答案存入 Redis `captcha:{id}`，TTL 120 秒
  - 将图片编码为 base64
  - 返回 `{code: 200, data: {captcha_id, captcha_image}}`

### 修改文件

**`internal/controller/auth_controller.go`**
- `LoginRequest` 结构体新增 `CaptchaID` 和 `CaptchaCode` 字段
- `AuthController` 新增 `settingService` 依赖
- `Login()` 方法开头增加验证码校验逻辑：
  1. 调用 `settingService.GetAll()` 读取 `captcha_enabled`
  2. 若值为 `"1"`：从 Redis 读取 `captcha:{captcha_id}` 比对 `captcha_code`，校验后立即 DEL
  3. 若验证码无效：返回错误，不走登录逻辑

**`internal/router/router.go`**
- 新增公共路由：`GET /api/auth/captcha` → `captchaCtrl.Generate`

**`cmd/server/main.go`**
- 创建 `CaptchaController` 实例，注入 Redis 客户端
- 将 `settingService` 注入 `AuthController`

### 新增依赖

```
go get github.com/dchest/captcha
```

## 前端改动

### 修改文件

**`web/src/api/auth.js`**
- 新增 `getCaptcha()` — `GET /auth/captcha`

**`web/src/views/login/index.vue`**
- 表单新增验证码输入框 + 图片展示区域（`v-if="captchaEnabled"`）
- `created()` 中读取 settings 的 `captcha_enabled` 字段
- 页面加载时调用 `getCaptcha()` 获取验证码图片
- 点击图片可刷新验证码
- `handleLogin()` 中将 `captcha_id` 和 `captcha_code` 一起提交
- 登录失败时自动刷新验证码

**UI 布局：**
```
┌─────────────────────────┐
│       后台管理系统        │
│                         │
│  用户名: [__________]   │
│  密  码: [__________]   │
│  验证码: [____] [图片]   │  ← 新增行，图片可点击刷新
│                         │
│       [登  录]          │
└─────────────────────────┘
```

**`web/src/views/setting/index.vue`**
- 新增「验证码开关」的 `el-switch` 组件
- 保存时随其他设置一起提交

## 数据库迁移

**`migrations/000017_add_captcha_setting.up.sql`**
```sql
INSERT IGNORE INTO `system_settings` (`setting_key`, `setting_value`) VALUES
('captcha_enabled', '0');
```

**`migrations/000017_add_captcha_setting.down.sql`**
```sql
DELETE FROM `system_settings` WHERE `setting_key` = 'captcha_enabled';
```

默认禁用，管理员在系统设置页面手动开启。

## 错误处理

- 验证码过期/不存在：返回 "验证码已过期，请重新获取"
- 验证码错误：返回 "验证码错误"，消耗当前验证码
- Redis 不可用：降级跳过验证码校验（记录警告日志），不影响正常登录

## 测试要点

1. 验证码开关禁用时，登录流程不变，无需输入验证码
2. 验证码开关启用时，登录必须携带正确的验证码
3. 验证码过期后校验失败
4. 验证码一次性消耗，使用后立即失效
5. 点击验证码图片可刷新
6. 登录失败后自动刷新验证码
7. 系统设置页面可正常切换验证码开关
