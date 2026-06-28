package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const tokenBucketLua = `
local capacity = tonumber(ARGV[1])
local period_sec = tonumber(ARGV[2])
local now_ms = tonumber(ARGV[3])
local cost = tonumber(ARGV[4])
local refill_per_ms = capacity / (period_sec * 1000)

local data = redis.call('HMGET', KEYS[1], 'tokens', 'ts')
local tokens = tonumber(data[1])
local ts = tonumber(data[2])

if tokens == nil then
  tokens = capacity
  ts = now_ms
end

local elapsed = now_ms - ts
if elapsed > 0 then
  tokens = math.min(capacity, tokens + elapsed * refill_per_ms)
  ts = now_ms
end

if tokens < cost then
  local deficit = cost - tokens
  local wait_ms = deficit / refill_per_ms
  local retry_sec = math.ceil(wait_ms / 1000)
  if retry_sec < 1 then
    retry_sec = 1
  end
  return {0, retry_sec}
end

tokens = tokens - cost
redis.call('HMSET', KEYS[1], 'tokens', tokens, 'ts', ts)
redis.call('EXPIRE', KEYS[1], period_sec * 2)
return {1, 0}
`

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
