package usecase

import (
	"context"

	"crowdfund-app/auth"
	"crowdfund-app/entity"

	"github.com/google/uuid"
)

func (a *AuthUseCase) GetUser(ctx context.Context, userID uuid.UUID) (*entity.User, error) {
	var (
		u   *entity.User
		err error
	)

	u, err = a.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrNotFound
	}
	return u, nil
}

func (a *AuthUseCase) ParseToken(raw string) (auth.AuthClaims, error) {
	return a.Tokens.Parse(raw)
}

func (a *AuthUseCase) SigningKey() []byte {
	return a.TOTPKey
}
