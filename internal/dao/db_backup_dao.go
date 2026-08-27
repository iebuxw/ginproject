package dao

import (
	"ginproject/internal/model"
	"gorm.io/gorm"
)

type DbBackupDAO struct {
	db *gorm.DB
}

func NewDbBackupDAO(db *gorm.DB) *DbBackupDAO {
	return &DbBackupDAO{db: db}
}

func (d *DbBackupDAO) Create(backup *model.DbBackup) error {
	return d.db.Create(backup).Error
}

func (d *DbBackupDAO) FindPage(page, pageSize int, startTime, endTime string) ([]model.DbBackup, int64, error) {
	var list []model.DbBackup
	var total int64

	query := d.db.Model(&model.DbBackup{})
	if startTime != "" {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime != "" {
		query = query.Where("created_at <= ?", endTime)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (d *DbBackupDAO) FindByID(id int64) (*model.DbBackup, error) {
	var backup model.DbBackup
	err := d.db.First(&backup, id).Error
	return &backup, err
}

func (d *DbBackupDAO) Delete(id int64) error {
	return d.db.Delete(&model.DbBackup{}, id).Error
}

func (d *DbBackupDAO) FindOlderThan(days int) ([]model.DbBackup, error) {
	var list []model.DbBackup
	err := d.db.Where("created_at < DATE_SUB(NOW(), INTERVAL ? DAY)", days).Find(&list).Error
	return list, err
}

func (d *DbBackupDAO) BatchDelete(ids []int64) error {
	return d.db.Delete(&model.DbBackup{}, ids).Error
}
