package usecase

import (
	"context"

	"github.com/google/uuid"
)

// ISessionAuthUsecase はログイン・セッション管理のポート。
type ISessionAuthUsecase interface {
	Login(ctx context.Context, email, password string, meta RequestMeta) (*LoginResult, error)
	Refresh(ctx context.Context, refreshTokenPlain string, meta RequestMeta) (authToken, refreshToken string, err error)
	Logout(ctx context.Context, refreshTokenPlain string, meta RequestMeta) error
	LogoutAllDevices(ctx context.Context, userID uuid.UUID, meta RequestMeta) error
	ParseToken(raw string) (AuthClaims, error)
	ValidateSession(ctx context.Context, claims AuthClaims) (SessionInfo, error)
}

// IRegistrationUsecase は購入者・出品者登録のポート。
type IRegistrationUsecase interface {
	RegisterBuyer(ctx context.Context, email, password string, name *string, meta RequestMeta) (UserView, error)
	RegisterSeller(ctx context.Context, in SellerRegIn, meta RequestMeta) (*SellerRegOut, error)
	EmailFree(ctx context.Context, email string) (bool, error)
}

// SessionAuthUseCase はセッション認証を担当する。
type SessionAuthUseCase struct {
	deps *authDeps
}

// RegistrationUseCase はアカウント登録を担当する。
type RegistrationUseCase struct {
	deps *authDeps
}
