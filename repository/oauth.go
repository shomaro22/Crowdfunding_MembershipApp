package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type OAuthRepo interface {
	Create(ctx context.Context, row *entity.OAuthAccount) error
	FindSub(ctx context.Context, provider entity.OAuthProvider, sub string) (*entity.OAuthAccount, error)
	FindUserID(ctx context.Context, userID uuid.UUID, provider entity.OAuthProvider) (*entity.OAuthAccount, error)
}

type oauthRepo struct {
	db *gorm.DB
}

func NewOAuthRepo(db *gorm.DB) OAuthRepo {
	return &oauthRepo{db: db}
}

func (r *oauthRepo) Create(ctx context.Context, row *entity.OAuthAccount) error {
	return dbFromCtx(ctx, r.db).Create(row).Error
}

func (r *oauthRepo) FindSub(ctx context.Context, provider entity.OAuthProvider, sub string) (*entity.OAuthAccount, error) {
	var (
		m   entity.OAuthAccount
		err error
	)
	err = dbFromCtx(ctx, r.db).Where("provider = ? AND provider_sub = ?", provider, sub).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *oauthRepo) FindUserID(ctx context.Context, userID uuid.UUID, provider entity.OAuthProvider) (*entity.OAuthAccount, error) {
	var (
		m   entity.OAuthAccount
		err error
	)
	err = dbFromCtx(ctx, r.db).Where("user_id = ? AND provider = ?", userID, provider).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

type OAuthCache interface {
	SetState(ctx context.Context, state, from string, ttl time.Duration) error
	PopState(ctx context.Context, state string) (from string, ok bool, err error)
	SetCallbackCode(ctx context.Context, code string, userID uuid.UUID, ttl time.Duration) error
	PopCallback(ctx context.Context, code string) (uuid.UUID, bool, error)
}

type redisOAuthCache struct {
	rdb *redis.Client
}

func NewOAuthCache(rdb *redis.Client) OAuthCache {
	return &redisOAuthCache{rdb: rdb}
}

func (c *redisOAuthCache) SetState(ctx context.Context, state, from string, ttl time.Duration) error {
	if c.rdb == nil {
		return fmt.Errorf("redis unavailable")
	}
	return c.rdb.Set(ctx, "oauth:state:"+state, from, ttl).Err()
}

func (c *redisOAuthCache) PopState(ctx context.Context, state string) (string, bool, error) {
	var (
		key  string
		from string
		err  error
	)
	if c.rdb == nil {
		return "", false, fmt.Errorf("redis unavailable")
	}
	key = "oauth:state:" + state
	from, err = c.rdb.GetDel(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return from, true, nil
}

func (c *redisOAuthCache) SetCallbackCode(ctx context.Context, code string, userID uuid.UUID, ttl time.Duration) error {
	if c.rdb == nil {
		return fmt.Errorf("redis unavailable")
	}
	return c.rdb.Set(ctx, "oauth:cb:"+code, userID.String(), ttl).Err()
}

func (c *redisOAuthCache) PopCallback(ctx context.Context, code string) (uuid.UUID, bool, error) {
	var (
		raw string
		id  uuid.UUID
		err error
	)
	if c.rdb == nil {
		return uuid.Nil, false, fmt.Errorf("redis unavailable")
	}
	raw, err = c.rdb.GetDel(ctx, "oauth:cb:"+code).Result()
	if errors.Is(err, redis.Nil) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, err
	}
	id, err = uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, false, nil
	}
	return id, true, nil
}
