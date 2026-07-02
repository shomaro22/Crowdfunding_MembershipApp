package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"crowdfund-app/model"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// OAuthStateData は OAuth state に紐づくメタデータ（PKCE・リプレイ防止）。
type OAuthStateData struct {
	From         string `json:"from"`
	CodeVerifier string `json:"code_verifier,omitempty"`
	IP           string `json:"ip,omitempty"`
	UserAgent    string `json:"ua,omitempty"`
	Nonce        string `json:"nonce,omitempty"`
}

type IOAuthRepo interface {
	Create(ctx context.Context, row *model.OAuthAccount) error
	GetSub(ctx context.Context, provider model.OAuthProvider, sub string) (model.OAuthAccount, error)
	GetUserID(ctx context.Context, userID uuid.UUID, provider model.OAuthProvider) (model.OAuthAccount, error)
}

type oauthRepo struct {
	db *gorm.DB
}

func NewOAuthRepo(db *gorm.DB) IOAuthRepo {
	return &oauthRepo{db: db}
}

func (r *oauthRepo) Create(ctx context.Context, row *model.OAuthAccount) error {
	return dbFromCtx(ctx, r.db).Create(row).Error
}

func (r *oauthRepo) GetSub(ctx context.Context, provider model.OAuthProvider, sub string) (model.OAuthAccount, error) {
	return getOne[model.OAuthAccount](
		dbFromCtx(ctx, r.db).Where("provider = ? AND provider_sub = ?", provider, sub),
	)
}

func (r *oauthRepo) GetUserID(ctx context.Context, userID uuid.UUID, provider model.OAuthProvider) (model.OAuthAccount, error) {
	return getOne[model.OAuthAccount](
		dbFromCtx(ctx, r.db).Where("user_id = ? AND provider = ?", userID, provider),
	)
}

type IOAuthCache interface {
	SetState(ctx context.Context, state, from string, ttl time.Duration) error
	PopState(ctx context.Context, state string) (from string, ok bool, err error)
	SetOAuthState(ctx context.Context, state string, data OAuthStateData, ttl time.Duration) error
	PopOAuthState(ctx context.Context, state string) (OAuthStateData, bool, error)
	SetCallbackCode(ctx context.Context, code string, userID uuid.UUID, ttl time.Duration) error
	PopCallback(ctx context.Context, code string) (uuid.UUID, bool, error)
}

type redisOAuthCache struct {
	rdb *redis.Client
}

func NewOAuthCache(rdb *redis.Client) IOAuthCache {
	return &redisOAuthCache{rdb: rdb}
}

func (c *redisOAuthCache) SetState(ctx context.Context, state, from string, ttl time.Duration) error {
	return c.SetOAuthState(ctx, state, OAuthStateData{From: from}, ttl)
}

func (c *redisOAuthCache) SetOAuthState(ctx context.Context, state string, data OAuthStateData, ttl time.Duration) error {
	var (
		raw []byte
		err error
	)
	if c.rdb == nil {
		return fmt.Errorf("redis unavailable")
	}
	raw, err = json.Marshal(data)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, "oauth:state:"+state, raw, ttl).Err()
}

func (c *redisOAuthCache) PopOAuthState(ctx context.Context, state string) (OAuthStateData, bool, error) {
	var (
		key  string
		raw  string
		data OAuthStateData
		err  error
	)
	if c.rdb == nil {
		return OAuthStateData{}, false, fmt.Errorf("redis unavailable")
	}
	key = "oauth:state:" + state
	raw, err = c.rdb.GetDel(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return OAuthStateData{}, false, nil
	}
	if err != nil {
		return OAuthStateData{}, false, err
	}
	if err = json.Unmarshal([]byte(raw), &data); err != nil {
		// 旧形式（from のみ）との互換
		return OAuthStateData{From: raw}, true, nil
	}
	return data, true, nil
}

func (c *redisOAuthCache) PopState(ctx context.Context, state string) (string, bool, error) {
	var (
		data OAuthStateData
		ok   bool
		err  error
	)
	data, ok, err = c.PopOAuthState(ctx, state)
	if err != nil || !ok {
		return "", ok, err
	}
	return data.From, true, nil
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
