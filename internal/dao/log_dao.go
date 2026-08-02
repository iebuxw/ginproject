package dao

import (
	"ginproject/internal/model"

	"gorm.io/gorm"
)

type LogDAO struct{ db *gorm.DB }

func NewLogDAO(db *gorm.DB) *LogDAO { return &LogDAO{db} }

func (d *LogDAO) Create(log *model.OperationLog) error {
	return d.db.Create(log).Error
}

func (d *LogDAO) FindPage(page, pageSize int, module, method string) ([]model.OperationLog, int64, error) {
	var logs []model.OperationLog
	var total int64
	q := d.db.Model(&model.OperationLog{})
	if module != "" {
		q = q.Where("module = ?", module)
	}
	if method != "" {
		q = q.Where("method = ?", method)
	}
	q.Count(&total)
	err := q.Offset((page - 1) * pageSize).Limit(pageSize).Order("id DESC").Find(&logs).Error
	return logs, total, err
}

func (d *LogDAO) FindAll(module, method string) ([]model.OperationLog, error) {
	var logs []model.OperationLog
	q := d.db.Model(&model.OperationLog{})
	if module != "" {
		q = q.Where("module = ?", module)
	}
	if method != "" {
		q = q.Where("method = ?", method)
	}
	err := q.Order("id DESC").Find(&logs).Error
	return logs, err
}

func (d *LogDAO) FindBatch(module, method string, offset, limit int) ([]model.OperationLog, error) {
	var logs []model.OperationLog
	q := d.db.Model(&model.OperationLog{})
	if module != "" {
		q = q.Where("module = ?", module)
	}
	if method != "" {
		q = q.Where("method = ?", method)
	}
	err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&logs).Error
	return logs, err
}
