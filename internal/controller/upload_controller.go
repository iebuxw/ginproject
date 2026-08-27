package controller

import (
	"crypto/rand"
	"fmt"
	"ginproject/internal/dao"
	"ginproject/internal/utils"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

type UploadController struct {
	uploadDir string
	userDAO   *dao.UserDAO
}

func NewUploadController(userDAO *dao.UserDAO) *UploadController {
	return &UploadController{uploadDir: "./uploads/avatars", userDAO: userDAO}
}

// UploadAvatar 上传头像
func (ctl *UploadController) UploadAvatar(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		utils.Error(c, 400, "请选择文件")
		return
	}
	defer file.Close()

	if err := validateAvatar(header); err != nil {
		utils.Error(c, 400, err.Error())
		return
	}

	if err := os.MkdirAll(ctl.uploadDir, 0755); err != nil {
		utils.Error(c, 500, "创建目录失败")
		return
	}

	filename := randomHex(16) + strings.ToLower(filepath.Ext(header.Filename))
	savePath := filepath.Join(ctl.uploadDir, filename)

	dst, err := os.Create(savePath)
	if err != nil {
		utils.Error(c, 500, "保存文件失败")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		utils.Error(c, 500, "保存文件失败")
		return
	}

	avatarURL := "/api/uploads/avatars/" + filename

	// 上传成功后直接更新当前用户的 avatar 字段
	userID, _ := c.Get("user_id")
	if uid, ok := userID.(uint); ok {
		if err := ctl.userDAO.UpdateAvatar(uid, avatarURL); err != nil {
			utils.Error(c, 500, "保存头像记录失败")
			return
		}
	}

	utils.Success(c, gin.H{"url": avatarURL})
}

var allowedExts = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true}

func validateAvatar(header *multipart.FileHeader) error {
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExts[ext] {
		return fmt.Errorf("仅支持 jpg/jpeg/png/gif 格式")
	}
	if header.Size > 2*1024*1024 {
		return fmt.Errorf("文件大小不能超过 2MB")
	}
	return nil
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
