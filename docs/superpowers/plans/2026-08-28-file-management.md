# 文件管理功能实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 新增「运维管理 → 文件管理」公共文件库：上传（≤100MB、挡可执行扩展名）、按文件名搜索、列表（图片缩略图预览）、下载、删除（记录+物理文件）。

**Architecture:** 本地磁盘存储 `./uploads/files/<randomHex><ext>`，复用已有静态路由 `/api/uploads` 做图片预览，下载走 `ctx.FileAttachment` 专用接口。新增 `files` 元数据表 + 迁移 000013（含新建「运维管理」一级目录菜单种子）。后端照 db_backup 模块分层（controller→service→dao），前端照 backup 页面结构。

**Tech Stack:** Go/Gin + GORM + golang-migrate；Vue 2 + Element UI；验证用 Docker 重建 + curl + agent-browser（项目无单测惯例，CLAUDE.md 规定以 Docker 重建+接口调用验证，故本计划不做 TDD）。

**Spec:** `docs/superpowers/specs/2026-08-28-file-management-design.md`

**关键既有事实（已核实）：**
- 菜单 id 最大 55；JWT 中间件 `c.Set` 了 `user_id`(uint) 和 `username`(string)（`internal/middleware/jwt.go:38-39`）
- `r.Static("/api/uploads", "./uploads")` 已存在（`internal/router/router.go:43`），公开无 JWT
- `randomHex` 在 `internal/controller/upload_controller.go:87`（controller 包内，service 不能引用，service 内自带一份）
- `.gitignore` 目前**没有** `uploads/`，需补
- `model.DateTime` 需手动赋值 `model.DateTime(time.Now())`
- 前端下载 blob 模式照抄 `web/src/views/backup/index.vue` 的 `handleDownload`（L211-228）

---

### Task 1: 迁移 000013（files 表 + 菜单种子）

**Files:**
- Create: `migrations/000013_add_file_management.up.sql`
- Create: `migrations/000013_add_file_management.down.sql`

- [ ] **Step 1: 写 up 迁移**

`migrations/000013_add_file_management.up.sql`：

```sql
-- 文件管理：上传文件元数据表
CREATE TABLE IF NOT EXISTS `files` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `original_name` VARCHAR(255) NOT NULL,
  `stored_name` VARCHAR(128) NOT NULL,
  `size` BIGINT DEFAULT 0,
  `ext` VARCHAR(32) DEFAULT '',
  `uploader_id` BIGINT DEFAULT 0,
  `uploader_name` VARCHAR(64) DEFAULT '',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='上传文件记录';

-- 一级目录：运维管理（现有菜单 id 最大 55，从 56 起分配）
INSERT IGNORE INTO menus (id, parent_id, name, path, permission, type, icon, sort, created_at, updated_at)
VALUES (56, 0, '运维管理', '/system/ops-mgr', '', 1, 'el-icon-s-tools', 5, NOW(), NOW());

-- 二级菜单：文件管理（file:list 挂在页面上，与 db_backup 模式一致）
INSERT IGNORE INTO menus (id, parent_id, name, path, permission, type, icon, sort, created_at, updated_at)
VALUES (57, 56, '文件管理', '/system/file', 'file:list', 2, 'el-icon-folder', 1, NOW(), NOW());

-- 按钮权限点
INSERT IGNORE INTO menus (id, parent_id, name, path, permission, type, icon, sort, created_at, updated_at)
VALUES (58, 57, '上传', '', 'file:upload', 3, '', 1, NOW(), NOW()),
       (59, 57, '下载', '', 'file:download', 3, '', 2, NOW(), NOW()),
       (60, 57, '删除', '', 'file:delete', 3, '', 3, NOW(), NOW());

-- admin 角色绑定
INSERT IGNORE INTO role_menus (role_id, menu_id)
VALUES (1, 56), (1, 57), (1, 58), (1, 59), (1, 60);
```

- [ ] **Step 2: 写 down 迁移**

`migrations/000013_add_file_management.down.sql`：

```sql
DELETE FROM role_menus WHERE menu_id IN (56, 57, 58, 59, 60);
DELETE FROM menus WHERE id IN (56, 57, 58, 59, 60);
DROP TABLE IF EXISTS `files`;
```

