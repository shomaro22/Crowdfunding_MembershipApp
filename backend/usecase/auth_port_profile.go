package usecase

import (
	"context"

	"github.com/google/uuid"
)

// IProfileUsecase はプロフィール取得・更新のポート。
type IProfileUsecase interface {
	GetUser(ctx context.Context, userID uuid.UUID) (UserView, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, in UpdateProfileIn) (UserView, error)
}

// ProfileUseCase はユーザープロフィールを担当する。
type ProfileUseCase struct {
	deps *authDeps
}
