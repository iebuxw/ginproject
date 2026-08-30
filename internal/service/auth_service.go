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

// ErrAccountLocked 账号因连续登录失败被临时锁定，剩余等待分钟数由 controller 拼入提示文案
var ErrAccountLocked = errors.New("账号已锁定")

func NewAuthService(userDAO *dao.UserDAO, rdb *redis.Client, cfg *config.Config) *AuthService {
	return &AuthService{userDAO, rdb, cfg}
}

func (s *AuthService) Login(username, password string) (string, *model.User, error) {
	if locked, remain := s.isLocked(username); locked {
		return "", nil, fmt.Errorf("%w，请 %d 分钟后再试", ErrAccountLocked, remain)
	}
	user, err := s.userDAO.FindByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.recordFailure(username)
			return "", nil, ErrInvalidCredentials
		}
		return "", nil, err
	}
	if user.Status != 1 {
		return "", nil, errors.New("账号已被禁用")
	}
	if !utils.CheckPassword(password, user.Password) {
		s.recordFailure(username)
		return "", nil, ErrInvalidCredentials
	}
	s.clearFailures(username)
	token, err := utils.GenerateToken(user.ID, user.Username, s.cfg.JWT.Secret, s.cfg.JWT.ExpireHours)
	if err != nil {
		return "", nil, err
	}
	now := time.Now()
	user.LastLoginAt = &now
	_ = s.userDAO.Update(user)
	return token, user, nil
}

// loginFailKey 用户名维度的失败计数 key，TTL 即累计窗口与锁定时长
func loginFailKey(username string) string {
	return fmt.Sprintf("login_fail:%s", username)
}

// isLocked 失败计数已达阈值即锁定；Redis 异常时放行（降级不阻断登录）
func (s *AuthService) isLocked(username string) (bool, int) {
	if s.cfg.LoginLock.MaxAttempts <= 0 {
		return false, 0
	}
	val, err := s.rdb.Get(context.Background(), loginFailKey(username)).Int()
	if err != nil {
		return false, 0
	}
	if val < s.cfg.LoginLock.MaxAttempts {
		return false, 0
	}
	ttl, err := s.rdb.TTL(context.Background(), loginFailKey(username)).Result()
	if err != nil || ttl <= 0 {
		return true, 1
	}
	// 剩余分钟数向上取整，避免显示"0分钟后"
	return true, int((ttl + time.Minute - 1) / time.Minute)
}

// recordFailure 记一次失败：首次失败设累计窗口，达阈值刷新 TTL 为完整锁定时长
func (s *AuthService) recordFailure(username string) {
	if s.cfg.LoginLock.MaxAttempts <= 0 || s.cfg.LoginLock.DurationMinutes <= 0 {
		return
	}
	key := loginFailKey(username)
	ctx := context.Background()
	val, err := s.rdb.Incr(ctx, key).Result()
	if err != nil {
		return
	}
	window := time.Duration(s.cfg.LoginLock.DurationMinutes) * time.Minute
	if val == int64(s.cfg.LoginLock.MaxAttempts) {
		// 达到阈值：刷新 TTL 为完整锁定时长，此刻起锁定开始计时
		s.rdb.Expire(ctx, key, window)
	} else if val == 1 {
		s.rdb.Expire(ctx, key, window)
	}
}

// clearFailures 登录成功清零计数
func (s *AuthService) clearFailures(username string) {
	s.rdb.Del(context.Background(), loginFailKey(username))
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
