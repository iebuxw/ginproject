package controller

import (
	"ginproject/internal/dao"
	"ginproject/internal/model"
	"ginproject/internal/service"
	"ginproject/internal/utils"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	authService      *service.AuthService
	menuDAO          *dao.MenuDAO
	loginLogService  *service.LoginLogService
}

func NewAuthController(authService *service.AuthService, menuDAO *dao.MenuDAO, loginLogService *service.LoginLogService) *AuthController {
	return &AuthController{authService, menuDAO, loginLogService}
}

func (ctl *AuthController) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	token, user, err := ctl.authService.Login(req.Username, req.Password)
	if err != nil {
		_ = ctl.loginLogService.Create(&model.LoginLog{
			Username: req.Username, Status: 0, Message: err.Error(), IP: c.ClientIP(),
		})
		utils.Error(c, 401, err.Error())
		return
	}
	_ = ctl.loginLogService.Create(&model.LoginLog{
		Username: req.Username, Status: 1, IP: c.ClientIP(),
	})
	utils.Success(c, gin.H{"token": token, "user": user})
}

func (ctl *AuthController) Logout(c *gin.Context) {
	token, _ := c.Get("token")
	if ts, ok := token.(string); ok {
		_ = ctl.authService.Logout(ts)
	}
	utils.Success(c, nil)
}

func (ctl *AuthController) UserInfo(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	rolesVal, _ := c.Get("roles")

	var roleIDs []uint
	if roles, ok := rolesVal.([]model.Role); ok {
		for _, r := range roles {
			roleIDs = append(roleIDs, r.ID)
		}
	}
	menus, _ := ctl.menuDAO.FindByRoleIDs(roleIDs)
	menuTree := dao.BuildMenuTree(menus, 0)
	var permissions []string
	for _, m := range menus {
		if m.Permission != "" {
			permissions = append(permissions, m.Permission)
		}
	}
	utils.Success(c, gin.H{
		"id":          userID,
		"username":    username,
		"menus":       menuTree,
		"permissions": permissions,
	})
}
