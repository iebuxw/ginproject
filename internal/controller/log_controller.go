package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"ginproject/internal/es"
	"ginproject/internal/service"
	"ginproject/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

type LogController struct {
	logService      *service.LogService
	loginLogService *service.LoginLogService
	rdb             *redis.Client
	amqpCh          *amqp091.Channel
	cleanupSecret   string
}

func NewLogController(logService *service.LogService, loginLogService *service.LoginLogService, rdb *redis.Client, amqpCh *amqp091.Channel, cleanupSecret string) *LogController {
	return &LogController{logService: logService, loginLogService: loginLogService, rdb: rdb, amqpCh: amqpCh, cleanupSecret: cleanupSecret}
}

// List 查询操作日志
// @Summary 查询操作日志
// @Description 分页查询操作日志，优先走 ES 全文检索，失败回退 MySQL
// @Tags 操作日志
// @Security BearerAuth
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param keyword query string false "搜索关键词"
// @Param module query string false "模块筛选"
// @Param method query string false "请求方法筛选"
// @Param start_time query string false "开始时间 (RFC3339)"
// @Param end_time query string false "结束时间 (RFC3339)"
// @Success 200 {object} utils.Response{data=object{list=array,total=int,data_source=string}} "成功"
// @Router /logs [get]
func (ctl *LogController) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	keyword := c.Query("keyword")
	module := c.Query("module")
	method := c.Query("method")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")

	// 优先走 ES 全文检索；失败或未启用时回退 MySQL
	if ctl.logService.ESEnabled() {
		hits, total, err := ctl.logService.SearchFromES(es.SearchQuery{
			Keyword:   keyword,
			Module:    module,
			Method:    method,
			StartTime: startTime,
			EndTime:   endTime,
			From:      (page - 1) * pageSize,
			Size:      pageSize,
		})
		if err == nil {
			utils.Success(c, gin.H{"list": hits, "total": total, "data_source": "es"})
			return
		}
		log.Printf("ES 查询失败，回退 MySQL: %v", err)
	}

	logs, total, err := ctl.logService.FindPage(page, pageSize, module, method)
	if err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, gin.H{"list": logs, "total": total, "data_source": "mysql"})
}

// Export 发起日志导出任务
// @Summary 发起日志导出任务
// @Description 异步导出操作日志为 Excel 文件，通过 WebSocket 通知完成
// @Tags 操作日志
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body object false "导出参数"
// @Param body.body.method body string false "请求方法筛选"
// @Param body.body.keyword body string false "搜索关键词（路径/参数）"
// @Param body.body.start_time body string false "开始时间 (yyyy-MM-dd HH:mm:ss)"
// @Param body.body.end_time body string false "结束时间 (yyyy-MM-dd HH:mm:ss)"
// @Success 200 {object} utils.Response{data=object{task_id=string}} "成功，返回任务 ID"
// @Router /logs/export [post]
func (ctl *LogController) Export(c *gin.Context) {
	var req struct {
		Method    string `json:"method"`
		Keyword   string `json:"keyword"`
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
	}
	c.ShouldBindJSON(&req)
	uid := c.GetUint("user_id")

	taskID := utils.NewUUID()
	taskKey := "excel:task:" + taskID

	ctx := context.Background()
	ctl.rdb.HSet(ctx, taskKey, "status", "pending")
	ctl.rdb.HSet(ctx, taskKey, "user_id", fmt.Sprintf("%d", uid))
	ctl.rdb.HSet(ctx, taskKey, "method", req.Method)
	ctl.rdb.HSet(ctx, taskKey, "keyword", req.Keyword)
	ctl.rdb.HSet(ctx, taskKey, "start_time", req.StartTime)
	ctl.rdb.HSet(ctx, taskKey, "end_time", req.EndTime)
	ctl.rdb.Expire(ctx, taskKey, 24*time.Hour)

	body, _ := json.Marshal(map[string]string{"task_id": taskID})
	ctl.amqpCh.Publish("", "excel.export", false, false, amqp091.Publishing{
		ContentType: "application/json",
		Body:        body,
	})

	utils.Success(c, gin.H{"task_id": taskID})
}

// ExportStatus 查询导出任务状态
// @Summary 查询导出任务状态
// @Description 根据任务 ID 查询导出进度
// @Tags 操作日志
// @Security BearerAuth
// @Produce json
// @Param task_id query string true "任务 ID"
// @Success 200 {object} utils.Response{data=object{status=string,filename=string}} "成功"
// @Failure 200 {object} utils.Response "任务不存在"
// @Router /logs/export-status [get]
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

// Download 下载导出文件
// @Summary 下载导出文件
// @Description 下载已完成的导出文件，下载后服务端自动删除
// @Tags 操作日志
// @Security BearerAuth
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param taskID path string true "任务 ID"
// @Success 200 {file} binary "Excel 文件"
// @Failure 200 {object} utils.Response "无权下载或文件不存在"
// @Router /logs/download/{taskID} [get]
func (ctl *LogController) Download(c *gin.Context) {
	taskID := c.Param("taskID")
	taskKey := "excel:task:" + taskID

	uid := c.GetUint("user_id")
	taskUserID, err := ctl.rdb.HGet(context.Background(), taskKey, "user_id").Result()
	if err != nil || taskUserID != fmt.Sprintf("%d", uid) {
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

// Cleanup 定时清理旧日志（公开路由，secret 校验）
// @Summary 清理旧日志
// @Description 按保留天数分批删除旧日志（操作日志/登录日志），ES 同步清理。供定时任务调用，需携带 secret
// @Tags 操作日志
// @Produce json
// @Param secret query string true "清理密钥（与 LOG_CLEANUP_SECRET 比对）"
// @Param days query int false "保留天数（删除创建时间早于 now-days 的日志）" default(30)
// @Param scope query string false "清理范围：operation/login/all，默认 all"
// @Success 200 {object} utils.Response{data=object{operation_deleted=int,login_deleted=int}} "成功"
// @Failure 200 {object} utils.Response "参数非法"
// @Router /logs/cleanup [post]
func (ctl *LogController) Cleanup(c *gin.Context) {
	secret := c.Query("secret")
	if ctl.cleanupSecret == "" || secret != ctl.cleanupSecret {
		utils.ErrorWithStatus(c, http.StatusForbidden, 403, "密钥无效")
		return
	}
	days, err := strconv.Atoi(c.DefaultQuery("days", "30"))
	if err != nil || days < 1 || days > 3650 {
		utils.Error(c, 400, "days 参数非法（1~3650）")
		return
	}
	scope := c.DefaultQuery("scope", "all")
	if scope != "operation" && scope != "login" && scope != "all" {
		utils.Error(c, 400, "scope 参数非法（operation/login/all）")
		return
	}
	result := gin.H{}
	if scope == "operation" || scope == "all" {
		n, err := ctl.logService.Cleanup(days)
		if err != nil {
			utils.Error(c, 500, "操作日志清理失败: "+err.Error())
			return
		}
		result["operation_deleted"] = n
	}
	if scope == "login" || scope == "all" {
		n, err := ctl.loginLogService.Cleanup(days)
		if err != nil {
			utils.Error(c, 500, "登录日志清理失败: "+err.Error())
			return
		}
		result["login_deleted"] = n
	}
	utils.Success(c, result)
}
