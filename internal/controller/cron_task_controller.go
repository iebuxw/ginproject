package controller

import (
	"ginproject/internal/model"
	"ginproject/internal/service"
	"ginproject/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CronTaskController struct {
	cronTaskService *service.CronTaskService
}

func NewCronTaskController(s *service.CronTaskService) *CronTaskController {
	return &CronTaskController{cronTaskService: s}
}

// List 获取定时任务分页列表
// @Summary 获取定时任务分页列表
// @Description 分页查询定时任务列表，支持关键词搜索（名称/URL）
// @Tags 定时任务
// @Security BearerAuth
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param keyword query string false "搜索关键词（名称/URL）"
// @Success 200 {object} utils.Response{data=object{list=[]model.CronTask,total=int}} "成功"
// @Router /cron-tasks [get]
func (ctl *CronTaskController) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	keyword := c.Query("keyword")
	list, total, err := ctl.cronTaskService.FindPage(page, pageSize, keyword)
	if err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, gin.H{"list": list, "total": total})
}

// Get 获取定时任务详情
// @Summary 获取定时任务详情
// @Description 根据 ID 查询定时任务详情
// @Tags 定时任务
// @Security BearerAuth
// @Produce json
// @Param id path int true "任务 ID"
// @Success 200 {object} utils.Response{data=model.CronTask} "成功"
// @Failure 200 {object} utils.Response "任务不存在"
// @Router /cron-tasks/{id} [get]
func (ctl *CronTaskController) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	t, err := ctl.cronTaskService.FindByID(uint(id))
	if err != nil {
		utils.Error(c, 404, "任务不存在")
		return
	}
	utils.Success(c, t)
}

// Create 新建定时任务
// @Summary 新建定时任务
// @Tags 定时任务
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body model.CronTask true "任务信息"
// @Success 200 {object} utils.Response "成功"
// @Router /cron-tasks [post]
func (ctl *CronTaskController) Create(c *gin.Context) {
	var m model.CronTask
	if err := c.ShouldBindJSON(&m); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	if err := ctl.cronTaskService.Create(&m); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}

// Update 编辑定时任务
// @Summary 编辑定时任务
// @Tags 定时任务
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "任务 ID"
// @Param body body model.CronTask true "任务信息"
// @Success 200 {object} utils.Response "成功"
// @Router /cron-tasks/{id} [put]
func (ctl *CronTaskController) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var m model.CronTask
	if err := c.ShouldBindJSON(&m); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	m.ID = uint(id)
	if err := ctl.cronTaskService.Update(&m); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}

// Delete 删除定时任务
// @Summary 删除定时任务
// @Tags 定时任务
// @Security BearerAuth
// @Produce json
// @Param id path int true "任务 ID"
// @Success 200 {object} utils.Response "成功"
// @Router /cron-tasks/{id} [delete]
func (ctl *CronTaskController) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := ctl.cronTaskService.Delete(uint(id)); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}

// UpdateStatus 启停定时任务
// @Summary 启停定时任务
// @Tags 定时任务
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "任务 ID"
// @Param body body object{status=int} true "状态（1=启用 0=停用）"
// @Success 200 {object} utils.Response "成功"
// @Router /cron-tasks/{id}/status [put]
func (ctl *CronTaskController) UpdateStatus(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var m struct {
		Status int `json:"status"`
	}
	if err := c.ShouldBindJSON(&m); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	if err := ctl.cronTaskService.UpdateStatus(uint(id), m.Status); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}

// Run 立即执行一次
// @Summary 立即执行一次
// @Description 手动触发任务立即执行，写入 manual 执行日志
// @Tags 定时任务
// @Security BearerAuth
// @Produce json
// @Param id path int true "任务 ID"
// @Success 200 {object} utils.Response "成功"
// @Router /cron-tasks/{id}/run [post]
func (ctl *CronTaskController) Run(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := ctl.cronTaskService.RunNow(uint(id)); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}

// Executions 获取任务执行日志
// @Summary 获取任务执行日志
// @Description 分页查询指定任务的执行日志
// @Tags 定时任务
// @Security BearerAuth
// @Produce json
// @Param id path int true "任务 ID"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} utils.Response{data=object{list=[]model.CronTaskExecution,total=int}} "成功"
// @Router /cron-tasks/{id}/executions [get]
func (ctl *CronTaskController) Executions(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	list, total, err := ctl.cronTaskService.FindExecutions(uint(id), page, pageSize)
	if err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, gin.H{"list": list, "total": total})
}
