package dao

import (
	"ginproject/internal/model"

	"gorm.io/gorm"
)

type RoleDAO struct{ db *gorm.DB }

func NewRoleDAO(db *gorm.DB) *RoleDAO { return &RoleDAO{db} }

func (d *RoleDAO) Create(r *model.Role) error {
	return d.db.Create(r).Error
}

func (d *RoleDAO) Update(r *model.Role) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(r).Association("Menus").Replace(r.Menus); err != nil {
			return err
		}
		return tx.Omit("created_at").Save(r).Error
	})
}

func (d *RoleDAO) Delete(id uint) error {
	return d.db.Delete(&model.Role{}, id).Error
}

func (d *RoleDAO) FindByID(id uint) (*model.Role, error) {
	var r model.Role
	err := d.db.Preload("Menus").First(&r, id).Error
	return &r, err
}

func (d *RoleDAO) FindPage(page, pageSize int, keyword string) ([]model.Role, int64, error) {
	var roles []model.Role
	var total int64
	q := d.db.Model(&model.Role{})
	if keyword != "" {
		q = q.Where("name LIKE ? OR code LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	q.Count(&total)
	err := q.Offset((page - 1) * pageSize).Limit(pageSize).Order("id DESC").Find(&roles).Error
	return roles, total, err
}

func (d *RoleDAO) FindAll() ([]model.Role, error) {
	var roles []model.Role
	err := d.db.Where("status = 1").Find(&roles).Error
	return roles, err
}
