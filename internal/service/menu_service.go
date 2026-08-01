package service

import (
	"errors"
	"ginproject/internal/dao"
	"ginproject/internal/model"
)

type MenuService struct{ menuDAO *dao.MenuDAO }

func NewMenuService(menuDAO *dao.MenuDAO) *MenuService { return &MenuService{menuDAO} }

func (s *MenuService) Create(m *model.Menu) error { return s.menuDAO.Create(m) }

func (s *MenuService) Update(m *model.Menu) error { return s.menuDAO.Update(m) }

func (s *MenuService) Delete(id uint) error {
	has, err := s.menuDAO.HasChildren(id)
	if err != nil {
		return err
	}
	if has {
		return errors.New("该菜单下有子菜单，无法删除")
	}
	return s.menuDAO.Delete(id)
}

func (s *MenuService) FindByID(id uint) (*model.Menu, error) { return s.menuDAO.FindByID(id) }

func (s *MenuService) GetTree() ([]model.Menu, error) {
	menus, err := s.menuDAO.FindAll()
	if err != nil {
		return nil, err
	}
	return dao.BuildMenuTree(menus, 0), nil
}
