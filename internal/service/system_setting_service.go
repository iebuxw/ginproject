package service

import (
	"ginproject/internal/dao"
)

type SystemSettingService struct {
	settingDAO *dao.SystemSettingDAO
}

func NewSystemSettingService(settingDAO *dao.SystemSettingDAO) *SystemSettingService {
	return &SystemSettingService{settingDAO: settingDAO}
}

func (s *SystemSettingService) GetAll() (map[string]string, error) {
	list, err := s.settingDAO.FindAll()
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(list))
	for _, v := range list {
		m[v.SettingKey] = v.SettingValue
	}
	return m, nil
}

func (s *SystemSettingService) Save(settings map[string]string) error {
	for k, v := range settings {
		if err := s.settingDAO.Upsert(k, v); err != nil {
			return err
		}
	}
	return nil
}