- [ ] **Step 3: 验证迁移可执行**

Run: `docker compose up -d --build go-app && sleep 5 && docker compose logs go-app --tail 20`
Expected: 日志含「数据库迁移完成」且无 migrate 报错。

- [ ] **Step 4: 核对菜单种子落库**

Run: `docker compose exec -T mysql mysql -uroot -padmin123 --default-character-set=utf8mb4 ginadmin -e "SELECT id,parent_id,name,path,permission,type FROM menus WHERE id>=56;"`
Expected: 5 行（56 运维管理/57 文件管理/58 上传/59 下载/60 删除）。

- [ ] **Step 5: Commit**

```bash
git add migrations/000013_add_file_management.up.sql migrations/000013_add_file_management.down.sql
git commit -m "feat: 迁移 000013 files 表与文件管理菜单种子"
```

---

### Task 2: 后端 model + dao

**Files:**
- Create: `internal/model/file.go`
- Create: `internal/dao/file_dao.go`

- [ ] **Step 1: 写 model**

`internal/model/file.go`：

```go
package model

type File struct {
	ID           int64    `json:"id" gorm:"primaryKey;autoIncrement"`
	OriginalName string   `json:"original_name" gorm:"size:255;not null"`
	StoredName   string   `json:"stored_name" gorm:"size:128;not null"`
	Size         int64    `json:"size"`
	Ext          string   `json:"ext" gorm:"size:32"`
	UploaderID   int64    `json:"uploader_id"`
	UploaderName string   `json:"uploader_name" gorm:"size:64"`
	CreatedAt    DateTime `json:"created_at"`
}

func (File) TableName() string {
	return "files"
}
```

- [ ] **Step 2: 写 dao**

`internal/dao/file_dao.go`：

```go
package dao

import (
	"ginproject/internal/model"
	"gorm.io/gorm"
)

type FileDAO struct {
	db *gorm.DB
}

func NewFileDAO(db *gorm.DB) *FileDAO {
	return &FileDAO{db: db}
}

func (d *FileDAO) Create(file *model.File) error {
	return d.db.Create(file).Error
}

func (d *FileDAO) FindPage(page, pageSize int, name string) ([]model.File, int64, error) {
	var list []model.File
	var total int64

	query := d.db.Model(&model.File{})
	if name != "" {
		query = query.Where("original_name LIKE ?", "%"+name+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (d *FileDAO) FindByID(id int64) (*model.File, error) {
	var file model.File
	err := d.db.First(&file, id).Error
	return &file, err
}

func (d *FileDAO) Delete(id int64) error {
	return d.db.Delete(&model.File{}, id).Error
}
```

- [ ] **Step 3: 编译验证**

Run: `go build ./...`
Expected: 无输出（编译通过）。

- [ ] **Step 4: Commit**

```bash
git add internal/model/file.go internal/dao/file_dao.go
git commit -m "feat: 文件管理 model 与 dao"
```

---

### Task 3: 后端 service

**Files:**
- Create: `internal/service/file_service.go`

- [ ] **Step 1: 写 service**

注意：`randomHex` 在 controller 包（`upload_controller.go:87`），service 不能反向引用（controller 已 import service，会循环依赖），故本文件自带 `randomHexName`。

`internal/service/file_service.go`：

