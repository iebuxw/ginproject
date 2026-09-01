package middleware

import (
	"bytes"
	"encoding/json"
	"ginproject/internal/dao"
	"ginproject/internal/es"
	"ginproject/internal/logger"
	"ginproject/internal/model"
	"io"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"

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

		params := maskSensitiveParams(body)
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
				logger.Warn("ES 日志写入失败", zap.Error(err))
			}
		}
	}
}

var (
	jsonSensitiveRe = regexp.MustCompile(`(?i)("[^"]*password[^"]*"\s*:\s*)("[^"]*"|[^,}\s]+)`)
	formSensitiveRe = regexp.MustCompile(`(?i)\b(password[a-z_]*)=[^&\s"]+`)
)

// maskSensitiveParams 将请求体中密码类字段的值替换为 ***，JSON 与非 JSON 均处理
func maskSensitiveParams(body string) string {
	var data interface{}
	if err := json.Unmarshal([]byte(body), &data); err == nil {
		maskJSONValue(data)
		if out, err := json.Marshal(data); err == nil {
			return string(out)
		}
	}
	masked := jsonSensitiveRe.ReplaceAllString(body, `${1}"***"`)
	return formSensitiveRe.ReplaceAllString(masked, `$1=***`)
}

func maskJSONValue(v interface{}) {
	switch t := v.(type) {
	case map[string]interface{}:
		for k, val := range t {
			if strings.Contains(strings.ToLower(k), "password") {
				t[k] = "***"
			} else {
				maskJSONValue(val)
			}
		}
	case []interface{}:
		for _, item := range t {
			maskJSONValue(item)
		}
	}
}
