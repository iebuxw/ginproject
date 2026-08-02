package controller

import (
	"ginproject/internal/service"
	"ginproject/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type LoginLogController struct {
	loginLogService *service.LoginLogService
}

func NewLoginLogController(loginLogService *service.LoginLogService) *LoginLogController {
	return &LoginLogController{loginLogService: loginLogService}
}

func (ctl *LoginLogController) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	username := c.Query("username")
	status, _ := strconv.Atoi(c.DefaultQuery("status", "0"))
	logs, total, err := ctl.loginLogService.FindPage(page, pageSize, username, status)
	if err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, gin.H{"list": logs, "total": total})
}
