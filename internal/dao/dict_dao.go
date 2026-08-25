package dao

import (
	"ginproject/internal/model"

	"gorm.io/gorm"
)

// ---- DictType DAO ----

type DictTypeDAO struct{ db *gorm.DB }

func NewDictTypeDAO(db *gorm.DB) *DictTypeDAO { return &DictTypeDAO{db: db} }

func (d *DictTypeDAO) Create(dt *model.DictType) error {
	return d.db.Create(dt).Error
}

func (d *DictTypeDAO) Update(dt *model.DictType) error {
	return d.db.Omit("created_at").Save(dt).Error
}

func (d *DictTypeDAO) Delete(id uint) error {
	return d.db.Delete(&model.DictType{}, id).Error
}

func (d *DictTypeDAO) FindByID(id uint) (*model.DictType, error) {
	var dt model.DictType
	err := d.db.First(&dt, id).Error
	return &dt, err
}

func (d *DictTypeDAO) FindByCode(code string) (*model.DictType, error) {
	var dt model.DictType
	err := d.db.Where("code = ?", code).First(&dt).Error
	return &dt, err
}

func (d *DictTypeDAO) FindPage(page, pageSize int, keyword string) ([]model.DictType, int64, error) {
	var list []model.DictType
	var total int64
	q := d.db.Model(&model.DictType{})
	if keyword != "" {
		q = q.Where("name LIKE ? OR code LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	q.Count(&total)
	err := q.Offset((page - 1) * pageSize).Limit(pageSize).Order("id DESC").Find(&list).Error
	return list, total, err
}

// ---- DictData DAO ----

type DictDataDAO struct{ db *gorm.DB }

func NewDictDataDAO(db *gorm.DB) *DictDataDAO { return &DictDataDAO{db: db} }

func (d *DictDataDAO) Create(dd *model.DictData) error {
	return d.db.Create(dd).Error
}

func (d *DictDataDAO) Update(dd *model.DictData) error {
	return d.db.Omit("created_at").Save(dd).Error
}

func (d *DictDataDAO) Delete(id uint) error {
	return d.db.Delete(&model.DictData{}, id).Error
}

func (d *DictDataDAO) FindByID(id uint) (*model.DictData, error) {
	var dd model.DictData
	err := d.db.First(&dd, id).Error
	return &dd, err
}

func (d *DictDataDAO) FindByDictTypeID(dictTypeID uint) ([]model.DictData, error) {
	var list []model.DictData
	err := d.db.Where("dict_type_id = ? AND status = 1", dictTypeID).Order("sort ASC").Find(&list).Error
	return list, err
}

func (d *DictDataDAO) FindPage(page, pageSize int, dictTypeID uint) ([]model.DictData, int64, error) {
	var list []model.DictData
	var total int64
	q := d.db.Model(&model.DictData{})
	if dictTypeID > 0 {
		q = q.Where("dict_type_id = ?", dictTypeID)
	}
	q.Count(&total)
	err := q.Offset((page - 1) * pageSize).Limit(pageSize).Order("sort ASC").Find(&list).Error
	return list, total, err
}

func (d *DictDataDAO) HasData(dictTypeID uint) (bool, error) {
	var count int64
	err := d.db.Model(&model.DictData{}).Where("dict_type_id = ?", dictTypeID).Count(&count).Error
	return count > 0, err
}

func (d *DictDataDAO) DeleteByDictTypeID(dictTypeID uint) error {
	return d.db.Where("dict_type_id = ?", dictTypeID).Delete(&model.DictData{}).Error
}

func (d *DictDataDAO) FindByDictTypeCode(code string) ([]model.DictData, error) {
	var list []model.DictData
	err := d.db.Table("dict_data").
		Joins("JOIN dict_types ON dict_types.id = dict_data.dict_type_id").
		Where("dict_types.code = ? AND dict_data.status = 1 AND dict_types.status = 1", code).
		Order("dict_data.sort ASC").
		Find(&list).Error
	return list, err
}
