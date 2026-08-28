# 用户导入功能实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 用户管理页面新增 Excel 批量导入用户功能（同步导入，返回成功/跳过/失败汇总），含模板下载。

**Architecture:** 后端复用 excelize 解析上传的 xlsx，按表头名称映射列；Service 逐行校验并入库（角色名经 RoleDAO.FindAll 映射为 ID）；前端 `el-upload` 手动模式 + 结果明细对话框。

**Tech Stack:** Go (gin + excelize/v2 + GORM)、Vue 2 + Element UI。

**规格：** `docs/superpowers/specs/2026-08-28-user-import-design.md`

**验证策略说明：** 本项目无 mock 基建、测试极少（仅 logger_test.go），按项目 CLAUDE.md 约定以 Docker 重建 + curl + agent-browser 集成验证替代单元测试。

**注意事项：**

- 工作区当前有未提交的「用户导出」功能改动（user_controller.go、router.go、request.js、user.js、user/index.vue），与本功能改同一批文件。**Task 0 先将其单独提交，否则提交粒度会混。**
- 导入路由**不挂 OperationLogger**（与 `/files/upload` 一致：中间件会把 multipart body 整体读入内存并把二进制写入日志）。
- Excel 列按**表头名称**映射（非固定列序），这样用户在导出文件基础上补列也能用，前提是表头名一致。

---

### Task 0: 提交遗留的导出功能改动

**Files:**
- 无新增文件，仅 git 操作（涉及 user_controller.go、router.go、request.js、user.js、user/index.vue 的既有改动）

- [ ] **Step 1: 确认遗留改动内容**

Run: `git diff --stat`
Expected: 6 个文件的改动，均为导出功能（user_controller.go +87 行为 Export 等）。若与预期不符，停下询问用户。

- [ ] **Step 2: 单独提交导出功能**

```bash
git add internal/controller/log_controller.go internal/controller/user_controller.go internal/router/router.go web/src/api/request.js web/src/api/user.js web/src/views/user/index.vue
git commit -m "feat: 用户列表导出 Excel"
```

---

### Task 1: Service 层导入逻辑

**Files:**
- Modify: `internal/service/user_service.go`
- Modify: `cmd/server/main.go:86`

- [ ] **Step 1: 修改 UserService 依赖与结构定义**

`internal/service/user_service.go` 顶部 import 增加 `"strings"`；结构体与方法签名修改：

```go
type UserService struct {
	userDAO *dao.UserDAO
	roleDAO *dao.RoleDAO
}

func NewUserService(userDAO *dao.UserDAO, roleDAO *dao.RoleDAO) *UserService {
	return &UserService{userDAO: userDAO, roleDAO: roleDAO}
}
```

文件末尾追加导入相关类型与方法：

