package usecase

import "context"

// IOAuthUsecase は Google OAuth のポート。
type IOAuthUsecase interface {
	GoogleAuthURL(from string, meta RequestMeta) (string, error)
	GoogleCallback(ctx context.Context, code, state string, meta RequestMeta) (*OAuthCBOut, error)
	ExchangeOAuthCode(ctx context.Context, code string, meta RequestMeta) (*LoginResult, error)
	ConfirmOAuthLink(ctx context.Context, linkCode string, meta RequestMeta) (*LoginResult, error)
	OAuthEnabled() bool
}

// OAuthUseCase は OAuth 連携を担当する。
type OAuthUseCase struct {
	deps *authDeps
}
