package usecase

import (
	"context"

	"crowdfund-app/auth"
	"crowdfund-app/entity"
)

type SessionInfo struct {
	Role      entity.UserRole
	AdminRole *entity.AdminRole
}

func (a *AuthUseCase) ValidateSession(ctx context.Context, claims auth.AuthClaims) (SessionInfo, error) {
	var (
		u   *entity.User
		err error
	)
	u, err = a.Users.FindID(ctx, claims.UserID)
	if err != nil {
		return SessionInfo{}, err
	}
	if u == nil || !u.IsActive() {
		return SessionInfo{}, ErrInactiveUser
	}
	if claims.TokenVersion != u.TokenVersion {
		return SessionInfo{}, ErrTokenRevoked
	}
	return SessionInfo{Role: u.Role, AdminRole: u.AdminRole}, nil
}
