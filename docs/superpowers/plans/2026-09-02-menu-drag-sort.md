# 菜单拖拽排序 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为菜单管理页添加同级拖拽排序功能，用户可通过拖拽排序列调整菜单顺序。

**Architecture:** 前端使用 sortablejs 绑定 el-table tbody 实现行拖拽，拖拽结束后批量更新 sort 值；后端新增 `PUT /api/menus/sort` 批量排序接口，事务内逐条更新。

**Tech Stack:** Vue 2 + Element UI + sortablejs (前端) / Go + Gin + GORM (后端)

---

## 改动文件清单

| 文件 | 操作 | 职责 |
|------|------|------|
| `internal/model/menu.go` | 修改 | 新增 `MenuItemSort` 结构体 |
| `internal/dao/menu_dao.go` | 修改 | 新增 `BatchUpdateSort` 方法 |
| `internal/service/menu_service.go` | 修改 | 新增 `BatchUpdateSort` 透传 |
| `internal/controller/menu_controller.go` | 修改 | 新增 `Sort` 方法 |
| `internal/router/router.go` | 修改 | 注册 `PUT /api/menus/sort` 路由 |
| `web/package.json` | 修改 | 新增 `sortablejs` 依赖 |
| `web/src/api/menu.js` | 修改 | 新增 `sortMenus` API |
| `web/src/views/menu/index.vue` | 修改 | 集成 sortablejs 拖拽逻辑 |

---

### Task 1: Backend - Model & DAO 层

**Files:**
- Modify: `internal/model/menu.go`
- Modify: `internal/dao/menu_dao.go`

- [ ] **Step 1: 在 model/menu.go 中添加 MenuItemSort 结构体**

在 `Menu` 结构体之后添加：

```go
// MenuItemSort 批量排序请求项
type MenuItemSort struct {
	ID   uint `json:"id"`
	Sort int  `json:"sort"`
}
```

- [ ] **Step 2: 在 dao/menu_dao.go 中添加 BatchUpdateSort 方法**

在文件末尾（`BuildMenuTree` 函数之前）添加：

```go
// BatchUpdateSort 批量更新菜单排序值
func (d *MenuDAO) BatchUpdateSort(items []model.MenuItemSort) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if err := tx.Model(&model.Menu{}).Where("id = ?", item.ID).
				Update("sort", item.Sort).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
```

- [ ] **Step 3: 验证编译通过**

```bash
cd D:/phpStudy/PHPTutorial/WWW/ginproject
go build ./...
```

Expected: 编译成功，无输出

- [ ] **Step 4: Commit**

```bash
git add internal/model/menu.go internal/dao/menu_dao.go
git commit -m "feat: 菜单排序 - 新增 MenuItemSort 模型和 BatchUpdateSort DAO 方法"
```

---

### Task 2: Backend - Service, Controller & Router

**Files:**
- Modify: `internal/service/menu_service.go`
- Modify: `internal/controller/menu_controller.go`
- Modify: `internal/router/router.go`

- [ ] **Step 1: 在 service/menu_service.go 中添加 BatchUpdateSort**

在 `FindByID` 方法之后添加：

```go
func (s *MenuService) BatchUpdateSort(items []model.MenuItemSort) error {
	return s.menuDAO.BatchUpdateSort(items)
}
```

- [ ] **Step 2: 在 controller/menu_controller.go 中添加 Sort 方法**

在 `Delete` 方法之后添加：

```go
// Sort 批量更新菜单排序
// @Summary 批量更新菜单排序
// @Description 批量更新菜单的 sort 字段，用于拖拽排序
// @Tags 菜单管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body []model.MenuItemSort true "排序列表"
// @Success 200 {object} utils.Response "成功"
// @Failure 200 {object} utils.Response "业务错误"
// @Router /menus/sort [put]
func (ctl *MenuController) Sort(c *gin.Context) {
	var items []model.MenuItemSort
	if err := c.ShouldBindJSON(&items); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	if err := ctl.menuService.BatchUpdateSort(items); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}
```

