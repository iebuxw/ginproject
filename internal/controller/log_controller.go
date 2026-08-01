package controller

import (
	"ginproject/internal/service"
	"ginproject/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type LogController struct{ logService *service.LogService }

func NewLogController(logService *service.LogService) *LogController {
	return &LogController{logService}
}

func (ctl *LogController) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	module := c.Query("module")
	method := c.Query("method")
	logs, total, err := ctl.logService.FindPage(page, pageSize, module, method)
	if err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, gin.H{"list": logs, "total": total})
}
