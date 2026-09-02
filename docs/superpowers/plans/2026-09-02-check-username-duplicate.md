# 管理员用户名查重 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 管理员新增/编辑时，用户名重复返回友好错误提示，而非原始数据库错误。

**Architecture:** 后端 service 层捕获 MySQL 唯一索引冲突错误并替换为中文提示，前端 handleSubmit 加 try/catch 确保失败时弹窗不关闭。仅改动 2 个文件。

**Tech Stack:** Go 1.18, Gin, GORM, Vue 2, Element UI

---

### Task 1: 后端 — service 层捕获唯一索引冲突

**Files:**
- Modify: `internal/service/user_service.go`

- [ ] **Step 1: 修改 Create 方法**

在 `Create` 中，将 `return s.userDAO.Create(u)` 改为捕获错误并判断：

```go
func (s *UserService) Create(u *model.User, roleIDs []uint) error {
	hashed, err := utils.HashPassword(u.Password)
	if err != nil {
		return err
	}
	u.Password = hashed
	u.Roles = buildRoles(roleIDs)
	if err := s.userDAO.Create(u); err != nil {
		return friendlyDuplicateError(err)
	}
	return nil
}
```

- [ ] **Step 2: 修改 Update 方法**

在 `Update` 中，将 `return s.userDAO.Update(u)` 改为：

```go
func (s *UserService) Update(u *model.User, roleIDs []uint) error {
	if u.Password != "" {
		hashed, err := utils.HashPassword(u.Password)
		if err != nil {
			return err
		}
		u.Password = hashed
	}
	u.Roles = buildRoles(roleIDs)
	if err := s.userDAO.Update(u); err != nil {
		return friendlyDuplicateError(err)
	}
	return nil
}
```

- [ ] **Step 3: 添加 friendlyDuplicateError 辅助函数**

在 `buildRoles` 函数之后添加：

```go
// friendlyDuplicateError 将 MySQL 唯一索引冲突错误转为友好提示
func friendlyDuplicateError(err error) error {
	if err != nil && strings.Contains(err.Error(), "Duplicate entry") && strings.Contains(err.Error(), "username") {
		return fmt.Errorf("用户名已存在")
	}
	return err
}
```

- [ ] **Step 4: 添加 fmt 导入**

import 块中增加 `"fmt"`，改为：

```go
import (
	"fmt"
	"strings"

	"ginproject/internal/dao"
	"ginproject/internal/model"
	"ginproject/internal/utils"
)
```

- [ ] **Step 5: Docker 重建验证**

```bash
docker compose up -d --build go-app
```

用 curl 测试重复用户名创建，确认返回"用户名已存在"而非原始 MySQL 错误：

```bash
# 先登录获取 token
curl -k -s https://localhost:8443/api/auth/login -X POST -H "Content-Type: application/json" -d '{"username":"admin","password":"123456"}'

# 用返回的 token 尝试创建已存在的用户名 admin
curl -k -s https://localhost:8443/api/users -X POST -H "Content-Type: application/json" -H "Authorization: Bearer <token>" -d '{"username":"admin","password":"123456"}'
```

Expected: `{"code":500,"message":"用户名已存在","data":null}`

- [ ] **Step 6: Commit**

```bash
git add internal/service/user_service.go
git commit -m "fix: 管理员创建/编辑时用户名重复返回友好提示"
```

---

### Task 2: 前端 — handleSubmit 加 try/catch

**Files:**
- Modify: `web/src/views/user/index.vue`

- [ ] **Step 1: 修改 handleSubmit 方法**

将当前的：

```js
async handleSubmit() {
  try { await this.$refs.userForm.validate() } catch { return }
  if (this.isEdit) { await updateUser(this.form.id, this.form) }
  else { await addUser(this.form) }
  this.dialogVisible = false; this.fetchData()
},
```

改为：

```js
async handleSubmit() {
  try { await this.$refs.userForm.validate() } catch { return }
  try {
    if (this.isEdit) { await updateUser(this.form.id, this.form) }
    else { await addUser(this.form) }
    this.dialogVisible = false
    this.fetchData()
  } catch {
    // 错误已由全局拦截器 Message.error 展示，弹窗不关闭
  }
},
```

- [ ] **Step 2: Docker 重建验证**

```bash
cd .. && docker compose up -d --build nginx
```

用 agent-browser 验证：打开新增管理员弹窗，输入已存在的用户名 admin，点击确定，确认页面弹出"用户名已存在"错误提示，弹窗不关闭。

- [ ] **Step 3: Commit**

```bash
git add web/src/views/user/index.vue
git commit -m "fix: 管理员表单提交失败时弹窗不关闭"
```
