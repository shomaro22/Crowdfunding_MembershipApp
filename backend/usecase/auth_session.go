package usecase

import (
	"context"
	"strings"
	"time"

	
	"crowdfund-app/model"
	"crowdfund-app/repository"
	"crowdfund-app/validator"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (a *AuthUseCase) issueSession(ctx context.Context, u model.User, verified2FA bool) (*SessionTokens, error) {
	var (
		authToken string
		err       error
		plain     string
		now       time.Time
		row       *model.RefreshToken
	)
	authToken, err = a.Tokens.Issue(AuthClaims{
		UserID:        u.ID,
		Role:          u.Role,
		TwoFAVerified: verified2FA,
		AdminRole:     u.AdminRole,
		TokenVersion:  u.TokenVersion,
	})
	if err != nil {
		return nil, err
	}
	plain, err = OpaqueHex()
	if err != nil {
		return nil, err
	}
	now = time.Now().UTC()
	row = &model.RefreshToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: OpaqueHash(plain),
		ExpiresAt: now.Add(refreshTokenTTL),
		CreatedAt: now,
	}
	if err = a.RefreshTokens.Create(ctx, row); err != nil {
		return nil, err
	}
	return &SessionTokens{AuthToken: authToken, RefreshToken: plain}, nil
}

func (a *AuthUseCase) ValidateSession(ctx context.Context, claims AuthClaims) (SessionInfo, error) {
	var (
		u   model.User
		err error
	)
	u, err = a.Users.GetID(ctx, claims.UserID)
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

func (a *AuthUseCase) ParseToken(raw string) (AuthClaims, error) {
	return a.Tokens.Parse(raw)
}

func (a *AuthUseCase) Login(ctx context.Context, email, password string, meta RequestMeta) (*LoginResult, error) {
	var (
		normalized string
		u          model.User
		err        error
		tokens     *SessionTokens
	)

	normalized = model.NormalizeEmail(email)
	if normalized == "" || password == "" {
		a.logActivity(ctx, nil, model.UserLoginFail, "user", normalized, meta, nil)
		return nil, ErrInvalidCredentials
	}
	u, err = a.Users.GetEmail(ctx, normalized)
	if err != nil {
		return nil, mapRepoNotFoundAs(err, ErrInvalidCredentials)
	}
	if ComparePassword(u.PasswordHash, password) != nil {
		a.logActivity(ctx, nil, model.UserLoginFail, "user", normalized, meta, nil)
		return nil, ErrInvalidCredentials
	}
	if !u.CanLogin() {
		a.logActivity(ctx, &u.ID, model.UserLoginFail, "user", u.ID.String(), meta, nil)
		return nil, ErrInactiveUser
	}
	if u.NeedsTOTP() && u.TOTPEnabled {
		a.logActivity(ctx, &u.ID, model.UserLogin, "user", u.ID.String(), meta, model.StepMeta("2fa_pending"))
		return &LoginResult{User: UserViewFrom(u), Requires2FA: true}, nil
	}
	tokens, err = a.issueSession(ctx, u, twoFAVerified(u))
	if err != nil {
		return nil, err
	}
	a.logActivity(ctx, &u.ID, model.UserLogin, "user", u.ID.String(), meta, nil)
	return &LoginResult{
		User:         UserViewFrom(u),
		AuthToken:    tokens.AuthToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (a *AuthUseCase) Refresh(ctx context.Context, refreshTokenPlain string, meta RequestMeta) (string, string, error) {
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
	row, err = a.RefreshTokens.GetValidHash(ctx, hash, now)
	if err != nil {
		revokedRow, err = a.RefreshTokens.GetByHash(ctx, hash)
		if err == nil && revokedRow.IsRevoked() {
			_ = InvalidateUserSessions(ctx, a.DB, a.Users, a.RefreshTokens, revokedRow.UserID)
			a.logActivity(ctx, &revokedRow.UserID, model.UserLoginFail, "user", revokedRow.UserID.String(), meta,
				model.StepMeta("refresh_token_reuse"))
			return "", "", ErrTokenRevoked
		}
		return "", "", mapRepoNotFoundAs(err, ErrInvalidToken)
	}
	u, err = a.Users.GetID(ctx, row.UserID)
	if err != nil {
		return "", "", mapRepoNotFoundAs(err, ErrInvalidToken)
	}
	if !u.CanLogin() {
		return "", "", ErrInvalidToken
	}
	err = repository.WithTx(ctx, a.DB, func(txCtx context.Context) error {
		if err = a.RefreshTokens.Revoke(txCtx, row.ID, now); err != nil {
			return err
		}
		tokens, err = a.issueSession(txCtx, u, twoFAVerified(u))
		return err
	})
	if err != nil {
		return "", "", err
	}
	return tokens.AuthToken, tokens.RefreshToken, nil
}

func (a *AuthUseCase) Logout(ctx context.Context, refreshTokenPlain string, meta RequestMeta) error {
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
	row, err = a.RefreshTokens.GetValidHash(ctx, hash, now)
	if err != nil {
		return mapRepoNotFoundAs(err, ErrInvalidToken)
	}
	if err = a.RefreshTokens.Revoke(ctx, row.ID, now); err != nil {
		return err
	}
	a.logActivity(ctx, &row.UserID, model.UserLogout, "user", row.UserID.String(), meta, nil)
	return nil
}

func (a *AuthUseCase) LogoutAllDevices(ctx context.Context, userID uuid.UUID, meta RequestMeta) error {
	var (
		err error
	)
	if userID == uuid.Nil {
		return ErrInvalidToken
	}
	err = InvalidateUserSessions(ctx, a.DB, a.Users, a.RefreshTokens, userID)
	if err != nil {
		return err
	}
	a.logActivity(ctx, &userID, model.UserLogout, "user", userID.String(), meta, model.StepMeta("all_devices"))
	return nil
}

func (a *AuthUseCase) ForgotPassword(ctx context.Context, email string) error {
	var (
		normalized string
		u          model.User
		err        error
		plain      string
		now        time.Time
		row        *model.ResetToken
		mailer     Mailer
	)

	if a.ResetToks == nil {
		return nil
	}
	normalized = model.NormalizeEmail(email)
	u, err = a.Users.GetEmail(ctx, normalized)
	if isRepoNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !u.IsActive() {
		return nil
	}
	plain, err = OpaqueHex()
	if err != nil {
		return err
	}
	now = time.Now().UTC()
	row = &model.ResetToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: OpaqueHash(plain),
		ExpiresAt: now.Add(resetTokenTTL),
		CreatedAt: now,
	}
	if err = a.ResetToks.Create(ctx, row); err != nil {
		return err
	}
	mailer = a.Mailer
	if mailer == nil {
		return nil
	}
	return mailer.SendPasswordReset(ctx, u.Email, plain)
}

func (a *AuthUseCase) ResetPassword(ctx context.Context, plainToken, newPassword string) error {
	var (
		err  error
		now  time.Time
		row  model.ResetToken
		u    model.User
		hash string
	)

	if err = validator.ValidatePassword(newPassword); err != nil {
		return ErrWeakPassword
	}
	plainToken = strings.TrimSpace(plainToken)
	if plainToken == "" {
		return ErrInvalidToken
	}
	now = time.Now().UTC()
	row, err = a.ResetToks.GetValidHash(ctx, OpaqueHash(plainToken), now)
	if err != nil {
		return mapRepoNotFoundAs(err, ErrInvalidToken)
	}
	u, err = a.Users.GetID(ctx, row.UserID)
	if err != nil {
		return mapRepoNotFoundAs(err, ErrInvalidToken)
	}
	hash, err = HashPassword(newPassword)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	u.UpdatedAt = now
	if err = a.Users.Update(ctx, &u); err != nil {
		return err
	}
	if err = a.ResetToks.MarkUsed(ctx, row.ID, now); err != nil {
		return err
	}
	return InvalidateUserSessions(ctx, a.DB, a.Users, a.RefreshTokens, u.ID)
}

func (a *AuthUseCase) ChangePassword(ctx context.Context, userID uuid.UUID, current, newPassword, totpCode string) error {
	var (
		err  error
		u    model.User
		hash string
		now  time.Time
	)

	if err = validator.ValidatePassword(newPassword); err != nil {
		return ErrWeakPassword
	}
	u, err = a.Users.GetID(ctx, userID)
	if err != nil {
		return mapRepoNotFound(err)
	}
	if u.MustSetup2FA() {
		if strings.TrimSpace(totpCode) == "" {
			return ErrRequires2FA
		}
		if !u.TOTPEnabled {
			return Err2FANotConfigured
		}
		if !a.verifyTOTPOrBackup(ctx, u, totpCode) {
			return ErrInvalid2FACode
		}
	}
	if ComparePassword(u.PasswordHash, current) != nil {
		return ErrInvalidCredentials
	}
	hash, err = HashPassword(newPassword)
	if err != nil {
		return err
	}
	now = time.Now().UTC()
	u.PasswordHash = hash
	u.UpdatedAt = now
	if err = a.Users.Update(ctx, &u); err != nil {
		return err
	}
	return InvalidateUserSessions(ctx, a.DB, a.Users, a.RefreshTokens, u.ID)
}

type UpdateProfileIn struct {
	Name      *string
	AvatarURL *string
}

func (a *AuthUseCase) GetUser(ctx context.Context, userID uuid.UUID) (UserView, error) {
	var (
		u   model.User
		err error
	)

	u, err = a.Users.GetID(ctx, userID)
	if err != nil {
		return UserView{}, mapRepoNotFound(err)
	}
	return UserViewFrom(u), nil
}

func (a *AuthUseCase) UpdateProfile(ctx context.Context, userID uuid.UUID, in UpdateProfileIn) (UserView, error) {
	var (
		norm validator.UpdateProfileNormalized
		u    model.User
		err  error
	)

	norm, err = validator.ValidateUpdateProfile(validator.UpdateProfileInput{
		Name:      in.Name,
		AvatarURL: in.AvatarURL,
	})
	if err != nil {
		return UserView{}, mapProfileModelErr(err)
	}
	u, err = a.Users.GetID(ctx, userID)
	if err != nil {
		return UserView{}, mapRepoNotFound(err)
	}
	if norm.NameSet {
		u.Name = norm.Name
	}
	if norm.AvatarURLSet {
		u.AvatarURL = norm.AvatarURL
	}
	if err = a.Users.Update(ctx, &u); err != nil {
		return UserView{}, err
	}
	return UserViewFrom(u), nil
}

func InvalidateUserSessions(
	ctx context.Context,
	db *gorm.DB,
	users repository.IUserRepo,
	refresh repository.IRefreshTokenRepo,
	userID uuid.UUID,
) error {
	var err error

	return repository.WithTx(ctx, db, func(txCtx context.Context) error {
		if _, err = users.BumpTokenVer(txCtx, userID); err != nil {
			return err
		}
		if refresh == nil {
			return nil
		}
		return refresh.RevokeAllByUser(txCtx, userID, time.Now().UTC())
	})
}
