package main

import (
	"fmt"
	"ginproject/internal/config"
	"ginproject/internal/controller"
	"ginproject/internal/dao"
	"ginproject/internal/model"
	"ginproject/internal/router"
	"ginproject/internal/service"
	"ginproject/internal/worker"
	"ginproject/internal/ws"
	"log"

	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()

	// MySQL
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.User, cfg.Database.Password,
		cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("MySQL 连接失败: %v", err)
	}

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
	})

	// AutoMigrate
	db.AutoMigrate(&model.User{}, &model.Role{}, &model.Menu{}, &model.OperationLog{}, &model.LoginLog{})

	// DAO
	userDAO := dao.NewUserDAO(db)
	roleDAO := dao.NewRoleDAO(db)
	menuDAO := dao.NewMenuDAO(db)
	logDAO := dao.NewLogDAO(db)
	loginLogDAO := dao.NewLoginLogDAO(db)

	// Service
	authService := service.NewAuthService(userDAO, rdb, cfg)
	userService := service.NewUserService(userDAO)
	roleService := service.NewRoleService(roleDAO)
	menuService := service.NewMenuService(menuDAO)
	logService := service.NewLogService(logDAO)
	loginLogService := service.NewLoginLogService(loginLogDAO)

	// RabbitMQ
	amqpConn, err := amqp091.Dial(cfg.RabbitMQ.DSN())
	if err != nil {
		log.Fatalf("RabbitMQ 连接失败: %v", err)
	}
	defer amqpConn.Close()

	publishCh, err := amqpConn.Channel()
	if err != nil {
		log.Fatalf("RabbitMQ Channel 创建失败: %v", err)
	}
	defer publishCh.Close()

	// WebSocket Hub
	hub := ws.NewHub()
	wsCtrl := controller.NewWSController(hub, rdb, cfg)

	// Export Worker
	exportWorker := worker.NewExportWorker(rdb, amqpConn, logService, hub)
	go exportWorker.Start()

	// Controller
	authCtrl := controller.NewAuthController(authService, menuDAO, loginLogService)
	userCtrl := controller.NewUserController(userService)
	roleCtrl := controller.NewRoleController(roleService)
	menuCtrl := controller.NewMenuController(menuService)
	logCtrl := controller.NewLogController(logService, rdb, publishCh)
	loginLogCtrl := controller.NewLoginLogController(loginLogService)

	// 默认数据初始化
	seedDefaultData(db)

	// Router
	r := router.Setup(cfg, authCtrl, userCtrl, roleCtrl, menuCtrl, logCtrl, loginLogCtrl, wsCtrl, authService, userDAO, menuDAO, logDAO)

	log.Printf("Server running on :%s", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}

func seedDefaultData(db *gorm.DB) {
	var count int64
	db.Model(&model.User{}).Count(&count)
	if count > 0 {
		return
	}

	// 创建菜单（先父后子，利用自动 ID 递增）
	menus := []model.Menu{
		// 系统管理 (ID: 1)
		{Name: "系统管理", Icon: "el-icon-setting", Path: "/system", Type: 1, Sort: 1, Status: 1},
		{Name: "用户管理", Icon: "el-icon-user", Path: "/system/user", Type: 2, Sort: 1, Status: 1, ParentID: 1},
		{Name: "角色管理", Icon: "el-icon-s-custom", Path: "/system/role", Type: 2, Sort: 2, Status: 1, ParentID: 1},
		{Name: "菜单管理", Icon: "el-icon-menu", Path: "/system/menu", Type: 2, Sort: 3, Status: 1, ParentID: 1},
		// 日志管理 (ID: 5)
		{Name: "日志管理", Icon: "el-icon-document", Path: "/system/log-mgr", Type: 1, Sort: 2, Status: 1},
		{Name: "操作日志", Icon: "el-icon-document", Path: "/system/log", Type: 2, Sort: 1, Status: 1, ParentID: 5},
		{Name: "登录日志", Icon: "el-icon-document-checked", Path: "/system/login-log", Type: 2, Sort: 2, Status: 1, ParentID: 5},
		// 用户管理按钮 (ParentID: 2)
		{Name: "用户列表", Permission: "user:list", Type: 3, Sort: 1, Status: 1, ParentID: 2},
		{Name: "用户查询", Permission: "user:query", Type: 3, Sort: 2, Status: 1, ParentID: 2},
		{Name: "用户新增", Permission: "user:add", Type: 3, Sort: 3, Status: 1, ParentID: 2},
		{Name: "用户编辑", Permission: "user:edit", Type: 3, Sort: 4, Status: 1, ParentID: 2},
		{Name: "用户删除", Permission: "user:delete", Type: 3, Sort: 5, Status: 1, ParentID: 2},
		// 角色管理按钮 (ParentID: 3)
		{Name: "角色列表", Permission: "role:list", Type: 3, Sort: 1, Status: 1, ParentID: 3},
		{Name: "角色查询", Permission: "role:query", Type: 3, Sort: 2, Status: 1, ParentID: 3},
		{Name: "角色新增", Permission: "role:add", Type: 3, Sort: 3, Status: 1, ParentID: 3},
		{Name: "角色编辑", Permission: "role:edit", Type: 3, Sort: 4, Status: 1, ParentID: 3},
		{Name: "角色删除", Permission: "role:delete", Type: 3, Sort: 5, Status: 1, ParentID: 3},
		// 菜单管理按钮 (ParentID: 4)
		{Name: "菜单列表", Permission: "menu:list", Type: 3, Sort: 1, Status: 1, ParentID: 4},
		{Name: "菜单查询", Permission: "menu:query", Type: 3, Sort: 2, Status: 1, ParentID: 4},
		{Name: "菜单新增", Permission: "menu:add", Type: 3, Sort: 3, Status: 1, ParentID: 4},
		{Name: "菜单编辑", Permission: "menu:edit", Type: 3, Sort: 4, Status: 1, ParentID: 4},
		{Name: "菜单删除", Permission: "menu:delete", Type: 3, Sort: 5, Status: 1, ParentID: 4},
		// 操作日志按钮 (ParentID: 6)
		{Name: "日志列表", Permission: "log:list", Type: 3, Sort: 1, Status: 1, ParentID: 6},
		{Name: "日志导出", Permission: "log:export", Type: 3, Sort: 2, Status: 1, ParentID: 6},
		// 登录日志按钮 (ParentID: 7)
		{Name: "日志列表", Permission: "login-log:list", Type: 3, Sort: 1, Status: 1, ParentID: 7},
	}
	for i := range menus {
		db.Create(&menus[i])
	}

	// 创建 admin 用户
	hashed, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	adminUser := model.User{Username: "admin", Password: string(hashed), Status: 1}
	db.Create(&adminUser)

	// 创建超级管理员角色，关联所有菜单
	adminRole := model.Role{Name: "超级管理员", Code: "admin", Description: "系统超级管理员", Status: 1}
	adminRole.Menus = menus
	db.Create(&adminRole)

	db.Model(&adminUser).Association("Roles").Append([]model.Role{adminRole})

	log.Println("默认数据初始化完成: admin/admin")
}
