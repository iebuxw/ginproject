package captcha

import (
	"context"
	"time"

	"ginproject/internal/logger"

	"go.uber.org/zap"

	"github.com/redis/go-redis/v9"
)

// captchaTTL 验证码有效期
const captchaTTL = 2 * time.Minute

// RedisStore 实现 dchest/captcha.Store 接口，使用 Redis 存储验证码
type RedisStore struct {
	rdb *redis.Client
}

func NewRedisStore(rdb *redis.Client) *RedisStore {
	return &RedisStore{rdb: rdb}
}

// Set 将验证码答案存入 Redis
func (s *RedisStore) Set(id string, digits []byte) {
	key := "captcha:" + id
	if err := s.rdb.Set(context.Background(), key, string(digits), captchaTTL).Err(); err != nil {
		logger.Error("验证码存储失败", zap.Error(err))
	}
}

// Get 从 Redis 获取验证码答案；clear=true 时读后删除（一次性消耗）
func (s *RedisStore) Get(id string, clear bool) []byte {
	key := "captcha:" + id
	ctx := context.Background()
	val, err := s.rdb.Get(ctx, key).Result()
	if err != nil {
		return nil
	}
	if clear {
		s.rdb.Del(ctx, key)
	}
	return []byte(val)
}
