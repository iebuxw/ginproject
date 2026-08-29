# 消息中心（通知）实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现统一消息中心：管理端发布公告/定向站内信（全员/角色/指定用户），系统事件（导出结果）落库，导航栏铃铛未读数 + 消息中心页。

**Architecture:** 写扩散模型——`notifications` 存消息本体，发布时按收件范围展开 user_id 写入 `notification_recipients`（含 `read_at` 已读标记）；WebSocket Hub 推送在线用户，离线补拉接口兜底。分层沿用 router -> middleware -> controller -> service -> dao -> model。

**Tech Stack:** Go + Gin + GORM + golang-migrate、Vue 2 + Element UI、gorilla/websocket（复用现有 ws.Hub，其 `Message` 结构已含 `type` 字段，新增 `notification` 事件即可）。

**设计文档:** `docs/superpowers/specs/2026-08-29-notification-center-design.md`

**关键上下文（写代码前必读）:**

- 菜单种子 id 已用到 63（000016），本次新菜单从 **70** 开始留余量
- `ws.Message`（`internal/ws/hub.go`）字段：`Type/TaskID/Filename/DownloadURL/Error`；Hub 推送方法是 `hub.Send(userID, msg)`（**不是** SendToUser）
- Controller 取当前用户：`userID, _ := c.Get("user_id"); uid, _ := userID.(uint)`
- 迁移 up 种子统一 `INSERT IGNORE` + 显式 id；建表 `IF NOT EXISTS`
- `DateTime` 类型（`internal/model/json_time.go`）不触发 GORM 自动时间戳，`created_at` 需手动赋值
- 菜单 id=1 是「系统管理」顶级目录（000002 种子）
- 项目测试极少，`go test ./...` 只有 middleware 一个测试文件；本计划为 service 补测试时**不引入新框架**，用标准 testing + sqlite 不可行（项目无 sqlite 驱动），故测试仅覆盖纯逻辑函数（收件人去重等），DAO 层靠 Docker 集成验证
- 前端 `componentMap` 在 `web/src/store/modules/permission.js`，路由由后端菜单树动态生成
- Element UI 是 Vue 2 版本，`el-popover`/`el-badge`/`el-table` 均可用

---

## Task 1: 迁移 000017 + Notification model

**Files:**
- Create: `migrations/000017_add_notifications.up.sql`
- Create: `migrations/000017_add_notifications.down.sql`
- Create: `internal/model/notification.go`

- [ ] **Step 1: 写 up 迁移**

`migrations/000017_add_notifications.up.sql`:

