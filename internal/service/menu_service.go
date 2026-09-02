package service

import (
	"errors"
	"fmt"
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
	count, err := s.menuDAO.CountRoles(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("该菜单已分配给 %d 个角色，请先移除关联再删除", count)
	}
	return s.menuDAO.Delete(id)
}

func (s *MenuService) FindByID(id uint) (*model.Menu, error) { return s.menuDAO.FindByID(id) }

func (s *MenuService) BatchUpdateSort(items []model.MenuItemSort) error {
	return s.menuDAO.BatchUpdateSort(items)
}

func (s *MenuService) GetTree() ([]model.Menu, error) {
	menus, err := s.menuDAO.FindAll()
	if err != nil {
		return nil, err
	}
	return dao.BuildMenuTree(menus, 0), nil
}
