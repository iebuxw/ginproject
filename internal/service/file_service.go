package service

import (
	"crypto/rand"
	"fmt"
	"ginproject/internal/dao"
	"ginproject/internal/model"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// 黑名单：仅挡可执行/脚本类扩展名，其余类型不限
var forbiddenExts = map[string]bool{
	".exe": true, ".dll": true, ".bat": true, ".cmd": true, ".com": true,
	".msi": true, ".vbs": true, ".sh": true, ".reg": true,
}

const MaxFileSize = 100 * 1024 * 1024

type FileService struct {
	fileDAO   *dao.FileDAO
	uploadDir string
}

func NewFileService(fileDAO *dao.FileDAO) *FileService {
	return &FileService{fileDAO: fileDAO, uploadDir: "./uploads/files"}
}

func (s *FileService) Upload(originalName string, size int64, src io.Reader, uploaderID int64, uploaderName string) (*model.File, error) {
	ext := strings.ToLower(filepath.Ext(originalName))
	if forbiddenExts[ext] {
		return nil, fmt.Errorf("不允许上传该文件类型: %s", ext)
	}
	if size > MaxFileSize {
		return nil, fmt.Errorf("文件大小不能超过 100MB")
	}

	if err := os.MkdirAll(s.uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("创建上传目录失败: %w", err)
	}

	storedName := randomHexName(16) + ext
	savePath := filepath.Join(s.uploadDir, storedName)

	dst, err := os.Create(savePath)
	if err != nil {
		return nil, fmt.Errorf("保存文件失败: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(savePath)
		return nil, fmt.Errorf("保存文件失败: %w", err)
	}

	file := &model.File{
		OriginalName: originalName,
		StoredName:   storedName,
		Size:         size,
		Ext:          strings.TrimPrefix(ext, "."),
		UploaderID:   uploaderID,
		UploaderName: uploaderName,
		CreatedAt:    model.DateTime(time.Now()),
	}
	if err := s.fileDAO.Create(file); err != nil {
		os.Remove(savePath)
		return nil, fmt.Errorf("保存文件记录失败: %w", err)
	}
	return file, nil
}

func (s *FileService) List(page, pageSize int, name string) ([]model.File, int64, error) {
	return s.fileDAO.FindPage(page, pageSize, name)
}

func (s *FileService) Delete(id int64) error {
	file, err := s.fileDAO.FindByID(id)
	if err != nil {
		return fmt.Errorf("文件记录不存在: %w", err)
	}

	if err := s.fileDAO.Delete(id); err != nil {
		return fmt.Errorf("删除文件记录失败: %w", err)
	}

	// 物理文件删除失败仅记日志，不阻断（记录已删，孤儿文件留待手工清理）
	savePath := filepath.Join(s.uploadDir, file.StoredName)
	if _, err := os.Stat(savePath); err == nil {
		if err := os.Remove(savePath); err != nil {
			log.Printf("警告: 物理文件删除失败 %s: %v", savePath, err)
		}
	}
	return nil
}

func (s *FileService) GetByID(id int64) (*model.File, error) {
	return s.fileDAO.FindByID(id)
}

func (s *FileService) GetFilePath(storedName string) string {
	return filepath.Join(s.uploadDir, storedName)
}

func randomHexName(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
