package controller

import (
	"encoding/json"
	"strconv"

	"ginproject/internal/service"
	"ginproject/internal/utils"

	"github.com/gin-gonic/gin"
)

type LogSettingController struct {
	settingService *service.SystemSettingService
}

func NewLogSettingController(settingService *service.SystemSettingService) *LogSettingController {
	return &LogSettingController{settingService: settingService}
}

// Get 获取日志清理配置
// @Summary 获取日志清理配置
// @Tags 日志设置
// @Produce json
// @Success 200 {object} utils.Response{data=object{days=int,scope=[]string}} "成功"
// @Router /log-settings [get]
func (ctl *LogSettingController) Get(c *gin.Context) {
	cfg, err := ctl.settingService.GetAll()
	if err != nil {
		utils.Error(c, 500, "配置读取失败: "+err.Error())
		return
	}

	// 保留天数，默认 180
	days := 180
	if v, ok := cfg["log_cleanup_days"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 3650 {
			days = n
		}
	}

	// 清理范围，默认全部
	scope := []string{"operation", "login"}
	if v, ok := cfg["log_cleanup_scope"]; ok && v != "" {
		var parsed []string
		if json.Unmarshal([]byte(v), &parsed) == nil && len(parsed) > 0 {
			scope = parsed
		}
	}

	utils.Success(c, gin.H{
		"days":  days,
		"scope": scope,
	})
}

// Update 保存日志清理配置
// @Summary 保存日志清理配置
// @Tags 日志设置
// @Accept json
// @Produce json
// @Param body body object{days=int,scope=[]string} true "配置"
// @Success 200 {object} utils.Response "成功"
// @Failure 200 {object} utils.Response "参数非法"
// @Router /log-settings [put]
func (ctl *LogSettingController) Update(c *gin.Context) {
	var req struct {
		Days  int      `json:"days"`
		Scope []string `json:"scope"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数格式错误")
		return
	}

	// 校验保留天数
	if req.Days < 1 || req.Days > 3650 {
		utils.Error(c, 400, "保留天数必须在 1~3650 之间")
		return
	}

	// 校验清理范围
	if len(req.Scope) == 0 {
		utils.Error(c, 400, "清理范围不能为空")
		return
	}
	validScopes := map[string]bool{"operation": true, "login": true}
	for _, s := range req.Scope {
		if !validScopes[s] {
			utils.Error(c, 400, "无效的清理范围: "+s)
			return
		}
	}

	// 保存配置
	scopeJSON, _ := json.Marshal(req.Scope)
	settings := map[string]string{
		"log_cleanup_days":  strconv.Itoa(req.Days),
		"log_cleanup_scope": string(scopeJSON),
	}
	if err := ctl.settingService.Save(settings); err != nil {
		utils.Error(c, 500, "保存失败: "+err.Error())
		return
	}

	utils.Success(c, nil)
}