```go
package service

import (
	"crypto/rand"
	"fmt"
	"ginproject/internal/dao"
	"ginproject/internal/model"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// 黑名单：仅挡可执行/脚本类扩展名，其余类型不限
var forbiddenExts = map[string]bool{
	".exe": true, ".dll": true, ".bat": true, ".cmd": true, ".com": true,
	".msi": true, ".vbs": true, ".sh": true, ".reg": true,
}

const MaxFileSize = 100 * 1024 * 1024

type FileService struct {
	fileDAO   *dao.FileDAO
	uploadDir string
}

func NewFileService(fileDAO *dao.FileDAO) *FileService {
	return &FileService{fileDAO: fileDAO, uploadDir: "./uploads/files"}
}

func (s *FileService) Upload(originalName string, size int64, src io.Reader, uploaderID int64, uploaderName string) (*model.File, error) {
	ext := strings.ToLower(filepath.Ext(originalName))
	if forbiddenExts[ext] {
		return nil, fmt.Errorf("不允许上传该文件类型: %s", ext)
	}
	if size > MaxFileSize {
		return nil, fmt.Errorf("文件大小不能超过 100MB")
	}

	if err := os.MkdirAll(s.uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("创建上传目录失败: %w", err)
	}

	storedName := randomHexName(16) + ext
	savePath := filepath.Join(s.uploadDir, storedName)

	dst, err := os.Create(savePath)
	if err != nil {
		return nil, fmt.Errorf("保存文件失败: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(savePath)
		return nil, fmt.Errorf("保存文件失败: %w", err)
	}

	file := &model.File{
		OriginalName: originalName,
		StoredName:   storedName,
		Size:         size,
		Ext:          strings.TrimPrefix(ext, "."),
		UploaderID:   uploaderID,
		UploaderName: uploaderName,
		CreatedAt:    model.DateTime(time.Now()),
	}
	if err := s.fileDAO.Create(file); err != nil {
		os.Remove(savePath)
		return nil, fmt.Errorf("保存文件记录失败: %w", err)
	}
	return file, nil
}

func (s *FileService) List(page, pageSize int, name string) ([]model.File, int64, error) {
	return s.fileDAO.FindPage(page, pageSize, name)
}

func (s *FileService) Delete(id int64) error {
	file, err := s.fileDAO.FindByID(id)
	if err != nil {
		return fmt.Errorf("文件记录不存在: %w", err)
	}

	if err := s.fileDAO.Delete(id); err != nil {
		return fmt.Errorf("删除文件记录失败: %w", err)
	}

	// 物理文件删除失败仅记日志，不阻断（记录已删，孤儿文件留待手工清理）
	savePath := filepath.Join(s.uploadDir, file.StoredName)
	if _, err := os.Stat(savePath); err == nil {
		if err := os.Remove(savePath); err != nil {
			log.Printf("警告: 物理文件删除失败 %s: %v", savePath, err)
		}
	}
	return nil
}

func (s *FileService) GetByID(id int64) (*model.File, error) {
	return s.fileDAO.FindByID(id)
}

func (s *FileService) GetFilePath(storedName string) string {
	return filepath.Join(s.uploadDir, storedName)
}

func randomHexName(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
```

- [ ] **Step 2: 编译验证**

Run: `go build ./...`
Expected: 无输出（编译通过）。

- [ ] **Step 3: Commit**

```bash
git add internal/service/file_service.go
git commit -m "feat: 文件管理 service（上传校验/删除物理文件）"
```

---

### Task 4: 后端 controller + 路由 + main 装配

**Files:**
- Create: `internal/controller/file_controller.go`
- Modify: `internal/router/router.go`（Setup 参数列表 + 路由注册 + imports 无需改）
- Modify: `cmd/server/main.go`（DAO/Service/Controller 装配 + Setup 调用）

- [ ] **Step 1: 写 controller**

`internal/controller/file_controller.go`：

```go
package controller

import (
	"ginproject/internal/service"
	"ginproject/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type FileController struct {
	fileService *service.FileService
}

func NewFileController(fileService *service.FileService) *FileController {
	return &FileController{fileService: fileService}
}

func (c *FileController) List(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))
	name := ctx.Query("name")

	list, total, err := c.fileService.List(page, pageSize, name)
	if err != nil {
		utils.Error(ctx, 500, "查询失败")
		return
	}

	utils.Success(ctx, gin.H{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (c *FileController) Upload(ctx *gin.Context) {
	file, header, err := ctx.Request.FormFile("file")
	if err != nil {
		utils.Error(ctx, 400, "请选择文件")
		return
	}
	defer file.Close()

	userID, _ := ctx.Get("user_id")
	uploaderID, _ := userID.(uint)
	username, _ := ctx.Get("username")
	uploaderName, _ := username.(string)

	record, err := c.fileService.Upload(header.Filename, header.Size, file, int64(uploaderID), uploaderName)
	if err != nil {
		utils.Error(ctx, 400, err.Error())
		return
	}

	utils.Success(ctx, record)
}

func (c *FileController) Delete(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, 400, "参数错误")
		return
	}

	if err := c.fileService.Delete(id); err != nil {
		utils.Error(ctx, 500, err.Error())
		return
	}

	utils.Success(ctx, nil)
}

func (c *FileController) Download(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, 400, "参数错误")
		return
	}

	file, err := c.fileService.GetByID(id)
	if err != nil {
		utils.Error(ctx, 404, "文件不存在")
		return
	}

	ctx.FileAttachment(c.fileService.GetFilePath(file.StoredName), file.OriginalName)
}
```

