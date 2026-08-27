package controller

import (
	"ginproject/internal/service"
	"ginproject/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type DbBackupController struct {
	backupService *service.DbBackupService
}

func NewDbBackupController(backupService *service.DbBackupService) *DbBackupController {
	return &DbBackupController{backupService: backupService}
}

func (c *DbBackupController) List(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))
	startTime := ctx.Query("start_time")
	endTime := ctx.Query("end_time")

	list, total, err := c.backupService.List(page, pageSize, startTime, endTime)
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

func (c *DbBackupController) Create(ctx *gin.Context) {
	backup, err := c.backupService.Backup("manual")
	if err != nil {
		utils.Error(ctx, 500, err.Error())
		return
	}

	utils.Success(ctx, backup)
}

func (c *DbBackupController) Restore(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, 400, "参数错误")
		return
	}

	if err := c.backupService.Restore(id); err != nil {
		utils.Error(ctx, 500, err.Error())
		return
	}

	utils.Success(ctx, nil)
}

func (c *DbBackupController) Delete(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, 400, "参数错误")
		return
	}

	if err := c.backupService.Delete(id); err != nil {
		utils.Error(ctx, 500, err.Error())
		return
	}

	utils.Success(ctx, nil)
}

func (c *DbBackupController) Download(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, 400, "参数错误")
		return
	}

	backup, err := c.backupService.GetByID(id)
	if err != nil {
		utils.Error(ctx, 404, "备份不存在")
		return
	}

	filepath := c.backupService.GetFilePath(backup.Filename)
	ctx.FileAttachment(filepath, backup.Filename)
}
