package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestTokenBucketAllow(t *testing.T) {
	var (
		mr   *miniredis.Miniredis
		rdb  *redis.Client
		repo RateLimitRepo
		ctx  context.Context
		ok   bool
		i    int
		err  error
	)

	mr, err = miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()
	rdb = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()
	repo = NewRateLimitRepo(rdb)
	ctx = context.Background()

	for i = 0; i < 5; i++ {
		ok, _, err = repo.Allow(ctx, "login:1.2.3.4", 5, time.Minute)
		if err != nil || !ok {
			t.Fatalf("request %d: ok=%v err=%v", i+1, ok, err)
		}
	}
	ok, _, err = repo.Allow(ctx, "login:1.2.3.4", 5, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected 6th request to be rate limited")
	}
}

func TestTokenBucketFailOpenNilRedis(t *testing.T) {
	var (
		repo RateLimitRepo
		ok   bool
		err  error
	)

	repo = NewRateLimitRepo(nil)
	ok, _, err = repo.Allow(context.Background(), "k", 5, time.Minute)
	if err != nil || !ok {
		t.Fatalf("nil redis should fail-open: ok=%v err=%v", ok, err)
	}
}
