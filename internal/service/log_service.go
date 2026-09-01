package service

import (
	"context"
	"time"

	"ginproject/internal/dao"
	"ginproject/internal/es"
	"ginproject/internal/logger"
	"ginproject/internal/model"

	"go.uber.org/zap"
)

type LogService struct {
	logDAO  *dao.LogDAO
	logRepo *es.LogRepo
}

func NewLogService(logDAO *dao.LogDAO, logRepo *es.LogRepo) *LogService {
	return &LogService{logDAO: logDAO, logRepo: logRepo}
}

func (s *LogService) Create(log *model.OperationLog) error { return s.logDAO.Create(log) }

// ESEnabled 判断 ES 是否可用（决定查询是否走 ES）
func (s *LogService) ESEnabled() bool { return s.logRepo.Enabled() }

// SearchFromES 走 ES 全文检索
func (s *LogService) SearchFromES(q es.SearchQuery) ([]es.SearchHitDoc, int64, error) {
	return s.logRepo.Search(context.Background(), q)
}

func (s *LogService) FindPage(page, pageSize int, module, method string) ([]model.OperationLog, int64, error) {
	return s.logDAO.FindPage(page, pageSize, module, method)
}

func (s *LogService) FindAll(module, method string) ([]model.OperationLog, error) {
	return s.logDAO.FindAll(module, method)
}

func (s *LogService) FindBatch(f dao.LogFilter, offset, limit int) ([]model.OperationLog, error) {
	return s.logDAO.FindBatch(f, offset, limit)
}

// Cleanup 分批删除保留天数之外的操作日志，并同步清理 ES；ES 不可用仅告警不阻断
func (s *LogService) Cleanup(days int) (int64, error) {
	before := time.Now().AddDate(0, 0, -days)
	var total int64
	for {
		n, err := s.logDAO.DeleteOlderThan(before, 1000)
		if err != nil {
			return total, err
		}
		total += n
		if n == 0 {
			break
		}
	}
	if n, err := s.logRepo.DeleteByTime(before); err != nil {
		logger.Warn("ES 清理旧操作日志失败（已降级仅清 MySQL）", zap.Error(err))
	} else if n > 0 {
		logger.Info("ES 已清理旧操作日志", zap.Int64("count", n))
	}
	return total, nil
}
