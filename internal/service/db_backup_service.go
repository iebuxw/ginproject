package service

import (
	"fmt"
	"ginproject/internal/config"
	"ginproject/internal/dao"
	"ginproject/internal/model"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type DbBackupService struct {
	backupDAO *dao.DbBackupDAO
	cfg       *config.Config
	backupDir string
}

func NewDbBackupService(backupDAO *dao.DbBackupDAO, cfg *config.Config) *DbBackupService {
	return &DbBackupService{
		backupDAO: backupDAO,
		cfg:       cfg,
		backupDir: "backups",
	}
}

func (s *DbBackupService) Backup(triggerType string) (*model.DbBackup, error) {
	if err := os.MkdirAll(s.backupDir, 0755); err != nil {
		return nil, fmt.Errorf("创建备份目录失败: %w", err)
	}

	filename := fmt.Sprintf("%s_%s.sql.gz", s.cfg.Database.DBName, time.Now().Format("20060102_150405"))
	filepath := filepath.Join(s.backupDir, filename)

	cmd := exec.Command("mysqldump",
		"-h"+s.cfg.Database.Host,
		"-P"+s.cfg.Database.Port,
		"-u"+s.cfg.Database.User,
		"-p"+s.cfg.Database.Password,
		s.cfg.Database.DBName,
	)

	gzipCmd := exec.Command("gzip")
	gzipCmd.Stdin, _ = cmd.StdoutPipe()
	outFile, err := os.Create(filepath)
	if err != nil {
		return nil, fmt.Errorf("创建备份文件失败: %w", err)
	}
	gzipCmd.Stdout = outFile

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("启动 mysqldump 失败: %w", err)
	}
	if err := gzipCmd.Start(); err != nil {
		return nil, fmt.Errorf("启动 gzip 失败: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		os.Remove(filepath)
		return nil, fmt.Errorf("mysqldump 执行失败: %w", err)
	}
	if err := gzipCmd.Wait(); err != nil {
		os.Remove(filepath)
		return nil, fmt.Errorf("gzip 执行失败: %w", err)
	}
	outFile.Close()

	info, err := os.Stat(filepath)
	if err != nil {
		return nil, fmt.Errorf("获取文件大小失败: %w", err)
	}

	backup := &model.DbBackup{
		Filename:    filename,
		FileSize:    info.Size(),
		TriggerType: triggerType,
		Status:      0,
		Type:        "backup",
		CreatedAt:   model.DateTime(time.Now()),
	}

	if err := s.backupDAO.Create(backup); err != nil {
		return nil, fmt.Errorf("保存备份记录失败: %w", err)
	}

	return backup, nil
}

func (s *DbBackupService) Restore(id int64) error {
	backup, err := s.backupDAO.FindByID(id)
	if err != nil {
		return fmt.Errorf("备份记录不存在: %w", err)
	}

	filepath := filepath.Join(s.backupDir, backup.Filename)
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在: %s", backup.Filename)
	}

	mysqlConn := fmt.Sprintf("mysql -h%s -P%s -u%s -p%s %s",
		s.cfg.Database.Host, s.cfg.Database.Port, s.cfg.Database.User, s.cfg.Database.Password, s.cfg.Database.DBName)

	cmds := []string{
		mysqlConn + " -e \"SET FOREIGN_KEY_CHECKS=0\"",
		fmt.Sprintf("gunzip -c %s | %s", filepath, mysqlConn),
		mysqlConn + " -e \"SET FOREIGN_KEY_CHECKS=1\"",
	}

	for _, cmdStr := range cmds {
		cmd := exec.Command("sh", "-c", cmdStr)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("恢复失败: %s", string(output))
		}
	}

	return nil
}

func (s *DbBackupService) Delete(id int64) error {
	backup, err := s.backupDAO.FindByID(id)
	if err != nil {
		return fmt.Errorf("备份记录不存在: %w", err)
	}

	filepath := filepath.Join(s.backupDir, backup.Filename)
	if _, err := os.Stat(filepath); err == nil {
		os.Remove(filepath)
	}

	return s.backupDAO.Delete(id)
}

func (s *DbBackupService) GetByID(id int64) (*model.DbBackup, error) {
	return s.backupDAO.FindByID(id)
}

func (s *DbBackupService) GetFilePath(filename string) string {
	return filepath.Join(s.backupDir, filename)
}

func (s *DbBackupService) List(page, pageSize int, startTime, endTime string) ([]model.DbBackup, int64, error) {
	return s.backupDAO.FindPage(page, pageSize, startTime, endTime)
}

func (s *DbBackupService) Cleanup(days int) (int, error) {
	if days < 1 {
		days = 90
	}

	backups, err := s.backupDAO.FindOlderThan(days)
	if err != nil {
		return 0, err
	}

	deleted := 0
	ids := make([]int64, 0, len(backups))
	for _, b := range backups {
		filepath := filepath.Join(s.backupDir, b.Filename)
		if _, err := os.Stat(filepath); err == nil {
			os.Remove(filepath)
		}
		ids = append(ids, b.ID)
		deleted++
	}

	if len(ids) > 0 {
		if err := s.backupDAO.BatchDelete(ids); err != nil {
			return deleted, err
		}
	}

	return deleted, nil
}