- [ ] **Step 2: 注册路由**

`internal/router/router.go`：
1. Setup 参数列表在 `dbBackupCtrl *controller.DbBackupController,` 之后加一行：

```go
	fileCtrl *controller.FileController,
```

2. 在「// 数据库备份」路由块（L162-172）之后、「// 仪表盘」之前插入：

```go
		// 文件管理（上传不挂 OperationLogger：中间件会把 multipart body 整体读入内存并把二进制写入日志）
		authorized.GET("/files",
			middleware.RequirePerm("file:list"), middleware.RBAC(menuDAO), fileCtrl.List)
		authorized.POST("/files/upload",
			middleware.RequirePerm("file:upload"), middleware.RBAC(menuDAO), fileCtrl.Upload)
		authorized.GET("/files/:id/download",
			middleware.RequirePerm("file:download"), middleware.RBAC(menuDAO), fileCtrl.Download)
		authorized.DELETE("/files/:id",
			middleware.RequirePerm("file:delete"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), fileCtrl.Delete)
```

- [ ] **Step 3: main.go 装配**

`cmd/server/main.go` 三处：

1. DAO 区（L80 `dbBackupDAO := ...` 之后）：

```go
	fileDAO := dao.NewFileDAO(db)
```

2. Service 区（L92 `dbBackupService := ...` 之后）：

```go
	fileService := service.NewFileService(fileDAO)
```

3. Controller 区（L188 `uploadCtrl := ...` 之后）：

```go
	fileCtrl := controller.NewFileController(fileService)
```

4. L191 `router.Setup(...)` 调用末尾参数 `dashboardCtrl` 后追加 `fileCtrl`：

```go
	r := router.Setup(cfg, authCtrl, userCtrl, roleCtrl, menuCtrl, logCtrl, loginLogCtrl, wsCtrl, uploadCtrl, authService, userDAO, menuDAO, logDAO, logRepo, dictTypeCtrl, dictDataCtrl, cronTaskCtrl, dbBackupCtrl, dashboardCtrl, fileCtrl)
```

- [ ] **Step 4: 编译验证**

Run: `go build ./... && go vet ./...`
Expected: 无输出（编译通过）。

- [ ] **Step 5: Commit**

```bash
git add internal/controller/file_controller.go internal/router/router.go cmd/server/main.go
git commit -m "feat: 文件管理后端接口（列表/上传/下载/删除）"
```

---

### Task 5: 后端 Docker 重建 + curl 验证

**Files:** 无新文件（验证任务）

- [ ] **Step 1: 重建 go-app**

Run: `docker compose up -d --build go-app`
Expected: 构建成功、容器启动。`docker compose logs go-app --tail 20` 无报错。

- [ ] **Step 2: 登录拿 token**

Run:
```bash
TOKEN=$(curl -sk -X POST https://localhost:8443/api/auth/login -H "Content-Type: application/json" -d '{"username":"admin","password":"123456"}' | sed 's/.*"token":"\([^"]*\)".*/\1/')
echo $TOKEN
```
Expected: 输出非空 JWT（密码实际为 123456，不是 admin）。

- [ ] **Step 3: 上传正常文件**

Run:
```bash
echo "hello file mgmt" > /tmp/test-upload.txt
curl -sk -X POST https://localhost:8443/api/files/upload -H "Authorization: Bearer $TOKEN" -F "file=@/tmp/test-upload.txt"
```
Expected: `{"code":200,...,"data":{...,"original_name":"test-upload.txt","ext":"txt","uploader_name":"admin",...}}`。

