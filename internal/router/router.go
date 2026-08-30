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
	uploadCtrl *controller.UploadController,
	authService *service.AuthService,
	userDAO *dao.UserDAO,
	menuDAO *dao.MenuDAO,
	logDAO *dao.LogDAO,
	logRepo *es.LogRepo,
	dictTypeCtrl *controller.DictTypeController,
	dictDataCtrl *controller.DictDataController,
	taskCtrl *controller.CronTaskController,
	dbBackupCtrl *controller.DbBackupController,
	fileCtrl *controller.FileController,
	dashboardCtrl *controller.DashboardController,
	settingCtrl *controller.SystemSettingController,
	notificationCtrl *controller.NotificationController,
) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORS())

	// 静态文件服务（头像等上传文件）
	r.Static("/api/uploads", "./uploads")

	api := r.Group("/api")

	// 公开路由
	api.POST("/auth/login", authCtrl.Login)
	api.GET("/settings", settingCtrl.Get)

	// 日志清理（公开：定时任务调度器无 JWT，靠 secret 参数防滥用）
	api.POST("/logs/cleanup", logCtrl.Cleanup)

	// 需登录（JWT 中间件在 group 级别）
	authorized := api.Group("")
	authorized.Use(middleware.JWTAuth(cfg, authService, userDAO))
	{
		authorized.POST("/auth/logout", authCtrl.Logout)
		authorized.POST("/auth/change-password", middleware.OperationLogger(logDAO, logRepo), authCtrl.ChangePassword)
		authorized.GET("/auth/userinfo", authCtrl.UserInfo)
		authorized.PUT("/auth/profile", middleware.OperationLogger(logDAO, logRepo), authCtrl.Profile)

		// 文件上传
		authorized.POST("/upload/avatar", uploadCtrl.UploadAvatar)

		// 管理员管理
		authorized.GET("/users",
			middleware.RequirePerm("user:list"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), userCtrl.List)
		authorized.GET("/users/export",
			middleware.RequirePerm("user:list"), middleware.RBAC(menuDAO), userCtrl.Export)
		// 用户导入（不挂 OperationLogger：中间件会把 multipart body 整体读入内存并把二进制写入日志）
		authorized.POST("/users/import",
			middleware.RequirePerm("user:add"), middleware.RBAC(menuDAO), userCtrl.Import)
		authorized.GET("/users/import-template",
			middleware.RequirePerm("user:add"), middleware.RBAC(menuDAO), userCtrl.ImportTemplate)
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
		authorized.GET("/roles/export",
			middleware.RequirePerm("role:list"), middleware.RBAC(menuDAO), roleCtrl.Export)
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

		// 数据库备份
		authorized.GET("/db-backups",
			middleware.RequirePerm("db_backup:list"), middleware.RBAC(menuDAO), dbBackupCtrl.List)
		authorized.POST("/db-backups",
			middleware.RequirePerm("db_backup:add"), middleware.RBAC(menuDAO), dbBackupCtrl.Create)
		authorized.POST("/db-backups/:id/restore",
			middleware.RequirePerm("db_backup:restore"), middleware.RBAC(menuDAO), dbBackupCtrl.Restore)
		authorized.DELETE("/db-backups/:id",
			middleware.RequirePerm("db_backup:delete"), middleware.RBAC(menuDAO), dbBackupCtrl.Delete)
		authorized.GET("/db-backups/:id/download",
			middleware.RequirePerm("db_backup:download"), middleware.RBAC(menuDAO), dbBackupCtrl.Download)

		// 文件管理（上传不挂 OperationLogger：中间件会把 multipart body 整体读入内存并把二进制写入日志）
		authorized.GET("/files",
			middleware.RequirePerm("file:list"), middleware.RBAC(menuDAO), fileCtrl.List)
		authorized.POST("/files/upload",
			middleware.RequirePerm("file:upload"), middleware.RBAC(menuDAO), fileCtrl.Upload)
		authorized.GET("/files/:id/download",
			middleware.RequirePerm("file:download"), middleware.RBAC(menuDAO), fileCtrl.Download)
		authorized.DELETE("/files/:id",
			middleware.RequirePerm("file:delete"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), fileCtrl.Delete)

		// 系统配置（读取已移到公开路由；写入需认证+权限）
		authorized.PUT("/settings",
			middleware.RequirePerm("setting:save"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), settingCtrl.Update)

		// 消息中心
		authorized.POST("/notifications",
			middleware.RequirePerm("notification:send"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), notificationCtrl.Send)
		authorized.GET("/notifications",
			middleware.RequirePerm("notification:list"), middleware.RBAC(menuDAO), notificationCtrl.List)
		authorized.DELETE("/notifications/:id",
			middleware.RequirePerm("notification:delete"), middleware.RBAC(menuDAO), middleware.OperationLogger(logDAO, logRepo), notificationCtrl.Delete)
		authorized.GET("/notifications/mine",
			notificationCtrl.Mine)
		authorized.POST("/notifications/read",
			notificationCtrl.Read)
		authorized.GET("/notifications/unread-count",
			notificationCtrl.UnreadCount)

		// Logo 上传
		authorized.POST("/upload/logo", uploadCtrl.UploadLogo)

		// 仪表盘
		authorized.GET("/dashboard/server-info", dashboardCtrl.GetServerInfo)
	}

	// WebSocket
	r.GET("/api/ws", wsCtrl.Handle)

	// Swagger UI（无需认证）
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}
