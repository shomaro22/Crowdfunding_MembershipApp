package usecase

import (
	"context"
	"strings"
	"time"

	"crowdfund-app/auth"
	"crowdfund-app/entity"

	"github.com/google/uuid"
)

func (a *AuthUseCase) RegisterBuyer(ctx context.Context, email, password string, name *string, meta AuthMeta) (*entity.User, error) {
	var (
		normalized  string
		err         error
		exists      bool
		hash        string
		now         time.Time
		trimmedName *string
		n           string
		u           *entity.User
	)

	normalized, err = entity.ValidateBuyerReg(email, password)
	if err != nil {
		switch err {
		case entity.ErrBadEmail:
			return nil, ErrInvalidEmail
		case entity.ErrWeakPassword:
			return nil, ErrWeakPassword
		default:
			return nil, err
		}
	}
	exists, err = a.Users.EmailExists(ctx, normalized)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailInUse
	}
	hash, err = auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	now = time.Now().UTC()
	if name != nil {
		n = strings.TrimSpace(*name)
		if n != "" {
			trimmedName = &n
		}
	}
	u = &entity.User{
		ID:            uuid.New(),
		Email:         normalized,
		Name:          trimmedName,
		PasswordHash:  hash,
		Role:          entity.UserRoleBuyer,
		AccountStatus: entity.AccountStatusPending,
		SellerStatus:  entity.SellerStatusNotApplicable,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err = a.Users.Create(ctx, u); err != nil {
		return nil, err
	}
	a.logActivity(ctx, &u.ID, entity.UserRegister, "user", u.ID.String(), meta, nil)
	return u, nil
}

func (a *AuthUseCase) EmailFree(ctx context.Context, email string) (bool, error) {
	var (
		normalized string
		err        error
		exists     bool
	)

	normalized, err = entity.ValidateEmail(email)
	if err != nil {
		return false, ErrInvalidEmail
	}
	exists, err = a.Users.EmailExists(ctx, normalized)
	if err != nil {
		return false, err
	}
	return !exists, nil
}
