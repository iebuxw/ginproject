package middleware

import (
	"bytes"
	"ginproject/internal/dao"
	"ginproject/internal/es"
	"ginproject/internal/model"
	"io"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func OperationLogger(logDAO *dao.LogDAO, logRepo *es.LogRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		var body string
		if c.Request.Body != nil {
			b, _ := io.ReadAll(c.Request.Body)
			body = string(b)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(b))
		}
		c.Next()

		if c.Request.Method == "GET" {
			return
		}

		params := body
		if params == "" {
			params = c.Request.URL.Path
		}

		duration := int(time.Since(start).Milliseconds())
		userID, _ := c.Get("user_id")
		uid, _ := userID.(uint)
		logEntry := &model.OperationLog{
			OperatorID: uid,
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			Params:     params,
			Duration:   duration,
			IP:         c.ClientIP(),
			CreatedAt:  model.DateTime(time.Now()),
		}
		_ = logDAO.Create(logEntry)

		// 双写 ES：同步写入，失败仅告警，不阻断请求
		if logRepo != nil && logRepo.Enabled() {
			if err := logRepo.Index(logEntry); err != nil {
				log.Printf("ES 日志写入失败: %v", err)
			}
		}
	}
}