- [ ] **Step 4: 上传被拒类型与超限文件**

Run:
```bash
cp /tmp/test-upload.txt /tmp/evil.exe
curl -sk -X POST https://localhost:8443/api/files/upload -H "Authorization: Bearer $TOKEN" -F "file=@/tmp/evil.exe"
head -c 104857600 /dev/zero > /tmp/big.bin
curl -sk -X POST https://localhost:8443/api/files/upload -H "Authorization: Bearer $TOKEN" -F "file=@/tmp/big.bin"
```
Expected: 第一次返回业务码 400 +「不允许上传该文件类型」；第二次业务码 400 +「文件大小不能超过 100MB」。

- [ ] **Step 5: 列表与搜索**

Run:
```bash
curl -sk "https://localhost:8443/api/files?page=1&page_size=10" -H "Authorization: Bearer $TOKEN"
curl -sk "https://localhost:8443/api/files?name=test-upload" -H "Authorization: Bearer $TOKEN"
curl -sk "https://localhost:8443/api/files?name=不存在xyz" -H "Authorization: Bearer $TOKEN"
```
Expected: 第一次 total≥1；第二次能搜到 test-upload.txt；第三次 `list:[]` total=0。

- [ ] **Step 6: 下载并比对内容**

Run:
```bash
ID=$(curl -sk "https://localhost:8443/api/files?name=test-upload" -H "Authorization: Bearer $TOKEN" | sed 's/.*"id":\([0-9]*\).*/\1/' | head -c 10)
curl -sk "https://localhost:8443/api/files/$ID/download" -H "Authorization: Bearer $TOKEN" -o /tmp/downloaded.txt
cat /tmp/downloaded.txt
```
Expected: 内容为 `hello file mgmt`，响应头 `Content-Disposition` 含 `test-upload.txt`。

- [ ] **Step 7: 删除后物理文件消失**

Run:
```bash
STORED=$(docker compose exec -T mysql mysql -uroot -padmin123 ginadmin -N -e "SELECT stored_name FROM files ORDER BY id DESC LIMIT 1;")
ls -la docker 2>/dev/null; docker compose exec -T go-app ls -la uploads/files/
curl -sk -X DELETE "https://localhost:8443/api/files/$ID" -H "Authorization: Bearer $TOKEN"
docker compose exec -T go-app ls -la uploads/files/
```
Expected: 删除前 `uploads/files/` 中有该文件；删除后返回 code 200 且目录中文件已消失；`files` 表中记录已删。

- [ ] **Step 8: 无权限用户被拒（可选，若无第二角色可跳过）**

Expected: 非 admin 角色无 `file:list` 权限时接口返回业务错误码。

---

### Task 6: 前端（api + 页面 + componentMap + gitignore）

**Files:**
- Create: `web/src/api/file.js`
- Create: `web/src/views/file/index.vue`
- Modify: `web/src/store/modules/permission.js`（componentMap，L3-13）
- Modify: `.gitignore`（补 `uploads/`）

- [ ] **Step 1: 写 api**

`web/src/api/file.js`：

```js
import request from './request'

export const getFiles = (params) => request.get('/files', { params })

export const uploadFile = (formData) => request.post('/files/upload', formData)

export const deleteFile = (id) => request.delete(`/files/${id}`)
```

- [ ] **Step 2: 写页面**

`web/src/views/file/index.vue`：

