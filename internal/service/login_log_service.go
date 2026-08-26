package service

import (
	"time"

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

// Cleanup 分批删除保留天数之外的登录日志
func (s *LoginLogService) Cleanup(days int) (int64, error) {
	before := time.Now().AddDate(0, 0, -days)
	var total int64
	for {
		n, err := s.loginLogDAO.DeleteOlderThan(before, 1000)
		if err != nil {
			return total, err
		}
		total += n
		if n == 0 {
			break
		}
	}
	return total, nil
}
