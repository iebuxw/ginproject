# 用户导入功能设计

日期：2026-08-28
模块：/system/user 用户管理

## 需求

用户管理页面新增 Excel 批量导入用户功能：

- 导入字段与导出对称，外加密码列：`用户名 | 密码 | 邮箱 | 手机号 | 描述 | 状态 | 角色`
- 同步导入（方案 A），一次性返回结果汇总
- 用户名已存在的行跳过不导入；校验失败的行记为失败
- 提供模板下载

## API

### POST /api/users/import

- multipart/form-data，文件字段名 `file`
- 权限：复用 `user:add`，不新增权限点/菜单/迁移
- 不挂 `OperationLogger`（与 `/files/upload` 一致：中间件会把 multipart body 整体读入内存并把二进制写入日志）

### GET /api/users/import-template

- 生成仅含表头行的 xlsx，复用导出的 StreamWriter 逻辑
- 权限：`user:add`

## 解析规则

- 首行为表头，跳过；空行跳过
- 用户名、密码必填；邮箱/手机号/描述可选
- 状态列：`启用`/`禁用` 文本，空默认启用
- 角色列：逗号分隔角色名，按 `RoleDAO.FindAll()` 建名称->ID 映射；找不到的角色名记为该行失败
- 同一文件内重复用户名：只导第一行，后续记为跳过
- 复用 `utils.HashPassword` 哈希密码，复用 `userDAO.Create`（含角色关联）

## 结果分类

- 跳过：用户名已存在（DB 中或文件内重复），不动数据
- 失败：校验不通过（密码为空、角色名不存在）

## Service 层

`UserService` 新增 `roleDAO` 依赖（`NewUserService` 签名随之调整，main.go 同步），用于角色名->ID 映射。新增导入方法，逐行处理：

- 每行独立校验 -> 有效则哈希密码 + 逐条 `userDAO.Create`（不用批量事务，单行失败不影响其他行）
- 已存在用户名经 `FindByUsername` 判定 -> 计入跳过
- 返回结构：`{ total, success, skipped, skipped_usernames: [], failed: [{row, reason}] }`

## 前端

`web/src/views/user/index.vue`：

- 页面头部新增「导入用户」按钮
- 对话框内含「下载模板」链接 + `el-upload`（手动模式，选择文件后点「开始导入」）
- 结果展示：成功提示 + 展开明细（跳过用户名列表、失败行原因）
- `web/src/api/user.js` 新增 `importUsers` / `importTemplate` 方法

## 验证

- `docker compose up -d --build go-app`，curl multipart 上传验证返回 JSON
- agent-browser 登录后打开用户管理页验证渲染与导入流程
