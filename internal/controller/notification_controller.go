package controller

import (
	"ginproject/internal/dao"
	"ginproject/internal/model"
	"ginproject/internal/service"
	"ginproject/internal/utils"
	"strconv"
	"time"

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
		CreatedAt:  model.DateTime(time.Now()),
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
	var list []dao.NotificationItem
	var total int64
	var err error
	list, total, err = ctl.notificationService.FindUserPage(uid, page, pageSize, readStatus, notifType)
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