- [ ] **Step 3: 在 router/router.go 中注册排序路由**

在 `authorized.DELETE("/menus/:id", ...)` 之后添加：

```go
		authorized.PUT("/menus/sort",
			middleware.RequirePerm("menu:edit"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), menuCtrl.Sort)
```

- [ ] **Step 4: 验证编译通过**

```bash
cd D:/phpStudy/PHPTutorial/WWW/ginproject
go build ./...
```

Expected: 编译成功，无输出

- [ ] **Step 5: Commit**

```bash
git add internal/service/menu_service.go internal/controller/menu_controller.go internal/router/router.go
git commit -m "feat: 菜单排序 - 新增 PUT /api/menus/sort 批量排序接口"
```

---

### Task 3: Frontend - 安装依赖 & API

**Files:**
- Modify: `web/package.json` (via npm install)
- Modify: `web/src/api/menu.js`

- [ ] **Step 1: 安装 sortablejs**

```bash
cd D:/phpStudy/PHPTutorial/WWW/ginproject/web
npm install sortablejs --save
```

Expected: package.json 中出现 `"sortablejs": "^x.x.x"`

- [ ] **Step 2: 在 api/menu.js 中添加 sortMenus**

在文件末尾添加：

```js
export const sortMenus = (data) => request.put('/menus/sort', data)
```

- [ ] **Step 3: Commit**

```bash
git add web/package.json web/package-lock.json web/src/api/menu.js
git commit -m "feat: 菜单排序 - 安装 sortablejs 并添加排序 API"
```

---

### Task 4: Frontend - 实现拖拽排序

**Files:**
- Modify: `web/src/views/menu/index.vue`

- [ ] **Step 1: 修改 script 部分，引入 sortablejs 并实现拖拽**

将整个 `<script>` 部分替换为：

```vue
<script>
import Sortable from 'sortablejs'
import { getMenus, addMenu, updateMenu, deleteMenu, sortMenus } from '@/api/menu'
export default {
  data() {
    return {
      menuList: [], dialogVisible: false, isEdit: false,
      form: { parent_id: 0, name: '', icon: '', path: '', type: 1, permission: '', sort: 0, status: 1 },
      cascaderOptions: [],
      sortableInstance: null
    }
  },
  created() { this.fetchData() },
  mounted() { this.initSortable() },
  beforeDestroy() {
    if (this.sortableInstance) { this.sortableInstance.destroy(); this.sortableInstance = null }
  },
  methods: {
    async fetchData() {
      const res = await getMenus()
      this.menuList = res.data
      this.cascaderOptions = res.data
      this.$nextTick(() => { this.initSortable() })
    },
    initSortable() {
      const el = this.$el.querySelector('.el-table__body-wrapper tbody')
      if (!el) return
      if (this.sortableInstance) { this.sortableInstance.destroy() }
      this.sortableInstance = Sortable.create(el, {
        handle: '.sort-handle',
        animation: 150,
        onMove: (evt) => {
          // 拖拽行和目标行必须同级（同一 parent_id）
          const draggedRow = evt.dragged
          const relatedRow = evt.related
          const draggedId = parseInt(draggedRow.getAttribute('data-row-key'))
          const relatedId = parseInt(relatedRow.getAttribute('data-row-key'))
          const draggedMenu = this.findMenuById(this.menuList, draggedId)
          const relatedMenu = this.findMenuById(this.menuList, relatedId)
          if (!draggedMenu || !relatedMenu) return false
          return draggedMenu.parent_id === relatedMenu.parent_id
        },
        onEnd: async (evt) => {
          const draggedId = parseInt(evt.item.getAttribute('data-row-key'))
          const menu = this.findMenuById(this.menuList, draggedId)
          if (!menu) return
          // 获取同级菜单（保持拖拽后的 DOM 顺序）
          const tbody = evt.from
          const rows = tbody.querySelectorAll('tr[data-row-key]')
          const siblings = []
          rows.forEach(row => {
            const id = parseInt(row.getAttribute('data-row-key'))
            const m = this.findMenuById(this.menuList, id)
            if (m && m.parent_id === menu.parent_id) {
              siblings.push(m)
            }
          })
          // 重新分配 sort 值
          const sortData = siblings.map((m, index) => ({ id: m.id, sort: index }))
          try {
            await sortMenus(sortData)
            await this.fetchData()
            this.$message.success('排序成功')
          } catch {
            this.$message.error('排序失败')
            await this.fetchData()
          }
        }
      })
    },
    // 在菜单树中递归查找指定 id 的菜单
    findMenuById(menus, id) {
      for (const m of menus) {
        if (m.id === id) return m
        if (m.children && m.children.length > 0) {
          const found = this.findMenuById(m.children, id)
          if (found) return found
        }
      }
      return null
    },
    openDialog(row) {
      if (row) { this.isEdit = true; this.form = { ...row } }
      else { this.isEdit = false; this.form = { parent_id: 0, name: '', icon: '', path: '', type: 1, permission: '', sort: 0, status: 1 } }
      this.dialogVisible = true
    },
    async handleSubmit() {
      if (this.isEdit) { await updateMenu(this.form.id, this.form) }
      else { await addMenu(this.form) }
      this.dialogVisible = false; this.fetchData()
    },
    async handleDelete(id) {
      await this.$confirm('确认删除该菜单?', '提示', { type: 'warning' })
      try {
        await deleteMenu(id); this.fetchData(); this.$message.success('删除成功')
      } catch { this.$message.error('删除失败，可能存在子菜单') }
    }
  }
}
</script>
```

