# 管理员用户名查重设计

## 背景

管理员新增/编辑时，用户名重复会触发 MySQL 唯一索引冲突，当前后端返回原始数据库错误（如 "Error 1062: Duplicate entry..."），前端展示不友好。

## 方案

采用方案 B：后端改进报错 + 前端提交后捕获。

## 改动范围

| 文件 | 改动 |
|------|------|
| `internal/service/user_service.go` | Create/Update 捕获唯一索引冲突，返回"用户名已存在" |
| `web/src/views/user/index.vue` | handleSubmit 加 try/catch，失败时弹窗不关闭 |

## 后端设计

`UserService.Create()` 和 `UserService.Update()` 中，对 `userDAO.Create()` / `userDAO.Update()` 返回的 error 做判断：

- 错误信息包含 `Duplicate entry` 且包含 `username` → 返回 `fmt.Errorf("用户名已存在")`
- 其他错误 → 原样返回

使用 `strings.Contains` 判断，不引入新依赖。

## 前端设计

`handleSubmit` 方法加 try/catch：

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
}
```

## 不改动

- 不新增接口、路由、DAO 方法
- 不改全局拦截器逻辑
- 不改其他模块
