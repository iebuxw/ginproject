package dao

import (
	"ginproject/internal/model"
	"gorm.io/gorm"
)

type FileDAO struct {
	db *gorm.DB
}

func NewFileDAO(db *gorm.DB) *FileDAO {
	return &FileDAO{db: db}
}

func (d *FileDAO) Create(file *model.File) error {
	return d.db.Create(file).Error
}

func (d *FileDAO) FindPage(page, pageSize int, name string) ([]model.File, int64, error) {
	var list []model.File
	var total int64

	query := d.db.Model(&model.File{})
	if name != "" {
		query = query.Where("original_name LIKE ?", "%"+name+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (d *FileDAO) FindByID(id int64) (*model.File, error) {
	var file model.File
	err := d.db.First(&file, id).Error
	return &file, err
}

func (d *FileDAO) Delete(id int64) error {
	return d.db.Delete(&model.File{}, id).Error
}
