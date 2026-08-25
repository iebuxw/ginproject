package service

import (
	"context"

	"ginproject/internal/dao"
	"ginproject/internal/es"
	"ginproject/internal/model"
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