```go
// ImportRow 导入文件中的一行原始数据（Excel 行号从 2 开始，含表头行）
type ImportRow struct {
	Username    string
	Password    string
	Email       string
	Phone       string
	Description string
	Status      int
	RoleNames   string
	Row         int
}

// ImportFailure 校验失败的行及原因
type ImportFailure struct {
	Row    int    `json:"row"`
	Reason string `json:"reason"`
}

// ImportResult 导入结果汇总
type ImportResult struct {
	Total            int             `json:"total"`
	Success          int             `json:"success"`
	Skipped          int             `json:"skipped"`
	SkippedUsernames []string        `json:"skipped_usernames"`
	Failed           []ImportFailure `json:"failed"`
}

// Import 批量导入用户：用户名已存在（库中或文件内重复）跳过，校验失败记入 failed，逐条创建互不影响
func (s *UserService) Import(rows []ImportRow) *ImportResult {
	result := &ImportResult{
		Total:            len(rows),
		SkippedUsernames: []string{},
		Failed:           []ImportFailure{},
	}

	roles, err := s.roleDAO.FindAll()
	if err != nil {
		return result
	}
	roleIDs := make(map[string]uint, len(roles))
	for _, r := range roles {
		roleIDs[r.Name] = r.ID
	}

	seen := make(map[string]bool, len(rows))
	for _, row := range rows {
		if row.Username == "" {
			result.Failed = append(result.Failed, ImportFailure{Row: row.Row, Reason: "用户名为空"})
			continue
		}
		if row.Password == "" {
			result.Failed = append(result.Failed, ImportFailure{Row: row.Row, Reason: "密码为空"})
			continue
		}
		if seen[row.Username] {
			result.Skipped++
			result.SkippedUsernames = append(result.SkippedUsernames, row.Username)
			continue
		}
		if _, err := s.userDAO.FindByUsername(row.Username); err == nil {
			result.Skipped++
			result.SkippedUsernames = append(result.SkippedUsernames, row.Username)
			continue
		}
		seen[row.Username] = true

		ids := make([]uint, 0)
		unknown := make([]string, 0)
		for _, name := range strings.Split(row.RoleNames, ",") {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if id, ok := roleIDs[name]; ok {
				ids = append(ids, id)
			} else {
				unknown = append(unknown, name)
			}
		}
		if len(unknown) > 0 {
			result.Failed = append(result.Failed, ImportFailure{Row: row.Row, Reason: "角色不存在: " + strings.Join(unknown, ",")})
			continue
		}

		hashed, err := utils.HashPassword(row.Password)
		if err != nil {
			result.Failed = append(result.Failed, ImportFailure{Row: row.Row, Reason: "密码哈希失败: " + err.Error()})
			continue
		}
		user := &model.User{
			Username:    row.Username,
			Password:    hashed,
			Email:       row.Email,
			Phone:       row.Phone,
			Description: row.Description,
			Status:      row.Status,
			Roles:       buildRoles(ids),
		}
		if err := s.userDAO.Create(user); err != nil {
			result.Failed = append(result.Failed, ImportFailure{Row: row.Row, Reason: "创建失败: " + err.Error()})
			continue
		}
		result.Success++
	}
	return result
}
```

（`FindAll` 失败时直接返回空 result 的做法：角色映射为空将导致所有带角色的行失败、无角色行仍可导入——这里选择直接返回，行级统计不启动。改为整体报错也可，实现时保持简单即可。）

**更正：** `FindAll` 失败应让调用方感知整体失败而非静默返回空结果。将 `Import` 返回值改为 `(result *ImportResult, err error)`，开头：

```go
roles, err := s.roleDAO.FindAll()
if err != nil {
	return nil, err
}
```

结尾 `return result, nil`。后续 Task 2 的 controller 调用处按两返回值处理（err 非空时 `utils.Error(c, 500, "获取角色列表失败")`）。

- [ ] **Step 2: 同步 main.go 调用点**

`cmd/server/main.go:86`：

```go
userService := service.NewUserService(userDAO, roleDAO)
```

（roleDAO 已在 main.go:72 创建，可直接使用。）

- [ ] **Step 3: 编译验证**

Run: `go build ./...`
Expected: 编译通过，无错误。

- [ ] **Step 4: 提交**

```bash
git add internal/service/user_service.go cmd/server/main.go
git commit -m "feat: 用户导入 Service 层逻辑（逐行校验+角色名映射）"
```

---

### Task 2: Controller 与路由

**Files:**
- Modify: `internal/controller/user_controller.go`
- Modify: `internal/router/router.go`（用户管理段，`/users/export` 注册之后）

- [ ] **Step 1: Controller 新增 Import 与 ImportTemplate 方法**

`internal/controller/user_controller.go` import 增加 `"strings"`；文件末尾追加：

