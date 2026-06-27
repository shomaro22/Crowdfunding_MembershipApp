package controller

import (
	"context"

	"crowdfund-app/auth"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
)

type AuthUseCase interface {
	RegisterBuyer(ctx context.Context, email, password string, name *string, meta usecase.RequestMeta) (usecase.UserView, error)
	RegisterSeller(ctx context.Context, in usecase.SellerRegIn, meta usecase.RequestMeta) (*usecase.SellerRegOut, error)
	Login(ctx context.Context, email, password string, meta usecase.RequestMeta) (*usecase.LoginResult, error)
	Refresh(ctx context.Context, refreshTokenPlain string, meta usecase.RequestMeta) (authToken, refreshToken string, err error)
	Logout(ctx context.Context, refreshTokenPlain string, meta usecase.RequestMeta) error
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, plainToken, newPassword string) error
	ChangePassword(ctx context.Context, userID uuid.UUID, current, newPassword, totpCode string) error
	GetUser(ctx context.Context, userID uuid.UUID) (usecase.UserView, error)
	EmailFree(ctx context.Context, email string) (bool, error)
	ParseToken(raw string) (auth.AuthClaims, error)
	Setup2FA(ctx context.Context, userID uuid.UUID) (*usecase.TOTPSetupResult, error)
	Verify2FASetup(ctx context.Context, userID uuid.UUID, code string) (*usecase.TOTPVerifyResult, error)
	Challenge2FA(ctx context.Context, pendingToken, code string, meta usecase.RequestMeta) (*usecase.LoginResult, error)
	GoogleAuthURL(from string) (string, error)
	GoogleCallback(ctx context.Context, code, state string, meta usecase.RequestMeta) (*usecase.OAuthCBOut, error)
	ExchangeOAuthCode(ctx context.Context, code string, meta usecase.RequestMeta) (*usecase.LoginResult, error)
	SigningKey() []byte
}
