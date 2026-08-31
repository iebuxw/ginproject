package controller

import (
	"bytes"
	"encoding/base64"

	"ginproject/internal/utils"

	"github.com/dchest/captcha"
	"github.com/gin-gonic/gin"
)

// CaptchaController 验证码接口
type CaptchaController struct{}

func NewCaptchaController() *CaptchaController {
	return &CaptchaController{}
}

// Generate 生成验证码
// @Summary 获取验证码
// @Description 生成 4 位数字图片验证码，返回 captcha_id 和 base64 编码的图片
// @Tags 认证
// @Produce json
// @Success 200 {object} utils.Response{data=object{captcha_id=string,captcha_image=string}} "成功"
// @Router /auth/captcha [get]
func (ctl *CaptchaController) Generate(c *gin.Context) {
	id := captcha.NewLen(4)

	var buf bytes.Buffer
	if err := captcha.WriteImage(&buf, id, 240, 80); err != nil {
		utils.Error(c, 500, "验证码生成失败")
		return
	}

	imgBase64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	utils.Success(c, gin.H{
		"captcha_id":    id,
		"captcha_image": "data:image/png;base64," + imgBase64,
	})
}