```vue
<template>
  <div>
    <el-card>
      <div slot="header"><span>文件管理</span></div>
      <el-form :inline="true" @submit.native.prevent>
        <el-form-item label="文件名">
          <el-input v-model="filters.name" placeholder="请输入文件名" clearable @keyup.enter.native="handleSearch"></el-input>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSearch">搜索</el-button>
          <el-button @click="handleReset">重置</el-button>
        </el-form-item>
        <el-form-item style="float:right">
          <el-upload
            :show-file-list="false"
            :before-upload="beforeUpload"
            :http-request="handleUpload">
            <el-button type="primary" icon="el-icon-upload2" :loading="uploading">上传文件</el-button>
          </el-upload>
        </el-form-item>
      </el-form>

      <el-table :data="list" border v-loading="loading">
        <el-table-column prop="id" label="ID" width="80"></el-table-column>
        <el-table-column label="预览" width="100" align="center">
          <template slot-scope="{row}">
            <el-image
              v-if="isImage(row.ext)"
              :src="previewUrl(row)"
              :preview-src-list="[previewUrl(row)]"
              fit="cover"
              style="width:40px;height:40px">
            </el-image>
            <i v-else class="el-icon-document" style="font-size:26px;color:#909399"></i>
          </template>
        </el-table-column>
        <el-table-column prop="original_name" label="文件名" min-width="200"></el-table-column>
        <el-table-column label="大小" width="120">
          <template slot-scope="{row}">{{ formatSize(row.size) }}</template>
        </el-table-column>
        <el-table-column prop="ext" label="类型" width="90"></el-table-column>
        <el-table-column prop="uploader_name" label="上传者" width="110"></el-table-column>
        <el-table-column prop="created_at" label="上传时间" width="180"></el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template slot-scope="{row}">
            <el-button type="text" size="small" @click="handleDownload(row)">下载</el-button>
            <el-button type="text" size="small" style="color:#F56C6C" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination style="margin-top:15px" @current-change="pageChange" :current-page="page" :page-size="pageSize" :total="total" layout="total,prev,pager,next"></el-pagination>
    </el-card>
  </div>
</template>

<script>
import { getFiles, uploadFile, deleteFile } from '@/api/file'
import axios from 'axios'

export default {
  data() {
    return {
      list: [], page: 1, pageSize: 10, total: 0,
      filters: { name: '' },
      loading: false,
      uploading: false,
    }
  },
  created() { this.fetchData() },
  methods: {
    async fetchData() {
      this.loading = true
      try {
        const params = { page: this.page, page_size: this.pageSize }
        if (this.filters.name) params.name = this.filters.name
        const res = await getFiles(params)
        this.list = res.data.list
        this.total = res.data.total
      } finally {
        this.loading = false
      }
    },
    handleSearch() { this.page = 1; this.fetchData() },
    handleReset() { this.filters.name = ''; this.handleSearch() },
    pageChange(p) { this.page = p; this.fetchData() },
    isImage(ext) {
      return ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp'].includes((ext || '').toLowerCase())
    },
    previewUrl(row) {
      return '/api/uploads/files/' + row.stored_name
    },
    formatSize(bytes) {
      if (!bytes || isNaN(bytes)) return '-'
      if (bytes === 0) return '0 B'
      const k = 1024
      const sizes = ['B', 'KB', 'MB', 'GB']
      const i = Math.floor(Math.log(bytes) / Math.log(k))
      return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
    },
    beforeUpload(file) {
      const forbidden = ['exe', 'dll', 'bat', 'cmd', 'com', 'msi', 'vbs', 'sh', 'reg']
      const parts = file.name.split('.')
      const ext = parts.length > 1 ? parts.pop().toLowerCase() : ''
      if (forbidden.includes(ext)) {
        this.$message.error('不允许上传该文件类型')
        return false
      }
      if (file.size > 100 * 1024 * 1024) {
        this.$message.error('文件大小不能超过 100MB')
        return false
      }
      return true
    },
    handleUpload({ file }) {
      this.uploading = true
      const formData = new FormData()
      formData.append('file', file)
      uploadFile(formData).then(() => {
        this.$message.success('上传成功')
        this.page = 1
        this.fetchData()
      }).catch(() => {
        this.$message.error('上传失败')
      }).finally(() => {
        this.uploading = false
      })
    },
    handleDownload(row) {
      const token = this.$store.state.user.token
      axios.get(`/api/files/${row.id}/download`, {
        responseType: 'blob',
        headers: { Authorization: 'Bearer ' + token }
      }).then(res => {
        const url = window.URL.createObjectURL(new Blob([res.data]))
        const link = document.createElement('a')
        link.href = url
        link.download = row.original_name
        document.body.appendChild(link)
        link.click()
        link.remove()
        window.URL.revokeObjectURL(url)
      }).catch(() => {
        this.$message.error('下载失败')
      })
    },
    handleDelete(row) {
      this.$confirm(`确定删除文件「${row.original_name}」？删除后不可恢复`, '提示', {
        type: 'warning'
      }).then(async () => {
        await deleteFile(row.id)
        this.$message.success('删除成功')
        this.fetchData()
      }).catch(() => {})
    },
  }
}
</script>
```

