package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimitRepo interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (allowed bool, retryAfterSec int, err error)
}

type redisRateLimitRepo struct {
	rdb *redis.Client
}

func NewRateLimitRepo(rdb *redis.Client) RateLimitRepo {
	return &redisRateLimitRepo{rdb: rdb}
}

func (r *redisRateLimitRepo) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int, error) {
	var (
		fullKey string
		n       int64
		ttl     time.Duration
		sec     int
		err     error
	)
	if r.rdb == nil {
		return true, 0, nil
	}
	fullKey = "rl:" + key
	n, err = r.rdb.Incr(ctx, fullKey).Result()
	if err != nil {
		return true, 0, err
	}
	if n == 1 {
		_ = r.rdb.Expire(ctx, fullKey, window).Err()
	}
	if n > int64(limit) {
		ttl, _ = r.rdb.TTL(ctx, fullKey).Result()
		sec = int(ttl.Seconds())
		if sec < 1 {
			sec = 1
		}
		return false, sec, nil
	}
	return true, 0, nil
}
