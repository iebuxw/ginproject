package controller

import (
	"ginproject/internal/service"
	"ginproject/internal/utils"

	"github.com/gin-gonic/gin"
)

type SystemSettingController struct {
	settingService *service.SystemSettingService
}

func NewSystemSettingController(settingService *service.SystemSettingService) *SystemSettingController {
	return &SystemSettingController{settingService: settingService}
}

// Get 获取系统配置
func (ctl *SystemSettingController) Get(c *gin.Context) {
	settings, err := ctl.settingService.GetAll()
	if err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, settings)
}

// Update 更新系统配置
func (ctl *SystemSettingController) Update(c *gin.Context) {
	var settings map[string]string
	if err := c.ShouldBindJSON(&settings); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	if err := ctl.settingService.Save(settings); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}