- [ ] **Step 3: componentMap 加映射**

`web/src/store/modules/permission.js` L12 `'/system/backup': ...` 之后加一行：

```js
  '/system/file': () => import('@/views/file/index.vue')
```

（注意前一行结尾补逗号。）

- [ ] **Step 4: .gitignore 补 uploads/**

`.gitignore` 末尾 `backups/` 之后加：

```
uploads/
```

- [ ] **Step 5: Commit**

```bash
git add web/src/api/file.js web/src/views/file/index.vue web/src/store/modules/permission.js .gitignore
git commit -m "feat: 文件管理前端页面与路由映射"
```

---

### Task 7: 前端 Docker 重建 + agent-browser 全流程验证

**Files:** 无新文件（验证任务）

- [ ] **Step 1: 重建 nginx**

Run: `docker compose up -d --build nginx`
Expected: 构建成功（Vue 构建在镜像内完成）。

- [ ] **Step 2: agent-browser 打开并登录**

Run:
```bash
agent-browser --ignore-https-errors --args "--no-sandbox" open "https://localhost:8443"
```
Expected: 登录页正常。用 admin/123456 登录（`fill` 用户名/密码输入框 + `click` 登录按钮）。
Expected: 登录成功，侧边栏出现「运维管理」目录。

- [ ] **Step 3: 验证菜单与页面渲染**

Run: 点击展开「运维管理」→ 点击「文件管理」。
Expected: 跳转 `/#/system/file`，页面渲染出搜索栏、上传按钮、空表格（或已有 Task 5 上传的记录）。用 `eval` 确认 DOM：`document.querySelector('.el-table')` 非空。

- [ ] **Step 4: 验证搜索**

Run: 文件名输入框填 `test` → 点搜索。
Expected: 表格只剩含 test 的记录；点重置恢复全量。

- [ ] **Step 5: 验证上传与预览**

Run: 用 agent-browser 上传一张本机图片（`setInputFiles` 或按 agent-browser 上传方式；可先生成小图片文件）。
Expected: 提示「上传成功」，列表出现新记录，预览列显示缩略图；点击缩略图可看大图。再上传一个 `.txt`，预览列显示文件图标。

- [ ] **Step 6: 验证下载**

Run: 点击任一行「下载」。
Expected: 浏览器触发下载，文件名与列表一致、内容正确。

- [ ] **Step 7: 验证删除**

Run: 点击某行「删除」→ 确认对话框点确定。
Expected: 提示「删除成功」，行从列表消失；`docker compose exec -T go-app ls uploads/files/` 对应物理文件已删。

- [ ] **Step 8: 回归抽查**

Run: 依次打开 定时任务、数据库备份、用户管理 页面。
Expected: 均正常渲染（TagsView 标签、路由无报错）。

- [ ] **Step 9: 清理验证残留**

Run: 删除验证期间上传的测试文件（页面上删除即可），删除 `/tmp/test-upload.txt`、`/tmp/evil.exe`、`/tmp/big.bin`、`/tmp/downloaded.txt`。

---

## 自审记录

- **Spec 覆盖**：表结构/菜单种子（Task 1）、4 接口与权限点（Task 2-4）、上传校验与物理删除（Task 3）、上传不挂 OperationLogger（Task 4 Step 2 注释）、前端页面/componentMap（Task 6）、gitignore uploads（Task 6 Step 4）、Docker+curl+agent-browser 验证（Task 5/7）——spec 各节均有对应任务。
- **类型一致**：`FileService.Upload(originalName string, size int64, src io.Reader, uploaderID int64, uploaderName string)` 在 Task 3 定义、Task 4 以 `header.Filename, header.Size, file, int64(uploaderID), uploaderName` 调用一致；路由路径 `/files/:id/download` 与前端 `handleDownload` 一致；componentMap key `/system/file` 与迁移菜单 path 一致。
- **无占位符**：所有步骤含完整代码/命令。