```go
// Import 导入用户
// @Summary Excel 批量导入用户
// @Description 上传 xlsx 文件批量创建用户；用户名已存在的行跳过，校验失败的行返回原因
// @Tags 用户管理
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "xlsx 文件"
// @Success 200 {object} utils.Response{data=service.ImportResult} "导入结果汇总"
// @Failure 200 {object} utils.Response "业务错误"
// @Router /users/import [post]
func (ctl *UserController) Import(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		utils.Error(c, 400, "请选择文件")
		return
	}
	defer file.Close()

	f, err := excelize.OpenReader(file)
	if err != nil {
		utils.Error(c, 400, "文件格式错误，仅支持 xlsx")
		return
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		utils.Error(c, 400, "Excel 无工作表")
		return
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		utils.Error(c, 400, "读取 Excel 失败")
		return
	}
	if len(rows) < 2 {
		utils.Error(c, 400, "Excel 无数据行")
		return
	}

	colIdx := make(map[string]int)
	for i, h := range rows[0] {
		colIdx[strings.TrimSpace(h)] = i
	}
	for _, required := range []string{"用户名", "密码"} {
		if _, ok := colIdx[required]; !ok {
			utils.Error(c, 400, "缺少必需列: "+required)
			return
		}
	}

	cell := func(row []string, header string) string {
		i, ok := colIdx[header]
		if !ok || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}

	importRows := make([]service.ImportRow, 0, len(rows)-1)
	for i, row := range rows[1:] {
		empty := true
		for _, v := range row {
			if strings.TrimSpace(v) != "" {
				empty = false
				break
			}
		}
		if empty {
			continue
		}
		status := 1
		if cell(row, "状态") == "禁用" {
			status = 0
		}
		importRows = append(importRows, service.ImportRow{
			Username:    cell(row, "用户名"),
			Password:    cell(row, "密码"),
			Email:       cell(row, "邮箱"),
			Phone:       cell(row, "手机号"),
			Description: cell(row, "描述"),
			Status:      status,
			RoleNames:   cell(row, "角色"),
			Row:         i + 2,
		})
	}

	result, err := ctl.userService.Import(importRows)
	if err != nil {
		utils.Error(c, 500, "获取角色列表失败")
		return
	}
	utils.Success(c, result)
}

// ImportTemplate 下载用户导入模板
// @Summary 下载用户导入模板
// @Description 生成仅含表头的 xlsx 模板（用户名/密码/邮箱/手机号/描述/状态/角色）
// @Tags 用户管理
// @Security BearerAuth
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Success 200 {file} binary "Excel 模板"
// @Router /users/import-template [get]
func (ctl *UserController) ImportTemplate(c *gin.Context) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "用户导入模板"
	f.SetSheetName("Sheet1", sheet)

	sw, _ := f.NewStreamWriter(sheet)
	sw.SetRow("A1", []interface{}{"用户名", "密码", "邮箱", "手机号", "描述", "状态", "角色"})
	sw.Flush()

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
	})
	f.SetCellStyle(sheet, "A1", "G1", headerStyle)

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape("用户导入模板.xlsx"))

	if _, err := f.WriteTo(c.Writer); err != nil {
		utils.ErrorWithStatus(c, http.StatusInternalServerError, 500, "模板生成失败")
	}
}
```

- [ ] **Step 2: 注册路由**

`internal/router/router.go`，在 `/users/export` 注册行之后追加（注意不挂 OperationLogger，写明原因，与 `/files/upload` 注释风格一致）：

```go
		// 用户导入（不挂 OperationLogger：中间件会把 multipart body 整体读入内存并把二进制写入日志）
		authorized.POST("/users/import",
			middleware.RequirePerm("user:add"), middleware.RBAC(menuDAO), userCtrl.Import)
		authorized.GET("/users/import-template",
			middleware.RequirePerm("user:add"), middleware.RBAC(menuDAO), userCtrl.ImportTemplate)
```

（gin 路由静态段优先于 `:id` 参数段，现有 `GET /users/export` 与 `GET /users/:id` 已共存，`/users/import` 与 `/users/import-template` 同理不冲突。）

- [ ] **Step 3: 重新生成 Swagger 文档**

Run: `swag init -g cmd/server/main.go`
Expected: docs/docs.go 更新，无报错。

- [ ] **Step 4: 编译验证**

Run: `go build ./...`
Expected: 编译通过。

- [ ] **Step 5: 提交**

```bash
git add internal/controller/user_controller.go internal/router/router.go docs/
git commit -m "feat: 用户导入 API 与模板下载（POST /users/import）"
```

---

### Task 3: 前端导入对话框

**Files:**
- Modify: `web/src/api/user.js`
- Modify: `web/src/views/user/index.vue`

- [ ] **Step 1: 新增 API 方法**

`web/src/api/user.js` 末尾追加：

```js
export const importUsers = (formData) => request.post('/users/import', formData, {
  headers: { 'Content-Type': 'multipart/form-data' }
})
export const downloadImportTemplate = () => request.get('/users/import-template', { responseType: 'blob' })
```

- [ ] **Step 2: 页面新增导入按钮与对话框**

`web/src/views/user/index.vue` 修改点：

1. import 行追加 `importUsers, downloadImportTemplate`：

```js
import { getUsers, addUser, updateUser, deleteUser, exportUsers, importUsers, downloadImportTemplate } from '@/api/user'
```

