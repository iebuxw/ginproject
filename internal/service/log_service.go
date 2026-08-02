package service

import (
	"ginproject/internal/dao"
	"ginproject/internal/model"
)

type LogService struct{ logDAO *dao.LogDAO }

func NewLogService(logDAO *dao.LogDAO) *LogService { return &LogService{logDAO} }

func (s *LogService) Create(log *model.OperationLog) error { return s.logDAO.Create(log) }

func (s *LogService) FindPage(page, pageSize int, module, method string) ([]model.OperationLog, int64, error) {
	return s.logDAO.FindPage(page, pageSize, module, method)
}

func (s *LogService) FindAll(module, method string) ([]model.OperationLog, error) {
	return s.logDAO.FindAll(module, method)
}

func (s *LogService) FindBatch(module, method string, offset, limit int) ([]model.OperationLog, error) {
	return s.logDAO.FindBatch(module, method, offset, limit)
}