```sql
-- 建表
CREATE TABLE IF NOT EXISTS `notifications` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `type` TINYINT NOT NULL DEFAULT 2 COMMENT '1=公告 2=站内信 3=系统事件',
  `title` VARCHAR(200) NOT NULL,
  `content` TEXT,
  `sender_id` BIGINT NOT NULL DEFAULT 0,
  `target_type` TINYINT NOT NULL DEFAULT 1 COMMENT '1=全员 2=角色 3=指定用户',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='消息通知';

CREATE TABLE IF NOT EXISTS `notification_recipients` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `notification_id` BIGINT NOT NULL,
  `user_id` BIGINT NOT NULL,
  `read_at` DATETIME DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_notif_user` (`notification_id`, `user_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='消息收件人';
```

- [ ] **Step 2: 写 down 迁移**

`migrations/000017_add_notifications.down.sql`:

```sql
DROP TABLE IF EXISTS `notification_recipients`;
DROP TABLE IF EXISTS `notifications`;
```

- [ ] **Step 3: 写 model**

`internal/model/notification.go`:

```go
package model

// Notification 消息通知本体
type Notification struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Type       int       `json:"type"`        // 1=公告 2=站内信 3=系统事件
	Title      string    `gorm:"size:200;not null" json:"title"`
	Content    string    `gorm:"type:text" json:"content"`
	SenderID   uint      `json:"sender_id"`
	TargetType int       `json:"target_type"` // 1=全员 2=角色 3=指定用户
	CreatedAt  DateTime  `json:"created_at"`
}

func (Notification) TableName() string { return "notifications" }

// NotificationRecipient 消息收件人（写扩散展开行）
type NotificationRecipient struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	NotificationID uint      `gorm:"index;not null" json:"notification_id"`
	UserID         uint      `gorm:"index;not null" json:"user_id"`
	ReadAt         *DateTime `json:"read_at"` // NULL=未读
}

func (NotificationRecipient) TableName() string { return "notification_recipients" }
```

- [ ] **Step 4: 编译验证**

Run: `go build ./...`
Expected: 无报错

- [ ] **Step 5: Commit**

```bash
git add migrations/000017_add_notifications.up.sql migrations/000017_add_notifications.down.sql internal/model/notification.go
git commit -m "feat: 消息中心数据模型与建表迁移"
```

---

## Task 2: NotificationDAO

**Files:**
- Create: `internal/dao/notification_dao.go`

- [ ] **Step 1: 写 DAO**

`internal/dao/notification_dao.go`:

```go
package dao

import (
	"time"

	"ginproject/internal/model"

	"gorm.io/gorm"
)

// NotificationItem 用户视角的消息行（join 结果）
type NotificationItem struct {
	ID          uint     `json:"id"`
	Type        int      `json:"type"`
	Title       string   `json:"title"`
	Content     string   `json:"content"`
	SenderID    uint     `json:"sender_id"`
	CreatedAt   model.DateTime `json:"created_at"`
	ReadAt      *model.DateTime `json:"read_at"`
}

type NotificationDAO struct{ db *gorm.DB }

func NewNotificationDAO(db *gorm.DB) *NotificationDAO {
	return &NotificationDAO{db: db}
}

// CreateWithRecipients 事务写入消息本体+收件人
func (d *NotificationDAO) CreateWithRecipients(n *model.Notification, userIDs []uint) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(n).Error; err != nil {
			return err
		}
		if len(userIDs) == 0 {
			return nil
		}
		recipients := make([]model.NotificationRecipient, len(userIDs))
		for i, uid := range userIDs {
			recipients[i] = model.NotificationRecipient{
				NotificationID: n.ID,
				UserID:         uid,
			}
		}
		return tx.Create(&recipients).Error
	})
}

// Delete 连带删除收件记录
func (d *NotificationDAO) Delete(id uint) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("notification_id = ?", id).Delete(&model.NotificationRecipient{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Notification{}, id).Error
	})
}

// FindPage 管理端分页（发布人视角）
func (d *NotificationDAO) FindPage(page, pageSize int, keyword string) ([]model.Notification, int64, error) {
	var list []model.Notification
	var total int64
	q := d.db.Model(&model.Notification{})
	if keyword != "" {
		q = q.Where("title LIKE ?", "%"+keyword+"%")
	}
	q.Count(&total)
	err := q.Offset((page - 1) * pageSize).Limit(pageSize).Order("id DESC").Find(&list).Error
	return list, total, err
}

// FindUserPage 用户视角分页：join 收件人表，readStatus 0=全部 1=未读 2=已读；type 0=全部
func (d *NotificationDAO) FindUserPage(userID uint, page, pageSize, readStatus, notifType int) ([]NotificationItem, int64, error) {
	var list []NotificationItem
	var total int64
	q := d.db.Table("notifications n").
		Select("n.id, n.type, n.title, n.content, n.sender_id, n.created_at, r.read_at").
		Joins("JOIN notification_recipients r ON r.notification_id = n.id AND r.user_id = ?", userID)
	if readStatus == 1 {
		q = q.Where("r.read_at IS NULL")
	} else if readStatus == 2 {
		q = q.Where("r.read_at IS NOT NULL")
	}
	if notifType > 0 {
		q = q.Where("n.type = ?", notifType)
	}
	q.Count(&total)
	err := q.Offset((page - 1) * pageSize).Limit(pageSize).Order("n.id DESC").Find(&list).Error
	return list, total, err
}

// MarkRead 批量已读：置 read_at（仅当前用户自己的记录）
func (d *NotificationDAO) MarkRead(userID uint, ids []uint) error {
	now := model.DateTime(time.Now())
	q := d.db.Model(&model.NotificationRecipient{}).
		Where("user_id = ? AND read_at IS NULL", userID)
	if len(ids) > 0 {
		q = q.Where("notification_id IN ?", ids)
	}
	return q.Update("read_at", now).Error
}

// UnreadCount 当前用户未读数
func (d *NotificationDAO) UnreadCount(userID uint) (int64, error) {
	var count int64
	err := d.db.Model(&model.NotificationRecipient{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Count(&count).Error
	return count, err
}
```

注意：`FindUserPage`/`MarkRead`/`UnreadCount` 都以 `user_id = ?` 收敛到当前用户，防越权。

- [ ] **Step 2: 编译验证**

Run: `go build ./...`
Expected: 无报错

- [ ] **Step 3: Commit**

```bash
git add internal/dao/notification_dao.go
git commit -m "feat: 消息中心 DAO（写扩散事务、用户视角查询、已读、未读数）"
```

---

## Task 3: NotificationService（收件人展开 + 单测）

**Files:**
- Create: `internal/service/notification_service.go`
- Create: `internal/service/notification_service_test.go`
- Modify: `internal/ws/hub.go:10-16`（Message 结构补三个 omitempty 字段）

- [ ] **Step 1: 写失败测试**

`internal/service/notification_service_test.go`（只测纯逻辑 `expandTargets` 收件人去重合并，不碰数据库）:

```go
package service

import (
	"sort"
	"testing"
)

func TestExpandTargets(t *testing.T) {
	cases := []struct {
		name     string
		target   SendTarget
		want     []uint
		wantErr  bool
	}{
		{
			name: "全员广播",
			target: SendTarget{TargetType: 1, UserIDs: []uint{1, 2, 3}},
			want: []uint{1, 2, 3},
		},
		{
			name: "指定用户去重",
			target: SendTarget{TargetType: 3, UserIDs: []uint{5, 3, 5, 1}},
			want: []uint{1, 3, 5},
		},
		{
			name: "指定用户为空报错",
			target: SendTarget{TargetType: 3, UserIDs: []uint{}},
			wantErr: true,
		},
		{
			name: "角色用户为空报错",
			target: SendTarget{TargetType: 2, UserIDs: []uint{}},
			wantErr: true,
		},
		{
			name: "非法 target_type",
			target: SendTarget{TargetType: 9, UserIDs: []uint{1}},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := expandTargets(tc.target)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("期望报错，实际返回 %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("不期望报错: %v", err)
			}
			sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/service/ -run TestExpandTargets -v`
Expected: FAIL（`expandTargets`、`SendTarget` 未定义）

- [ ] **Step 3: 扩展 ws.Message 并写 Service 实现**

先改 `internal/ws/hub.go` 的 Message 结构（加三个 omitempty 字段，不破坏现有 JSON 契约）:

```go
type Message struct {
	Type        string `json:"type"`
	TaskID      string `json:"task_id,omitempty"`
	Filename    string `json:"filename,omitempty"`
	DownloadURL string `json:"download_url,omitempty"`
	Error       string `json:"error,omitempty"`
	Title       string `json:"title,omitempty"`
	Content     string `json:"content,omitempty"`
	ID          uint   `json:"id,omitempty"`
}
```

然后写 `internal/service/notification_service.go`:

```go
package service

import (
	"errors"
	"log"
	"sort"

	"ginproject/internal/dao"
	"ginproject/internal/model"
	"ginproject/internal/ws"
)

// SendTarget 发布请求的收件范围
type SendTarget struct {
	TargetType int    `json:"target_type"` // 1=全员 2=角色 3=指定用户
	RoleIDs    []uint `json:"role_ids"`
	UserIDs    []uint `json:"user_ids"`
}

type NotificationService struct {
	notificationDAO *dao.NotificationDAO
	userDAO         *dao.UserDAO
	hub             *ws.Hub
}

func NewNotificationService(nd *dao.NotificationDAO, ud *dao.UserDAO, hub *ws.Hub) *NotificationService {
	return &NotificationService{notificationDAO: nd, userDAO: ud, hub: hub}
}

// Send 发布消息并推送在线收件人
func (s *NotificationService) Send(n *model.Notification, target SendTarget) error {
	userIDs, err := expandTargets(target)
	if err != nil {
		return err
	}
	if err := s.notificationDAO.CreateWithRecipients(n, userIDs); err != nil {
		return err
	}
	// 推送在线用户（离线静默，属正常情况）
	for _, uid := range userIDs {
		s.hub.Send(uid, ws.Message{
			Type:    "notification",
			ID:      n.ID,
			Title:   n.Title,
			Content: n.Content,
		})
	}
	return nil
}

// SendSystemEvent 系统事件落库（worker 调用；落库失败不阻断业务流程）
func (s *NotificationService) SendSystemEvent(title, content string, userID uint) {
	n := &model.Notification{
		Type:       3,
		Title:      title,
		Content:    content,
		SenderID:   0,
		TargetType: 3,
	}
	if err := s.notificationDAO.CreateWithRecipients(n, []uint{userID}); err != nil {
		log.Printf("系统事件消息落库失败: %v", err)
	}
	s.hub.Send(userID, ws.Message{
		Type:    "notification",
		ID:      n.ID,
		Title:   title,
		Content: content,
	})
}

func (s *NotificationService) Delete(id uint) error { return s.notificationDAO.Delete(id) }

func (s *NotificationService) FindPage(page, pageSize int, keyword string) ([]model.Notification, int64, error) {
	return s.notificationDAO.FindPage(page, pageSize, keyword)
}

func (s *NotificationService) FindUserPage(userID uint, page, pageSize, readStatus, notifType int) ([]dao.NotificationItem, int64, error) {
	return s.notificationDAO.FindUserPage(userID, page, pageSize, readStatus, notifType)
}

func (s *NotificationService) MarkRead(userID uint, ids []uint) error {
	return s.notificationDAO.MarkRead(userID, ids)
}

func (s *NotificationService) UnreadCount(userID uint) (int64, error) {
	return s.notificationDAO.UnreadCount(userID)
}
```

`expandTargets` 纯函数放同文件（文件内私有辅助，不建公共抽象）：

```go
// expandTargets 校验收件范围并展开为去重排序的 user_id 列表。
// 全员广播由调用方传入 userIDs=全部用户 id，此处统一做校验与去重。
func expandTargets(target SendTarget) ([]uint, error) {
	switch target.TargetType {
	case 1:
		if len(target.UserIDs) == 0 {
			return nil, errors.New("全员广播收件人为空")
		}
	case 2, 3:
		if len(target.UserIDs) == 0 {
			return nil, errors.New("收件人不能为空")
		}
	default:
		return nil, errors.New("无效的收件范围")
	}
	seen := make(map[uint]struct{}, len(target.UserIDs))
	result := make([]uint, 0, len(target.UserIDs))
	for _, id := range target.UserIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/service/ -run TestExpandTargets -v`
Expected: PASS（5 个子测试全过）

- [ ] **Step 5: Commit**

```bash
git add internal/service/notification_service.go internal/service/notification_service_test.go internal/ws/hub.go
git commit -m "feat: 消息中心 Service（收件人展开去重、发布推送、系统事件）"
```

---

## Task 4: NotificationController + 路由 + main 装配

**Files:**
- Create: `internal/controller/notification_controller.go`
- Modify: `internal/router/router.go`（notification 参数注入 + 路由注册）
- Modify: `cmd/server/main.go`（DAO/Service/Controller 装配）

- [ ] **Step 1: 写 Controller**

`internal/controller/notification_controller.go`（Swagger 注释风格与 dict_controller.go 一致）:

```go
package controller

import (
	"ginproject/internal/dao"
	"ginproject/internal/model"
	"ginproject/internal/service"
	"ginproject/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type NotificationController struct {
	notificationService *service.NotificationService
}

func NewNotificationController(s *service.NotificationService) *NotificationController {
	return &NotificationController{notificationService: s}
}

// Send 发布消息
// @Summary 发布消息（公告/站内信）
// @Description 按收件范围发布消息并实时推送在线收件人
// @Tags 消息中心
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body object{type=int,title=string,content=string,target_type=int,role_ids=[]int,user_ids=[]int} true "消息内容"
// @Success 200 {object} utils.Response "成功"
// @Router /notifications [post]
func (ctl *NotificationController) Send(c *gin.Context) {
	var req struct {
		Type       int    `json:"type" binding:"required"`
		Title      string `json:"title" binding:"required"`
		Content    string `json:"content"`
		TargetType int    `json:"target_type" binding:"required"`
		RoleIDs    []uint `json:"role_ids"`
		UserIDs    []uint `json:"user_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	if req.Type != 1 && req.Type != 2 {
		utils.Error(c, 400, "消息类型无效")
		return
	}
	if req.TargetType == 3 && len(req.UserIDs) == 0 {
		utils.Error(c, 400, "请选择收件用户")
		return
	}
	if req.TargetType == 2 && len(req.RoleIDs) == 0 {
		utils.Error(c, 400, "请选择收件角色")
		return
	}

	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)

	n := &model.Notification{
		Type:       req.Type,
		Title:      req.Title,
		Content:    req.Content,
		SenderID:   uid,
		TargetType: req.TargetType,
		CreatedAt:  model.DateTime(now()),
	}

	target, err := ctl.notificationService.BuildTarget(req.TargetType, req.RoleIDs, req.UserIDs)
	if err != nil {
		utils.Error(c, 400, err.Error())
		return
	}
	if err := ctl.notificationService.Send(n, target); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}

