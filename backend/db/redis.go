package db

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func OpenRedis() *redis.Client {
	var (
		url     string
		addr    string
		dbNum   int
		s       string
		n       int
		opts    *redis.Options
		rdb     *redis.Client
		ctxPing context.Context
		cancel  context.CancelFunc
		err     error
	)
	url = strings.TrimSpace(os.Getenv("REDIS_URL"))
	if url != "" {
		opts, err = redis.ParseURL(url)
		if err != nil {
			return nil
		}
		rdb = redis.NewClient(opts)
		ctxPing, cancel = context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = rdb.Ping(ctxPing).Err()
		return rdb
	}
	addr = strings.TrimSpace(os.Getenv("REDIS_ADDR"))
	if addr == "" {
		return nil
	}
	dbNum = 0
	s = strings.TrimSpace(os.Getenv("REDIS_DB"))
	if s != "" {
		n, err = strconv.Atoi(s)
		if err == nil {
			dbNum = n
		}
	}
	rdb = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       dbNum,
	})
	ctxPing, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = rdb.Ping(ctxPing).Err()
	return rdb
}

func PingRedis(ctx context.Context, rdb *redis.Client) error {
	if rdb == nil {
		return nil
	}
	return rdb.Ping(ctx).Err()
}