2. 头部按钮区（「导出Excel」按钮之前）加：

```html
<el-button size="small" @click="importDialogVisible = true">导入用户</el-button>
```

3. `</template>` 前追加对话框（与编辑对话框并列）：

```html
    <el-dialog title="导入用户" :visible.sync="importDialogVisible" width="520px">
      <el-alert type="info" :closable="false" show-icon title="请先下载模板，按模板格式填写后上传导入"
        style="margin-bottom:15px"></el-alert>
      <el-upload drag action="" :auto-upload="false" :limit="1"
        :on-change="handleImportChange" :on-remove="handleImportRemove" :file-list="importFileList">
        <i class="el-icon-upload"></i>
        <div class="el-upload__text">将文件拖到此处，或<em>点击选择</em></div>
        <div class="el-upload__tip" slot="tip">仅支持 .xlsx 文件，<el-link type="primary" :underline="false" style="font-size:12px" @click="downloadTemplate">下载模板</el-link></div>
      </el-upload>
      <div v-if="importResult" style="margin-top:15px">
        <el-alert :type="importResult.failed.length ? 'warning' : 'success'" :closable="false" show-icon
          :title="'共 ' + importResult.total + ' 条：成功 ' + importResult.success + '，跳过 ' + importResult.skipped + '，失败 ' + importResult.failed.length"></el-alert>
        <div v-if="importResult.skipped_usernames.length" style="margin-top:10px;font-size:13px">
          跳过的用户名：{{ importResult.skipped_usernames.join('、') }}
        </div>
        <div v-if="importResult.failed.length" style="margin-top:10px">
          <div v-for="fitem in importResult.failed" :key="fitem.row" style="font-size:13px;color:#F56C6C">
            第 {{ fitem.row }} 行：{{ fitem.reason }}
          </div>
        </div>
      </div>
      <span slot="footer">
        <el-button @click="importDialogVisible = false">关闭</el-button>
        <el-button type="primary" :loading="importing" :disabled="!importFile" @click="handleImport">开始导入</el-button>
      </span>
    </el-dialog>
```

4. data 追加：

```js
importDialogVisible: false, importing: false, importFile: null, importFileList: [], importResult: null
```

5. methods 追加（blob 下载复用导出已有的文件名解析逻辑，抽成本文件内私有 helper，`exportExcel` 中相应段落改调 `saveBlob`）：

```js
    saveBlob(res, fallbackName) {
      const url = window.URL.createObjectURL(new Blob([res.data]))
      const link = document.createElement('a')
      link.href = url
      const disposition = res.headers['content-disposition'] || ''
      let filename = fallbackName
      const rfc5987 = disposition.match(/filename\*=UTF-8''(.+)/i)
      if (rfc5987) {
        filename = decodeURIComponent(rfc5987[1])
      } else {
        const fallback = disposition.match(/filename="?([^";]+)"?/)
        if (fallback) filename = fallback[1]
      }
      link.download = filename
      document.body.appendChild(link)
      link.click()
      link.remove()
      window.URL.revokeObjectURL(url)
    },
    async downloadTemplate() {
      const res = await downloadImportTemplate()
      this.saveBlob(res, '用户导入模板.xlsx')
    },
    handleImportChange(file, fileList) {
      this.importFile = file.raw
      this.importFileList = fileList.slice(-1)
    },
    handleImportRemove() {
      this.importFile = null
      this.importFileList = []
    },
    async handleImport() {
      this.importing = true
      this.importResult = null
      try {
        const fd = new FormData()
        fd.append('file', this.importFile)
        const res = await importUsers(fd)
        this.importResult = res.data
        this.fetchData()
      } finally {
        this.importing = false
      }
    },
```

`exportExcel` 改为：

```js
    async exportExcel() {
      this.exporting = true
      try {
        const res = await exportUsers({ keyword: this.keyword })
        this.saveBlob(res, '用户列表.xlsx')
      } catch {
        this.$message.error('导出失败')
      } finally {
        this.exporting = false
      }
    },
```

（JSON 业务错误由 request.js 拦截器统一 Message.error，`handleImport` 无需 catch 弹错。）

- [ ] **Step 3: 提交**

```bash
git add web/src/api/user.js web/src/views/user/index.vue
git commit -m "feat: 用户管理页面导入对话框与模板下载"
```

---

