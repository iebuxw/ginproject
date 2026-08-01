package service

import (
	"ginproject/internal/dao"
	"ginproject/internal/model"
)

type RoleService struct{ roleDAO *dao.RoleDAO }

func NewRoleService(roleDAO *dao.RoleDAO) *RoleService { return &RoleService{roleDAO} }

func (s *RoleService) Create(r *model.Role) error { return s.roleDAO.Create(r) }

func (s *RoleService) Update(r *model.Role) error { return s.roleDAO.Update(r) }

func (s *RoleService) Delete(id uint) error { return s.roleDAO.Delete(id) }

func (s *RoleService) FindByID(id uint) (*model.Role, error) { return s.roleDAO.FindByID(id) }

func (s *RoleService) FindPage(page, pageSize int, keyword string) ([]model.Role, int64, error) {
	return s.roleDAO.FindPage(page, pageSize, keyword)
}

func (s *RoleService) FindAll() ([]model.Role, error) { return s.roleDAO.FindAll() }
