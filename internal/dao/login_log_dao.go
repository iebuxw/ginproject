package dao

import (
	"ginproject/internal/model"

	"gorm.io/gorm"
)

type LoginLogDAO struct{ db *gorm.DB }

func NewLoginLogDAO(db *gorm.DB) *LoginLogDAO { return &LoginLogDAO{db: db} }

func (d *LoginLogDAO) Create(log *model.LoginLog) error {
	return d.db.Create(log).Error
}

func (d *LoginLogDAO) FindPage(page, pageSize int, username string, status int) ([]model.LoginLog, int64, error) {
	var logs []model.LoginLog
	var total int64
	q := d.db.Model(&model.LoginLog{})
	if username != "" {
		q = q.Where("username LIKE ?", "%"+username+"%")
	}
	if status > 0 {
		q = q.Where("status = ?", status)
	}
	q.Count(&total)
	err := q.Offset((page - 1) * pageSize).Limit(pageSize).Order("id DESC").Find(&logs).Error
	return logs, total, err
}
