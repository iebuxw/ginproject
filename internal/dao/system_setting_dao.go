package dao

import (
	"ginproject/internal/model"
	"gorm.io/gorm"
)

type SystemSettingDAO struct {
	db *gorm.DB
}

func NewSystemSettingDAO(db *gorm.DB) *SystemSettingDAO {
	return &SystemSettingDAO{db: db}
}

func (d *SystemSettingDAO) FindByKey(key string) (*model.SystemSetting, error) {
	var s model.SystemSetting
	err := d.db.Where("setting_key = ?", key).First(&s).Error
	return &s, err
}

func (d *SystemSettingDAO) FindAll() ([]model.SystemSetting, error) {
	var list []model.SystemSetting
	err := d.db.Find(&list).Error
	return list, err
}

func (d *SystemSettingDAO) Upsert(key, value string) error {
	var s model.SystemSetting
	err := d.db.Where("setting_key = ?", key).First(&s).Error
	if err == gorm.ErrRecordNotFound {
		return d.db.Exec("INSERT INTO system_settings (setting_key, setting_value, created_at, updated_at) VALUES (?, ?, NOW(), NOW())", key, value).Error
	}
	if err != nil {
		return err
	}
	return d.db.Model(&s).Update("setting_value", value).Error
}
