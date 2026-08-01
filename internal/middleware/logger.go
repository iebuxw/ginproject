package middleware

import (
	"bytes"
	"ginproject/internal/dao"
	"ginproject/internal/model"
	"io"
	"time"

	"github.com/gin-gonic/gin"
)

func OperationLogger(logDAO *dao.LogDAO) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		var body string
		if c.Request.Body != nil {
			b, _ := io.ReadAll(c.Request.Body)
			body = string(b)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(b))
		}
		c.Next()
		duration := int(time.Since(start).Milliseconds())
		userID, _ := c.Get("user_id")
		uid, _ := userID.(uint)
		log := &model.OperationLog{
			OperatorID: uid,
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			Params:     body,
			Duration:   duration,
			IP:         c.ClientIP(),
		}
		_ = logDAO.Create(log)
	}
}
