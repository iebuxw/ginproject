# 菜单拖拽排序设计文档

## 背景

菜单管理页 (`/system/menu`) 目前只能通过编辑对话框逐个修改 `sort` 值来调整顺序。用户要求支持拖拽排序，提升操作效率。

## 需求

- 一级菜单（顶层目录）之间支持拖拽排序
- 二级菜单（同一父级下的子项）之间支持拖拽排序
- 仅支持同级排序，不支持跨级拖拽（不改变 parent_id）
- 排序列作为拖拽手柄

## 技术方案

### 方案选型

选择 `sortablejs` 直接操作 el-table tbody DOM。理由：轻量（~10KB），el-table 树形模式下行在同一个 tbody 内，sortablejs 可直接操作，实现简洁。

### 后端

**新增路由**：`PUT /api/menus/sort`

**权限**：复用 `menu:edit`（排序是编辑的子操作）

**请求体**：`[{ "id": 5, "sort": 0 }, { "id": 4, "sort": 1 }]`

**处理逻辑**：事务内逐条 `UPDATE menus SET sort = ? WHERE id = ?`

**改动文件**：

| 文件 | 改动 |
|------|------|
| `internal/model/menu.go` | 新增 `MenuItemSort{ID uint, Sort int}` |
| `internal/dao/menu_dao.go` | 新增 `BatchUpdateSort(items []MenuItemSort) error` |
| `internal/service/menu_service.go` | 新增 `BatchUpdateSort`（透传到 DAO） |
| `internal/controller/menu_controller.go` | 新增 `Sort` 方法，绑定 JSON 数组，调用 service |
| `internal/router/router.go` | 注册 `PUT /api/menus/sort`，中间件复用 `menu:edit` 权限 |

### 前端

**依赖**：`sortablejs`

**改动文件**：

| 文件 | 改动 |
|------|------|
| `web/package.json` | 新增 `sortablejs` 依赖 |
| `web/src/api/menu.js` | 新增 `sortMenus(data)` — `PUT /menus/sort` |
| `web/src/views/menu/index.vue` | mounted 初始化 Sortable + 拖拽逻辑 |

**拖拽逻辑**：

1. `mounted` 中初始化 Sortable，绑定到 el-table 的 tbody
2. `handle` 选项限定拖拽手柄为排序列（`.sort-handle` CSS 类）
3. `onMove` 回调：校验拖拽行与目标行的 `parent_id` 相同，否则拒绝放置
4. `onEnd` 回调：
   - 从 DOM 获取拖拽后所有同级行的顺序
   - 通过行的 `data-row-key` 属性提取菜单 ID
   - 重新计算 sort 值（0, 1, 2, ...）
   - 调用 `sortMenus` API 保存
   - 刷新列表数据

### 不涉及的部分

- 无数据库迁移
- 无新增权限点
- 不改变现有菜单树构建逻辑
- 不影响侧边栏导航（排序结果通过 `GET /api/menus` 自动反映）
