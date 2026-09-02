package dao

import (
	"ginproject/internal/model"

	"gorm.io/gorm"
)

type MenuDAO struct{ db *gorm.DB }

func NewMenuDAO(db *gorm.DB) *MenuDAO { return &MenuDAO{db} }

func (d *MenuDAO) Create(m *model.Menu) error {
	return d.db.Create(m).Error
}

func (d *MenuDAO) Update(m *model.Menu) error {
	return d.db.Omit("created_at").Save(m).Error
}

func (d *MenuDAO) Delete(id uint) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		// 先清除菜单与角色的关联关系，避免外键约束报错
		if err := tx.Table("role_menus").Where("menu_id = ?", id).Delete(nil).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Menu{}, id).Error
	})
}

func (d *MenuDAO) FindByID(id uint) (*model.Menu, error) {
	var m model.Menu
	err := d.db.First(&m, id).Error
	return &m, err
}

func (d *MenuDAO) HasChildren(parentID uint) (bool, error) {
	var count int64
	err := d.db.Model(&model.Menu{}).Where("parent_id = ?", parentID).Count(&count).Error
	return count > 0, err
}

func (d *MenuDAO) FindAll() ([]model.Menu, error) {
	var menus []model.Menu
	err := d.db.Where("status = 1").Order("sort ASC").Find(&menus).Error
	return menus, err
}

func (d *MenuDAO) FindByRoleIDs(roleIDs []uint) ([]model.Menu, error) {
	var menus []model.Menu
	err := d.db.Distinct("menus.*").
		Joins("JOIN role_menus ON role_menus.menu_id = menus.id").
		Where("role_menus.role_id IN ? AND menus.status = 1", roleIDs).
		Order("menus.sort ASC").
		Find(&menus).Error
	return menus, err
}

func BuildMenuTree(menus []model.Menu, parentID uint) []model.Menu {
	var tree []model.Menu
	for _, m := range menus {
		if m.ParentID == parentID {
			m.Children = BuildMenuTree(menus, m.ID)
			tree = append(tree, m)
		}
	}
	return tree
}
