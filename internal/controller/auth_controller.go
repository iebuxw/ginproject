package controller

import (
	"context"
	"encoding/json"
	"errors"
	"ginproject/internal/dao"
	"ginproject/internal/model"
	"ginproject/internal/service"
	"ginproject/internal/utils"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

type AuthController struct {
	authService     *service.AuthService
	menuDAO         *dao.MenuDAO
	loginLogService *service.LoginLogService
	rdb             *redis.Client
	publishCh       *amqp091.Channel
}

func NewAuthController(authService *service.AuthService, menuDAO *dao.MenuDAO, loginLogService *service.LoginLogService, rdb *redis.Client, publishCh *amqp091.Channel) *AuthController {
	return &AuthController{authService, menuDAO, loginLogService, rdb, publishCh}
}

// alertMailLimitTTL 同一 IP 的登录告警邮件限频窗口
const alertMailLimitTTL = 5 * time.Minute

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
			Username: req.Username, Status: 0, Message: err.Error(), IP: c.ClientIP(), CreatedAt: model.DateTime(time.Now()),
		})
		if errors.Is(err, service.ErrInvalidCredentials) {
			ctl.publishLoginAlert(req.Username, c.ClientIP(), err.Error())
		}
		utils.Error(c, 401, err.Error())
		return
	}
	_ = ctl.loginLogService.Create(&model.LoginLog{
		Username: req.Username, Status: 1, IP: c.ClientIP(), CreatedAt: model.DateTime(time.Now()),
	})
	utils.Success(c, gin.H{"token": token, "user": user})
}

// publishLoginAlert 发布登录告警邮件任务到队列；同一 IP 限频窗口内只发一封，发布失败不影响登录
func (ctl *AuthController) publishLoginAlert(username, ip, message string) {
	key := "alert_mail:" + ip
	if ok, err := ctl.rdb.SetNX(context.Background(), key, "1", alertMailLimitTTL).Result(); err != nil || !ok {
		return
	}

	body, err := json.Marshal(map[string]string{
		"username": username,
		"ip":       ip,
		"message":  message,
	})
	if err != nil {
		return
	}
	if err := ctl.publishCh.Publish("", service.LoginAlertQueue, false, false, amqp091.Publishing{
		ContentType: "application/json",
		Body:        body,
	}); err != nil {
		log.Printf("登录告警任务发布失败: %v", err)
	}
}

func (ctl *AuthController) Logout(c *gin.Context) {
	token, _ := c.Get("token")
	if ts, ok := token.(string); ok {
		_ = ctl.authService.Logout(ts)
	}
	utils.Success(c, nil)
}

func (ctl *AuthController) ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)
	if err := ctl.authService.ChangePassword(uid, req.OldPassword, req.NewPassword); err != nil {
		utils.Error(c, 400, err.Error())
		return
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
