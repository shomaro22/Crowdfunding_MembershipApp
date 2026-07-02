package usecase

import (
	"context"

	"github.com/google/uuid"
)

// ITwoFactorUsecase は TOTP 二要素認証のポート。
type ITwoFactorUsecase interface {
	Setup2FA(ctx context.Context, userID uuid.UUID) (*TOTPSetupResult, error)
	Verify2FASetup(ctx context.Context, userID uuid.UUID, code string) (*TOTPVerifyResult, error)
	Challenge2FA(ctx context.Context, pendingToken, code string, meta RequestMeta) (*LoginResult, error)
	CreatePending2FAToken(ctx context.Context, userID uuid.UUID) (string, error)
}

// TwoFactorUseCase は 2FA を担当する。
type TwoFactorUseCase struct {
	deps *authDeps
}
