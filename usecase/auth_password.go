package usecase

import (
	"context"
	"strings"
	"time"

	"crowdfund-app/auth"
	"crowdfund-app/entity"

	"github.com/google/uuid"
)

func (a *AuthUseCase) ForgotPassword(ctx context.Context, email string) error {
	var (
		normalized string
		u          *entity.User
		err        error
		plain      string
		now        time.Time
		row        *entity.ResetToken
		base       string
		url        string
		mailer     Mailer
	)

	if a.ResetToks == nil {
		return nil
	}
	normalized = entity.NormalizeEmail(email)
	u, err = a.Users.FindByEmail(ctx, normalized)
	if err != nil {
		return err
	}
	if u == nil || !u.IsActive() {
		return nil
	}
	plain, err = auth.OpaqueHex()
	if err != nil {
		return err
	}
	now = time.Now().UTC()
	row = &entity.ResetToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: auth.OpaqueHash(plain),
		ExpiresAt: now.Add(resetTokenTTL),
		CreatedAt: now,
	}
	if err = a.ResetToks.Create(ctx, row); err != nil {
		return err
	}
	base = strings.TrimRight(strings.TrimSpace(a.AppPublicURL), "/")
	url = base + "/reset-password?token=" + plain
	mailer = a.Mailer
	if mailer == nil {
		mailer = LogMailer{}
	}
	return mailer.SendPasswordReset(ctx, u.Email, url)
}

func (a *AuthUseCase) ResetPassword(ctx context.Context, plainToken, newPassword string) error {
	var (
		err   error
		now   time.Time
		row   *entity.ResetToken
		u     *entity.User
		hash  string
	)

	if err = entity.ValidatePassword(newPassword); err != nil {
		return ErrWeakPassword
	}
	plainToken = strings.TrimSpace(plainToken)
	if plainToken == "" {
		return ErrInvalidToken
	}
	now = time.Now().UTC()
	row, err = a.ResetToks.FindValidByHash(ctx, auth.OpaqueHash(plainToken), now)
	if err != nil || row == nil {
		return ErrInvalidToken
	}
	u, err = a.Users.FindByID(ctx, row.UserID)
	if err != nil || u == nil {
		return ErrInvalidToken
	}
	hash, err = auth.HashPassword(newPassword)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	u.UpdatedAt = now
	if err = a.Users.Update(ctx, u); err != nil {
		return err
	}
	if err = a.ResetToks.MarkUsed(ctx, row.ID, now); err != nil {
		return err
	}
	return a.invalidateSessions(ctx, u.ID)
}

func (a *AuthUseCase) ChangePassword(ctx context.Context, userID uuid.UUID, current, newPassword, totpCode string) error {
	var (
		err  error
		u    *entity.User
		hash string
		now  time.Time
	)

	if err = entity.ValidatePassword(newPassword); err != nil {
		return ErrWeakPassword
	}
	u, err = a.Users.FindByID(ctx, userID)
	if err != nil || u == nil {
		return ErrNotFound
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
	if auth.ComparePassword(u.PasswordHash, current) != nil {
		return ErrInvalidCredentials
	}
	hash, err = auth.HashPassword(newPassword)
	if err != nil {
		return err
	}
	now = time.Now().UTC()
	u.PasswordHash = hash
	u.UpdatedAt = now
	if err = a.Users.Update(ctx, u); err != nil {
		return err
	}
	return a.invalidateSessions(ctx, u.ID)
}