// List 管理端消息列表
// @Summary 管理端消息分页列表
// @Tags 消息中心
// @Security BearerAuth
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param keyword query string false "标题关键词"
// @Success 200 {object} utils.Response{data=object{list=[]model.Notification,total=int}} "成功"
// @Router /notifications [get]
func (ctl *NotificationController) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	keyword := c.Query("keyword")
	list, total, err := ctl.notificationService.FindPage(page, pageSize, keyword)
	if err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, gin.H{"list": list, "total": total})
}

// Delete 删除消息
// @Summary 删除消息（连带收件记录）
// @Tags 消息中心
// @Security BearerAuth
// @Produce json
// @Param id path int true "消息 ID"
// @Success 200 {object} utils.Response "成功"
// @Router /notifications/{id} [delete]
func (ctl *NotificationController) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := ctl.notificationService.Delete(uint(id)); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}

// Mine 我的消息分页
// @Summary 当前用户的消息分页列表
// @Description 按已读状态与消息类型筛选
// @Tags 消息中心
// @Security BearerAuth
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param read_status query int false "已读状态 0=全部 1=未读 2=已读" default(0)
// @Param type query int false "消息类型 0=全部 1=公告 2=站内信 3=系统事件" default(0)
// @Success 200 {object} utils.Response{data=object{list=[]dao.NotificationItem,total=int}} "成功"
// @Router /notifications/mine [get]
func (ctl *NotificationController) Mine(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	readStatus, _ := strconv.Atoi(c.DefaultQuery("read_status", "0"))
	notifType, _ := strconv.Atoi(c.DefaultQuery("type", "0"))
	list, total, err := ctl.notificationService.FindUserPage(uid, page, pageSize, readStatus, notifType)
	if err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, gin.H{"list": list, "total": total})
}

