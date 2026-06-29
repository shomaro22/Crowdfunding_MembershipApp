package usecase

import (
	"context"
	"time"

	"crowdfund-app/repository"
)

type RateLimitUseCase struct {
	Repo repository.IRateLimitRepo
}

func (u *RateLimitUseCase) Allow(ctx context.Context, key string, limit int, period time.Duration) (bool, int, error) {
	if u == nil || u.Repo == nil {
		return true, 0, nil
	}
	return u.Repo.Allow(ctx, key, limit, period)
}
