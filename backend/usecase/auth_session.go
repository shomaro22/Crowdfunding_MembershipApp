package usecase

import (
	"context"
	"strings"
	"time"

	"crowdfund-app/model"
	"crowdfund-app/repository"

	"github.com/google/uuid"
)

func (s *SessionAuthUseCase) ValidateSession(ctx context.Context, claims AuthClaims) (SessionInfo, error) {
	var (
		u   model.User
		err error
	)
	u, err = s.deps.Users.GetID(ctx, claims.UserID)
	if err != nil {
		return SessionInfo{}, mapRepoNotFound(err)
	}
	if !u.IsActive() {
		return SessionInfo{}, ErrInactiveUser
	}
	if claims.TokenVersion != u.TokenVersion {
		return SessionInfo{}, ErrTokenRevoked
	}
	return SessionInfo{Role: u.Role, AdminRole: u.AdminRole}, nil
}

func (s *SessionAuthUseCase) ParseToken(raw string) (AuthClaims, error) {
	return s.deps.Tokens.Parse(raw)
}

func (s *SessionAuthUseCase) Login(ctx context.Context, email, password string, meta RequestMeta) (*LoginResult, error) {
	var (
		normalized string
		u          model.User
		err        error
		tokens     *SessionTokens
	)

	normalized = model.NormalizeEmail(email)
	if normalized == "" || password == "" {
		s.deps.logActivity(ctx, nil, model.UserLoginFail, "user", normalized, meta, nil)
		return nil, ErrInvalidCredentials
	}
	u, err = s.deps.Users.GetEmail(ctx, normalized)
	if err != nil {
		return nil, mapRepoNotFoundAs(err, ErrInvalidCredentials)
	}
	if ComparePassword(u.PasswordHash, password) != nil {
		s.deps.logActivity(ctx, nil, model.UserLoginFail, "user", normalized, meta, nil)
		return nil, ErrInvalidCredentials
	}
	if !u.CanLogin() {
		s.deps.logActivity(ctx, &u.ID, model.UserLoginFail, "user", u.ID.String(), meta, nil)
		return nil, ErrInactiveUser
	}
	if u.NeedsTOTP() && u.TOTPEnabled {
		s.deps.logActivity(ctx, &u.ID, model.UserLogin, "user", u.ID.String(), meta, model.StepMeta("2fa_pending"))
		return &LoginResult{User: UserViewFrom(u), Requires2FA: true}, nil
	}
	tokens, err = s.deps.issueSession(ctx, u, twoFAVerified(u))
	if err != nil {
		return nil, err
	}
	s.deps.logActivity(ctx, &u.ID, model.UserLogin, "user", u.ID.String(), meta, nil)
	return &LoginResult{
		User:         UserViewFrom(u),
		AuthToken:    tokens.AuthToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (s *SessionAuthUseCase) Refresh(ctx context.Context, refreshTokenPlain string, meta RequestMeta) (string, string, error) {
	var (
		plain      string
		hash       string
		now        time.Time
		row        model.RefreshToken
		revokedRow model.RefreshToken
		err        error
		u          model.User
		tokens     *SessionTokens
	)

	plain = strings.TrimSpace(refreshTokenPlain)
	if plain == "" {
		return "", "", ErrInvalidToken
	}
	hash = OpaqueHash(plain)
	now = time.Now().UTC()
	row, err = s.deps.RefreshTokens.GetValidHash(ctx, hash, now)
	if err != nil {
		revokedRow, err = s.deps.RefreshTokens.GetByHash(ctx, hash)
		if err == nil && revokedRow.IsRevoked() {
			_ = InvalidateUserSessions(ctx, s.deps.DB, s.deps.Users, s.deps.RefreshTokens, revokedRow.UserID)
			s.deps.logActivity(ctx, &revokedRow.UserID, model.UserLoginFail, "user", revokedRow.UserID.String(), meta,
				model.StepMeta("refresh_token_reuse"))
			return "", "", ErrTokenRevoked
		}
		return "", "", mapRepoNotFoundAs(err, ErrInvalidToken)
	}
	u, err = s.deps.Users.GetID(ctx, row.UserID)
	if err != nil {
		return "", "", mapRepoNotFoundAs(err, ErrInvalidToken)
	}
	if !u.CanLogin() {
		return "", "", ErrInvalidToken
	}
	err = repository.WithTx(ctx, s.deps.DB, func(txCtx context.Context) error {
		if err = s.deps.RefreshTokens.Revoke(txCtx, row.ID, now); err != nil {
			return err
		}
		tokens, err = s.deps.issueSession(txCtx, u, twoFAVerified(u))
		return err
	})
	if err != nil {
		return "", "", err
	}
	return tokens.AuthToken, tokens.RefreshToken, nil
}

func (s *SessionAuthUseCase) Logout(ctx context.Context, refreshTokenPlain string, meta RequestMeta) error {
	var (
		plain string
		hash  string
		now   time.Time
		row   model.RefreshToken
		err   error
	)

	plain = strings.TrimSpace(refreshTokenPlain)
	if plain == "" {
		return ErrInvalidToken
	}
	hash = OpaqueHash(plain)
	now = time.Now().UTC()
	row, err = s.deps.RefreshTokens.GetValidHash(ctx, hash, now)
	if err != nil {
		return mapRepoNotFoundAs(err, ErrInvalidToken)
	}
	if err = s.deps.RefreshTokens.Revoke(ctx, row.ID, now); err != nil {
		return err
	}
	s.deps.logActivity(ctx, &row.UserID, model.UserLogout, "user", row.UserID.String(), meta, nil)
	return nil
}

func (s *SessionAuthUseCase) LogoutAllDevices(ctx context.Context, userID uuid.UUID, meta RequestMeta) error {
	var (
		err error
	)
	if userID == uuid.Nil {
		return ErrInvalidToken
	}
	err = InvalidateUserSessions(ctx, s.deps.DB, s.deps.Users, s.deps.RefreshTokens, userID)
	if err != nil {
		return err
	}
	s.deps.logActivity(ctx, &userID, model.UserLogout, "user", userID.String(), meta, model.StepMeta("all_devices"))
	return nil
}
