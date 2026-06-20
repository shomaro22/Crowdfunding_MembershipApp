package controller

import (
	"context"

	"crowdfund-app/auth"
	"crowdfund-app/entity"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
)

type AuthUseCase interface {
	RegisterBuyer(ctx context.Context, email, password string, name *string, meta usecase.AuthMeta) (*entity.User, error)
	RegisterSeller(ctx context.Context, in usecase.SellerRegIn, meta usecase.AuthMeta) (*usecase.SellerRegOut, error)
	Login(ctx context.Context, email, password string, meta usecase.AuthMeta) (*usecase.LoginResult, error)
	Reissue(ctx context.Context, reTokenPlain string, meta usecase.AuthMeta) (authToken, reToken string, err error)
	Logout(ctx context.Context, reTokenPlain string, meta usecase.AuthMeta) error
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, plainToken, newPassword string) error
	ChangePassword(ctx context.Context, userID uuid.UUID, current, newPassword string) error
	GetUser(ctx context.Context, userID uuid.UUID) (*entity.User, error)
	EmailFree(ctx context.Context, email string) (bool, error)
	ParseToken(raw string) (auth.AuthClaims, error)
}
