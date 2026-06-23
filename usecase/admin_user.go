package usecase

import (
	"context"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
)

func (a *AdminUseCase) ResetUser2FA(ctx context.Context, adminID, targetUserID uuid.UUID, reason *string) (*entity.User, error) {
	var (
		u   *entity.User
		err error
		now time.Time
	)

	u, err = a.Users.FindByID(ctx, targetUserID)
	if err != nil || u == nil {
		return nil, ErrNotFound
	}
	if !u.TOTPEnabled {
		return nil, ErrInvalidState
	}
	now = time.Now().UTC()
	u.TOTPEnabled = false
	u.TOTPSecret = nil
	u.UpdatedAt = now
	if err = a.Users.Update(ctx, u); err != nil {
		return nil, err
	}
	if a.TOTPBackup != nil {
		if err = a.TOTPBackup.DeleteByUser(ctx, targetUserID); err != nil {
			return nil, err
		}
	}
	if err = a.invalidateSessions(ctx, targetUserID); err != nil {
		return nil, err
	}
	a.writeAudit(ctx, adminID, "user.reset_2fa", "user", u.ID.String(), reason, nil)
	return u, nil
}

func (a *AdminUseCase) invalidateSessions(ctx context.Context, userID uuid.UUID) error {
	var err error

	if _, err = a.Users.BumpTokenVer(ctx, userID); err != nil {
		return err
	}
	if a.ReTokens == nil {
		return nil
	}
	return a.ReTokens.RevokeAllForUser(ctx, userID, time.Now().UTC())
}
