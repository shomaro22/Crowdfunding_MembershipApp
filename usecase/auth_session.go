package usecase

import (
	"context"
	"strings"
	"time"

	"crowdfund-app/auth"
	"crowdfund-app/entity"
)

func (a *AuthUseCase) Login(ctx context.Context, email, password string, meta AuthMeta) (*LoginResult, error) {
	normalized := entity.NormalizeEmail(email)
	if normalized == "" || password == "" {
		a.logActivity(ctx, nil, entity.UserLoginFail, "user", normalized, meta, nil)
		return nil, ErrInvalidCredentials
	}
	u, err := a.Users.FindByEmail(ctx, normalized)
	if err != nil {
		return nil, err
	}
	if u == nil || auth.ComparePassword(u.PasswordHash, password) != nil {
		a.logActivity(ctx, nil, entity.UserLoginFail, "user", normalized, meta, nil)
		return nil, ErrInvalidCredentials
	}
	if !u.CanLogin() {
		a.logActivity(ctx, &u.ID, entity.UserLoginFail, "user", u.ID.String(), meta, nil)
		return nil, ErrInactiveUser
	}
	if u.NeedsTOTP() && u.TOTPEnabled {
		a.logActivity(ctx, &u.ID, entity.UserLogin, "user", u.ID.String(), meta, entity.JSONMetadata{"step": "2fa_pending"})
		return &LoginResult{User: u, Requires2FA: true}, nil
	}
	tokens, err := a.issueSession(ctx, u, twoFAVerified(u))
	if err != nil {
		return nil, err
	}
	a.logActivity(ctx, &u.ID, entity.UserLogin, "user", u.ID.String(), meta, nil)
	return &LoginResult{
		User:      u,
		AuthToken: tokens.AuthToken,
		ReToken:   tokens.ReToken,
	}, nil
}

func (a *AuthUseCase) Reissue(ctx context.Context, reTokenPlain string, meta AuthMeta) (string, string, error) {
	plain := strings.TrimSpace(reTokenPlain)
	if plain == "" {
		return "", "", ErrInvalidToken
	}
	hash := auth.OpaqueHash(plain)
	now := time.Now().UTC()
	row, err := a.ReTokens.FindValidByHash(ctx, hash, now)
	if err != nil || row == nil {
		return "", "", ErrInvalidToken
	}
	u, err := a.Users.FindByID(ctx, row.UserID)
	if err != nil || u == nil || !u.CanLogin() {
		return "", "", ErrInvalidToken
	}
	if err := a.ReTokens.Revoke(ctx, row.ID, now); err != nil {
		return "", "", err
	}
	tokens, err := a.issueSession(ctx, u, twoFAVerified(u))
	if err != nil {
		return "", "", err
	}
	return tokens.AuthToken, tokens.ReToken, nil
}

func (a *AuthUseCase) Logout(ctx context.Context, reTokenPlain string, meta AuthMeta) error {
	plain := strings.TrimSpace(reTokenPlain)
	if plain == "" {
		return ErrInvalidToken
	}
	hash := auth.OpaqueHash(plain)
	now := time.Now().UTC()
	row, err := a.ReTokens.FindValidByHash(ctx, hash, now)
	if err != nil {
		return err
	}
	if row == nil {
		return ErrInvalidToken
	}
	if err := a.ReTokens.Revoke(ctx, row.ID, now); err != nil {
		return err
	}
	a.logActivity(ctx, &row.UserID, entity.UserLogout, "user", row.UserID.String(), meta, nil)
	return nil
}
