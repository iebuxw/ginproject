package middleware

import (
	"ginproject/internal/dao"
	"ginproject/internal/model"
	"ginproject/internal/utils"

	"github.com/gin-gonic/gin"
)

func RBAC(menuDAO *dao.MenuDAO) gin.HandlerFunc {
	return func(c *gin.Context) {
		permission, _ := c.Get("required_permission")
		perm, ok := permission.(string)
		if !ok || perm == "" {
			c.Next()
			return
		}

		rolesVal, exists := c.Get("roles")
		if !exists {
			utils.ErrorWithStatus(c, 403, 403, "无权访问")
			c.Abort()
			return
		}
		roles := rolesVal.([]model.Role)
		var roleIDs []uint
		for _, r := range roles {
			roleIDs = append(roleIDs, r.ID)
		}

		menus, err := menuDAO.FindByRoleIDs(roleIDs)
		if err != nil {
			utils.ErrorWithStatus(c, 500, 500, "内部错误")
			c.Abort()
			return
		}
		for _, m := range menus {
			if m.Permission == perm {
				c.Next()
				return
			}
		}
		utils.ErrorWithStatus(c, 403, 403, "无权访问")
		c.Abort()
	}
}

func RequirePerm(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("required_permission", perm)
		c.Next()
	}
}
