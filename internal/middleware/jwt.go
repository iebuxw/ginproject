package middleware

import (
	"ginproject/internal/config"
	"ginproject/internal/dao"
	"ginproject/internal/service"
	"ginproject/internal/utils"

	"github.com/gin-gonic/gin"
)

func JWTAuth(cfg *config.Config, authService *service.AuthService, userDAO *dao.UserDAO) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if len(token) < 7 || token[:7] != "Bearer " {
			utils.ErrorWithStatus(c, 401, 401, "未授权")
			c.Abort()
			return
		}
		token = token[7:]
		if authService.IsBlacklisted(token) {
			utils.ErrorWithStatus(c, 401, 401, "Token已失效")
			c.Abort()
			return
		}
		claims, err := utils.ParseToken(token, cfg.JWT.Secret)
		if err != nil {
			utils.ErrorWithStatus(c, 401, 401, "Token无效")
			c.Abort()
			return
		}
		user, err := userDAO.FindByID(claims.UserID)
		if err != nil || user.Status != 1 {
			utils.ErrorWithStatus(c, 401, 401, "用户不存在或已禁用")
			c.Abort()
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("roles", user.Roles)
		c.Set("token", token)
		c.Next()
	}
}
