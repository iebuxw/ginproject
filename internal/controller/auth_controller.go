package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	userDAO         *dao.UserDAO
	loginLogService *service.LoginLogService
	rdb             *redis.Client
	publishCh       *amqp091.Channel
	settingService  *service.SystemSettingService
}

func NewAuthController(authService *service.AuthService, menuDAO *dao.MenuDAO, userDAO *dao.UserDAO, loginLogService *service.LoginLogService, rdb *redis.Client, publishCh *amqp091.Channel, settingService *service.SystemSettingService) *AuthController {
	return &AuthController{authService, menuDAO, userDAO, loginLogService, rdb, publishCh, settingService}
}

// alertMailLimitTTL 同一 IP 的登录告警邮件限频窗口
const alertMailLimitTTL = 5 * time.Minute

// LoginRequest 登录请求参数
type LoginRequest struct {
	Username    string `json:"username" binding:"required" example:"admin"`
	Password    string `json:"password" binding:"required" example:"123456"`
	CaptchaID   string `json:"captcha_id"`
	CaptchaCode string `json:"captcha_code"`
}

// Login 用户登录
// @Summary 用户登录
// @Description 使用用户名和密码登录，返回 JWT Token。连续失败达到阈值后账号临时锁定，到期自动解锁
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body LoginRequest true "登录参数"
// @Success 200 {object} utils.Response{data=object{token=string,user=model.User}} "成功"
// @Failure 200 {object} utils.Response "业务错误"
// @Router /auth/login [post]
func (ctl *AuthController) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}

	// 验证码校验（根据系统设置决定是否启用）
	if err := ctl.checkCaptcha(req.CaptchaID, req.CaptchaCode); err != nil {
		utils.Error(c, 400, err.Error())
		return
	}

	token, user, err := ctl.authService.Login(req.Username, req.Password)
	if err != nil {
		_ = ctl.loginLogService.Create(&model.LoginLog{
			Username: req.Username, Status: 0, Message: err.Error(), IP: c.ClientIP(), CreatedAt: model.DateTime(time.Now()),
		})
		// 仅凭据错误发告警邮件；锁定期间（ErrAccountLocked）不重复发
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

// checkCaptcha 校验验证码；未启用时跳过；Redis 不可用时降级放行
func (ctl *AuthController) checkCaptcha(captchaID, captchaCode string) error {
	settings, err := ctl.settingService.GetAll()
	if err != nil {
		log.Printf("读取系统设置失败: %v", err)
		return nil // 降级放行
	}
	if settings["captcha_enabled"] != "1" {
		return nil
	}

	if captchaID == "" || captchaCode == "" {
		return fmt.Errorf("请输入验证码")
	}

	key := "captcha:" + captchaID
	ctx := context.Background()
	stored, err := ctl.rdb.Get(ctx, key).Result()
	if err != nil {
		log.Printf("验证码读取失败（可能已过期）: %v", err)
		return fmt.Errorf("验证码已过期，请重新获取")
	}
	// 一次性消耗
	ctl.rdb.Del(ctx, key)

	if stored != captchaCode {
		return fmt.Errorf("验证码错误")
	}
	return nil
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

// Logout 用户登出
// @Summary 用户登出
// @Description 将当前 Token 加入 Redis 黑名单
// @Tags 认证
// @Security BearerAuth
// @Produce json
// @Success 200 {object} utils.Response "成功"
// @Router /auth/logout [post]
func (ctl *AuthController) Logout(c *gin.Context) {
	token, _ := c.Get("token")
	if ts, ok := token.(string); ok {
		_ = ctl.authService.Logout(ts)
	}
	utils.Success(c, nil)
}

// ChangePassword 修改密码
// @Summary 修改密码
// @Description 修改当前用户的登录密码
// @Tags 认证
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body object true "密码参数" schema(type=object,required(old_password,new_password),properties(old_password(type=string,description=旧密码),new_password(type=string,description=新密码)))
// @Success 200 {object} utils.Response "成功"
// @Failure 200 {object} utils.Response "业务错误"
// @Router /auth/change-password [post]
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

// UserInfo 获取当前用户信息
// @Summary 获取当前用户信息
// @Description 返回当前用户的 ID、用户名、菜单树和权限列表
// @Tags 认证
// @Security BearerAuth
// @Produce json
// @Success 200 {object} utils.Response{data=object{id=uint,username=string,menus=array,permissions=[]string}} "成功"
// @Router /auth/userinfo [get]
func (ctl *AuthController) UserInfo(c *gin.Context) {
	userID, _ := c.Get("user_id")
	rolesVal, _ := c.Get("roles")

	uid, _ := userID.(uint)
	var roleIDs []uint
	var avatar string
	var email string
	if roles, ok := rolesVal.([]model.Role); ok {
		for _, r := range roles {
			roleIDs = append(roleIDs, r.ID)
		}
	}
	// 查库获取头像等完整信息
	if user, err := ctl.userDAO.FindByID(uid); err == nil {
		avatar = user.Avatar
		email = user.Email
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
		"username":    c.GetString("username"),
		"avatar":      avatar,
		"email":       email,
		"menus":       menuTree,
		"permissions": permissions,
	})
}

// Profile 更新当前用户个人信息
// @Summary 更新个人信息
// @Description 更新当前用户的邮箱等个人信息
// @Tags 认证
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body object true "个人信息"
// @Param body.body.email body string false "邮箱"
// @Success 200 {object} utils.Response "成功"
// @Failure 200 {object} utils.Response "业务错误"
// @Router /auth/profile [put]
func (ctl *AuthController) Profile(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)
	if err := ctl.authService.UpdateProfile(uid, req.Email); err != nil {
		utils.Error(c, 400, err.Error())
		return
	}
	utils.Success(c, nil)
}
