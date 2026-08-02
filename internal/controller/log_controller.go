package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"ginproject/internal/service"
	"ginproject/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

type LogController struct {
	logService *service.LogService
	rdb        *redis.Client
	amqpCh     *amqp091.Channel
}

func NewLogController(logService *service.LogService, rdb *redis.Client, amqpCh *amqp091.Channel) *LogController {
	return &LogController{logService: logService, rdb: rdb, amqpCh: amqpCh}
}

func (ctl *LogController) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	module := c.Query("module")
	method := c.Query("method")
	logs, total, err := ctl.logService.FindPage(page, pageSize, module, method)
	if err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, gin.H{"list": logs, "total": total})
}

func (ctl *LogController) Export(c *gin.Context) {
	var req struct {
		Method string `json:"method"`
	}
	c.ShouldBindJSON(&req)
	userID, _ := c.Get("user_id")

	taskID := utils.NewUUID()
	taskKey := "excel:task:" + taskID

	ctx := context.Background()
	ctl.rdb.HSet(ctx, taskKey,
		"status", "pending",
		"user_id", fmt.Sprintf("%d", userID),
		"method", req.Method,
	)
	ctl.rdb.Expire(ctx, taskKey, 24*time.Hour)

	body, _ := json.Marshal(map[string]string{"task_id": taskID})
	ctl.amqpCh.Publish("", "excel.export", false, false, amqp091.Publishing{
		ContentType: "application/json",
		Body:        body,
	})

	utils.Success(c, gin.H{"task_id": taskID})
}

func (ctl *LogController) ExportStatus(c *gin.Context) {
	taskID := c.Query("task_id")
	if taskID == "" {
		utils.Error(c, 400, "缺少task_id")
		return
	}
	fields, err := ctl.rdb.HGetAll(context.Background(), "excel:task:"+taskID).Result()
	if err != nil || len(fields) == 0 {
		utils.Error(c, 404, "任务不存在或已过期")
		return
	}
	utils.Success(c, fields)
}

func (ctl *LogController) Download(c *gin.Context) {
	taskID := c.Param("taskID")
	taskKey := "excel:task:" + taskID

	userID, _ := c.Get("user_id")
	taskUserID, err := ctl.rdb.HGet(context.Background(), taskKey, "user_id").Result()
	if err != nil || taskUserID != fmt.Sprintf("%d", userID) {
		utils.Error(c, 403, "无权下载或任务不存在")
		return
	}

	status, _ := ctl.rdb.HGet(context.Background(), taskKey, "status").Result()
	if status != "success" {
		utils.Error(c, 400, "文件尚未生成或已失败")
		return
	}

	filename, _ := ctl.rdb.HGet(context.Background(), taskKey, "filename").Result()
	filePath := filepath.Join("exports", taskID+".xlsx")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		utils.Error(c, 404, "文件已被下载或不存在")
		return
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.File(filePath)

	os.Remove(filePath)
}
