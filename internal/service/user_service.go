package service

import (
	"ginproject/internal/dao"
	"ginproject/internal/model"
	"ginproject/internal/utils"
)

type UserService struct{ userDAO *dao.UserDAO }

func NewUserService(userDAO *dao.UserDAO) *UserService { return &UserService{userDAO} }

func (s *UserService) Create(u *model.User) error {
	hashed, err := utils.HashPassword(u.Password)
	if err != nil {
		return err
	}
	u.Password = hashed
	return s.userDAO.Create(u)
}

func (s *UserService) Update(u *model.User) error {
	if u.Password != "" {
		hashed, err := utils.HashPassword(u.Password)
		if err != nil {
			return err
		}
		u.Password = hashed
	}
	return s.userDAO.Update(u)
}

func (s *UserService) Delete(id uint) error { return s.userDAO.Delete(id) }

func (s *UserService) FindByID(id uint) (*model.User, error) { return s.userDAO.FindByID(id) }

func (s *UserService) FindPage(page, pageSize int, keyword string) ([]model.User, int64, error) {
	return s.userDAO.FindPage(page, pageSize, keyword)
}
