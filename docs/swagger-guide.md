# Swagger 入门指南

## 一、Swagger 是什么

### 1.1 简单理解

Swagger 就是 **API 文档自动生成工具**。以前后端写完接口后要手动写文档告诉前端"这个接口要传什么参数、返回什么格式"，Swagger 让你用代码注释替代手写文档，自动生成一个漂亮的 API 文档页面。

### 1.2 核心概念

| 概念 | 说明 |
|------|------|
| **OpenAPI / Swagger Spec** | 一个描述 API 的标准规范（JSON/YAML 格式） |
| **Swagger UI** | 根据规范渲染出的可视化网页，可以在线调试接口 |
| **swaggo** | Go 生态的 Swagger 工具，通过 Go 注释自动生成 Swagger 规范文档 |

### 1.3 本项目方案

本项目使用 **swaggo/swag**：
- 在 Controller 方法上方写特殊格式的 Go 注释
- 运行 `swag init` 命令，自动生成 `docs/docs.go` + `swagger.json` + `swagger.yaml`
- 注册一个路由提供 Swagger UI 页面，访问即可看到 API 文档

---

## 二、核心名词解释

| 名词 | 含义 | 本项目对应 |
|------|------|-----------|
| **Swagger UI** | 浏览器中的 API 文档网页界面，可在线发送请求测试接口 | `http://localhost:8000/swagger/index.html` |
| **Tags** | 接口分组标签，在 Swagger UI 上按标签折叠显示 | `认证`、`管理员管理`、`角色管理`、`菜单管理`、`操作日志`、`登录日志` |
| **Parameters** | 请求参数，说明接口需要传什么数据 | `path`（URL路径参数）、`query`（查询字符串）、`body`（请求体JSON） |
| **Response** | 响应格式定义，告诉前端返回什么 JSON 结构 | `utils.Response{data=...}` |
| **Security** | 认证/鉴权方式 | `BearerAuth`（JWT Token） |
| **Summary** | 接口的简短摘要（显示在 Swagger UI 列表中） | "用户登录"、"获取用户分页列表" |
| **$ref** | JSON Schema 中的类型引用，指向 definitions 中的定义 | `$ref: "#/definitions/utils.Response"` |
| **BasePath** | 所有 API 的公共前缀路径 | `/api` |

---

## 三、注释语法速查（swaggo 格式）

### 3.1 全局注释（写在 `main.go` 文件顶部，package 声明之前）

```go
// @title GinAdmin API                    // API 标题
// @version 1.0                           // 版本号
// @description 后台管理系统 API 文档       // 描述
// @host localhost:8000                    // 服务器地址
// @BasePath /api                         // 基础路径
// @securityDefinitions.apikey BearerAuth  // 安全方案名称
// @in header                              // Token 放在 Header 中
// @name Authorization                     // Header 字段名
// @description JWT Token，格式: Bearer {token}  // 描述
package main
```

### 3.2 接口注释（写在 Controller 方法上方）

```go
// Login 用户登录
// @Summary 用户登录
// @Description 使用用户名和密码登录，返回 JWT Token
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body object true "登录参数"
// @Param body.body.username body string true "用户名"
// @Param body.body.password body string true "密码"
// @Success 200 {object} utils.Response{data=object{token=string,user=model.User}} "成功"
// @Failure 200 {object} utils.Response "业务错误"
// @Router /auth/login [post]
func (ctl *AuthController) Login(c *gin.Context) {
```

### 3.3 各注释字段含义

| 注释 | 含义 | 必填 |
|------|------|------|
| `@Summary` | 简短摘要（Swagger UI 列表中显示） | ✅ |
| `@Description` | 详细描述（展开后显示） | ❌ |
| `@Tags` | 分组标签（Swagger UI 左侧分组） | ✅ |
| `@Accept` | 接收的数据格式（json/xml/form） | ❌ |
| `@Produce` | 返回的数据格式（json/xml） | ❌ |
| `@Param` | 请求参数定义 | ✅ |
| `@Success` | 成功时的响应格式 | ✅ |
| `@Failure` | 失败时的响应格式 | ❌ |
| `@Router` | 路由路径和方法 | ✅ |
| `@Security` | 需要的认证方案 | ❌（需登录的接口要加） |

### 3.4 `@Param` 详解

格式：`@Param name in type required "description"`

| in 值 | 含义 | 示例 |
|-------|------|------|
| `path` | URL 路径参数 | `@Param id path int true "用户 ID"` |
| `query` | 查询字符串参数 | `@Param page query int false "页码" default(1)` |
| `body` | 请求体 JSON | `@Param body body object true "用户信息"` |

嵌套 body 字段写法：`@Param body.body.fieldname body type required "描述"`

### 3.5 `@Success` / `@Failure` 详解

格式：`@Success status {object} type "description"`

```
@Success 200 {object} utils.Response "成功"
@Success 200 {object} utils.Response{data=model.User} "成功"
@Success 200 {object} utils.Response{data=object{list=[]model.User,total=int}} "成功"
@Failure 200 {object} utils.Response "业务错误"
@Success 200 {file} binary "Excel 文件"
```