// Read 标记已读
// @Summary 批量/全部标记已读
// @Description ids 数组批量已读；all=true 全部已读
// @Tags 消息中心
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body object{ids=[]int,all=boolean} true "已读请求"
// @Success 200 {object} utils.Response "成功"
// @Router /notifications/read [post]
func (ctl *NotificationController) Read(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)
	var req struct {
		IDs []uint `json:"ids"`
		All bool   `json:"all"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	if !req.All && len(req.IDs) == 0 {
		utils.Error(c, 400, "请指定消息或选择全部已读")
		return
	}
	var ids []uint
	if !req.All {
		ids = req.IDs
	}
	if err := ctl.notificationService.MarkRead(uid, ids); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}

// UnreadCount 未读数
// @Summary 当前用户未读消息数
// @Tags 消息中心
// @Security BearerAuth
// @Produce json
// @Success 200 {object} utils.Response{data=int} "成功"
// @Router /notifications/unread-count [get]
func (ctl *NotificationController) UnreadCount(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)
	count, err := ctl.notificationService.UnreadCount(uid)
	if err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, count)
}
```

注意两个引用的辅助：`now()` 用 `time.Now()`（import "time" 后直接内联 `model.DateTime(time.Now())`，**不要真写 now() 函数**——上面 Send 里的 `model.DateTime(now())` 是示意，实现时写 `model.DateTime(time.Now())`）；`BuildTarget` 见 Step 2。

- [ ] **Step 2: Service 补 BuildTarget（收件范围落库前的展开）**

`internal/service/notification_service.go` 中新增（把「全员/角色」在 Controller 边界解析为具体 user_id 列表）:

```go
// BuildTarget 把收件范围解析为具体 user_id 列表
func (s *NotificationService) BuildTarget(targetType int, roleIDs, userIDs []uint) (SendTarget, error) {
	switch targetType {
	case 1: // 全员
		ids, err := s.userDAO.FindAllIDs()
		if err != nil {
			return SendTarget{}, err
		}
		return SendTarget{TargetType: 1, UserIDs: ids}, nil
	case 2: // 角色
		ids, err := s.userDAO.FindIDsByRoleIDs(roleIDs)
		if err != nil {
			return SendTarget{}, err
		}
		return SendTarget{TargetType: 2, UserIDs: ids}, nil
	case 3: // 指定用户
		return SendTarget{TargetType: 3, UserIDs: userIDs}, nil
	}
	return SendTarget{}, errors.New("无效的收件范围")
}
```

并在 `internal/dao/user_dao.go` 末尾新增两个查询（复用现有 DAO 风格）:

```go
// FindAllIDs 全部启用用户的 id
func (d *UserDAO) FindAllIDs() ([]uint, error) {
	var ids []uint
	err := d.db.Model(&model.User{}).Where("status = 1").Pluck("id", &ids).Error
	return ids, err
}

// FindIDsByRoleIDs 角色绑定的用户 id（去重）
func (d *UserDAO) FindIDsByRoleIDs(roleIDs []uint) ([]uint, error) {
	var ids []uint
	err := d.db.Table("user_roles").
		Where("role_id IN ?", roleIDs).
		Distinct().Pluck("user_id", &ids).Error
	return ids, err
}
```

- [ ] **Step 3: main.go 装配**

`cmd/server/main.go` 三处修改：

DAO 段（`systemSettingDAO := ...` 之后）:

```go
	notificationDAO := dao.NewNotificationDAO(db)
```

Service 段（`systemSettingService := ...` 之后，注意 hub 在后面才创建，所以 service 创建放到 hub 之后）:

```go
	// WebSocket Hub
	hub := ws.NewHub()
	notificationService := service.NewNotificationService(notificationDAO, userDAO, hub)
```

（即：在现有 `hub := ws.NewHub()` 行后加一行 `notificationService := ...`，不移动 hub 的位置）

Controller 段:

```go
	notificationCtrl := controller.NewNotificationController(notificationService)
```

`router.Setup` 调用追加参数 `notificationCtrl`。

- [ ] **Step 4: router.go 注册路由**

`internal/router/router.go`:

Setup 函数签名参数追加 `notificationCtrl *controller.NotificationController`（放 `settingCtrl` 之后）。

authorized 组内（`settings` 段之后）追加:

```go
		// 消息中心
		authorized.POST("/notifications",
			middleware.RequirePerm("notification:send"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), notificationCtrl.Send)
		authorized.GET("/notifications",
			middleware.RequirePerm("notification:list"), middleware.RBAC(menuDAO), notificationCtrl.List)
		authorized.DELETE("/notifications/:id",
			middleware.RequirePerm("notification:delete"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), notificationCtrl.Delete)
		authorized.GET("/notifications/mine",
			notificationCtrl.Mine)
		authorized.POST("/notifications/read",
			notificationCtrl.Read)
		authorized.GET("/notifications/unread-count",
			notificationCtrl.UnreadCount)
```

**路由顺序注意**：Gin 会把 `/notifications/:id` 与 `/notifications/mine` 视为同段冲突。必须把 `mine`/`read`/`unread-count` 这三条静态路由放在 `DELETE /notifications/:id` **之前**？不需要——Gin 的 httprouter 允许静态与参数路由共存于不同 method（DELETE :id 与 GET mine 不同 method 树，不冲突）。但 `GET /notifications/mine` 与假想的 `GET /notifications/:id` 会冲突，本计划没有 GET :id，无冲突。

- [ ] **Step 5: 编译 + 跑全部测试**

Run: `go build ./... && go test ./...`
Expected: 编译通过，middleware + notification 测试全 PASS

- [ ] **Step 6: Docker 集成验证**

Run: `docker compose up -d --build go-app`
Expected: 容器正常启动，日志显示「数据库迁移完成」（000017 应用成功）

然后 curl 验证（分步执行，不嵌套）:

```bash
# 1. 登录取 token（密码受阻先试 123456，未经用户同意不得重置密码）
curl -s -X POST http://localhost:8000/api/auth/login -H "Content-Type: application/json" -d '{"username":"admin","password":"123456"}'
# 记下 data.token

# 2. 发布全员公告（先手工把菜单种子 70/71 插入或直接等 Task 5 迁移——此时尚无权限点，403 是预期）
curl -s -X POST http://localhost:8000/api/notifications -H "Authorization: Bearer <token>" -H "Content-Type: application/json" -d '{"type":1,"title":"测试公告","content":"内容","target_type":1}'
# 预期：因 notification:send 权限点未种入菜单，RBAC 返回 403——说明路由已挂载

# 3. 未读数（仅登录，无权限要求）
curl -s http://localhost:8000/api/notifications/unread-count -H "Authorization: Bearer <token>"
# 预期：{"code":200,"data":0}
```

- [ ] **Step 7: 重新生成 Swagger（可选但推荐）**

Run: `swag init -g cmd/server/main.go`
Expected: docs 目录更新；确认无报错后把 `docs/` 变更一并提交

- [ ] **Step 8: Commit**

```bash
git add internal/controller/notification_controller.go internal/router/router.go cmd/server/main.go internal/service/notification_service.go internal/dao/user_dao.go docs/
git commit -m "feat: 消息中心 API（发布/列表/删除/我的消息/已读/未读数）"
```

---

## Task 5: 菜单与权限种子迁移

**Files:**
- Create: `migrations/000018_add_notification_menus.up.sql`
- Create: `migrations/000018_add_notification_menus.down.sql`

- [ ] **Step 1: 写 up 迁移**

`migrations/000018_add_notification_menus.up.sql`（菜单 id 从 70 起；挂在「系统管理」id=1 下；消息中心仅登录即可见，permission 留空）:

```sql
-- 二级菜单：消息中心（所有登录用户可见，permission 留空走 JWT 层校验）
INSERT IGNORE INTO menus (id, parent_id, name, path, permission, type, icon, sort, created_at, updated_at)
VALUES (70, 1, '消息中心', '/system/notification', '', 2, 'el-icon-bell', 5, NOW(), NOW());

-- 二级菜单：消息发送（管理端）
INSERT IGNORE INTO menus (id, parent_id, name, path, permission, type, icon, sort, created_at, updated_at)
VALUES (71, 1, '消息发送', '/system/notification-send', 'notification:send', 2, 'el-icon-s-promotion', 6, NOW(), NOW());

-- 按钮权限点（挂在消息发送下）
INSERT IGNORE INTO menus (id, parent_id, name, path, permission, type, icon, sort, created_at, updated_at)
VALUES (72, 71, '列表', '', 'notification:list', 3, '', 1, NOW(), NOW()),
       (73, 71, '删除', '', 'notification:delete', 3, '', 2, NOW(), NOW());

-- admin 角色绑定（消息中心所有角色都要见：绑定到现有全部角色）
INSERT IGNORE INTO role_menus (role_id, menu_id)
SELECT r.id, 70 FROM roles r WHERE r.status = 1;
INSERT IGNORE INTO role_menus (role_id, menu_id)
VALUES (1, 71), (1, 72), (1, 73);
```

注意：`role_menus (role_id, menu_id)` 表名与列名以 000011/000016 的既有写法为准（`INSERT IGNORE INTO role_menus (role_id, menu_id)`），上面 SELECT 写法在 MySQL 中合法。

- [ ] **Step 2: 写 down 迁移**

`migrations/000018_add_notification_menus.down.sql`:

```sql
DELETE FROM role_menus WHERE menu_id IN (70, 71, 72, 73);
DELETE FROM menus WHERE id IN (70, 71, 72, 73);
```

- [ ] **Step 3: Docker 验证迁移与权限**

Run: `docker compose up -d --build go-app`
Expected: 启动迁移完成

```bash
# 登录后重试发布
curl -s -X POST http://localhost:8000/api/notifications -H "Authorization: Bearer <token>" -H "Content-Type: application/json" -d '{"type":1,"title":"测试公告","content":"内容","target_type":1}'
# 预期：{"code":200}

# 我的消息（admin 收到全员公告）
curl -s "http://localhost:8000/api/notifications/mine?page=1&page_size=10" -H "Authorization: Bearer <token>"
# 预期：list 含一条 type=1 标题"测试公告"的消息，read_at 为 null

# 未读数
curl -s http://localhost:8000/api/notifications/unread-count -H "Authorization: Bearer <token>"
# 预期：{"code":200,"data":1}

# 全部已读
curl -s -X POST http://localhost:8000/api/notifications/read -H "Authorization: Bearer <token>" -H "Content-Type: application/json" -d '{"all":true}'
# 预期：{"code":200}，再查未读数为 0
```

- [ ] **Step 4: Commit**

```bash
git add migrations/000018_add_notification_menus.up.sql migrations/000018_add_notification_menus.down.sql
git commit -m "feat: 消息中心菜单与权限种子"
```

---

## Task 6: 前端 API 封装 + 路由映射

**Files:**
- Create: `web/src/api/notification.js`
- Modify: `web/src/store/modules/permission.js:3-15`（componentMap 增加两条）

- [ ] **Step 1: 写 API 封装**

`web/src/api/notification.js`（风格同 dict.js）:

```js
import request from './request'

// 发布消息
export const sendNotification = (data) => request.post('/notifications', data)
// 管理端列表
export const getNotifications = (params) => request.get('/notifications', { params })
// 删除消息
export const deleteNotification = (id) => request.delete('/notifications/' + id)
// 我的消息
export const getMyNotifications = (params) => request.get('/notifications/mine', { params })
// 标记已读
export const markRead = (data) => request.post('/notifications/read', data)
// 未读数
export const getUnreadCount = () => request.get('/notifications/unread-count')
```

- [ ] **Step 2: componentMap 加映射**

`web/src/store/modules/permission.js` 的 componentMap 对象内追加:

```js
  '/system/notification': () => import('@/views/notification/index.vue'),
  '/system/notification-send': () => import('@/views/notification/send.vue')
```

- [ ] **Step 3: Commit（与页面一起提交也可，本任务独立提交保持粒度）**

```bash
git add web/src/api/notification.js web/src/store/modules/permission.js
git commit -m "feat: 消息中心前端 API 封装与路由映射"
```

---

## Task 7: 消息中心页（用户视角）

**Files:**
- Create: `web/src/views/notification/index.vue`

- [ ] **Step 1: 写页面**

`web/src/views/notification/index.vue`（Vue 2 + Element UI，风格对照 dict/index.vue）:

```vue
<template>
  <div class="notification-container">
    <el-card shadow="never">
      <div slot="header" class="clearfix">
        <span>消息中心</span>
        <el-button size="mini" style="float:right" @click="handleReadAll">全部已读</el-button>
      </div>
      <div style="margin-bottom:10px">
        <el-radio-group v-model="activeType" size="small" @change="handleFilter">
          <el-radio-button :label="0">全部</el-radio-button>
          <el-radio-button :label="1">公告</el-radio-button>
          <el-radio-button :label="2">站内信</el-radio-button>
          <el-radio-button :label="3">系统事件</el-radio-button>
        </el-radio-group>
        <el-radio-group v-model="readStatus" size="small" style="margin-left:15px" @change="handleFilter">
          <el-radio-button :label="0">全部</el-radio-button>
          <el-radio-button :label="1">未读</el-radio-button>
          <el-radio-button :label="2">已读</el-radio-button>
        </el-radio-group>
      </div>
      <el-table :data="list" border v-loading="loading">
        <el-table-column label="类型" width="90" align="center">
          <template slot-scope="s">
            <el-tag :type="typeTag(s.row.type)" size="mini">{{ typeText(s.row.type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="title" label="标题" show-overflow-tooltip>
          <template slot-scope="s">
            <span :style="s.row.read_at ? '' : 'font-weight:bold'">{{ s.row.title }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="发布时间" width="160" align="center"></el-table-column>
        <el-table-column label="状态" width="70" align="center">
          <template slot-scope="s">
            <el-tag :type="s.row.read_at ? 'info' : 'danger'" size="mini">{{ s.row.read_at ? '已读' : '未读' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="80" align="center">
          <template slot-scope="s">
            <el-button size="mini" type="primary" plain @click="openDetail(s.row)">查看</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination small @current-change="pageChange" :current-page="page" :page-size="pageSize" :total="total" layout="total, prev, pager, next" style="margin-top:10px;text-align:right"></el-pagination>
    </el-card>

    <!-- 消息详情 -->
    <el-dialog :title="detail ? detail.title : ''" :visible.sync="detailVisible" width="500px">
      <div style="white-space:pre-wrap;line-height:1.8">{{ detail ? detail.content : '' }}</div>
      <span slot="footer">
        <el-button type="primary" @click="detailVisible = false">关闭</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import { getMyNotifications, markRead } from '@/api/notification'

export default {
  name: 'NotificationCenter',
  data() {
    return {
      list: [],
      page: 1,
      pageSize: 10,
      total: 0,
      activeType: 0,
      readStatus: 0,
      loading: false,
      detail: null,
      detailVisible: false
    }
  },
  created() {
    this.fetchList()
  },
  methods: {
    typeText(t) {
      return { 1: '公告', 2: '站内信', 3: '系统事件' }[t] || '未知'
    },
    typeTag(t) {
      return { 1: 'success', 2: '', 3: 'warning' }[t] || 'info'
    },
    handleFilter() {
      this.page = 1
      this.fetchList()
    },
    pageChange(p) {
      this.page = p
      this.fetchList()
    },
    async fetchList() {
      this.loading = true
      try {
        const res = await getMyNotifications({
          page: this.page,
          page_size: this.pageSize,
          read_status: this.readStatus,
          type: this.activeType
        })
        if (res.code === 200) {
          this.list = res.data.list || []
          this.total = res.data.total || 0
        }
      } finally {
        this.loading = false
      }
    },
    async openDetail(row) {
      this.detail = row
      this.detailVisible = true
      if (!row.read_at) {
        try {
          await markRead({ ids: [row.id] })
          row.read_at = 'just-now'
          this.$store.commit('notification/DEC_UNREAD')
        } catch (e) { /* 已读失败不阻塞阅读 */ }
      }
    },
    async handleReadAll() {
      try {
        await markRead({ all: true })
        this.$message.success('已全部标记为已读')
        this.$store.commit('notification/CLEAR_UNREAD')
        this.fetchList()
      } catch (e) { /* request.js 已统一提示 */ }
    }
  }
}
</script>

<style scoped>
.notification-container { padding: 0; }
</style>
```

注意：页面里引用了 `notification` store 模块（`DEC_UNREAD`/`CLEAR_UNREAD`），该模块在 Task 8 创建。**本任务先不注册 store 模块**，页面此时先不引用 commit（把两行 `this.$store.commit('notification/...')` 删掉），Task 8 一并加上——避免引用不存在模块报错。**实现顺序调整：本任务创建页面时直接不含 store commit 两行，Task 8 补上。**

- [ ] **Step 2: Docker 构建验证**

Run: `docker compose up -d --build nginx`
Expected: 构建成功，容器启动

agent-browser 验证（全局选项在 open 之前）:

```bash
agent-browser --ignore-https-errors --args "--no-sandbox" open "https://localhost:8443"
# 登录 admin/123456，左侧「系统管理」下出现「消息中心」菜单
# 点击进入，表格显示 Task 5 中发布的测试公告
# 点「查看」弹层显示内容，列表该行状态变已读
```

- [ ] **Step 3: Commit**

```bash
git add web/src/views/notification/index.vue
git commit -m "feat: 消息中心前端页面（我的消息、筛选、已读、详情）"
```

---

## Task 8: 铃铛 + notification store 模块

**Files:**
- Create: `web/src/store/modules/notification.js`
- Modify: `web/src/store/index.js`（注册模块）
- Modify: `web/src/layout/index.vue:35-48,60-117`（顶栏铃铛 + WS 事件）

- [ ] **Step 1: 写 store 模块**

`web/src/store/modules/notification.js`:

```js
const state = {
  unreadCount: 0
}

const mutations = {
  SET_UNREAD(state, n) { state.unreadCount = n },
  INC_UNREAD(state) { state.unreadCount++ },
  DEC_UNREAD(state) { if (state.unreadCount > 0) state.unreadCount-- },
  CLEAR_UNREAD(state) { state.unreadCount = 0 }
}

export default { namespaced: true, state, mutations }
```

`web/src/store/index.js` 注册模块（对照现有 user/settings 模块的注册写法，import + modules 对象加 `notification`）。

- [ ] **Step 2: 消息中心页补回 store commit**

`web/src/views/notification/index.vue` 的 `openDetail`/`handleReadAll` 中加回（Task 7 预留的两行）:

```js
this.$store.commit('notification/DEC_UNREAD')
this.$store.commit('notification/CLEAR_UNREAD')
```

- [ ] **Step 3: 布局加铃铛**

`web/src/layout/index.vue` 修改：

模板 header 内（el-dropdown 之前，左侧对齐用 float 或 flex）:

```vue
<el-popover placement="bottom" width="320" trigger="click" @show="fetchRecent">
  <div style="max-height:300px;overflow-y:auto">
    <div v-for="item in recentList" :key="item.id" style="padding:8px 0;border-bottom:1px solid #eee;cursor:pointer" @click="readOne(item)">
      <div :style="item.read_at ? '' : 'font-weight:bold'">{{ item.title }}</div>
      <div style="color:#909399;font-size:12px">{{ item.created_at }}</div>
    </div>
    <div v-if="recentList.length === 0" style="text-align:center;color:#909399;padding:15px">暂无未读消息</div>
  </div>
  <div style="text-align:center;margin-top:8px">
    <el-button type="text" size="mini" @click="readAll">全部已读</el-button>
    <el-button type="text" size="mini" @click="goCenter">查看全部</el-button>
  </div>
  <el-badge slot="reference" :value="unreadCount" :max="99" :hidden="unreadCount === 0" class="item">
    <i class="el-icon-bell" style="font-size:20px;cursor:pointer"></i>
  </el-badge>
</el-popover>
```

script 部分：

```js
import { mapState } from 'vuex'
import { connectWS, disconnectWS, onWSMessage, offWSMessage } from '@/utils/ws'
import { getUnreadCount, getMyNotifications, markRead } from '@/api/notification'
```

computed 增加:

```js
    ...mapState('notification', ['unreadCount'])
```

data 增加 `recentList: []`，created 中（connectWS 之后）:

```js
    this.fetchUnread()
    this._onNotify = () => { this.fetchUnread() }
    onWSMessage('notification', this._onNotify)
```

beforeDestroy 增加:

```js
    offWSMessage('notification', this._onNotify)
```

methods 增加:

```js
    async fetchUnread() {
      try {
        const res = await getUnreadCount()
        if (res.code === 200) this.$store.commit('notification/SET_UNREAD', res.data)
      } catch (e) { /* 静默 */ }
    },
    async fetchRecent() {
      try {
        const res = await getMyNotifications({ page: 1, page_size: 5, read_status: 1 })
        if (res.code === 200) this.recentList = res.data.list || []
      } catch (e) { /* 静默 */ }
    },
    async readOne(item) {
      try {
        await markRead({ ids: [item.id] })
        item.read_at = 'just-now'
        this.$store.commit('notification/DEC_UNREAD')
        this.fetchRecent()
      } catch (e) { /* 静默 */ }
    },
    async readAll() {
      try {
        await markRead({ all: true })
        this.$store.commit('notification/CLEAR_UNREAD')
        this.recentList = []
      } catch (e) { /* 静默 */ }
    },
    goCenter() {
      this.$router.push('/system/notification')
    }
```

注意：WS 推送的 `notification` 事件（`ws.Message` 只有 `Type/Title/...`）到达时 `fetchUnread` 重新拉未读数（不本地自增，避免与真实计数漂移；设计文档中「本地自增」以「补拉」实现更稳，此处按补拉实现，偏差已在计划中收敛为同一语义）。

- [ ] **Step 4: Docker 验证**

Run: `docker compose up -d --build nginx`
Expected: 构建成功

agent-browser 验证:

```bash
agent-browser --ignore-https-errors --args "--no-sandbox" open "https://localhost:8443"
# 登录后顶栏出现铃铛；Task 5 发布的未读公告使 badge 显示数字
# 点铃铛弹层列出未读消息；点「查看全部」跳转消息中心页
# 双开思路：另开一个登录会话（或 curl 直接 POST /notifications）发布新消息
# 观察本页铃铛数字变化（WS 实时推）
```

- [ ] **Step 5: Commit**

```bash
git add web/src/store/modules/notification.js web/src/store/index.js web/src/layout/index.vue web/src/views/notification/index.vue
git commit -m "feat: 导航栏消息铃铛与未读数（WebSocket 实时刷新）"
```

---

## Task 9: 消息发送页（管理端）

**Files:**
- Create: `web/src/views/notification/send.vue`

- [ ] **Step 1: 写页面**

`web/src/views/notification/send.vue`:

```vue
<template>
  <div class="notification-send-container">
    <el-card shadow="never">
      <div slot="header"><span>消息发送</span></div>
      <el-form :model="form" :rules="rules" ref="sendForm" label-width="90px" style="max-width:600px">
        <el-form-item label="消息类型" prop="type">
          <el-radio-group v-model="form.type">
            <el-radio :label="1">公告</el-radio>
            <el-radio :label="2">站内信</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="标题" prop="title">
          <el-input v-model="form.title" maxlength="200" show-word-limit placeholder="请输入标题"></el-input>
        </el-form-item>
        <el-form-item label="内容" prop="content">
          <el-input v-model="form.content" type="textarea" :rows="6" placeholder="请输入内容"></el-input>
        </el-form-item>
        <el-form-item label="接收范围" prop="target_type">
          <el-radio-group v-model="form.target_type">
            <el-radio :label="1">全员</el-radio>
            <el-radio :label="2">按角色</el-radio>
            <el-radio :label="3">指定用户</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="form.target_type === 2" label="收件角色" prop="role_ids">
          <el-select v-model="form.role_ids" multiple filterable placeholder="请选择角色" style="width:100%">
            <el-option v-for="r in roles" :key="r.id" :label="r.name" :value="r.id"></el-option>
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.target_type === 3" label="收件用户" prop="user_ids">
          <el-select v-model="form.user_ids" multiple filterable placeholder="请选择用户" style="width:100%">
            <el-option v-for="u in users" :key="u.id" :label="u.username" :value="u.id"></el-option>
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="sending" @click="handleSubmit">发 送</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script>
import { sendNotification } from '@/api/notification'
import { getRoles } from '@/api/role'
import { getUsers } from '@/api/user'

export default {
  name: 'NotificationSend',
  data() {
    const checkTargets = (rule, value, callback) => {
      if (this.form.target_type === 2 && this.form.role_ids.length === 0) {
        callback(new Error('请选择收件角色'))
      } else if (this.form.target_type === 3 && this.form.user_ids.length === 0) {
        callback(new Error('请选择收件用户'))
      } else {
        callback()
      }
    }
    return {
      form: {
        type: 1,
        title: '',
        content: '',
        target_type: 1,
        role_ids: [],
        user_ids: []
      },
      rules: {
        title: [{ required: true, message: '请输入标题', trigger: 'blur' }],
        target_type: [{ validator: checkTargets, trigger: 'change' }]
      },
      roles: [],
      users: [],
      sending: false
    }
  },
  created() {
    this.fetchRoles()
    this.fetchUsers()
  },
  methods: {
    async fetchRoles() {
      try {
        const res = await getRoles({ page: 1, page_size: 100 })
        if (res.code === 200) this.roles = res.data.list || []
      } catch (e) { /* 静默 */ }
    },
    async fetchUsers() {
      try {
        const res = await getUsers({ page: 1, page_size: 200 })
        if (res.code === 200) this.users = res.data.list || []
      } catch (e) { /* 静默 */ }
    },
    async handleSubmit() {
      this.$refs.sendForm.validate(async valid => {
        if (!valid) return
        this.sending = true
        try {
          const res = await sendNotification(this.form)
          if (res.code === 200) {
            this.$message.success('发送成功')
            this.$refs.sendForm.resetFields()
            this.form.role_ids = []
            this.form.user_ids = []
          }
        } finally {
          this.sending = false
        }
      })
    }
  }
}
</script>
```

注意：`getRoles`/`getUsers` 的导出名以 `web/src/api/role.js`、`web/src/api/user.js` 实际导出为准（先 grep 确认再 import；如果导出名是 `getRoleList`/`getUserList` 之类，按实际名写）。

管理端消息列表/删除（`notification:list`/`notification:delete`）不单独建页——发送页右侧不做列表，管理列表暂由消息发送菜单承载权限点即可（YAGNI：管理端消息列表页本设计不强制，API 已提供，前端暂不建页）。

- [ ] **Step 2: Docker 验证**

Run: `docker compose up -d --build nginx`
Expected: 构建成功

agent-browser 验证:

```bash
agent-browser --ignore-https-errors --args "--no-sandbox" open "https://localhost:8443"
# admin 登录，「系统管理」下出现「消息发送」菜单
# 选择按角色 → 选角色发送；选择指定用户 → 选用户发送
# 发送成功后，铃铛未读数增加（若收件人在线），消息中心页出现新消息
```

- [ ] **Step 3: Commit**

```bash
git add web/src/views/notification/send.vue
git commit -m "feat: 消息发送页面（公告/站内信、全员/角色/指定用户）"
```

---

## Task 10: 导出任务结果接入消息中心

**Files:**
- Modify: `internal/worker/export_worker.go:29-42,99-121`（注入 NotificationService、落库系统事件）

- [ ] **Step 1: 注入 Service**

`internal/worker/export_worker.go`：

ExportWorker 结构体加字段 `notificationService *service.NotificationService`，NewExportWorker 签名追加参数并赋值。

`cmd/server/main.go` 中 `exportWorker := worker.NewExportWorker(rdb, amqpConn, logService, hub)` 改为追加 `notificationService` 参数（注意装配顺序：notificationService 在 hub 之后创建，exportWorker 在其后创建，顺序已兼容；若 main.go 中 exportWorker 创建位置在 notificationService 之前，把 notificationService 的创建上移到 exportWorker 之前，保持 hub 先于两者）。

- [ ] **Step 2: 失败分支落库**

`processTask` 失败分支（`w.hub.Send(uid, ...export_failed...)` 之后）追加:

```go
	w.notificationService.SendSystemEvent(
		fmt.Sprintf("导出失败: %s", filename),
		fmt.Sprintf("任务 %s 导出失败: %s", taskID, err.Error()),
		uid,
	)
```

- [ ] **Step 3: 成功分支落库**

成功分支（`w.hub.Send(uid, ...export_complete...)` 之后）追加:

```go
	w.notificationService.SendSystemEvent(
		fmt.Sprintf("导出完成: %s", filename),
		fmt.Sprintf("导出任务完成，文件 %s 可在消息记录中查看下载链接: %s", filename, "/api/logs/download/"+taskID),
		uid,
	)
```

注意：`SendSystemEvent` 内部落库失败只 log.Printf 不 return error，不阻断导出流程（设计文档明确）。

- [ ] **Step 4: 编译 + Docker 验证**

Run: `go build ./...`，然后 `docker compose up -d --build go-app`

```bash
# 登录后触发一次日志导出
curl -s -X POST http://localhost:8000/api/logs/export -H "Authorization: Bearer <token>" -H "Content-Type: application/json" -d '{}'
# 等 worker 处理完（几秒），查我的消息
curl -s "http://localhost:8000/api/notifications/mine?type=3" -H "Authorization: Bearer <token>"
# 预期：list 含一条 type=3「导出完成」系统事件
```

- [ ] **Step 5: Commit**

```bash
git add internal/worker/export_worker.go cmd/server/main.go
git commit -m "feat: 导出任务结果接入消息中心（系统事件落库）"
```

---

## Task 11: 收尾自查与手工回归清单

**Files:**
- 无新文件（检查项）

- [ ] **Step 1: 全量验证**

Run: `go build ./... && go test ./...`
Expected: 全 PASS

- [ ] **Step 2: 清理检查**

grep 确认无残留调试输出:

```bash
grep -rn "fmt.Println\|console.log" internal/ web/src/views/notification/ web/src/layout/index.vue
# 预期：无匹配（log.Printf 仅限设计允许的降级告警处）
```

- [ ] **Step 3: git diff 自查范围**

Run: `git log --oneline -10 && git diff master --stat`（或按任务逐个 `git show --stat`）
Expected: 每个任务一个 commit，无超出范围的文件

- [ ] **Step 4: 生成手工回归测试清单（交给用户）**

自动化覆盖不了的点整理成清单交用户验证：

1. 两个浏览器分别登录不同用户，A 发布全员公告，B 的铃铛实时 +1（WS 推送）
2. B 离线（关闭浏览器）期间 A 发定向站内信，B 重新登录后铃铛显示正确未读数（离线补拉）
3. 消息中心页各 Tab/已读筛选组合切换数据正确
4. 全部已读后铃铛 badge 消失
5. 导出日志成功/失败两种路径，消息中心出现对应系统事件
6. 非 admin 普通角色登录：看不到「消息发送」菜单，直接调 `POST /api/notifications` 返回 403（RBAC）
7. 普通用户调 `GET /api/notifications/mine?…` 修改不了别人的已读状态（user_id 收敛）

---

## Self-Review 记录

- 规格覆盖：公告/站内信发布（T3/T4/T9）、三种收件范围（T3/T4）、已读/未读（T2/T4/T7/T8）、铃铛+独立页（T7/T8）、WS 推送（T3/T8）、导出事件接入（T10）、菜单种子（T5）、事务/降级（T2/T3/T10）——全覆盖
- 类型一致性：`SendTarget`/`expandTargets`/`BuildTarget`/`hub.Send`/`NotificationItem` 各任务间签名已核对
- 占位符：Task 4 Step 1 的 `now()` 是实现提示（写 `model.DateTime(time.Now())`），已显式说明，非 TBD
- 与设计文档的偏差已在计划内声明：WS 未读数用「补拉」代替「本地自增」（更稳）；管理端消息列表前端页暂不建（YAGNI，API 已备）
