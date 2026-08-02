package router

import (
	"ginproject/internal/config"
	"ginproject/internal/controller"
	"ginproject/internal/dao"
	"ginproject/internal/middleware"
	"ginproject/internal/service"

	"github.com/gin-gonic/gin"
)

func Setup(
	cfg *config.Config,
	authCtrl *controller.AuthController,
	userCtrl *controller.UserController,
	roleCtrl *controller.RoleController,
	menuCtrl *controller.MenuController,
	logCtrl *controller.LogController,
	loginLogCtrl *controller.LoginLogController,
	wsCtrl *controller.WSController,
	authService *service.AuthService,
	userDAO *dao.UserDAO,
	menuDAO *dao.MenuDAO,
	logDAO *dao.LogDAO,
) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORS())

	api := r.Group("/api")

	// 公开路由
	api.POST("/auth/login", authCtrl.Login)

	// 需登录（JWT 中间件在 group 级别）
	authorized := api.Group("")
	authorized.Use(middleware.JWTAuth(cfg, authService, userDAO))
	{
		authorized.POST("/auth/logout", authCtrl.Logout)
		authorized.POST("/auth/change-password", authCtrl.ChangePassword)
		authorized.GET("/auth/userinfo", authCtrl.UserInfo)

		// 用户管理
		authorized.GET("/users",
			middleware.RequirePerm("user:list"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO), userCtrl.List)
		authorized.GET("/users/:id",
			middleware.RequirePerm("user:query"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO), userCtrl.Get)
		authorized.POST("/users",
			middleware.RequirePerm("user:add"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO), userCtrl.Create)
		authorized.PUT("/users/:id",
			middleware.RequirePerm("user:edit"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO), userCtrl.Update)
		authorized.DELETE("/users/:id",
			middleware.RequirePerm("user:delete"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO), userCtrl.Delete)

		// 角色管理
		authorized.GET("/roles",
			middleware.RequirePerm("role:list"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO), roleCtrl.List)
		authorized.GET("/roles/:id",
			middleware.RequirePerm("role:query"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO), roleCtrl.Get)
		authorized.POST("/roles",
			middleware.RequirePerm("role:add"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO), roleCtrl.Create)
		authorized.PUT("/roles/:id",
			middleware.RequirePerm("role:edit"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO), roleCtrl.Update)
		authorized.DELETE("/roles/:id",
			middleware.RequirePerm("role:delete"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO), roleCtrl.Delete)

		// 菜单管理
		authorized.GET("/menus",
			middleware.RequirePerm("menu:list"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO), menuCtrl.List)
		authorized.GET("/menus/:id",
			middleware.RequirePerm("menu:query"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO), menuCtrl.Get)
		authorized.POST("/menus",
			middleware.RequirePerm("menu:add"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO), menuCtrl.Create)
		authorized.PUT("/menus/:id",
			middleware.RequirePerm("menu:edit"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO), menuCtrl.Update)
		authorized.DELETE("/menus/:id",
			middleware.RequirePerm("menu:delete"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO), menuCtrl.Delete)

		// 登录日志
		authorized.GET("/login-logs",
			middleware.RequirePerm("login-log:list"), middleware.RBAC(menuDAO), loginLogCtrl.List)

		// 操作日志
		authorized.GET("/logs",
			middleware.RequirePerm("log:list"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO), logCtrl.List)
		authorized.POST("/logs/export",
			middleware.RequirePerm("log:export"), middleware.RBAC(menuDAO), logCtrl.Export)
		authorized.GET("/logs/export-status",
			middleware.RequirePerm("log:export"), middleware.RBAC(menuDAO), logCtrl.ExportStatus)
		authorized.GET("/logs/download/:taskID",
			middleware.RequirePerm("log:export"), middleware.RBAC(menuDAO), logCtrl.Download)
	}

	// WebSocket
	r.GET("/api/ws", wsCtrl.Handle)

	return r
}
