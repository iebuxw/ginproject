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

// List 查询登录日志
// @Summary 查询登录日志
// @Description 分页查询登录日志记录
// @Tags 登录日志
// @Security BearerAuth
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param username query string false "用户名筛选"
// @Param status query int false "状态筛选 1=成功 0=失败 -1=不筛选" default(-1)
// @Success 200 {object} utils.Response{data=object{list=[]model.LoginLog,total=int}} "成功"
// @Router /login-logs [get]
func (ctl *LoginLogController) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	username := c.Query("username")
	status := -1
	if s := c.Query("status"); s != "" {
		status, _ = strconv.Atoi(s)
	}
	logs, total, err := ctl.loginLogService.FindPage(page, pageSize, username, status)
	if err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, gin.H{"list": logs, "total": total})
}
