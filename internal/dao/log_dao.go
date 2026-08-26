package dao

import (
	"time"

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

// LogFilter 日志查询筛选条件
type LogFilter struct {
	Module    string
	Method    string
	Keyword   string
	StartTime string
	EndTime   string
}

func (d *LogDAO) FindBatch(f LogFilter, offset, limit int) ([]model.OperationLog, error) {
	var logs []model.OperationLog
	q := d.db.Model(&model.OperationLog{})
	if f.Module != "" {
		q = q.Where("module = ?", f.Module)
	}
	if f.Method != "" {
		q = q.Where("method = ?", f.Method)
	}
	if f.Keyword != "" {
		q = q.Where("(path LIKE ? OR params LIKE ?)", "%"+f.Keyword+"%", "%"+f.Keyword+"%")
	}
	if f.StartTime != "" {
		q = q.Where("created_at >= ?", f.StartTime)
	}
	if f.EndTime != "" {
		q = q.Where("created_at <= ?", f.EndTime)
	}
	err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&logs).Error
	return logs, err
}

// DeleteOlderThan 删除创建时间早于 before 的日志，最多删除 limit 条；返回实际删除行数
func (d *LogDAO) DeleteOlderThan(before time.Time, limit int) (int64, error) {
	res := d.db.Where("created_at < ?", before).Limit(limit).Delete(&model.OperationLog{})
	return res.RowsAffected, res.Error
}
