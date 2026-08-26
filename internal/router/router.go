package router

import (
	"ginproject/internal/config"
	"ginproject/internal/controller"
	"ginproject/internal/dao"
	"ginproject/internal/es"
	"ginproject/internal/middleware"
	"ginproject/internal/service"

	_ "ginproject/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
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
	logRepo *es.LogRepo,
	dictTypeCtrl *controller.DictTypeController,
	dictDataCtrl *controller.DictDataController,
	taskCtrl *controller.CronTaskController,
) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORS())

	api := r.Group("/api")

	// 公开路由
	api.POST("/auth/login", authCtrl.Login)

	// 日志清理（公开：定时任务调度器无 JWT，靠 secret 参数防滥用）
	api.POST("/logs/cleanup", logCtrl.Cleanup)

	// 需登录（JWT 中间件在 group 级别）
	authorized := api.Group("")
	authorized.Use(middleware.JWTAuth(cfg, authService, userDAO))
	{
		authorized.POST("/auth/logout", authCtrl.Logout)
		authorized.POST("/auth/change-password", middleware.OperationLogger(logDAO, logRepo), authCtrl.ChangePassword)
		authorized.GET("/auth/userinfo", authCtrl.UserInfo)

		// 用户管理
		authorized.GET("/users",
			middleware.RequirePerm("user:list"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), userCtrl.List)
		authorized.GET("/users/:id",
			middleware.RequirePerm("user:query"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), userCtrl.Get)
		authorized.POST("/users",
			middleware.RequirePerm("user:add"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), userCtrl.Create)
		authorized.PUT("/users/:id",
			middleware.RequirePerm("user:edit"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), userCtrl.Update)
		authorized.DELETE("/users/:id",
			middleware.RequirePerm("user:delete"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), userCtrl.Delete)

		// 角色管理
		authorized.GET("/roles",
			middleware.RequirePerm("role:list"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), roleCtrl.List)
		authorized.GET("/roles/:id",
			middleware.RequirePerm("role:query"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), roleCtrl.Get)
		authorized.POST("/roles",
			middleware.RequirePerm("role:add"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), roleCtrl.Create)
		authorized.PUT("/roles/:id",
			middleware.RequirePerm("role:edit"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), roleCtrl.Update)
		authorized.DELETE("/roles/:id",
			middleware.RequirePerm("role:delete"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), roleCtrl.Delete)

		// 菜单管理
		authorized.GET("/menus",
			middleware.RequirePerm("menu:list"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), menuCtrl.List)
		authorized.GET("/menus/:id",
			middleware.RequirePerm("menu:query"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), menuCtrl.Get)
		authorized.POST("/menus",
			middleware.RequirePerm("menu:add"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), menuCtrl.Create)
		authorized.PUT("/menus/:id",
			middleware.RequirePerm("menu:edit"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), menuCtrl.Update)
		authorized.DELETE("/menus/:id",
			middleware.RequirePerm("menu:delete"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), menuCtrl.Delete)

		// 登录日志
		authorized.GET("/login-logs",
			middleware.RequirePerm("login-log:list"), middleware.RBAC(menuDAO), loginLogCtrl.List)

		// 操作日志
		authorized.GET("/logs",
			middleware.RequirePerm("log:list"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), logCtrl.List)
		authorized.POST("/logs/export",
			middleware.RequirePerm("log:export"), middleware.RBAC(menuDAO), logCtrl.Export)
		authorized.GET("/logs/export-status",
			middleware.RequirePerm("log:export"), middleware.RBAC(menuDAO), logCtrl.ExportStatus)
		authorized.GET("/logs/download/:taskID",
			middleware.RequirePerm("log:export"), middleware.RBAC(menuDAO), logCtrl.Download)

		// 数据字典 - 类型管理
		authorized.GET("/dict-types",
			middleware.RequirePerm("dict:list"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), dictTypeCtrl.List)
		authorized.GET("/dict-types/:id",
			middleware.RequirePerm("dict:query"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), dictTypeCtrl.Get)
		authorized.POST("/dict-types",
			middleware.RequirePerm("dict:add"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), dictTypeCtrl.Create)
		authorized.PUT("/dict-types/:id",
			middleware.RequirePerm("dict:edit"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), dictTypeCtrl.Update)
		authorized.DELETE("/dict-types/:id",
			middleware.RequirePerm("dict:delete"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), dictTypeCtrl.Delete)

		// 数据字典 - 数据管理
		authorized.GET("/dict-data",
			middleware.RequirePerm("dict:list"), middleware.RBAC(menuDAO), dictDataCtrl.List)
		authorized.GET("/dict-data/by-code/:code",
			middleware.RequirePerm("dict:list"), middleware.RBAC(menuDAO), dictDataCtrl.GetByCode)
		authorized.GET("/dict-data/:id",
			middleware.RequirePerm("dict:query"), middleware.RBAC(menuDAO), dictDataCtrl.Get)
		authorized.POST("/dict-data",
			middleware.RequirePerm("dict:add"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), dictDataCtrl.Create)
		authorized.PUT("/dict-data/:id",
			middleware.RequirePerm("dict:edit"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), dictDataCtrl.Update)
		authorized.DELETE("/dict-data/:id",
			middleware.RequirePerm("dict:delete"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), dictDataCtrl.Delete)

		// 定时任务
		authorized.GET("/cron-tasks/commands",
			middleware.RequirePerm("cron:list"), middleware.RBAC(menuDAO), taskCtrl.Commands)
		authorized.GET("/cron-tasks/executions",
			middleware.RequirePerm("cron:log"), middleware.RBAC(menuDAO), taskCtrl.ListAllExecutions)
		authorized.GET("/cron-tasks",
			middleware.RequirePerm("cron:list"), middleware.RBAC(menuDAO), taskCtrl.List)
		authorized.GET("/cron-tasks/:id",
			middleware.RequirePerm("cron:query"), middleware.RBAC(menuDAO), taskCtrl.Get)
		authorized.POST("/cron-tasks",
			middleware.RequirePerm("cron:add"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), taskCtrl.Create)
		authorized.PUT("/cron-tasks/:id",
			middleware.RequirePerm("cron:edit"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), taskCtrl.Update)
		authorized.DELETE("/cron-tasks/:id",
			middleware.RequirePerm("cron:delete"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), taskCtrl.Delete)
		authorized.PUT("/cron-tasks/:id/status",
			middleware.RequirePerm("cron:edit"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), taskCtrl.UpdateStatus)
		authorized.POST("/cron-tasks/:id/run",
			middleware.RequirePerm("cron:run"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), taskCtrl.Run)
		authorized.GET("/cron-tasks/:id/executions",
			middleware.RequirePerm("cron:log"), middleware.RBAC(menuDAO), taskCtrl.Executions)
	}

	// WebSocket
	r.GET("/api/ws", wsCtrl.Handle)

	// Swagger UI（无需认证）
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}
