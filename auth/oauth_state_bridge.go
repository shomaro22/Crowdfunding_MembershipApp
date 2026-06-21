package auth

import (
	"context"
	"time"

	"crowdfund-app/repository"
)

type oauthStateBridge struct {
	cache repository.OAuthCache
	ttl   time.Duration
}

func NewStateBridge(cache repository.OAuthCache) OAuthStateStore {
	return &oauthStateBridge{cache: cache, ttl: 10 * time.Minute}
}

func (b *oauthStateBridge) SetState(ctx context.Context, state, from string) error {
	if b.cache == nil {
		return ErrInvalidToken
	}
	return b.cache.SetState(ctx, state, from, b.ttl)
}

func (b *oauthStateBridge) PopState(ctx context.Context, state string) (string, bool, error) {
	if b.cache == nil {
		return "", false, ErrInvalidToken
	}
	return b.cache.PopState(ctx, state)
}
