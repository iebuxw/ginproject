package dao

import (
	"ginproject/internal/model"

	"gorm.io/gorm"
)

type UserDAO struct{ db *gorm.DB }

func NewUserDAO(db *gorm.DB) *UserDAO { return &UserDAO{db} }

func (d *UserDAO) Create(u *model.User) error {
	return d.db.Create(u).Error
}

func (d *UserDAO) Update(u *model.User) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(u).Association("Roles").Replace(u.Roles); err != nil {
			return err
		}
		q := tx.Omit("created_at")
		// 密码为空时表示不修改密码，避免全字段覆盖把密码清空
		if u.Password == "" {
			q = q.Omit("password")
		}
		return q.Save(u).Error
	})
}

func (d *UserDAO) Delete(id uint) error {
	return d.db.Delete(&model.User{}, id).Error
}

func (d *UserDAO) FindByID(id uint) (*model.User, error) {
	var u model.User
	err := d.db.Preload("Roles").First(&u, id).Error
	return &u, err
}

func (d *UserDAO) FindByUsername(username string) (*model.User, error) {
	var u model.User
	err := d.db.Where("username = ?", username).Preload("Roles").First(&u).Error
	return &u, err
}

func (d *UserDAO) FindPage(page, pageSize int, keyword string) ([]model.User, int64, error) {
	var users []model.User
	var total int64
	q := d.db.Model(&model.User{})
	if keyword != "" {
		q = q.Where("username LIKE ? OR email LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	q.Count(&total)
	err := q.Preload("Roles").Offset((page - 1) * pageSize).Limit(pageSize).Order("id DESC").Find(&users).Error
	return users, total, err
}
