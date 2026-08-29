# 消息中心（公告 + 定向站内信 + 系统事件通知）设计

日期：2026-08-29
状态：已确认

## 背景与目标

项目现有通知能力为零散实现：导出任务结果通过 WebSocket 直推（`export_complete` / `export_failed`），离线用户错过结果；公告、定向站内信缺失。本设计建立统一消息中心：

1. 管理端可发布**公告**（全员）与**定向站内信**（按角色/指定用户）
2. 系统事件（导出任务结果）落库，离线用户可补看
3. 导航栏铃铛未读数 + 独立消息中心页

非目标：短信/邮件渠道推送、消息撤回、富文本编辑器。

## 总体方案

**写扩散模型（方案 A）**：消息本体一张表，发布时按收件范围把 user_id 展开写入收件人表。后台系统用户量小（千级以内），写放大可忽略，换取最简查询与统一的收件范围表达。

```
发布 ──> notifications（本体）
     └─> notification_recipients（收件人展开，user_id × N 行）
阅读 ──> UPDATE recipients.read_at
推送 ──> ws.Hub.SendToUser（在线加速，不承担可靠性）
```

## 数据模型（迁移 000017_notifications）

```sql
notifications
  id BIGINT PK AUTO_INCREMENT
  type TINYINT              -- 1=公告 2=站内信 3=系统事件
  title VARCHAR(200)
  content TEXT
  sender_id BIGINT          -- 发布人 id；系统事件写 0
  target_type TINYINT       -- 1=全员 2=角色 3=指定用户
  created_at DATETIME       -- 手动赋值（DateTime 类型不触发 GORM 自动时间戳）

notification_recipients
  id BIGINT PK AUTO_INCREMENT
  notification_id BIGINT    -- 索引 idx_notification_id
  user_id BIGINT            -- 索引 idx_user_id
  read_at DATETIME NULL     -- NULL = 未读
  UNIQUE KEY uk_notif_user (notification_id, user_id)
```

- 建表用 `IF NOT EXISTS`；down 迁移 DROP 两表
- 收件人展开规则：全员=用户表全量 id；角色=角色绑定用户 id；指定用户=入参列表

## 后端分层

沿用 router -> controller -> service -> dao -> model，新增 `NotificationController` / `NotificationService` / `NotificationDAO`。Service 发布时按 target_type 展开收件人，本体+收件人同一事务批量写入。

## API 设计

| 接口 | 权限 | 说明 |
|---|---|---|
| `POST /api/notifications` | `notification:send` | 发布（type、title、content、target_type、target_ids） |
| `GET /api/notifications` | `notification:list` | 管理端消息列表（分页，发布人视角） |
| `DELETE /api/notifications/:id` | `notification:delete` | 删除消息（连带收件记录） |
| `GET /api/notifications/mine` | 仅登录 | 我的消息分页（筛选已读/未读、type） |
| `POST /api/notifications/read` | 仅登录 | 批量已读（ids 数组）或全部已读（all=true） |
| `GET /api/notifications/unread-count` | 仅登录 | 未读数，铃铛用 |

- 用户端接口（mine / read / unread-count）强制 `user_id = 当前登录用户`，防越权
- 错误返回沿用 `utils.Error`（HTTP 200 + 业务码）

### 菜单与权限种子

菜单树迁移种子（`INSERT IGNORE` + 显式 id）：

- 「消息管理」目录：消息中心（所有登录用户可见）+ 消息发送（`notification:send`）
- 按钮权限点：`notification:send`、`notification:list`、`notification:delete`

## 实时推送

- 发布成功后，Service 按收件人 id 逐个 `hub.SendToUser(userID, ...)`，事件类型 `notification`，payload `{id, type, title, content}`
- 前端收到后铃铛未读数 +1 并弹 ElMessage 提醒
- 离线补拉：登录成功后与消息中心页打开时拉 `unread-count` 初始化；WebSocket 收到事件时本地自增，不重复请求

## 系统事件接入（导出任务结果）

worker 中现有 `export_complete` / `export_failed` WebSocket 推送处，同时调用 `NotificationService` 落一条 type=3 消息（收件人=发起导出的用户）。前端现有导出结果提示逻辑不变。落库失败仅 `log.Printf` 告警，不阻断导出流程（与 ES 双写降级同思路）。

## 前端设计

**铃铛（`web/src/layout/index.vue` 顶栏，头像左侧）**

- `el-badge` + `el-popover`：未读数（>99 显示 99+）
- 弹层显示最近 5 条未读（标题+时间），底部「查看全部」跳消息中心、「全部已读」
- 点消息即标已读并打开内容

**消息中心页（`web/src/views/notification/index.vue`）**

- Tab：全部 / 公告 / 站内信 / 系统事件；筛选：已读/未读
- 表格列：类型（el-tag）、标题、发布人、发布时间、状态、操作（查看）
- 「查看」对话框显示完整内容，打开自动标已读；工具栏「全部已读」
- `el-pagination` 分页，与项目现有列表页一致

**发送页（`web/src/views/notification/send.vue`）**

- 表单：类型（公告/站内信）、标题、内容 textarea、接收范围（全员 radio / 按角色多选 / 按用户多选）
- 角色下拉来自现有角色接口；用户选择 `el-select` multiple + filterable 一次拉取
- `notification:send` 权限控制入口

**路由与 WebSocket**

- `web/src/store/modules/permission.js` 的 `componentMap` 添加两个页面映射（硬性要求）
- WebSocket `notification` 事件处理与 `export_complete` 同处注册

## 错误处理

- Hub 推送用户不在线：静默跳过（正常情况）
- worker 落库失败：`log.Printf`，不阻断
- 发布写入：事务，本体+收件人同事务，失败整体回滚
- target_ids 为空（角色/指定用户模式）：返回业务码 400 类参数错误

## 测试策略

- Service 单测：收件人展开（全员/角色/用户三种 target）、批量已读、未读计数
- 复用 `go test ./...`，不引入新测试框架
- 集成验证：Docker 重建 go-app / nginx，curl 验证 API，agent-browser 验证页面（登录后查 DOM）

## 实施分期

1. 迁移 000017 + model/dao/service/controller + API 路由注册（后端地基）
2. 菜单种子迁移 + 前端铃铛 + 消息中心页 + mine/read/unread-count 联调
3. 发送页 + 发布推送 + worker 导出事件接入

每期独立提交，不混 commit。
