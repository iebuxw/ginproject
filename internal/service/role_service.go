package service

import (
	"fmt"

	"ginproject/internal/dao"
	"ginproject/internal/model"

	"gorm.io/gorm"
)

type RoleService struct{ roleDAO *dao.RoleDAO }

func NewRoleService(roleDAO *dao.RoleDAO) *RoleService { return &RoleService{roleDAO} }

func (s *RoleService) Create(r *model.Role) error {
	// 创建前查重角色名
	existing, err := s.roleDAO.FindByName(r.Name)
	if err == nil && existing.ID > 0 {
		return fmt.Errorf("角色名「%s」已存在", r.Name)
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return s.roleDAO.Create(r)
}

func (s *RoleService) Update(r *model.Role) error {
	// 编辑时查重角色名（排除自身）
	existing, err := s.roleDAO.FindByName(r.Name)
	if err == nil && existing.ID > 0 && existing.ID != r.ID {
		return fmt.Errorf("角色名「%s」已存在", r.Name)
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return s.roleDAO.Update(r)
}

func (s *RoleService) Delete(id uint) error { return s.roleDAO.Delete(id) }

func (s *RoleService) FindByID(id uint) (*model.Role, error) { return s.roleDAO.FindByID(id) }

func (s *RoleService) FindPage(page, pageSize int, keyword string) ([]model.Role, int64, error) {
	return s.roleDAO.FindPage(page, pageSize, keyword)
}

func (s *RoleService) FindBatch(keyword string, offset, limit int) ([]model.Role, error) {
	return s.roleDAO.FindBatch(keyword, offset, limit)
}

func (s *RoleService) FindAll() ([]model.Role, error) { return s.roleDAO.FindAll() }
