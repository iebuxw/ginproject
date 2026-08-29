package dao

import (
	"time"

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
	if status >= 0 {
		q = q.Where("status = ?", status)
	}
	q.Count(&total)
	err := q.Offset((page - 1) * pageSize).Limit(pageSize).Order("id DESC").Find(&logs).Error
	return logs, total, err
}

// DeleteOlderThan 删除创建时间早于 before 的登录日志，最多删除 limit 条；返回实际删除行数
func (d *LoginLogDAO) DeleteOlderThan(before time.Time, limit int) (int64, error) {
	res := d.db.Where("created_at < ?", before).Limit(limit).Delete(&model.LoginLog{})
	return res.RowsAffected, res.Error
}
