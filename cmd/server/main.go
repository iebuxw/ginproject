// @title GinAdmin API
// @version 1.0
// @description 后台管理系统 API 文档
// @host localhost:8000
// @BasePath /api
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT Token，格式: Bearer {token}
package main

import (
	"fmt"
	"ginproject/internal/config"
	"ginproject/internal/controller"
	"ginproject/internal/dao"
	"ginproject/internal/es"
	"ginproject/internal/router"
	"ginproject/internal/scheduler"
	"ginproject/internal/service"
	"ginproject/internal/worker"
	"ginproject/internal/ws"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
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

	// 数据库迁移
	if err := runMigrations(cfg); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	// Elasticsearch（学习用；失败不 fatal：写入跳过、查询回退 MySQL）
	esClient, esErr := es.NewClient(&cfg.Elasticsearch)
	if esErr != nil || esClient.Ping() != nil {
		log.Printf("警告: Elasticsearch 不可用，日志全文搜索将回退 MySQL: %v", esErr)
		esClient = nil
	}
	logRepo := es.NewLogRepo(esClient)
	if logRepo.Enabled() {
		if err := es.EnsureIndex(esClient.RawClient()); err != nil {
			log.Printf("警告: ES 索引初始化失败: %v", err)
		}
	}

	// DAO
	userDAO := dao.NewUserDAO(db)
	roleDAO := dao.NewRoleDAO(db)
	menuDAO := dao.NewMenuDAO(db)
	logDAO := dao.NewLogDAO(db)
	loginLogDAO := dao.NewLoginLogDAO(db)
	dictTypeDAO := dao.NewDictTypeDAO(db)
	dictDataDAO := dao.NewDictDataDAO(db)
	cronTaskDAO := dao.NewCronTaskDAO(db)
	cronTaskExecutionDAO := dao.NewCronTaskExecutionDAO(db)

	// Service
	authService := service.NewAuthService(userDAO, rdb, cfg)
	userService := service.NewUserService(userDAO)
	roleService := service.NewRoleService(roleDAO)
	menuService := service.NewMenuService(menuDAO)
	logService := service.NewLogService(logDAO, logRepo)
	loginLogService := service.NewLoginLogService(loginLogDAO)
	alertMailService := service.NewAlertMailService(cfg)
	dictTypeService := service.NewDictTypeService(dictTypeDAO, dictDataDAO)
	dictDataService := service.NewDictDataService(dictDataDAO)

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

	// Mail Worker
	mailWorker := worker.NewMailWorker(amqpConn, alertMailService)
	go mailWorker.Start()

	// 定时任务调度器
	taskScheduler := scheduler.NewScheduler(cronTaskDAO, cronTaskExecutionDAO)
	// 种子清理任务占位符密钥注入（__LOG_CLEANUP_SECRET__ → .env 实际值）
	if n, err := cronTaskDAO.InjectCleanupSecret(cfg.LogCleanupSecret); err != nil {
		log.Printf("清理任务密钥注入失败: %v", err)
	} else if n > 0 {
		log.Printf("已为 %d 个清理任务注入密钥", n)
	}
	go taskScheduler.Start()

	// Controller
	authCtrl := controller.NewAuthController(authService, menuDAO, loginLogService, rdb, publishCh)
	userCtrl := controller.NewUserController(userService)
	roleCtrl := controller.NewRoleController(roleService)
	menuCtrl := controller.NewMenuController(menuService)
	logCtrl := controller.NewLogController(logService, loginLogService, rdb, publishCh, cfg.LogCleanupSecret)
	loginLogCtrl := controller.NewLoginLogController(loginLogService)
	dictTypeCtrl := controller.NewDictTypeController(dictTypeService)
	dictDataCtrl := controller.NewDictDataController(dictDataService)
	cronTaskService := service.NewCronTaskService(cronTaskDAO, cronTaskExecutionDAO, taskScheduler)
	cronTaskCtrl := controller.NewCronTaskController(cronTaskService)

	// Router
	r := router.Setup(cfg, authCtrl, userCtrl, roleCtrl, menuCtrl, logCtrl, loginLogCtrl, wsCtrl, authService, userDAO, menuDAO, logDAO, logRepo, dictTypeCtrl, dictDataCtrl, cronTaskCtrl)

	log.Printf("Server running on :%s", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}

func runMigrations(cfg *config.Config) error {
	dsn := fmt.Sprintf("mysql://%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.User, cfg.Database.Password,
		cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)

	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		return fmt.Errorf("创建迁移实例失败: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("执行迁移失败: %w", err)
	}

	log.Println("数据库迁移完成")
	return nil
}
