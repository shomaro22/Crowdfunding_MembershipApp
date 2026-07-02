package usecase

import (
	"context"

	"github.com/google/uuid"
)

// IPasswordUsecase はパスワードリセット・変更のポート。
type IPasswordUsecase interface {
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, plainToken, newPassword string) error
	ChangePassword(ctx context.Context, userID uuid.UUID, current, newPassword, totpCode string) error
}

// PasswordUseCase はパスワード管理を担当する。
type PasswordUseCase struct {
	deps *authDeps
}
