package service

import (
	"context"
	"errors"
	"fmt"
	"ginproject/internal/config"
	"ginproject/internal/dao"
	"ginproject/internal/model"
	"ginproject/internal/utils"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type AuthService struct {
	userDAO *dao.UserDAO
	rdb     *redis.Client
	cfg     *config.Config
}

// ErrInvalidCredentials 用户名或密码错误，供 controller 精确判断登录失败场景
var ErrInvalidCredentials = errors.New("用户名或密码错误")

func NewAuthService(userDAO *dao.UserDAO, rdb *redis.Client, cfg *config.Config) *AuthService {
	return &AuthService{userDAO, rdb, cfg}
}

func (s *AuthService) Login(username, password string) (string, *model.User, error) {
	user, err := s.userDAO.FindByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, ErrInvalidCredentials
		}
		return "", nil, err
	}
	if user.Status != 1 {
		return "", nil, errors.New("账号已被禁用")
	}
	if !utils.CheckPassword(password, user.Password) {
		return "", nil, ErrInvalidCredentials
	}
	token, err := utils.GenerateToken(user.ID, user.Username, s.cfg.JWT.Secret, s.cfg.JWT.ExpireHours)
	if err != nil {
		return "", nil, err
	}
	now := time.Now()
	user.LastLoginAt = &now
	_ = s.userDAO.Update(user)
	return token, user, nil
}

func (s *AuthService) Logout(token string) error {
	claims, err := utils.ParseToken(token, s.cfg.JWT.Secret)
	if err != nil {
		return nil
	}
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl > 0 {
		key := fmt.Sprintf("blacklist:%s", token)
		return s.rdb.Set(context.Background(), key, "1", ttl).Err()
	}
	return nil
}

func (s *AuthService) ChangePassword(userID uint, oldPassword, newPassword string) error {
	user, err := s.userDAO.FindByID(userID)
	if err != nil {
		return errors.New("用户不存在")
	}
	if !utils.CheckPassword(oldPassword, user.Password) {
		return errors.New("原密码错误")
	}
	hashed, err := utils.HashPassword(newPassword)
	if err != nil {
		return err
	}
	user.Password = hashed
	return s.userDAO.Update(user)
}

func (s *AuthService) UpdateProfile(userID uint, email string) error {
	user, err := s.userDAO.FindByID(userID)
	if err != nil {
		return errors.New("用户不存在")
	}
	user.Email = email
	return s.userDAO.Update(user)
}

func (s *AuthService) IsBlacklisted(token string) bool {
	key := fmt.Sprintf("blacklist:%s", token)
	_, err := s.rdb.Get(context.Background(), key).Result()
	return err == nil
}
