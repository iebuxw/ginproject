package controller

import (
	"context"
	"net/http"
	"time"

	"ginproject/internal/es"
	"ginproject/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// HealthController 健康检查
type HealthController struct {
	db        *gorm.DB
	rdb       *redis.Client
	esClient  *es.Client
	amqpConn  *amqp091.Connection
}

func NewHealthController(db *gorm.DB, rdb *redis.Client, esClient *es.Client, amqpConn *amqp091.Connection) *HealthController {
	return &HealthController{db: db, rdb: rdb, esClient: esClient, amqpConn: amqpConn}
}

type serviceStatus struct {
	Status  string `json:"status" example:"ok"`
	Message string `json:"message,omitempty" example:""`
}

type healthResponse struct {
	Status   string                   `json:"status" example:"ok"`
	Services map[string]serviceStatus `json:"services"`
}

// Check 健康检查端点，检查各依赖服务连通性
// @Summary      健康检查
// @Description  检查 MySQL、Redis、Elasticsearch、RabbitMQ 连通性，全部正常返回 200，任一异常返回 503
// @Tags         health
// @Produce      json
// @Success      200  {object}  utils.Response{data=healthResponse}
// @Failure      503  {object}  utils.Response{data=healthResponse}
// @Router       /api/health [get]
func (ctl *HealthController) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	resp := healthResponse{Services: make(map[string]serviceStatus)}
	allOK := true

	// MySQL
	if err := ctl.checkMySQL(ctx); err != nil {
		resp.Services["mysql"] = serviceStatus{Status: "down", Message: err.Error()}
		allOK = false
	} else {
		resp.Services["mysql"] = serviceStatus{Status: "ok"}
	}

	// Redis
	if err := ctl.rdb.Ping(ctx).Err(); err != nil {
		resp.Services["redis"] = serviceStatus{Status: "down", Message: err.Error()}
		allOK = false
	} else {
		resp.Services["redis"] = serviceStatus{Status: "ok"}
	}

	// Elasticsearch（可选组件，不可用不标记整体 down）
	if ctl.esClient == nil {
		resp.Services["elasticsearch"] = serviceStatus{Status: "disabled", Message: "未配置"}
	} else if err := ctl.esClient.Ping(); err != nil {
		resp.Services["elasticsearch"] = serviceStatus{Status: "down", Message: err.Error()}
	} else {
		resp.Services["elasticsearch"] = serviceStatus{Status: "ok"}
	}

	// RabbitMQ
	if ctl.amqpConn.IsClosed() {
		resp.Services["rabbitmq"] = serviceStatus{Status: "down", Message: "连接已关闭"}
		allOK = false
	} else {
		resp.Services["rabbitmq"] = serviceStatus{Status: "ok"}
	}

	if allOK {
		resp.Status = "ok"
		utils.Success(c, resp)
	} else {
		resp.Status = "degraded"
		c.JSON(http.StatusServiceUnavailable, utils.Response{
			Code:    503,
			Message: "service degraded",
			Data:    resp,
		})
	}
}

func (ctl *HealthController) checkMySQL(ctx context.Context) error {
	sqlDB, err := ctl.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