- [ ] **Step 2: 修改 template 排序列，添加拖拽手柄样式**

将排序列：

```vue
<el-table-column prop="sort" label="排序" width="60"></el-table-column>
```

替换为：

```vue
<el-table-column label="排序" width="80">
  <template slot-scope="s">
    <span class="sort-handle" style="cursor:move;font-size:16px">☰</span>
    <span style="margin-left:4px">{{ s.row.sort }}</span>
  </template>
</el-table-column>
```

- [ ] **Step 3: 验证前端编译**

```bash
cd D:/phpStudy/PHPTutorial/WWW/ginproject/web
npm run build
```

Expected: 编译成功，无报错

- [ ] **Step 4: Commit**

```bash
git add web/src/views/menu/index.vue
git commit -m "feat: 菜单排序 - 前端集成 sortablejs 拖拽排序"
```

---

### Task 5: Docker 重建 & 验证

- [ ] **Step 1: 重建并重启 Go 应用**

```bash
cd D:/phpStudy/PHPTutorial/WWW/ginproject
docker compose up -d --build go-app
```

Expected: 容器启动成功

- [ ] **Step 2: 重建并重启 Nginx（含前端构建）**

```bash
docker compose up -d --build nginx
```

Expected: 容器启动成功

- [ ] **Step 3: curl 验证排序 API**

先获取 token：

```bash
curl -s -k https://localhost:8443/api/auth/login -H "Content-Type: application/json" -d '{"username":"admin","password":"123456"}'
```

用返回的 token 调用排序 API：

```bash
curl -s -k -X PUT https://localhost:8443/api/menus/sort \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '[{"id":1,"sort":0},{"id":2,"sort":1},{"id":3,"sort":2}]'
```

Expected: `{"code":200,"msg":"success","data":null}`

- [ ] **Step 4: 浏览器验证**

打开 `https://localhost:8443`，登录后进入菜单管理页，确认：
- 排列显示拖拽手柄图标（☰）
- 同级菜单可拖拽排序
- 跨级拖拽被阻止
- 刷新后排序保持
