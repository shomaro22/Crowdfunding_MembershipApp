package usecase

import (
	"context"
	"strings"
	"time"

	"crowdfund-app/model"
	"crowdfund-app/validator"

	"github.com/google/uuid"
)

func (p *PasswordUseCase) ForgotPassword(ctx context.Context, email string) error {
	var (
		normalized string
		u          model.User
		err        error
		plain      string
		now        time.Time
		row        *model.ResetToken
		mailer     Mailer
	)

	if p.deps.ResetToks == nil {
		return nil
	}
	normalized = model.NormalizeEmail(email)
	u, err = p.deps.Users.GetEmail(ctx, normalized)
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
	if err = p.deps.ResetToks.Create(ctx, row); err != nil {
		return err
	}
	mailer = p.deps.Mailer
	if mailer == nil {
		return nil
	}
	return mailer.SendPasswordReset(ctx, u.Email, plain)
}

func (p *PasswordUseCase) ResetPassword(ctx context.Context, plainToken, newPassword string) error {
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
	row, err = p.deps.ResetToks.GetValidHash(ctx, OpaqueHash(plainToken), now)
	if err != nil {
		return mapRepoNotFoundAs(err, ErrInvalidToken)
	}
	u, err = p.deps.Users.GetID(ctx, row.UserID)
	if err != nil {
		return mapRepoNotFoundAs(err, ErrInvalidToken)
	}
	hash, err = HashPassword(newPassword)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	u.UpdatedAt = now
	if err = p.deps.Users.Update(ctx, &u); err != nil {
		return err
	}
	if err = p.deps.ResetToks.MarkUsed(ctx, row.ID, now); err != nil {
		return err
	}
	return InvalidateUserSessions(ctx, p.deps.Tx, p.deps.Users, p.deps.RefreshTokens, u.ID)
}

func (p *PasswordUseCase) ChangePassword(ctx context.Context, userID uuid.UUID, current, newPassword, totpCode string) error {
	var (
		err  error
		u    model.User
		hash string
		now  time.Time
	)

	if err = validator.ValidatePassword(newPassword); err != nil {
		return ErrWeakPassword
	}
	u, err = p.deps.Users.GetID(ctx, userID)
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
		if !p.deps.verifyTOTPOrBackup(ctx, u, totpCode) {
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
	if err = p.deps.Users.Update(ctx, &u); err != nil {
		return err
	}
	return InvalidateUserSessions(ctx, p.deps.Tx, p.deps.Users, p.deps.RefreshTokens, u.ID)
}

