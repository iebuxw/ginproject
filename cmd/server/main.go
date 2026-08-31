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
	"encoding/json"
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
	"strconv"
	"strings"

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
	dbBackupDAO := dao.NewDbBackupDAO(db)
	fileDAO := dao.NewFileDAO(db)
	systemSettingDAO := dao.NewSystemSettingDAO(db)
	notificationDAO := dao.NewNotificationDAO(db)

	// Service
	authService := service.NewAuthService(userDAO, rdb, cfg)
	userService := service.NewUserService(userDAO, roleDAO)
	roleService := service.NewRoleService(roleDAO)
	menuService := service.NewMenuService(menuDAO)
	logService := service.NewLogService(logDAO, logRepo)
	loginLogService := service.NewLoginLogService(loginLogDAO)
	alertMailService := service.NewAlertMailService(cfg)
	dictTypeService := service.NewDictTypeService(dictTypeDAO, dictDataDAO)
	dictDataService := service.NewDictDataService(dictDataDAO)
	dbBackupService := service.NewDbBackupService(dbBackupDAO, cfg)
	fileService := service.NewFileService(fileDAO)
	systemSettingService := service.NewSystemSettingService(systemSettingDAO)

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
	hub := ws.NewHub()   // hub 就是 ws连接容器
	wsCtrl := controller.NewWSController(hub, rdb, cfg)  // wsCtrl 负责 /api/ws 接口
	notificationService := service.NewNotificationService(notificationDAO, userDAO, hub)

	// Export Worker（消费导出任务）
	exportWorker := worker.NewExportWorker(rdb, amqpConn, logService, hub, notificationService)
	go exportWorker.Start()

	// Mail Worker（消费发邮件任务）
	mailWorker := worker.NewMailWorker(amqpConn, alertMailService)
	go mailWorker.Start()

	// 定时任务调度器（内置命令进程内执行，注入真实清理实现）
	scheduler.Commands["clean_logs"] = scheduler.CommandDef{
		Name:  "clean_logs",
		Label: "清理过期日志",
		Handler: func(days int) (scheduler.CommandResult, error) {
			// 1. 保留天数：入参无效时读取配置，再无效 fallback 30
			if days < 1 || days > 3650 {
				cfg, _ := systemSettingService.GetAll()
				if v, ok := cfg["log_cleanup_days"]; ok {
					if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 3650 {
						days = n
					}
				}
				if days < 1 || days > 3650 {
					days = 30
				}
			}

			// 2. 清理范围：读取配置，默认全部
			scope := []string{"operation", "login"}
			cfg, _ := systemSettingService.GetAll()
			if v, ok := cfg["log_cleanup_scope"]; ok && v != "" {
				var parsed []string
				if json.Unmarshal([]byte(v), &parsed) == nil && len(parsed) > 0 {
					scope = parsed
				}
			}

			// 3. 按 scope 执行清理
			var msgs []string
			for _, s := range scope {
				switch s {
				case "operation":
					n, err := logService.Cleanup(days)
					if err != nil {
						return scheduler.CommandResult{}, err
					}
					msgs = append(msgs, fmt.Sprintf("操作日志 %d 条", n))
				case "login":
					n, err := loginLogService.Cleanup(days)
					if err != nil {
						return scheduler.CommandResult{}, err
					}
					msgs = append(msgs, fmt.Sprintf("登录日志 %d 条", n))
				}
			}
			return scheduler.CommandResult{
				Message: "清理完成：" + strings.Join(msgs, "，"),
			}, nil
		},
	}

	scheduler.Commands["backup_db"] = scheduler.CommandDef{
		Name:   "backup_db",
		Label:  "数据库备份",
		Handler: func(days int) (scheduler.CommandResult, error) {
			backup, err := dbBackupService.Backup("cron")
			if err != nil {
				return scheduler.CommandResult{}, err
			}
			return scheduler.CommandResult{
				Message: fmt.Sprintf("备份完成: %s (%d bytes)", backup.Filename, backup.FileSize),
			}, nil
		},
	}

	scheduler.Commands["clean_backup"] = scheduler.CommandDef{
		Name:   "clean_backup",
		Label:  "清理过期备份",
		Handler: func(days int) (scheduler.CommandResult, error) {
			if days < 1 || days > 3650 {
				days = 90
			}
			n, err := dbBackupService.Cleanup(days)
			if err != nil {
				return scheduler.CommandResult{}, err
			}
			return scheduler.CommandResult{
				Message: fmt.Sprintf("清理完成: %d 条过期备份", n),
			}, nil
		},
	}

	taskScheduler := scheduler.NewScheduler(cronTaskDAO, cronTaskExecutionDAO)
	go taskScheduler.Start()

	// Controller
	authCtrl := controller.NewAuthController(authService, menuDAO, userDAO, loginLogService, rdb, publishCh)
	userCtrl := controller.NewUserController(userService)
	roleCtrl := controller.NewRoleController(roleService)
	menuCtrl := controller.NewMenuController(menuService)
	logCtrl := controller.NewLogController(logService, loginLogService, rdb, publishCh, cfg.LogCleanupSecret)
	loginLogCtrl := controller.NewLoginLogController(loginLogService)
	dictTypeCtrl := controller.NewDictTypeController(dictTypeService)
	dictDataCtrl := controller.NewDictDataController(dictDataService)
	cronTaskService := service.NewCronTaskService(cronTaskDAO, cronTaskExecutionDAO, taskScheduler)
	cronTaskCtrl := controller.NewCronTaskController(cronTaskService)
	dbBackupCtrl := controller.NewDbBackupController(dbBackupService)
	dashboardCtrl := controller.NewDashboardController()
	healthCtrl := controller.NewHealthController(db, rdb, esClient, amqpConn)
	uploadCtrl := controller.NewUploadController(userDAO)
	fileCtrl := controller.NewFileController(fileService)
	settingCtrl := controller.NewSystemSettingController(systemSettingService)
	notificationCtrl := controller.NewNotificationController(notificationService)
	logSettingCtrl := controller.NewLogSettingController(systemSettingService)

	// Router
	r := router.Setup(cfg, authCtrl, userCtrl, roleCtrl, menuCtrl, logCtrl, loginLogCtrl, wsCtrl, uploadCtrl, authService, userDAO, menuDAO, logDAO, logRepo, dictTypeCtrl, dictDataCtrl, cronTaskCtrl, dbBackupCtrl, fileCtrl, dashboardCtrl, healthCtrl, settingCtrl, notificationCtrl, logSettingCtrl)

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
