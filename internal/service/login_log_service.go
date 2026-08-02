package service

import (
	"ginproject/internal/dao"
	"ginproject/internal/model"
)

type LoginLogService struct {
	loginLogDAO *dao.LoginLogDAO
}

func NewLoginLogService(loginLogDAO *dao.LoginLogDAO) *LoginLogService {
	return &LoginLogService{loginLogDAO: loginLogDAO}
}

func (s *LoginLogService) Create(log *model.LoginLog) error {
	return s.loginLogDAO.Create(log)
}

func (s *LoginLogService) FindPage(page, pageSize int, username string, status int) ([]model.LoginLog, int64, error) {
	return s.loginLogDAO.FindPage(page, pageSize, username, status)
}