### Task 4: 集成验证

**Files:**
- 无代码改动；生成临时测试文件（不进仓库）

- [ ] **Step 1: 生成测试用 xlsx（临时脚本，不入库）**

在仓库外临时目录（如 `$TEMP/gen_xlsx.go`）写并运行：

```go
//go:build ignore
package main

import (
	"fmt"
	"os"

	"github.com/xuri/excelize/v2"
)

func main() {
	f := excelize.NewFile()
	headers := []interface{}{"用户名", "密码", "邮箱", "手机号", "描述", "状态", "角色"}
	f.SetSheetRow("Sheet1", "A1", &headers)
	rows := [][]interface{}{
		{"importuser1", "Passw0rd!", "u1@test.com", "13800000001", "导入测试1", "启用", "编辑"},
		{"importuser2", "Passw0rd!", "u2@test.com", "13800000002", "导入测试2", "禁用", ""},
		{"importuser3", "", "u3@test.com", "", "密码为空", "启用", ""},
		{"importuser4", "Passw0rd!", "", "", "角色不存在", "启用", "不存在角色"},
		{"importuser1", "Passw0rd!", "", "", "文件内重复", "启用", ""},
		{"admin", "Passw0rd!", "", "", "库中已存在", "启用", ""},
	}
	for i, row := range rows {
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		f.SetSheetRow("Sheet1", cell, &row)
	}
	if err := f.SaveAs("import_test.xlsx"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
```

Run: `go run $TEMP/gen_xlsx.go`（在仓库目录内运行以解析 go.mod 的 excelize 依赖）
Expected: 当前目录生成 `import_test.xlsx`（测试后删除）。

- [ ] **Step 2: Docker 重建后端**

Run: `docker compose up -d --build go-app`
Expected: 容器启动无报错。

- [ ] **Step 3: curl 验证登录与导入**

```bash
TOKEN=$(curl -sk -X POST https://localhost:8443/api/auth/login -H 'Content-Type: application/json' -d '{"username":"admin","password":"123456"}' | python -c "import sys,json;print(json.load(sys.stdin)['data']['token'])")
curl -sk -X POST https://localhost:8443/api/users/import -H "Authorization: Bearer $TOKEN" -F "file=@import_test.xlsx"
```

Expected 返回（密码为 admin/123456 时）：
- `total: 6, success: 2`（importuser1、importuser2）
- `skipped: 2`（文件内重复的 importuser1、已存在的 admin）
- `failed: 2`（importuser3 密码为空、importuser4 角色不存在）

再验证模板下载返回 xlsx 二进制：
```bash
curl -sk -O -J https://localhost:8443/api/users/import-template -H "Authorization: Bearer $TOKEN"
```
Expected: 保存的 `用户导入模板.xlsx` 可打开且仅一行表头。

- [ ] **Step 4: Docker 重建前端并浏览器验证**

Run: `docker compose up -d --build nginx`

agent-browser（全局选项在 open 之前）：
```bash
agent-browser --ignore-https-errors --args "--no-sandbox" open "https://localhost:8443"
```
登录后进入用户管理页，验证：
- 头部出现「导入用户」按钮
- 点击弹出对话框，「下载模板」能下载 xlsx
- 上传 `import_test.xlsx` 点「开始导入」，显示结果汇总（成功 2、跳过 2、失败 2 及明细）
- 列表刷新后出现 importuser1 / importuser2

- [ ] **Step 5: 清理测试数据与临时文件**

- 删除 `import_test.xlsx`、`$TEMP/gen_xlsx.go`、下载的模板文件
- 测试导入的 importuser1/importuser2 两个用户在页面上手动删除（保留前询问用户是否需要保留）

- [ ] **Step 6: 最终自查**

Run: `git status --short && git diff`
Expected: 工作区干净（所有改动已分 3 个功能 commit 提交），无 console.log / fmt.Println 调试残留。

---

## Self-Review 记录

- 规格覆盖：导入字段/冲突处理/汇总结构（Task 1、2）、模板下载（Task 2、3）、前端交互（Task 3）、验证（Task 4）——全覆盖
- 无占位符；类型一致（ImportRow/ImportResult/importUsers 等名称各任务一致）
- 已知取舍：axios 全局 timeout 15s，同步导入超大文件可能超时；本场景数据量小，按 YAGNI 不单独调大
