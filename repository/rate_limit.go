package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimitRepo interface {
	Allow(ctx context.Context, key string, limit int, period time.Duration) (allowed bool, retryAfterSec int, err error)
}

type redisRateLimitRepo struct {
	rdb    *redis.Client
	script *redis.Script
}

func NewRateLimitRepo(rdb *redis.Client) RateLimitRepo {
	return &redisRateLimitRepo{
		rdb:    rdb,
		script: redis.NewScript(tokenBucketLua),
	}
}

func (r *redisRateLimitRepo) Allow(ctx context.Context, key string, limit int, period time.Duration) (bool, int, error) {
	var (
		fullKey   string
		periodSec int64
		now       time.Time
		res       interface{}
		vals      []int64
		ok        bool
		err       error
		allowed   bool
		retryAfter int
	)

	if r.rdb == nil {
		return true, 0, nil
	}
	if limit <= 0 {
		return true, 0, nil
	}
	if period <= 0 {
		period = time.Minute
	}
	fullKey = "rl:tb:" + key
	periodSec = int64(period.Seconds())
	if periodSec < 1 {
		periodSec = 1
	}
	now = time.Now().UTC()
	res, err = r.script.Run(ctx, r.rdb, []string{fullKey}, limit, periodSec, now.UnixMilli(), 1).Result()
	if err != nil {
		return true, 0, err
	}
	vals, ok = res.([]int64)
	if !ok || len(vals) < 2 {
		var (
			raw []interface{}
			i   int
			n   int64
		)
		raw, ok = res.([]interface{})
		if !ok || len(raw) < 2 {
			return true, 0, nil
		}
		vals = make([]int64, 2)
		for i = 0; i < 2; i++ {
			switch v := raw[i].(type) {
			case int64:
				n = v
			case int:
				n = int64(v)
			default:
				return true, 0, nil
			}
			vals[i] = n
		}
	}
	allowed = vals[0] == 1
	retryAfter = int(vals[1])
	if retryAfter < 0 {
		retryAfter = 0
	}
	return allowed, retryAfter, nil
}