### 3.6 `@Router` 详解

格式：`@Router /路径 [方法]`

```
@Router /auth/login [post]
@Router /users [get]
@Router /users/{id} [put]
@Router /logs/download/{taskID} [get]
```

---

## 四、运行流程

```
┌──────────────────────────────────────────────────────┐
│  1. 开发者在 Controller 方法上方写 Go 注释            │
│     (summary / tags / params / response / router)   │
└───────────────────────┬──────────────────────────────┘
                        │
                        ▼
┌──────────────────────────────────────────────────────┐
│  2. 运行命令：swag init -g cmd/server/main.go        │
│     swag 工具扫描项目中所有 @ 注释                    │
└───────────────────────┬──────────────────────────────┘
                        │
                        ▼
┌──────────────────────────────────────────────────────┐
│  3. 自动生成 docs/ 目录下的文件：                     │
│     - docs.go      (Go 代码，嵌入 swagger.json)      │
│     - swagger.json  (Swagger 2.0 JSON 规范)          │
│     - swagger.yaml  (Swagger 2.0 YAML 规范)          │
└───────────────────────┬──────────────────────────────┘
                        │
                        ▼
┌──────────────────────────────────────────────────────┐
│  4. router.go 注册路由：                              │
│     r.GET("/swagger/*any", ginSwagger.WrapHandler())  │
│     将 swagger.json 暴露给 Swagger UI 前端            │
└───────────────────────┬──────────────────────────────┘
                        │
                        ▼
┌──────────────────────────────────────────────────────┐
│  5. 启动服务后，浏览器访问：                           │
│     http://localhost:8000/swagger/index.html          │
│     即可看到交互式 API 文档                           │
└──────────────────────────────────────────────────────┘
```

---

## 五、常用命令

```bash
# 重新生成 Swagger 文档（修改注释后必须执行）
swag init -g cmd/server/main.go

# 编译验证
go build ./cmd/server/

# 安装 swag CLI（首次使用）
go install github.com/swaggo/swag/cmd/swag@latest
```

---

## 六、注意事项

1. **修改注释后必须重新生成**
   每次修改了 Controller 上方的注释，都要重新运行 `swag init` 才会生效。

2. **docs/ 目录要提交到 git**
   `docs/docs.go`、`docs/swagger.json`、`docs/swagger.yaml` 是自动生成的，但需要版本控制，方便其他人直接使用。

3. **避免循环引用**
   如果类型 A 引用类型 B，类型 B 又引用类型 A，swag 会报 `recursion detected` 错误。本项目的 `model.Menu` 有自引用（Children 字段），swag 会自动跳过。

4. **Body 参数的嵌套写法**
   swag 不支持直接解析 struct 嵌入，需要用 `body.body.fieldname` 的方式逐个声明字段。

5. **Swagger UI 无需认证**
   `/swagger/*any` 路由没有加 JWT 中间件，任何人都能访问（生产环境注意安全）。

6. **支持在线调试**
   Swagger UI 上点击 "Try it out" 可以直接发送请求，需要登录的接口先在右上角填入 JWT Token。

---

## 七、本项目接口一览

| Tags | 接口 | 方法 | 说明 |
|------|------|------|------|
| 认证 | /auth/login | POST | 用户登录（无需认证） |
| 认证 | /auth/logout | POST | 用户登出 |
| 认证 | /auth/change-password | POST | 修改密码 |
| 认证 | /auth/userinfo | GET | 获取当前用户信息 |
| 管理员管理 | /users | GET | 管理员分页列表 |
| 管理员管理 | /users/:id | GET | 管理员详情 |
| 管理员管理 | /users | POST | 新建管理员 |
| 管理员管理 | /users/:id | PUT | 编辑管理员 |
| 管理员管理 | /users/:id | DELETE | 删除管理员 |
| 角色管理 | /roles | GET | 角色分页列表 |
| 角色管理 | /roles/:id | GET | 角色详情 |
| 角色管理 | /roles | POST | 新建角色 |
| 角色管理 | /roles/:id | PUT | 编辑角色 |
| 角色管理 | /roles/:id | DELETE | 删除角色 |
| 菜单管理 | /menus | GET | 菜单树 |
| 菜单管理 | /menus/:id | GET | 菜单详情 |
| 菜单管理 | /menus | POST | 新建菜单 |
| 菜单管理 | /menus/:id | PUT | 编辑菜单 |
| 菜单管理 | /menus/:id | DELETE | 删除菜单 |
| 操作日志 | /logs | GET | 操作日志查询 |
| 操作日志 | /logs/export | POST | 发起导出任务 |
| 操作日志 | /logs/export-status | GET | 查询导出状态 |
| 操作日志 | /logs/download/:taskID | GET | 下载导出文件 |
| 登录日志 | /login-logs | GET | 登录日志查询 |
